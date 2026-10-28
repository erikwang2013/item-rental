package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

func init() { web.BConfig.RunMode = "test" }

func makeCtx(method, url, body, ct string, hdrs map[string]string) (*context.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	req.Header.Set("User-Agent", "curl/8.4.0")
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	ctx := context.NewContext()
	ctx.Reset(rec, req)
	return ctx, rec
}

// TestProfileNoToken401 回归：profile 路由已挂 JWTAuth，无 token 请求应 401。
// （此前该路由漏挂鉴权过滤器，controller 内 GetUserID 恒空、鉴权语义失效。）
func TestProfileNoToken401(t *testing.T) {
	t.Setenv("ITEM_RENTAL_JWT_SECRET", "regression-test-secret-x9k2")
	ctx, rec := makeCtx("GET", "/api/v1/user/profile", "", "", nil)
	defer func() {
		if v := recover(); v != nil {
			if rec.Code != 401 {
				t.Errorf("无 token 的 profile 请求应 401，实际 %d (panic=%v)", rec.Code, v)
			}
		}
	}()
	JWTAuth(ctx)
	if rec.Code != 401 {
		t.Errorf("无 token 的 profile 请求应 401，实际 %d", rec.Code)
	}
}

// TestProfileWithTokenOK 带合法 access token 时 JWTAuth 放行并注入 uid。
func TestProfileWithTokenOK(t *testing.T) {
	t.Setenv("ITEM_RENTAL_JWT_SECRET", "regression-test-secret-x9k2")
	access, err := GenerateAccessToken(1, "user")
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := makeCtx("GET", "/api/v1/user/profile", "", "", map[string]string{
		"Authorization": "Bearer " + access,
	})
	JWTAuth(ctx)
	uid, ok := GetUserID(ctx)
	if !ok || uid != 1 {
		t.Errorf("带 token 的 profile 请求应注入 uid=1，实际 (%d,%v)", uid, ok)
	}
}
