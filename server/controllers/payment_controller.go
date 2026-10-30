// 支付控制器：创建支付单 + 接收微信回调
package controllers

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/payments"
	"github.com/erikwang2013/item-rental/server/services"
)

// PaymentController 支付相关接口
type PaymentController struct {
	BaseController
}

// unifiedOrderRequest 创建支付单请求
type unifiedOrderRequest struct {
	OrderNo string `json:"order_no"` // 订单号
	Channel string `json:"channel"`  // native / jsapi
	OpenID  string `json:"openid"`   // JSAPI 支付必传
}

// UnifiedOrder 创建支付单并调用微信统一下单。
// POST /api/v1/pay/unifiedorder（需登录）
// 请求：{"order_no":"ORD...","channel":"native","openid":""}
func (c *PaymentController) UnifiedOrder() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	var req unifiedOrderRequest
	if err := c.Ctx.BindJSON(&req); err != nil || req.OrderNo == "" {
		c.Fail(400, "参数错误")
		return
	}
	if req.Channel == "" {
		req.Channel = payments.ChannelNative
	}
	if req.Channel != payments.ChannelNative && req.Channel != payments.ChannelJSAPI {
		c.Fail(400, "不支持的支付渠道")
		return
	}

	o := orm.NewOrm()

	// 查询订单，校验归属与可支付状态
	order := &models.Order{OrderNo: req.OrderNo}
	if err := o.Read(order, "order_no"); err != nil {
		c.Fail(404, "订单不存在")
		return
	}
	if order.RenterId != uid {
		c.Fail(403, "无权为该订单支付")
		return
	}
	if order.Status != models.OrderStatusPending {
		c.Fail(400, "订单状态不允许支付")
		return
	}

	// 退款抢占后订单为待支付但已有一笔成功支付：禁止再发起支付，
	// 否则用户二次扣款，后续退款只会处理最新一笔，旧成功单被遗留。
	if n, err := o.QueryTable(new(models.Payment)).
		Filter("order_id", order.Id).
		Filter("status", models.PaymentStatusSuccess).
		Count(); err != nil {
		c.Fail(500, "查询支付单失败")
		return
	} else if n > 0 {
		c.Fail(409, "订单已有成功支付，请重试退款或取消")
		return
	}

	// 幂等：复用 2 小时内同渠道的待支付单，避免重复下单；
	// 过期或换渠道时旧单置为失败并新建（微信预支付单已失效，复用同一 out_trade_no 无法支付）。
	var existing models.Payment
	var payID int64
	var outTradeNo string
	err := o.QueryTable(new(models.Payment)).
		Filter("order_id", order.Id).
		Filter("status", models.PaymentStatusPending).
		OrderBy("-id").
		One(&existing)
	if err == nil {
		if canReusePending(existing.Channel, existing.CreatedAt, time.Now(), req.Channel) {
			payID, outTradeNo = existing.Id, existing.OutTradeNo
		} else {
			// 过期/换渠道：旧待支付单置为失败，随后新建
			if _, err := o.QueryTable(new(models.Payment)).
				Filter("id", existing.Id).
				Filter("status", models.PaymentStatusPending).
				Update(map[string]any{"status": models.PaymentStatusFailed}); err != nil {
				c.Fail(500, "更新支付单失败")
				return
			}
		}
	} else if err != orm.ErrNoRows {
		// 真实查询错误不能当成"无待支付单"，否则会多建一张 pending 支付单
		c.Fail(500, "查询支付单失败")
		return
	}
	if outTradeNo == "" {
		// 新建支付单
		outTradeNo = genTradeNo(order.Id)
		pay := &models.Payment{
			Id:         services.NextID(),
			OrderId:    order.Id,
			OutTradeNo: outTradeNo,
			Channel:    req.Channel,
			Amount:     order.RentAmount,
			Status:     models.PaymentStatusPending,
		}
		if _, err := o.Insert(pay); err != nil {
			c.Fail(500, "创建支付单失败")
			return
		}
		payID = pay.Id
	}

	// 调用微信网关统一下单
	clientIP := c.Ctx.Input.IP()
	gw := payments.DefaultGateway()
	res, err := gw.CreatePrepay(payments.UnifiedOrderReq{
		OutTradeNo: outTradeNo,
		Body:       "租赁订单 " + req.OrderNo,
		Amount:     order.RentAmount,
		Channel:    req.Channel,
		OpenID:     req.OpenID,
		ClientIP:   clientIP,
	})
	if err != nil {
		c.Fail(500, "统一下单失败: "+err.Error())
		return
	}

	// mock 模式没有真实微信支付：服务端直接完成支付，
	// 前端沿用"发起支付后轮询订单状态"的逻辑即可（mock 回调已在 HandleNotify 拒绝）。
	if res.Mock {
		txID := "MOCK-" + outTradeNo
		// 条件更新：0 行说明支付单已成功，视为幂等成功
		if _, err := o.QueryTable(new(models.Payment)).
			Filter("id", payID).
			Filter("status", models.PaymentStatusPending).
			Update(map[string]any{
				"status":         models.PaymentStatusSuccess,
				"transaction_id": txID,
			}); err != nil {
			c.Fail(500, "mock 支付失败")
			return
		}
		if err := payments.NewOrderService().MarkPaid(order.OrderNo, txID); err != nil {
			c.Fail(500, "mock 支付失败: "+err.Error())
			return
		}
	}

	c.OK(map[string]any{
		"out_trade_no": outTradeNo,
		"amount":       order.RentAmount,
		"prepay":       res,
	})
}

