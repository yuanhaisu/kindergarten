<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh, onShareAppMessage } from "@dcloudio/uni-app"
import { fetchTeacherHomeOverview, fetchTeacherTodos, updateTeacherTodoAction } from "@/api/teacher-home"
import { ensureLogin } from "@/utils/login"

const loading = ref(false)
const overview = ref({})
const todos = ref([])
const loadError = ref("")

const teacherName = computed(() => overview.value.teacher_name || "老师")
const className = computed(() => overview.value.class_info?.name || overview.value.class_name || "小一班")
const classRole = computed(() => overview.value.class_info?.role || overview.value.role || "主班老师")
const visibleTodos = computed(() => todos.value.slice(0, 3))

const todayDate = computed(() => {
  const now = new Date()
  const y = now.getFullYear()
  const m = now.getMonth() + 1
  const d = now.getDate()
  const weekdays = ["星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"]
  return `${y}年${m}月${d}日 ${weekdays[now.getDay()]}`
})

const quickActions = [
  { title: "发布通知", icon: "通", bg: "#DBEAFE", fg: "#3B82F6", path: "/pages/notice/create" },
  { title: "成长记录", icon: "录", bg: "#DCFCE7", fg: "#22C55E", path: "/pages/growth/index" },
  { title: "接送管理", icon: "接", bg: "#FFEDD5", fg: "#F97316", path: "/pages/pickup/index" },
  { title: "亲子任务", icon: "任", bg: "#F3E8FF", fg: "#A855F7", path: "/pages/tasks/index" },
  { title: "班级日历", icon: "历", bg: "#CFFAFE", fg: "#06B6D4", path: "/pages/calendar/index" },
  { title: "请假审批", icon: "审", bg: "#FEF9C3", fg: "#CA8A04", path: "/pages/leave/index" },
  { title: "班级相册", icon: "册", bg: "#FCE7F3", fg: "#EC4899", path: "/pages/album/index" },
  { title: "更多", icon: "更", bg: "#F1F5F9", fg: "#64748B", path: "/pages/more/index" }
]

const todoIconMap = {
  approve: { bg: "#FEE2E2", fg: "#EF4444", ch: "审" },
  reject: { bg: "#FEE2E2", fg: "#EF4444", ch: "审" },
  share: { bg: "#FFEDD5", fg: "#F97316", ch: "通" },
  go_record: { bg: "#F3E8FF", fg: "#A855F7", ch: "录" }
}

const todoColorMap = {
  approve: { bg: "#F0FDF4", fg: "#10B981" },
  reject: { bg: "#FEF2F2", fg: "#EF4444" },
  share: { bg: "#EFF6FF", fg: "#3B82F6" },
  go_record: { bg: "#FAF5FF", fg: "#8B5CF6" }
}

