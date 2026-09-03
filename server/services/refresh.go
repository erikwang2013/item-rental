// Refresh token 轮换与会话管理：单活跃会话（Redis 记录每用户当前有效 refresh token）
package services

import (
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/redis/go-redis/v9"
)

const (
	refreshPrefix     = "auth:refresh:"    // auth:refresh:{uid} 存储 hash {token: <jti>, exp: <ts>}
	refreshTTL        = 7 * 24 * time.Hour // 与 refresh token 有效期一致
	refreshFieldToken = "token"
	refreshFieldExp   = "exp"
)

// ErrRefreshRejected refresh token 校验不通过（无效/过期/已被轮换）
var ErrRefreshRejected = errors.New("refresh token rejected")

// RefreshTokenStore 描述 refresh token 会话存储。
// 默认实现走 Redis；测试可替换为内存 stub（见 refresh_test.go）。
type RefreshTokenStore interface {
	// Save 记录用户当前有效 refresh token（jti 标识），覆盖旧值使旧 refresh 立即失效。
	Save(uid int64, jti string, exp time.Time) error
	// Check 校验 jti 是否为该用户当前有效 refresh token。
	// Redis 故障时降级放行（返回 true）；会话不存在返回 false。
	Check(uid int64, jti string) bool
	// Delete 删除该用户的 refresh 会话（登出），使 refresh 立即失效。
	Delete(uid int64) error
}

// refreshStore 全局会话存储实例，测试中可整体替换。
var refreshStore RefreshTokenStore = redisRefreshStore{}

// redisRefreshStore 基于 Redis hash 的实现：auth:refresh:{uid} -> {token: jti, exp: ts}
type redisRefreshStore struct{}

func (redisRefreshStore) Save(uid int64, jti string, exp time.Time) error {
	if web.AppConfig.DefaultString("sms_provider", "mock") == "mock" {
		return nil // mock 模式不依赖 Redis
	}
	key := refreshPrefix + strconv.FormatInt(uid, 10)
	c := initRedis()
	if err := c.HSet(ctx, key, refreshFieldToken, jti, refreshFieldExp, exp.Unix()).Err(); err != nil {
		// Redis 不可用：优雅降级，不阻断登录/刷新（与项目现有降级风格一致）
		log.Printf("refresh: save failed uid=%d jti=%s: %v", uid, jti, err)
		return nil
	}
	c.Expire(ctx, key, refreshTTL)
	return nil
}

func (redisRefreshStore) Check(uid int64, jti string) bool {
	if web.AppConfig.DefaultString("sms_provider", "mock") == "mock" {
		return true // mock 模式放行
	}
	key := refreshPrefix + strconv.FormatInt(uid, 10)
	got, err := initRedis().HGet(ctx, key, refreshFieldToken).Result()
	if err == redis.Nil {
		return false // 会话不存在（旧 token / 未登录）：拒绝
	}
	if err != nil {
		// Redis 故障：优雅降级放行
		log.Printf("refresh: check failed uid=%d: %v", uid, err)
		return true
	}
	return got == jti
}

func (redisRefreshStore) Delete(uid int64) error {
	if web.AppConfig.DefaultString("sms_provider", "mock") == "mock" {
		return nil
	}
	c := initRedis()
	return c.Del(ctx, refreshPrefix+strconv.FormatInt(uid, 10)).Err()
}

// Logout 使该用户当前 refresh 会话失效（登出）。
func Logout(uid int64) error {
	return refreshStore.Delete(uid)
}

// SaveRefreshSession 将 refresh token 注册为该用户当前会话（写入其 jti）。
// 登录、轮换后调用；覆盖后旧的 refresh token 立即失效。
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

// RotateRefresh 轮换 refresh token：
//  1. 校验 presented token 签名/类型/有效期；
//  2. 单活跃会话比对：仅当前有效 token 放行（已轮换/旧 token 拒绝）；
//  3. 签发新的 access + refresh，并将新 refresh 写入会话（旧 refresh 立即失效）。
func RotateRefresh(presented string) (newAccess, newRefresh string, err error) {
	claims, err := middleware.ParseToken(presented)
	if err != nil || claims.TokenTyp != "refresh" || claims.ID == "" {
		return "", "", ErrRefreshRejected
	}
	if !refreshStore.Check(claims.UserID, claims.ID) {
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
