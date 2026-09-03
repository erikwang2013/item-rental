// 物品控制器纯函数单元测试：无 ORM/搜索/基建依赖，可离线运行。
package controllers

import (
	"testing"

	"github.com/erikwang2013/item-rental/server/search"
)

func TestBuildSearchParams(t *testing.T) {
	got := buildSearchParams("相机", 5, 10, 200, "price_asc", 3, 50, "北京")
	want := search.SearchParams{
		Query:      "相机",
		CategoryID: 5,
		MinPrice:   10,
		MaxPrice:   200,
		OrderBy:    "price_asc",
		Page:       3,
		PageSize:   50,
		City:       "北京",
	}
	if got != want {
		t.Errorf("buildSearchParams = %+v, 期望 %+v", got, want)
	}

	// 零值参数（未传）原样透传，由 search.BuildSearchQuery 兜底默认值
	got = buildSearchParams("", 0, 0, 0, "", 0, 0, "")
	want = search.SearchParams{Page: 0, PageSize: 0}
	if got != want {
		t.Errorf("零值透传失败: %+v", got)
	}
}

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

func TestValueOrZero(t *testing.T) {
	if got := valueOrZero(nil); got != 0 {
		t.Errorf("valueOrZero(nil) = %v, 期望 0", got)
	}
	v := 88.5
	if got := valueOrZero(&v); got != 88.5 {
		t.Errorf("valueOrZero(&88.5) = %v, 期望 88.5", got)
	}
	z := 0.0
	if got := valueOrZero(&z); got != 0 {
		t.Errorf("valueOrZero(&0) = %v, 期望 0", got)
	}
}
