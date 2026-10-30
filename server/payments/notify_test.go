// 支付回调决策测试：订单状态矩阵、mock 拒收回调、退款单号幂等键
package payments

import (
	"strings"
	"testing"

	"github.com/erikwang2013/item-rental/server/models"
)

// TestNotifyAction 覆盖回调处置矩阵：
// 本订单支付流水的重试无论订单走到哪步都仅确认；待支付标记已付；取消必退；
// 缺交易号的历史数据按支付单是否已成功区分重试与新到账；其余归属另一笔支付的退款。
func TestNotifyAction(t *testing.T) {
	cases := []struct {
		name            string
		status          int
		orderPayTradeNo string
		txID            string
		alreadySuccess  bool
		want            notifyActionType
	}{
		{"待支付标记已付", models.OrderStatusPending, "", "wx1", false, notifyMarkPaid},
		{"待支付但支付单已成功自愈", models.OrderStatusPending, "", "wx1", true, notifyMarkPaid},

		{"待取且交易号一致仅确认", models.OrderStatusToPickup, "wx1", "wx1", true, notifyDone},
		{"待取且空交易号且支付单已成功仅确认", models.OrderStatusToPickup, "", "wx1", true, notifyDone},
		{"待取且空交易号且非重试自动退款", models.OrderStatusToPickup, "", "wx1", false, notifyRefund},
		{"待取但交易号不同自动退款", models.OrderStatusToPickup, "wx0", "wx1", false, notifyRefund},

		// 关键回归：订单已完成流转后，同一笔支付的重试回调不能退款
		{"租赁中相同交易号重试仅确认", models.OrderStatusRenting, "wx1", "wx1", true, notifyDone},
		{"租赁中不相同交易号自动退款", models.OrderStatusRenting, "wx0", "wx1", false, notifyRefund},
		{"待归还不相同交易号自动退款", models.OrderStatusToReturn, "wx0", "wx1", false, notifyRefund},
		{"待归还相同交易号重试仅确认", models.OrderStatusToReturn, "wx1", "wx1", true, notifyDone},
		{"已归还不相同交易号自动退款", models.OrderStatusReturned, "wx0", "wx1", false, notifyRefund},
		{"已归还相同交易号重试仅确认", models.OrderStatusReturned, "wx1", "wx1", true, notifyDone},
		{"已归还空交易号且支付单已成功仅确认", models.OrderStatusReturned, "", "wx1", true, notifyDone},
		{"违约不相同交易号自动退款", models.OrderStatusBreach, "wx0", "wx1", false, notifyRefund},
		{"违约相同交易号重试仅确认", models.OrderStatusBreach, "wx1", "wx1", true, notifyDone},

		{"已取消空交易号自动退款", models.OrderStatusCancelled, "", "wx1", false, notifyRefund},
		{"已取消不同交易号自动退款", models.OrderStatusCancelled, "wx0", "wx1", false, notifyRefund},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := notifyAction(c.status, c.orderPayTradeNo, c.txID, c.alreadySuccess); got != c.want {
				t.Errorf("notifyAction(%d, %q, %q, %v) = %v, 期望 %v",
					c.status, c.orderPayTradeNo, c.txID, c.alreadySuccess, got, c.want)
			}
		})
	}
}

// TestPaymentUpdateStatuses 支付单状态是否可条件更新为成功：
// 待支付/失败可更新（Failed 迟到成功 = 实际扣款成功），已成功不再改写，已退款绝不翻回成功。
func TestPaymentUpdateStatuses(t *testing.T) {
	cases := []struct {
		name      string
		payStatus int
		want      bool
	}{
		{"待支付可更新", models.PaymentStatusPending, true},
		{"失败可更新", models.PaymentStatusFailed, true},
		{"已成功不再改写", models.PaymentStatusSuccess, false},
		{"已退款不可翻回成功", models.PaymentStatusRefunded, false},
		{"未知状态不改写", 99, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := paymentUpdateStatuses(c.payStatus); got != c.want {
				t.Errorf("paymentUpdateStatuses(%d) = %v, 期望 %v", c.payStatus, got, c.want)
			}
		})
	}
}

// TestAutoRefund 自动退款：mock 网关退款成功后只标记支付单已退款；svc 缺失时报错。
func TestAutoRefund(t *testing.T) {
	svc := &stubOrderService{}
	g := newTestGateway(true) // mock 退款直接成功
	g.svc = svc
	if err := g.autoRefund("RENT1", 10); err != nil {
		t.Fatalf("autoRefund err = %v", err)
	}
	if len(svc.paymentRefunded) != 1 || svc.paymentRefunded[0] != "RENT1" {
		t.Errorf("应调用 MarkPaymentRefunded(RENT1), got %v", svc.paymentRefunded)
	}

	noSvc := newTestGateway(true)
	if err := noSvc.autoRefund("RENT1", 10); err == nil {
		t.Error("svc 未配置时 autoRefund 应报错")
	}
}

// TestHandleNotifyRejectsMock mock 模式必须在解析前拒绝回调（伪造回调无法落单）。
func TestHandleNotifyRejectsMock(t *testing.T) {
	g := newTestGateway(true)
	err := g.HandleNotify([]byte("<xml></xml>"))
	if err == nil || !strings.Contains(err.Error(), "mock") {
		t.Fatalf("mock 模式应拒绝回调, err = %v", err)
	}
}

// TestRefundNo 退款单号只由支付单号决定：同一笔支付的重试必须得到相同单号。
func TestRefundNo(t *testing.T) {
	if got := RefundNo("RENT123"); got != "REFRENT123" {
		t.Errorf("RefundNo(RENT123) = %q, 期望 REFRENT123", got)
	}
	if RefundNo("RENT1") == RefundNo("RENT2") {
		t.Error("不同支付单的退款单号不应相同")
	}
}
