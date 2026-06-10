<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchPickupRecords } from "@/api/parent-pickup"

const records = ref([])
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchPickupRecords()
    records.value = res.data?.list || []
  } catch (err) {
    uni.showToast({ title: err?.message || "加载失败", icon: "none" })
  } finally { loading.value = false }
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <view class="header">
      <text class="page-title">接送记录</text>
      <button class="link-btn" @click="uni.navigateTo({ url: '/pages/pickup/qrcode' })">接送二维码</button>
    </view>
    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="records.length === 0" class="state">暂无接送记录</view>
    <view v-else class="list">
      <view v-for="item in records" :key="item.id" class="card">
        <view class="card-head">
          <text class="type" :class="item.type === 2 ? 'out' : 'in'">{{ item.type_label }}</text>
          <text class="date">{{ item.date }} {{ item.recorded_at?.slice(11, 16) }}</text>
        </view>
        <text class="person">{{ item.pickup_name }} ({{ item.pickup_relation }})</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx; background: $parent-bg; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24rpx; }
.page-title { color: $parent-text; font-size: 36rpx; font-weight: 800; }
.link-btn { padding: 14rpx 28rpx; border-radius: 999rpx; background: $parent-blue; color: #fff; font-size: 24rpx; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $parent-muted; font-size: 28rpx; }
.list { display: flex; flex-direction: column; gap: 18rpx; }
.card { padding: 26rpx; border-radius: 24rpx; background: $parent-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.card-head { display: flex; justify-content: space-between; margin-bottom: 8rpx; }
.type { padding: 6rpx 16rpx; border-radius: 999rpx; font-size: 22rpx; }
.type.in { background: #d4f0d0; color: #3a7a33; }
.type.out { background: #cfe1ff; color: #2f5a8c; }
.date { color: $parent-muted; font-size: 22rpx; }
.person { display: block; color: $parent-text; font-size: 26rpx; }
</style>
