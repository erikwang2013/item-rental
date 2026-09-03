// 信用分流水模型（映射 credit_events 表）
package models

import "time"

// CreditEvent 信用分变动流水（审计用）：正值加分、负值扣分
type CreditEvent struct {
	Id        int64     `orm:"column(id)" json:"id,string"`
	UserId    int64     `orm:"column(user_id)" json:"user_id,string"`
	Change    int       `orm:"column(change)" json:"change"`
	Reason    string    `orm:"column(reason);size(32)" json:"reason"` // return_on_time / breach / cancel_after_paid
	Ref       string    `orm:"column(ref);size(64)" json:"ref"`       // 关联订单号
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

// TableName 指定表名
func (c *CreditEvent) TableName() string {
	return "credit_events"
}
