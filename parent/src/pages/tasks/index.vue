<script setup>
/**
 * 亲子任务 · 任务节奏
 *
 * 三段筛选走服务端 scope（B15）：active / checked / all。
 * 契约里的 active 指「正在进行的任务」（status=1 且在起止日期内），
 * 其中可能包含本周期已提交的卡片；「进行中」红点与顶部提醒只统计
 * 这些卡片里未提交的数量，避免把已提交的任务算成待办。
 */
import { computed, ref } from "vue"
import { onLoad, onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchTasks, withStudentScope } from "@/api/parent-task"
import { getCurrentStudentId } from "@/utils/currentStudent"
import { formatDateLabel } from "@/utils/time"
import {
  frequencyTaskLabel,
  starJarLabel,
  taskBadge,
  cycleState,
  primaryAction
} from "@/utils/tasks"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  wall: "", // image-line
  task: "", // task-line
  star: "", // star-fill
  check: "", // check-line
  chat: "", // chat-3-line
  refresh: "", // refresh-line
  alert: "", // error-warning-line
  chevron: "" // arrow-right-s-line
}

const SCOPES = [
  { key: "active", label: "进行中" },
  { key: "checked", label: "已打卡" },
  { key: "all", label: "全部" }
]

// 后端单页上限 50；任务量远小于这个数，一次拉全量避免翻页状态。
const PAGE_SIZE = 50

const tasks = ref([])
// 进行中分段的缓存：非进行中分段也要显示「进行中（N）」与待提交提醒。
const activeTasks = ref([])
const activeScope = ref("active")
const loading = ref(false)
const retrying = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const unbound = ref(false)
const studentId = ref(getCurrentStudentId())

// 只允许最后一次请求写数据：切分段与下拉刷新可能并发。
let loadSeq = 0

const isFirstLoading = computed(() => loading.value && tasks.value.length === 0)
const showList = computed(() => tasks.value.length > 0)
const showFullError = computed(() => !loading.value && Boolean(loadError.value) && tasks.value.length === 0)
const showInlineError = computed(() => Boolean(loadError.value) && tasks.value.length > 0)
const showEmpty = computed(
  () => hasLoaded.value && !loading.value && !loadError.value && tasks.value.length === 0
)

// 待提交数只算「进行中且本周期未提交」——已完成的任务不应被读成待办。
const pendingCount = computed(
  () => activeTasks.value.filter(item => !item.submitted_current_period).length
)

const viewTasks = computed(() =>
  tasks.value.map(item => {
    const due = formatDateLabel(item.end_date)
    const count = Number(item.checkin_count) || 0
    return {
      id: item.id,
      title: item.title,
      desc: item.description || item.intro || "",
      unread: Boolean(item.has_unread_supplement),
      ended: Number(item.status) === 0,
      badge: taskBadge(item),
      cycle: cycleState(item),
      primary: primaryAction(item),
      meta: `${frequencyTaskLabel(item.frequency)} · 累计打卡 ${count} 次`,
      due: due ? `截止 ${due}` : "",
      star: starJarLabel(item)
    }
  })
)

const emptyTitle = computed(() => (activeScope.value === "active" ? "当前没有待完成的任务" : "暂无任务"))
const emptyDesc = computed(() => {
  if (activeScope.value === "active") return "老师发布新任务后会出现在这里"
  if (activeScope.value === "checked") return "本周期还没有提交记录"
  return "老师发布任务后会出现在这里"
})

function taskUrl(id) {
  const sid = studentId.value
  return `/pages/tasks/detail?id=${id}${sid ? `&student_id=${sid}` : ""}`
}

// 任务页不是 tabBar 页（家长端 tabBar 只有 index/growth/pickup/profile），用 navigateTo。
function goDetail(id) {
  uni.navigateTo({ url: taskUrl(id) })
}

// 主按钮与「查看详情」落在同一页：详情里既有提交入口，也有记录与老师回应。
function goPrimary(item) {
  if (item.primary.disabled) return
  goDetail(item.id)
}

