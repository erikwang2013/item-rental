// 租赁物品模型（映射 items 表）
package models

import (
	"strconv"
	"time"
)

// Item 租赁物品表结构
type Item struct {
	Id         int64     `orm:"column(id);auto" json:"id"`
	OwnerId    int64     `orm:"column(owner_id)" json:"owner_id"`
	CategoryId int64     `orm:"column(category_id)" json:"category_id"`
	Title      string    `orm:"column(title);size(128)" json:"title"`
	Desc       string    `orm:"column(desc);type(text)" json:"desc"`
	Images     string    `orm:"column(images);type(text)" json:"images"`
	DailyPrice float64   `orm:"column(daily_price);digits(12);decimals(2)" json:"daily_price"`
	Deposit    float64   `orm:"column(deposit);digits(12);decimals(2)" json:"deposit"`
	Stock      int       `orm:"column(stock)" json:"stock"`
	Status     int       `orm:"column(status)" json:"status"` // 1上架 0下架 2已售罄
	City       string    `orm:"column(city);size(64)" json:"city"`
	Lat        float64   `orm:"column(lat);digits(10);decimals(7)" json:"lat"`
	Lng        float64   `orm:"column(lng);digits(10);decimals(7)" json:"lng"`
	CreatedAt  time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt  time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

// TableName 指定表名
func (i *Item) TableName() string {
	return "items"
}

// IsOnShelf 是否上架可租
func (i *Item) IsOnShelf() bool {
	return i.Status == 1
}

// --- go-scout ScoutModel 接口实现（用于搜索索引同步） ---

// ScoutKey 返回主键（索引文档 ID）
func (i *Item) ScoutKey() any { return i.Id }

// ToSearchableArray 返回写入搜索索引的字段。
func (i *Item) ToSearchableArray() map[string]any {
	return map[string]any{
		"id":          i.Id,
		"owner_id":    i.OwnerId,
		"category_id": i.CategoryId,
		"title":       i.Title,
		"desc":        i.Desc,
		"daily_price": i.DailyPrice,
		"deposit":     i.Deposit,
		"stock":       i.Stock,
		"status":      i.Status,
		"city":        i.City,
		"lat":         i.Lat,
		"lng":         i.Lng,
		"created_at":  i.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ShouldBeSearchable 是否参与搜索索引（仅上架物品可被搜索）。
func (i *Item) ShouldBeSearchable() bool {
	return i.IsOnShelf()
}

// IDString 返回主键的字符串形式（便于断言回填）。
func (i *Item) IDString() string {
	return strconv.FormatInt(i.Id, 10)
}
