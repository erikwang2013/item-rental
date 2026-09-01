// 物品搜索服务：封装 go-scout Builder 链式查询
package search

import (
	"context"

	"github.com/erikwang2013/item-rental/server/models"
)

// SearchParams 搜索入参
type SearchParams struct {
	Query      string  // 关键字（全文搜索 title/desc）
	CategoryID int64   // 品类过滤，0 表示不过滤
	MinPrice   float64 // 最低日租金，0 表示不限
	MaxPrice   float64 // 最高日租金，0 表示不限
	City       string  // 城市过滤，空表示不限
	OrderBy    string  // 排序字段：default/price_asc/price_desc
	Page       int     // 页码，从 1 开始
	PageSize   int     // 每页条数
}

// SearchResult 搜索结果
type SearchResult struct {
	Items []*models.Item `json:"items"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
}

// SearchItems 执行物品搜索，返回回填后的模型列表。
// 使用 go-scout Builder 链：Search → Where/WhereRange → OrderBy → Paginate。
func SearchItems(ctx context.Context, p SearchParams) (*SearchResult, error) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}

	b := Searchable().Search(ctx, p.Query, nil).
		Where("status", 1) // 仅搜索上架物品

	if p.CategoryID > 0 {
		b.Where("category_id", p.CategoryID)
	}
	if p.City != "" {
		b.Where("city", p.City)
	}
	if p.MinPrice > 0 || p.MaxPrice > 0 {
		opts := map[string]any{}
		if p.MinPrice > 0 {
			opts["gte"] = p.MinPrice
		}
		if p.MaxPrice > 0 {
			opts["lte"] = p.MaxPrice
		}
		b.WhereRange("daily_price", opts, true)
	}

	switch p.OrderBy {
	case "price_asc":
		b.OrderBy("daily_price", "asc")
	case "price_desc":
		b.OrderBy("daily_price", "desc")
	}

	result, err := b.Paginate(ctx, p.PageSize, p.Page, "page")
	if err != nil {
		return nil, err
	}

	items := make([]*models.Item, 0, len(result.Items))
	for _, it := range result.Items {
		if item, ok := it.Model.(*models.Item); ok {
			items = append(items, item)
		}
	}

	return &SearchResult{
		Items: items,
		Total: int64(result.Total),
		Page:  p.Page,
	}, nil
}
