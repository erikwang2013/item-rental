// 站内消息发送服务：写入 messages 表，不发送短信/邮件/推送
package services

import (
	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/models"
)

// Send 向 userID 写入一条站内消息。
// typ 建议值：payment_success / payment_refunded / return_confirmed / breach / order_cancelled
func Send(userID int64, typ, title, content string) error {
	o := orm.NewOrm()
	msg := &models.Message{
		Id:      NextID(),
		UserID:  userID,
		Type:    typ,
		Title:   title,
		Content: content,
		Read:    false,
	}
	_, err := o.Insert(msg)
	return err
}

// CountUnread 统计用户未读消息数
func CountUnread(userID int64) (int64, error) {
	o := orm.NewOrm()
	return o.QueryTable(new(models.Message)).Filter("user_id", userID).Filter("is_read", false).Count()
}
