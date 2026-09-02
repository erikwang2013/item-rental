// JWT 鉴权中间件：校验 access token，将用户信息注入请求上下文
package middleware

import (
	"strings"

	"github.com/beego/beego/v2/server/web/context"
)

// JWTAuth 校验 Authorization: Bearer <token>，失败返回 401。
// 成功后通过 ctx.Input.SetData("uid", ...) 向后续 handler 传递用户信息。
func JWTAuth(ctx *context.Context) {
	authHeader := ctx.Input.Header("Authorization")
	if authHeader == "" {
		unauthorized(ctx, "缺少认证信息")
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		unauthorized(ctx, "认证格式错误")
		return
	}

	claims, err := ParseToken(strings.TrimSpace(parts[1]))
	if err != nil {
		unauthorized(ctx, "令牌无效或已过期")
		return
	}
	// 仅接受 access token
	if claims.TokenTyp != "access" {
		unauthorized(ctx, "令牌类型错误")
		return
	}

	ctx.Input.SetData("uid", claims.UserID)
	ctx.Input.SetData("role", claims.Role)
}

// GetUserID 从请求上下文取当前登录用户 ID
func GetUserID(ctx *context.Context) (int64, bool) {
	if v, ok := ctx.Input.GetData("uid").(int64); ok {
		return v, true
	}
	return 0, false
}

// GetRole 从请求上下文取当前用户角色
func GetRole(ctx *context.Context) string {
	if v, ok := ctx.Input.GetData("role").(string); ok {
		return v
	}
	return ""
}

func unauthorized(ctx *context.Context, msg string) {
	ctx.Output.SetStatus(401)
	_ = ctx.Output.JSON(map[string]any{"code": 401, "msg": msg}, false, false)
	ctx.Abort(401, "Unauthorized")
}
