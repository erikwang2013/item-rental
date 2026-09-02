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
	// MarkRefunded 标记支付单已退款。
	// outTradeNo 为商户订单号（payments.out_trade_no）。
	// 条件更新支付单 1成功 -> 3已退款（幂等），并把关联订单从 1待取 重置回 0待支付，
	// 使订单重新可处理（重新支付或取消）。
	MarkRefunded(outTradeNo string) error
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

// MarkRefunded 标记支付单已退款（1 -> 3）并把订单重置回待支付（1 -> 0）。
// outTradeNo 为支付单的 out_trade_no（退款回调/退款端点传此值）。
// 幂等：支付单不在"已成功"状态时条件更新不生效，直接返回成功；
// 订单不在"待取"状态时跳过订单重置（并发/重复退款安全）。
func (s *orderServiceImpl) MarkRefunded(outTradeNo string) error {
	o := ormer()

	// 以 out_trade_no（唯一键）定位支付单
	pay := &models.Payment{OutTradeNo: outTradeNo}
	if err := o.Read(pay, "out_trade_no"); err != nil {
		return err
	}

	// 条件更新支付单：仅当 1成功 -> 3已退款，天然幂等
	n, err := o.QueryTable(new(models.Payment)).
		Filter("id", pay.Id).
		Filter("status", models.PaymentStatusSuccess).
		Update(map[string]any{"status": models.PaymentStatusRefunded})
	if err != nil {
		return err
	}
	if n == 0 {
		// 支付单不在已成功状态（已退款/待支付），幂等返回成功
		return nil
	}

	// 订单关联：已付租金待取(1)的订单回到待支付(0)，可重新支付或取消
	order := &models.Order{Id: pay.OrderId}
	if err := o.Read(order); err != nil {
		return err
	}
	if order.Status == models.OrderStatusToPickup {
		_, err := o.QueryTable(new(models.Order)).
			Filter("id", order.Id).
			Filter("status", models.OrderStatusToPickup).
			Update(map[string]any{"status": models.OrderStatusPending})
		if err != nil {
			return err
		}
	}
	return nil
}
