// 物品索引同步：将物品写入/移出 OpenSearch 索引
package search

import (
	"context"

	"github.com/erikwang2013/item-rental/server/models"
)

// SyncItem 将单个物品同步到搜索索引（新增/更新时调用）。
// go-scout 依据 ShouldBeSearchable 决定写入还是移除（未上架自动移出）。
func SyncItem(ctx context.Context, item *models.Item) error {
	return Searchable().SearchableSync(ctx, item)
}

// RemoveItem 从索引移除单个物品（下架/删除时调用）。
func RemoveItem(ctx context.Context, item *models.Item) error {
	return Searchable().Searchable(ctx, item)
}

// ReindexAll 全量重建索引（启动时或手动触发，chunk 传 0 用默认块大小）。
func ReindexAll(ctx context.Context) error {
	return Searchable().MakeAllSearchable(ctx, 0)
}
