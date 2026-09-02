<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listCategories } from '@/api/category'
import { searchItems } from '@/api/item'

const cats = ref<any[]>([])
const items = ref<any[]>([])
const kw = ref('')
onMounted(async () => {
  try { cats.value = await listCategories() } catch {}
  try { const d: any = await searchItems({ page: 1, page_size: 10 }); items.value = d.list || [] } catch {}
})
function goList(catId?: number) { uni.navigateTo({ url: '/pages/items/list' + (catId ? '?category_id='+catId : '') }) }
function goSearch() { if (kw.value.trim()) uni.navigateTo({ url: '/pages/items/search?keyword=' + encodeURIComponent(kw.value) }) }
</script>
<template>
  <view class="home">
    <view class="search" @tap="goSearch">
      <input v-model="kw" placeholder="搜索物品…" />
    </view>
    <view class="cats">
      <view v-for="c in cats" :key="c.id" class="cat" @tap="goList(c.id)">
        <text>{{ c.name }}</text>
      </view>
    </view>
    <view class="sec"><text class="t">推荐物品</text></view>
    <view class="grid">
      <view v-for="it in items" :key="it.id" class="card" @tap="uni.navigateTo({url:'/pages/items/detail?id='+it.id})">
        <image :src="it.images" mode="aspectFill" />
        <text class="name">{{ it.title }}</text>
        <text class="price">¥{{ it.daily_price }}/天</text>
      </view>
    </view>
  </view>
</template>
<style scoped>
.home{padding:20rpx}
.search{background:#f5f5f5;border-radius:30rpx;padding:16rpx 24rpx;margin-bottom:20rpx}
.search input{font-size:28rpx;height:50rpx}
.cats{display:flex;flex-wrap:wrap;gap:16rpx;margin-bottom:20rpx}
.cat{background:#e8f5e9;padding:16rpx 24rpx;border-radius:30rpx;font-size:24rpx;color:#2E7D32}
.t{font-size:30rpx;font-weight:bold;margin:20rpx 0}
.grid{display:flex;flex-wrap:wrap;gap:16rpx}
.card{width:calc(50% - 8rpx);background:#fff;border-radius:16rpx;overflow:hidden;box-shadow:0 2rpx 8rpx #eee}
.card image{width:100%;height:200rpx}
.card .name{display:block;padding:10rpx 16rpx 0;font-size:24rpx;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.card .price{display:block;padding:6rpx 16rpx 16rpx;font-size:26rpx;color:#E53935;font-weight:bold}
</style>