async function loadHome() {
  loading.value = true
  loadError.value = ""

  try {
    await ensureLogin()
    const [overviewRes, todosRes] = await Promise.all([fetchTeacherHomeOverview(), fetchTeacherTodos()])
    overview.value = overviewRes.data || {}
    todos.value = todosRes.data?.list || todosRes.data || []
  } catch (err) {
    loadError.value = err?.message || "首页数据加载失败"
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

function handleAction(action) {
  uni.navigateTo({ url: action.path })
}

function navigateTo(path) {
  if (path === "/pages/index/index") {
    uni.pageScrollTo({ scrollTop: 0, duration: 300 })
    return
  }
  uni.navigateTo({ url: path })
}

function todoActionLabel(action) {
  const labels = { approve: "同意", reject: "拒绝", share: "转发", go_record: "去记录" }
  return labels[action] || "处理"
}

function todoActionBg(action) {
  return todoColorMap[action] || { bg: "#EFF6FF", fg: "#3B82F6" }
}

function getTodoIcon(todo) {
  const primaryAction = todo.actions?.[0] || ""
  return todoIconMap[primaryAction] || { bg: "#F1F5F9", fg: "#64748B", ch: "待" }
}

function askRejectReason() {
  return new Promise((resolve) => {
    uni.showModal({
      title: "拒绝原因",
      editable: true,
      placeholderText: "请填写拒绝原因",
      success(res) {
        resolve(res.confirm ? res.content || "" : "")
      },
      fail() {
        resolve("")
      }
    })
  })
}

async function handleTodoAction(todo, action) {
  if (action === "go_record") {
    uni.navigateTo({ url: "/pages/growth/create" })
    return
  }
  let remark = ""
  if (action === "reject") {
    remark = await askRejectReason()
    if (!remark.trim()) {
      uni.showToast({ title: "请填写拒绝原因", icon: "none" })
      return
    }
  }

  try {
    const res = await updateTeacherTodoAction(todo.reference_id || todo.id, action, remark)
    uni.showToast({ title: res.data?.message || "处理成功", icon: "none" })
    await loadHome()
  } catch (err) {
    uni.showToast({ title: err?.message || "处理失败", icon: "none" })
  }
}

onLoad(loadHome)
onPullDownRefresh(loadHome)

onShareAppMessage((res) => {
  if (res.from === "button") {
    return {
      title: `班级通知未读提醒 - ${className.value}`,
      path: "/pages/notice/index"
    }
  }
  return {
    title: "养成星球",
    path: "/pages/index/index"
  }
})
</script>

<template>
  <view class="page">
    <!-- 顶部渐变区域 -->
    <view class="header-area">
      <view class="status-bar" :style="{ height: 'var(--status-bar-height)' }" />
      <view class="header-row">
        <view class="user-info">
          <view class="avatar">
            <text class="avatar-icon">{{ teacherName[0] }}</text>
          </view>
          <view class="user-text">
            <text class="user-name">{{ teacherName }}</text>
            <text class="user-sub">{{ className }} · {{ classRole }}</text>
          </view>
        </view>
        <view class="bell-btn" @click="navigateTo('/pages/messages/index')">
          <view class="bell-dot" v-if="overview.unread_notices > 0" />
          <text class="bell-icon">铃</text>
        </view>
      </view>

      <!-- 今日数据卡片 -->
      <view class="stats-card">
        <view class="stats-head">
          <text class="stats-title">今日班级概况</text>
          <text class="stats-date">{{ todayDate }}</text>
        </view>
        <view class="stats-grid">
          <view class="stat-item">
            <text class="stat-num is-green">{{ overview.attendance?.present ?? overview.present_count ?? 0 }}</text>
            <text class="stat-label">已出勤</text>
          </view>
          <view class="stat-item">
            <text class="stat-num is-amber">{{ overview.attendance?.leave ?? overview.leave_count ?? 0 }}</text>
            <text class="stat-label">请假</text>
          </view>
          <view class="stat-item">
            <text class="stat-num is-red">{{ overview.pending_approvals ?? todos.length ?? 0 }}</text>
            <text class="stat-label">待审批</text>
          </view>
          <view class="stat-item">
            <text class="stat-num is-blue">{{ overview.unread_notices ?? overview.unread_count ?? 0 }}</text>
            <text class="stat-label">未读通知</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 功能入口 -->
    <view class="feature-card">
      <view class="feature-grid">
        <view
          v-for="action in quickActions"
          :key="action.title"
          class="feature-item"
          @click="handleAction(action)"
        >
          <view class="feature-icon-box" :style="{ backgroundColor: action.bg }">
            <text class="feature-icon" :style="{ color: action.fg }">{{ action.icon }}</text>
          </view>
          <text class="feature-label">{{ action.title }}</text>
        </view>
      </view>
    </view>

    <!-- 待办事项 -->
    <view class="todo-section">
      <view class="todo-head">
        <text class="todo-title">待办事项</text>
        <text class="todo-more" @click="navigateTo('/pages/more/index')">查看全部</text>
      </view>

      <view v-if="loadError" class="state-card">
        <text class="state-text">{{ loadError }}</text>
        <button class="retry-btn" @click="loadHome">重试</button>
      </view>

      <view v-else-if="!loading && todos.length === 0" class="state-card">
        <text class="state-text">今天没有新的待办，保持轻松节奏。</text>
      </view>

      <view v-else class="todo-list">
        <view v-for="todo in visibleTodos" :key="todo.id || todo.title" class="todo-card">
          <view class="todo-icon-box" :style="{ backgroundColor: getTodoIcon(todo).bg }">
            <text class="todo-icon" :style="{ color: getTodoIcon(todo).fg }">{{ getTodoIcon(todo).ch }}</text>
          </view>
          <view class="todo-body">
            <text class="todo-item-title">{{ todo.title || "待处理事项" }}</text>
            <text class="todo-item-desc">{{ todo.description || todo.content || "请及时处理班级事务" }}</text>
          </view>
          <view v-if="todo.actions?.length" class="todo-btns">
            <template v-for="action in todo.actions" :key="action">
              <button
                v-if="action === 'share'"
                class="todo-btn share-btn"
                :style="{
                  backgroundColor: todoActionBg(action).bg,
                  color: todoActionBg(action).fg
                }"
                open-type="share"
              >
                {{ todoActionLabel(action) }}
              </button>
              <view
                v-else
                class="todo-btn"
                :style="{
                  backgroundColor: todoActionBg(action).bg,
                  color: todoActionBg(action).fg
                }"
                @click="handleTodoAction(todo, action)"
              >
                {{ todoActionLabel(action) }}
              </view>
            </template>
          </view>
        </view>
      </view>
    </view>

    <!-- 底部导航栏 -->
    <view class="bottom-nav">
      <view class="nav-item active" @click="navigateTo('/pages/index/index')">
        <text class="nav-icon active-icon">首</text>
        <text class="nav-label active-label">首页</text>
      </view>
      <view class="nav-item" @click="navigateTo('/pages/class/index')">
        <text class="nav-icon">班</text>
        <text class="nav-label">班级</text>
      </view>
      <view class="nav-item" @click="navigateTo('/pages/messages/index')">
        <text class="nav-icon">消</text>
        <text class="nav-label">消息</text>
      </view>
      <view class="nav-item" @click="navigateTo('/pages/profile/index')">
        <text class="nav-icon">我</text>
        <text class="nav-label">我的</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