// 班级墙不是 tabBar 页，同样走 navigateTo。
function goWall() {
  uni.navigateTo({ url: "/pages/tasks/wall" })
}

function selectScope(key) {
  if (loading.value || retrying.value || activeScope.value === key) return
  activeScope.value = key
  tasks.value = []
  loadData()
}

function viewAll() {
  selectScope("all")
}

async function loadData(options = {}) {
  const keepView = Boolean(options.keepView)
  const scope = activeScope.value
  const seq = ++loadSeq
  if (!keepView) {
    loading.value = true
  }
  loadError.value = ""
  try {
    await ensureLogin()
    // 当前孩子解析与 403 重试统一走 withStudentScope（与详情页同一条路径）。
    const scoped = await withStudentScope(async (sid) => {
      const current = fetchTasks({ student_id: sid, page: 1, page_size: PAGE_SIZE, scope })
      // 非进行中分段并行补一次进行中列表，只用于红点/提醒；失败不影响主列表。
      const active = scope === "active"
        ? null
        : fetchTasks({ student_id: sid, page: 1, page_size: PAGE_SIZE, scope: "active" })
          .catch((err) => {
            console.warn("[tasks-index] pending count failed:", err?.message || err)
            return null
          })
      return { current: await current, active: active ? await active : null }
    })
    if (seq !== loadSeq) return
    if (!scoped) {
      tasks.value = []
      activeTasks.value = []
      unbound.value = true
      loadError.value = "未绑定孩子信息"
      return
    }
    unbound.value = false
    studentId.value = scoped.sid
    tasks.value = scoped.result.current?.data?.list || []
    if (scope === "active") {
      activeTasks.value = tasks.value
    } else if (scoped.result.active?.data?.list) {
      activeTasks.value = scoped.result.active.data.list
    }
    hasLoaded.value = true
  } catch (err) {
    if (seq !== loadSeq) return
    console.warn("[tasks-index] load failed:", err?.message || err)
    // 已有列表时保留旧内容，只显示顶部 inline 错误；清空会把刷新失败伪装成「没有任务」。
    if (keepView && tasks.value.length > 0) {
      loadError.value = "加载失败"
      return
    }
    tasks.value = []
    unbound.value = false
    loadError.value = "任务加载失败"
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
  }
}

async function handleRetry() {
  if (retrying.value || loading.value) return
  retrying.value = true
  await loadData({ keepView: true })
  retrying.value = false
  uni.showToast({
    title: loadError.value ? "仍无法加载，请稍后再试" : "已重新加载",
    icon: "none"
  })
}

async function handleRefresh() {
  if (loading.value || retrying.value) {
    uni.stopPullDownRefresh()
    return
  }
  await loadData({ keepView: true })
}

onLoad(() => {
  loadData()
})
// 从详情页提交后返回不重跑 onLoad：onShow 静默回刷，卡片与红点才能立刻更新。
onShow(() => {
  if (hasLoaded.value && !loading.value && !retrying.value) loadData({ keepView: true })
})
onPullDownRefresh(handleRefresh)
</script>

