// 订单流转 + 押金台账单元测试：注入桩 Store，验证状态机迁移、身份/状态守卫、幂等性
package services

import (
	"errors"
	"testing"

	"github.com/erikwang2013/item-rental/server/models"
)

// stubStore 内存版 OrderStore 桩，记录所有写操作以便断言
type stubStore struct {
	orders   map[int64]*models.Order
	deposits []models.Deposit
	balance  map[int64]float64
	credits  map[int64]int
	events   []models.CreditEvent
	notifies []notifyRec
}

type notifyRec struct {
	uid     int64
	typ     string
	content string
}

func (s *stubStore) GetOrder(id int64) (*models.Order, error) {
	o, ok := s.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

func (s *stubStore) TransitionOrder(id int64, from, to int) (int64, error) {
	o, ok := s.orders[id]
	if !ok {
		return 0, ErrOrderNotFound
	}
	if o.Status != from {
		return 0, nil
	}
	o.Status = to
	return 1, nil
}

func (s *stubStore) InsertDeposit(d *models.Deposit) error {
	s.deposits = append(s.deposits, *d)
	return nil
}

func (s *stubStore) AdjustDepositBal(userID int64, delta float64) error {
	s.balance[userID] += delta
	return nil
}

func (s *stubStore) Notify(userID int64, typ, title, content string) error {
	s.notifies = append(s.notifies, notifyRec{uid: userID, typ: typ, content: content})
	return nil
}

func (s *stubStore) AdjustCredit(userID int64, delta int) error {
	s.credits[userID] += delta
	return nil
}

func (s *stubStore) InsertCreditEvent(e *models.CreditEvent) error {
	s.events = append(s.events, *e)
	return nil
}

// newStub 构造一个已支付待取货(1)的租约，租客=100，房东=200，押金=50
func newStub() *stubStore {
	st := &stubStore{
		orders:   map[int64]*models.Order{1: {Id: 1, RenterId: 100, OwnerId: 200, Deposit: 50, Status: models.OrderStatusToPickup, OrderNo: "O20260901001"}},
		balance:  map[int64]float64{100: 80},
		credits:  map[int64]int{},
		deposits: nil,
	}
	return st
}

func TestPickup(t *testing.T) {
	t.Run("租客取货 1→2 冻结押金并扣减余额", func(t *testing.T) {
		st := newStub()
		ok, err := Pickup(st, 1, 100)
		if err != nil {
			t.Fatalf("Pickup err = %v", err)
		}
		if !ok {
			t.Fatalf("Pickup 未发生迁移")
		}
		if st.orders[1].Status != models.OrderStatusRenting {
			t.Fatalf("状态 = %d, 期望 %d", st.orders[1].Status, models.OrderStatusRenting)
		}
		if len(st.deposits) != 1 || st.deposits[0].Type != models.DepositTypeFreeze || st.deposits[0].Amount != 50 || st.deposits[0].UserId != 100 {
			t.Fatalf("押金冻结流水不符: %+v", st.deposits)
		}
		if st.balance[100] != 30 {
			t.Fatalf("租客押金余额 = %v, 期望 30(扣 50)", st.balance[100])
		}
	})
	t.Run("房东不能取货", func(t *testing.T) {
		st := newStub()
		_, err := Pickup(st, 1, 200)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("err = %v, 期望 ErrForbidden", err)
		}
	})
	t.Run("状态非法拒绝迁移且不写流水", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusPending // 未支付不可取货
		_, err := Pickup(st, 1, 100)
		if !errors.Is(err, ErrInvalidState) {
			t.Fatalf("err = %v, 期望 ErrInvalidState", err)
		}
		if len(st.deposits) != 0 {
			t.Fatalf("不应写入押金流水: %+v", st.deposits)
		}
	})
	t.Run("重复取货幂等且不重复写流水", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusRenting // 已被并发迁移
		ok, err := Pickup(st, 1, 100)
		if err != nil {
			t.Fatalf("幂等调用 err = %v", err)
		}
		if ok {
			t.Fatalf("幂等调用不应发生迁移")
		}
		if len(st.deposits) != 0 {
			t.Fatalf("幂等调用不应重复写押金流水: %+v", st.deposits)
		}
		if st.balance[100] != 80 {
			t.Fatalf("幂等调用不应重复扣减余额: %v", st.balance[100])
		}
	})
}

