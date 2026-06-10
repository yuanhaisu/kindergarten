<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchTeacherPickupOverview, fetchTeacherPickupStudents, manualTeacherPickup } from "@/api/teacher-pickup"

const overview = ref({})
const students = ref([])

const todayDate = computed(() => {
  const now = new Date()
  return `${now.getFullYear()}年${now.getMonth() + 1}月${now.getDate()}日`
})

async function loadPickup() {
  const [overviewRes, studentsRes] = await Promise.all([
    fetchTeacherPickupOverview(),
    fetchTeacherPickupStudents()
  ])
  overview.value = overviewRes.data || {}
  students.value = studentsRes.data?.list || []
  uni.stopPullDownRefresh()
}

async function record(student) {
  try {
    await manualTeacherPickup({
      student_id: student.id,
      type: 2,
      pickup_name: "老师登记",
      pickup_relation: "teacher"
    })
    uni.showToast({ title: "登记成功", icon: "success" })
    loadPickup()
  } catch (err) {
    uni.showToast({ title: err?.message || "登记失败", icon: "none" })
  }
}

function goQrScan() {
  uni.scanCode({ success: () => {} })
}

function goManualRegister() {
  uni.showToast({ title: "手动登记功能开发中", icon: "none" })
}

function goRecords() {
  uni.navigateTo({ url: "/pages/pickup/records" })
}

onLoad(loadPickup)
onPullDownRefresh(loadPickup)
</script>

<template>
  <view class="page">
    <!-- 今日接送统计卡片 -->
    <view class="stats-card">
      <view class="stats-head">
        <text class="stats-title">今日接送统计</text>
        <text class="stats-date">{{ todayDate }}</text>
      </view>
      <view class="stats-grid">
        <view class="stat-item">
          <text class="stat-num">{{ overview.arrived || 0 }}</text>
          <text class="stat-label">已入园</text>
        </view>
        <view class="stat-item">
          <text class="stat-num">{{ overview.picked_up || 0 }}</text>
          <text class="stat-label">已接走</text>
        </view>
        <view class="stat-item">
          <text class="stat-num">{{ overview.not_picked || overview.total - overview.picked_up || 0 }}</text>
          <text class="stat-label">未接走</text>
        </view>
        <view class="stat-item">
          <text class="stat-num">{{ overview.abnormal || 0 }}</text>
          <text class="stat-label">异常</text>
        </view>
      </view>
    </view>

    <!-- 功能操作区 -->
    <view class="action-card">
      <view class="action-grid">
        <view class="action-item" @click="goQrScan">
          <view class="action-icon-box is-green">
            <text class="action-icon-text">扫</text>
          </view>
          <text class="action-label">扫码接送</text>
        </view>
        <view class="action-item" @click="goManualRegister">
          <view class="action-icon-box is-blue">
            <text class="action-icon-text">记</text>
          </view>
          <text class="action-label">手动登记</text>
        </view>
        <view class="action-item" @click="goRecords">
          <view class="action-icon-box is-purple">
            <text class="action-icon-text">录</text>
          </view>
          <text class="action-label">接送记录</text>
        </view>
      </view>
      <view class="action-hint">
        <text class="hint-icon">提</text>
        <text class="hint-text">点击右上角扫码按钮，扫描家长出示的接送二维码即可完成接送登记，操作仅需3秒</text>
      </view>
    </view>

    <!-- 学生列表 -->
    <view class="student-section">
      <view class="student-head">
        <text class="student-title">未接走学生（{{ students.length }}人）</text>
        <view class="filter-btn">
          <text>筛选</text>
        </view>
      </view>

      <view v-if="students.length === 0" class="empty-state">
        <text>所有学生已全部接走</text>
      </view>
      <view v-else class="student-list">
        <view v-for="student in students" :key="student.id" class="student-card">
          <view class="student-info">
            <view class="student-avatar">
              <image
                v-if="student.avatar"
                :src="student.avatar"
                mode="aspectFill"
                class="avatar-img"
              />
              <text v-else class="avatar-placeholder">{{ student.name?.[0] || "生" }}</text>
            </view>
            <view class="student-text">
              <text class="student-name">{{ student.name }}</text>
              <text class="student-auth">
                授权接送人：{{ student.authorized_persons || "暂无" }}
              </text>
            </view>
          </view>
          <view class="register-btn" @click="record(student)">
            <text>登记</text>
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
$slate-50: #F8FAFC;
$slate-400: #94A3B8;
$slate-500: #64748B;
$slate-800: #1E293B;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 24rpx 28rpx;
  background: $slate-50;
}

// ===== 统计卡片 =====
.stats-card {
  padding: 48rpx 48rpx 40rpx;
  border-radius: 32rpx;
  background: linear-gradient(135deg, #F97316 0%, #EA580C 100%);
  margin-bottom: 24rpx;
}

.stats-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 32rpx;
}

.stats-title {
  font-size: 32rpx;
  font-weight: 700;
  color: #fff;
}

.stats-date {
  font-size: 28rpx;
  color: #FFEDD5;
}

.stats-grid {
  display: flex;
  justify-content: space-around;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8rpx;
}

.stat-num {
  font-size: 56rpx;
  font-weight: 800;
  color: #fff;
}

.stat-label {
  font-size: 26rpx;
  color: #FFEDD5;
}

// ===== 操作区 =====
.action-card {
  padding: 32rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.action-grid {
  display: flex;
  justify-content: space-around;
  margin-bottom: 32rpx;
}

.action-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18rpx;
}

.action-icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  border-radius: 32rpx;
}

.is-green  { background: #DCFCE7; }
.is-blue   { background: #DBEAFE; }
.is-purple { background: #F3E8FF; }

.action-icon-text {
  font-size: 46rpx;
  font-weight: 700;
}

.is-green  .action-icon-text { color: #16A34A; }
.is-blue   .action-icon-text { color: #2563EB; }
.is-purple .action-icon-text { color: #9333EA; }

.action-label {
  font-size: 26rpx;
  font-weight: 500;
  color: #334155;
}

.action-hint {
  display: flex;
  align-items: flex-start;
  gap: 24rpx;
  padding: 24rpx;
  border-radius: 24rpx;
  background: $slate-50;
}

.hint-icon {
  flex-shrink: 0;
  font-size: 28rpx;
  font-weight: 700;
  color: $blue-500;
}

.hint-text {
  font-size: 26rpx;
  color: $slate-500;
  line-height: 1.5;
}

// ===== 学生列表 =====
.student-section {
  margin-bottom: 24rpx;
}

.student-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.student-title {
  font-size: 32rpx;
  font-weight: 700;
  color: $slate-800;
}

.filter-btn {
  display: flex;
  align-items: center;
  gap: 4rpx;

  text {
    font-size: 28rpx;
    color: $blue-500;
  }
}

.empty-state {
  padding: 80rpx 0;
  text-align: center;

  text {
    font-size: 28rpx;
    color: $slate-400;
  }
}

.student-list {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.student-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 32rpx;
  border-radius: 24rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
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
  width: 96rpx;
  height: 96rpx;
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
  font-size: 36rpx;
  font-weight: 700;
  color: $blue-500;
}

.student-text {
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

.student-auth {
  font-size: 24rpx;
  color: $slate-500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.register-btn {
  flex-shrink: 0;
  padding: 12rpx 24rpx;
  border-radius: 16rpx;
  background: #EFF6FF;

  text {
    font-size: 26rpx;
    font-weight: 500;
    color: $blue-500;
  }
}

.bottom-safe {
  height: calc(60rpx + env(safe-area-inset-bottom));
}
</style>
