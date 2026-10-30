// 微信支付回调处理：验签 + 金额校验 + 幂等（防重放）
package payments

import (
	"errors"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/models"
)

// HandleNotify 处理微信支付/退款结果通知。
//
// 处理顺序（任一环节失败都返回错误，微信会重试）：
//  0. mock 模式直接拒绝：mock 支付在统一支付接口内完成，回调只可能是伪造的
//  1. 解析回调 XML
//  2. 业务成功判定（return_code/result_code == SUCCESS）
//  3. 验签：防止伪造回调
//  4. 退款通知（带 refund_status）分流：SUCCESS 标记退款，CHANGE/REFUNDCLOSE 仅确认
//  5. 支付通知：幂等 + 金额校验 + 持久化支付成功 + 按订单状态标记已付/确认/自动退款
func (g *gateway) HandleNotify(rawXML []byte) error {
	// mock 模式没有真实微信订单，回调只可能是伪造的（支付在统一支付接口内直接完成）
	if g.cfg.Mock {
		return errors.New("mock 模式不接受支付回调")
	}

	params, err := xmlToMap(rawXML)
	if err != nil {
		return errors.New("回调 XML 解析失败")
	}
	if params["return_code"] != "SUCCESS" || params["result_code"] != "SUCCESS" {
		return fmt.Errorf("支付结果非成功: return=%s result=%s", params["return_code"], params["result_code"])
	}

	// 验签（签名字段不参与计算）
	if !verifySign(params, g.cfg.MchKey, g.cfg.SignType) {
		return errors.New("回调签名校验失败")
	}

	// 退款结果通知：带 refund_status，无 transaction_id/total_fee，走独立分支
	if params["refund_status"] != "" {
		return g.handleRefundNotify(params, rawXML)
	}

	outTradeNo := params["out_trade_no"]
	transactionID := params["transaction_id"]
	if outTradeNo == "" || transactionID == "" {
		return errors.New("回调缺少关键字段")
	}

	// 回调金额（分）转为元
	callbackFen := parseFen(params["total_fee"])
	if callbackFen <= 0 {
		return errors.New("回调金额非法")
	}

	o := ormer()

	// ---- 幂等检查 ----
	// 以 out_trade_no（唯一键）查询支付单
	pay := &models.Payment{OutTradeNo: outTradeNo}
	if err := o.Read(pay, "out_trade_no"); err != nil {
		if err == orm.ErrNoRows {
			return fmt.Errorf("支付单不存在: %s", outTradeNo)
		}
		return err
	}
	// 已退款支付单的迟到回调：直接确认，避免把已退款订单重新标为已支付
	if pay.Status == models.PaymentStatusRefunded {
		return nil
	}
	// ---- 金额校验 ----
	// 回调金额必须等于支付单应付金额（防止金额被篡改）
	if fen(pay.Amount) != callbackFen {
		return fmt.Errorf("回调金额不一致: 期望%d分 实际%d分", fen(pay.Amount), callbackFen)
	}

	// 回调前支付单已是成功：本次只是微信重试，没有新扣款
	alreadySuccess := pay.Status == models.PaymentStatusSuccess

	// ---- 持久化支付成功（幂等）----
	// Failed 单的迟到成功回调 = 微信实际扣款成功，接受并进入正常订单决策；
	// Success 单不再改写（重复回调继续走订单决策自愈）。
	// 条件更新只允许 0待支付/2失败 -> 1成功，防止并发退款把已退款单翻回成功。
	if paymentUpdateStatuses(pay.Status) {
		if _, err := o.QueryTable(new(models.Payment)).
			Filter("id", pay.Id).
			Filter("status__in", []int{models.PaymentStatusPending, models.PaymentStatusFailed}).
			Update(map[string]any{
				"status":         models.PaymentStatusSuccess,
				"transaction_id": transactionID,
				"raw_callback":   string(rawXML),
			}); err != nil {
			return err
		}
	}

	// ---- 按订单状态决定处置方式（订单状态机由订单模块负责）----
	// 必须用支付单关联订单的 order_no（ORD 前缀）定位订单：out_trade_no 是
	// RENT 前缀的支付单号，直接传给按 order_no 查单的 MarkPaid 会查无此单。
	order := &models.Order{Id: pay.OrderId}
	if err := o.Read(order); err != nil {
		return fmt.Errorf("支付单关联订单不存在: %d", pay.OrderId)
	}
	switch notifyAction(order.Status, order.PayTradeNo, transactionID, alreadySuccess) {
	case notifyMarkPaid:
		if err := g.svc.MarkPaid(order.OrderNo, transactionID); err != nil {
			return err
		}
		// 并发到账复核：两笔支付同时回调时只有一笔能完成 0->1 条件更新，
		// 另一笔 MarkPaid 会静默返回 nil；不复核就会出现钱收了、订单只认一笔、另一笔既不退也不记账。
		latest, err := g.svc.GetOrder(order.OrderNo)
		if err != nil {
			return err
		}
		switch {
		case latest.Status == models.OrderStatusPending:
			// MarkPaid 没落定（异常/并发），返回错误让微信重试
			return errors.New("订单支付状态未落定，等待微信重试")
		case notifyAction(latest.Status, latest.PayTradeNo, transactionID, true) == notifyRefund:
			// 订单已由另一笔支付占用/已取消：本笔原路退回
			return g.autoRefund(outTradeNo, pay.Amount)
		default:
			return nil
		}
	case notifyDone:
		// 本笔支付就是该订单的支付流水（含订单后续流转后的重试）：仅确认
		return nil
	default: // notifyRefund
		// 订单已取消/已归还/违约/被另一笔支付占用：本笔支付原路退回，只改支付单，绝不重置订单
		return g.autoRefund(outTradeNo, pay.Amount)
	}
}

