// 用户模型（映射 users 表）
package models

import "time"

// User 用户表结构
type User struct {
	Id          int64     `orm:"column(id);auto" json:"id"`
	Phone       string    `orm:"column(phone);size(20);unique" json:"-"`
	Nickname    string    `orm:"column(nickname);size(64)" json:"nickname"`
	Avatar      string    `orm:"column(avatar);size(255)" json:"avatar"`
	RealName    string    `orm:"column(real_name);size(64)" json:"real_name"`
	CreditScore int       `orm:"column(credit_score)" json:"credit_score"`
	DepositBal  float64   `orm:"column(deposit_bal);digits(12);decimals(2)" json:"deposit_bal"`
	Status      int       `orm:"column(status)" json:"status"`
	CreatedAt   time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt   time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

// TableName 指定表名
func (u *User) TableName() string {
	return "users"
}
