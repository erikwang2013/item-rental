// 用户控制器：个人资料（需登录）
package controllers

import (
	"strings"
	"unicode/utf8"

	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/services"
)

// UserController 用户相关接口
type UserController struct {
	BaseController
}

// updateProfileRequest 更新资料请求
type updateProfileRequest struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	RealName string `json:"real_name"` // 明文入参，服务端加密落库
}

// Profile 获取当前登录用户资料
// GET /api/v1/user/profile
func (c *UserController) Profile() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	o := orm.NewOrm()
	user := models.User{Id: uid}
	if err := o.Read(&user); err != nil {
		c.Fail(404, "用户不存在")
		return
	}
	// 解密实名后再返回（real_name 密文存储）
	realName, err := services.DecryptRealName(user.RealName)
	if err != nil {
		c.Fail(500, "数据处理失败")
		return
	}
	user.RealName = realName
	c.OK(user)
}

// UpdateProfile 更新昵称/头像/实名
// PUT /api/v1/user/profile  {"nickname":"...","avatar":"...","real_name":"..."}
func (c *UserController) UpdateProfile() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	var req updateProfileRequest
	if err := c.Ctx.BindJSON(&req); err != nil {
		c.Fail(400, "请求参数错误")
		return
	}

	o := orm.NewOrm()
	user := models.User{Id: uid}
	if err := o.Read(&user); err != nil {
		c.Fail(404, "用户不存在")
		return
	}
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	// real_name：明文入参 → 加密落库（列内为密文）
	if req.RealName != "" {
		rn := strings.TrimSpace(req.RealName)
		if utf8.RuneCountInString(rn) > 32 {
			c.Fail(400, "姓名过长")
			return
		}
		enc, err := services.EncryptRealName(rn)
		if err != nil {
			c.Fail(500, "加密失败")
			return
		}
		user.RealName = enc
	}
	if _, err := o.Update(&user, "nickname", "avatar", "real_name"); err != nil {
		c.Fail(500, "更新失败")
		return
	}
	// 响应前解密（与 Profile 一致，避免把密文回给客户端）
	realName, err := services.DecryptRealName(user.RealName)
	if err != nil {
		c.Fail(500, "数据处理失败")
		return
	}
	user.RealName = realName
	c.OK(user)
}
