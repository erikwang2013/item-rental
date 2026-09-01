// 订单服务接口：支付模块只依赖此接口，避免把订单状态机逻辑写死在支付模块里。
// 订单状态机的具体实现属于独立任务（M4），本包提供默认的 ORM 实现占位，
// 后续可替换为独立的 order 服务实现而不影响支付网关。
package payments

import (
	"github.com/erikwang2013/item-rental/server/models"
)

// OrderService 定义支付模块需要的外部订单能力。
type OrderService interface {
	// GetOrder 根据订单号查询订单
	GetOrder(orderNo string) (*models.Order, error)
	// MarkPaid 标记订单已支付。
	// 由订单状态机负责状态迁移（0待支付 -> 1待取）与并发保护。
	// transactionId 为微信支付单号。
	MarkPaid(orderNo, transactionId string) error
}

// orderServiceImpl 基于 beego ORM 的默认实现。
// 注意：此处仅为占位，订单完整状态机（押金冻结、并发、补偿）在订单模块任务中实现。
type orderServiceImpl struct{}

// NewOrderService 返回默认订单服务实现。
func NewOrderService() OrderService {
	return &orderServiceImpl{}
}

// GetOrder 按订单号查询订单
func (s *orderServiceImpl) GetOrder(orderNo string) (*models.Order, error) {
	o := ormer()
	order := &models.Order{OrderNo: orderNo}
	if err := o.Read(order, "order_no"); err != nil {
		return nil, err
	}
	return order, nil
}

// MarkPaid 标记订单已支付（0 -> 1）。
// 使用条件更新：仅当 status=0（待支付）时迁移到已支付，
// 天然保证幂等——重复调用不会把已支付的订单再次迁移。
func (s *orderServiceImpl) MarkPaid(orderNo, transactionId string) error {
	o := ormer()
	// 先校验订单存在
	order := &models.Order{OrderNo: orderNo}
	if err := o.Read(order, "order_no"); err != nil {
		return err
	}
	if order.Status != models.OrderStatusPending {
		// 已不是待支付状态，视为已处理（幂等），直接返回成功
		return nil
	}
	order.Status = models.OrderStatusToPickup
	order.PayTradeNo = transactionId
	// 条件更新：仅当原 status 仍为待支付时生效，防止并发重复迁移
	n, err := o.QueryTable(new(models.Order)).
		Filter("id", order.Id).
		Filter("status", models.OrderStatusPending).
		Update(map[string]any{"status": models.OrderStatusToPickup, "pay_trade_no": transactionId})
	if err != nil {
		return err
	}
	if n == 0 {
		// 并发下已被其他请求迁移，幂等返回成功
		return nil
	}
	return nil
}
