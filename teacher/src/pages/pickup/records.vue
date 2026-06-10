<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"

// Placeholder: replace with real API when available
// import { fetchTeacherPickupRecords } from "@/api/teacher-pickup"

const records = ref([])
const loading = ref(false)
const selectedDate = ref(new Date())

const dateLabel = computed(() => {
  const d = selectedDate.value
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
})

const summary = computed(() => {
  const total = records.value.length
  const picked = records.value.filter(r => r.status === "picked").length
  return { total, picked, pending: total - picked }
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
    // const res = await fetchTeacherPickupRecords({ date: dateStr })
    // records.value = res.data?.list || []
    uni.stopPullDownRefresh()
  } catch (err) {
    uni.showToast({ title: err?.message || "加载失败", icon: "none" })
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

const mockRecords = [
  { id: 1, time: "16:42", type: "放学接送", status: "picked", student_name: "王小明", student_class: "中一班", student_no: "20240301", pickup_by: "爸爸", pickup_method: "扫码接送", pickup_method_color: "blue", avatar: "" },
  { id: 2, time: "16:35", type: "放学接送", status: "picked", student_name: "李小花", student_class: "中二班", student_no: "20240315", pickup_by: "妈妈", pickup_method: "扫码接送", pickup_method_color: "blue", avatar: "" },
  { id: 3, time: "16:28", type: "放学接送", status: "picked", student_name: "张小强", student_class: "大一班", student_no: "20240208", pickup_by: "外婆", pickup_method: "手动登记", pickup_method_color: "purple", avatar: "" },
  { id: 4, time: "08:15", type: "入园签到", status: "pending", student_name: "陈雨欣", student_class: "中一班", student_no: "20240322", pickup_by: "妈妈", pickup_method: "刷卡入园", pickup_method_color: "gray", avatar: "" },
  { id: 5, time: "08:10", type: "入园签到", status: "picked", student_name: "刘宇轩", student_class: "大二班", student_no: "20240116", pickup_by: "爷爷", pickup_method: "刷卡入园", pickup_method_color: "gray", avatar: "" },
]

onLoad(() => {
  records.value = mockRecords
  // loadRecords()
})
onPullDownRefresh(() => {
  records.value = mockRecords
  // loadRecords()
})
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
          <text class="summary-icon">待</text>
        </view>
        <view class="summary-text">
          <text class="summary-label">待接走</text>
          <view class="summary-value">
            <text class="summary-num is-amber-num">{{ summary.pending }}</text>
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
            <view class="time-dot" :class="item.status === 'picked' ? 'is-green' : 'is-amber'" />
            <text class="time-text">{{ item.time }}</text>
            <text class="time-type">· {{ item.type }}</text>
          </view>
          <view class="status-badge" :class="item.status === 'picked' ? 'is-green' : 'is-amber'">
            <text>{{ item.status === "picked" ? "已接走" : "待接走" }}</text>
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
              <text class="student-meta">{{ item.student_class }} · 学号{{ item.student_no }}</text>
            </view>
          </view>

          <view class="pickup-info">
            <view class="pickup-row">
              <text class="pickup-icon">接</text>
              <text class="pickup-text">{{ item.pickup_by }}</text>
            </view>
            <view class="pickup-row">
              <text class="pickup-icon method-icon" :class="`is-${item.pickup_method_color}`">方</text>
              <text class="pickup-method" :class="`is-${item.pickup_method_color}`">{{ item.pickup_method }}</text>
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
$blue-500: #3B82F6;
$blue-50: #EFF6FF;
$slate-50: #F8FAFC;
$slate-200: #E2E8F0;
$slate-400: #94A3B8;
$slate-500: #64748B;
$slate-700: #334155;
$slate-800: #1E293B;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 24rpx 28rpx;
  background: $slate-50;
}

// ===== 日期筛选栏 =====
.filter-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24rpx 32rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.date-selector {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.date-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  border-radius: 999rpx;
  background: #F1F5F9;
}

.arrow-text {
  font-size: 32rpx;
  color: $slate-500;
}

.date-display {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.date-icon {
  font-size: 32rpx;
  color: $blue-500;
}

.date-text {
  font-size: 28rpx;
  font-weight: 500;
  color: $slate-800;
}

.date-chevron {
  font-size: 24rpx;
  color: $slate-400;
}

.filter-btn {
  display: flex;
  align-items: center;
  gap: 4rpx;
  padding: 12rpx 24rpx;
  border-radius: 16rpx;
  background: $blue-50;

  text {
    font-size: 26rpx;
    color: $blue-500;
  }
}

// ===== 统计摘要条 =====
.summary-card {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 16rpx;
  flex: 1;
}

.summary-icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border-radius: 24rpx;
}

.summary-icon-box.is-blue  { background: $blue-50; }
.summary-icon-box.is-green { background: #DCFCE7; }
.summary-icon-box.is-amber { background: #FEF3C7; }

.summary-icon {
  font-size: 32rpx;
  font-weight: 700;

  .is-blue &  { color: $blue-500; }
  .is-green & { color: #16A34A; }
  .is-amber & { color: #D97706; }
}

.summary-text {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.summary-label {
  font-size: 22rpx;
  color: $slate-400;
}

.summary-value {
  display: flex;
  align-items: baseline;
  gap: 4rpx;
}

.summary-num {
  font-size: 32rpx;
  font-weight: 800;
  color: $slate-800;
}

.summary-num.is-green-num { color: #16A34A; }
.summary-num.is-amber-num { color: #D97706; }

.summary-unit {
  font-size: 24rpx;
  color: $slate-500;
}

.summary-divider {
  width: 4rpx;
  height: 64rpx;
  border-radius: 2rpx;
  background: $slate-200;
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
  font-size: 28rpx;
  color: $slate-400;
}

// ===== 记录列表 =====
.record-list {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.record-card {
  padding: 32rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.time-info {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.time-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 999rpx;
}

.time-dot.is-green { background: #16A34A; }
.time-dot.is-amber { background: #F59E0B; }

.time-text {
  font-size: 26rpx;
  font-weight: 500;
  color: $slate-500;
}

.time-type {
  font-size: 24rpx;
  color: $slate-400;
}

.status-badge {
  padding: 8rpx 20rpx;
  border-radius: 16rpx;
  font-size: 24rpx;
  font-weight: 500;
}

.status-badge.is-green {
  background: #DCFCE7;
  color: #16A34A;
}

.status-badge.is-amber {
  background: #FEF3C7;
  color: #D97706;
}

.card-body {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.student-info {
  display: flex;
  align-items: center;
  gap: 24rpx;
  flex: 1;
  min-width: 0;
}

.student-avatar {
  flex-shrink: 0;
  width: 88rpx;
  height: 88rpx;
  border-radius: 999rpx;
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
  background: #DBEAFE;
  font-size: 34rpx;
  font-weight: 700;
  color: $blue-500;
}

.student-detail {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
}

.student-name {
  font-size: 30rpx;
  font-weight: 600;
  color: $slate-800;
}

.student-meta {
  font-size: 24rpx;
  color: $slate-400;
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
  gap: 8rpx;
}

.pickup-icon {
  font-size: 24rpx;
  font-weight: 700;
  color: $slate-500;
}

.method-icon.is-blue   { color: $blue-500; }
.method-icon.is-purple { color: #9333EA; }
.method-icon.is-gray   { color: $slate-400; }

.pickup-text {
  font-size: 26rpx;
  font-weight: 500;
  color: $slate-700;
}

.pickup-method {
  font-size: 22rpx;
}

.pickup-method.is-blue   { color: $blue-500; }
.pickup-method.is-purple { color: #9333EA; }
.pickup-method.is-gray   { color: $slate-400; }

.bottom-safe {
  height: calc(60rpx + env(safe-area-inset-bottom));
}
</style>
