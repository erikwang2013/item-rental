<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getProfile, updateProfile } from '@/api/user'
import { useUserStore } from '@/stores/user'
const form = ref({ nickname:'', avatar:'' })
const store = useUserStore()
onMounted(async () => { try { const p:any=await getProfile(); form.value={nickname:p.nickname||'',avatar:p.avatar||''}; store.setProfile(p) } catch{} })
async function save(){ try { await updateProfile(form.value); uni.showToast({title:'已保存'}) } catch(e:any){ uni.showToast({title:e.msg||'保存失败',icon:'none'}) } }
</script>
<template>
  <view class="pf">
    <view class="row"><text class="k">昵称</text><input v-model="form.nickname" placeholder="昵称" /></view>
    <view class="row"><text class="k">头像</text><input v-model="form.avatar" placeholder="头像 URL" /></view>
    <button class="btn" @tap="save">保存</button>
  </view>
</template>
<style scoped>
.pf{padding:30rpx}
.row{display:flex;align-items:center;background:#fff;margin-bottom:20rpx;padding:24rpx;border-radius:12rpx}
.k{font-size:28rpx;color:#666;width:140rpx}
.row input{flex:1;font-size:28rpx;height:60rpx}
.btn{width:100%;background:#2E7D32;color:#fff;height:80rpx;font-size:32rpx;border-radius:40rpx;margin-top:30rpx}
</style>
