// 订单生命周期流转 + 押金台账（纯逻辑层，DB 经 Store 接口注入以便单测）
package services

import (
	"errors"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/models"
)

// 流转层业务错误（控制器据此映射 HTTP 状态码）
var (
	ErrOrderNotFound = errors.New("订单不存在")
	ErrForbidden     = errors.New("无权操作该订单")
	ErrInvalidState  = errors.New("当前订单状态不允许该操作")
)

// OrderStore 订单流转所需的最小 DB 能力。生产用 defaultOrderStore（ORM），
// 单测注入桩实现，保证流转逻辑可离线验证。
type OrderStore interface {
	// GetOrder 按 ID 读取订单（不存在返回 ErrOrderNotFound）
	GetOrder(id int64) (*models.Order, error)
	// TransitionOrder 条件更新：仅当 status == from 时迁移到 to。
	// n == 0 表示状态已被并发迁移，符合期望即幂等成功。
	TransitionOrder(id int64, from, to int) (int64, error)
	// InsertDeposit 写一条押金流水
	InsertDeposit(d *models.Deposit) error
	// AdjustDepositBal 给用户押金余额增/减 delta（pickup 扣减、return_confirm 返还）
	AdjustDepositBal(userID int64, delta float64) error
}

// ensureRole 身份校验：取货/申请归还需要是租客，确认归还/违约需要是物主
func ensureRole(order *models.Order, uid int64, wantRenter bool) error {
	if wantRenter && order.RenterId == uid {
		return nil
	}
	if !wantRenter && order.OwnerId == uid {
		return nil
	}
	return ErrForbidden
}

// transition 状态流转核心：读单 → 身份/状态校验 → 条件更新。
// 返回 (是否发生了迁移, error)。未迁移分两种情况：
//   - 订单已是目标状态：并发下已处理，幂等返回成功（transitioned=false, err=nil）
//   - 订单处于其他状态：非法迁移，返回 ErrInvalidState
func transition(st OrderStore, orderID, uid int64, wantRenter bool, from, to int) (bool, error) {
	order, err := st.GetOrder(orderID)
	if err != nil {
		return false, err
	}
	if err := ensureRole(order, uid, wantRenter); err != nil {
		return false, err
	}
	switch order.Status {
	case to:
		return false, nil // 幂等：已被并发迁移到目标状态
	case from:
		// 继续条件更新
	default:
		return false, ErrInvalidState
	}
	n, err := st.TransitionOrder(orderID, from, to)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Pickup 取货（1→2 租赁中）：租客操作，冻结押金并同步扣减租客押金余额。
func Pickup(st OrderStore, orderID, uid int64) (bool, error) {
	ok, err := transition(st, orderID, uid, true, models.OrderStatusToPickup, models.OrderStatusRenting)
	if err != nil || !ok {
		return ok, err
	}
	order, err := st.GetOrder(orderID)
	if err != nil {
		return false, err
	}
	amount := round2(order.Deposit)
	if err := st.InsertDeposit(&models.Deposit{
		OrderId: orderID, UserId: order.RenterId,
		Amount: amount, Type: models.DepositTypeFreeze, Ref: order.OrderNo,
	}); err != nil {
		return false, err
	}
	// 押金余额不足不拦截：按冻结语义记录即可
	if err := st.AdjustDepositBal(order.RenterId, -amount); err != nil {
		return false, err
	}
	return true, nil
}

// RequestReturn 申请归还（2→3 待归还）：租客操作，仅状态迁移，不动押金。
func RequestReturn(st OrderStore, orderID, uid int64) (bool, error) {
	return transition(st, orderID, uid, true, models.OrderStatusRenting, models.OrderStatusToReturn)
}

// ConfirmReturn 确认归还（3→4 已归还结算）：物主操作，解冻押金返还租客余额。
// 租金结算已在支付时完成，此处不再重复计算。
func ConfirmReturn(st OrderStore, orderID, uid int64) (bool, error) {
	ok, err := transition(st, orderID, uid, false, models.OrderStatusToReturn, models.OrderStatusReturned)
	if err != nil || !ok {
		return ok, err
	}
	order, err := st.GetOrder(orderID)
	if err != nil {
		return false, err
	}
	amount := round2(order.Deposit)
	if err := st.InsertDeposit(&models.Deposit{
		OrderId: orderID, UserId: order.RenterId,
		Amount: amount, Type: models.DepositTypeUnfreeze, Ref: order.OrderNo,
	}); err != nil {
		return false, err
	}
	if err := st.AdjustDepositBal(order.RenterId, amount); err != nil {
		return false, err
	}
	return true, nil
}

// Breach 违约（3→6 违约扣押金）：物主操作，押金扣除不退还（不增加租客余额）。
func Breach(st OrderStore, orderID, uid int64) (bool, error) {
	ok, err := transition(st, orderID, uid, false, models.OrderStatusToReturn, models.OrderStatusBreach)
	if err != nil || !ok {
		return ok, err
	}
	order, err := st.GetOrder(orderID)
	if err != nil {
		return false, err
	}
	amount := round2(order.Deposit)
	if err := st.InsertDeposit(&models.Deposit{
		OrderId: orderID, UserId: order.RenterId,
		Amount: amount, Type: models.DepositTypeDeduct, Ref: order.OrderNo,
	}); err != nil {
		return false, err
	}
	return true, nil
}

// --- 生产实现：Beego ORM ---

// defaultOrderStore 基于 ORM 的 Store 生产实现
type defaultOrderStore struct {
	o orm.Ormer
}

// NewOrderStore 返回生产 Store
func NewOrderStore() OrderStore {
	return &defaultOrderStore{o: orm.NewOrm()}
}

func (s *defaultOrderStore) GetOrder(id int64) (*models.Order, error) {
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

func (s *defaultOrderStore) TransitionOrder(id int64, from, to int) (int64, error) {
	return s.o.QueryTable(new(models.Order)).
		Filter("id", id).
		Filter("status", from).
		Update(map[string]any{"status": to})
}

func (s *defaultOrderStore) InsertDeposit(d *models.Deposit) error {
	_, err := s.o.Insert(d)
	return err
}

func (s *defaultOrderStore) AdjustDepositBal(userID int64, delta float64) error {
	_, err := s.o.QueryTable(new(models.User)).
		Filter("id", userID).
		Update(map[string]any{"deposit_bal": fmt.Sprintf("deposit_bal + %.2f", delta)})
	return err
}
