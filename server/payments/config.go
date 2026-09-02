// 支付网关配置：从 app.conf / 环境变量读取微信支付参数
package payments

import (
	"os"

	"github.com/beego/beego/v2/server/web"
)

// Config 微信支付网关配置
type Config struct {
	// Mock 是否启用 mock 模式（无真实商户号时的本地测试）
	Mock bool
	// AppID 小程序/公众号 AppID
	AppID string
	// MchID 商户号
	MchID string
	// MchKey 商户 API 密钥（用于签名）
	MchKey string
	// NotifyURL 支付结果回调地址
	NotifyURL string
	// SignType 签名算法：MD5 或 HMAC-SHA256
	SignType string
	// Timeout 下单 HTTP 超时（秒）
	Timeout int
	// CertFile 商户证书文件路径（退款等需双向 TLS 的接口）
	CertFile string
	// CertKey 商户证书私钥文件路径
	CertKey string
}

// LoadConfig 读取支付配置。支持环境变量覆盖（与 JWT 密钥读取风格一致）。
// WECHAT_MOCK=1 时启用 mock 模式。
func LoadConfig() Config {
	cfg := Config{
		Mock:      web.AppConfig.DefaultString("wechat_mock", "") == "1" || os.Getenv("WECHAT_MOCK") == "1",
		AppID:     web.AppConfig.DefaultString("wechat_appid", ""),
		MchID:     web.AppConfig.DefaultString("wechat_mchid", ""),
		MchKey:    web.AppConfig.DefaultString("wechat_mchkey", ""),
		NotifyURL: web.AppConfig.DefaultString("wechat_notify_url", ""),
		SignType:  web.AppConfig.DefaultString("wechat_sign_type", "HMAC-SHA256"),
		Timeout:   int(web.AppConfig.DefaultInt("wechat_timeout", 10)),
		CertFile:  web.AppConfig.DefaultString("wechat_cert_file", ""),
		CertKey:   web.AppConfig.DefaultString("wechat_cert_key", ""),
	}
	if env := os.Getenv("WECHAT_APPID"); env != "" {
		cfg.AppID = env
	}
	if env := os.Getenv("WECHAT_MCHID"); env != "" {
		cfg.MchID = env
	}
	if env := os.Getenv("WECHAT_MCHKEY"); env != "" {
		cfg.MchKey = env
	}
	if env := os.Getenv("WECHAT_NOTIFY_URL"); env != "" {
		cfg.NotifyURL = env
	}
	if env := os.Getenv("WECHAT_CERT_FILE"); env != "" {
		cfg.CertFile = env
	}
	if env := os.Getenv("WECHAT_CERT_KEY"); env != "" {
		cfg.CertKey = env
	}
	return cfg
}
