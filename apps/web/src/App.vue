<script setup lang="ts">
import { useUserStore } from '@/stores/user'
onLaunch(() => { useUserStore().init() })
</script>
<style>
page { font-size:28rpx; color:#333; }
</style>
