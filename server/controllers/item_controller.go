// 物品控制器：物品 CRUD 与搜索
package controllers

import (
	"context"
	"log"

	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/search"
	"github.com/erikwang2013/item-rental/server/services"
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
	City       string   `json:"city"`
	Lat        *float64 `json:"lat"`
	Lng        *float64 `json:"lng"`
}

// valueOrZero 取指针值，nil（请求未传）返回 0。
func valueOrZero(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// buildSearchParams 组装搜索参数（纯函数，便于单测：页面入参 → search.SearchParams）。
func buildSearchParams(q string, categoryID int64, minPrice, maxPrice float64, orderBy string, page, pageSize int, city string) search.SearchParams {
	return search.SearchParams{
		Query:      q,
		CategoryID: categoryID,
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		OrderBy:    orderBy,
		Page:       page,
		PageSize:   pageSize,
		City:       city,
	}
}

// resolveItemOwnerScope 解析 owner_id 查询语义（纯函数，便于单测）。
// ownerID<=0 → 公开模式，返回 (0,0)；
// ownerID>0（「我的物品」视图）→ 需登录且为本人才返回 (ownerID,0)，否则返回 (0,401/403)。
func resolveItemOwnerScope(ownerID int64, authed bool, uid int64) (int64, int) {
	if ownerID <= 0 {
		return 0, 0
	}
	if !authed {
		return 0, 401
	}
	if uid != ownerID {
		return 0, 403
	}
	return ownerID, 0
}

// List 物品列表（分页 + 品类过滤）。
// 公开视图（无 owner_id）：仅上架；owner 视图（owner_id=本人 uid）：「我的物品」，含下架。
// GET /api/v1/items?page=1&page_size=20&category_id=1
// GET /api/v1/items?owner_id=<uid>&page=1&page_size=20
func (c *ItemController) List() {
	page, _ := c.GetInt("page", 1)
	pageSize, _ := c.GetInt("page_size", 20)
	categoryID, _ := c.GetInt64("category_id", 0)
	ownerID, _ := c.GetInt64("owner_id", 0)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	// owner 视图需登录：GET /items 公开挂载,JWTAuth 过滤器不会执行,
	// 带 owner_id 时在 controller 内显式鉴权(未带/无效 token → 401 Abort)。
	if ownerID > 0 {
		middleware.JWTAuth(c.Ctx)
	}
	uid, authed := middleware.GetUserID(c.Ctx)
	scopeOwner, code := resolveItemOwnerScope(ownerID, authed, uid)
	if code != 0 {
		if code == 401 {
			c.Fail(401, "未登录")
		} else {
			c.Fail(403, "无权查看他人物品")
		}
		return
	}

	o := orm.NewOrm()
	qs := o.QueryTable(new(models.Item))
	if scopeOwner > 0 {
		qs = qs.Filter("owner_id", scopeOwner)
	} else {
		qs = qs.Filter("status", 1)
	}
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
	// 富化房东公开信息(昵称/头像/信用分);owner 行缺失(孤儿数据)容错为 nil 不 500
	owner := models.User{Id: item.OwnerId}
	if err := o.Read(&owner); err == nil {
		pub := owner.ToPublic()
		item.Owner = &pub
	} else if err != orm.ErrNoRows {
		c.Fail(500, "查询物品失败")
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
	if err := services.ValidateItem(services.ItemValidateRequest{
		Title:      req.Title,
		Images:     req.Images,
		DailyPrice: req.DailyPrice,
		Deposit:    req.Deposit,
		Stock:      req.Stock,
		Lat:        req.Lat,
		Lng:        req.Lng,
	}); err != nil {
		c.Fail(400, err.Error())
		return
	}

	o := orm.NewOrm()
	if err := o.Read(&models.Category{Id: req.CategoryId}); err != nil {
		c.Fail(400, "品类不存在")
		return
	}

	item := models.Item{
		Id:         services.NextID(),
		OwnerId:    uid,
		CategoryId: req.CategoryId,
		Title:      req.Title,
		Desc:       req.Desc,
		Images:     req.Images,
		DailyPrice: req.DailyPrice,
		Deposit:    req.Deposit,
		Stock:      req.Stock,
		City:       req.City,
		Lat:        valueOrZero(req.Lat),
		Lng:        valueOrZero(req.Lng),
		Status:     1, // 上架
	}

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
	// 仅请求显式传了 lat/lng 才覆写；同时只校验本次传入的坐标
	var latP, lngP *float64
	if req.Lat != nil {
		item.Lat = *req.Lat
		latP = req.Lat
	}
	if req.Lng != nil {
		item.Lng = *req.Lng
		lngP = req.Lng
	}
	if err := services.ValidateItem(services.ItemValidateRequest{
		Title:      item.Title,
		Images:     item.Images,
		DailyPrice: item.DailyPrice,
		Deposit:    item.Deposit,
		Stock:      item.Stock,
		Lat:        latP,
		Lng:        lngP,
	}); err != nil {
		c.Fail(400, err.Error())
		return
	}

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

// Search 搜索物品（关键字/品类/价格区间/排序/城市/地理半径）
// GET /api/v1/items/search?q=相机&category_id=5&min_price=10&max_price=100&order_by=price_asc&city=北京&lat=39.9&lng=116.4&radius_km=50
func (c *ItemController) Search() {
	q := c.GetString("q")
	categoryID, _ := c.GetInt64("category_id", 0)
	minPrice, _ := c.GetFloat("min_price", 0)
	maxPrice, _ := c.GetFloat("max_price", 0)
	orderBy := c.GetString("order_by", "")
	city := c.GetString("city")
	page, _ := c.GetInt("page", 1)
	pageSize, _ := c.GetInt("page_size", 20)

	p := buildSearchParams(q, categoryID, minPrice, maxPrice, orderBy, page, pageSize, city)
	// 地理半径过滤（可选）：lat/lng/radius_km 齐全时启用，缺失坐标的物品安全跳过
	lat, _ := c.GetFloat("lat", 0)
	lng, _ := c.GetFloat("lng", 0)
	if radiusKm, _ := c.GetFloat("radius_km", 0); radiusKm > 0 {
		p.Lat = lat
		p.Lng = lng
		p.RadiusKm = &radiusKm
	}

	result, err := search.SearchItems(context.Background(), p)
	if err != nil {
		c.Fail(500, "搜索失败: "+err.Error())
		return
	}
	c.OK(result)
}
