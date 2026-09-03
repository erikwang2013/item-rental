// Refresh token 轮换与会话管理：多端并存会话（Redis 按用户存 per-jti 会话集合）
package services

import (
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/erikwang2013/item-rental/server/middleware"
)

const (
	refreshPrefix = "auth:refresh:"    // auth:refresh:{uid} 存储 hash {jti: exp}，每端一个字段
	refreshTTL    = 7 * 24 * time.Hour // 与 refresh token 有效期一致
)

// ErrRefreshRejected refresh token 校验不通过（无效/过期/已被轮换）
var ErrRefreshRejected = errors.New("refresh token rejected")

// RefreshTokenStore 描述 refresh token 会话存储。
// 默认实现走 Redis；测试可替换为内存 stub（见 refresh_test.go）。
type RefreshTokenStore interface {
	// Save 记录该用户一个有效 refresh token（jti 标识）；多端并存互不覆盖。
	Save(uid int64, jti string, exp time.Time) error
	// Check 校验 jti 是否为该用户当前有效 refresh token 之一。
	// Redis 故障时降级放行（返回 true）；会话不存在返回 false。
	Check(uid int64, jti string) bool
	// Delete 删除该用户单个 refresh 会话（单端登出）。
	Delete(uid int64, jti string) error
	// DeleteAll 删除该用户全部 refresh 会话（登出所有端）。
	DeleteAll(uid int64) error
}

// refreshStore 全局会话存储实例，测试中可整体替换。
var refreshStore RefreshTokenStore = redisRefreshStore{}

// redisRefreshStore 基于 Redis hash 的实现：auth:refresh:{uid} -> {<jti>: <exp>}
type redisRefreshStore struct{}

func (redisRefreshStore) Save(uid int64, jti string, exp time.Time) error {
	if web.AppConfig.DefaultString("sms_provider", "mock") == "mock" {
		return nil // mock 模式不依赖 Redis
	}
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

func (redisRefreshStore) Check(uid int64, jti string) bool {
	if web.AppConfig.DefaultString("sms_provider", "mock") == "mock" {
		return true // mock 模式放行
	}
	key := refreshPrefix + strconv.FormatInt(uid, 10)
	exists, err := initRedis().HExists(ctx, key, jti).Result()
	if err != nil {
		// Redis 故障：优雅降级放行
		log.Printf("refresh: check failed uid=%d: %v", uid, err)
		return true
	}
	return exists
}

func (redisRefreshStore) Delete(uid int64, jti string) error {
	if web.AppConfig.DefaultString("sms_provider", "mock") == "mock" {
		return nil
	}
	c := initRedis()
	return c.HDel(ctx, refreshPrefix+strconv.FormatInt(uid, 10), jti).Err()
}

func (redisRefreshStore) DeleteAll(uid int64) error {
	if web.AppConfig.DefaultString("sms_provider", "mock") == "mock" {
		return nil
	}
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
//  2. 会话比对：仅仍有效的 jti 放行（已轮换/登出过的 token 拒绝）；
//  3. 消费 presented jti（从会话集移除，防重放），签发新的 access + refresh 并注册新 jti。
//     其他端的会话不受影响。
func RotateRefresh(presented string) (newAccess, newRefresh string, err error) {
	claims, err := middleware.ParseToken(presented)
	if err != nil || claims.TokenTyp != "refresh" || claims.ID == "" {
		return "", "", ErrRefreshRejected
	}
	if !refreshStore.Check(claims.UserID, claims.ID) {
		return "", "", ErrRefreshRejected
	}

	// 消费本次使用的 jti（该 refresh 一次性）；仅删本字段，他端会话保留
	if err := refreshStore.Delete(claims.UserID, claims.ID); err != nil {
		return "", "", err
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
