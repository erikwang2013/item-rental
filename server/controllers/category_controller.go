// 品类控制器：公开读取品类列表
package controllers

import (
	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/models"
)

// CategoryController 品类相关接口
type CategoryController struct {
	BaseController
}

// List 获取启用状态的品类列表
// GET /api/v1/categories
func (c *CategoryController) List() {
	o := orm.NewOrm()
	var list []models.Category
	_, err := o.QueryTable(new(models.Category)).
		Filter("status", 1).
		OrderBy("sort").
		All(&list)
	if err != nil {
		c.Fail(500, "查询品类失败")
		return
	}
	c.OK(list)
}