<template>
  <PageShell bottom-space="48rpx" max-width="430px">
    <!-- ===== 班级墙入口：低矮白卡，暖色只落在图标底 ===== -->
    <view class="wall-entry" @click="goWall">
      <view class="wall-icon">
        <text class="wall-icon-glyph">{{ ICON.wall }}</text>
      </view>
      <view class="wall-copy">
        <text class="wall-title">班级墙</text>
        <text class="wall-desc">看看本班孩子的分享</text>
      </view>
      <text class="wall-chevron">{{ ICON.chevron }}</text>
    </view>

    <!-- ===== 三段分类：服务端 scope ===== -->
    <view class="segmented">
      <view
        v-for="scope in SCOPES"
        :key="scope.key"
        class="segment"
        :class="{ 'is-selected': activeScope === scope.key }"
        @click="selectScope(scope.key)"
      >
        <text class="segment-text">{{ scope.label }}</text>
        <view v-if="scope.key === 'active' && pendingCount > 0" class="segment-badge">
          <text class="segment-badge-text">{{ pendingCount }}</text>
        </view>
      </view>
    </view>

    <!-- ===== 待提交提醒：只在进行中且确有未提交时出现 ===== -->
    <view v-if="activeScope === 'active' && pendingCount > 0 && !showFullError && !isFirstLoading" class="pending-note">
      <view class="pending-dot" />
      <text class="pending-text">本周期还有 <text class="pending-count">{{ pendingCount }}</text> 个任务待提交</text>
    </view>

    <!-- ===== 已有列表时刷新失败：保留旧列表 ===== -->
    <view v-if="showInlineError" class="inline-error" @click="handleRetry">
      <text class="inline-error-icon">{{ ICON.refresh }}</text>
      <text class="inline-error-text">{{ retrying ? "正在重试…" : "加载失败，点击重试" }}</text>
    </view>

    <!-- ===== 任务列表 ===== -->
    <view v-if="showList" class="task-list">
      <view
        v-for="task in viewTasks"
        :key="task.id"
        class="task-card"
        :class="{ 'is-ended': task.ended }"
      >
        <view class="task-top">
          <view class="task-icon">
            <text class="task-icon-glyph">{{ ICON.task }}</text>
            <!-- 补充回应红点：与消息未读是两套红点，这里只标任务内未读 -->
            <view v-if="task.unread" class="task-icon-dot" />
          </view>
          <view class="task-heading">
            <view class="task-titleline">
              <text class="task-title">{{ task.title }}</text>
              <text class="task-badge" :class="'badge--' + task.badge.tone">{{ task.badge.label }}</text>
            </view>
            <text v-if="task.desc" class="task-desc">{{ task.desc }}</text>
          </view>
        </view>

        <view class="task-period">
          <view class="task-meta-line">
            <text class="task-meta">{{ task.meta }}</text>
            <text v-if="task.due" class="task-due">{{ task.due }}</text>
          </view>
          <view class="cycle-row">
            <view class="cycle-state" :class="'cycle--' + task.cycle.tone">
              <view class="cycle-mark" :class="'mark--' + task.cycle.tone">
                <text v-if="task.cycle.tone === 'done'" class="cycle-glyph">{{ ICON.check }}</text>
                <text v-else-if="task.cycle.tone === 'reply'" class="cycle-glyph">{{ ICON.chat }}</text>
                <view v-else-if="task.cycle.tone === 'todo'" class="cycle-hollow" />
                <view v-else class="cycle-dash" />
              </view>
              <text class="cycle-label">{{ task.cycle.label }}</text>
            </view>
            <view class="star-jar">
              <text class="star-glyph">{{ ICON.star }}</text>
              <text class="star-label">{{ task.star }}</text>
            </view>
          </view>
        </view>

        <!-- 补充回应未读：与「已提交/待完成」并列说明原因，避免红点无声无息 -->
        <view v-if="task.unread" class="unread-note">
          <view class="unread-dot" />
          <text class="unread-text">老师又给你留了一句悄悄话</text>
        </view>

        <view class="task-actions">
          <view
            class="task-btn task-btn--primary"
            :class="{ 'is-disabled': task.primary.disabled }"
            @click="goPrimary(task)"
          >
            <text class="task-btn-text">{{ task.primary.label }}</text>
          </view>
          <view class="task-btn task-btn--secondary" @click="goDetail(task.id)">
            <text class="task-btn-text">查看详情</text>
          </view>
        </view>
      </view>
      <text class="list-foot">任务由老师发布 · 下拉可刷新</text>
    </view>

    <!-- ===== 加载骨架：3 张同构卡 ===== -->
    <view v-if="isFirstLoading" class="skeleton-list">
      <view v-for="i in 3" :key="i" class="skeleton-card">
        <view class="sk-head">
          <view class="sk sk-icon" />
          <view class="sk sk-title" :class="{ 'sk-title--short': i === 2 }" />
        </view>
        <view class="sk sk-line" />
        <view class="sk sk-meta" />
        <view class="sk-status">
          <view class="sk sk-mark" />
          <view class="sk sk-status-text" />
        </view>
        <view class="sk-buttons">
          <view class="sk sk-button" />
          <view class="sk sk-button" />
        </view>
      </view>
    </view>

    <!-- ===== 空态 ===== -->
    <view v-if="showEmpty" class="state-card">
      <view class="state-orb">
        <text class="state-icon">{{ ICON.task }}</text>
      </view>
      <text class="state-title">{{ emptyTitle }}</text>
      <text class="state-desc">{{ emptyDesc }}</text>
      <view v-if="activeScope === 'active'" class="state-action" @click="viewAll">
        <text class="state-action-text">查看全部</text>
        <text class="state-action-glyph">{{ ICON.chevron }}</text>
      </view>
    </view>

    <!-- ===== 首次失败 / 未绑定 ===== -->
    <view v-if="showFullError" class="state-card state-card--tall">
      <view class="state-orb state-orb--danger">
        <text class="state-icon state-icon--danger">{{ ICON.alert }}</text>
      </view>
      <text class="state-title">{{ unbound ? "未绑定孩子信息" : "任务加载失败" }}</text>
      <text class="state-desc">{{ unbound ? "请先在首页绑定孩子，再查看亲子任务" : "网络开了小差，请稍后再试" }}</text>
      <view class="state-action" :class="{ 'is-busy': retrying }" @click="handleRetry">
        <text class="state-action-text">{{ retrying ? "重试中…" : "重新加载" }}</text>
        <text class="state-action-glyph">{{ ICON.refresh }}</text>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 任务节奏：白卡 + 冷雾底；暖色只落在班级墙图标、选中分段、
