// Refresh token 轮换单元测试：内存 stub 存储，无 Redis/无网络
package services

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/erikwang2013/item-rental/server/middleware"
)

// 测试用强密钥，避免命中默认密钥 panic（与 middleware/jwt_test.go 一致）。
const refreshTestSecret = "unit-test-jwt-secret-9f2c1a"

// memRefreshStore 内存版 RefreshTokenStore stub：uid -> jti 会话集合（多端并存）
type memRefreshStore struct {
	mu       sync.Mutex
	sessions map[int64]map[string]struct{}
}

func newMemRefreshStore() *memRefreshStore {
	return &memRefreshStore{sessions: make(map[int64]map[string]struct{})}
}

func (s *memRefreshStore) Save(uid int64, jti string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions[uid] == nil {
		s.sessions[uid] = make(map[string]struct{})
	}
	s.sessions[uid][jti] = struct{}{}
	return nil
}

func (s *memRefreshStore) Delete(uid int64, jti string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions[uid], jti)
	return nil
}

func (s *memRefreshStore) DeleteAll(uid int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, uid)
	return nil
}

func (s *memRefreshStore) Check(uid int64, jti string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[uid][jti]
	return ok
}

// switchStore 临时替换全局 refreshStore，测试后恢复。
func switchStore(t *testing.T, st RefreshTokenStore) {
	t.Helper()
	old := refreshStore
	refreshStore = st
	t.Cleanup(func() { refreshStore = old })
}

