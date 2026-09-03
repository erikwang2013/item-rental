// 认证控制器：手机号验证码登录 / 令牌刷新
package controllers

import (
	"errors"

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

// logoutRequest 登出请求（可选：带 refresh_token 则仅撤销该端会话）
type logoutRequest struct {
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
	if !isValidPhone(req.Phone) {
		c.Fail(400, "手机号格式不正确")
		return
	}

	// 发送频率限制：1 分钟 1 条（real 模式走 Redis，mock 模式不限制）
	allowed, err := services.CheckSmsRateLimit(req.Phone)
	if err != nil {
		c.Fail(500, "验证码发送失败")
		return
	}
	if !allowed {
		c.Fail(429, "发送过于频繁，请稍后再试")
		return
	}

	// 生成随机验证码并保存（real 模式存 Redis）；响应/日志绝不回显验证码
	// 开发环境（mock）校验仍接受固定 123456
	code := services.GenerateSmsCode()
	if err := services.SaveSmsCode(req.Phone, code); err != nil {
		c.Fail(500, "验证码发送失败")
		return
	}

	c.OK(map[string]any{"msg": "验证码已发送"})
}

// isValidPhone 手机号校验：11 位数字且以 1 开头
func isValidPhone(phone string) bool {
	if len(phone) != 11 || phone[0] != '1' {
		return false
	}
	for _, ch := range phone {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
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

	// 手机号落库/查询一律用单向哈希（PII 保护；原始号仅在本请求内存中）
	phoneHash := services.PhoneHash(req.Phone)
	o := orm.NewOrm()
	user := models.User{Phone: phoneHash}
	err := o.Read(&user, "phone")

	if err == orm.ErrNoRows {
		// 自动注册（昵称取末 4 位用原始明文，先取后覆写）
		user = models.User{Id: services.NextID(), Phone: phoneHash, Nickname: "用户" + req.Phone[len(req.Phone)-4:], Status: 1, CreditScore: 100}
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

	// 响应前解密实名（real_name 密文存储，AES-GCM）
	realName, err := services.DecryptRealName(user.RealName)
	if err != nil {
		c.Fail(500, "数据处理失败")
		return
	}
	user.RealName = realName

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
	// 注册为当前会话：单活跃会话，重新登录/刷新后旧 refresh 立即失效
	if err := services.SaveRefreshSession(user.Id, refreshToken); err != nil {
		c.Fail(500, "令牌生成失败")
		return
	}

	c.OK(map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

// Refresh 使用 refresh token 轮换换取新的 access + refresh token
// POST /api/v1/auth/refresh  {"refresh_token":"..."}
// 轮换语义：旧 refresh token 一次性使用，刷新后立即失效（单活跃会话）。
func (c *AuthController) Refresh() {
	var req refreshRequest
	if err := c.Ctx.BindJSON(&req); err != nil || req.RefreshToken == "" {
		c.Fail(400, "refresh_token 不能为空")
		return
	}

	newAccess, newRefresh, err := services.RotateRefresh(req.RefreshToken)
	if err != nil {
		if errors.Is(err, services.ErrRefreshRejected) {
			c.Fail(401, "refresh_token 无效、已过期或已被轮换")
		} else {
			c.Fail(500, "令牌生成失败")
		}
		return
	}
	c.OK(map[string]any{"access_token": newAccess, "refresh_token": newRefresh})
}

// Logout 登出。POST /api/v1/auth/logout (JWT)
// 请求体可选 {"refresh_token": "..."}：带且能解析为本用户 refresh → 仅撤销该端会话；
// 缺失/解析失败/uid 不匹配 → 撤销该用户全部会话（登出永不失败）。
func (c *AuthController) Logout() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}
	var req logoutRequest
	_ = c.Ctx.BindJSON(&req)
	if req.RefreshToken != "" {
		if claims, err := middleware.ParseToken(req.RefreshToken); err == nil &&
			claims.TokenTyp == "refresh" && claims.UserID == uid {
			if err := services.Logout(uid, claims.ID); err != nil {
				c.Fail(500, "登出失败")
				return
			}
			c.OK(nil)
			return
		}
	}
	if err := services.LogoutAll(uid); err != nil {
		c.Fail(500, "登出失败")
		return
	}
	c.OK(nil)
}
