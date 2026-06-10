<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchLeaves } from "@/api/parent-leave"

const leaves = ref([])
const loading = ref(false)
const activeStatus = ref("")

const statusTabs = [
  { key: "", label: "全部" },
  { key: "pending", label: "待审批" },
  { key: "approved", label: "已同意" },
  { key: "rejected", label: "已拒绝" }
]

const stats = computed(() => ({
  total: leaves.value.length,
  pending: leaves.value.filter(l => l.status === 0).length,
  approved: leaves.value.filter(l => l.status === 1).length
}))

const filteredLeaves = computed(() => {
  if (!activeStatus.value) return leaves.value
  return leaves.value.filter(l => {
    const map = { pending: 0, approved: 1, rejected: 2 }
    return l.status === map[activeStatus.value]
  })
})

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchLeaves({ status: activeStatus.value || undefined })
    leaves.value = res.data?.list || []
  } catch (err) {
    uni.showToast({ title: err?.message || "加载失败", icon: "none" })
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

function switchStatus(key) {
  if (activeStatus.value === key) return
  activeStatus.value = key
  loadData()
}

function goCreate() { uni.navigateTo({ url: "/pages/leave/create" }) }
function goDetail(id) { uni.navigateTo({ url: `/pages/leave/detail?id=${id}` }) }

onLoad(loadData)
onPullDownRefresh(loadData)
</script>

<template>
  <view class="page">
    <!-- 统计卡片 -->
    <view class="stats-card">
      <view class="stat-item">
        <text class="stat-num">{{ stats.total }}</text>
        <text class="stat-label">全部请假</text>
      </view>
      <view class="stat-divider" />
      <view class="stat-item">
        <text class="stat-num highlight">{{ stats.pending }}</text>
        <text class="stat-label">待审批</text>
      </view>
      <view class="stat-divider" />
      <view class="stat-item">
        <text class="stat-num">{{ stats.approved }}</text>
        <text class="stat-label">已同意</text>
      </view>
      <view class="stat-divider" />
      <view class="stat-item">
        <button class="create-btn" @click="goCreate">+ 请假</button>
      </view>
    </view>

    <!-- 状态筛选 -->
    <view class="tab-row">
      <view v-for="t in statusTabs" :key="t.key"
        class="tab-item" :class="{ active: activeStatus === t.key }"
        @click="switchStatus(t.key)">
        <text class="tab-text">{{ t.label }}</text>
      </view>
    </view>

    <!-- 列表 -->
    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="filteredLeaves.length === 0" class="state">暂无请假记录</view>
    <view v-else class="list">
      <view v-for="item in filteredLeaves" :key="item.id" class="card" @click="goDetail(item.id)">
        <view class="card-head">
          <view class="student-row">
            <view class="student-avatar">
              <image v-if="item.student?.avatar" :src="item.student.avatar" class="avatar-img" mode="aspectFill" />
              <text v-else class="avatar-placeholder">?</text>
            </view>
            <view class="spacer-w12" />
            <view>
              <text class="student-name">{{ item.student?.name || "" }}</text>
              <text class="leave-days">{{ item.days }}天</text>
            </view>
          </view>
          <view class="status-badge" :class="'status-' + item.status">
            <text>{{ item.status_label }}</text>
          </view>
        </view>
        <view class="spacer-h12" />
        <text class="reason">{{ item.reason }}</text>
        <view class="spacer-h8" />
        <text class="date-range">{{ item.start_date }} ~ {{ item.end_date }}</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx calc(150rpx + env(safe-area-inset-bottom)); background: $parent-bg; }
.spacer-w12 { width: 24rpx; flex-shrink: 0; }
.spacer-h8 { height: 16rpx; }
.spacer-h12 { height: 24rpx; }

.stats-card {
  display: flex; align-items: center;
  background: $parent-card; border-radius: $parent-radius-md; padding: 32rpx 16rpx;
  box-shadow: $parent-shadow; margin-bottom: 24rpx;
}
.stat-item { flex: 1; text-align: center; }
.stat-num { display: block; font-size: 40rpx; font-weight: 700; color: $parent-text; }
.stat-num.highlight { color: $parent-primary; }
.stat-label { display: block; margin-top: 8rpx; font-size: 24rpx; color: $parent-hint; }
.stat-divider { width: 2rpx; height: 60rpx; background: #E2E8F0; }
.create-btn { padding: 14rpx 24rpx; border-radius: 9999rpx; background: $parent-primary; color: #fff; font-size: 24rpx; line-height: 1.4; }

.tab-row { display: flex; gap: 16rpx; margin-bottom: 24rpx; }
.tab-item { padding: 16rpx 32rpx; border-radius: 9999rpx; background: $parent-card; }
.tab-item.active { background: $parent-primary; }
.tab-text { font-size: 26rpx; color: $parent-body; }
.tab-item.active .tab-text { color: #fff; font-weight: 600; }

.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $parent-hint; font-size: 28rpx; }
.list { display: flex; flex-direction: column; gap: 16rpx; }

.card { padding: 32rpx; border-radius: $parent-radius; background: $parent-card; box-shadow: $parent-shadow; }
.card-head { display: flex; justify-content: space-between; align-items: flex-start; }
.student-row { display: flex; align-items: center; }
.student-avatar { width: 80rpx; height: 80rpx; border-radius: 9999rpx; overflow: hidden; background: #DBEAFE; display: flex; align-items: center; justify-content: center; }
.avatar-img { width: 100%; height: 100%; }
.avatar-placeholder { color: #2563EB; font-size: 32rpx; font-weight: 700; }
.student-name { display: block; font-size: 28rpx; font-weight: 600; color: $parent-text; }
.leave-days { display: block; font-size: 24rpx; color: $parent-muted; margin-top: 4rpx; }

.status-badge { padding: 8rpx 20rpx; border-radius: 9999rpx; background: #FFEDD5; font-size: 22rpx; color: #92400E; }
.status-badge.status-1 { background: #DCFCE7; color: #166534; }
.status-badge.status-2 { background: #FEE2E2; color: #991B1B; }
.status-badge.status-3 { background: #F1F5F9; color: #64748B; }

.reason { display: block; font-size: 26rpx; color: $parent-text; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.date-range { display: block; font-size: 24rpx; color: $parent-muted; }
</style>
