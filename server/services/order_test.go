// 订单服务单元测试：纯函数，无 DB 依赖，可离线运行
package services

import (
	"strings"
	"testing"

	"github.com/erikwang2013/item-rental/server/models"
)

func onShelfItem() *models.Item {
	return &models.Item{Id: 1, OwnerId: 2, DailyPrice: 50, Deposit: 100, Status: 1}
}

// TestGenerateOrderNoFormat 格式：ORD 前缀 + 至少 8 位数字
func TestGenerateOrderNoFormat(t *testing.T) {
	no := GenerateOrderNo()
	if !strings.HasPrefix(no, "ORD") {
		t.Errorf("订单号应以 ORD 开头, got %q", no)
	}
	digits := strings.TrimPrefix(no, "ORD")
	if len(digits) < 8 {
		t.Errorf("订单号时间戳+随机数部分应 >=8 位, got %q", no)
	}
	for _, ch := range digits {
		if ch < '0' || ch > '9' {
			t.Errorf("订单号数字段含非数字字符: %q", no)
		}
	}
}

// TestGenerateOrderNoUnique 批量生成不重复。
// 采用 n=1000：同一毫秒内约 1000 个样本落在 10^8 随机空间，
// 期望碰撞数 ≈ n²/2e8 ≈ 0.005，测试稳定不抖动。
func TestGenerateOrderNoUnique(t *testing.T) {
	const n = 1000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		no := GenerateOrderNo()
		if _, dup := seen[no]; dup {
			t.Fatalf("订单号重复: %s", no)
		}
		seen[no] = struct{}{}
	}
}

// TestBuildOrderValid 合法下单：天数、租金、押金计算正确
func TestBuildOrderValid(t *testing.T) {
	order, err := BuildOrder(onShelfItem(), 3, "2026-09-01", "2026-09-03")
	if err != nil {
		t.Fatalf("合法下单应成功, got %v", err)
	}
	if order.Days != 3 {
		t.Errorf("Days = %d, want 3", order.Days)
	}
	if order.RentAmount != 150 {
		t.Errorf("RentAmount = %v, want 150 (3×50)", order.RentAmount)
	}
	if order.Deposit != 100 {
		t.Errorf("Deposit = %v, want 100", order.Deposit)
	}
	if order.Status != models.OrderStatusPending {
		t.Errorf("Status = %d, want %d(待支付)", order.Status, models.OrderStatusPending)
	}
	if order.RenterId != 3 || order.OwnerId != 2 || order.ItemId != 1 {
		t.Errorf("订单归属字段异常: %+v", order)
	}
	if order.OrderNo == "" {
		t.Error("订单号不应为空")
	}
}

// TestBuildOrderSameDay 同一天租期为 1 天
func TestBuildOrderSameDay(t *testing.T) {
	order, err := BuildOrder(onShelfItem(), 3, "2026-09-01", "2026-09-01")
	if err != nil {
		t.Fatalf("同日租期应成功, got %v", err)
	}
	if order.Days != 1 || order.RentAmount != 50 {
		t.Errorf("同日租期 Days=%d RentAmount=%v, want 1/50", order.Days, order.RentAmount)
	}
}

// TestBuildOrderReject 非法下单：start>end、天数不合法、自租、物品下架/售罄、日期格式错误
func TestBuildOrderReject(t *testing.T) {
	cases := []struct {
		name  string
		item  *models.Item
		renter int64
		start string
		end   string
	}{
		{"start_after_end", onShelfItem(), 3, "2026-09-03", "2026-09-01"},
		{"start_before_epoch_bad", onShelfItem(), 3, "2026-99-01", "2026-09-03"},
		{"end_bad_format", onShelfItem(), 3, "2026-09-01", "2026/09/03"},
		{"renter_is_owner", onShelfItem(), 2, "2026-09-01", "2026-09-02"},
		{"item_offshelf", &models.Item{Id: 1, OwnerId: 2, DailyPrice: 50, Deposit: 100, Status: 0}, 3, "2026-09-01", "2026-09-02"},
		{"item_soldout", &models.Item{Id: 1, OwnerId: 2, DailyPrice: 50, Deposit: 100, Status: 2}, 3, "2026-09-01", "2026-09-02"},
		{"item_nil", nil, 3, "2026-09-01", "2026-09-02"},
	}
	for _, tc := range cases {
		_, err := BuildOrder(tc.item, tc.renter, tc.start, tc.end)
		if err == nil {
			t.Errorf("%s: 应校验失败", tc.name)
		}
	}
}

// TestBuildOrderDaysZero 结束=开始前一天应拒绝（天数<1）
func TestBuildOrderDaysZero(t *testing.T) {
	_, err := BuildOrder(onShelfItem(), 3, "2026-09-02", "2026-09-01")
	if err == nil {
		t.Error("结束早于开始应失败")
	}
}
