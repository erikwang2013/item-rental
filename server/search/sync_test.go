// 索引同步核心逻辑单测：不依赖真实 OpenSearch，用假索引记录 upsert/remove 调用
package search

import (
	"context"
	"testing"

	"github.com/erikwang2013/go-scout"
	"github.com/erikwang2013/item-rental/server/models"
)

// fakeIndex 记录索引操作，验证哪些物品被 upsert/删除。
type fakeIndex struct {
	upserted  []int64
	removed   []int64
	reindexed bool
}

func (f *fakeIndex) SearchableSync(_ context.Context, model scout.ScoutModel) error {
	f.upserted = append(f.upserted, model.ScoutKey().(int64))
	return nil
}

func (f *fakeIndex) UnsearchableSync(_ context.Context, model scout.ScoutModel) error {
	f.removed = append(f.removed, model.ScoutKey().(int64))
	return nil
}

func (f *fakeIndex) MakeAllSearchable(_ context.Context, _ int) error {
	f.reindexed = true
	return nil
}

func has(vs []int64, v int64) bool {
	for _, x := range vs {
		if x == v {
			return true
		}
	}
	return false
}

// TestSyncItemSkipsOffShelf 下架(0)/已售罄(2)物品不应被 upsert 进索引。
func TestSyncItemSkipsOffShelf(t *testing.T) {
	for _, status := range []int{0, 2} {
		f := &fakeIndex{}
		item := &models.Item{Id: int64(status), Status: status}
		if err := syncItem(context.Background(), f, item); err != nil {
			t.Fatalf("syncItem(status=%d) error: %v", status, err)
		}
		if len(f.upserted) != 0 || len(f.removed) != 0 {
			t.Fatalf("status=%d 物品不应触碰索引，got upserted=%v removed=%v", status, f.upserted, f.removed)
		}
	}
}

// TestSyncItemIndexesOnShelf 上架物品正常 upsert。
func TestSyncItemIndexesOnShelf(t *testing.T) {
	f := &fakeIndex{}
	item := &models.Item{Id: 1, Status: 1}
	if err := syncItem(context.Background(), f, item); err != nil {
		t.Fatalf("syncItem error: %v", err)
	}
	if !has(f.upserted, 1) {
		t.Fatalf("上架物品应被 upsert，got upserted=%v", f.upserted)
	}
	if len(f.removed) != 0 {
		t.Fatalf("上架物品不应被删除，got removed=%v", f.removed)
	}
}

// TestRemoveItemDeletes 下架/删除走真正删除路径，不留孤儿文档在索引里。
func TestRemoveItemDeletes(t *testing.T) {
	f := &fakeIndex{}
	item := &models.Item{Id: 7, Status: 0}
	if err := removeItem(context.Background(), f, item); err != nil {
		t.Fatalf("removeItem error: %v", err)
	}
	if !has(f.removed, 7) {
		t.Fatalf("应调用删除操作，got removed=%v", f.removed)
	}
	if len(f.upserted) != 0 {
		t.Fatalf("下架不应 upsert，got upserted=%v", f.upserted)
	}
}

// TestReindexAll 全量重建调用 MakeAllSearchable。
func TestReindexAll(t *testing.T) {
	f := &fakeIndex{}
	if err := reindexAll(context.Background(), f); err != nil {
		t.Fatalf("reindexAll error: %v", err)
	}
	if !f.reindexed {
		t.Fatal("应触发全量重建")
	}
}