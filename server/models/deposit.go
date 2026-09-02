// 押金台账模型（映射 deposits 表）
package models

import (
	"time"
)

// Deposit 押金流水：type 1冻结 2解冻 3扣款
type Deposit struct {
	Id        int64     `orm:"column(id);auto" json:"id"`
	OrderId   int64     `orm:"column(order_id)" json:"order_id"`
	UserId    int64     `orm:"column(user_id)" json:"user_id"`
	Amount    float64   `orm:"column(amount);digits(12);decimals(2)" json:"amount"`
	Type      int       `orm:"column(type)" json:"type"` // 1冻结 2解冻 3扣款
	Ref       string    `orm:"column(ref);size(64)" json:"ref"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

// TableName 指定表名
func (d *Deposit) TableName() string {
	return "deposits"
}

// 押金流水类型
const (
	DepositTypeFreeze   = 1 // 冻结（取货时押金冻结）
	DepositTypeUnfreeze = 2 // 解冻（确认归还后退还）
	DepositTypeDeduct   = 3 // 扣款（违约扣押金）
)
