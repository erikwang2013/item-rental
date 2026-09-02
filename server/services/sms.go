// 短信验证码服务：Redis 存储 + TTL，开发环境固定验证码
package services

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/redis/go-redis/v9"
)

var (
	redisOnce sync.Once
	rdb       *redis.Client
	ctx       = context.Background()
)

const (
	smsPrefix     = "sms:code:"
	smsTTL        = 5 * time.Minute // 验证码有效期 5 分钟
	devFixedCode  = "123456"        // 开发环境固定验证码
	smsLimitPrefix = "sms:limit:"   // 发送频率限制 key 前缀
	smsLimitTTL   = 60 * time.Second // 限频窗口：1 分钟 1 条
)

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

// GenerateSmsCode 生成 6 位随机数字验证码（crypto/rand，不可预测，不回显给客户端）。
func GenerateSmsCode() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// 理论上不会失败；极端回退保证接口可用
		return devFixedCode
	}
	return fmt.Sprintf("%d", 100000+binary.BigEndian.Uint32(b)%900000)
}

// CheckSmsRateLimit 发送频率限制：1 分钟 1 条（仅 real 模式，走 Redis；mock 模式不限制）。
// 返回 (true, nil) 表示允许发送；(false, nil) 表示触发限频；(false, err) 表示 Redis 异常。
func CheckSmsRateLimit(phone string) (bool, error) {
	if web.AppConfig.DefaultString("sms_provider", "mock") == "mock" {
		return true, nil
	}
	c := initRedis()
	key := smsLimitPrefix + phone
	n, err := c.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		// 首次触发：设置 60s 过期，窗口后自动清零
		c.Expire(ctx, key, smsLimitTTL)
	}
	return n <= 1, nil
}

// SaveSmsCode 保存验证码到 Redis。
func SaveSmsCode(phone, code string) error {
	if web.AppConfig.DefaultString("sms_provider", "mock") == "mock" {
		// 开发环境直接成功，不依赖 Redis
		return nil
	}
	return initRedis().Set(ctx, smsPrefix+phone, code, smsTTL).Err()
}

// VerifySmsCode 校验验证码（开发环境固定 123456）。
func VerifySmsCode(phone, code string) bool {
	if web.AppConfig.DefaultString("sms_provider", "mock") == "mock" {
		return code == devFixedCode
	}
	c := initRedis()
	key := smsPrefix + phone
	got, err := c.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	if got != code {
		return false
	}
	// 一次性使用：原子删除，避免并发下重复消费；删除失败则视为未消费（可重试）
	if err := c.Del(ctx, key).Err(); err != nil {
		return false
	}
	return true
}