// refundRequest 退款请求
// OrderId 为 snowflake 字符串形态(snowflake id > 2^53,JS 数字不安全,契约与 item_id 一致)
type refundRequest struct {
	OrderId      string `json:"order_id"`      // 订单ID(snowflake 字符串)
	RefundReason string `json:"refund_reason"` // 退款原因（可选）
}

// Refund 发起退款（需登录，仅订单租客或房东可操作）。
// POST /api/v1/pay/refund
// 请求：{"order_id":1,"refund_reason":"拍错了"}
// 流程：校验归属/支付状态 -> 抢占订单 1待取->0待支付（与取货互斥）-> 调用微信退款（mock 直接成功）
// -> 仅标记支付单已退款（订单保持待支付）
func (c *PaymentController) Refund() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	var req refundRequest
	if err := c.Ctx.BindJSON(&req); err != nil {
		c.Fail(400, "参数错误")
		return
	}
	orderId, err := strconv.ParseInt(req.OrderId, 10, 64)
	if err != nil || orderId <= 0 {
		c.Fail(400, "参数错误")
		return
	}

	o := orm.NewOrm()
	order := &models.Order{Id: orderId}
	if err := o.Read(order); err != nil {
		c.Fail(404, "订单不存在")
		return
	}
	if order.RenterId != uid && order.OwnerId != uid {
		c.Fail(403, "无权操作该订单")
		return
	}
	// 退款守卫：1待取先抢占回 0（与取货 1->2 互斥）；0待支付仅允许"微信已退款但
	// 标记支付单失败"的重试；租赁中/待归还/已归还/违约一律拒绝。
	// 支付单必须按成功状态筛选：换渠道后旧单可能被置 Failed 但用户实际已付款。
	// 订单已绑定支付流水时再按 transaction_id 过滤，确保退的是订单当前这笔支付。
	payQuery := o.QueryTable(new(models.Payment)).
		Filter("order_id", order.Id).
		Filter("status", models.PaymentStatusSuccess)
	if order.PayTradeNo != "" {
		payQuery = payQuery.Filter("transaction_id", order.PayTradeNo)
	}
	var pay models.Payment
	payErr := payQuery.OrderBy("-id").One(&pay)

	switch order.Status {
	case models.OrderStatusToPickup:
		if payErr != nil {
			c.Fail(404, "支付单不存在")
			return
		}
		// 调网关前先抢占 1->0：并发取货时只有一方条件更新成功，
		// 避免钱退了订单却已进入租赁中。
		update := map[string]any{"status": models.OrderStatusPending}
		if order.PayTradeNo == "" {
			// 历史数据回填支付流水：退款期间同 tx 的迟到回调才能判定为 Done，
			// 不会把订单重新标回已支付。
			update["pay_trade_no"] = pay.TransactionId
		}
		n, err := o.QueryTable(new(models.Order)).
			Filter("id", order.Id).
			Filter("status", models.OrderStatusToPickup).
			Update(update)
		if err != nil {
			c.Fail(500, "更新订单状态失败")
			return
		}
		if n == 0 {
			c.Fail(409, "订单状态已变化")
			return
		}
	case models.OrderStatusPending:
		// 上次退款成功后标记支付单失败的重试：订单已是待支付，无需抢占。
		// 必须确认退的就是订单当前支付流水，否则回调与退款并发会产生"待取+已退款"。
		if payErr != nil {
			c.Fail(404, "支付单不存在")
			return
		}
		if order.PayTradeNo == "" || order.PayTradeNo != pay.TransactionId {
			c.Fail(409, "订单支付流水未绑定，请稍后重试或改用取消")
			return
		}
	default:
		c.Fail(409, "当前订单状态不允许退款")
		return
	}

	// 调用微信退款（mock 模式直接成功）；退款单号按支付单固定，重试时微信幂等
	refundNo := payments.RefundNo(pay.OutTradeNo)
	gw := payments.DefaultGateway()
	res, err := gw.Refund(payments.RefundReq{
		OutTradeNo: pay.OutTradeNo,
		TotalFee:   pay.Amount,
		RefundFee:  pay.Amount,
		RefundNo:   refundNo,
	})
	if err != nil {
		// 不回滚 0->1：网关错误下退款结果不确定（可能已退款但响应丢失），
		// 回滚会让订单重新可 pickup，形成"已退款但可租赁"的状态。
		// 订单保持 0 即阻止取货；可直接重试本接口（RefundNo 固定、微信幂等），
		// 或走取消流程补退款。
		c.Fail(500, "退款失败，请重试: "+err.Error())
		return
	}

	// 退款成功：只标记支付单 1->3，绝不重置订单——订单已抢占为待支付，
	// 用 MarkRefunded 会把并发重新支付后的订单错误重置回 0。
	if err := payments.NewOrderService().MarkPaymentRefunded(pay.OutTradeNo); err != nil {
		c.Fail(500, "退款处理失败: "+err.Error())
		return
	}

	c.OK(map[string]any{
		"refund_id": res.RefundID,
		"refund_no": refundNo,
		"mock":      res.Mock,
	})
}

