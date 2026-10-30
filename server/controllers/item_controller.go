// 物品控制器：物品 CRUD
package controllers

import (
	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
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
	Stock      int     `json:"stock"`
	City       string  `json:"city"`
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
	// 下架物品仅本人可见；对外统一 404，避免泄露物品存在性
	if item.Status != 1 {
		if uid, ok := middleware.OptionalUserID(c.Ctx); !ok || uid != item.OwnerId {
			c.Fail(404, "物品不存在")
			return
		}
	}
	// 富化房东公开信息(昵称/头像);owner 行缺失(孤儿数据)容错为 nil 不 500
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
		Desc:       req.Desc,
		Images:     req.Images,
		DailyPrice: req.DailyPrice,
		Stock:      req.Stock,
		City:       req.City,
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
		Stock:      req.Stock,
		City:       req.City,
		Status:     1, // 上架
	}

	if _, err := o.Insert(&item); err != nil {
		c.Fail(500, "发布失败")
		return
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

	// 部分更新语义：仅非空字段覆盖，避免 PUT {"title":"x"} 清空描述/图片
	if req.Title != "" {
		item.Title = req.Title
	}
	if req.Desc != "" {
		item.Desc = req.Desc
	}
	if req.Images != "" {
		item.Images = req.Images
	}
	if req.DailyPrice > 0 {
		item.DailyPrice = req.DailyPrice
	}
	if req.Stock > 0 {
		item.Stock = req.Stock
	}
	if req.City != "" {
		item.City = req.City
	}
	if err := services.ValidateItem(services.ItemValidateRequest{
		Title:      item.Title,
		Desc:       item.Desc,
		Images:     item.Images,
		DailyPrice: item.DailyPrice,
		Stock:      item.Stock,
		City:       item.City,
	}); err != nil {
		c.Fail(400, err.Error())
		return
	}

	if _, err := o.Update(&item, "title", "desc", "images", "daily_price", "stock", "city"); err != nil {
		c.Fail(500, "更新失败")
		return
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
	c.OK(map[string]any{"msg": "已下架"})
}
