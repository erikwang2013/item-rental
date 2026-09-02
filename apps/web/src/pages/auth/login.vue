<script setup lang="ts">
import { ref } from 'vue'
import { sms, login } from '@/api/auth'
import { useUserStore } from '@/stores/user'

const phone = ref('')
const code = ref('')
const countdown = ref(0)
const loading = ref(false)
const store = useUserStore()

async function sendSms() {
  if (!/^1\d{10}$/.test(phone.value)) return uni.showToast({ title: '手机号格式错误', icon: 'none' })
  if (countdown.value > 0) return
  await sms(phone.value)
  uni.showToast({ title: '验证码已发送' })
  countdown.value = 60
  const t = setInterval(() => { countdown.value--; if (countdown.value <= 0) clearInterval(t) }, 1000)
}

async function doLogin() {
  if (!phone.value || !code.value) return uni.showToast({ title: '请填写完整', icon: 'none' })
  loading.value = true
  try {
    const data = await login(phone.value, code.value)
    store.setTokens(data.access_token, data.refresh_token)
    uni.switchTab({ url: '/pages/home/index' })
  } catch (e: any) {
    uni.showToast({ title: e.msg || '登录失败', icon: 'none' })
  } finally { loading.value = false }
}
</script>
<template>
  <view class="login">
    <image class="logo" src="/static/logo.png" mode="widthFix" />
    <text class="brand">租租</text>
    <view class="field"><input v-model="phone" type="number" maxlength="11" placeholder="手机号" /></view>
    <view class="field">
      <input v-model="code" type="number" maxlength="6" placeholder="验证码" />
      <text class="sms" :class="{disabled: countdown>0}" @tap="sendSms">
        {{ countdown>0 ? countdown+'s' : '获取验证码' }}
      </text>
    </view>
    <button class="btn" :loading="loading" @tap="doLogin">登录 / 注册</button>
  </view>
</template>
<style scoped>
.login{padding:80rpx 60rpx;display:flex;flex-direction:column;align-items:center}
.logo{width:160rpx;height:160rpx;margin-bottom:20rpx}
.brand{font-size:48rpx;font-weight:bold;color:#2E7D32;margin-bottom:60rpx}
.field{width:100%;display:flex;align-items:center;border-bottom:1rpx solid #ddd;padding-bottom:16rpx;margin-bottom:30rpx}
.field input{flex:1;font-size:30rpx;height:60rpx}
.sms{font-size:26rpx;color:#2E7D32;padding:0 10rpx}
.sms.disabled{color:#999}
.btn{width:100%;background:#2E7D32;color:#fff;font-size:32rpx;border-radius:38rpx;height:80rpx;margin-top:20rpx}
</style>
