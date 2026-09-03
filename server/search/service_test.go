// 搜索 DSL 纯函数单元测试：无 go-scout/ORM/基建依赖，可离线运行。
package search

import (
	"context"
	"os"
	"testing"
)

// TestMain 确保本包测试在确定的降级空引擎下运行：
// 显式将 SCOUT_DRIVER 置为 null 再懒初始化，避免依赖运行环境变量，
// 保证 SearchItems 空安全路径（空结果 / nil error / 不 panic 不阻塞）可复现。
func TestMain(m *testing.M) {
	_ = os.Setenv("SCOUT_DRIVER", "null")
	Init()
	os.Exit(m.Run())
}

func TestBuildSearchQueryOrderByWhitelist(t *testing.T) {
	tests := []struct {
		name    string
		orderBy string
		wantQ   bool // 是否预期合法（进入正常构造流程）
		wantErr bool
	}{
		{name: "空字符串走默认排序", orderBy: "", wantQ: true},
		{name: "default 合法", orderBy: "default", wantQ: true},
		{name: "price_asc 合法", orderBy: "price_asc", wantQ: true},
		{name: "price_desc 合法", orderBy: "price_desc", wantQ: true},
		{name: "非法字段被拒", orderBy: "created_at", wantErr: true},
		{name: "随机字符串被拒", orderBy: "foobar", wantErr: true},
		{name: "大写变体被拒", orderBy: "PRICE_ASC", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := BuildSearchQuery(SearchParams{OrderBy: tt.orderBy})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("BuildSearchQuery(order_by=%q) 期望报错，实际 nil", tt.orderBy)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildSearchQuery(order_by=%q) 不应报错: %v", tt.orderBy, err)
			}
			if !tt.wantQ {
				return
			}
			if q == nil {
				t.Fatalf("BuildSearchQuery(order_by=%q) 返回 nil 规格", tt.orderBy)
			}
			if q.OrderBy != tt.orderBy {
				t.Errorf("OrderBy = %q, 期望 %q", q.OrderBy, tt.orderBy)
			}
			// price_desc → OrderDesc=true；其余（含 default/空）→ false
			if wantDesc := tt.orderBy == "price_desc"; q.OrderDesc != wantDesc {
				t.Errorf("OrderDesc = %v, 期望 %v", q.OrderDesc, wantDesc)
			}
		})
	}
}

func TestBuildSearchQueryPaginationDefaults(t *testing.T) {
	q, err := BuildSearchQuery(SearchParams{Page: 0, PageSize: 0})
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if q.Page != 1 || q.PageSize != 20 {
		t.Errorf("默认分页错误: page=%d pageSize=%d, 期望 1/20", q.Page, q.PageSize)
	}

	// 负数同样回退默认值
	q, err = BuildSearchQuery(SearchParams{Page: -3, PageSize: -5})
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if q.Page != 1 || q.PageSize != 20 {
		t.Errorf("负数分页未回退: page=%d pageSize=%d", q.Page, q.PageSize)
	}

	// 合法值原样保留
	q, err = BuildSearchQuery(SearchParams{Page: 3, PageSize: 50})
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if q.Page != 3 || q.PageSize != 50 {
		t.Errorf("合法分页被改写: page=%d pageSize=%d", q.Page, q.PageSize)
	}
}

func TestBuildSearchQueryFilterPassthrough(t *testing.T) {
	q, err := BuildSearchQuery(SearchParams{
		CategoryID: 7,
		City:       "上海",
		MinPrice:   10,
		MaxPrice:   200,
	})
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if q.CategoryID != 7 || q.City != "上海" || q.MinPrice != 10 || q.MaxPrice != 200 {
		t.Errorf("过滤条件未透传: %+v", q)
	}

	// 零值表示不过滤，原样保留
	q, err = BuildSearchQuery(SearchParams{})
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if q.CategoryID != 0 || q.City != "" || q.MinPrice != 0 || q.MaxPrice != 0 {
		t.Errorf("零值过滤条件被改写: %+v", q)
	}
}

// TestSearchItemsDegradedEmpty 空安全验证：在降级空引擎下，
// SearchItems 返回空结果且不 panic、不阻塞（回归基线：旧实现会因 database 驱动
// 的 nil *sql.DB 在查询时 panic）。
func TestSearchItemsDegradedEmpty(t *testing.T) {
	res, err := SearchItems(context.Background(), SearchParams{Query: "相机"})
	if err != nil {
		t.Fatalf("降级搜索不应报错: %v", err)
	}
	if res == nil {
		t.Fatal("降级搜索返回 nil 结果")
	}
	if res.Total != 0 || len(res.Items) != 0 {
		t.Errorf("降级搜索应返回空结果, got total=%d items=%d", res.Total, len(res.Items))
	}
	if res.Page != 1 {
		t.Errorf("Page 应为 1, got %d", res.Page)
	}

	// 非法 order_by 在进入引擎前被纯函数拦截
	if _, err := SearchItems(context.Background(), SearchParams{OrderBy: "hack"}); err == nil {
		t.Error("非法 order_by 应报错")
	}
}

// TestGeoMode 地理过滤路径判定矩阵：driver/半径 → engine/haversine/off。
func TestGeoMode(t *testing.T) {
	km := 5.0
	zero := 0.0
	cases := []struct {
		name   string
		driver string
		radius *float64
		want   string
	}{
		{"无半径", "opensearch", nil, "off"},
		{"半径为 0", "opensearch", &zero, "off"},
		{"opensearch+半径 → engine", "opensearch", &km, "engine"},
		{"null 驱动兜底 haversine", "null", &km, "haversine"},
		{"database 驱动兜底 haversine", "database", &km, "haversine"},
		{"collection 驱动兜底 haversine", "collection", &km, "haversine"},
	}
	for _, c := range cases {
		if got := geoMode(c.driver, c.radius); got != c.want {
			t.Errorf("%s: geoMode(%q) = %q, want %q", c.name, c.driver, got, c.want)
		}
	}
}
