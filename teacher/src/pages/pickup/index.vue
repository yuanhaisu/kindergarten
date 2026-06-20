<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchTeacherPickupOverview, fetchTeacherPickupStudents, manualTeacherPickup, scanTeacherPickup } from "@/api/teacher-pickup"

const overview = ref({})
const students = ref([])
const filterStatus = ref("not_picked_up") // not_picked_up | all | arrived | picked_up

const todayDate = computed(() => {
  const now = new Date()
  return `${now.getFullYear()}年${now.getMonth() + 1}月${now.getDate()}日`
})

async function loadPickup() {
  const [overviewRes, studentsRes] = await Promise.all([
    fetchTeacherPickupOverview(),
    fetchTeacherPickupStudents({ status: filterStatus.value }),
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
  uni.scanCode({
    success: async (res) => {
      try {
        const qrCode = res.result || res.scanCode
        if (!qrCode) {
          uni.showToast({ title: "二维码识别失败", icon: "none" })
          return
        }
        const result = await scanTeacherPickup({ qr_code: qrCode })
        const data = result.data
        uni.showToast({
          title: `${data.student?.name || ""} ${data.type_label || "接送"}登记成功`,
          icon: "success",
        })
        loadPickup()
      } catch (err) {
        uni.showToast({ title: err?.message || "扫码登记失败", icon: "none" })
      }
    },
    fail: () => {
      uni.showToast({ title: "扫码已取消", icon: "none" })
    },
  })
}

function goManualRegister() {
  filterStatus.value = "all"
  loadPickup()
}

function goRecords() {
  uni.navigateTo({ url: "/pages/pickup/records" })
}

const filterOptions = [
  { key: "not_picked_up", label: "未接走" },
  { key: "all", label: "全部" },
  { key: "arrived", label: "已入园" },
  { key: "picked_up", label: "已接走" },
]
let filterIdx = 0
function toggleFilter() {
  filterIdx = (filterIdx + 1) % filterOptions.length
  filterStatus.value = filterOptions[filterIdx].key
  loadPickup()
}

const studentSectionTitle = computed(() => {
  const label = filterOptions[filterIdx].label
  return `${label}学生（${students.value.length}人）`
})

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
        <text class="student-title">{{ studentSectionTitle }}</text>
        <view class="filter-btn" @click="toggleFilter">
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
.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: $spacing-lg 28rpx;
  background: $color-bg-card;
}

// ===== 统计卡片 =====
.stats-card {
  padding: 48rpx 48rpx 40rpx;
  border-radius: $border-radius-xl;
  background: linear-gradient(135deg, #F97316 0%, #EA580C 100%);
  margin-bottom: $spacing-lg;
}

.stats-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: $spacing-xl;
}

.stats-title {
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: #fff;
}

.stats-date {
  font-size: $typography-font-size-md;
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
  gap: $spacing-xs;
}

.stat-num {
  font-size: 56rpx;
  font-weight: 800;
  color: #fff;
}

.stat-label {
  font-size: $typography-font-size-base;
  color: #FFEDD5;
}

// ===== 操作区 =====
.action-card {
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: $spacing-lg;
}

.action-grid {
  display: flex;
  justify-content: space-around;
  margin-bottom: $spacing-xl;
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
  border-radius: $border-radius-xl;
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
  font-size: $typography-font-size-base;
  font-weight: 500;
  color: #334155;
}

.action-hint {
  display: flex;
  align-items: flex-start;
  gap: $spacing-lg;
  padding: $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
}

.hint-icon {
  flex-shrink: 0;
  font-size: $typography-font-size-md;
  font-weight: 700;
  color: $color-semantic-info;
}

.hint-text {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
  line-height: 1.5;
}

// ===== 学生列表 =====
.student-section {
  margin-bottom: $spacing-lg;
}

.student-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: $spacing-lg;
}

.student-title {
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

.filter-btn {
  display: flex;
  align-items: center;
  gap: 4rpx;

  text {
    font-size: $typography-font-size-md;
    color: $color-semantic-info;
  }
}

.empty-state {
  padding: 80rpx 0;
  text-align: center;

  text {
    font-size: $typography-font-size-md;
    color: $color-text-hint;
  }
}

.student-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.student-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
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
  width: 96rpx;
  height: 96rpx;
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
  background: #DBEAFE;
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: $color-semantic-info;
}

.student-text {
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

.student-auth {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.register-btn {
  flex-shrink: 0;
  padding: $spacing-sm $spacing-lg;
  border-radius: $border-radius-lg;
  background: #EFF6FF;

  text {
    font-size: $typography-font-size-base;
    font-weight: 500;
    color: $color-semantic-info;
  }
}

.bottom-safe {
  height: calc(60rpx + env(safe-area-inset-bottom));
}
</style>
