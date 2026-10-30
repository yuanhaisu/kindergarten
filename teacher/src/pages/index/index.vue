<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh, onShareAppMessage } from "@dcloudio/uni-app"
import { fetchTeacherHomeOverview, fetchTeacherTodos, updateTeacherTodoAction } from "@/api/teacher-home"
import { ensureLogin } from "@/utils/login"
import { goTab } from "@/utils/nav"
import BottomNav from "@/components/BottomNav.vue"
import PageShell from "@/components/PageShell.vue"
import AppIcon from "@/components/AppIcon.vue"

const loading = ref(false)
const overview = ref({})
const todos = ref([])
const processed = ref({})
const loadError = ref("")

const teacherName = computed(() => overview.value.teacher_name || "老师")
const className = computed(() => overview.value.class_info?.name || overview.value.class_name || "小一班")
const classRole = computed(() => overview.value.class_info?.role || overview.value.role || "主班老师")
const visibleTodos = computed(() => todos.value.slice(0, 3))
const unreadCount = computed(() => Number(overview.value.unread_notices || 0))
const presentCount = computed(() => overview.value.attendance?.present ?? overview.value.present_count ?? 0)
const leaveCount = computed(() => overview.value.attendance?.leave ?? overview.value.leave_count ?? 0)
const pendingCount = computed(() => overview.value.pending_approvals ?? todos.value.length ?? 0)

const todayDate = computed(() => {
  const now = new Date()
  const y = now.getFullYear()
  const m = now.getMonth() + 1
  const d = now.getDate()
  const weekdays = ["星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"]
  return `${y}年${m}月${d}日 ${weekdays[now.getDay()]}`
})

// 图标改为品牌插画位图（统一暖橙渐变，消除原先 8 个入口 8 种颜色的「彩虹」）。
// 位图不随主题换色，故所有入口共用同一套图标；语义区分交给图形本身。
// 页面只记图标名，路径与降级由 AppIcon 统一处理。
const quickActions = [
  { title: "发布通知", icon: "notice", path: "/pages/notice/create" },
  { title: "成长记录", icon: "growth", path: "/pages/growth/index" },
  { title: "接送管理", icon: "pickup", path: "/pages/pickup/index" },
  { title: "亲子任务", icon: "task", path: "/pages/tasks/index" },
  { title: "班级日历", icon: "calendar", path: "/pages/calendar/index" },
  { title: "请假审批", icon: "leave", path: "/pages/leave/index" },
  { title: "班级相册", icon: "album", path: "/pages/album/index" },
  { title: "更多", icon: "more", path: "/pages/more/index" }
]

const todoIconMap = {
  approve: "审",
  reject: "审",
  share: "通",
  go_record: "录"
}