func TestRequestReturn(t *testing.T) {
	t.Run("租客申请归还 2→3", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusRenting
		ok, err := RequestReturn(st, 1, 100)
		if err != nil || !ok {
			t.Fatalf("RequestReturn ok=%v err=%v", ok, err)
		}
		if st.orders[1].Status != models.OrderStatusToReturn {
			t.Fatalf("状态 = %d, 期望 %d", st.orders[1].Status, models.OrderStatusToReturn)
		}
		if len(st.deposits) != 0 {
			t.Fatalf("申请归还不应写押金流水")
		}
	})
	t.Run("房东不能申请归还", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusRenting
		_, err := RequestReturn(st, 1, 200)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("err = %v, 期望 ErrForbidden", err)
		}
	})
}

func TestConfirmReturn(t *testing.T) {
	t.Run("房东确认归还 3→4 解冻押金返还余额", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusToReturn
		ok, err := ConfirmReturn(st, 1, 200)
		if err != nil || !ok {
			t.Fatalf("ConfirmReturn ok=%v err=%v", ok, err)
		}
		if st.orders[1].Status != models.OrderStatusReturned {
			t.Fatalf("状态 = %d, 期望 %d", st.orders[1].Status, models.OrderStatusReturned)
		}
		if len(st.deposits) != 1 || st.deposits[0].Type != models.DepositTypeUnfreeze || st.deposits[0].Amount != 50 || st.deposits[0].UserId != 100 {
			t.Fatalf("押金解冻流水不符: %+v", st.deposits)
		}
		if st.balance[100] != 130 {
			t.Fatalf("租客押金余额 = %v, 期望 130(返 50)", st.balance[100])
		}
	})
	t.Run("租客不能确认归还", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusToReturn
		_, err := ConfirmReturn(st, 1, 100)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("err = %v, 期望 ErrForbidden", err)
		}
	})
	t.Run("确认归还后租客信用+5 并收通知", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusToReturn
		ok, err := ConfirmReturn(st, 1, 200)
		if err != nil || !ok {
			t.Fatalf("ConfirmReturn ok=%v err=%v", ok, err)
		}
		if st.credits[100] != 5 {
			t.Fatalf("租客信用变动 = %d, 期望 +5", st.credits[100])
		}
		if len(st.events) != 1 || st.events[0].Reason != "return_on_time" || st.events[0].Change != 5 || st.events[0].UserId != 100 {
			t.Fatalf("信用流水不符: %+v", st.events)
		}
		if len(st.notifies) != 1 || st.notifies[0].uid != 100 || st.notifies[0].typ != "return_confirmed" {
			t.Fatalf("通知不符: %+v", st.notifies)
		}
	})
}

func TestBreach(t *testing.T) {
	t.Run("房东判定违约 3→6 扣押金且不退还余额", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusToReturn
		ok, err := Breach(st, 1, 200)
		if err != nil || !ok {
			t.Fatalf("Breach ok=%v err=%v", ok, err)
		}
		if st.orders[1].Status != models.OrderStatusBreach {
			t.Fatalf("状态 = %d, 期望 %d", st.orders[1].Status, models.OrderStatusBreach)
		}
		if len(st.deposits) != 1 || st.deposits[0].Type != models.DepositTypeDeduct || st.deposits[0].Amount != 50 || st.deposits[0].UserId != 100 {
			t.Fatalf("押金扣款流水不符: %+v", st.deposits)
		}
		if st.balance[100] != 80 {
			t.Fatalf("违约不应返还押金余额: %v", st.balance[100])
		}
	})
	t.Run("租客不能判定违约", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusToReturn
		_, err := Breach(st, 1, 100)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("err = %v, 期望 ErrForbidden", err)
		}
	})
	t.Run("违约后押金入物主账 租客信用-30 并收通知", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusToReturn
		ok, err := Breach(st, 1, 200)
		if err != nil || !ok {
			t.Fatalf("Breach ok=%v err=%v", ok, err)
		}
		// 押金赔付物主（此前不入账）
		if st.balance[200] != 50 {
			t.Fatalf("物主押金余额 = %v, 期望 50(入账)", st.balance[200])
		}
		if st.credits[100] != -30 {
			t.Fatalf("租客信用变动 = %d, 期望 -30", st.credits[100])
		}
		if len(st.events) != 1 || st.events[0].Reason != "breach" || st.events[0].Change != -30 {
			t.Fatalf("信用流水不符: %+v", st.events)
		}
		if len(st.notifies) != 1 || st.notifies[0].uid != 100 || st.notifies[0].typ != "breach" {
			t.Fatalf("通知不符: %+v", st.notifies)
		}
	})
}