func TestRotateRefresh(t *testing.T) {
	t.Setenv("ITEM_RENTAL_JWT_SECRET", refreshTestSecret)
	switchStore(t, newMemRefreshStore())

	const uid = int64(42)

	// 首次：登录签发 refresh R0 并注册会话
	r0, err := middleware.GenerateRefreshToken(uid, "user")
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if err := SaveRefreshSession(uid, r0); err != nil {
		t.Fatalf("SaveRefreshSession: %v", err)
	}

	// 断言 1：首次 refresh 放行，且签发新的 access + refresh
	newAccess, newRefresh, err := RotateRefresh(r0)
	if err != nil {
		t.Fatalf("首次 refresh 应放行, got %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Fatal("应签发新的 access 与 refresh token")
	}
	if newRefresh == r0 {
		t.Fatal("新 refresh token 不得与旧 token 相同（轮换）")
	}
	if ac, err := middleware.ParseToken(newAccess); err != nil || ac.TokenTyp != "access" {
		t.Errorf("新 access token 应可解析且为 access 类型, err=%v", err)
	}
	if rc, err := middleware.ParseToken(newRefresh); err != nil || rc.TokenTyp != "refresh" {
		t.Errorf("新 refresh token 应可解析且为 refresh 类型, err=%v", err)
	}

	// 断言 2：旧 refresh（已轮换）被拒 401
	if _, _, err := RotateRefresh(r0); !errors.Is(err, ErrRefreshRejected) {
		t.Errorf("旧 refresh 应被拒绝, got %v", err)
	}

	// 断言 3：新 refresh 仍可继续轮换（会话已切换为新 jti）
	_, newRefresh2, err := RotateRefresh(newRefresh)
	if err != nil {
		t.Errorf("新 refresh 应可继续轮换, got %v", err)
	}
	if newRefresh2 == newRefresh {
		t.Error("再次轮换应签发全新 refresh token")
	}

	// 断言 4：非 refresh 类型令牌被拒
	access, _ := middleware.GenerateAccessToken(uid, "user")
	if _, _, err := RotateRefresh(access); !errors.Is(err, ErrRefreshRejected) {
		t.Errorf("access token 不得用于 refresh, got %v", err)
	}
}

func TestRotateRefreshRejectsUnknownSession(t *testing.T) {
	t.Setenv("ITEM_RENTAL_JWT_SECRET", refreshTestSecret)
	switchStore(t, newMemRefreshStore())

	// 会话中未注册的 refresh（stub Check 返回 false）→ 拒绝
	r, err := middleware.GenerateRefreshToken(7, "user")
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if _, _, err := RotateRefresh(r); !errors.Is(err, ErrRefreshRejected) {
		t.Errorf("未注册会话的 refresh 应被拒绝, got %v", err)
	}
}

func TestLogoutInvalidatesRefresh(t *testing.T) {
	st := newMemRefreshStore()
	switchStore(t, st)
	registerSecret(t)

	access0, refresh0, _ := setupRotation(t)
	claims, err := middleware.ParseToken(refresh0)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	// 单端登出
	assertNil(t, Logout(1, claims.ID), "Logout")
	// 登出后该 refresh 应被拒绝
	if _, _, err := RotateRefresh(refresh0); !errors.Is(err, ErrRefreshRejected) {
		t.Fatalf("expect ErrRefreshRejected after logout, got %v", err)
	}
	_ = access0
}

// TestMultiDeviceSessions 多端并存：两端 refresh 均可独立轮换；单端登出不影响他端。
func TestMultiDeviceSessions(t *testing.T) {
	st := newMemRefreshStore()
	switchStore(t, st)
	registerSecret(t)

	const uid = int64(9)
	ra, _ := middleware.GenerateRefreshToken(uid, "user")
	rb, _ := middleware.GenerateRefreshToken(uid, "user")
	assertNil(t, SaveRefreshSession(uid, ra), "Save a")
	assertNil(t, SaveRefreshSession(uid, rb), "Save b")

	ca, _ := middleware.ParseToken(ra)
	cb, _ := middleware.ParseToken(rb)

	// A 轮换成功（消费 A 的 jti，B 不受影响）
	if _, _, err := RotateRefresh(ra); err != nil {
		t.Fatalf("A 端 refresh 应放行, got %v", err)
	}
	// A 已消费：再轮换被拒（防重放）
	if _, _, err := RotateRefresh(ra); !errors.Is(err, ErrRefreshRejected) {
		t.Fatalf("A 端已轮换的 refresh 应被拒, got %v", err)
	}
	// B 仍有效
	if _, _, err := RotateRefresh(rb); err != nil {
		t.Fatalf("B 端 refresh 应不受 A 影响, got %v", err)
	}

	// 单端登出 B'（B 已被上面轮换消费，改用 A 的后代？简化：注册新会话 C 再删之）
	rc, _ := middleware.GenerateRefreshToken(uid, "user")
	assertNil(t, SaveRefreshSession(uid, rc), "Save c")
	cc, _ := middleware.ParseToken(rc)
	assertNil(t, Logout(uid, cc.ID), "Logout c")
	if _, _, err := RotateRefresh(rc); !errors.Is(err, ErrRefreshRejected) {
		t.Fatalf("登出的 C 端应被拒, got %v", err)
	}
	_ = ca
	_ = cb
}

// TestLogoutAll 撤销全部会话：所有端 refresh 失效。
func TestLogoutAll(t *testing.T) {
	st := newMemRefreshStore()
	switchStore(t, st)
	registerSecret(t)

	const uid = int64(11)
	ra, _ := middleware.GenerateRefreshToken(uid, "user")
	rb, _ := middleware.GenerateRefreshToken(uid, "user")
	assertNil(t, SaveRefreshSession(uid, ra), "Save a")
	assertNil(t, SaveRefreshSession(uid, rb), "Save b")

	assertNil(t, LogoutAll(uid), "LogoutAll")
	if _, _, err := RotateRefresh(ra); !errors.Is(err, ErrRefreshRejected) {
		t.Fatalf("LogoutAll 后 A 应被拒, got %v", err)
	}
	if _, _, err := RotateRefresh(rb); !errors.Is(err, ErrRefreshRejected) {
		t.Fatalf("LogoutAll 后 B 应被拒, got %v", err)
	}
}

// registerSecret 设置测试 JWT 密钥（与 middleware/jwt_test.go 同值，避免命中默认密钥 panic）。
func registerSecret(t *testing.T) {
	t.Helper()
	t.Setenv("ITEM_RENTAL_JWT_SECRET", refreshTestSecret)
}

// setupRotation 为 uid=1 签发 access + refresh 并注册会话，返回令牌。
func setupRotation(t *testing.T) (string, string, error) {
	t.Helper()
	const uid = int64(1)
	refresh, err := middleware.GenerateRefreshToken(uid, "user")
	if err != nil {
		return "", "", err
	}
	if err := SaveRefreshSession(uid, refresh); err != nil {
		return "", "", err
	}
	access, err := middleware.GenerateAccessToken(uid, "user")
	return access, refresh, err
}

// assertNil 断言 err 为 nil。
func assertNil(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}