// Notify 接收微信支付回调（公开，内部做验签）。
// POST /api/v1/pay/notify
// 微信要求成功返回 <xml><return_code><![CDATA[SUCCESS]]></return_code></xml>，
// 失败返回 FAIL 以便微信重试。
func (c *PaymentController) Notify() {
	raw, err := io.ReadAll(c.Ctx.Request.Body)
	if err != nil {
		c.writeNotifyResult(false)
		return
	}

	gw := payments.DefaultGateway()
	if err := gw.HandleNotify(raw); err != nil {
		// 处理失败：返回 FAIL，微信稍后重试
		c.writeNotifyResult(false)
		return
	}
	// 处理成功：返回 SUCCESS，微信不再重发
	c.writeNotifyResult(true)
}

// writeNotifyResult 输出微信要求的 XML 应答。
func (c *PaymentController) writeNotifyResult(success bool) {
	code := "FAIL"
	if success {
		code = "SUCCESS"
	}
	c.Ctx.Output.Header("Content-Type", "text/xml; charset=utf-8")
	_ = c.Ctx.Output.Body([]byte("<xml><return_code><![CDATA[" + code + "]]></return_code></xml>"))
}

// genTradeNo 生成商户订单号：RENT + 时间戳毫秒 + 订单ID，保证唯一。
func genTradeNo(orderID int64) string {
	return fmt.Sprintf("RENT%d%s", time.Now().UnixMilli(), strconv.FormatInt(orderID, 10))
}

// canReusePending 判断待支付单能否复用：渠道一致且创建未满 2 小时。
// 微信预支付单有效期约 2 小时，过期或换渠道后必须换新 out_trade_no，否则用户无法支付。
func canReusePending(channel string, createdAt, now time.Time, reqChannel string) bool {
	return channel == reqChannel && now.Sub(createdAt) < 2*time.Hour
}
