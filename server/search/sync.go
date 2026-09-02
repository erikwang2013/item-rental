// 物品索引同步：将物品写入/移出 OpenSearch 索引
package search

import (
	"context"

	"github.com/erikwang2013/go-scout"
	"github.com/erikwang2013/item-rental/server/models"
)

// syncIndex 索引操作的接口抽象。生产实现在 Searchable()；单测可注入假实现，
// 记录 upsert/remove 调用而不依赖真实 OpenSearch。
type syncIndex interface {
	SearchableSync(ctx context.Context, model scout.ScoutModel) error
	UnsearchableSync(ctx context.Context, model scout.ScoutModel) error
	MakeAllSearchable(ctx context.Context, chunk int) error
}

// SyncItem 将单个物品同步到搜索索引（新增/更新时调用）。
// 仅上架物品（ShouldBeSearchable）写入索引，下架/已售罄物品跳过。
func SyncItem(ctx context.Context, item *models.Item) error {
	return syncItem(ctx, Searchable(), item)
}

// RemoveItem 从索引真正删除单个物品（下架/删除时调用），而非 upsert。
func RemoveItem(ctx context.Context, item *models.Item) error {
	return removeItem(ctx, Searchable(), item)
}

// ReindexAll 全量重建索引（启动时或手动触发，chunk 传 0 用默认块大小）。
func ReindexAll(ctx context.Context) error {
	return reindexAll(ctx, Searchable())
}

// syncItem 核心逻辑：未上架物品不入索引。
func syncItem(ctx context.Context, idx syncIndex, item *models.Item) error {
	if !item.ShouldBeSearchable() {
		return nil
	}
	return idx.SearchableSync(ctx, item)
}

// removeItem 核心逻辑：真正从索引删除。
func removeItem(ctx context.Context, idx syncIndex, item *models.Item) error {
	return idx.UnsearchableSync(ctx, item)
}

// reindexAll 核心逻辑：全量重建。
func reindexAll(ctx context.Context, idx syncIndex) error {
	return idx.MakeAllSearchable(ctx, 0)
}