<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { searchItems } from '@/api/item'
const kw = ref('')
const items = ref<any[]>([])
onMounted(async () => {
  const p = new URLSearchParams(location.search)
  kw.value = p.get('keyword') || ''
  await doSearch()
})
async function doSearch(){ try { const d:any=await searchItems({keyword:kw.value,page:1,page_size:20}); items.value=d.list||[] } catch{} }
</script>
<template>
  <view class="search">
    <view class="bar"><input v-model="kw" placeholder="关键字…" /><text class="go" @tap="doSearch">搜索</text></view>
    <view v-for="it in items" :key="it.id" class="row" @tap="uni.navigateTo({url:'/pages/items/detail?id='+it.id})">
      <text class="name">{{ it.title }}</text>
      <text class="price">¥{{ it.daily_price }}/天</text>
    </view>
  </view>
</template>
<style scoped>
.bar{display:flex;padding:16rpx;gap:16rpx}
.bar input{flex:1;background:#f5f5f5;border-radius:30rpx;padding:0 24rpx;height:64rpx;font-size:28rpx}
.go{color:#2E7D32;font-size:28rpx;align-self:center}
.row{display:flex;justify-content:space-between;padding:24rpx;border-bottom:1rpx solid #eee}
.name{font-size:28rpx}
.price{font-size:26rpx;color:#E53935;font-weight:bold}
</style>
