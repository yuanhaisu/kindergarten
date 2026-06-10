<script setup>
import { ref, computed } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchTeacherClassInfo, fetchTeacherClassStudents } from "@/api/teacher-class"
import BottomNav from "@/components/BottomNav.vue"

const info = ref({})
const students = ref([])
const loading = ref(false)
const searchKeyword = ref("")
const activeFilter = ref("全部")

const filters = ["全部", "出勤", "请假", "缺勤", "迟到"]

const statusClassMap = {
  "出勤": "status-present",
  "请假": "status-leave",
  "缺勤": "status-absent",
  "迟到": "status-late",
}

// Mock today snapshot
const todaySnapshot = ref({
  present: 26, leave: 2, late: 1, not_picked: 5,
})

// Mock notices
const recentNotices = ref([
  { id: 1, title: "明早亲子运动会通知", time: "今天 09:30", readRate: 85 },
  { id: 2, title: "请带水彩笔和画纸", time: "昨天 16:00", readRate: 92 },
])

// Mock: assign a random status to each student for visual demo
const statusOptions = ["出勤", "出勤", "出勤", "出勤", "请假", "缺勤", "迟到"]

const filteredStudents = computed(() => {
  let list = students.value
  if (searchKeyword.value) {
    const kw = searchKeyword.value.toLowerCase()
    list = list.filter(s =>
      (s.name || "").toLowerCase().includes(kw) ||
      (s.student_no || "").toLowerCase().includes(kw)
    )
  }
  if (activeFilter.value !== "全部") {
    list = list.filter(s => (s._status || "出勤") === activeFilter.value)
  }
  return list
})

function onSearchInput(e) {
  searchKeyword.value = e.detail.value
}

function switchFilter(f) {
  activeFilter.value = f
}

function openClassSearch() {
  // scroll to search bar or focus
}

function openNotices() {
  uni.showToast({ title: "通知铃铛", icon: "none" })
}

function goNotice(notice) {
  uni.showToast({ title: `通知: ${notice.title}`, icon: "none" })
}

function goStudent(student) {
  uni.showToast({ title: `查看: ${student.name}`, icon: "none" })
}

function viewAllStudents() {
  // scroll to roster
}

function quickAction(type) {
  const routes = {
    notice: "/pages/notice/create",
    growth: "/pages/growth/create",
    pickup: "/pages/pickup/index",
  }
  uni.navigateTo({ url: routes[type] })
}

function formatDate() {
  const d = new Date()
  return `${d.getMonth() + 1}月${d.getDate()}日`
}

