<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchTeacherPickupRecords } from "@/api/teacher-pickup"

const records = ref([])
const loading = ref(false)
const selectedDate = ref(new Date())

const dateLabel = computed(() => {
  const d = selectedDate.value
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
})

const summary = computed(() => {
  const total = records.value.length
  const picked = records.value.filter(r => r.type === "放学接送").length
  const arrived = records.value.filter(r => r.type === "入园签到").length
  return { total, picked, arrived }
})

function prevDay() {
  const d = new Date(selectedDate.value)
  d.setDate(d.getDate() - 1)
  selectedDate.value = d
  loadRecords()
}
function nextDay() {
  const d = new Date(selectedDate.value)
  d.setDate(d.getDate() + 1)
  selectedDate.value = d
  loadRecords()
}
function goBack() { uni.navigateBack() }
function openSearch() { uni.showToast({ title: "搜索功能开发中", icon: "none" }) }
function openFilter() { uni.showToast({ title: "筛选功能开发中", icon: "none" }) }

async function loadRecords() {
  loading.value = true
  try {
    await ensureLogin()
    const dateStr = selectedDate.value.toISOString().slice(0, 10)
    const res = await fetchTeacherPickupRecords({ date: dateStr })
    records.value = (res.data?.list || []).map((item) => ({
      id: item.id,
      time: item.recorded_at?.slice(11, 16) || "",
      type: item.type === 1 ? "入园签到" : "放学接送",
      // status 只对「接走」有意义：入园签到不是 pending 等待，而是已完成的到场动作
      status: item.type === 2 ? "picked" : "arrived",
      student_name: item.student?.name || "",
      student_class: item.student?.class_name || "",
      pickup_by: item.pickup_name || "",
      pickup_method: item.method_label || "",
      pickup_method_color: item.method === 1 ? "blue" : "purple",
      avatar: item.student?.avatar || "",
      remark: item.remark || "",
      is_override: !!item.is_authorized_override,
      is_abnormal: !!item.is_abnormal,
    }))
  } catch (err) {
    uni.showToast({ title: err?.message || "加载失败", icon: "none" })
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

onLoad(loadRecords)
onPullDownRefresh(loadRecords)
</script>

<template>
  <view class="page">
    <!-- 日期筛选栏 -->
    <view class="filter-card">
      <view class="date-selector">
        <view class="date-arrow left" @click="prevDay">
          <text class="arrow-text">◂</text>
        </view>
        <view class="date-display">
          <text class="date-icon">日</text>
          <text class="date-text">{{ dateLabel }}</text>
          <text class="date-chevron">▾</text>
        </view>
        <view class="date-arrow right" @click="nextDay">
          <text class="arrow-text">▸</text>
        </view>
      </view>
      <view class="filter-btn" @click="openFilter">
        <text>筛选</text>
      </view>
    </view>

    <!-- 统计摘要条 -->
    <view class="summary-card">
      <view class="summary-item">
        <view class="summary-icon-box is-blue">
          <text class="summary-icon">总</text>
        </view>
        <view class="summary-text">
          <text class="summary-label">今日接送</text>
          <view class="summary-value">
            <text class="summary-num">{{ summary.total }}</text>
            <text class="summary-unit">人</text>
          </view>
        </view>
      </view>
      <view class="summary-divider" />
      <view class="summary-item">
        <view class="summary-icon-box is-green">
          <text class="summary-icon">接</text>
        </view>
        <view class="summary-text">
          <text class="summary-label">已接走</text>
          <view class="summary-value">
            <text class="summary-num is-green-num">{{ summary.picked }}</text>
            <text class="summary-unit">人</text>
          </view>
        </view>
      </view>
      <view class="summary-divider" />
      <view class="summary-item">
        <view class="summary-icon-box is-amber">
          <text class="summary-icon">入</text>
        </view>
        <view class="summary-text">
          <text class="summary-label">入园签到</text>
          <view class="summary-value">
            <text class="summary-num is-amber-num">{{ summary.arrived }}</text>
            <text class="summary-unit">人</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 记录列表 -->
    <view v-if="loading" class="state-wrap">
      <text class="state-text">加载中...</text>
    </view>
    <view v-else-if="records.length === 0" class="state-wrap">
      <text class="state-text">暂无接送记录</text>
    </view>
    <view v-else class="record-list">
      <view v-for="item in records" :key="item.id" class="record-card">
        <!-- 卡片头部 -->
        <view class="card-head">
          <view class="time-info">
            <view class="time-dot" :class="item.status === 'picked' ? 'is-green' : 'is-blue'" />
            <text class="time-text">{{ item.time }}</text>
            <text class="time-type">· {{ item.type }}</text>
          </view>
          <view class="status-badge" :class="item.status === 'picked' ? 'is-green' : 'is-blue'">
            <text>{{ item.status === "picked" ? "已接走" : "已入园" }}</text>
          </view>
        </view>

        <!-- 卡片主体 -->
        <view class="card-body">
          <view class="student-info">
            <view class="student-avatar">
              <image
                v-if="item.avatar"
                :src="item.avatar"
                mode="aspectFill"
                class="avatar-img"
              />
              <text v-else class="avatar-placeholder">{{ item.student_name?.[0] || "生" }}</text>
            </view>
            <view class="student-detail">
              <text class="student-name">{{ item.student_name }}</text>
              <text class="student-meta">{{ item.student_class }}</text>
            </view>
          </view>

          <view class="pickup-info">
            <view class="pickup-row">
              <text class="pickup-icon">接</text>
              <text class="pickup-text">{{ item.pickup_by }}</text>
              <text v-if="item.is_override" class="override-badge">例外</text>
            </view>
            <view class="pickup-row">
              <text class="pickup-icon method-icon" :class="`is-${item.pickup_method_color}`">方</text>
              <text class="pickup-method" :class="`is-${item.pickup_method_color}`">{{ item.pickup_method }}</text>
            </view>
            <view v-if="item.remark" class="remark-row">
              <text class="remark-label">理由：</text>
              <text class="remark-text">{{ item.remark }}</text>
            </view>
            <view v-if="item.is_abnormal" class="abnormal-row">
              <text class="abnormal-tag">异常</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <!-- 底部留白 -->
    <view class="bottom-safe" />
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: $spacing-lg $component-page-padding-x;
  background: $color-bg-page;
}

// ===== 日期筛选栏 =====
.filter-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-lg $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  margin-bottom: $spacing-lg;
}

