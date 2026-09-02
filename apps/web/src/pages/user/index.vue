<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getProfile } from '@/api/user'
import { useUserStore } from '@/stores/user'
const u = ref<any>(null)
const store = useUserStore()
onMounted(async () => {
  if (!store.isLoggedIn){ uni.redirectTo({url:'/pages/auth/login'}); return }
  try { u.value = await getProfile(); store.setProfile(u.value) } catch { store.logout() }
})
</script>
<template>
  <view class="me" v-if="u">
    <view class="hd">
      <image :src="u.avatar||'/static/avatar.png'" class="av" />
      <text class="nm">{{ u.nickname || u.phone || '用户' }}</text>
    </view>
    <view class="menu">
      <view class="it" @tap="uni.navigateTo({url:'/pages/user/profile'})"><text>个人资料</text></view>
      <view class="it" @tap="uni.navigateTo({url:'/pages/seller/items'})"><text>我发布的物品</text></view>
      <view class="it" @tap="store.logout()"><text>退出登录</text></view>
    </view>
  </view>
</template>
<style scoped>
.hd{display:flex;align-items:center;padding:40rpx;background:#2E7D32;color:#fff}
.av{width:100rpx;height:100rpx;border-radius:50%;margin-right:24rpx}
.nm{font-size:34rpx;font-weight:bold}
.menu{margin-top:20rpx}
.it{padding:30rpx;background:#fff;border-bottom:1rpx solid #f0f0f0;font-size:28rpx}
</style>
