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

// PendingOrderHold 待支付订单占用库存的支付窗口；超时订单由后台任务取消
const PendingOrderHold = 30 * time.Minute

// ErrItemUnavailable 所选日期内物品已被租满
var ErrItemUnavailable = errors.New("该物品在所选日期已被租出")

// DateRange 闭区间日期范围（含首尾两天）
type DateRange struct {
	Start, End time.Time
}

// ParseDateRange 解析 YYYY-MM-DD 闭区间
func ParseDateRange(start, end string) (DateRange, error) {
	s, err := time.Parse(orderDateLayout, start)
	if err != nil {
		return DateRange{}, err
	}
	e, err := time.Parse(orderDateLayout, end)
	if err != nil {
		return DateRange{}, err
	}
	return DateRange{Start: s, End: e}, nil
}

// PeakConcurrent 计算 newRange 内任意一天的最大同时占用数（含新订单本身）。
// 逐日遍历 newRange（BuildOrder 已限制最多 365 天）。
func PeakConcurrent(newRange DateRange, existing []DateRange) int {
	peak := 0
	for d := newRange.Start; !d.After(newRange.End); d = d.AddDate(0, 0, 1) {
		active := 1
		for _, r := range existing {
			if !d.Before(r.Start) && !d.After(r.End) {
				active++
			}
		}
		if active > peak {
			peak = active
		}
	}
	return peak
}

// CheckAvailability 库存校验：租期内峰值并发超过库存即不可租
func CheckAvailability(newRange DateRange, existing []DateRange, stock int) error {
	if PeakConcurrent(newRange, existing) > stock {
		return ErrItemUnavailable
	}
	return nil
}

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
//   - 开始日期不得早于今天（服务器本地时区，今天允许）
//   - days = end - start + 1，必须在 1..365 之间
//   - rent_amount = days * daily_price（四舍五入到分）
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

	// 日期串按 UTC 解析，这里用本地日历的今天构造同基准零点，避免时区偏移误判
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if startT.Before(today) {
		return nil, errors.New("开始日期不能早于今天")
	}

	days := int(endT.Sub(startT).Hours()/24) + 1
	if days < 1 {
		return nil, errors.New("租赁天数不合法")
	}
	if days > 365 {
		return nil, errors.New("租赁天数不能超过 365 天")
	}

	return &models.Order{
		Id:         NextID(),
		OrderNo:    GenerateOrderNo(),
		ItemId:     item.Id,
		RenterId:   renterID,
		OwnerId:    item.OwnerId,
		StartDate:  start,
		EndDate:    end,
		Days:       days,
		RentAmount: round2(float64(days) * item.DailyPrice),
		Status:     models.OrderStatusPending,
	}, nil
}

// round2 金额四舍五入到分
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