// autoRefund 把重复/无效到账的支付原路退回，并只标记支付单已退款（不动订单状态）。
// 退款失败返回错误让微信重试；重试复用同一 RefundNo，微信按 out_refund_no 幂等。
func (g *gateway) autoRefund(outTradeNo string, amount float64) error {
	if g.svc == nil {
		return errors.New("退款服务未配置")
	}
	if _, err := g.Refund(RefundReq{
		OutTradeNo: outTradeNo,
		TotalFee:   amount,
		RefundFee:  amount,
		RefundNo:   RefundNo(outTradeNo),
	}); err != nil {
		return fmt.Errorf("重复支付自动退款失败: %w", err)
	}
	return g.svc.MarkPaymentRefunded(outTradeNo)
}

// paymentUpdateStatuses 判断读到支付单状态后是否需要条件更新为成功：
// 待支付/失败 -> 是（Failed 的迟到成功回调说明微信实际扣款成功）；
// 已成功 -> 否（保持原样，订单决策负责自愈）；已退款 -> 否（调用方已提前返回）。
func paymentUpdateStatuses(payStatus int) bool {
	return payStatus == models.PaymentStatusPending || payStatus == models.PaymentStatusFailed
}

// notifyActionType 支付回调对订单的处置动作
type notifyActionType int

const (
	notifyMarkPaid notifyActionType = iota // 订单待支付：标记已支付
	notifyDone                             // 已由本笔支付完成：仅确认
	notifyRefund                           // 订单不可支付：自动退款
)

// notifyAction 根据订单状态与本笔交易号决定回调处置方式（纯函数，便于单测）。
// alreadySuccess 表示读到支付单时它已是成功——本次回调只是微信重试，没有新扣款。
//   - order_pay_trade_no 等于本笔交易号：这笔就是订单的支付流水，无论订单后续流转到哪步，重试仅确认
//   - 0待支付：标记已支付（含支付单已成功但订单仍待支付的自愈）
//   - 5已取消：取消后到账的钱必须退
//   - 历史数据缺交易号：支付单已成功则视为合法支付重试（仅确认），否则是重复支付（退款）
//   - 其余（待取/租赁中/待归还/已归还/违约且是另一笔支付）：自动原路退款
func notifyAction(status int, orderPayTradeNo, txID string, alreadySuccess bool) notifyActionType {
	if txID != "" && orderPayTradeNo == txID {
		return notifyDone
	}
	if status == models.OrderStatusPending {
		return notifyMarkPaid
	}
	if status == models.OrderStatusCancelled {
		return notifyRefund
	}
	if orderPayTradeNo == "" {
		// 无交易号的历史数据无法比对：重试的合法支付不能退，新到账的重复支付必须退
		if alreadySuccess {
			return notifyDone
		}
		return notifyRefund
	}
	return notifyRefund
}

// handleRefundNotify 处理微信退款结果通知。
//
// refund_status 取值：
//   - SUCCESS：退款成功，标记支付单已退款并把订单重置回可处理状态
//   - CHANGE：状态变更，需主动查询（本系统无部分退款，仅确认不重复重试）
//   - REFUNDCLOSE：退款关闭（如原单未支付成功），仅确认不动状态
func (g *gateway) handleRefundNotify(params map[string]string, rawXML []byte) error {
	outTradeNo := params["out_trade_no"]
	if outTradeNo == "" {
		return errors.New("退款回调缺少 out_trade_no")
	}

	if params["refund_status"] != "SUCCESS" {
		// CHANGE / REFUNDCLOSE：确认收到但不标记退款，避免微信无限重试
		return nil
	}

	// 退款成功：标记支付单已退款 + 订单回到待支付（幂等）
	if g.svc == nil {
		return errors.New("退款服务未配置")
	}
	return g.svc.MarkRefunded(outTradeNo)
}

// parseFen 解析微信金额字符串（分）为 int64。
func parseFen(s string) int64 {
	var v int64
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return 0
	}
	return v
}
