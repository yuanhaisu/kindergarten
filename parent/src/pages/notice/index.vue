<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchNotices } from "@/api/parent-notice"

const loading = ref(false)
const notices = ref([])
const total = ref(0)
const activeTab = ref("")

const tabs = [
  { key: "", label: "全部" },
  { key: "task", label: "任务" },
  { key: "activity", label: "活动" },
  { key: "other", label: "通知" }
]

const statCounts = computed(() => ({
  total: total.value,
  unread: notices.value.filter(n => !n.is_read).length,
  today: notices.value.filter(n => n.published_at?.slice(0, 10) === new Date().toISOString().slice(0, 10)).length
}))

function getCategoryMapping(key) {
  // "other" tab = homework + bring + health + other category
  if (activeTab.value === "other") return { category: "" }
  return { category: activeTab.value }
}

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const params = getCategoryMapping()
    if (activeTab.value === "other") {
      // front-end filter: homework/bring/health/other
      const res = await fetchNotices(params)
      const all = res.data?.list || []
      notices.value = all.filter(n =>
        ["homework", "bring", "health", "other"].includes(n.category)
      )
      total.value = res.data?.total || 0
    } else {
      const res = await fetchNotices(params)
      notices.value = res.data?.list || []
      total.value = res.data?.total || 0
    }
  } catch (err) {
    uni.showToast({ title: err?.message || "加载失败", icon: "none" })
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

function switchTab(key) {
  if (activeTab.value === key) return
  activeTab.value = key
  loadData()
}

function goDetail(id) {
  uni.navigateTo({ url: `/pages/notice/detail?id=${id}` })
}

onLoad(loadData)
onPullDownRefresh(loadData)
</script>

<template>
  <view class="page">
    <!-- 统计卡片 -->
    <view class="stats-card">
      <view class="stat-item">
        <text class="stat-num">{{ statCounts.total }}</text>
        <text class="stat-label">全部通知</text>
      </view>
      <view class="stat-divider" />
      <view class="stat-item">
        <text class="stat-num highlight">{{ statCounts.unread }}</text>
        <text class="stat-label">未读</text>
      </view>
      <view class="stat-divider" />
      <view class="stat-item">
        <text class="stat-num">{{ statCounts.today }}</text>
        <text class="stat-label">今日新增</text>
      </view>
    </view>

    <!-- 分类 Tab -->
    <view class="tab-row">
      <view
        v-for="tab in tabs" :key="tab.key"
        class="tab-item"
        :class="{ active: activeTab === tab.key }"
        @click="switchTab(tab.key)"
      >
        <text class="tab-text">{{ tab.label }}</text>
      </view>
    </view>

    <!-- 列表 -->
    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="notices.length === 0" class="state">暂无通知</view>
    <view v-else class="list">
      <view
        v-for="item in notices" :key="item.id"
        class="card"
        :class="{ unread: !item.is_read }"
        @click="goDetail(item.id)"
      >
        <view class="card-icon-box">
          <text class="card-icon"></text>
        </view>
        <view class="spacer-w12" />
        <view class="card-body">
          <view class="card-title-row">
            <text class="card-title">{{ item.title }}</text>
            <view v-if="!item.is_read" class="unread-dot" />
          </view>
          <view class="spacer-h4" />
          <text class="card-preview">{{ item.content_preview }}</text>
          <view class="spacer-h8" />
          <view class="card-foot">
            <view class="foot-left">
              <text class="foot-icon"></text>
              <view class="spacer-w4" />
              <text class="foot-text">{{ item.creator?.nickname || "" }}</text>
            </view>
            <text class="foot-text">{{ item.published_at?.slice(0, 10) }}</text>
          </view>
        </view>
        <view class="spacer-w8" />
        <text class="arrow"></text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  padding: 24rpx 28rpx calc(150rpx + env(safe-area-inset-bottom));
  background: $color-bg-page;
}

.spacer-w4 { width: 8rpx; flex-shrink: 0; }
.spacer-w8 { width: 16rpx; flex-shrink: 0; }
.spacer-w12 { width: 24rpx; flex-shrink: 0; }
.spacer-h4 { height: 8rpx; }
.spacer-h8 { height: 16rpx; }

// 统计卡片
.stats-card {
  display: flex;
  align-items: center;
  background: $color-bg-card;
  border-radius: $border-radius-lg;
  padding: 32rpx 16rpx;
  box-shadow: $shadow-sm;
  margin-bottom: 24rpx;
}

.stat-item {
  flex: 1;
  text-align: center;
}

.stat-num {
  display: block;
  font-size: 40rpx;
  font-weight: 700;
  color: $color-text-primary;
}

.stat-num.highlight {
  color: $color-brand-primary;
}

.stat-label {
  display: block;
  margin-top: 8rpx;
  font-size: 24rpx;
  color: $color-text-hint;
}

.stat-divider {
  width: 2rpx;
  height: 60rpx;
  background: #E2E8F0;
}

// 分类 Tab
.tab-row {
  display: flex;
  gap: 16rpx;
  margin-bottom: 24rpx;
}

.tab-item {
  padding: 16rpx 32rpx;
  border-radius: 9999rpx;
  background: $color-bg-card;
}

.tab-item.active {
  background: $color-brand-primary;
}

.tab-text {
  font-size: 26rpx;
  color: $color-text-body;
}

.tab-item.active .tab-text {
  color: #FFFFFF;
  font-weight: 600;
}

// 列表
.state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  color: $color-text-hint;
  font-size: 28rpx;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.card {
  display: flex;
  align-items: flex-start;
  padding: 32rpx;
  border-radius: $border-radius-md;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.card.unread {
  border-left: 6rpx solid $color-brand-primary;
}

.card-icon-box {
  width: 80rpx;
  height: 80rpx;
  border-radius: 9999rpx;
  background: #DBEAFE;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.card-icon {
  font-family: remixicon;
  font-size: 36rpx;
  color: #2563EB;
}

.card-body {
  flex: 1;
  overflow: hidden;
}

.card-title-row {
  display: flex;
  align-items: center;
}

.card-title {
  font-size: 30rpx;
  font-weight: 600;
  color: $color-text-primary;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.unread-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 9999rpx;
  background: #EF4444;
  flex-shrink: 0;
  margin-left: 8rpx;
}

.card-preview {
  font-size: 26rpx;
  color: $color-text-muted;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.foot-left {
  display: flex;
  align-items: center;
}

.foot-icon {
  font-family: remixicon;
  font-size: 24rpx;
  color: $color-text-hint;
}

.foot-text {
  font-size: 24rpx;
  color: $color-text-hint;
}

.arrow {
  font-family: remixicon;
  font-size: 36rpx;
  color: $color-text-hint;
  align-self: center;
}
</style>