$blue-500: #3B82F6;
$blue-600: #2563EB;
$slate-800: #1E293B;
$slate-500: #64748B;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  width: 100%;
  max-width: 430px;
  margin: 0 auto;
  background: #F8FAFC;
}

// ===== 顶部渐变区域 =====
.header-area {
  padding: 0 46rpx 46rpx;
  background: linear-gradient(135deg, $blue-500 0%, $blue-600 100%);
}

.status-bar {
  width: 100%;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 96rpx;
  margin-top: 24rpx;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 24rpx;
}

.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 96rpx;
  height: 96rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 1);
}

.avatar-icon {
  font-size: 40rpx;
  font-weight: 700;
  color: $blue-500;
}

.user-text {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.user-name {
  font-size: 36rpx;
  font-weight: 700;
  color: #fff;
}

.user-sub {
  font-size: 28rpx;
  color: #DBEAFE;
}

.bell-btn {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: 999rpx;
  background: rgba(255, 255, 255, 0.2);
}

.bell-dot {
  position: absolute;
  top: 2rpx;
  right: 2rpx;
  width: 20rpx;
  height: 20rpx;
  border-radius: 999rpx;
  background: #EF4444;
}

.bell-icon {
  font-size: 36rpx;
  font-weight: 700;
  color: #fff;
}

// ===== 今日数据卡片 =====
.stats-card {
  margin-top: 48rpx;
  padding: 32rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.06);
}

