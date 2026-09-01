// JWT 签发与校验单元测试：round-trip、篡改、alg:none、过期、密钥 fail-fast
package middleware

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

// 测试用强密钥（FakeConfig 下 conf 不存在，env 覆盖生效，避免命中默认密钥 panic）。
const testSecret = "unit-test-jwt-secret-9f2c1a"

func registerSecret(t *testing.T) {
	t.Setenv("ITEM_RENTAL_JWT_SECRET", testSecret)
}

func TestTokenRoundTrip(t *testing.T) {
	registerSecret(t)

	access, err := GenerateAccessToken(42, "user")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	claims, err := ParseToken(access)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.UserID != 42 || claims.Role != "user" || claims.TokenTyp != "access" {
		t.Errorf("access claims 异常: %+v", claims)
	}

	refresh, err := GenerateRefreshToken(7, "admin")
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	rc, err := ParseToken(refresh)
	if err != nil {
		t.Fatalf("ParseToken refresh: %v", err)
	}
	if rc.TokenTyp != "refresh" || rc.UserID != 7 {
		t.Errorf("refresh claims 异常: %+v", rc)
	}
}

func TestParseTokenRejectsTampered(t *testing.T) {
	registerSecret(t)

	tok, _ := GenerateAccessToken(1, "user")
	// 篡改 payload 中的 uid
	parts := strings.Split(tok, ".")
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"uid":999,"typ":"access"}`))
	if _, err := ParseToken(parts[0] + "." + payload + "." + parts[2]); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("篡改 token 应返回 ErrInvalidToken, got %v", err)
	}
}

func TestParseTokenRejectsAlgNone(t *testing.T) {
	registerSecret(t)

	// 手工构造 alg:none 攻击载荷（伪造任意用户）
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"uid":1,"typ":"access"}`))
	forge := header + "." + payload + "."
	if _, err := ParseToken(forge); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("alg:none 应被拒绝, got %v", err)
	}
}

func TestParseTokenExpired(t *testing.T) {
	registerSecret(t)

	// generateToken 直接签一个已过期（负 TTL）的 token
	expired, err := generateToken(1, "user", "access", -10)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	if _, err := ParseToken(expired); !errors.Is(err, ErrExpiredToken) {
		t.Errorf("过期 token 应返回 ErrExpiredToken, got %v", err)
	}
}

func TestJWTSecretFailFast(t *testing.T) {
	// 空密钥（且非默认占位符）必须 panic，不允许静默使用弱密钥
	for _, secret := range []string{"", "change-me-in-prod-rental-secret"} {
		t.Run("secret="+secret, func(t *testing.T) {
			t.Setenv("ITEM_RENTAL_JWT_SECRET", secret)
			defer func() {
				if r := recover(); r == nil {
					t.Error("弱密钥应触发 panic（fail-fast）")
				}
			}()
			_ = jwtSecret()
		})
	}
}