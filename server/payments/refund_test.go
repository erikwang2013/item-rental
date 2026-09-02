// 退款单元测试：mock 退款成功路径、真实模式证书守卫、退款回调参数化分支
package payments

import (
	"strings"
	"testing"

	"github.com/erikwang2013/item-rental/server/models"
)

// stubOrderService 注入网关的订单服务桩：记录 MarkRefunded 调用以便断言
type stubOrderService struct {
	refunded []string
}

func (s *stubOrderService) GetOrder(orderNo string) (*models.Order, error) {
	return nil, nil
}

func (s *stubOrderService) MarkPaid(orderNo, transactionId string) error {
	return nil
}

func (s *stubOrderService) MarkRefunded(outTradeNo string) error {
	s.refunded = append(s.refunded, outTradeNo)
	return nil
}

// TestRefundValidation 参数校验：mock 与真实模式行为一致（校验先于模式分流）
func TestRefundValidation(t *testing.T) {
	cases := []struct {
		name string
		req  RefundReq
	}{
		{"空商户订单号", RefundReq{TotalFee: 1, RefundFee: 1, RefundNo: "R1"}},
		{"空商户退款单号", RefundReq{OutTradeNo: "T1", TotalFee: 1, RefundFee: 1}},
		{"订单金额非正", RefundReq{OutTradeNo: "T1", TotalFee: 0, RefundFee: 1, RefundNo: "R1"}},
		{"退款金额非正", RefundReq{OutTradeNo: "T1", TotalFee: 1, RefundFee: 0, RefundNo: "R1"}},
		{"退款金额大于订单金额", RefundReq{OutTradeNo: "T1", TotalFee: 1, RefundFee: 2, RefundNo: "R1"}},
	}
	for _, mock := range []bool{true, false} {
		g := newTestGateway(mock)
		for _, c := range cases {
			name := c.name
			if mock {
				name += " [mock]"
			} else {
				name += " [real]"
			}
			t.Run(name, func(t *testing.T) {
				if _, err := g.Refund(c.req); err == nil {
					t.Error("非法退款参数应返回错误")
				}
			})
		}
	}
}

// TestRefundMock mock 模式退款直接成功，返回模拟 refund_id
func TestRefundMock(t *testing.T) {
	g := newTestGateway(true)

	res, err := g.Refund(RefundReq{OutTradeNo: "T1", TotalFee: 58.8, RefundFee: 58.8, RefundNo: "REF20260901001"})
	if err != nil {
		t.Fatalf("mock 退款失败: %v", err)
	}
	if !res.Mock {
		t.Error("mock 模式应标记 Mock=true")
	}
	if res.RefundID != "mock_refund_REF20260901001" {
		t.Errorf("refund_id = %q, 期望 mock_refund_REF20260901001", res.RefundID)
	}

	// 部分退款金额也直接成功（本系统当前为全额退款，mock 不校验比例以外的约束）
	partial, err := g.Refund(RefundReq{OutTradeNo: "T2", TotalFee: 100, RefundFee: 30, RefundNo: "REF2"})
	if err != nil {
		t.Fatalf("mock 部分退款失败: %v", err)
	}
	if !partial.Mock || partial.RefundID != "mock_refund_REF2" {
		t.Errorf("partial refund 异常: %+v", partial)
	}
}

// TestRefundRealRequiresCert 真实模式未配置商户证书：返回明确错误而非 panic
func TestRefundRealRequiresCert(t *testing.T) {
	g := newTestGateway(false) // 未配置 wechat_cert_file/wechat_cert_key

	res, err := g.Refund(RefundReq{OutTradeNo: "T1", TotalFee: 1, RefundFee: 1, RefundNo: "R1"})
	if err == nil {
		t.Fatalf("真实模式未配置证书应报错, got res=%+v", res)
	}
	if !strings.Contains(err.Error(), "wechat_cert_file") {
		t.Errorf("错误应提示证书配置键, got: %v", err)
	}
}

