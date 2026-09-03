// 订单控制器：创建订单 + 订单列表/详情（需登录，仅本人可见）
package controllers

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/payments"
	"github.com/erikwang2013/item-rental/server/services"
)

// OrderController 订单相关接口
type OrderController struct {
	BaseController
}

// createOrderRequest 创建订单请求
type createOrderRequest struct {
	ItemId    string `json:"item_id"`
	StartDate string `json:"start_date"` // YYYY-MM-DD
	EndDate   string `json:"end_date"`   // YYYY-MM-DD
}

// Create 创建订单（需登录，当前用户为租客）
// POST /api/v1/orders  {"item_id":1,"start_date":"2026-09-01","end_date":"2026-09-03"}
func (c *OrderController) Create() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	var req createOrderRequest
	if err := c.Ctx.BindJSON(&req); err != nil {
		c.Fail(400, "请求参数错误")
		return
	}

	// item_id 为 snowflake 字符串(JS 安全);解析失败按参数错误处理
	itemID, err := strconv.ParseInt(req.ItemId, 10, 64)
	if err != nil {
		c.Fail(400, "参数错误")
		return
	}
	o := orm.NewOrm()
	item := models.Item{Id: itemID}
	if err := o.Read(&item); err != nil {
		c.Fail(404, "物品不存在")
		return
	}

	order, err := services.BuildOrder(&item, uid, req.StartDate, req.EndDate)
	if err != nil {
		c.Fail(400, err.Error())
		return
	}
	if _, err := o.Insert(order); err != nil {
		c.Fail(500, "创建订单失败")
		return
	}
	c.OK(order)
}

// List 订单列表（当前用户作为租客或房东，分页 + 可选状态过滤）
// GET /api/v1/orders?page=1&page_size=20&status=0
func (c *OrderController) List() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	page, _ := c.GetInt("page", 1)
	pageSize, _ := c.GetInt("page_size", 20)
	status, _ := c.GetInt("status", -1)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	o := orm.NewOrm()
	qs := o.QueryTable(new(models.Order)).
		SetCond(orm.NewCondition().Or("renter_id", uid).Or("owner_id", uid))
	if status >= 0 {
		qs = qs.Filter("status", status)
	}

	total, _ := qs.Count()
	var orders []models.Order
	if _, err := qs.OrderBy("-id").Limit(pageSize, (page-1)*pageSize).All(&orders); err != nil {
		c.Fail(500, "查询订单失败")
		return
	}
	c.OK(map[string]any{"orders": orders, "total": total, "page": page})
}

// Detail 订单详情（仅订单的租客或房东可见）
// GET /api/v1/orders/:id
func (c *OrderController) Detail() {
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

	o := orm.NewOrm()
	order := models.Order{Id: id}
	if err := o.Read(&order); err != nil {
		c.Fail(404, "订单不存在")
		return
	}
	if order.RenterId != uid && order.OwnerId != uid {
		c.Fail(403, "无权查看该订单")
		return
	}
	// 富化双方公开信息(昵称/头像/信用分),客户端按自身角色读取对方;缺失容错为 nil
	for _, u := range []struct {
		id     int64
		target **models.UserPublic
	}{{order.OwnerId, &order.Owner}, {order.RenterId, &order.Renter}} {
		user := models.User{Id: u.id}
		if err := o.Read(&user); err == nil {
			pub := user.ToPublic()
			*u.target = &pub
		} else if err != orm.ErrNoRows {
			c.Fail(500, "查询订单失败")
			return
		}
	}
	c.OK(order)
}

// flowHandler 订单流转的统一处理：鉴权 + 参数 + 调用流转服务 + 错误映射
type flowHandler func(st services.OrderStore, orderID, uid int64) (bool, error)

func (c *OrderController) runFlow(fn flowHandler, okMsg string) {
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
	_, err := fn(services.NewOrderStore(), id, uid)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrOrderNotFound):
			c.Fail(404, "订单不存在")
		case errors.Is(err, services.ErrForbidden):
			c.Fail(403, "无权操作该订单")
		case errors.Is(err, services.ErrInvalidState):
			c.Fail(409, "当前订单状态不允许该操作")
		default:
			c.Fail(500, "操作失败")
		}
		return
	}
	c.OK(map[string]string{"msg": okMsg})
}

