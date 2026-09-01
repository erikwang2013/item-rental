// go-scout 搜索集成：OpenSearch 引擎初始化与全局访问
package search

import (
	"github.com/erikwang2013/go-scout"
	"github.com/erikwang2013/go-scout/engines"
	"github.com/erikwang2013/item-rental/server/models"
)

var (
	// scoutInstance 全局 go-scout 实例（驱动由 SCOUT_DRIVER 决定，默认 opensearch）
	scoutInstance *scout.Scout
)

// Init 初始化 go-scout 搜索引擎并注册引擎驱动。
// 在 routers init 中调用。
func Init() {
	// DefaultConfig 从环境变量读取驱动配置（SCOUT_DRIVER / OPENSEARCH_HTTP_HOST 等）
	cfg := scout.DefaultConfig()

	s := scout.NewWithConfig(cfg)
	// 注册全部引擎驱动（null/collection/database/opensearch/...）
	engines.Register(s.Manager)
	scoutInstance = s
}

// Instance 返回全局 go-scout 实例。
func Instance() *scout.Scout {
	if scoutInstance == nil {
		Init()
	}
	return scoutInstance
}

// Searchable 返回物品的 Searchable API，使用 MySQL 数据源回填结果。
func Searchable() *scout.Searchable {
	return Instance().Searchable(&models.Item{}, &ItemSource{})
}
