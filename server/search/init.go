// go-scout 搜索集成：OpenSearch 引擎初始化与全局访问
package search

import (
	"log"
	"os"
	"sync"

	"github.com/erikwang2013/go-scout"
	"github.com/erikwang2013/go-scout/engines"
	"github.com/erikwang2013/item-rental/server/models"
)

var (
	// scoutInstance 全局 go-scout 实例（驱动由 SCOUT_DRIVER 决定）
	scoutInstance *scout.Scout

	// initOnce 保证懒初始化并发安全（ReindexAll 启动协程与请求处理可能并发触发）
	initOnce sync.Once
)

// Init 懒初始化 go-scout 搜索引擎并注册引擎驱动。可重复调用，幂等。
// 空安全：当 SCOUT_DRIVER 环境变量未配置时，DefaultConfig 会落到 "database" 驱动；
// 而 database 引擎依赖 engines.SetDatabase 提供的连接（本服务从未调用），
// 直接使用会在查询时以 nil *sql.DB panic。此处显式降级为空引擎(null)：
// 返回空结果而非 panic/阻塞，并打印降级通知。
func Init() {
	initOnce.Do(func() {
		// 未配置驱动时降级为空引擎，避免 database 驱动的 nil *sql.DB panic
		if os.Getenv("SCOUT_DRIVER") == "" {
			log.Printf("[search] SCOUT_DRIVER 未配置，搜索索引降级为空引擎（返回空结果，不阻塞）")
			os.Setenv("SCOUT_DRIVER", "null")
		}

		// DefaultConfig 从环境变量读取驱动配置（SCOUT_DRIVER / OPENSEARCH_HTTP_HOST 等）
		cfg := scout.DefaultConfig()

		s := scout.NewWithConfig(cfg)
		// 注册全部引擎驱动（null/collection/database/opensearch/...）
		engines.Register(s.Manager)
		scoutInstance = s
	})
}

// Instance 返回全局 go-scout 实例（首次访问时懒初始化）。
func Instance() *scout.Scout {
	Init()
	return scoutInstance
}

// Searchable 返回物品的 Searchable API，使用 MySQL 数据源回填结果。
func Searchable() *scout.Searchable {
	return Instance().Searchable(&models.Item{}, &ItemSource{})
}