// Pickup 取货（1→2 租赁中，冻结押金）租客操作
// POST /api/v1/orders/:id/pickup
func (c *OrderController) Pickup() {
	c.runFlow(services.Pickup, "取货成功，押金已冻结")
}

// ReturnRequest 申请归还（2→3 待归还）租客操作
// POST /api/v1/orders/:id/return_request
func (c *OrderController) ReturnRequest() {
	c.runFlow(services.RequestReturn, "已申请归还，等待房东确认")
}

// ReturnConfirm 确认归还（3→4 已归还结算，解冻押金）房东操作
// POST /api/v1/orders/:id/return_confirm
func (c *OrderController) ReturnConfirm() {
	c.runFlow(services.ConfirmReturn, "已确认归还，押金已退还")
}

// Breach 违约（3→6 违约扣押金）房东操作
// POST /api/v1/orders/:id/breach
func (c *OrderController) Breach() {
	c.runFlow(services.Breach, "已判定违约，押金扣除")
}

// cancelRequest 取消订单请求
type cancelRequest struct {
	CancelReason string `json:"cancel_reason"` // 取消原因（可选）
}

// Cancel 取消订单（需登录，仅订单租客或房东可操作）。
// POST /api/v1/orders/:id/cancel
// 仅 0待支付 状态可取消；已取消幂等成功；其他状态 409 拒绝。
// 流程：身份/状态校验 -> 先退款既有支付单（mock 直接成功）-> 条件更新 0→5 + 取消原因
func (c *OrderController) Cancel() {
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
	var req cancelRequest
	_ = c.Ctx.BindJSON(&req) // cancel_reason 可选，解析失败不阻塞

	o := orm.NewOrm()
	order := &models.Order{Id: id}
	if err := o.Read(order); err != nil {
		c.Fail(404, "订单不存在")
		return
	}
	if order.RenterId != uid && order.OwnerId != uid {
		c.Fail(403, "无权操作该订单")
		return
	}
	// 状态守卫（先于退款，避免对不可取消订单误退款）
	switch order.Status {
	case models.OrderStatusCancelled:
		c.OK(map[string]string{"msg": "订单已取消"}) // 幂等
		return
	case models.OrderStatusPending:
		// 可取消，继续
	default:
		c.Fail(409, "当前订单状态不允许取消")
		return
	}

	// 先退款既有支付单（存在已成功支付单才退款，mock 直接成功）
	if err := c.refundOrderPayment(o, order.Id); err != nil {
		c.Fail(500, err.Error())
		return
	}

	// 条件更新 0→5（幂等，原子迁移由 services.CancelOrder 保证）
	transitioned, err := services.CancelOrder(services.NewCancelStore(), id, uid, req.CancelReason)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrOrderNotFound):
			c.Fail(404, "订单不存在")
		case errors.Is(err, services.ErrForbidden):
			c.Fail(403, "无权操作该订单")
		case errors.Is(err, services.ErrInvalidState):
			c.Fail(409, "当前订单状态不允许取消")
		default:
			c.Fail(500, "取消订单失败")
		}
		return
	}
	if !transitioned {
		c.Fail(409, "当前订单状态不允许取消")
		return
	}
	c.OK(map[string]string{"msg": "订单已取消"})
}

// refundOrderPayment 若订单存在已成功支付单，则先退款（mock 直接成功）并标记退款。
// 无支付单或未支付成功时跳过（幂等安全）。
func (c *OrderController) refundOrderPayment(o orm.Ormer, orderID int64) error {
	var pay models.Payment
	if err := o.QueryTable(new(models.Payment)).
		Filter("order_id", orderID).
		OrderBy("-id").
		One(&pay); err != nil {
		return nil // 无支付单，无需退款
	}
	if !pay.IsPaid() {
		return nil // 未支付成功，无需退款
	}

	refundNo := genRefundNo(orderID)
	gw := payments.DefaultGateway()
	if _, err := gw.Refund(payments.RefundReq{
		OutTradeNo: pay.OutTradeNo,
		TotalFee:   pay.Amount,
		RefundFee:  pay.Amount,
		RefundNo:   refundNo,
	}); err != nil {
		return fmt.Errorf("退款失败: %s", err.Error())
	}
	return payments.NewOrderService().MarkRefunded(pay.OutTradeNo)
}
