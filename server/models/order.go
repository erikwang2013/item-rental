// 订单模型（映射 orders 表）
package models

import "time"

// Order 订单表结构。
// 订单状态机（见 schema 注释）：
//
//	0待支付 1待取(已付租金) 2租赁中(押金冻结) 3待归还 4已归还(结算) 5已取消 6违约(扣押金)
type Order struct {
	Id           int64     `orm:"column(id);auto" json:"id"`
	OrderNo      string    `orm:"column(order_no);size(64);unique" json:"order_no"` // 订单号
	ItemId       int64     `orm:"column(item_id)" json:"item_id"`
	RenterId     int64     `orm:"column(renter_id)" json:"renter_id"` // 租客用户ID
	OwnerId      int64     `orm:"column(owner_id)" json:"owner_id"`   // 房东用户ID
	StartDate    string    `orm:"column(start_date);type(date)" json:"start_date"`
	EndDate      string    `orm:"column(end_date);type(date)" json:"end_date"`
	Days         int       `orm:"column(days)" json:"days"` // 租赁天数
	RentAmount   float64   `orm:"column(rent_amount);digits(12);decimals(2)" json:"rent_amount"`
	Deposit      float64   `orm:"column(deposit);digits(12);decimals(2)" json:"deposit"`
	Status       int       `orm:"column(status)" json:"status"` // 状态机见注释
	PayTradeNo   string    `orm:"column(pay_trade_no);size(64)" json:"pay_trade_no"`
	CancelReason string    `orm:"column(cancel_reason);size(255)" json:"cancel_reason"`
	CreatedAt    time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt    time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

// TableName 指定表名
func (o *Order) TableName() string {
	return "orders"
}

// 订单状态常量（与 schema 注释对应）
const (
	OrderStatusPending   = 0 // 待支付
	OrderStatusToPickup  = 1 // 待取（已付租金）
	OrderStatusRenting   = 2 // 租赁中（押金冻结）
	OrderStatusToReturn  = 3 // 待归还
	OrderStatusReturned  = 4 // 已归还（结算）
	OrderStatusCancelled = 5 // 已取消
	OrderStatusBreach    = 6 // 违约（扣押金）
)
