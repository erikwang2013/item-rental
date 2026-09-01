// 微信支付回调处理：验签 + 金额校验 + 幂等（防重放）
package payments

import (
	"errors"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/models"
)

// HandleNotify 处理微信支付结果通知。
//
// 处理顺序（任一环节失败都返回错误，微信会重试）：
//  1. 解析回调 XML
//  2. 业务成功判定（return_code/result_code == SUCCESS）
//  3. 验签：防止伪造回调
//  4. 幂等：该 out_trade_no 已处理成功则直接返回（不重复扣款/改状态）
//  5. 金额校验：回调金额必须等于订单应付金额，防止篡改
//  6. 持久化支付成功 + 调用订单服务标记已支付
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

// parseFen 解析微信金额字符串（分）为 int64。
func parseFen(s string) int64 {
	var v int64
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return 0
	}
	return v
}
