// 短信验证码服务：Redis 存储 + TTL，开发环境固定验证码
package services

import (
	"context"
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
	smsPrefix    = "sms:code:"
	smsTTL       = 5 * time.Minute // 验证码有效期 5 分钟
	devFixedCode = "123456"        // 开发环境固定验证码
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
