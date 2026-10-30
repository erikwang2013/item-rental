// 模型纯方法单元测试：支付状态判定、公开视图裁剪
package models

import (
	"encoding/json"
	"strings"
	"testing"
)

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
	u := User{Id: 7, Phone: "hash", Nickname: "房东", Avatar: "http://x/a.png", RealName: "enc", Status: 1}
	pub := u.ToPublic()
	if pub.Id != 7 || pub.Nickname != "房东" || pub.Avatar != "http://x/a.png" {
		t.Errorf("ToPublic 字段不符: %+v", pub)
	}
	// 公开视图不可携带 PII/私密字段(结构上无这些字段,编译期已保证;此处断言 JSON 不含其名)
	data, _ := json.Marshal(pub)
	for _, forbidden := range []string{"phone", "real_name", "credit_score", "deposit_bal", "status", "created_at"} {
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
	pub := User{Id: 7, Nickname: "房东", Avatar: "http://x/a.png"}.ToPublic()
	with, err := json.Marshal(Item{Id: 1, OwnerId: 7, Owner: &pub})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(with)
	for _, want := range []string{`"owner":{`, `"id":"7"`, `"nickname":"房东"`, `"avatar":"http://x/a.png"`} {
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
	oPub := User{Id: 2, Nickname: "房东"}.ToPublic()
	rPub := User{Id: 3, Nickname: "租客"}.ToPublic()
	with, err := json.Marshal(Order{Id: 1, Owner: &oPub, Renter: &rPub})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(with)
	if !strings.Contains(s, `"owner":{"id":"2","nickname":"房东"`) || !strings.Contains(s, `"renter":{"id":"3","nickname":"租客"`) {
		t.Errorf("owner/renter 对象不符: %s", s)
	}
}