// 未提交空心点与主按钮。进度条已由「周期状态行」替代。
// ============================================================

// ---- 班级墙入口 ----------------------------------------------
.wall-entry {
  display: flex;
  align-items: center;
  gap: 20rpx;
  width: 100%;
  min-height: 128rpx;
  padding: 20rpx $spacing-lg;
  border: 1rpx solid $color-border-subtle;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;

  &:active {
    transform: scale(0.99);
  }
}

.wall-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 72rpx;
  height: 72rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.wall-icon-glyph {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.wall-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2rpx;
  min-width: 0;
}

.wall-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.wall-desc {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.45;
}

.wall-chevron {
  flex: none;
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

// ---- 三段分类 ------------------------------------------------
.segmented {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-xs;
  margin-top: 20rpx;
  padding: $spacing-xs;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.segment {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6rpx;
  min-width: 0;
  min-height: 88rpx;
  padding: 0 $spacing-sm;
  border-radius: $border-radius-full;

  &:active {
    transform: scale(0.97);
  }
}

.segment-text {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.segment.is-selected {
  background: $color-brand-primary-bg;
}

.segment.is-selected .segment-text {
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

// 红点数字用 danger 深底 + 白字，保证 11px 也过 AA。
.segment-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  min-width: 32rpx;
  height: 32rpx;
  padding: 0 6rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-strong;
}

.segment-badge-text {
  color: $color-text-on-brand;
  font-size: 20rpx;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

// ---- 待提交提醒 ----------------------------------------------
.pending-note {
  display: flex;
  align-items: center;
  gap: 10rpx;
  margin: 20rpx 4rpx 0;
}

.pending-dot {
  flex: none;
  width: 12rpx;
  height: 12rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-warning;
}

.pending-text {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.pending-count {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
  font-variant-numeric: tabular-nums;
}

// ---- 已有列表时刷新失败 --------------------------------------
.inline-error {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-height: 88rpx;
  margin-top: 20rpx;
  padding: $spacing-md $spacing-lg;
  border: 1rpx solid $color-semantic-danger-border;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;

  &:active {
    transform: scale(0.995);
  }
}

.inline-error-icon {
  flex: none;
  color: $color-semantic-danger-strong;
  font-family: remixicon;
  font-size: $typography-font-size-lg;
  line-height: 1;
}

.inline-error-text {
  flex: 1;
  min-width: 0;
  color: $color-semantic-danger-strong;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.45;
}

// ---- 任务卡 --------------------------------------------------
.task-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  margin-top: 20rpx;
}

.task-card {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 28rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;

  &.is-ended {
    background: $color-bg-page;
    box-shadow: none;
  }
}

.task-top {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  min-width: 0;
}

.task-icon {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 84rpx;
  height: 84rpx;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
}

.task-card.is-ended .task-icon {
  background: $color-border-subtle;
}

.task-icon-glyph {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: 44rpx;
  line-height: 1;
}

.task-card.is-ended .task-icon-glyph {
  color: $color-text-muted;
}

.task-icon-dot {
  position: absolute;
  top: 6rpx;
  right: 6rpx;
  width: 16rpx;
  height: 16rpx;
  border: 2rpx solid $color-bg-card;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
}

.task-heading {
  flex: 1;
  min-width: 0;
}

.task-titleline {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 6rpx $spacing-sm;
  min-width: 0;
}

.task-title {
  flex: 0 1 auto;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  overflow-wrap: break-word;
}

.task-badge {
  flex: none;
  display: inline-flex;
  align-items: center;
  min-height: 40rpx;
  padding: 0 $component-badge-padding-x;
  border-radius: $component-badge-radius;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.badge--todo {
  background: $color-brand-primary-bg;
  color: $color-text-primary;
}

.badge--done {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.badge--reply {
  background: $color-semantic-info-bg;
  color: $color-text-primary;
}

.badge--ended {
  background: $color-border-subtle;
  color: $color-text-body;
}

.task-desc {
  display: -webkit-box;
  margin-top: 6rpx;
  overflow: hidden;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.55;
  overflow-wrap: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

// ---- 周期事实 ------------------------------------------------
.task-period {
  display: flex;
  flex-direction: column;
  gap: 10rpx;
  margin-top: 20rpx;
  padding-top: 18rpx;
  border-top: 1rpx solid $color-border-subtle;
}

.task-meta-line {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 4rpx $spacing-md;
}

.task-meta {
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  overflow-wrap: break-word;
}

.task-due {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
}

.cycle-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-xs $spacing-md;
  min-width: 0;
}

.cycle-state {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  min-width: 0;
}

.cycle-label {
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.cycle--done .cycle-label {
  color: $color-semantic-success-strong;
}

.cycle--reply .cycle-label {
  color: $color-semantic-info;
}

.cycle--todo .cycle-label {
  color: $color-brand-primary-dark;
}

.cycle--ended .cycle-label {
  color: $color-text-muted;
}

.cycle-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 36rpx;
  height: 36rpx;
  border-radius: $border-radius-full;
}

.mark--done {
  background: $color-semantic-success-bg;
}

.mark--reply {
  background: $color-semantic-info-bg;
}

.mark--ended {
  background: $color-border-subtle;
}

.mark--todo {
  background: transparent;
}

.cycle-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-xs;
  line-height: 1;
}

.mark--done .cycle-glyph {
  color: $color-semantic-success-strong;
}

.mark--reply .cycle-glyph {
  color: $color-semantic-info;
}

.cycle-hollow {
  width: 18rpx;
  height: 18rpx;
  border: 3rpx solid $color-brand-primary;
  border-radius: $border-radius-full;
}

.cycle-dash {
  width: 18rpx;
  height: 3rpx;
  border-radius: $border-radius-full;
  background: $color-text-muted;
}

.star-jar {
  display: flex;
  align-items: center;
  gap: 6rpx;
  min-width: 0;
}

.star-glyph {
  flex: none;
  color: $color-semantic-warning;
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  line-height: 1;
}

.star-label {
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

// ---- 未读悄悄话 ----------------------------------------------
.unread-note {
  display: flex;
  align-items: flex-start;
  gap: 10rpx;
  margin-top: 18rpx;
  padding: 14rpx 18rpx;
  border-radius: $border-radius-lg;
  background: $color-brand-primary-bg;
}

.unread-dot {
  flex: none;
  width: 12rpx;
  height: 12rpx;
  margin-top: 10rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.unread-text {
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

// ---- 动作 ----------------------------------------------------
.task-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: $spacing-md;
  margin-top: 20rpx;
  padding-top: 20rpx;
  border-top: 1rpx solid $color-border-subtle;
}

.task-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  min-height: 88rpx;
  padding: 0 $spacing-md;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;

  &:active {
    transform: scale(0.98);
  }
}

// 白字压品牌橙不足 4.5:1，主按钮按设计稿改用深色字。
.task-btn--primary {
  background: $gradient-brand;
}

.task-btn--primary .task-btn-text {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

.task-btn--primary.is-disabled {
  background: $color-border-subtle;

  &:active {
    transform: none;
  }
}

.task-btn--primary.is-disabled .task-btn-text {
  color: $color-text-hint;
  font-weight: $typography-font-weight-medium;
}

.task-btn--secondary {
  border-color: $color-border-subtle;
  background: $color-bg-card;
}

.task-btn--secondary .task-btn-text {
  color: $color-text-body;
  font-weight: $typography-font-weight-medium;
}

.task-btn-text {
  font-size: $typography-font-size-base;
  line-height: 1.2;
  text-align: center;
}

.task-card.is-ended .task-btn--secondary {
  background: $color-bg-page;
}

.list-foot {
  padding-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  text-align: center;
}

// ---- 骨架 ----------------------------------------------------
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  margin-top: 20rpx;
}

.skeleton-card {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  padding: 28rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sk {
  display: block;
  height: 20rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  animation: sk-breathe 1.3s $animation-easing-default infinite alternate;
}

.sk-head {
  display: flex;
  align-items: center;
  gap: 20rpx;
}

.sk-icon {
  flex: none;
  width: 84rpx;
  height: 84rpx;
  border-radius: $border-radius-xl;
}

.sk-title {
  width: 46%;
  height: 28rpx;
}

.sk-title--short {
  width: 34%;
}

.sk-line {
  width: 86%;
  height: 18rpx;
}

.sk-meta {
  width: 60%;
  height: 18rpx;
}

.sk-status {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  padding-top: 18rpx;
  border-top: 1rpx solid $color-border-subtle;
}

.sk-mark {
  width: 36rpx;
  height: 36rpx;
}

.sk-status-text {
  width: 34%;
  height: 18rpx;
}

.sk-buttons {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: $spacing-md;
}

.sk-button {
  height: 88rpx;
  border-radius: $border-radius-full;
  animation: sk-breathe 1.3s $animation-easing-default infinite alternate;
}

// ---- 空态 / 失败态 -------------------------------------------
.state-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 20rpx;
  padding: 60rpx $spacing-lg 52rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.state-card--tall {
  justify-content: center;
  min-height: 46vh;
}

.state-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.state-orb--danger {
  background: $color-semantic-danger-bg;
}

.state-icon {
  color: $color-text-hint;
  font-family: remixicon;
  font-size: 52rpx;
  line-height: 1;
}

.state-icon--danger {
  color: $color-semantic-danger-strong;
}

.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.state-desc {
  max-width: 480rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.state-action {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6rpx;
  min-width: 208rpx;
  min-height: 88rpx;
  margin-top: $spacing-xl;
  padding: 0 40rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;

  &:active {
    transform: scale(0.98);
  }
}

.state-action.is-busy {
  opacity: 0.7;
}

.state-action-text {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
}

.state-action-glyph {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-base;
  line-height: 1;
}

@keyframes sk-breathe {
  0%,
  100% {
    opacity: 0.55;
  }
  50% {
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sk,
  .sk-button {
    animation: none;
  }

  .wall-entry,
  .segment,
  .inline-error,
  .task-btn,
  .state-action {
    transition: none;
  }
}
</style>
