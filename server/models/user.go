// 用户模型（映射 users 表）
package models

import "time"

// User 用户表结构
type User struct {
	Id          int64     `orm:"column(id)" json:"id,string"`
	Phone       string    `orm:"column(phone);size(64);unique" json:"-"` // sha256(hex)，非明文
	Nickname    string    `orm:"column(nickname);size(64)" json:"nickname"`
	Avatar      string    `orm:"column(avatar);size(255)" json:"avatar"`
	RealName    string    `orm:"column(real_name);size(255)" json:"real_name"` // AES-GCM base64，非明文
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

// UserPublic 对外公开的用户精简视图（物品/订单详情中的 owner/renter 信息）。
// 仅暴露昵称/头像/信用分，严禁携带 phone/real_name/deposit_bal/status 等 PII 或私密字段。
type UserPublic struct {
	Id          int64  `json:"id,string"`
	Nickname    string `json:"nickname"`
	Avatar      string `json:"avatar"`
	CreditScore int    `json:"credit_score"`
}

// ToPublic 将完整用户裁剪为公开视图(纯投影,值接收便于字面量调用)。
func (u User) ToPublic() UserPublic {
	return UserPublic{Id: u.Id, Nickname: u.Nickname, Avatar: u.Avatar, CreditScore: u.CreditScore}
}
