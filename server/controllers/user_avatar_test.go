// 头像旧文件清理纯函数单元测试：无 DB/文件系统依赖，可离线运行
package controllers

import "testing"

func TestLocalAvatarFileName(t *testing.T) {
	const own = "http://x/static/uploads/avatars/42_123.png"
	cases := []struct {
		name   string
		uid    int64
		oldURL string
		newURL string
		want   string
	}{
		{"own_absolute", 42, own, "http://x/static/uploads/avatars/42_456.png", "42_123.png"},
		{"own_relative", 42, "/static/uploads/avatars/42_9.webp", "", "42_9.webp"},
		{"other_user", 42, "http://x/static/uploads/avatars/43_123.png", "", ""},
		{"uid_prefix_only", 42, "http://x/static/uploads/avatars/420_1.png", "", ""},
		{"external_url", 42, "https://cdn.example.com/42_123.png", "", ""},
		{"same_url", 42, own, own, ""},
		{"contains_slash", 42, "http://x/static/uploads/avatars/42_1/2.png", "", ""},
		{"contains_backslash", 42, `http://x/static/uploads/avatars/42_1\2.png`, "", ""},
		{"dot", 42, "http://x/static/uploads/avatars/.", "", ""},
		{"dotdot", 42, "http://x/static/uploads/avatars/..", "", ""},
	}
	for _, tc := range cases {
		if got := localAvatarFileName(tc.uid, tc.oldURL, tc.newURL); got != tc.want {
			t.Errorf("%s: localAvatarFileName(%d,%q,%q) = %q, 期望 %q",
				tc.name, tc.uid, tc.oldURL, tc.newURL, got, tc.want)
		}
	}
}
