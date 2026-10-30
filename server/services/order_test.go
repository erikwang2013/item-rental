// 订单服务单元测试：纯函数，无 DB 依赖，可离线运行
package services

import (
	"strings"
	"testing"
	"time"

	"github.com/erikwang2013/item-rental/server/models"
)

func onShelfItem() *models.Item {
	return &models.Item{Id: 1, OwnerId: 2, DailyPrice: 50, Status: 1}
}

// futureDate 返回相对今天偏移 n 天的日期，避免固定日期随日历过期
func futureDate(n int) string {
	return time.Now().AddDate(0, 0, n).Format(orderDateLayout)
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

// TestBuildOrderValid 合法下单：天数与租金计算正确
func TestBuildOrderValid(t *testing.T) {
	order, err := BuildOrder(onShelfItem(), 3, futureDate(1), futureDate(3))
	if err != nil {
		t.Fatalf("合法下单应成功, got %v", err)
	}
	if order.Days != 3 {
		t.Errorf("Days = %d, want 3", order.Days)
	}
	if order.RentAmount != 150 {
		t.Errorf("RentAmount = %v, want 150 (3×50)", order.RentAmount)
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
	order, err := BuildOrder(onShelfItem(), 3, futureDate(1), futureDate(1))
	if err != nil {
		t.Fatalf("同日租期应成功, got %v", err)
	}
	if order.Days != 1 || order.RentAmount != 50 {
		t.Errorf("同日租期 Days=%d RentAmount=%v, want 1/50", order.Days, order.RentAmount)
	}
}

// TestBuildOrderReject 非法下单：start>end、过去日期、超 365 天、自租、物品下架、日期格式错误
func TestBuildOrderReject(t *testing.T) {
	cases := []struct {
		name   string
		item   *models.Item
		renter int64
		start  string
		end    string
	}{
		{"start_after_end", onShelfItem(), 3, futureDate(3), futureDate(1)},
		{"start_in_past", onShelfItem(), 3, futureDate(-1), futureDate(1)},
		{"over_365_days", onShelfItem(), 3, futureDate(1), futureDate(367)},
		{"start_before_epoch_bad", onShelfItem(), 3, "2026-99-01", futureDate(3)},
		{"end_bad_format", onShelfItem(), 3, futureDate(1), "2026/09/03"},
		{"renter_is_owner", onShelfItem(), 2, futureDate(1), futureDate(2)},
		{"item_offshelf", &models.Item{Id: 1, OwnerId: 2, DailyPrice: 50, Status: 0}, 3, futureDate(1), futureDate(2)},
		{"item_soldout", &models.Item{Id: 1, OwnerId: 2, DailyPrice: 50, Status: 2}, 3, futureDate(1), futureDate(2)},
		{"item_nil", nil, 3, futureDate(1), futureDate(2)},
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
	_, err := BuildOrder(onShelfItem(), 3, futureDate(2), futureDate(1))
	if err == nil {
		t.Error("结束早于开始应失败")
	}
}

// TestBuildOrderTodayAllowed 今天下单允许（只拒绝过去日期）
func TestBuildOrderTodayAllowed(t *testing.T) {
	if _, err := BuildOrder(onShelfItem(), 3, futureDate(0), futureDate(0)); err != nil {
		t.Errorf("今天租期应成功, got %v", err)
	}
}

// TestBuildOrderMaxDays 365 天边界应允许，366 天应拒绝
func TestBuildOrderMaxDays(t *testing.T) {
	if _, err := BuildOrder(onShelfItem(), 3, futureDate(1), futureDate(365)); err != nil {
		t.Errorf("365 天租期应成功, got %v", err)
	}
	if _, err := BuildOrder(onShelfItem(), 3, futureDate(1), futureDate(366)); err == nil {
		t.Error("366 天租期应拒绝")
	}
}

// day 构造测试用 UTC 日期（纯日期运算，与今天无关）
func day(n int) time.Time {
	return time.Date(2026, 1, n, 0, 0, 0, 0, time.UTC)
}

// TestPeakConcurrent 峰值并发：按天取重叠订单数的最大值
func TestPeakConcurrent(t *testing.T) {
	cases := []struct {
		name     string
		newRange DateRange
		existing []DateRange
		want     int
	}{
		{"无既有订单", DateRange{day(1), day(10)}, nil, 1},
		{"单日重叠", DateRange{day(1), day(3)}, []DateRange{{day(2), day(2)}}, 2},
		{"日期不相交", DateRange{day(1), day(3)}, []DateRange{{day(4), day(5)}}, 1},
	}
	for _, tc := range cases {
		if got := PeakConcurrent(tc.newRange, tc.existing); got != tc.want {
			t.Errorf("%s: PeakConcurrent = %d, want %d", tc.name, got, tc.want)
		}
	}
}

// TestCheckAvailability 库存边界：租期内峰值并发不得超过库存
func TestCheckAvailability(t *testing.T) {
	cases := []struct {
		name     string
		newRange DateRange
		existing []DateRange
		stock    int
		wantErr  bool
	}{
		{"库存1无重叠", DateRange{day(1), day(2)}, nil, 1, false},
		{"库存1重叠拒绝", DateRange{day(1), day(2)}, []DateRange{{day(2), day(3)}}, 1, true},
		// 旧算法按重叠订单总数=2 会误拒；实际峰值仅 2（首日+新单、末日照旧）
		{"库存2首尾各一单跨区间允许", DateRange{day(1), day(10)}, []DateRange{{day(1), day(1)}, {day(10), day(10)}}, 2, false},
		{"库存2峰值3拒绝", DateRange{day(5), day(5)}, []DateRange{{day(1), day(10)}, {day(5), day(15)}}, 2, true},
	}
	for _, tc := range cases {
		err := CheckAvailability(tc.newRange, tc.existing, tc.stock)
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr = %v", tc.name, err, tc.wantErr)
		}
	}
}
