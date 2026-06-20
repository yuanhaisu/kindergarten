<script setup>
import { ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchTasks } from "@/api/parent-task"

const tasks = ref([])
const loading = ref(false)
const activeTab = ref("进行中")

const tabs = ["进行中", "已打卡", "全部"]

// Mock points data
const points = ref({ total: 520, earned: 1280, monthEarned: 85, rank: 12, streak: 7 })

// Mock badge wall
const badges = ref([
  { name: "全勤达人", iconColor: "#F59E0B", bgColor: "#FEF3C7", locked: false },
  { name: "阅读之星", iconColor: "#3B82F6", bgColor: "#DBEAFE", locked: false },
  { name: "运动健将", iconColor: "#10B981", bgColor: "#D1FAE5", locked: false },
  { name: "小小画家", iconColor: "#EC4899", bgColor: "#FCE7F3", locked: false },
  { name: "劳动能手", iconColor: "#8B5CF6", bgColor: "#EDE9FE", locked: false },
  { name: "爱心天使", iconColor: "#EF4444", bgColor: "#FEE2E2", locked: false },
  { name: "???", iconColor: "#94A3B8", bgColor: "#F1F5F9", locked: true },
  { name: "???", iconColor: "#94A3B8", bgColor: "#F1F5F9", locked: true },
])

function goDetail(id) { uni.navigateTo({ url: `/pages/tasks/detail?id=${id}` }) }
function goCheckIn(id) { uni.navigateTo({ url: `/pages/tasks/detail?id=${id}` }) }
function goRule() { uni.showToast({ title: "积分规则", icon: "none" }) }
function goExchange() { uni.showToast({ title: "兑换奖励", icon: "none" }) }
function goBadges() { uni.navigateTo({ url: "/pages/points/index" }) }

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchTasks()
    const list = res.data?.list || []
    tasks.value = list
  } catch (err) {
    tasks.value = []
    uni.showToast({ title: "加载任务失败", icon: "none" })
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

onLoad(loadData)
onPullDownRefresh(loadData)
</script>

<template>
  <view class="page">
    <scroll-view scroll-y class="content-scroll" :refresher-enabled="true" @refresherrefresh="loadData">
      <!-- 积分卡片 -->
      <view class="points-card">
        <view class="pc-top">
          <view class="pc-left">
            <text class="pc-label">我的积分</text>
            <view class="pc-num-row">
              <text class="pc-star">&#xea8a;</text>
              <text class="pc-num">{{ points.total }}</text>
            </view>
          </view>
          <view class="pc-exchange" @click="goExchange">
            <text>兑换奖励</text>
          </view>
        </view>
        <view class="pc-stats">
          <view class="pcs-item">
            <text class="pcs-num">{{ points.earned }}</text>
            <text class="pcs-label">累计获得</text>
          </view>
          <view class="pcs-item">
            <text class="pcs-num">{{ points.monthEarned }}</text>
            <text class="pcs-label">本月获得</text>
          </view>
          <view class="pcs-item">
            <text class="pcs-num">{{ points.rank }}</text>
            <text class="pcs-label">排名</text>
          </view>
          <view class="pcs-item">
            <text class="pcs-num">{{ points.streak }}天</text>
            <text class="pcs-label">连续打卡</text>
          </view>
        </view>
      </view>

      <!-- 任务分类栏 -->
      <view class="tab-bar">
        <view
          v-for="tab in tabs"
          :key="tab"
          class="tab-item"
          :class="{ active: activeTab === tab }"
          @click="activeTab = tab"
        >
          <text>{{ tab }}</text>
        </view>
      </view>

      <!-- 任务列表 -->
      <view v-if="loading" class="state-wrap">
        <text class="state-text">加载中...</text>
      </view>
      <view v-else-if="tasks.length === 0" class="state-wrap">
        <text class="state-text">暂无任务</text>
      </view>
      <view v-else class="task-list">
        <view v-for="item in tasks" :key="item.id" class="task-card">
          <!-- 任务头部 -->
          <view class="tc-head">
            <view class="tc-icon-box" :style="{ backgroundColor: item.iconBg }">
              <text class="tc-icon" :style="{ color: item.iconColor }">&#xf1e8;</text>
            </view>
            <view class="tc-head-mid">
              <text class="tc-title">{{ item.title }}</text>
              <text class="tc-desc">{{ item.desc }}</text>
            </view>
            <view class="tc-points-badge">
              <text>+{{ item.points }}</text>
            </view>
          </view>

          <!-- 进度条 -->
          <view class="tc-progress">
            <view class="tcp-bg">
              <view class="tcp-fill" :style="{ width: (item.progress * 100) + '%' }" />
            </view>
            <text class="tcp-label">{{ item.progressLabel }}</text>
          </view>

          <!-- 积分说明 -->
          <view class="tc-info">
            <text class="tci-icon">&#xea8a;</text>
            <text class="tci-text">{{ item.infoLabel }}</text>
          </view>

          <!-- 操作按钮 -->
          <view class="tc-actions">
            <view class="tca-checkin" @click="goCheckIn(item.id)">
              <text>去打卡</text>
            </view>
            <view class="tca-detail" @click="goDetail(item.id)">
              <text>查看详情</text>
            </view>
          </view>
        </view>
      </view>

      <!-- 徽章墙 -->
      <view class="badge-section">
        <view class="bs-head">
          <text class="bs-title">徽章墙</text>
          <text class="bs-more" @click="goBadges">查看全部</text>
        </view>
        <view class="bs-grid">
          <view
            v-for="(b, i) in badges"
            :key="i"
            class="bs-item"
            :class="{ locked: b.locked }"
          >
            <view class="bs-icon-box" :style="{ backgroundColor: b.bgColor }">
              <text class="bs-icon" :style="{ color: b.iconColor }">&#xea8a;</text>
            </view>
            <text class="bs-name">{{ b.name }}</text>
          </view>
        </view>
      </view>

      <view class="bottom-safe" />
    </scroll-view>

  </view>
</template>

<style scoped lang="scss">
$orange-100: #FFEDD5;
$slate-200: #E2E8F0;
$slate-600: #475569;
$purple-500: #A855F7;
$purple-50: #F3E8FF;
$purple-100: #EDE9FE;

.page {
  min-height: 100vh;
  background: $color-bg-page;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow-x: hidden;
}

.content-scroll {
  flex: 1;
  padding: $spacing-lg 28rpx 0;
  height: 0;
  box-sizing: border-box;
  width: 100%;
}

// ===== 积分卡片 =====
.points-card {
  background: linear-gradient(135deg, $color-brand-primary-light, $color-brand-primary);
  border-radius: $border-radius-xl;
  padding: 48rpx;
  margin-bottom: 24rpx;
}

.pc-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.pc-label {
  font-size: $typography-font-size-md;
  color: $orange-100;
}

.pc-num-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: 8rpx;
}

