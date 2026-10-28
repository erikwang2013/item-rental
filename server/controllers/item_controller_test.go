// 物品控制器纯函数单元测试：无 ORM 依赖，可离线运行。
package controllers

import "testing"

func TestResolveItemOwnerScope(t *testing.T) {
	cases := []struct {
		name     string
		ownerID  int64
		authed   bool
		uid      int64
		wantID   int64
		wantCode int
	}{
		{"public_no_param", 0, false, 0, 0, 0},
		{"mine_no_token", 5, false, 0, 0, 401},
		{"mine_other_user", 5, true, 3, 0, 403},
		{"mine_self", 5, true, 5, 5, 0},
	}
	for _, tc := range cases {
		gotID, gotCode := resolveItemOwnerScope(tc.ownerID, tc.authed, tc.uid)
		if gotID != tc.wantID || gotCode != tc.wantCode {
			t.Errorf("%s: resolveItemOwnerScope(%d,%v,%d) = (%d,%d), 期望 (%d,%d)",
				tc.name, tc.ownerID, tc.authed, tc.uid, gotID, gotCode, tc.wantID, tc.wantCode)
		}
	}
}
