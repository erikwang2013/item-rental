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

// TestItemSearchableArrayLocation geo_point 合并字段：有坐标才发 location，0,0 不发。
func TestItemSearchableArrayLocation(t *testing.T) {
	geo := Item{Id: 1, Lat: 31.23, Lng: 121.47}
	arr := geo.ToSearchableArray()
	if arr["location"] != "31.23,121.47" {
		t.Errorf("有坐标应含 location=%q, got %v", "31.23,121.47", arr["location"])
	}
	zero := Item{Id: 2}
	if _, ok := zero.ToSearchableArray()["location"]; ok {
		t.Error("无坐标(0,0)不应发 location 字段")
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

// --- F1 富化:UserPublic 公开视图与 Item/Order 详情嵌入 ---

func TestUserToPublic(t *testing.T) {
	u := User{Id: 7, Phone: "hash", Nickname: "房东", Avatar: "http://x/a.png", RealName: "enc", CreditScore: 90, DepositBal: 12.5, Status: 1}
	pub := u.ToPublic()
	if pub.Id != 7 || pub.Nickname != "房东" || pub.Avatar != "http://x/a.png" || pub.CreditScore != 90 {
		t.Errorf("ToPublic 字段不符: %+v", pub)
	}
	// 公开视图不可携带 PII/私密字段(结构上无这些字段,编译期已保证;此处断言 JSON 不含其名)
	data, _ := json.Marshal(pub)
	for _, forbidden := range []string{"phone", "real_name", "deposit_bal", "status", "created_at"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("UserPublic JSON 泄露字段 %q: %s", forbidden, data)
		}
	}
}

func TestItemOwnerOmitEmpty(t *testing.T) {
	// nil Owner:序列化不含 owner 键(列表场景)
	plain, err := json.Marshal(Item{Id: 1, OwnerId: 7})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(plain), `"owner"`) {
		t.Errorf("nil Owner 不应输出 owner 键: %s", plain)
	}
	// set Owner:输出嵌套对象且字段完整
	pub := User{Id: 7, Nickname: "房东", Avatar: "http://x/a.png", CreditScore: 90}.ToPublic()
	with, err := json.Marshal(Item{Id: 1, OwnerId: 7, Owner: &pub})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(with)
	for _, want := range []string{`"owner":{`, `"id":"7"`, `"nickname":"房东"`, `"avatar":"http://x/a.png"`, `"credit_score":90`} {
		if !strings.Contains(s, want) {
			t.Errorf("owner 对象缺 %q: %s", want, s)
		}
	}
}

func TestOrderCounterpartyOmitEmpty(t *testing.T) {
	plain, err := json.Marshal(Order{Id: 1})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(plain), `"owner"`) || strings.Contains(string(plain), `"renter"`) {
		t.Errorf("nil 双方不应输出 owner/renter 键: %s", plain)
	}
	oPub := User{Id: 2, Nickname: "房东", CreditScore: 80}.ToPublic()
	rPub := User{Id: 3, Nickname: "租客", CreditScore: 100}.ToPublic()
	with, err := json.Marshal(Order{Id: 1, Owner: &oPub, Renter: &rPub})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(with)
	if !strings.Contains(s, `"owner":{"id":"2","nickname":"房东"`) || !strings.Contains(s, `"renter":{"id":"3","nickname":"租客"`) {
		t.Errorf("owner/renter 对象不符: %s", s)
	}
}
