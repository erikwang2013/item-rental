// 支付配置校验测试：真实模式必须齐全，mock 模式跳过
package payments

import (
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	full := Config{AppID: "wx1", MchID: "m1", MchKey: "k1", NotifyURL: "https://example.com/notify"}

	if err := ValidateConfig(full); err != nil {
		t.Errorf("完整配置应通过校验, err = %v", err)
	}
	if err := ValidateConfig(Config{Mock: true}); err != nil {
		t.Errorf("mock 模式应跳过校验, err = %v", err)
	}

	// 只配 AppID：错误必须列出其余三个缺失键，且不列已配置的键
	err := ValidateConfig(Config{AppID: "wx1"})
	if err == nil {
		t.Fatal("缺三项配置应返回错误")
	}
	for _, key := range []string{"wechat_mchid", "wechat_mchkey", "wechat_notify_url"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("错误应列出缺失键 %s, got: %v", key, err)
		}
	}
	if strings.Contains(err.Error(), "wechat_appid") {
		t.Errorf("已配置的键不应列为缺失, got: %v", err)
	}

	// 空白字符视为缺失（避免用空白密钥签名）
	blank := full
	blank.MchKey = "   "
	if err := ValidateConfig(blank); err == nil || !strings.Contains(err.Error(), "wechat_mchkey") {
		t.Errorf("空白 MchKey 应按缺失报错, err = %v", err)
	}
}
