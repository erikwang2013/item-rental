<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listItems } from '@/api/item'
const items = ref<any[]>([])
onMounted(async () => { try { items.value = (await listItems({page:1,page_size:20})) as any } catch{} })
</script>
<template>
  <view class="seller">
    <button class="add" @tap="uni.showToast({title:'发布暂用接口',icon:'none'})">+ 发布物品</button>
    <view v-for="it in items" :key="it.id" class="row">
      <image :src="it.images" class="thumb" mode="aspectFill"/>
      <view><text class="n">{{it.title}}</text><text class="s">¥{{it.daily_price}}/天</text></view>
    </view>
  </view>
</template>
<style scoped>
.add{margin:20rpx;background:#2E7D32;color:#fff;height:70rpx;font-size:28rpx;border-radius:35rpx}
.row{display:flex;align-items:center;background:#fff;margin-bottom:16rpx;padding:16rpx;border-radius:12rpx}
.thumb{width:100rpx;height:100rpx;border-radius:8rpx;margin-right:16rpx}
.n{font-size:26rpx;display:block}
.s{font-size:24rpx;color:#E53935}
</style>
