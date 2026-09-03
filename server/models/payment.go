// 支付流水模型（映射 payments 表）
package models

import "time"

// Payment 支付流水表结构。
// 状态常量见 PaymentStatus*：
//
//	0 待支付 1 成功 2 失败 3 已退款
type Payment struct {
	Id            int64     `orm:"column(id)" json:"id,string"`
	OrderId       int64     `orm:"column(order_id)" json:"order_id,string"`
	OutTradeNo    string    `orm:"column(out_trade_no);size(64);unique" json:"out_trade_no"` // 商户订单号
	TransactionId string    `orm:"column(transaction_id);size(64)" json:"transaction_id"`    // 微信支付单号
	Channel       string    `orm:"column(channel);size(16)" json:"channel"`                  // 支付渠道 wechat
	Amount        float64   `orm:"column(amount);digits(12);decimals(2)" json:"amount"`      // 支付金额
	Status        int       `orm:"column(status)" json:"status"`                             // 0待支付 1成功 2失败 3已退款
	RawCallback   string    `orm:"column(raw_callback);type(text)" json:"-"`                 // 微信回调原始报文
	CreatedAt     time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt     time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

// TableName 指定表名
func (p *Payment) TableName() string {
	return "payments"
}

// 支付状态常量
const (
	PaymentStatusPending  = 0 // 待支付
	PaymentStatusSuccess  = 1 // 成功
	PaymentStatusFailed   = 2 // 失败
	PaymentStatusRefunded = 3 // 已退款
)

// IsPaid 是否已支付成功
func (p *Payment) IsPaid() bool {
	return p.Status == PaymentStatusSuccess
}
