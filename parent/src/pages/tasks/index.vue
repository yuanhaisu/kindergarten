<script setup>
import { computed, ref } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchTasks, withStudentScope } from "@/api/parent-task"
import { getCurrentStudentId } from "@/utils/currentStudent"

const tasks = ref([])
const loading = ref(false)
const loadError = ref("")
const activeTab = ref("进行中")

const tabs = ["进行中", "已打卡", "全部"]

const freqLabels = { daily: "每日", weekly: "每周", monthly: "每月", once: "一次性" }

// 当前孩子 id 一并带上：详情页要靠它把「这是哪个孩子的任务」传下去，
// 否则多孩家庭在详情里只能靠本地缓存猜。
const studentId = ref(getCurrentStudentId())

// 进度按「当前孩子本周期是否已提交」算，不是全班累计，也不是缩水的 7 天/4 次折算。
// checkin_count 是当前孩子在该任务下的累计提交次数，只作为副标题展示。
const formattedTasks = computed(() => tasks.value.map((item) => {
  const submitted = !!item.submitted_current_period
  return {
    ...item,
    desc: item.description || item.intro || "",
    progress: submitted ? 1 : 0,
    progressLabel: submitted ? "本周期已提交" : "本周期待提交",
    countLabel: `累计打卡 ${Number(item.checkin_count) || 0} 次`,
    infoLabel: freqLabels[item.frequency] ? `${freqLabels[item.frequency]}任务` : "亲子任务",
    // 列表接口没有「老师是否已回应」字段（契约 06-task.md 只给了提交口径），
    // 所以这里只区分待完成/已提交；是否已回应在详情页按 responses 判断。
    statusLabel: submitted ? "已提交" : "待完成",
    // 补充回应未读与消息未读是两套红点，不合并。
    hasUnread: !!item.has_unread_supplement
  }
}))

const pendingCount = computed(() => formattedTasks.value.filter(t => !t.submitted_current_period).length)
const unreadCount = computed(() => formattedTasks.value.filter(t => t.hasUnread).length)

const filteredTasks = computed(() => {
  if (activeTab.value === "已打卡") return formattedTasks.value.filter(t => t.submitted_current_period)
  if (activeTab.value === "全部") return formattedTasks.value
  return formattedTasks.value.filter(t => t.status === 1 && !t.submitted_current_period)
})

function taskUrl(id) {
  const sid = studentId.value
  return `/pages/tasks/detail?id=${id}${sid ? `&student_id=${sid}` : ""}`
}
// 任务页不是 tabBar 页（家长端 tabBar 只有 index/growth/pickup/profile），用 navigateTo。
function goDetail(id) { uni.navigateTo({ url: taskUrl(id) }) }
function goCheckIn(id) { uni.navigateTo({ url: taskUrl(id) }) }

async function loadData() {
  // 已有数据时静默刷新：从详情页返回会走 onShow，整块列表闪成「加载中...」体验很差。
  // 这个刷新也负责清红点：详情页里 markTaskRead 之后，返回列表重新取数即消失。
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
    studentId.value = scoped.sid
    tasks.value = scoped.result?.data?.list || []
  } catch (err) {
    // 不清空已有列表：清空会把「刷新失败」伪装成「没有任务」。
    loadError.value = err?.message || "加载任务失败"
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

function switchTab(tab) { activeTab.value = tab }

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
          @click="switchTab(tab)"
        >
          <text>{{ tab }}</text>
          <view v-if="tab === '进行中' && pendingCount" class="tab-count">
            <text class="tab-count-text">{{ pendingCount }}</text>
          </view>
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
      <view v-else-if="filteredTasks.length === 0" class="state-wrap">
        <text class="state-text">{{ activeTab === "进行中" ? "当前没有待完成的任务" : "暂无任务" }}</text>
      </view>
      <view v-else class="task-list">
        <!-- 已有数据但本次刷新失败：保留旧列表并提示，避免看起来像最新结果 -->
        <view v-if="loadError" class="list-error" @click="loadData">
          <text>{{ loadError }}，点击重试</text>
        </view>
        <view v-for="item in filteredTasks" :key="item.id" class="task-card">
          <!-- 任务头部 -->
          <view class="tc-head">
            <view class="tc-icon-box">
              <text class="tc-icon">📚</text>
              <view v-if="item.hasUnread" class="tc-dot" />
            </view>
            <view class="tc-head-mid">
              <text class="tc-title">{{ item.title }}</text>
              <text class="tc-desc">{{ item.desc }}</text>
            </view>
            <view class="tc-status-badge" :class="{ submitted: item.submitted_current_period }">
              <text class="tc-status-text">{{ item.statusLabel }}</text>
            </view>
          </view>

          <!-- 补充回应未读：与「已提交/待完成」并列说明原因，避免红点无声无息 -->
          <view v-if="item.hasUnread" class="tc-unread">
            <text>老师又给你留了一句悄悄话</text>
          </view>

          <!-- 本周期进度 -->
          <view class="tc-progress">
            <view class="tcp-bg">
              <view class="tcp-fill" :style="{ width: (item.progress * 100) + '%' }" />
            </view>
            <text class="tcp-label">{{ item.progressLabel }}</text>
          </view>

          <!-- 频次与累计次数 -->
          <view class="tc-info">
            <text class="tci-text">{{ item.infoLabel }} · {{ item.countLabel }}</text>
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
  padding: $component-page-padding-y $component-page-padding-x 0;
  height: 0;
  box-sizing: border-box;
  width: 100%;
}

