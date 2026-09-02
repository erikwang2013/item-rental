<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getItem } from '@/api/item'
import { useUserStore } from '@/stores/user'
const item = ref<any>(null)
onMounted(async () => {
  const p = new URLSearchParams(location.search)
  const id = Number(p.get('id'))
  try { item.value = await getItem(id) } catch {}
})
function goPay(){
  if (!useUserStore().isLoggedIn){ uni.navigateTo({url:'/pages/auth/login'}); return }
  uni.navigateTo({ url:'/pages/order/confirm?id=' + item.value?.id })
}
</script>
<template>
  <view class="detail" v-if="item">
    <image :src="item.images" mode="widthFix" class="img" />
    <view class="body">
      <text class="title">{{ item.title }}</text>
      <text class="city">{{ item.city }}</text>
      <view class="meta">
        <text class="price">¥{{ item.daily_price }}<text class="small">/天</text></text>
        <text class="dep">押金 ¥{{ item.deposit }}</text>
      </view>
      <text class="desc">{{ item.desc }}</text>
    </view>
    <view class="foot"><button class="btn" @tap="goPay">立即租用</button></view>
  </view>
</template>
<style scoped>
.img{width:100%;height:400rpx}
.body{padding:24rpx}
.title{font-size:34rpx;font-weight:bold;display:block}
.city{font-size:24rpx;color:#999;display:block;margin-top:8rpx}
.meta{display:flex;align-items:baseline;justify-content:space-between;margin:20rpx 0}
.price{font-size:40rpx;color:#E53935;font-weight:bold}
.small{font-size:24rpx}
.dep{font-size:26rpx;color:#666}
.desc{font-size:26rpx;color:#444;line-height:1.6}
.foot{position:fixed;left:0;right:0;bottom:0;padding:16rpx;background:#fff;border-top:1rpx solid #eee}
.btn{width:100%;background:#2E7D32;color:#fff;height:80rpx;font-size:32rpx;border-radius:40rpx}
</style>