.date-selector {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.date-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.arrow-text {
  font-size: $typography-font-size-xl;
  color: $color-text-muted;
}

.date-display {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.date-icon {
  font-size: $typography-font-size-xl;
  color: $color-semantic-info;
}

.date-text {
  font-size: $typography-font-size-md;
  font-weight: 500;
  color: $color-text-primary;
}

.date-chevron {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.filter-btn {
  display: flex;
  align-items: center;
  gap: 4rpx;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-semantic-info-bg;

  text {
    font-size: $typography-font-size-base;
    color: $color-semantic-info;
  }
}

// ===== 统计摘要条 =====
.summary-card {
  display: flex;
  align-items: center;
  padding: $spacing-lg $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  margin-bottom: $spacing-lg;
}

.summary-item {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  flex: 1;
}

.summary-icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border-radius: $border-radius-xl;
}

.summary-icon-box.is-blue  { background: $color-semantic-info-bg; }
.summary-icon-box.is-green { background: $color-semantic-success-bg; }
.summary-icon-box.is-amber { background: $color-semantic-warning-bg; }

.summary-icon {
  font-size: $typography-font-size-xl;
  font-weight: 700;

  .is-blue &  { color: $color-semantic-info; }
  .is-green & { color: $color-semantic-success; }
  .is-amber & { color: $color-semantic-warning; }
}

.summary-text {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.summary-label {
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.summary-value {
  display: flex;
  align-items: baseline;
  gap: 4rpx;
}

.summary-num {
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

.summary-num.is-green-num { color: $color-semantic-success; }
.summary-num.is-amber-num { color: $color-semantic-warning; }

.summary-unit {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.summary-divider {
  width: 4rpx;
  height: 64rpx;
  border-radius: 2rpx;
  background: #E2E8F0;
}

// ===== 状态 =====
.state-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 50vh;
}

.state-text {
  font-size: $typography-font-size-md;
  color: $color-text-hint;
}

// ===== 记录列表 =====
.record-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.record-card {
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: $spacing-lg;
}

.time-info {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.time-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: $border-radius-full;
}

.time-dot.is-green { background: $color-semantic-success; }
.time-dot.is-blue { background: $color-semantic-info; }
.time-dot.is-amber { background: $color-semantic-warning; }

.time-text {
  font-size: $typography-font-size-base;
  font-weight: 500;
  color: $color-text-muted;
}

.time-type {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.status-badge {
  padding: $spacing-xs $component-badge-padding-x;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-sm;
  font-weight: 500;
}

.status-badge.is-green {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.status-badge.is-blue {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.status-badge.is-amber {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}

.card-body {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.student-info {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  flex: 1;
  min-width: 0;
}

.student-avatar {
  flex-shrink: 0;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  overflow: hidden;
}

.avatar-img {
  width: 100%;
  height: 100%;
}

.avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: $color-semantic-info-bg;
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: $color-semantic-info;
}

.student-detail {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
}

.student-name {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
}

.student-meta {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.pickup-info {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4rpx;
  flex-shrink: 0;
}

.pickup-row {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.pickup-icon {
  font-size: $typography-font-size-sm;
  font-weight: 700;
  color: $color-text-muted;
}

.method-icon.is-blue   { color: $color-semantic-info; }
.method-icon.is-purple { color: #9333EA; }
.method-icon.is-gray   { color: $color-text-hint; }

.pickup-text {
  font-size: $typography-font-size-base;
  font-weight: 500;
  color: $color-text-body;
}

.pickup-method {
  font-size: $typography-font-size-xs;
}

.pickup-method.is-blue   { color: $color-semantic-info; }
.pickup-method.is-purple { color: #9333EA; }
.pickup-method.is-gray   { color: $color-text-hint; }

.bottom-safe {
  height: calc(60rpx + env(safe-area-inset-bottom));
}

// ===== 理由 / 例外标记 =====
.override-badge {
  padding: 2rpx 12rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
  font-size: $typography-font-size-xs;
  font-weight: 600;
  margin-left: $spacing-xs;
}

.remark-row {
  display: flex;
  align-items: flex-start;
  margin-top: 4rpx;
}

.remark-label {
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
  flex-shrink: 0;
}

.remark-text {
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
  flex: 1;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.abnormal-row {
  margin-top: 4rpx;
}

.abnormal-tag {
  padding: 2rpx 12rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
  color: #fff;
  font-size: $typography-font-size-xs;
  font-weight: 600;
}
</style>
