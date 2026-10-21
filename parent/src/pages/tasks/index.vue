<script setup>
import { computed, ref } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchTasks, withStudentScope } from "@/api/parent-task"

const tasks = ref([])
const loading = ref(false)
const loadError = ref("")
const activeTab = ref("进行中")

const tabs = ["进行中", "已打卡", "全部"]

// 图标颜色轮换
const iconColors = [
  { bg: "#DBEAFE", fg: "#3B82F6" },
  { bg: "#D1FAE5", fg: "#10B981" },
  { bg: "#FEF3C7", fg: "#F59E0B" },
  { bg: "#F3E8FF", fg: "#A855F7" },
  { bg: "#FCE7F3", fg: "#EC4899" },
  { bg: "#FEE2E2", fg: "#EF4444" },
]

const freqLabels = { daily: "每日", weekly: "每周", monthly: "每月", once: "一次性" }

const formattedTasks = computed(() => tasks.value.map((item, idx) => {
  const color = iconColors[idx % iconColors.length]
  const freqLabel = freqLabels[item.frequency] || ""
  const totalDays = item.frequency === "daily" ? 7 : item.frequency === "weekly" ? 4 : 1
  const progress = Math.min(item.checkin_count / Math.max(totalDays, 1), 1)
  return {
    ...item,
    iconBg: color.bg,
    iconColor: color.fg,
    desc: item.description || item.intro || "",
    progress,
    progressLabel: item.submitted_current_period
      ? "今日已打卡"
      : `${item.checkin_count}/${totalDays}次`,
    infoLabel: freqLabel ? `${freqLabel}任务` : "亲子任务",
    statusLabel: item.submitted_current_period ? "已提交" : "待完成",
  }
}))

const filteredTasks = computed(() => {
  if (activeTab.value === "已打卡") return formattedTasks.value.filter(t => t.checked_in_today)
  if (activeTab.value === "全部") return formattedTasks.value
  return formattedTasks.value.filter(t => t.status === 1 && !t.checked_in_today)
})

function goDetail(id) { uni.navigateTo({ url: `/pages/tasks/detail?id=${id}` }) }
function goCheckIn(id) { uni.navigateTo({ url: `/pages/tasks/detail?id=${id}` }) }
function goBadges() { uni.navigateTo({ url: "/pages/points/index" }) }

async function loadData() {
  // 已有数据时静默刷新：从详情页返回会走 onShow，整块列表闪成「加载中...」体验很差。
  if (tasks.value.length === 0) loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    // 当前孩子解析与 403 重试统一走 withStudentScope（与详情页同一条路径）。
    const scoped = await withStudentScope((sid) => fetchTasks({ student_id: sid }))
    if (!scoped) {
      tasks.value = []
      loadError.value = "未绑定孩子信息"
      return
    }
    tasks.value = scoped.result?.data?.list || []
  } catch (err) {
    // 不清空已有列表：清空会把「刷新失败」伪装成「没有任务」。
    loadError.value = err?.message || "加载任务失败"
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

// 详情页提交后返回列表不重跑 onLoad，卡片会停在「待完成」；用 onShow 保证返回即刷新。
onShow(loadData)
onPullDownRefresh(loadData)
</script>

<template>
  <view class="page">
    <scroll-view scroll-y class="content-scroll" :refresher-enabled="true" @refresherrefresh="loadData">
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
      <!-- 错误态与空态分开：列表为空时加载失败不能显示成「暂无任务」 -->
      <view v-else-if="loadError && formattedTasks.length === 0" class="state-wrap">
        <text class="state-text">{{ loadError }}</text>
        <view class="state-action" @click="loadData">重试</view>
      </view>
      <view v-else-if="formattedTasks.length === 0" class="state-wrap">
        <text class="state-text">暂无任务</text>
      </view>
      <view v-else class="task-list">
        <!-- 已有数据但本次刷新失败：保留旧列表并提示，避免看起来像最新结果 -->
        <view v-if="loadError" class="list-error" @click="loadData">
          <text>{{ loadError }}，点击重试</text>
        </view>
        <view v-for="item in filteredTasks" :key="item.id" class="task-card">
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
              <text>{{ item.statusLabel }}</text>
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
              <text>{{ item.submitted_current_period ? "查看记录" : "去提交" }}</text>
            </view>
            <view class="tca-detail" @click="goDetail(item.id)">
              <text>查看详情</text>
            </view>
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
  flex-direction: column;
  gap: 24rpx;
  align-items: center;
  justify-content: center;
  padding: 200rpx 0;
}
.state-text { font-size: $typography-font-size-md; color: $color-text-hint; }
.state-action {
  padding: 12rpx 40rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
}
.list-error {
  padding: 20rpx 24rpx;
  border-radius: $border-radius-md;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
}

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

.bottom-safe {
  height: calc(120rpx + env(safe-area-inset-bottom));
}
</style>
