// 站内消息模型：支付成功/退款成功/归还确认/违约判定等通知
package models

import "time"

// Message 站内消息
type Message struct {
	Id        int64     `orm:"column(id)" json:"id,string"`
	UserID    int64     `orm:"column(user_id)" json:"user_id,string"`
	Type      string    `orm:"column(type);size(32)" json:"type"`
	Title     string    `orm:"column(title);size(128)" json:"title"`
	Content   string    `orm:"column(content);size(512)" json:"content"`
	Read      bool      `orm:"column(is_read);default(false)" json:"read"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (m *Message) TableName() string { return "messages" }
