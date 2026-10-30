// 认证控制器：手机号登录 / 令牌刷新
package controllers

import (
	"errors"
	"os"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/services"
)

// AuthController 认证相关接口
type AuthController struct {
	BaseController
}

// loginRequest 登录请求
type loginRequest struct {
	Phone string `json:"phone"`
}

// refreshRequest 刷新令牌请求
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// logoutRequest 登出请求（可选：带 refresh_token 则仅撤销该端会话）
type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
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

// loginAllowed 是否允许手机号 mock 登录：显式开启（env=1）或开发模式。
// 生产（prod）默认拒绝，避免“知道手机号即可登录”的账号接管。
func loginAllowed(runmode, env string) bool {
	if env == "1" {
		return true
	}
	return runmode == "dev"
}

// Login 手机号登录（mock/开发入口，无验证码），未注册自动注册。
// 生产环境默认关闭，需显式设置 ITEM_RENTAL_LOGIN_MOCK=1 或接入正式登录。
// POST /api/v1/auth/login  {"phone":"13800138000"}
func (c *AuthController) Login() {
	if !loginAllowed(web.BConfig.RunMode, os.Getenv("ITEM_RENTAL_LOGIN_MOCK")) {
		c.Fail(403, "手机号登录仅在开发模式开放，请接入正式登录或显式设置 ITEM_RENTAL_LOGIN_MOCK=1")
		return
	}
	var req loginRequest
	if err := c.Ctx.BindJSON(&req); err != nil || req.Phone == "" {
		c.Fail(400, "手机号不能为空")
		return
	}
	if !isValidPhone(req.Phone) {
		c.Fail(400, "手机号格式不正确")
		return
	}

	// 手机号落库/查询一律用带密钥哈希（PII 保护；原始号仅在本请求内存中）
	phoneHash := services.PhoneHash(req.Phone)
	o := orm.NewOrm()
	user := models.User{Phone: phoneHash}
	err := o.Read(&user, "phone")

	if err == orm.ErrNoRows {
		// 懒迁移历史无盐哈希：命中则回写为 HMAC 值后继续登录
		legacy := models.User{Phone: services.LegacyPhoneHash(req.Phone)}
		switch lerr := o.Read(&legacy, "phone"); lerr {
		case nil:
			legacy.Phone = phoneHash
			if _, uerr := o.Update(&legacy, "phone"); uerr != nil {
				c.Fail(500, "数据处理失败")
				return
			}
			user = legacy
			err = nil
		case orm.ErrNoRows:
			// 新旧哈希都未命中 → 走下面的自动注册
		default:
			c.Fail(500, "查询用户失败")
			return
		}
	}

	if err == orm.ErrNoRows {
		// 自动注册（昵称取末 4 位用原始明文，先取后覆写）
		user = models.User{Id: services.NextID(), Phone: phoneHash, Nickname: "用户" + req.Phone[len(req.Phone)-4:], Status: 1}
		if _, ierr := o.Insert(&user); ierr != nil {
			// 并发首次登录触发唯一索引冲突：按 HMAC 再读一次，读到且未封禁即继续登录
			existing := models.User{Phone: phoneHash}
			if rerr := o.Read(&existing, "phone"); rerr != nil || existing.Status != 1 {
				c.Fail(500, "注册失败")
				return
			}
			user = existing
		}
	} else if err != nil {
		c.Fail(500, "查询用户失败")
		return
	}

	if user.Status != 1 {
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
	// 注册为当前会话：多端并存，新登录/刷新不影响其他端已签发的 refresh
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
// 轮换语义：旧 refresh token 一次性使用，刷新后立即失效（各端会话相互独立）。
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
