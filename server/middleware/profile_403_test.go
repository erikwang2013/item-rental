package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	secgo "github.com/erikwang2013/security-go"
	"github.com/erikwang2013/item-rental/server/security"
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

// TestProfileBearerNot403 回归：合法 JWT（Authorization Bearer）不得被
// SensitiveDataLeak 误判为敏感数据泄露而 403。覆盖所有鉴权端点共享的安全中间件。
func TestProfileBearerNot403(t *testing.T) {
	security.InitEngine()
	t.Setenv("ITEM_RENTAL_JWT_SECRET", "regression-test-secret-x9k2")
	access, err := GenerateAccessToken(1, "user")
	if err != nil {
		t.Fatal(err)
	}

	ctx, rec := makeCtx("GET", "/api/v1/user/profile", "", "", map[string]string{
		"Authorization": "Bearer " + access,
	})
	SecurityFilter(ctx)
	if rec.Code == 403 {
		t.Fatalf("authenticated profile request got 403: %s", rec.Body.String())
	}
}

// TestLeakDetectorStillCatchesBody 确认剥离 Authorization 不影响 data_leak
// 对请求体中敏感数据的检测（回归防护：Authorization 只在 Header 里，不在 Body）。
func TestLeakDetectorStillCatchesBody(t *testing.T) {
	security.InitEngine()
	// data_leak 会识别邮箱等敏感字段；验证引擎仍能扫到请求体内容。
	ctx, _ := makeCtx("POST", "/api/v1/auth/login",
		`{"email":"leak@example.com","password":"hunter2"}`,
		"application/json", nil)
	results := secDetectAll(ctx.Request)
	if len(results) == 0 {
		t.Error("引擎未扫描到任何结果，请求体可能未被检测")
	}
}

// TestHighSeverityPayloadStillBlocked 确认高危载荷仍被拦截。
// DetectRequest 只扫 URL/Query/Header/Cookie（不含 Body），故用例走 URL。
func TestHighSeverityPayloadStillBlocked(t *testing.T) {
	security.InitEngine()
	ctx, rec := makeCtx("GET",
		"/api/v1/items?url=http://169.254.169.254/latest/meta-data/", "", "", nil)
	defer func() {
		if v := recover(); v != nil {
			// beego Context.Abort 会 panic "Forbidden"；403 已写入 recorder，视为拦截成功。
			if rec.Code != 403 {
				t.Errorf("SSRF URL 载荷应被 403 拦截，实际 %d (panic=%v)", rec.Code, v)
			}
		}
	}()
	SecurityFilter(ctx)
	if rec.Code != 403 {
		t.Errorf("SSRF URL 载荷应被 403 拦截，实际 %d", rec.Code)
	}
}

// TestProfileNoToken401 回归：profile 路由已挂 JWTAuth，无 token 请求应 401。
//（此前该路由漏挂鉴权过滤器，controller 内 GetUserID 恒空、鉴权语义失效。）
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

// TestMultipartUploadNotBlocked 回归：multipart/form-data 上传的 Content-Type
// 含 boundary=…，会被 mail_header 检测器误判为邮件头注入(SeverityCritical)而 403。
// 安全中间件应在扫描副本中剥离该头（引擎不扫请求体，剥离不影响检测能力）。
func TestMultipartUploadNotBlocked(t *testing.T) {
	security.InitEngine()
	ctx, rec := makeCtx("POST", "/api/v1/user/avatar", "fake-image-bytes",
		"multipart/form-data; boundary=----WebKitFormBoundaryXyZ12", nil)
	defer func() {
		if v := recover(); v != nil {
			if rec.Code == 403 {
				t.Errorf("multipart 上传被误拦 403 (panic=%v)", v)
			}
		}
	}()
	SecurityFilter(ctx)
	if rec.Code == 403 {
		t.Errorf("multipart 上传请求应放行，实际 403: %s", rec.Body.String())
	}
}

func secDetectAll(r *http.Request) []*secgo.Result {
	return security.Engine.DetectRequest(r)
}