async function loadClass() {
  loading.value = true
  try {
    const [infoRes, studentsRes] = await Promise.all([
      fetchTeacherClassInfo(),
      fetchTeacherClassStudents(),
    ])
    info.value = infoRes.data || {}
    const list = studentsRes.data?.list || []
    students.value = list.map((s, i) => ({
      ...s,
      _status: statusOptions[i % statusOptions.length],
    }))
  } catch (err) {
    // keep fallback data
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

const className = computed(() => info.value.name || "大一班")
const classGrade = computed(() => info.value.grade || "大班")
const studentCount = computed(() => info.value.student_count || students.value.length || 30)
const teacherName = computed(() => info.value.teacher_name || "张老师")
const roomInfo = computed(() => info.value.room || "101室")
const academicInfo = computed(() => "2025-2026学年 · 春季学期")

onLoad(loadClass)
onPullDownRefresh(loadClass)
</script>

<template>
  <view class="page">
    <!-- 顶部导航栏 -->
    <view class="header">
      <view class="header-top">
        <view class="class-title-row">
          <view class="class-icon-box">
            <text class="class-icon">&#xf1ae;</text>
          </view>
          <view class="class-name-col">
            <view class="class-name-row">
              <text class="class-name">{{ className }}</text>
              <text class="edit-icon">&#xea6e;</text>
            </view>
            <text class="class-subtitle">{{ academicInfo }}</text>
          </view>
        </view>
        <view class="header-actions">
          <view class="hdr-btn" @click="openClassSearch">
            <text class="hdr-btn-icon">&#xea76;</text>
          </view>
          <view class="hdr-btn" @click="openNotices">
            <text class="hdr-btn-icon">&#xef94;</text>
            <view class="hdr-badge" />
          </view>
        </view>
      </view>
    </view>

    <scroll-view scroll-y class="content-scroll" :refresher-enabled="true" @refresherrefresh="loadClass">
      <!-- 班级信息卡 -->
      <view class="info-card">
        <view class="info-head">
          <view class="info-school-icon">
            <text class="info-school-icon-text">&#xf036;</text>
          </view>
          <view class="info-head-text">
            <text class="info-head-title">阳光幼儿园 · {{ className }}</text>
            <text class="info-head-sub">{{ academicInfo }}</text>
          </view>
        </view>
        <view class="info-divider" />
        <view class="info-row">
          <view class="info-item">
            <view class="info-item-avatar">
              <text class="info-item-avatar-text">师</text>
            </view>
            <view class="info-item-text">
              <text class="info-item-label">班主任</text>
              <text class="info-item-value">{{ teacherName }}</text>
            </view>
          </view>
          <view class="info-v-divider" />
          <view class="info-item">
            <view class="info-item-icon-box">
              <text class="info-item-icon-text">&#xf1ae;</text>
            </view>
            <view class="info-item-text">
              <text class="info-item-label">学生总数</text>
              <text class="info-item-value">{{ studentCount }}人</text>
            </view>
          </view>
          <view class="info-v-divider" />
          <view class="info-item">
            <view class="info-item-icon-box">
              <text class="info-item-icon-text">&#xee16;</text>
            </view>
            <view class="info-item-text">
              <text class="info-item-label">教室</text>
              <text class="info-item-value">{{ roomInfo }}</text>
            </view>
          </view>
        </view>
      </view>

      <!-- 今日快照 -->
      <view class="section">
        <view class="section-head">
          <text class="section-title">今日快照</text>
          <text class="section-date">{{ formatDate() }}</text>
        </view>
        <view class="snapshot-row">
          <view class="snap-item">
            <text class="snap-num green">{{ todaySnapshot.present }}</text>
            <text class="snap-unit">人</text>
            <text class="snap-label">已出勤</text>
          </view>
          <view class="snap-item">
            <text class="snap-num yellow">{{ todaySnapshot.leave }}</text>
            <text class="snap-unit">人</text>
            <text class="snap-label">请假中</text>
          </view>
          <view class="snap-item">
            <text class="snap-num orange">{{ todaySnapshot.late }}</text>
            <text class="snap-unit">人</text>
            <text class="snap-label">迟到</text>
          </view>
          <view class="snap-item">
            <text class="snap-num blue">{{ todaySnapshot.not_picked }}</text>
            <text class="snap-unit">人</text>
            <text class="snap-label">未接走</text>
          </view>
        </view>
      </view>

      <!-- 班级动态 -->
      <view class="section">
        <view class="section-head">
          <text class="section-title">班级动态</text>
        </view>
        <view class="quick-row">
          <view class="quick-btn" @click="quickAction('notice')">
            <text class="quick-icon">&#xef94;</text>
            <text class="quick-label">发布通知</text>
          </view>
          <view class="quick-btn" @click="quickAction('growth')">
            <text class="quick-icon">&#xec29;</text>
            <text class="quick-label">记成长</text>
          </view>
          <view class="quick-btn" @click="quickAction('pickup')">
            <text class="quick-icon">&#xf1ae;</text>
            <text class="quick-label">接送</text>
          </view>
        </view>
        <view class="sub-title-row">
          <text class="sub-title">最新通知</text>
        </view>
        <view v-for="notice in recentNotices" :key="notice.id" class="notice-card" @click="goNotice(notice)">
          <view class="notice-icon-box">
            <text class="notice-icon">&#xef94;</text>
          </view>
          <view class="notice-body">
            <text class="notice-title">{{ notice.title }}</text>
            <text class="notice-time">{{ notice.time }}</text>
          </view>
          <view class="notice-rate">
            <view class="rate-bar-bg">
              <view class="rate-bar-fill" :style="{ width: notice.readRate + '%' }" />
            </view>
            <text class="rate-text">已读 {{ notice.readRate }}%</text>
          </view>
          <text class="arrow-icon">&#xeaa0;</text>
        </view>
      </view>

      <!-- 学生花名册 -->
      <view class="section">
        <view class="section-head">
          <text class="section-title">学生花名册</text>
          <view class="view-all" @click="viewAllStudents">
            <text class="view-all-text">查看全部</text>
            <text class="arrow-icon sm">&#xeaa0;</text>
          </view>
        </view>
        <view class="search-filter-row">
          <view class="search-box">
            <text class="search-icon">&#xea76;</text>
            <input
              class="search-input"
              placeholder="搜索学生姓名"
              placeholder-style="color:#94A3B8;font-size:26rpx"
              :value="searchKeyword"
              @input="onSearchInput"
            />
          </view>
          <scroll-view scroll-x class="filter-scroll" :show-scrollbar="false">
            <view class="filter-row">
              <view
                v-for="f in filters"
                :key="f"
                class="filter-pill"
                :class="{ active: activeFilter === f }"
                @click="switchFilter(f)"
              >
                <text>{{ f }}</text>
              </view>
            </view>
          </scroll-view>
        </view>
        <view v-if="loading" class="state-wrap">
          <text class="state-text">加载中...</text>
        </view>
        <view v-else-if="filteredStudents.length === 0" class="state-wrap">
          <text class="state-text">暂无匹配学生</text>
        </view>
        <view v-else class="student-list">
          <view
            v-for="student in filteredStudents"
            :key="student.id"
            class="student-card"
            @click="goStudent(student)"
          >
            <image
              v-if="student.avatar"
              class="stu-avatar"
              :src="student.avatar"
              mode="aspectFill"
            />
            <view v-else class="stu-avatar fallback">
              <text>{{ (student.name || "学").slice(0, 1) }}</text>
            </view>
            <view class="stu-info">
              <view class="stu-name-row">
                <text class="stu-name">{{ student.name }}</text>
                <text class="stu-no">{{ student.student_no || "" }}</text>
              </view>
            </view>
            <view class="stu-status" :class="statusClassMap[student._status]">
              <view class="status-dot" />
              <text class="status-text">{{ student._status }}</text>
            </view>
            <text class="arrow-icon">&#xeaa0;</text>
          </view>
        </view>
      </view>

      <view class="bottom-safe" />
    </scroll-view>

    <!-- 底部导航 -->
    <BottomNav :active="1" />
  </view>
</template>

<style scoped lang="scss">
// === Palette ===
$blue-500: #3B82F6;
$blue-600: #2563EB;
$slate-50: #F8FAFC;
$slate-100: #F1F5F9;
$slate-200: #E2E8F0;
$slate-400: #94A3B8;
$slate-500: #64748B;
$slate-600: #475569;
$slate-800: #1E293B;
$red-500: #EF4444;
$green-500: #10B981;
$yellow-500: #EAB308;
$orange-500: #F97316;

.page {
  min-height: 100vh;
  background: $slate-50;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow-x: hidden;
}

// ===== 顶部导航栏 =====
.header {
  background: linear-gradient(135deg, $blue-500 0%, $blue-600 100%);
  padding: 96rpx 48rpx 48rpx;
  box-sizing: border-box;
  width: 100%;
}

.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.class-title-row {
  display: flex;
  align-items: center;
  gap: 24rpx;
}

.class-icon-box {
  width: 96rpx;
  height: 96rpx;
  border-radius: 9999rpx;
  background: rgba(255,255,255,0.2);
  display: flex;
  align-items: center;
  justify-content: center;
}

.class-icon {
  font-size: 48rpx;
  color: #fff;
}

.class-name-col {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.class-name-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.class-name {
  font-size: 36rpx;
  font-weight: 800;
  color: #fff;
}

.edit-icon {
  font-size: 40rpx;
  color: rgba(255,255,255,0.7);
}

.class-subtitle {
  font-size: 24rpx;
  color: rgba(255,255,255,0.7);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.hdr-btn {
  width: 80rpx;
  height: 80rpx;
  border-radius: 9999rpx;
  background: rgba(255,255,255,0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.hdr-btn-icon {
  font-size: 44rpx;
  color: #fff;
}

.hdr-badge {
  position: absolute;
  top: 14rpx;
  right: 16rpx;
  width: 18rpx;
  height: 16rpx;
  background: $red-500;
  border-radius: 9999rpx;
  border: 4rpx solid rgba(255,255,255,0.2);
}

// ===== 内容滚动区 =====
.content-scroll {
  flex: 1;
  padding: 24rpx 28rpx;
  height: 0;
  box-sizing: border-box;
  width: 100%;
}

// ===== 班级信息卡 =====
.info-card {
  background: #fff;
  border-radius: 32rpx;
  padding: 32rpx;
  box-shadow: 0 4rpx 16rpx rgba(0,0,0,0.05);
  margin-bottom: 24rpx;
  box-sizing: border-box;
  width: 100%;
}

.info-head {
  display: flex;
  align-items: center;
  gap: 24rpx;
}

.info-school-icon {
  width: 88rpx;
  height: 88rpx;
  border-radius: 24rpx;
  background: $slate-100;
  display: flex;
  align-items: center;
  justify-content: center;
}

.info-school-icon-text {
  font-size: 48rpx;
  color: $slate-500;
}

.info-head-text {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.info-head-title {
  font-size: 32rpx;
  font-weight: 700;
  color: $slate-800;
}

.info-head-sub {
  font-size: 26rpx;
  color: $slate-400;
}

.info-divider {
  width: 100%;
  height: 2rpx;
  background: $slate-100;
  margin: 28rpx 0;
}

.info-row {
  display: flex;
  align-items: center;
}

.info-item {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.info-v-divider {
  width: 4rpx;
  height: 60rpx;
  border-radius: 2rpx;
  background: $slate-100;
  margin: 0 24rpx;
}

.info-item-avatar {
  width: 64rpx;
  height: 64rpx;
  border-radius: 9999rpx;
  background: $slate-100;
  display: flex;
  align-items: center;
  justify-content: center;
}

.info-item-avatar-text {
  font-size: 32rpx;
  color: $slate-500;
}

.info-item-icon-box {
  width: 64rpx;
  height: 64rpx;
  border-radius: 18rpx;
  background: $slate-100;
  display: flex;
  align-items: center;
  justify-content: center;
}

.info-item-icon-text {
  font-size: 36rpx;
  color: $slate-500;
}

.info-item-text {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.info-item-label {
  font-size: 22rpx;
  color: $slate-400;
}

.info-item-value {
  font-size: 28rpx;
  font-weight: 600;
  color: $slate-800;
}

// ===== Section 通用 =====
.section {
  margin-bottom: 24rpx;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16rpx;
}

.section-title {
  font-size: 32rpx;
  font-weight: 700;
  color: $slate-800;
}

.section-date {
  font-size: 24rpx;
  color: $slate-400;
}

// ===== 今日快照 =====
.snapshot-row {
  display: flex;
  gap: 20rpx;
}

.snap-item {
  flex: 1;
  background: #fff;
  border-radius: 24rpx;
  padding: 24rpx 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
  box-shadow: 0 2rpx 8rpx rgba(0,0,0,0.04);
}

.snap-num {
  font-size: 52rpx;
  font-weight: 800;
  &.green  { color: $green-500; }
  &.yellow { color: $yellow-500; }
  &.orange { color: $orange-500; }
  &.blue   { color: $blue-500; }
}

.snap-unit {
  font-size: 22rpx;
  color: $slate-400;
  margin-top: -4rpx;
}

.snap-label {
  font-size: 24rpx;
  color: $slate-500;
  font-weight: 500;
}

// ===== 快捷按钮行 =====
.quick-row {
  display: flex;
  gap: 20rpx;
  margin-bottom: 24rpx;
}

.quick-btn {
  flex: 1;
  background: #fff;
  border-radius: 24rpx;
  padding: 28rpx 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8rpx;
  box-shadow: 0 2rpx 8rpx rgba(0,0,0,0.04);
}

.quick-icon {
  font-size: 44rpx;
  color: $blue-500;
}

.quick-label {
  font-size: 26rpx;
  color: $slate-800;
  font-weight: 500;
}

// ===== 通知卡片 =====
.sub-title-row {
  margin-bottom: 16rpx;
}

.sub-title {
  font-size: 28rpx;
  font-weight: 600;
  color: $slate-800;
}

.notice-card {
  background: #fff;
  border-radius: 24rpx;
  padding: 24rpx 28rpx;
  display: flex;
  align-items: center;
  gap: 20rpx;
  box-shadow: 0 2rpx 8rpx rgba(0,0,0,0.04);
  margin-bottom: 16rpx;
}

.notice-icon-box {
  width: 80rpx;
  height: 80rpx;
  border-radius: 20rpx;
  background: #DBEAFE;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.notice-icon {
  font-size: 44rpx;
  color: $blue-500;
}

.notice-body {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.notice-title {
  font-size: 28rpx;
  font-weight: 600;
  color: $slate-800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notice-time {
  font-size: 22rpx;
  color: $slate-400;
}

.notice-rate {
  display: flex;
  align-items: center;
  gap: 8rpx;
  flex-shrink: 0;
}

.rate-bar-bg {
  width: 96rpx;
  height: 12rpx;
  border-radius: 9999rpx;
  background: $slate-200;
  overflow: hidden;
}

.rate-bar-fill {
  height: 100%;
  border-radius: 9999rpx;
  background: $blue-500;
}

.rate-text {
  font-size: 20rpx;
  color: $slate-400;
  white-space: nowrap;
}

// ===== 搜索筛选 =====
.search-filter-row {
  display: flex;
  align-items: center;
  gap: 16rpx;
  margin-bottom: 20rpx;
}

.search-box {
  flex: 1;
  height: 80rpx;
  background: #fff;
  border-radius: 20rpx;
  border: 2rpx solid $slate-200;
  display: flex;
  align-items: center;
  padding: 0 24rpx;
  gap: 16rpx;
  box-sizing: border-box;
}

.search-icon {
  font-size: 40rpx;
  color: $slate-400;
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  font-size: 26rpx;
  color: $slate-800;
}

.filter-scroll {
  white-space: nowrap;
  flex-shrink: 0;
}

.filter-row {
  display: flex;
  gap: 12rpx;
}

.filter-pill {
  flex-shrink: 0;
  padding: 16rpx 28rpx;
  border-radius: 9999rpx;
  background: #fff;
  border: 2rpx solid $slate-200;

  text {
    font-size: 26rpx;
    color: $slate-500;
  }
}

.filter-pill.active {
  background: $blue-500;
  border-color: $blue-500;

  text {
    color: #fff;
    font-weight: 600;
  }
}

// ===== 状态占位 =====
.state-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 120rpx 0;
}

.state-text {
  font-size: 28rpx;
  color: $slate-400;
}

// ===== 学生列表 =====
.student-list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.student-card {
  background: #fff;
  border-radius: 24rpx;
  padding: 24rpx 28rpx;
  display: flex;
  align-items: center;
  gap: 20rpx;
  box-shadow: 0 2rpx 8rpx rgba(0,0,0,0.04);
}

.stu-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 9999rpx;
  flex-shrink: 0;
}

.stu-avatar.fallback {
  background: $slate-100;
  display: flex;
  align-items: center;
  justify-content: center;

  text {
    font-size: 36rpx;
    font-weight: 700;
    color: $slate-500;
  }
}

.stu-info {
  flex: 1;
  overflow: hidden;
}

.stu-name-row {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.stu-name {
  font-size: 30rpx;
  font-weight: 600;
  color: $slate-800;
}

.stu-no {
  font-size: 24rpx;
  color: $slate-400;
}

.stu-status {
  display: flex;
  align-items: center;
  gap: 8rpx;
  flex-shrink: 0;
}

.status-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 9999rpx;
}

.status-text {
  font-size: 24rpx;
  font-weight: 500;
}

.status-present {
  .status-dot { background: $green-500; }
  .status-text { color: $green-500; }
}

.status-leave {
  .status-dot { background: $blue-500; }
  .status-text { color: $blue-500; }
}

.status-absent {
  .status-dot { background: $red-500; }
  .status-text { color: $red-500; }
}

.status-late {
  .status-dot { background: $orange-500; }
  .status-text { color: $orange-500; }
}

.arrow-icon {
  font-size: 40rpx;
  color: $slate-400;
  flex-shrink: 0;

  &.sm {
    font-size: 36rpx;
  }
}

.view-all {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.view-all-text {
  font-size: 26rpx;
  color: $blue-500;
}

.bottom-safe {
  height: calc(80rpx + env(safe-area-inset-bottom));
}
</style>
