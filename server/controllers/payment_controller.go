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

	// 幂等：复用该订单已有的待支付支付单，避免重复下单
	var existing models.Payment
	err := o.QueryTable(new(models.Payment)).
		Filter("order_id", order.Id).
		Filter("status", models.PaymentStatusPending).
		OrderBy("-id").
		One(&existing)
	var outTradeNo string
	if err == nil {
		outTradeNo = existing.OutTradeNo
	} else {
		// 新建支付单
		outTradeNo = genTradeNo(order.Id)
		pay := &models.Payment{
			OrderId:    order.Id,
			OutTradeNo: outTradeNo,
			Channel:    "wechat",
			Amount:     order.RentAmount,
			Status:     models.PaymentStatusPending,
		}
		if _, err := o.Insert(pay); err != nil {
			c.Fail(500, "创建支付单失败")
			return
		}
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

	c.OK(map[string]any{
		"out_trade_no": outTradeNo,
		"amount":       order.RentAmount,
		"prepay":       res,
	})
}

// refundRequest 退款请求
type refundRequest struct {
	OrderId      int64  `json:"order_id"`      // 订单ID
	RefundReason string `json:"refund_reason"` // 退款原因（可选）
}

// Refund 发起退款（需登录，仅订单租客或房东可操作）。
// POST /api/v1/pay/refund
// 请求：{"order_id":1,"refund_reason":"拍错了"}
// 流程：校验归属/支付状态 -> 调用微信退款（mock 直接成功）-> 标记支付单已退款并重置订单
func (c *PaymentController) Refund() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	var req refundRequest
	if err := c.Ctx.BindJSON(&req); err != nil || req.OrderId <= 0 {
		c.Fail(400, "参数错误")
		return
	}

	o := orm.NewOrm()
	order := &models.Order{Id: req.OrderId}
	if err := o.Read(order); err != nil {
		c.Fail(404, "订单不存在")
		return
	}
	if order.RenterId != uid && order.OwnerId != uid {
		c.Fail(403, "无权操作该订单")
		return
	}

	// 查询该订单的支付单（取最新一条）
	var pay models.Payment
	if err := o.QueryTable(new(models.Payment)).
		Filter("order_id", order.Id).
		OrderBy("-id").
		One(&pay); err != nil {
		c.Fail(404, "支付单不存在")
		return
	}
	if !pay.IsPaid() {
		c.Fail(400, "支付单未支付成功，无需退款")
		return
	}

	// 调用微信退款（mock 模式直接成功）
	refundNo := genRefundNo(order.Id)
	gw := payments.DefaultGateway()
	res, err := gw.Refund(payments.RefundReq{
		OutTradeNo: pay.OutTradeNo,
		TotalFee:   pay.Amount,
		RefundFee:  pay.Amount,
		RefundNo:   refundNo,
	})
	if err != nil {
		c.Fail(500, "退款失败: "+err.Error())
		return
	}

	// 退款成功：标记支付单已退款 + 订单回到待支付（可重新支付/取消）
	if err := payments.NewOrderService().MarkRefunded(pay.OutTradeNo); err != nil {
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
	c.Ctx.Output.Body([]byte("<xml><return_code><![CDATA[" + code + "]]></return_code></xml>"))
}

// genTradeNo 生成商户订单号：RENT + 时间戳毫秒 + 订单ID，保证唯一。
func genTradeNo(orderID int64) string {
	return fmt.Sprintf("RENT%d%s", time.Now().UnixMilli(), strconv.FormatInt(orderID, 10))
}

// genRefundNo 生成商户退款单号：REF + 时间戳毫秒 + 订单ID，保证唯一。
func genRefundNo(orderID int64) string {
	return fmt.Sprintf("REF%d%s", time.Now().UnixMilli(), strconv.FormatInt(orderID, 10))
}
