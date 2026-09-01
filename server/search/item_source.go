// ItemSource：go-scout 数据源，从 MySQL 加载物品用于搜索结果回填
package search

import (
	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/go-scout"
	"github.com/erikwang2013/item-rental/server/models"
)

// ItemSource 实现 scout.Source[scout.ScoutModel]，以 MySQL 为后端。
// go-scout 用它在索引命中后回填完整的 Item 模型。
type ItemSource struct{}

var _ scout.Source[scout.ScoutModel] = (*ItemSource)(nil)

// All 返回当前全部物品（含未上架，供全量索引导入）。
func (s *ItemSource) All(ctx any) ([]scout.ScoutModel, error) {
	o := orm.NewOrm()
	var items []models.Item
	if _, err := o.QueryTable(new(models.Item)).All(&items); err != nil {
		return nil, err
	}
	out := make([]scout.ScoutModel, 0, len(items))
	for i := range items {
		out = append(out, &items[i])
	}
	return out, nil
}

// ByIDs 按主键批量加载物品，保持 ids 顺序。
func (s *ItemSource) ByIDs(ctx any, ids []any) ([]scout.ScoutModel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	o := orm.NewOrm()
	var items []models.Item
	if _, err := o.QueryTable(new(models.Item)).Filter("id__in", ids).All(&items); err != nil {
		return nil, err
	}
	// 按请求顺序回填
	byID := make(map[int64]*models.Item, len(items))
	for i := range items {
		byID[items[i].Id] = &items[i]
	}
	out := make([]scout.ScoutModel, 0, len(ids))
	for _, id := range ids {
		if it, ok := byID[toInt64(id)]; ok {
			out = append(out, it)
		}
	}
	return out, nil
}

// Count 返回物品总数。
func (s *ItemSource) Count(ctx any) (int, error) {
	o := orm.NewOrm()
	c, err := o.QueryTable(new(models.Item)).Count()
	return int(c), err
}

// toInt64 将任意主键值转为 int64（支持 int/int64/int32/float64/string）。
func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case string:
		var out int64
		for _, c := range n {
			if c < '0' || c > '9' {
				break
			}
			out = out*10 + int64(c-'0')
		}
		return out
	default:
		return 0
	}
}
