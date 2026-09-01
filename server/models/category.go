// 品类模型（映射 categories 表）
package models

import "time"

// Category 品类结构
type Category struct {
	Id        int64     `orm:"column(id);auto" json:"id"`
	Name      string    `orm:"column(name);size(64)" json:"name"`
	ParentId  int64     `orm:"column(parent_id)" json:"parent_id"`
	Icon      string    `orm:"column(icon);size(255)" json:"icon"`
	Sort      int       `orm:"column(sort)" json:"sort"`
	Status    int       `orm:"column(status)" json:"status"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

// TableName 指定表名
func (c *Category) TableName() string {
	return "categories"
}