.stats-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.stats-title {
  font-size: 32rpx;
  font-weight: 700;
  color: $slate-800;
}

.stats-date {
  font-size: 26rpx;
  color: $slate-500;
}

.stats-grid {
  display: flex;
  justify-content: space-between;
  margin-top: 32rpx;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8rpx;
}

.stat-num {
  font-size: 48rpx;
  font-weight: 800;
}

.is-green { color: #10B981; }
.is-amber { color: #F59E0B; }
.is-red   { color: #EF4444; }
.is-blue  { color: $blue-500; }

.stat-label {
  font-size: 24rpx;
  color: $slate-500;
}

// ===== 功能入口 =====
.feature-card {
  margin: 32rpx 46rpx;
  padding: 32rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.06);
}

.feature-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  row-gap: 48rpx;
  column-gap: 20rpx;
}

.feature-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18rpx;
}

.feature-icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  border-radius: 32rpx;
}

.feature-icon {
  font-size: 46rpx;
  font-weight: 700;
}

.feature-label {
  font-size: 26rpx;
  font-weight: 500;
  color: #334155;
}

// ===== 待办事项 =====
.todo-section {
  padding: 0 46rpx 180rpx;
}

.todo-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 32rpx;
}

.todo-title {
  font-size: 36rpx;
  font-weight: 700;
  color: $slate-800;
}

.todo-more {
  font-size: 28rpx;
  color: $blue-500;
}

.state-card {
  padding: 56rpx 32rpx;
  border-radius: 28rpx;
  background: #fff;
  text-align: center;
  box-shadow: 0 4rpx 12rpx rgba(0, 0, 0, 0.04);
}

.state-text {
  font-size: 28rpx;
  color: $slate-500;
}

.retry-btn {
  width: 180rpx;
  height: 72rpx;
  margin: 28rpx auto 0;
  border-radius: 999rpx;
  background: $blue-500;
  color: #fff;
  font-size: 26rpx;
}

// 待办列表
.todo-list {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.todo-card {
  display: flex;
  align-items: center;
  gap: 24rpx;
  padding: 32rpx;
  border-radius: 24rpx;
  background: #fff;
  box-shadow: 0 4rpx 12rpx rgba(0, 0, 0, 0.04);
}

.todo-icon-box {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: 999rpx;
}

.todo-icon {
  font-size: 34rpx;
  font-weight: 700;
}

.todo-body {
  flex: 1;
  min-width: 0;
}

.todo-item-title {
  display: block;
  font-size: 30rpx;
  font-weight: 600;
  color: $slate-800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.todo-item-desc {
  display: block;
  margin-top: 8rpx;
  font-size: 24rpx;
  color: $slate-500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.todo-btns {
  display: flex;
  flex-shrink: 0;
  gap: 16rpx;
}

.todo-btn {
  padding: 12rpx 24rpx;
  border-radius: 16rpx;
  font-size: 26rpx;
  font-weight: 500;
  white-space: nowrap;
}

.share-btn {
  margin: 0;
  border: none;
  line-height: inherit;
}

.share-btn::after {
  border: none;
}

// ===== 底部导航栏 =====
.bottom-nav {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  display: flex;
  justify-content: space-between;
  max-width: 430px;
  margin: 0 auto;
  padding: 24rpx 46rpx calc(24rpx + env(safe-area-inset-bottom));
  background: #fff;
  box-shadow: 0 -2rpx 20rpx rgba(0, 0, 0, 0.06);
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
  padding: 8rpx 0;
}

.nav-icon {
  font-size: 48rpx;
  font-weight: 600;
  color: #94A3B8;
}

.nav-label {
  font-size: 22rpx;
  color: #94A3B8;
}

.active-icon {
  color: $blue-500;
}

.active-label {
  color: $blue-500;
  font-weight: 600;
}
</style>
