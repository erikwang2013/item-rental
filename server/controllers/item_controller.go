// 物品控制器：物品 CRUD 与搜索
package controllers

import (
	"context"
	"log"

	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/search"
)

// ItemController 物品相关接口
type ItemController struct {
	BaseController
}

// createItemRequest 发布/更新物品请求
type createItemRequest struct {
	CategoryId int64   `json:"category_id"`
	Title      string  `json:"title"`
	Desc       string  `json:"desc"`
	Images     string  `json:"images"`
	DailyPrice float64 `json:"daily_price"`
	Deposit    float64 `json:"deposit"`
	Stock      int     `json:"stock"`
	City       string  `json:"city"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
}

// List 物品列表（分页 + 品类过滤，仅上架）
// GET /api/v1/items?page=1&page_size=20&category_id=1
func (c *ItemController) List() {
	page, _ := c.GetInt("page", 1)
	pageSize, _ := c.GetInt("page_size", 20)
	categoryID, _ := c.GetInt64("category_id", 0)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	o := orm.NewOrm()
	qs := o.QueryTable(new(models.Item)).Filter("status", 1)
	if categoryID > 0 {
		qs = qs.Filter("category_id", categoryID)
	}

	total, _ := qs.Count()
	var items []models.Item
	if _, err := qs.OrderBy("-id").Limit(pageSize, (page-1)*pageSize).All(&items); err != nil {
		c.Fail(500, "查询物品失败")
		return
	}
	c.OK(map[string]any{"items": items, "total": total, "page": page})
}

// Detail 物品详情
// GET /api/v1/items/:id
func (c *ItemController) Detail() {
	id, _ := c.GetInt64(":id")
	if id <= 0 {
		c.Fail(400, "参数错误")
		return
	}
	o := orm.NewOrm()
	item := models.Item{Id: id}
	if err := o.Read(&item); err != nil {
		c.Fail(404, "物品不存在")
		return
	}
	c.OK(item)
}

// Create 发布物品（需登录）
// POST /api/v1/items  {...,"title":"...","daily_price":50,...}
func (c *ItemController) Create() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	var req createItemRequest
	if err := c.Ctx.BindJSON(&req); err != nil {
		c.Fail(400, "请求参数错误")
		return
	}
	if req.Title == "" || req.DailyPrice <= 0 {
		c.Fail(400, "标题和租金不能为空")
		return
	}

	item := models.Item{
		OwnerId:    uid,
		CategoryId: req.CategoryId,
		Title:      req.Title,
		Desc:       req.Desc,
		Images:     req.Images,
		DailyPrice: req.DailyPrice,
		Deposit:    req.Deposit,
		Stock:      req.Stock,
		City:       req.City,
		Lat:        req.Lat,
		Lng:        req.Lng,
		Status:     1, // 上架
	}
	if item.Stock <= 0 {
		item.Stock = 1
	}

	o := orm.NewOrm()
	if _, err := o.Insert(&item); err != nil {
		c.Fail(500, "发布失败")
		return
	}

	// 同步到搜索索引（失败不影响主流程，记录日志即可）
	if err := search.SyncItem(context.Background(), &item); err != nil {
		log.Printf("[search] 同步索引失败 item_id=%d: %v", item.Id, err)
	}

	c.OK(item)
}

// Update 更新物品（需登录，仅本人可改）
// PUT /api/v1/items/:id
func (c *ItemController) Update() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}
	id, _ := c.GetInt64(":id")
	if id <= 0 {
		c.Fail(400, "参数错误")
		return
	}

	var req createItemRequest
	if err := c.Ctx.BindJSON(&req); err != nil {
		c.Fail(400, "请求参数错误")
		return
	}

	o := orm.NewOrm()
	item := models.Item{Id: id}
	if err := o.Read(&item); err != nil {
		c.Fail(404, "物品不存在")
		return
	}
	if item.OwnerId != uid {
		c.Fail(403, "无权修改该物品")
		return
	}

	if req.Title != "" {
		item.Title = req.Title
	}
	item.Desc = req.Desc
	item.Images = req.Images
	if req.DailyPrice > 0 {
		item.DailyPrice = req.DailyPrice
	}
	if req.Deposit > 0 {
		item.Deposit = req.Deposit
	}
	if req.Stock > 0 {
		item.Stock = req.Stock
	}
	if req.City != "" {
		item.City = req.City
	}
	item.Lat = req.Lat
	item.Lng = req.Lng

	if _, err := o.Update(&item, "title", "desc", "images", "daily_price", "deposit", "stock", "city", "lat", "lng"); err != nil {
		c.Fail(500, "更新失败")
		return
	}
	if err := search.SyncItem(context.Background(), &item); err != nil {
		log.Printf("[search] 同步索引失败 item_id=%d: %v", item.Id, err)
	}
	c.OK(item)
}

// OffShelf 下架物品（需登录，仅本人可改）
// POST /api/v1/items/:id/offshelf
func (c *ItemController) OffShelf() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}
	id, _ := c.GetInt64(":id")
	o := orm.NewOrm()
	item := models.Item{Id: id}
	if err := o.Read(&item); err != nil {
		c.Fail(404, "物品不存在")
		return
	}
	if item.OwnerId != uid {
		c.Fail(403, "无权操作该物品")
		return
	}
	item.Status = 0
	if _, err := o.Update(&item, "status"); err != nil {
		c.Fail(500, "下架失败")
		return
	}
	// 从索引移除
	if err := search.RemoveItem(context.Background(), &item); err != nil {
		log.Printf("[search] 移除索引失败 item_id=%d: %v", item.Id, err)
	}
	c.OK(map[string]any{"msg": "已下架"})
}

// Search 搜索物品（关键字/品类/价格区间/排序）
// GET /api/v1/items/search?q=相机&category_id=5&min_price=10&max_price=100&order_by=price_asc
func (c *ItemController) Search() {
	q := c.GetString("q")
	categoryID, _ := c.GetInt64("category_id", 0)
	minPrice, _ := c.GetFloat("min_price", 0)
	maxPrice, _ := c.GetFloat("max_price", 0)
	orderBy := c.GetString("order_by", "")
	page, _ := c.GetInt("page", 1)
	pageSize, _ := c.GetInt("page_size", 20)

	result, err := search.SearchItems(context.Background(), search.SearchParams{
		Query:      q,
		CategoryID: categoryID,
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		OrderBy:    orderBy,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		c.Fail(500, "搜索失败: "+err.Error())
		return
	}
	c.OK(result)
}