// TestHandleNotifyRefund 退款回调参数化：SUCCESS 触发 MarkRefunded，其余仅确认
func TestHandleNotifyRefund(t *testing.T) {
	cases := []struct {
		name         string
		refundStatus string
		wantMarked   bool // 是否应调用 MarkRefunded
	}{
		{"退款成功 SUCCESS", "SUCCESS", true},
		{"状态变更 CHANGE 仅确认", "CHANGE", false},
		{"退款关闭 REFUNDCLOSE 仅确认", "REFUNDCLOSE", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := &stubOrderService{}
			g := newTestGateway(true)
			g.svc = svc

			params := map[string]string{
				"return_code":   "SUCCESS",
				"result_code":   "SUCCESS",
				"out_trade_no":  "T123",
				"refund_status": c.refundStatus,
			}
			params["sign"] = signParams(params, g.cfg.MchKey, g.cfg.SignType)
			raw, _ := mapToXML(params)

			if err := g.HandleNotify(raw); err != nil {
				t.Fatalf("HandleNotify err = %v", err)
			}
			marked := len(svc.refunded) > 0
			if marked != c.wantMarked {
				t.Errorf("MarkRefunded 调用标记 = %v, 期望 %v (记录=%v)", marked, c.wantMarked, svc.refunded)
			}
			if c.wantMarked && len(svc.refunded) != 1 {
				t.Errorf("SUCCESS 应恰好调用一次 MarkRefunded, got %v", svc.refunded)
			}
		})
	}
}

// TestHandleNotifyRefundRepeat 重复 SUCCESS 退款回调：再次调用 MarkRefunded 仍幂等返回成功
// （DB 层条件更新 1->3 保证幂等，此测试验证通知层对重复回调不报错）
func TestHandleNotifyRefundRepeat(t *testing.T) {
	svc := &stubOrderService{}
	g := newTestGateway(true)
	g.svc = svc

	params := map[string]string{
		"return_code":   "SUCCESS",
		"result_code":   "SUCCESS",
		"out_trade_no":  "T123",
		"refund_status": "SUCCESS",
	}
	params["sign"] = signParams(params, g.cfg.MchKey, g.cfg.SignType)
	raw, _ := mapToXML(params)

	for i := 0; i < 2; i++ {
		if err := g.HandleNotify(raw); err != nil {
			t.Fatalf("第 %d 次重复回调 err = %v", i+1, err)
		}
	}
	if len(svc.refunded) != 2 {
		t.Errorf("两次 SUCCESS 回调应各触发一次 MarkRefunded(其自身幂等), got %v", svc.refunded)
	}
}

// TestHandleNotifyRefundErrors 退款回调异常分支
func TestHandleNotifyRefundErrors(t *testing.T) {
	t.Run("缺少 out_trade_no", func(t *testing.T) {
		g := newTestGateway(true)
		params := map[string]string{
			"return_code":   "SUCCESS",
			"result_code":   "SUCCESS",
			"refund_status": "SUCCESS",
		}
		params["sign"] = signParams(params, g.cfg.MchKey, g.cfg.SignType)
		raw, _ := mapToXML(params)
		if err := g.HandleNotify(raw); err == nil {
			t.Error("缺 out_trade_no 的退款回调应报错")
		}
	})

	t.Run("SUCCESS 但订单服务未配置", func(t *testing.T) {
		g := newTestGateway(true) // svc=nil
		params := map[string]string{
			"return_code":   "SUCCESS",
			"result_code":   "SUCCESS",
			"out_trade_no":  "T123",
			"refund_status": "SUCCESS",
		}
		params["sign"] = signParams(params, g.cfg.MchKey, g.cfg.SignType)
		raw, _ := mapToXML(params)
		if err := g.HandleNotify(raw); err == nil {
			t.Error("svc 未配置时 SUCCESS 退款应报错")
		}
	})
}
