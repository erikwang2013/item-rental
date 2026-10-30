// 订单控制器：创建订单 + 订单列表/详情（需登录，仅本人可见）
package controllers

import (
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"

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
	// 事务 + 物品行锁串行化同一物品的下单，避免并发超租
	tx, err := orm.NewOrm().Begin()
	if err != nil {
		c.Fail(500, "创建订单失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	item := models.Item{Id: itemID}
	if err := tx.ReadForUpdate(&item); err != nil {
		if err == orm.ErrNoRows {
			c.Fail(404, "物品不存在")
		} else {
			c.Fail(500, "创建订单失败")
		}
		return
	}

	order, err := services.BuildOrder(&item, uid, req.StartDate, req.EndDate)
	if err != nil {
		c.Fail(400, err.Error())
		return
	}

	// 查询日期重叠的既有订单区间：待支付单从创建即占用库存，
	// 超时单只有被后台 ticker 置为已取消后才释放，避免"过期但还能支付"造成超租。
	type dateRow struct {
		Sd string `orm:"column(sd)"`
		Ed string `orm:"column(ed)"`
	}
	var rows []dateRow
	if _, err := tx.Raw(
		`SELECT DATE_FORMAT(start_date,'%Y-%m-%d') AS sd, DATE_FORMAT(end_date,'%Y-%m-%d') AS ed
		 FROM orders WHERE item_id=? AND status IN (0,1,2,3) AND start_date<=? AND end_date>=?`,
		item.Id, order.EndDate, order.StartDate,
	).QueryRows(&rows); err != nil {
		c.Fail(500, "创建订单失败")
		return
	}
	existing := make([]services.DateRange, 0, len(rows))
	for _, r := range rows {
		rng, err := services.ParseDateRange(r.Sd, r.Ed)
		if err != nil {
			c.Fail(500, "创建订单失败")
			return
		}
		existing = append(existing, rng)
	}
	newRange, err := services.ParseDateRange(order.StartDate, order.EndDate)
	if err != nil {
		c.Fail(500, "创建订单失败")
		return
	}
	if err := services.CheckAvailability(newRange, existing, item.Stock); err != nil {
		c.Fail(409, err.Error())
		return
	}

	if _, err := tx.Insert(order); err != nil {
		c.Fail(500, "创建订单失败")
		return
	}
	if err := tx.Commit(); err != nil {
		c.Fail(500, "创建订单失败")
		return
	}
	committed = true
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
	// 富化双方公开信息(昵称/头像),客户端按自身角色读取对方;缺失容错为 nil
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

// Pickup 取货（1→2 租赁中）租客操作
// POST /api/v1/orders/:id/pickup
func (c *OrderController) Pickup() {
	c.runFlow(services.Pickup, "取货成功")
}

// ReturnRequest 申请归还（2→3 待归还）租客操作
// POST /api/v1/orders/:id/return_request
func (c *OrderController) ReturnRequest() {
	c.runFlow(services.RequestReturn, "已申请归还，等待房东确认")
}

// ReturnConfirm 确认归还（3→4 已归还）房东操作
// POST /api/v1/orders/:id/return_confirm
func (c *OrderController) ReturnConfirm() {
	c.runFlow(services.ConfirmReturn, "已确认归还")
}

// Breach 违约（3→6 违约）房东操作
// POST /api/v1/orders/:id/breach
func (c *OrderController) Breach() {
	c.runFlow(services.Breach, "已判定违约")
}

// cancelRequest 取消订单请求
type cancelRequest struct {
	CancelReason string `json:"cancel_reason"` // 取消原因（可选）
}

// Cancel 取消订单（需登录，仅订单租客或房东可操作）。
// POST /api/v1/orders/:id/cancel
// 仅 0待支付 状态可取消；已取消幂等成功；其他状态 409 拒绝。
// 顺序：先以条件更新抢占 0→5，成功或已取消后再补退款——避免并发重复退款，
// 也避免"钱退了订单却仍待取"。退款失败返回 500，用户重试取消会再次尝试补退款。
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
	if utf8.RuneCountInString(req.CancelReason) > 255 {
		c.Fail(400, "取消原因过长")
		return
	}

	// transitioned=false 且 err=nil 表示订单此前已取消（幂等），同样继续补退款
	if _, err := services.CancelOrder(services.NewCancelStore(), id, uid, req.CancelReason); err != nil {
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
	if err := refundPaidPayment(orm.NewOrm(), id); err != nil {
		c.Fail(500, err.Error())
		return
	}
	c.OK(map[string]string{"msg": "订单已取消"})
}

// refundPaidPayment 为订单补退款：取最新一条支付成功流水退款，成功后标记支付单已退款。
// 无成功支付单视为无需退款；DB 错误必须返回，避免静默漏退款。
func refundPaidPayment(o orm.Ormer, orderID int64) error {
	var pay models.Payment
	err := o.QueryTable(new(models.Payment)).
		Filter("order_id", orderID).
		Filter("status", models.PaymentStatusSuccess).
		OrderBy("-id").
		One(&pay)
	if err == orm.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("查询支付单失败: %s", err.Error())
	}

	if _, err := payments.DefaultGateway().Refund(payments.RefundReq{
		OutTradeNo: pay.OutTradeNo,
		TotalFee:   pay.Amount,
		RefundFee:  pay.Amount,
		RefundNo:   payments.RefundNo(pay.OutTradeNo),
	}); err != nil {
		return fmt.Errorf("退款失败: %s", err.Error())
	}
	return payments.NewOrderService().MarkPaymentRefunded(pay.OutTradeNo)
}
