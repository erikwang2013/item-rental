<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { unifiedOrder } from '@/api/pay'
import { getOrder } from '@/api/order'
const loading = ref(false)
const status = ref('待支付')
const orderId = Number(new URLSearchParams(location.search).get('order_id')) || 1
onMounted(async () => { await poll() })
async function poll(){
  try { const o:any = await getOrder(orderId); status.value = ['待支付','待取货','租赁中','待归还','已归还','已取消','违约'][o.status]||o.status } catch{}
}
async function pay(){
  loading.value = true
  try {
    const p:any = await unifiedOrder(orderId)
    // mock 模式:后端返回 code_url，真实环境调 wx.requestPayment
    uni.showToast({title:'支付已发起(mock)'})
    let n=0
    const t=setInterval(async()=>{ n++; await poll(); if(status.value!=='待支付'||n>10) clearInterval(t) },1500)
  } catch(e:any){ uni.showToast({title:e.msg||'支付失败',icon:'none'}) }
  finally { loading.value=false }
}
</script>
<template>
  <view class="pay">
    <text class="st">{{status}}</text>
    <view class="tip" v-if="status==='待支付'">请完成支付以确认订单</view>
    <button class="btn" :loading="loading" :disabled="status!=='待支付'" @tap="pay">
      {{ status==='待支付' ? '去支付' : '已支付' }}
    </button>
  </view>
</template>
<style scoped>
.pay{padding:60rpx;display:flex;flex-direction:column;align-items:center}
.st{font-size:48rpx;font-weight:bold;color:#2E7D32;margin-bottom:20rpx}
.tip{font-size:26rpx;color:#999;margin-bottom:60rpx}
.btn{width:100%;background:#FB8C00;color:#fff;height:80rpx;font-size:32rpx;border-radius:40rpx}
.btn[disabled]{background:#ccc}
</style>
