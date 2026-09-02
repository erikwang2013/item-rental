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
//  1. 解析回调 XML
//  2. 业务成功判定（return_code/result_code == SUCCESS）
//  3. 验签：防止伪造回调
//  4. 退款通知（带 refund_status）分流：SUCCESS 标记退款，CHANGE/REFUNDCLOSE 仅确认
//  5. 支付通知：幂等 + 金额校验 + 持久化支付成功 + 调用订单服务标记已支付
func (g *gateway) HandleNotify(rawXML []byte) error {
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
	// 已成功：重复回调，直接返回成功（幂等，不重复处理）
	if pay.Status == models.PaymentStatusSuccess {
		return nil
	}

	// ---- 金额校验 ----
	// 回调金额必须等于支付单应付金额（防止金额被篡改）
	if fen(pay.Amount) != callbackFen {
		return fmt.Errorf("回调金额不一致: 期望%d分 实际%d分", fen(pay.Amount), callbackFen)
	}

	// ---- 持久化支付成功（幂等）----
	// 条件更新：仅当 status=0（待支付）时置为成功，并发/重复回调下第二次不会生效
	n, err := o.QueryTable(new(models.Payment)).
		Filter("id", pay.Id).
		Filter("status", models.PaymentStatusPending).
		Update(map[string]any{
			"status":         models.PaymentStatusSuccess,
			"transaction_id": transactionID,
			"raw_callback":   string(rawXML),
		})
	if err != nil {
		return err
	}
	// n==0 说明状态已被其他并发回调改为成功，幂等返回成功
	if n == 0 {
		return nil
	}

	// ---- 调用订单服务标记已支付（订单状态机由订单模块负责）----
	return g.svc.MarkPaid(outTradeNo, transactionID)
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
