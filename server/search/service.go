// 物品搜索服务：封装 go-scout Builder 链式查询
package search

import (
	"context"
	"fmt"
	"os"

	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/services"
)

// SearchParams 搜索入参
type SearchParams struct {
	Query      string   // 关键字（全文搜索 title/desc）
	CategoryID int64    // 品类过滤，0 表示不过滤
	MinPrice   float64  // 最低日租金，0 表示不限
	MaxPrice   float64  // 最高日租金，0 表示不限
	City       string   // 城市过滤，空表示不限
	OrderBy    string   // 排序字段：default/price_asc/price_desc
	Page       int      // 页码，从 1 开始
	PageSize   int      // 每页条数
	Lat        float64  // 查询中心纬度（配合 RadiusKm 做地理半径过滤）
	Lng        float64  // 查询中心经度（配合 RadiusKm 做地理半径过滤）
	RadiusKm   *float64 // 搜索半径（公里），nil 表示不启用地理过滤
}

// SearchResult 搜索结果
type SearchResult struct {
	Items []*models.Item `json:"items"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
}

// SearchQuery 搜索查询规格：由纯函数 BuildSearchQuery 产出，不含 go-scout/ORM 依赖。
type SearchQuery struct {
	CategoryID int64   // 品类过滤，0 表示不过滤
	City       string  // 城市过滤，空表示不限
	MinPrice   float64 // 最低日租金，0 表示不限
	MaxPrice   float64 // 最高日租金，0 表示不限
	OrderBy    string  // 排序字段名："default"/"" 表示默认排序
	OrderDesc  bool    // 排序方向（OrderBy 非 default 时生效）
	Page       int     // 页码，从 1 开始
	PageSize   int     // 每页条数
}

// 排序白名单：仅允许 default/price_asc/price_desc，其余视为非法入参。
var orderByWhitelist = map[string]bool{
	"default":    true,
	"price_asc":  true,
	"price_desc": true,
}

// geoMode 决定地理过滤实现路径（纯函数）：
//   - "engine":   驱动支持 geo_point 精确过滤（OpenSearch WhereGeoDistance），半径过滤交给引擎、Total 为真值
//   - "haversine": 其余驱动（null/database/collection 等）无 geo DSL 语义，保留逐条 Haversine 懒过滤兜底
//   - "off":       未启用地理过滤（无半径）
func geoMode(driver string, radiusKm *float64) string {
	if radiusKm == nil || *radiusKm <= 0 {
		return "off"
	}
	if driver == "opensearch" {
		return "engine"
	}
	return "haversine"
}

// BuildSearchQuery 将搜索入参转换为查询规格（纯函数，不依赖 go-scout / ORM，可离线单测）：
//   - 规范化 Page/PageSize（<=0 回退默认值 1/20）
//   - order_by 白名单校验，非法值返回错误
//   - 过滤条件（category_id / city / 价格区间）原样透传
func BuildSearchQuery(p SearchParams) (*SearchQuery, error) {
	page, pageSize := p.Page, p.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	if p.OrderBy != "" && !orderByWhitelist[p.OrderBy] {
		return nil, fmt.Errorf("不支持的排序方式: %q", p.OrderBy)
	}

	return &SearchQuery{
		CategoryID: p.CategoryID,
		City:       p.City,
		MinPrice:   p.MinPrice,
		MaxPrice:   p.MaxPrice,
		OrderBy:    p.OrderBy,
		OrderDesc:  p.OrderBy == "price_desc",
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// SearchItems 执行物品搜索。查询规格由 BuildSearchQuery 校验产出，再映射到 go-scout Builder。
func SearchItems(ctx context.Context, p SearchParams) (*SearchResult, error) {
	q, err := BuildSearchQuery(p)
	if err != nil {
		return nil, err
	}

	b := Searchable().Search(ctx, p.Query, nil).Where("status", 1)

	if q.CategoryID > 0 {
		b.Where("category_id", q.CategoryID)
	}
	if q.City != "" {
		b.Where("city", q.City)
	}
	if q.MinPrice > 0 || q.MaxPrice > 0 {
		opts := map[string]any{}
		if q.MinPrice > 0 {
			opts["gte"] = q.MinPrice
		}
		if q.MaxPrice > 0 {
			opts["lte"] = q.MaxPrice
		}
		b.WhereRange("daily_price", opts, true)
	}
	if q.OrderBy != "" && q.OrderBy != "default" {
		dir := "asc"
		if q.OrderDesc {
			dir = "desc"
		}
		b.OrderBy("daily_price", dir)
	}

	// 地理过滤路径：OpenSearch 引擎走 geo_point 精确过滤（前置、Total 真值）；
	// 其余驱动走分页后 Haversine 懒过滤兜底（见下方 post-filter）。
	if geoMode(os.Getenv("SCOUT_DRIVER"), p.RadiusKm) == "engine" {
		b.WhereGeoDistance("location", p.Lat, p.Lng, *p.RadiusKm)
	}

	result, err := b.Paginate(ctx, q.PageSize, q.Page, "page")
	if err != nil {
		return nil, err
	}

	items := make([]*models.Item, 0, len(result.Items))
	for _, it := range result.Items {
		if item, ok := it.Model.(*models.Item); ok {
			items = append(items, item)
		}
	}

	// 地理半径过滤（兜底路径，lazy 近似，无 geo 索引）：仅非 OpenSearch 驱动且
	// 未走引擎过滤时，逐条用 Haversine 距离过滤；物品缺失坐标（lat/lng 为 0）安全跳过。
	total := int64(result.Total)
	if geoMode(os.Getenv("SCOUT_DRIVER"), p.RadiusKm) == "haversine" {
		filtered := make([]*models.Item, 0, len(items))
		for _, item := range items {
			if item.Lat == 0 && item.Lng == 0 {
				continue // 缺失坐标，无法判定距离，跳过
			}
			if services.InRadius(*p.RadiusKm, p.Lat, p.Lng, item.Lat, item.Lng) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
		total = int64(len(items)) // lazy 近似：过滤后 Total 按当前页实收数量返回
	}
	return &SearchResult{Items: items, Total: total, Page: q.Page}, nil
}