// ===== 分类标签 =====
.tab-bar {
  display: flex;
  gap: $spacing-lg;
  margin-bottom: $spacing-lg;
}

.tab-item {
  flex: 1;
  position: relative;
  padding: $spacing-lg 0;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  text-align: center;
  text {
    font-size: $typography-font-size-md;
    font-weight: $typography-font-weight-medium;
    color: $color-text-muted;
  }
}

.tab-item.active {
  background: $color-brand-primary-bg;
  text {
    color: $color-brand-primary;
    font-weight: $typography-font-weight-semibold;
  }
}

.tab-count {
  position: absolute;
  top: 8rpx;
  right: 12rpx;
  min-width: 32rpx;
  height: 32rpx;
  padding: 0 $spacing-xs;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tab-count-text {
  color: $color-text-on-brand;
  font-size: 20rpx;
  line-height: 1;
}

// ===== 状态 =====
.state-wrap {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  align-items: center;
  justify-content: center;
  padding: 200rpx 0;
}
.state-text { font-size: $typography-font-size-md; color: $color-text-hint; }
.state-action {
  padding: $spacing-sm 40rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
}
.list-error {
  padding: 20rpx $spacing-lg;
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
  background: $color-bg-card;
  border-radius: $component-card-radius;
  padding: $component-card-padding;
  box-shadow: $shadow-sm;
}

.tc-head {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
}

.tc-icon-box {
  position: relative;
  width: 96rpx;
  height: 96rpx;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tc-icon {
  font-size: 44rpx;
}

.tc-dot {
  position: absolute;
  top: 4rpx;
  right: 4rpx;
  width: 16rpx;
  height: 16rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
}

.tc-head-mid {
  flex: 1;
  overflow: hidden;
}

.tc-title {
  display: block;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-bold;
  color: $color-text-primary;
}

.tc-desc {
  display: block;
  font-size: $typography-font-size-base;
  color: $color-text-hint;
  margin-top: $spacing-xs;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.tc-status-badge {
  padding: 6rpx 20rpx;
  border-radius: $border-radius-md;
  background: $color-brand-primary-bg;
  flex-shrink: 0;
  .tc-status-text {
    font-size: $typography-font-size-xs;
    font-weight: $typography-font-weight-semibold;
    color: $color-brand-primary;
  }
}

.tc-status-badge.submitted {
  background: $color-semantic-success-bg;
  .tc-status-text { color: $color-semantic-success; }
}

.tc-unread {
  margin-top: $spacing-md;
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius-md;
  background: $color-semantic-warning-bg;
  text {
    color: $color-semantic-warning;
    font-size: $typography-font-size-xs;
  }
}

// ===== 进度条 =====
.tc-progress {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-lg;
}

.tcp-bg {
  flex: 1;
  height: 16rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  overflow: hidden;
}

.tcp-fill {
  height: 100%;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.tcp-label {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
  flex-shrink: 0;
}

// ===== 频次说明 =====
.tc-info {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  margin-top: $spacing-md;
}

.tci-text {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

// ===== 操作按钮 =====
.tc-actions {
  display: flex;
  gap: $spacing-lg;
  margin-top: $spacing-lg;
}

.tca-checkin {
  flex: 1;
  padding: $spacing-lg 0;
  border-radius: $border-radius-xl;
  background: $color-brand-primary;
  text-align: center;
  text {
    font-size: $typography-font-size-md;
    font-weight: $typography-font-weight-semibold;
    color: $color-text-on-brand;
  }
}

.tca-detail {
  flex: 1;
  padding: $spacing-lg 0;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
  text-align: center;
  text {
    font-size: $typography-font-size-md;
    font-weight: $typography-font-weight-medium;
    color: $color-brand-primary;
  }
}

.bottom-safe {
  height: calc(120rpx + env(safe-area-inset-bottom));
}
</style>
