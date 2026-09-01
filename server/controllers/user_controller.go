// 用户控制器：个人资料（需登录）
package controllers

import (
	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
)

// UserController 用户相关接口
type UserController struct {
	BaseController
}

// updateProfileRequest 更新资料请求
type updateProfileRequest struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
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
	c.OK(user)
}

// UpdateProfile 更新昵称/头像
// PUT /api/v1/user/profile  {"nickname":"...","avatar":"..."}
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
	if _, err := o.Update(&user, "nickname", "avatar"); err != nil {
		c.Fail(500, "更新失败")
		return
	}
	c.OK(user)
}
