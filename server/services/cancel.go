// 订单取消：0待支付 -> 5已取消（纯逻辑层，DB 经 Store 接口注入以便单测）
package services

import (
	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/models"
)

// CancelStore 取消订单所需的最小 DB 能力。生产用 defaultCancelStore（ORM），
// 单测注入桩实现，保证取消逻辑可离线验证。
type CancelStore interface {
	// GetOrder 按 ID 读取订单（不存在返回 ErrOrderNotFound）
	GetOrder(id int64) (*models.Order, error)
	// CancelOrder 条件更新：仅当 status=0（待支付）时迁移到 5（已取消）并写取消原因。
	// n == 0 表示状态已被并发迁移（已取消/已支付），幂等。
	CancelOrder(id int64, reason string) (int64, error)
}

// CancelOrder 取消订单（0待支付 -> 5已取消）。
// 仅订单租客或房东可操作；已取消幂等返回（transitioned=false, err=nil）；
// 其他状态返回 ErrInvalidState。退款动作由调用方（controller）先执行。
func CancelOrder(st CancelStore, orderID, uid int64, reason string) (bool, error) {
	order, err := st.GetOrder(orderID)
	if err != nil {
		return false, err
	}
	if order.RenterId != uid && order.OwnerId != uid {
		return false, ErrForbidden
	}
	switch order.Status {
	case models.OrderStatusCancelled:
		return false, nil // 幂等：已被并发取消
	case models.OrderStatusPending:
		// 继续条件更新
	default:
		return false, ErrInvalidState
	}
	n, err := st.CancelOrder(orderID, reason)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// --- 生产实现：Beego ORM ---

// defaultCancelStore 基于 ORM 的 Store 生产实现
type defaultCancelStore struct {
	o orm.Ormer
}

// NewCancelStore 返回生产 Store
func NewCancelStore() CancelStore {
	return &defaultCancelStore{o: orm.NewOrm()}
}

func (s *defaultCancelStore) GetOrder(id int64) (*models.Order, error) {
	order := &models.Order{Id: id}
	err := s.o.Read(order)
	if err == orm.ErrNoRows {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *defaultCancelStore) CancelOrder(id int64, reason string) (int64, error) {
	return s.o.QueryTable(new(models.Order)).
		Filter("id", id).
		Filter("status", models.OrderStatusPending).
		Update(map[string]any{"status": models.OrderStatusCancelled, "cancel_reason": reason})
}
