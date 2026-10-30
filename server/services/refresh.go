// Refresh token 轮换与会话管理：多端并存会话（Redis 按用户存 per-jti 会话集合）
package services

import (
	"context"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/redis/go-redis/v9"
)

const (
	refreshPrefix = "auth:refresh:"    // auth:refresh:{uid} 存储 hash {jti: exp}，每端一个字段
	refreshTTL    = 7 * 24 * time.Hour // 与 refresh token 有效期一致
)

var (
	// ErrRefreshRejected refresh token 校验不通过（无效/过期/已被轮换）
	ErrRefreshRejected = errors.New("refresh token rejected")

	redisOnce sync.Once
	rdb       *redis.Client
	ctx       = context.Background()
)

// redisEnabled 控制 refresh 会话是否落 Redis。默认关闭以便本地/测试无 Redis 运行；
// 生产多实例部署应设为 true（app.conf redis_enabled = true）。
func redisEnabled() bool {
	return web.AppConfig.DefaultBool("redis_enabled", false)
}

// initRedis 初始化 Redis 连接，sync.Once 保证并发安全（只创建一次）。
func initRedis() *redis.Client {
	redisOnce.Do(func() {
		addr := web.AppConfig.DefaultString("redisaddr", "127.0.0.1:6379")
		pass := web.AppConfig.DefaultString("redispass", "")
		rdb = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: pass,
		})
	})
	return rdb
}

// refreshUserActive 校验用户存在且未被封禁；测试可替换以避免依赖真实 DB。
var refreshUserActive = func(uid int64) bool {
	o := orm.NewOrm()
	u := models.User{Id: uid}
	return o.Read(&u) == nil && u.Status == 1
}

// RefreshTokenStore 描述 refresh token 会话存储。
// 默认实现按 redis_enabled 选择 Redis 或进程内存储；测试可替换为 stub（见 refresh_test.go）。
type RefreshTokenStore interface {
	// Save 记录该用户一个有效 refresh token（jti 标识）；多端并存互不覆盖。
	Save(uid int64, jti string, exp time.Time) error
	// Consume 原子地校验并删除该 refresh 会话：存在且未过期返回 true（单次使用，防并发重放）。
	// Redis 故障时降级放行（返回 true）；会话不存在/已过期返回 false。
	Consume(uid int64, jti string) bool
	// Delete 删除该用户单个 refresh 会话（单端登出）。
	Delete(uid int64, jti string) error
	// DeleteAll 删除该用户全部 refresh 会话（登出所有端）。
	DeleteAll(uid int64) error
}

// refreshStore 全局会话存储实例，测试中可整体替换。
var refreshStore RefreshTokenStore = defaultRefreshStore{}

// defaultRefreshStore 按配置分发：redis_enabled=true 走 Redis，否则走进程内存储。
type defaultRefreshStore struct{}

func (defaultRefreshStore) Save(uid int64, jti string, exp time.Time) error {
	if redisEnabled() {
		return redisRefreshStore{}.Save(uid, jti, exp)
	}
	return memRefresh.Save(uid, jti, exp)
}

func (defaultRefreshStore) Consume(uid int64, jti string) bool {
	if redisEnabled() {
		return redisRefreshStore{}.Consume(uid, jti)
	}
	return memRefresh.Consume(uid, jti)
}

func (defaultRefreshStore) Delete(uid int64, jti string) error {
	if redisEnabled() {
		return redisRefreshStore{}.Delete(uid, jti)
	}
	return memRefresh.Delete(uid, jti)
}

func (defaultRefreshStore) DeleteAll(uid int64) error {
	if redisEnabled() {
		return redisRefreshStore{}.DeleteAll(uid)
	}
	return memRefresh.DeleteAll(uid)
}

// memRefresh 进程内会话存储：redis_enabled=false（默认）时生效，单实例够用。
// 多实例部署必须启用 Redis，否则各实例的会话互不可见。
var memRefresh = &memoryRefreshStore{sessions: make(map[int64]map[string]time.Time)}

type memoryRefreshStore struct {
	mu       sync.Mutex
	sessions map[int64]map[string]time.Time // uid -> jti -> 过期时间
}

func (m *memoryRefreshStore) Save(uid int64, jti string, exp time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prune(uid)
	if m.sessions[uid] == nil {
		m.sessions[uid] = make(map[string]time.Time)
	}
	m.sessions[uid][jti] = exp
	return nil
}

func (m *memoryRefreshStore) Consume(uid int64, jti string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.sessions[uid][jti]
	if !ok {
		return false
	}
	delete(m.sessions[uid], jti)
	if len(m.sessions[uid]) == 0 {
		delete(m.sessions, uid)
	}
	return time.Now().Before(exp)
}

