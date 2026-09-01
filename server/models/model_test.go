// 模型纯方法单元测试：物品上下架判定与搜索索引字段、支付状态判定
package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestItemOnShelf(t *testing.T) {
	on := Item{Status: 1}
	if !on.IsOnShelf() || !on.ShouldBeSearchable() {
		t.Error("status=1 应上架且可搜索")
	}
	for _, status := range []int{0, 2, 3} {
		off := Item{Status: status}
		if off.IsOnShelf() || off.ShouldBeSearchable() {
			t.Errorf("status=%d 不应上架或可搜索", status)
		}
	}
}

func TestItemSearchableArray(t *testing.T) {
	it := Item{Id: 9, OwnerId: 1, CategoryId: 2, Title: "相机", DailyPrice: 58.8, Status: 1, City: "上海"}
	arr := it.ToSearchableArray()
	for _, k := range []string{"id", "owner_id", "category_id", "title", "daily_price", "status", "city", "created_at"} {
		if _, ok := arr[k]; !ok {
			t.Errorf("ToSearchableArray 缺字段 %q", k)
		}
	}
	if arr["id"] != int64(9) || arr["title"] != "相机" || arr["status"] != 1 || arr["city"] != "上海" {
		t.Errorf("ToSearchableArray 字段值异常: %v", arr)
	}
	if it.IDString() != "9" {
		t.Errorf("IDString() = %s, want 9", it.IDString())
	}
}

func TestPaymentIsPaid(t *testing.T) {
	success := Payment{Status: PaymentStatusSuccess}
	if !success.IsPaid() {
		t.Error("status=1 应为已支付")
	}
	for _, s := range []int{PaymentStatusPending, PaymentStatusFailed, PaymentStatusRefunded} {
		p := Payment{Status: s}
		if p.IsPaid() {
			t.Errorf("status=%d 不应为已支付", s)
		}
	}
}

func TestSensitiveFieldsHidden(t *testing.T) {
	// Payment.RawCallback 挂 json:"-"，绝不可随业务 JSON 泄露
	data, err := json.Marshal(Payment{Status: PaymentStatusSuccess, RawCallback: "top-secret"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if !json.Valid(data) || strings.Contains(s, "RawCallback") || strings.Contains(s, "top-secret") {
		t.Errorf("RawCallback 不应出现在 JSON 输出中: %s", s)
	}
}