<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getOrder, pickup, returnRequest, returnConfirm, breach, cancel, refund } from '@/api/order'
const o = ref<any>(null)
const orderId = Number(new URLSearchParams(location.search).get('order_id')) || 1
const stateName = ['待支付','待取货','租赁中','待归还','已归还','已取消','违约']
async function load(){ try { o.value = await getOrder(orderId) } catch{} }
onMounted(load)
async function act(fn:any){ try { await fn(orderId); uni.showToast({title:'操作成功'}); load() } catch(e:any){ uni.showToast({title:e.msg||'操作失败',icon:'none'}) } }
</script>
<template>
  <view class="od" v-if="o">
    <view class="st">{{stateName[o.status]||o.status}}</view>
    <view class="info">
      <view><text>订单号</text><text>{{o.order_no}}</text></view>
      <view><text>物品</text><text>{{o.item_title||'-'}}</text></view>
      <view><text>租期</text><text>{{o.start_date}} ~ {{o.end_date}}</text></view>
      <view><text>租金</text><text>¥{{o.rent_amount}}</text></view>
      <view><text>押金</text><text>¥{{o.deposit}}</text></view>
    </view>
    <view class="acts">
      <button v-if="o.status===0" @tap="()=>uni.navigateTo({url:'/pages/order/pay?order_id='+o.id})">去支付</button>
      <button v-if="o.status===0" class="sec" @tap="()=>act(cancel)">取消订单</button>
      <button v-if="o.status===1" @tap="()=>act(pickup)">确认取货</button>
      <button v-if="o.status===2" @tap="()=>act(returnRequest)">申请归还</button>
      <button v-if="o.status===3" @tap="()=>act(returnConfirm)">确认归还</button>
      <button v-if="o.status===3" class="sec" @tap="()=>act(breach)">判定违约</button>
      <button v-if="o.status===5||o.status===6" class="sec" @tap="()=>act(refund)">申请退款</button>
    </view>
  </view>
</template>
<style scoped>
.od{padding:30rpx}
.st{text-align:center;font-size:40rpx;font-weight:bold;color:#2E7D32;margin:20rpx 0 40rpx}
.info{background:#fff;border-radius:12rpx;overflow:hidden;margin-bottom:30rpx}
.info view{display:flex;padding:24rpx;border-bottom:1rpx solid #f5f5f5;font-size:26rpx}
.info text:first-child{color:#999;width:180rpx}
.acts{display:flex;flex-wrap:wrap;gap:16rpx}
.acts button{flex:1;min-width:45%;background:#2E7D32;color:#fff;height:70rpx;font-size:26rpx;border-radius:35rpx}
.acts button.sec{background:#fff;color:#E53935;border:1rpx solid #E53935}
</style>
