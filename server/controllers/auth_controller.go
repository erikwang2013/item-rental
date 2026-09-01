// 认证控制器：手机号验证码登录 / 令牌刷新
package controllers

import (
	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/services"
)

// AuthController 认证相关接口
type AuthController struct {
	BaseController
}

// smsRequest 发送验证码请求
type smsRequest struct {
	Phone string `json:"phone"`
}

// loginRequest 登录请求
type loginRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// refreshRequest 刷新令牌请求
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// SendSms 发送手机验证码
// POST /api/v1/auth/sms  {"phone":"13800138000"}
func (c *AuthController) SendSms() {
	var req smsRequest
	if err := c.Ctx.BindJSON(&req); err != nil || req.Phone == "" {
		c.Fail(400, "手机号不能为空")
		return
	}

	// 开发环境使用固定验证码 123456，不真正下发短信
	// 生产环境接入短信网关，验证码存 Redis 并设 TTL
	code := "123456"
	if err := services.SaveSmsCode(req.Phone, code); err != nil {
		c.Fail(500, "验证码发送失败")
		return
	}

	// 仅返回提示，不返回验证码（生产环境）
	c.OK(map[string]any{"msg": "验证码已发送（开发环境固定 123456）"})
}

// Login 手机号 + 验证码登录，未注册自动注册
// POST /api/v1/auth/login  {"phone":"13800138000","code":"123456"}
func (c *AuthController) Login() {
	var req loginRequest
	if err := c.Ctx.BindJSON(&req); err != nil || req.Phone == "" || req.Code == "" {
		c.Fail(400, "手机号和验证码不能为空")
		return
	}

	// 校验验证码（开发环境固定 123456）
	if !services.VerifySmsCode(req.Phone, req.Code) {
		c.Fail(401, "验证码错误")
		return
	}

	o := orm.NewOrm()
	user := models.User{Phone: req.Phone}
	err := o.Read(&user, "phone")

	if err == orm.ErrNoRows {
		// 自动注册
		user = models.User{Phone: req.Phone, Nickname: "用户" + req.Phone[len(req.Phone)-4:], Status: 1, CreditScore: 100}
		if _, err := o.Insert(&user); err != nil {
			c.Fail(500, "注册失败")
			return
		}
	} else if err != nil {
		c.Fail(500, "查询用户失败")
		return
	} else if user.Status != 1 {
		c.Fail(403, "账号已被禁用")
		return
	}

	accessToken, err := middleware.GenerateAccessToken(user.Id, "user")
	if err != nil {
		c.Fail(500, "令牌生成失败")
		return
	}
	refreshToken, err := middleware.GenerateRefreshToken(user.Id, "user")
	if err != nil {
		c.Fail(500, "令牌生成失败")
		return
	}

	c.OK(map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

// Refresh 使用 refresh token 换取新的 access token
// POST /api/v1/auth/refresh  {"refresh_token":"..."}
func (c *AuthController) Refresh() {
	var req refreshRequest
	if err := c.Ctx.BindJSON(&req); err != nil || req.RefreshToken == "" {
		c.Fail(400, "refresh_token 不能为空")
		return
	}

	claims, err := middleware.ParseToken(req.RefreshToken)
	if err != nil {
		c.Fail(401, "refresh_token 无效或已过期")
		return
	}
	if claims.TokenTyp != "refresh" {
		c.Fail(401, "令牌类型错误")
		return
	}

	accessToken, err := middleware.GenerateAccessToken(claims.UserID, claims.Role)
	if err != nil {
		c.Fail(500, "令牌生成失败")
		return
	}
	c.OK(map[string]any{"access_token": accessToken})
}
