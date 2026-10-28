// 订单流转单元测试：注入桩 Store，验证状态机迁移、身份/状态守卫、幂等性
package services

import (
	"errors"
	"testing"

	"github.com/erikwang2013/item-rental/server/models"
)

// stubStore 内存版 OrderStore 桩
type stubStore struct {
	orders map[int64]*models.Order
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

// newStub 构造一个已支付待取货(1)的租约，租客=100，房东=200
func newStub() *stubStore {
	return &stubStore{
		orders: map[int64]*models.Order{1: {Id: 1, RenterId: 100, OwnerId: 200, Status: models.OrderStatusToPickup, OrderNo: "O20260901001"}},
	}
}

func TestPickup(t *testing.T) {
	t.Run("租客取货 1→2", func(t *testing.T) {
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
	})
	t.Run("房东不能取货", func(t *testing.T) {
		st := newStub()
		_, err := Pickup(st, 1, 200)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("err = %v, 期望 ErrForbidden", err)
		}
	})
	t.Run("状态非法拒绝迁移", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusPending // 未支付不可取货
		_, err := Pickup(st, 1, 100)
		if !errors.Is(err, ErrInvalidState) {
			t.Fatalf("err = %v, 期望 ErrInvalidState", err)
		}
	})
	t.Run("重复取货幂等", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusRenting // 已被并发迁移
		ok, err := Pickup(st, 1, 100)
		if err != nil {
			t.Fatalf("幂等调用 err = %v", err)
		}
		if ok {
			t.Fatalf("幂等调用不应发生迁移")
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
	t.Run("房东确认归还 3→4", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusToReturn
		ok, err := ConfirmReturn(st, 1, 200)
		if err != nil || !ok {
			t.Fatalf("ConfirmReturn ok=%v err=%v", ok, err)
		}
		if st.orders[1].Status != models.OrderStatusReturned {
			t.Fatalf("状态 = %d, 期望 %d", st.orders[1].Status, models.OrderStatusReturned)
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
}

func TestBreach(t *testing.T) {
	t.Run("房东判定违约 3→6", func(t *testing.T) {
		st := newStub()
		st.orders[1].Status = models.OrderStatusToReturn
		ok, err := Breach(st, 1, 200)
		if err != nil || !ok {
			t.Fatalf("Breach ok=%v err=%v", ok, err)
		}
		if st.orders[1].Status != models.OrderStatusBreach {
			t.Fatalf("状态 = %d, 期望 %d", st.orders[1].Status, models.OrderStatusBreach)
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
}
