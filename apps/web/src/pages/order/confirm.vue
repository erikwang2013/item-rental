<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getItem } from '@/api/item'
import { createOrder } from '@/api/order'
const item = ref<any>(null)
const start = ref('')
const end = ref('')
const days = ref(0)
const total = ref(0)
onMounted(async () => {
  const id = Number(new URLSearchParams(location.search).get('id'))
  try { item.value = await getItem(id) } catch {}
  const t = new Date(); t.setDate(t.getDate()+1)
  const t2 = new Date(); t2.setDate(t2.getDate()+3)
  start.value = fmt(t); end.value = fmt(t2); calc()
})
function fmt(d:Date){ return d.getFullYear()+'-'+String(d.getMonth()+1).padStart(2,'0')+'-'+String(d.getDate()).padStart(2,'0') }
function calc(){
  const s=new Date(start.value), e=new Date(end.value)
  days.value = Math.max(1, Math.ceil((e.getTime()-s.getTime())/86400000))
  total.value = days.value * (item.value?.daily_price||0)
}
async function submit(){
  if(!start.value||!end.value) return uni.showToast({title:'选择日期',icon:'none'})
  const id = Number(new URLSearchParams(location.search).get('id'))
  try { const o:any = await createOrder({item_id:id,start_date:start.value,end_date:end.value}) }
    catch(e:any){ uni.showToast({title:e.msg||'下单失败',icon:'none'}); return }
  uni.navigateTo({url:'/pages/order/pay?order_id=1'})
}
</script>
<template>
  <view class="cf">
    <view class="dt" @tap="()=>uni.showActionSheet({itemCount:1,success:()=>{}})">
      <text>租期 {{start}} ~ {{end}}</text><text class="d">{{days}}天</text>
    </view>
    <view class="cost">
      <view><text>租金 ¥{{item?.daily_price||0}} × {{days}}天</text><text>¥{{total}}</text></view>
      <view><text>押金</text><text>¥{{item?.deposit||0}}</text></view>
    </view>
    <button class="btn" @tap="submit">确认下单</button>
  </view>
</template>
<style scoped>
.cf{padding:30rpx}
.dt{display:flex;justify-content:space-between;background:#f5f5f5;padding:24rpx;border-radius:12rpx;font-size:28rpx;margin-bottom:20rpx}
.d{color:#2E7D32}
.cost{background:#fff;padding:24rpx;border-radius:12rpx;margin-bottom:30rpx}
.cost view{display:flex;justify-content:space-between;padding:16rpx 0;font-size:28rpx;border-bottom:1rpx solid #f0f0f0}
.cost view:last-child{border:none}
.btn{width:100%;background:#2E7D32;color:#fff;height:80rpx;font-size:32rpx;border-radius:40rpx}
</style>
