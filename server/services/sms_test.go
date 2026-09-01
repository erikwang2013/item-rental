// 短信验证码服务单元测试：mock 模式（无 Redis/无网络）
package services

import "testing"

// 测试环境无 conf/app.conf，sms_provider 走 FakeConfig 默认值 "mock"，
// 因此两端代码不触碰 Redis，可离线运行。
func TestVerifySmsCodeMock(t *testing.T) {
	if !VerifySmsCode("13800138000", devFixedCode) {
		t.Error("固定验证码应校验通过")
	}
	for _, bad := range []string{"", "000000", "12345"} {
		if VerifySmsCode("13800138000", bad) {
			t.Errorf("验证码 %q 应校验失败", bad)
		}
	}
}

func TestSaveSmsCodeMock(t *testing.T) {
	if err := SaveSmsCode("13800138000", devFixedCode); err != nil {
		t.Errorf("mock 模式保存验证码应成功, got %v", err)
	}
}