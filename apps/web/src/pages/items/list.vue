<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listItems } from '@/api/item'
const items = ref<any[]>([])
const catId = ref<number | undefined>()
onMounted(async () => {
  const q = (uni as any).getStorageSync && undefined
  try { items.value = (await listItems({ page:1, page_size:20, category_id: catId.value })) as any } catch {}
})
</script>
<template>
  <view class="list">
    <view v-for="it in items" :key="it.id" class="row" @tap="uni.navigateTo({url:'/pages/items/detail?id='+it.id})">
      <image :src="it.images" mode="aspectFill" class="thumb" />
      <view class="info">
        <text class="name">{{ it.title }}</text>
        <text class="city">{{ it.city }}</text>
        <text class="price">¥{{ it.daily_price }}/天 · 押金¥{{ it.deposit }}</text>
      </view>
    </view>
  </view>
</template>
<style scoped>
.row{display:flex;background:#fff;margin:16rpx 0;border-radius:16rpx;overflow:hidden;box-shadow:0 2rpx 6rpx #eee}
.thumb{width:200rpx;height:160rpx;flex-shrink:0}
.info{padding:16rpx;display:flex;flex-direction:column;justify-content:space-between}
.name{font-size:28rpx;font-weight:bold}
.city{font-size:22rpx;color:#999}
.price{font-size:26rpx;color:#E53935;font-weight:bold}
</style>
