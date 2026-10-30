// 认证控制器纯函数单元测试：无 ORM/网络依赖。
package controllers

import "testing"

func TestLoginAllowed(t *testing.T) {
	cases := []struct {
		name    string
		runmode string
		env     string
		want    bool
	}{
		{"prod_default_deny", "prod", "", false},
		{"prod_env_zero", "prod", "0", false},
		{"prod_env_explicit_on", "prod", "1", true},
		{"dev_default_allow", "dev", "", true},
		{"dev_env_on", "dev", "1", true},
		{"test_mode_deny", "test", "", false},
		{"empty_runmode_deny", "", "", false},
	}
	for _, tc := range cases {
		if got := loginAllowed(tc.runmode, tc.env); got != tc.want {
			t.Errorf("%s: loginAllowed(%q,%q) = %v, 期望 %v", tc.name, tc.runmode, tc.env, got, tc.want)
		}
	}
}
