// 租赁物品模型（映射 items 表）
package models

import (
	"time"
)

// Item 租赁物品表结构
type Item struct {
	Id         int64     `orm:"column(id)" json:"id,string"`
	OwnerId    int64     `orm:"column(owner_id)" json:"owner_id,string"`
	CategoryId int64     `orm:"column(category_id)" json:"category_id"`
	Title      string    `orm:"column(title);size(128)" json:"title"`
	Desc       string    `orm:"column(desc);type(text)" json:"desc"`
	Images     string    `orm:"column(images);type(text)" json:"images"`
	DailyPrice float64   `orm:"column(daily_price);digits(12);decimals(2)" json:"daily_price"`
	Stock      int       `orm:"column(stock)" json:"stock"`
	Status     int       `orm:"column(status)" json:"status"` // 1上架 0下架 2已售罄
	City       string    `orm:"column(city);size(64)" json:"city"`
	CreatedAt  time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt  time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
	// Owner 房东公开信息；仅详情接口填充（orm 不持久化），列表为空不输出
	Owner *UserPublic `orm:"-" json:"owner,omitempty"`
}

// TableName 指定表名
func (i *Item) TableName() string {
	return "items"
}

// IsOnShelf 是否上架可租
func (i *Item) IsOnShelf() bool {
	return i.Status == 1
}
