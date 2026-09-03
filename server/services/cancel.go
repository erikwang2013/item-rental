// 订单取消：0待支付 -> 5已取消（纯逻辑层，DB 经 Store 接口注入以便单测）
package services

import (
	"fmt"

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
	// Notify 向用户写一条站内信
	Notify(userID int64, typ, title, content string) error
	// AdjustCredit 调整用户信用分（clamp 0-100）
	AdjustCredit(userID int64, delta int) error
	// InsertCreditEvent 写一条信用分流水
	InsertCreditEvent(e *models.CreditEvent) error
	// HasPaidPayment 该订单是否存在已成功/已退款的支付单（用于"已支付后取消"判定）
	HasPaidPayment(orderID int64) bool
}

// CancelOrder 取消订单（0待支付 -> 5已取消）。
// 仅订单租客或房东可操作；已取消幂等返回（transitioned=false, err=nil）；
// 其他状态返回 ErrInvalidState。退款动作由调用方（controller）先执行。
// 取消生效后：通知对方；若曾支付过则租客信用 -10。
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
	ok := n > 0
	if !ok {
		return false, nil
	}

	// 取消生效：通知对方（租客取消→通知房东，反之亦然）
	other := order.OwnerId
	if order.RenterId != uid {
		other = order.RenterId
	}
	if err := st.Notify(other, "order_cancelled", "订单已取消", fmt.Sprintf("订单 %s 已被取消", order.OrderNo)); err != nil {
		return false, err
	}
	// 已支付后取消：租客信用 -10（退款由 controller 先执行）
	if st.HasPaidPayment(orderID) {
		if err := st.AdjustCredit(order.RenterId, -10); err != nil {
			return false, err
		}
		if err := st.InsertCreditEvent(&models.CreditEvent{
			UserId: order.RenterId, Change: -10, Reason: "cancel_after_paid", Ref: order.OrderNo,
		}); err != nil {
			return false, err
		}
	}
	return true, nil
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

func (s *defaultCancelStore) Notify(userID int64, typ, title, content string) error {
	return Send(userID, typ, title, content)
}

func (s *defaultCancelStore) AdjustCredit(userID int64, delta int) error {
	// SQL 表达式须走 Raw:ORMMap Update 的值会被当绑定参数(字符串字面量),SET 到 INT 列必报错
	_, err := s.o.Raw("UPDATE users SET credit_score = GREATEST(LEAST(credit_score + ?, 100), 0) WHERE id = ?",
		delta, userID).Exec()
	return err
}

func (s *defaultCancelStore) InsertCreditEvent(e *models.CreditEvent) error {
	e.Id = NextID()
	_, err := s.o.Insert(e)
	return err
}

func (s *defaultCancelStore) HasPaidPayment(orderID int64) bool {
	n, err := s.o.QueryTable(new(models.Payment)).
		Filter("order_id", orderID).
		Filter("status__in", []int{models.PaymentStatusSuccess, models.PaymentStatusRefunded}).
		Count()
	return err == nil && n > 0
}