func (m *memoryRefreshStore) Delete(uid int64, jti string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions[uid], jti)
	if len(m.sessions[uid]) == 0 {
		delete(m.sessions, uid)
	}
	return nil
}

func (m *memoryRefreshStore) DeleteAll(uid int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, uid)
	return nil
}

// prune 清理该用户已过期会话（调用方须持锁）。
func (m *memoryRefreshStore) prune(uid int64) {
	now := time.Now()
	for jti, exp := range m.sessions[uid] {
		if !now.Before(exp) {
			delete(m.sessions[uid], jti)
		}
	}
}

// redisRefreshStore 基于 Redis hash 的实现：auth:refresh:{uid} -> {<jti>: <exp>}
type redisRefreshStore struct{}

func (redisRefreshStore) Save(uid int64, jti string, exp time.Time) error {
	key := refreshPrefix + strconv.FormatInt(uid, 10)
	c := initRedis()
	if err := c.HSet(ctx, key, jti, exp.Unix()).Err(); err != nil {
		// Redis 不可用：优雅降级，不阻断登录/刷新（与项目现有降级风格一致）
		log.Printf("refresh: save failed uid=%d jti=%s: %v", uid, jti, err)
		return nil
	}
	c.Expire(ctx, key, refreshTTL)
	return nil
}

func (redisRefreshStore) Consume(uid int64, jti string) bool {
	key := refreshPrefix + strconv.FormatInt(uid, 10)
	n, err := initRedis().HDel(ctx, key, jti).Result()
	if err != nil {
		// Redis 故障：优雅降级放行（与项目现有降级风格一致；故障窗口内重放无法防御）
		log.Printf("refresh: consume failed uid=%d jti=%s: %v", uid, jti, err)
		return true
	}
	return n > 0
}

func (redisRefreshStore) Delete(uid int64, jti string) error {
	c := initRedis()
	return c.HDel(ctx, refreshPrefix+strconv.FormatInt(uid, 10), jti).Err()
}

func (redisRefreshStore) DeleteAll(uid int64) error {
	c := initRedis()
	return c.Del(ctx, refreshPrefix+strconv.FormatInt(uid, 10)).Err()
}

// Logout 撤销该用户单个 refresh 会话（单端登出）。
func Logout(uid int64, jti string) error {
	return refreshStore.Delete(uid, jti)
}

// LogoutAll 撤销该用户全部 refresh 会话（登出所有端）。
func LogoutAll(uid int64) error {
	return refreshStore.DeleteAll(uid)
}

// SaveRefreshSession 将 refresh token 注册为该用户一个有效会话（写入其 jti）。
// 登录、轮换后调用；多端并存，互不覆盖。
func SaveRefreshSession(uid int64, refreshToken string) error {
	claims, err := middleware.ParseToken(refreshToken)
	if err != nil {
		return err
	}
	if claims.ID == "" {
		return ErrRefreshRejected
	}
	exp := time.Now().Add(refreshTTL)
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Time
	}
	return refreshStore.Save(uid, claims.ID, exp)
}

// RotateRefresh 轮换 refresh token（每端独立、旧 token 防重放）：
//  1. 校验 presented token 签名/类型/有效期；
//  2. 校验用户仍存在且未封禁（封号后旧 refresh 不可再换发）；
//  3. 原子消费 presented jti：仅仍有效的会话放行（已轮换/登出过的 token 拒绝）；
//  4. 签发新的 access + refresh 并注册新 jti，其他端会话不受影响。
func RotateRefresh(presented string) (newAccess, newRefresh string, err error) {
	claims, err := middleware.ParseToken(presented)
	if err != nil || claims.TokenTyp != "refresh" || claims.ID == "" {
		return "", "", ErrRefreshRejected
	}
	if !refreshUserActive(claims.UserID) {
		return "", "", ErrRefreshRejected // 不消耗 token：解封后仍可继续使用
	}
	// 原子消费本次使用的 jti（该 refresh 一次性）；仅删本字段，他端会话保留。
	// 并发重放时只有一个请求能通过 Consume。
	if !refreshStore.Consume(claims.UserID, claims.ID) {
		return "", "", ErrRefreshRejected
	}

	newAccess, err = middleware.GenerateAccessToken(claims.UserID, claims.Role)
	if err != nil {
		return "", "", err
	}
	newRefresh, err = middleware.GenerateRefreshToken(claims.UserID, claims.Role)
	if err != nil {
		return "", "", err
	}
	if err := SaveRefreshSession(claims.UserID, newRefresh); err != nil {
		return "", "", err
	}
	return newAccess, newRefresh, nil
}
