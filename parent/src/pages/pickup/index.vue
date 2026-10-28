<script setup>
import { ref } from "vue"
import { onShow } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId } from "@/utils/currentStudent"
import { syncTabBar } from "@/utils/tabbar"
import { fetchPickupRecords } from "@/api/parent-pickup"

const records = ref([])
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    // ParentPickupRecords 走 parentCtx，按 query 上的 student_id 取当前孩子
    const res = await fetchPickupRecords({ student_id: getCurrentStudentId() || undefined })
    records.value = res.data?.list || []
  } catch (err) {
    uni.showToast({ title: err?.message || "加载失败", icon: "none" })
  } finally { loading.value = false }
}

// tabBar 页必须用 onShow：switchTab 切回来不重跑 onLoad。
// 同时把原生底栏高亮校正到「接送」（index 2）——组件内算不准当前 tab，见 utils/tabbar.js。
onShow(() => {
  syncTabBar(2)
  loadData()
})
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
/* 底部留白 = 原生 custom-tab-bar 约 126rpx + 安全区（见 custom-tab-bar/index.wxss） */
.page { min-height: 100vh; padding: $spacing-lg 28rpx calc(140rpx + env(safe-area-inset-bottom)); background: $color-bg-page; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24rpx; }
.page-title { color: $color-text-primary; font-size: $typography-font-size-2xl; font-weight: 800; }
.link-btn { padding: 14rpx 28rpx; border-radius: 999rpx; background: $color-semantic-info; color: #fff; font-size: $typography-font-size-sm; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }
.list { display: flex; flex-direction: column; gap: 18rpx; }
.card { padding: 26rpx; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.card-head { display: flex; justify-content: space-between; margin-bottom: 8rpx; }
.type { padding: 6rpx 16rpx; border-radius: 999rpx; font-size: $typography-font-size-xs; }
.type.in { background: #d4f0d0; color: #3a7a33; }
.type.out { background: #cfe1ff; color: #2f5a8c; }
.date { color: $color-text-muted; font-size: $typography-font-size-xs; }
.person { display: block; color: $color-text-primary; font-size: $typography-font-size-base; }
</style>