.pc-star {
  font-size: 72rpx;
  color: #fff;
}

.pc-num {
  font-size: 88rpx;
  font-weight: 800;
  color: #fff;
}

.pc-exchange {
  padding: 14rpx 28rpx;
  border-radius: $border-radius-lg;
  background: rgba(255,255,255,0.2);
  text {
    font-size: $typography-font-size-md;
    font-weight: 500;
    color: #fff;
  }
}

.pc-stats {
  display: flex;
  justify-content: space-between;
  margin-top: 32rpx;
}

.pcs-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
}

.pcs-num {
  font-size: $typography-font-size-3xl;
  font-weight: 800;
  color: #fff;
}

.pcs-label {
  font-size: $typography-font-size-sm;
  color: $orange-100;
}

// ===== 分类标签 =====
.tab-bar {
  display: flex;
  gap: $spacing-lg;
  margin-bottom: 24rpx;
}

.tab-item {
  flex: 1;
  padding: $spacing-lg 0;
  border-radius: 20rpx;
  background: #fff;
  text-align: center;
  text {
    font-size: $typography-font-size-md;
    font-weight: 500;
    color: $color-text-muted;
  }
}

.tab-item.active {
  background: $purple-50;
  text {
    color: $purple-500;
    font-weight: 600;
  }
}

// ===== 状态 =====
.state-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 200rpx 0;
}
.state-text { font-size: $typography-font-size-md; color: $color-text-hint; }

// ===== 任务卡片 =====
.task-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.task-card {
  background: #fff;
  border-radius: $border-radius-xl;
  padding: $spacing-xl;
  box-shadow: 0 2rpx 8rpx rgba(0,0,0,0.05);
}

.tc-head {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
}

.tc-icon-box {
  width: 96rpx;
  height: 96rpx;
  border-radius: $border-radius-xl;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tc-icon {
  font-size: 52rpx;
}

.tc-head-mid {
  flex: 1;
  overflow: hidden;
}

.tc-title {
  display: block;
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

.tc-desc {
  display: block;
  font-size: $typography-font-size-base;
  color: $color-text-hint;
  margin-top: 8rpx;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.tc-points-badge {
  padding: 6rpx 20rpx;
  border-radius: $border-radius-md;
  background: $purple-50;
  flex-shrink: 0;
  text {
    font-size: $typography-font-size-2xl;
    font-weight: 800;
    color: $purple-500;
  }
}

// ===== 进度条 =====
.tc-progress {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: 24rpx;
}

.tcp-bg {
  flex: 1;
  height: 16rpx;
  border-radius: 9999rpx;
  background: $color-bg-card;
  overflow: hidden;
}

.tcp-fill {
  height: 100%;
  border-radius: 9999rpx;
  background: $purple-500;
}

.tcp-label {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
  flex-shrink: 0;
}

// ===== 积分说明 =====
.tc-info {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  margin-top: 16rpx;
}

.tci-icon {
  font-size: $typography-font-size-md;
  color: $color-text-hint;
}

.tci-text {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

// ===== 操作按钮 =====
.tc-actions {
  display: flex;
  gap: $spacing-lg;
  margin-top: 24rpx;
}

.tca-checkin {
  flex: 1;
  padding: $spacing-lg 0;
  border-radius: 20rpx;
  background: $purple-500;
  text-align: center;
  text {
    font-size: $typography-font-size-md;
    font-weight: 600;
    color: #fff;
  }
}

.tca-detail {
  flex: 1;
  padding: $spacing-lg 0;
  border-radius: 20rpx;
  background: $purple-50;
  text-align: center;
  text {
    font-size: $typography-font-size-md;
    font-weight: 500;
    color: $purple-500;
  }
}

// ===== 徽章墙 =====
.badge-section {
  margin-top: 48rpx;
}

.bs-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.bs-title {
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  color: $color-text-primary;
}

.bs-more {
  font-size: $typography-font-size-md;
  color: $purple-500;
}

.bs-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 20rpx;
}

.bs-item {
  width: calc(25% - 15rpx);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
}

.bs-icon-box {
  width: 128rpx;
  height: 128rpx;
  border-radius: $border-radius-xl;
  display: flex;
  align-items: center;
  justify-content: center;
}

.bs-icon {
  font-size: 72rpx;
}

.bs-name {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  text-align: center;
}

.bs-item.locked {
  .bs-icon-box { opacity: 0.5; }
  .bs-name { color: $color-text-hint; }
}

.bottom-safe {
  height: calc(120rpx + env(safe-area-inset-bottom));
}
</style>
