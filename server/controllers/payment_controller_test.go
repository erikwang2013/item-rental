// 支付控制器纯函数单元测试：待支付单复用条件（无 ORM 依赖）。
package controllers

import (
	"testing"
	"time"
)

func TestCanReusePending(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		channel    string
		createdAt  time.Time
		reqChannel string
		want       bool
	}{
		{"同渠道且未满 2 小时可复用", "native", now.Add(-1 * time.Hour), "native", true},
		{"刚好 2 小时不可复用", "native", now.Add(-2 * time.Hour), "native", false},
		{"超过 2 小时不可复用", "native", now.Add(-3 * time.Hour), "native", false},
		{"换渠道不可复用", "native", now.Add(-time.Minute), "jsapi", false},
		{"历史 wechat 渠道不可复用", "wechat", now.Add(-time.Minute), "native", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := canReusePending(c.channel, c.createdAt, now, c.reqChannel); got != c.want {
				t.Errorf("canReusePending(%q, %v, now, %q) = %v, 期望 %v",
					c.channel, c.createdAt, c.reqChannel, got, c.want)
			}
		})
	}
}
