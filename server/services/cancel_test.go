// 订单取消纯逻辑单元测试：注入桩 Store，验证 0->5 迁移、身份/状态守卫、幂等性
package services

import (
	"errors"
	"testing"

	"github.com/erikwang2013/item-rental/server/models"
)

// cancelStub 内存版 CancelStore 桩，记录所有取消写操作以便断言
type cancelStub struct {
	orders  map[int64]*models.Order
	cancels []int64 // 记录 CancelOrder 生效的订单 ID
	reasons map[int64]string
}

func (s *cancelStub) GetOrder(id int64) (*models.Order, error) {
	o, ok := s.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

func (s *cancelStub) CancelOrder(id int64, reason string) (int64, error) {
	o, ok := s.orders[id]
	if !ok {
		return 0, ErrOrderNotFound
	}
	if o.Status != models.OrderStatusPending {
		return 0, nil // 并发下状态已迁移，条件更新不生效
	}
	o.Status = models.OrderStatusCancelled
	o.CancelReason = reason
	s.cancels = append(s.cancels, id)
	s.reasons[id] = reason
	return 1, nil
}

// newCancelStub 构造一个待支付(0)的订单：租客=100，房东=200
func newCancelStub() *cancelStub {
	return &cancelStub{
		orders:  map[int64]*models.Order{1: {Id: 1, RenterId: 100, OwnerId: 200, Status: models.OrderStatusPending, OrderNo: "O20260901001"}},
		cancels: nil,
		reasons: map[int64]string{},
	}
}

func TestCancelOrder(t *testing.T) {
	t.Run("租客取消 0->5 并写取消原因", func(t *testing.T) {
		st := newCancelStub()
		ok, err := CancelOrder(st, 1, 100, "行程变更")
		if err != nil || !ok {
			t.Fatalf("CancelOrder ok=%v err=%v", ok, err)
		}
		if st.orders[1].Status != models.OrderStatusCancelled {
			t.Fatalf("状态 = %d, 期望 %d", st.orders[1].Status, models.OrderStatusCancelled)
		}
		if st.orders[1].CancelReason != "行程变更" {
			t.Fatalf("cancel_reason = %q, 期望 行程变更", st.orders[1].CancelReason)
		}
		if len(st.cancels) != 1 {
			t.Fatalf("CancelOrder 应恰好生效一次, got %v", st.cancels)
		}
	})

	t.Run("房东取消 0->5 也可操作", func(t *testing.T) {
		st := newCancelStub()
		ok, err := CancelOrder(st, 1, 200, "房东下架")
		if err != nil || !ok {
			t.Fatalf("房东取消 ok=%v err=%v", ok, err)
		}
		if st.orders[1].Status != models.OrderStatusCancelled {
			t.Fatalf("状态 = %d, 期望 %d", st.orders[1].Status, models.OrderStatusCancelled)
		}
	})

	t.Run("已取消幂等返回", func(t *testing.T) {
		st := newCancelStub()
		st.orders[1].Status = models.OrderStatusCancelled // 已被并发取消
		ok, err := CancelOrder(st, 1, 100, "重复")
		if err != nil {
			t.Fatalf("幂等调用 err = %v", err)
		}
		if ok {
			t.Fatalf("幂等调用不应再发生迁移")
		}
		if len(st.cancels) != 0 {
			t.Fatalf("幂等调用不应重复写取消: %v", st.cancels)
		}
	})

	t.Run("已支付订单不可取消", func(t *testing.T) {
		st := newCancelStub()
		st.orders[1].Status = models.OrderStatusToPickup // 已付租金
		_, err := CancelOrder(st, 1, 100, "")
		if !errors.Is(err, ErrInvalidState) {
			t.Fatalf("err = %v, 期望 ErrInvalidState", err)
		}
		if len(st.cancels) != 0 {
			t.Fatalf("不应执行取消写操作: %v", st.cancels)
		}
	})

	t.Run("非参与者不可取消", func(t *testing.T) {
		st := newCancelStub()
		_, err := CancelOrder(st, 1, 300, "")
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("err = %v, 期望 ErrForbidden", err)
		}
	})

	t.Run("订单不存在", func(t *testing.T) {
		st := newCancelStub()
		_, err := CancelOrder(st, 99, 100, "")
		if !errors.Is(err, ErrOrderNotFound) {
			t.Fatalf("err = %v, 期望 ErrOrderNotFound", err)
		}
	})
}
