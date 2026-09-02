// JWT 工具：签发与校验 access/refresh 令牌
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken 令牌无效
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken 令牌过期
	ErrExpiredToken = errors.New("token expired")
)

// Claims 自定义 JWT 载荷
type Claims struct {
	UserID   int64  `json:"uid"`
	Role     string `json:"role,omitempty"`
	TokenTyp string `json:"typ"` // access / refresh
	jwt.RegisteredClaims
}

// jwtSecret 从配置读取 JWT 密钥。
// 安全要求：必须显式配置强密钥，未配置则启动失败（fail-fast），
// 禁止使用可被猜测的默认密钥（否则攻击者可伪造任意用户令牌）。
func jwtSecret() []byte {
	s := web.AppConfig.DefaultString("jwtsecret", "")
	if env := os.Getenv("ITEM_RENTAL_JWT_SECRET"); env != "" {
		s = env
	}
	if s == "" || s == "change-me-in-prod-rental-secret" {
		panic("JWT 密钥未配置或仍为默认值，请通过 ITEM_RENTAL_JWT_SECRET 设置强随机密钥")
	}
	return []byte(s)
}

// generateToken 签发指定类型令牌
func generateToken(userID int64, role, tokenTyp string, ttlSeconds int) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Role:     role,
		TokenTyp: tokenTyp,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        newJTI(), // 每枚令牌唯一 jti，供 refresh 轮换时比对会话
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttlSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "item-rental",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// newJTI 生成随机 token ID（crypto/rand，碰撞概率可忽略）。
func newJTI() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 理论上不会失败；极端回退保证令牌可用
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// GenerateAccessToken 签发 access token（默认 2 小时）
func GenerateAccessToken(userID int64, role string) (string, error) {
	ttl := web.AppConfig.DefaultInt("jwtttl", 7200)
	return generateToken(userID, role, "access", ttl)
}

// GenerateRefreshToken 签发 refresh token（默认 7 天）
func GenerateRefreshToken(userID int64, role string) (string, error) {
	ttl := web.AppConfig.DefaultInt("jwtrt_ttl", 604800)
	return generateToken(userID, role, "refresh", ttl)
}

// ParseToken 解析并校验令牌
func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		// 强制 HMAC 签名算法，防 alg:none 攻击
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return jwtSecret(), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
