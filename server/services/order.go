// 订单服务：订单号生成与订单草稿构建（纯函数，无 DB 依赖，可离线单测）
package services

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/erikwang2013/item-rental/server/models"
)

// orderDateLayout 订单日期格式（与 models.Order 的 DATE 字段一致）
const orderDateLayout = "2006-01-02"

// GenerateOrderNo 生成唯一订单号：ORD + 毫秒时间戳 + 8 位随机数。
// 高并发下即使同一毫秒也有 10^8 分之一量级的碰撞空间，DB 唯一索引兜底。
func GenerateOrderNo() string {
	return fmt.Sprintf("ORD%d%08d", time.Now().UnixMilli(), rand.Intn(100000000))
}

// BuildOrder 校验下单输入并构建订单草稿（status=0 待支付），供 controller 落库。
// 校验规则：
//   - 物品必须存在且已上架（status==1）
//   - 租客不能租赁自己的物品（renter != item.OwnerId）
//   - 日期格式必须为 YYYY-MM-DD，且 start <= end
//   - days = end - start + 1，必须 >= 1
//   - rent_amount = days * daily_price（四舍五入到分），deposit = item.deposit
func BuildOrder(item *models.Item, renterID int64, start, end string) (*models.Order, error) {
	if item == nil || item.Id <= 0 {
		return nil, errors.New("物品不存在")
	}
	if renterID <= 0 {
		return nil, errors.New("用户未登录")
	}
	if renterID == item.OwnerId {
		return nil, errors.New("不能租赁自己的物品")
	}
	if item.Status != 1 {
		return nil, errors.New("物品未上架，无法下单")
	}

	startT, err := time.Parse(orderDateLayout, start)
	if err != nil {
		return nil, errors.New("开始日期格式错误")
	}
	endT, err := time.Parse(orderDateLayout, end)
	if err != nil {
		return nil, errors.New("结束日期格式错误")
	}
	if endT.Before(startT) {
		return nil, errors.New("结束日期不能早于开始日期")
	}

	days := int(endT.Sub(startT).Hours()/24) + 1
	if days < 1 {
		return nil, errors.New("租赁天数不合法")
	}

	return &models.Order{
		OrderNo:    GenerateOrderNo(),
		ItemId:     item.Id,
		RenterId:   renterID,
		OwnerId:    item.OwnerId,
		StartDate:  start,
		EndDate:    end,
		Days:       days,
		RentAmount: round2(float64(days) * item.DailyPrice),
		Deposit:    item.Deposit,
		Status:     models.OrderStatusPending,
	}, nil
}

// round2 金额四舍五入到分
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
