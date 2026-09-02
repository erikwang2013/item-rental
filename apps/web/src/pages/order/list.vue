<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listOrders } from '@/api/order'
const list = ref<any[]>([])
const tabs = ['全部','待支付','待取货','租赁中','待归还','已归还','已取消','违约']
const cur = ref(0)
async function load(){ try { list.value = (await listOrders({page:1,page_size:20,status:cur.value===0?undefined:cur.value-1})) as any } catch{} }
onMounted(load)
function pick(i){ cur.value=i; load() }
</script>
<template>
  <view class="ol">
    <view class="tabs">
      <text v-for="(t,i) in tabs" :key="i" :class="['tab',{on:cur===i}]" @tap="pick(i)">{{t}}</text>
    </view>
    <view v-for="o in list" :key="o.id" class="card" @tap="uni.navigateTo({url:'/pages/order/detail?order_id='+o.id})">
      <text class="no">{{o.order_no}}</text>
      <text class="it">{{o.item_title||'物品'}}</text>
      <text class="st">¥{{o.rent_amount}}</text>
    </view>
  </view>
</template>
<style scoped>
.tabs{display:flex;background:#fff;border-bottom:1rpx solid #eee;overflow-x:auto}
.tab{padding:24rpx 20rpx;font-size:24rpx;color:#666;flex-shrink:0;border-bottom:4rpx solid transparent}
.tab.on{color:#2E7D32;border-bottom-color:#2E7D32}
.card{display:flex;align-items:center;justify-content:space-between;background:#fff;margin:16rpx;padding:24rpx;border-radius:12rpx;box-shadow:0 2rpx 6rpx #eee}
.no{font-size:24rpx;color:#999}
.it{font-size:28rpx;flex:1;margin-left:20rpx}
.st{font-size:28rpx;color:#E53935;font-weight:bold}
</style>