async function loadHome() {
  loading.value = true
  loadError.value = ""

  try {
    await ensureLogin()
    const [overviewRes, todosRes] = await Promise.all([fetchTeacherHomeOverview(), fetchTeacherTodos()])
    overview.value = overviewRes.data || {}
    const todoList = todosRes.data?.list ?? todosRes.data
    todos.value = Array.isArray(todoList) ? todoList : []
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

// 底部 tab 统一走 goTab：它自带「已是当前页则滚动置顶」的处理，
// 也避免用 navigateTo push 导致页面栈无上限增长（超过 10 层会静默失败）。
function navigateTo(path) {
  goTab(path)
}

function todoActionLabel(action) {
  const labels = { approve: "同意", reject: "拒绝", share: "转发", go_record: "去记录" }
  return labels[action] || "处理"
}

function todoKey(todo) {
  return String(todo.id || todo.reference_id || todo.title || "")
}

function isProcessed(todo) {
  return Boolean(processed.value[todoKey(todo)])
}

function processedActionLabel(todo, action) {
  const done = processed.value[todoKey(todo)]
  if (!done) return todoActionLabel(action)
  if (done === action) return done === "approve" ? "已同意" : done === "reject" ? "已拒绝" : "已处理"
  return "已处理"
}

function getTodoIcon(todo) {
  const primaryAction = todo.actions?.[0] || ""
  return { ch: todoIconMap[primaryAction] || "待" }
}

function goRecord() {
  uni.navigateTo({ url: "/pages/growth/create" })
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
    goRecord()
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
    if (action === "approve" || action === "reject") {
      processed.value[todoKey(todo)] = action
    }
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
  <PageShell bottom-space="180rpx" max-width="430px" flush-top>
    <!-- 问候区：头像进「我的」，班级行进「班级」，铃铛进「消息」 -->
    <view class="greeting">
      <view class="avatar" @click="navigateTo('/pages/profile/index')">
        <text class="avatar-icon">{{ teacherName[0] }}</text>
      </view>
      <view class="greet-text">
        <text class="greet-name">{{ teacherName }}</text>
        <view class="greet-class" @click="navigateTo('/pages/class/index')">
          <text class="greet-sub">{{ className }} · {{ classRole }}</text>
          <text class="greet-chev">›</text>
        </view>
      </view>
      <view class="bell" @click="navigateTo('/pages/messages/index')">
        <text class="bell-icon">铃</text>
        <view v-if="unreadCount > 0" class="bell-dot" />
      </view>
    </view>

    <!-- 今日概况数字带：只有「待审批」带强调色 -->
    <view class="section-head">
      <text class="section-title">今日班级概况</text>
      <text class="section-meta">{{ todayDate }}</text>
    </view>
    <view class="stat-card">
      <view class="stat">
        <view class="stat-v">
          <text v-if="!loading" class="stat-num">{{ presentCount }}</text>
          <view v-else class="sk sk-num" />
        </view>
        <text class="stat-k">已出勤</text>
      </view>
      <view class="stat">
        <view class="stat-v">
          <text v-if="!loading" class="stat-num">{{ leaveCount }}</text>
          <view v-else class="sk sk-num" />
        </view>
        <text class="stat-k">请假</text>
      </view>
      <view class="stat stat--accent">
        <view class="stat-v">
          <text v-if="!loading" class="stat-num stat-num--pill">{{ pendingCount }}</text>
          <view v-else class="sk sk-num" />
        </view>
        <text class="stat-k">待审批</text>
      </view>
      <view class="stat">
        <view class="stat-v">
          <text v-if="!loading" class="stat-num">{{ unreadCount }}</text>
          <view v-else class="sk sk-num" />
        </view>
        <text class="stat-k">未读通知</text>
      </view>
    </view>

    <!-- 功能入口 4 × 2 -->
    <view class="entries">
      <view
        v-for="action in quickActions"
        :key="action.title"
        class="entry"
        :class="{ 'entry--more': action.title === '更多' }"
        @click="handleAction(action)"
      >
        <view class="entry-ico">
          <AppIcon :name="action.icon" :size="72" :label="action.title" />
        </view>
        <text class="entry-label">{{ action.title }}</text>
      </view>
    </view>

    <!-- 待办事项：一张分格卡，操作在第二行右对齐 -->
    <view class="todos">
      <view class="section-head">
        <text class="section-title">待办事项</text>
        <text class="link-btn" @click="navigateTo('/pages/more/index')">查看全部 ›</text>
      </view>

      <view v-if="loading" class="todo-card">
        <view v-for="i in 2" :key="i" class="sk-row" :class="{ 'sk-row--first': i === 1 }">
          <view class="sk sk-chip" />
          <view class="sk-lines">
            <view class="sk sk-line" />
            <view class="sk sk-line sk-line--short" />
          </view>
        </view>
      </view>

      <view v-else-if="loadError" class="todo-card todo-state">
        <view class="state-ico state-ico--error"><text>!</text></view>
        <text class="state-text">{{ loadError }}</text>
        <button class="btn btn--soft" @click="loadHome">重试</button>
      </view>

      <view v-else-if="todos.length === 0" class="todo-card todo-state">
        <view class="state-ico"><text>今</text></view>
        <text class="state-text">今天没有新的待办，保持轻松节奏。</text>
        <view class="btn btn--soft" @click="goRecord">写今日记录</view>
      </view>

      <view v-else class="todo-card">
        <view v-for="todo in visibleTodos" :key="todo.id || todo.title" class="todo-item">
          <view class="todo-row">
            <view class="todo-chip">{{ getTodoIcon(todo).ch }}</view>
            <view class="todo-main">
              <text class="todo-title">{{ todo.title || "待处理事项" }}</text>
              <text class="todo-desc">{{ todo.description || todo.content || "请及时处理班级事务" }}</text>
              <view v-if="todo.actions?.length" class="todo-actions">
                <template v-for="action in todo.actions" :key="action">
                  <button
                    v-if="action === 'share'"
                    class="btn btn--quiet"
                    open-type="share"
                  >
                    {{ todoActionLabel(action) }}
                  </button>
                  <button
                    v-else
                    class="btn"
                    :class="[action === 'reject' ? 'btn--quiet' : 'btn--soft', { 'btn--disabled': isProcessed(todo) }]"
                    :disabled="isProcessed(todo)"
                    @click="!isProcessed(todo) && handleTodoAction(todo, action)"
                  >
                    {{ processedActionLabel(todo, action) }}
                  </button>
                </template>
              </view>
            </view>
          </view>
        </view>
      </view>
    </view>

    <!-- 底部导航栏：4 个常驻页共用组件，避免每页复制 -->
    <template #bottom>
      <BottomNav :active="0" :unread-count="unreadCount" />
    </template>
  </PageShell>
</template>

<style scoped lang="scss">

// ===== 问候区 =====
.greeting {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: $spacing-sm 0 $spacing-xl;
}

.avatar {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  border: 1rpx solid $color-border-hairline;
  color: $color-brand-primary-dark;
}

.avatar-icon {
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  line-height: 1;
}

.greet-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.greet-name {
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  line-height: 1.4;
  color: $color-text-primary;
}

.greet-class {
  display: inline-flex;
  align-items: center;
  gap: 2rpx;
  margin-top: 4rpx;
  padding: $spacing-xs 0;
}

.greet-sub {
  font-size: $typography-font-size-base;
  color: $color-text-body;
  line-height: 1.5;
}

.greet-chev {
  color: $color-text-muted;
  font-size: $typography-font-size-lg;
  line-height: 1;
}

.bell {
  position: relative;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: $component-input-height;
  height: $component-input-height;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.bell-icon {
  font-size: $typography-font-size-xl;
  font-weight: 600;
  color: $color-text-body;
  line-height: 1;
}

.bell-dot {
  position: absolute;
  top: 20rpx;
  right: 20rpx;
  width: 16rpx;
  height: 16rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
  outline: 4rpx solid $color-bg-card;
}

// ===== 区块标题 =====
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  min-height: $component-input-height;
}

.section-title {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  line-height: 1.4;
  color: $color-text-primary;
}

.section-meta {
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1.4;
  color: $color-text-body;
  font-variant-numeric: tabular-nums;
}

.link-btn {
  display: inline-flex;
  align-items: center;
  min-height: $component-input-height;
  padding: 0 2rpx;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: 500;
}

// ===== 今日概况数字带 =====
.stat-card {
  display: flex;
  margin-top: 4rpx;
  background: $color-bg-card;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
  overflow: hidden;
}

.stat {
  position: relative;
  flex: 1 1 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
  padding: $spacing-lg 4rpx;
}

.stat + .stat {
  border-left: 1rpx solid $color-border-hairline;
}

.stat-v {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 56rpx;
}

.stat-num {
  font-size: $typography-font-size-3xl;
  font-weight: 600;
  line-height: 1;
  color: $color-text-primary;
  font-variant-numeric: tabular-nums;
}

.stat-num--pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 68rpx;
  height: 56rpx;
  padding: 0 20rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.stat-k {
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1.4;
  color: $color-text-muted;
  white-space: nowrap;
}

// ===== 功能入口 =====
.entries {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: $spacing-lg 20rpx;
  margin: $spacing-xl 0 $spacing-md;
}

.entry {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10rpx;
  min-height: $component-input-height;
}

.entry-ico {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
}

.entry-label {
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1.4;
  color: $color-text-body;
  white-space: nowrap;
}

.entry--more .entry-ico {
  background: transparent;
  border: 1rpx solid $color-border-hairline;
}

.entry--more .entry-label {
  color: $color-text-muted;
}

// ===== 待办事项 =====
.todos .section-head {
  margin-bottom: 4rpx;
}

.todo-card {
  background: $color-bg-card;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
  overflow: hidden;
}

.todo-item + .todo-item {
  border-top: 1rpx solid $color-border-hairline;
}

.todo-row {
  display: flex;
  gap: $spacing-lg;
  padding: $spacing-lg $spacing-xl $spacing-xl;
}

.todo-chip {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 60rpx;
  height: 60rpx;
  margin-top: 2rpx;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: 600;
  line-height: 1;
}

.todo-main {
  flex: 1;
  min-width: 0;
}

.todo-title {
  display: block;
  font-size: $typography-font-size-md;
  font-weight: 600;
  line-height: 1.45;
  color: $color-text-primary;
}

.todo-desc {
  display: block;
  margin-top: 6rpx;
  font-size: $typography-font-size-base;
  line-height: 1.55;
  color: $color-text-muted;
}

.todo-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: $spacing-md;
  margin-top: $spacing-lg;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  padding: 0 36rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-base;
  font-weight: 600;
  line-height: 1;
}

.btn::after {
  border: none;
}

.btn--soft {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.btn--quiet {
  padding: 0 24rpx;
  background: transparent;
  color: $color-text-muted;
  font-weight: 500;
}

.btn--disabled {
  background: $color-bg-card;
  border: 1rpx solid $color-border-hairline;
  color: $color-text-muted;
}

// ===== 待办状态 =====
.todo-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $spacing-lg;
  padding: 64rpx $spacing-xl;
  text-align: center;
}

.state-ico {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-xl;
  font-weight: 600;
}

.state-ico--error {
  color: $color-semantic-danger;
  background: $color-semantic-danger-bg;
}

.state-text {
  max-width: 22em;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  color: $color-text-muted;
}

// ===== 骨架 =====
.sk {
  display: block;
  background: $color-border-hairline;
  border-radius: $border-radius-sm;
  opacity: 0.45;
  animation: skPulse 1.6s $animation-easing-default infinite;
}

.sk-num {
  width: 72rpx;
  height: 44rpx;
  border-radius: $border-radius-md;
}

.sk-row {
  display: flex;
  align-items: flex-start;
  gap: $spacing-lg;
  width: 100%;
  padding: $spacing-lg 0;
}

.sk-row--first {
  padding-top: $spacing-xl;
}

.sk-row + .sk-row {
  border-top: 1rpx solid $color-border-hairline;
}

.sk-chip {
  flex: none;
  width: 60rpx;
  height: 60rpx;
  border-radius: $border-radius-lg;
}

.sk-lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  padding-top: 6rpx;
}

.sk-line {
  width: 78%;
  height: 26rpx;
}

.sk-line--short {
  width: 46%;
  height: 22rpx;
}

@keyframes skPulse {
  0%, 100% { opacity: 0.45; }
  50% { opacity: 0.22; }
}

@media (prefers-reduced-motion: reduce) {
  .sk,
  .spinner {
    animation: none;
  }
}
</style>
