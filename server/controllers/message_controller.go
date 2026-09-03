// 站内消息控制器：消息列表 + 标记已读
package controllers

import (
	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/services"
)

// MessageController 站内消息接口
type MessageController struct {
	BaseController
}

// List 消息列表（仅当前用户可见，默认最新优先，支持 unread 过滤 + 分页）
// GET /api/v1/messages?unread=1&page=1&page_size=20
func (c *MessageController) List() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	page, _ := c.GetInt("page", 1)
	pageSize, _ := c.GetInt("page_size", 20)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	unread, _ := c.GetInt("unread", -1)

	o := orm.NewOrm()
	qs := o.QueryTable(new(models.Message)).Filter("user_id", uid)
	if unread == 1 {
		qs = qs.Filter("is_read", false)
	}

	total, _ := qs.Count()
	var msgs []models.Message
	if _, err := qs.OrderBy("-id").Limit(pageSize, (page-1)*pageSize).All(&msgs); err != nil {
		c.Fail(500, "查询失败")
		return
	}

	unreadCount, _ := services.CountUnread(uid)
	c.OK(map[string]any{"messages": msgs, "total": total, "page": page, "unread": unreadCount})
}

// MarkRead 标记消息已读（仅当前用户自己的消息可操作）
// POST /api/v1/messages/:id/read
func (c *MessageController) MarkRead() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}
	id, _ := c.GetInt64(":id")
	if id <= 0 {
		c.Fail(400, "参数错误")
		return
	}

	o := orm.NewOrm()
	msg := models.Message{Id: id}
	if err := o.Read(&msg); err != nil {
		c.Fail(404, "消息不存在")
		return
	}
	if msg.UserID != uid {
		c.Fail(403, "无权操作该消息")
		return
	}
	msg.Read = true
	if _, err := o.Update(&msg, "is_read"); err != nil {
		c.Fail(500, "更新失败")
		return
	}
	c.OK(map[string]string{"msg": "已读"})
}