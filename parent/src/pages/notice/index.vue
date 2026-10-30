<script setup>
/**
 * 通知公告 · 安静的收件箱
 *
 * 三个数字先回答「有多少、几条没看、今天几条」；类型只负责筛选，
 * 颜色只负责标记未读 —— 不做整行染色，也不加左边框。
 * 统计口径是「当前可见列表」，不是接口 total：切到「通知」时前端会把
 * 作业/带物/健康/其他四条合并过滤，用 total 会与屏幕上条数对不上。
 */
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchNotices, markNoticeRead } from "@/api/parent-notice"
import { formatClock, formatDateLabel, localDateKey, todayKey } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形（App.vue 已全局注册字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  task: "", // task-line
  activity: "", // flag-line
  homework: "", // file-text-line
  bring: "", // briefcase-4-line
  health: "", // heart-pulse-line
  other: "", // information-line
  inbox: "", // inbox-line
  alert: "", // error-warning-line
  chevron: "" // arrow-right-s-line
}

const CATEGORY_ICON = {
  task: ICON.task,
  activity: ICON.activity,
  homework: ICON.homework,
  bring: ICON.bring,
  health: ICON.health,
  other: ICON.other
}

const TAB_ALL = "all"
const TABS = [
  { key: TAB_ALL, label: "全部" },
  { key: "task", label: "任务" },
  { key: "activity", label: "活动" },
  { key: "notice", label: "通知" }
]
// 契约里没有「通知」这个聚合分类：它是作业/带物/健康/其他四类的前端合并视图。
const NOTICE_CATEGORIES = ["homework", "bring", "health", "other"]

const notices = ref([])
const activeTab = ref(TAB_ALL)
const loading = ref(false)
const retrying = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")

// 只允许最后一次请求写数据：切分类与下拉刷新可能并发。
let loadSeq = 0

const pageLocked = computed(() => loading.value || Boolean(loadError.value))
const showSkeleton = computed(() => loading.value || (!hasLoaded.value && !loadError.value))
const showError = computed(() => Boolean(loadError.value) && !loading.value)
const isEmptyList = computed(
  () => hasLoaded.value && !showSkeleton.value && !showError.value && notices.value.length === 0
)
const showEmpty = computed(() => isEmptyList.value && activeTab.value === TAB_ALL)
const showFilterEmpty = computed(() => isEmptyList.value && activeTab.value !== TAB_ALL)
const showList = computed(
  () => hasLoaded.value && !showSkeleton.value && !showError.value && notices.value.length > 0
)

function noticeTime(ts) {
  const d = new Date(ts)
  if (Number.isNaN(d.getTime())) return ""
  const now = new Date()
  const key = localDateKey(d)
  if (key === localDateKey(now)) return `今天 ${formatClock(ts)}`
  const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1)
  if (key === localDateKey(yesterday)) return `昨天 ${formatClock(ts)}`
  return formatDateLabel(key) || formatClock(ts)
}

const statValues = computed(() => {
  const today = todayKey()
  const list = notices.value
  return {
    total: String(list.length),
    unread: String(list.filter(item => !item.is_read).length),
    today: String(list.filter(item => localDateKey(item.published_at) === today).length)
  }
})

// 失败态没有可信数字，用「—」而不是 0，避免把「没拿到」读成「真的是 0」。
const statDisplay = computed(() =>
  loadError.value ? { total: "—", unread: "—", today: "—" } : statValues.value
)

const filtersLocked = computed(() => pageLocked.value)

function categoryIcon(category) {
  return CATEGORY_ICON[category] || ICON.other
}

function categoryLabel(item) {
  // B13 起后端返回中文 category_label；缺失时（旧后端/回滚）退回中性文案。
  if (item.category_label) return item.category_label
  const hit = TABS.find(tab => tab.key === item.category)
  return hit ? hit.label : "通知"
}

function creatorName(item) {
  // 作者被删除时后端给空昵称；显示「园所通知」而不是空串。
  return item.creator?.nickname || "园所通知"
}

function previewOf(item) {
  if (item.content_preview) return item.content_preview
  const text = String(item.content || "").replace(/\s+/g, " ").trim()
  if (!text) return ""
  const chars = Array.from(text)
  return chars.length > 80 ? `${chars.slice(0, 80).join("")}…` : text
}

function metaLine(item) {
  return [categoryLabel(item), creatorName(item), noticeTime(item.published_at)]
    .filter(Boolean)
    .join(" · ")
}

function requestParams() {
  const params = { page: 1, page_size: 50 }
  // 「通知」聚合视图没有后端分类可传，拉全量后在前端合并过滤。
  if (activeTab.value !== TAB_ALL && activeTab.value !== "notice") {
    params.category = activeTab.value
  }
  return params
}

async function loadData(options = {}) {
  const silent = Boolean(options.silent)
  // keepView：重试/空态刷新时保留当前卡片与按钮文案（重试中…/刷新中…），
  // 不立刻切成骨架屏 —— 设计稿的重试是先禁用按钮，再回正常态。
  const keepView = silent || Boolean(options.keepView)
  if (silent && loading.value) {
    uni.stopPullDownRefresh()
    return
  }
  const seq = ++loadSeq
  if (!keepView) {
    loading.value = true
    loadError.value = ""
  }
  try {
    await ensureLogin()
    const res = await fetchNotices(requestParams())
    if (seq !== loadSeq) return
    const list = Array.isArray(res?.data?.list) ? res.data.list : []
    notices.value =
      activeTab.value === "notice"
        ? list.filter(item => NOTICE_CATEGORIES.includes(item.category))
        : list
    hasLoaded.value = true
    // 下拉刷新也能从失败态恢复：成功就清掉错误，不只靠重试按钮。
    loadError.value = ""
  } catch (err) {
    if (seq !== loadSeq) return
    console.warn("[notice-index] load failed:", err?.message || err)
    // 下拉刷新失败时保留已加载内容；http 层已 toast，这里只留排查线索。
    // 只有「一句都还没拿到」才落到下面的整页错误态。
    if (silent && notices.value.length > 0) {
      return
    }
    notices.value = []
    loadError.value = "通知加载失败"
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
  }
}

function selectTab(key) {
  if (pageLocked.value || activeTab.value === key) return
  activeTab.value = key
  notices.value = []
  loadData()
}

function viewAll() {
  selectTab(TAB_ALL)
}

async function handleRetry() {
  if (loading.value || retrying.value) return
  retrying.value = true
  await loadData({ keepView: true })
  retrying.value = false
  uni.showToast({
    title: loadError.value ? "仍无法加载，请稍后再试" : "已重新加载",
    icon: "none"
  })
}

async function handleRefresh() {
  if (loading.value || retrying.value) return
  retrying.value = true
  await loadData({ keepView: true })
  retrying.value = false
  uni.showToast({
    title: loadError.value ? "仍无法加载，请稍后再试" : "已重新加载",
    icon: "none"
  })
}

function openNotice(item) {
  if (pageLocked.value) return
  if (!item.is_read) {
    // 先本地置读让列表立刻安静下来；请求失败时回滚，避免把「没记上」显示成已读。
    notices.value = notices.value.map(n => (n.id === item.id ? { ...n, is_read: true } : n))
    uni.showToast({ title: "已读 · 打开通知详情", icon: "none" })
    markNoticeRead(item.id).catch(err => {
      console.warn("[notice-index] mark read failed:", err?.message || err)
      notices.value = notices.value.map(n => (n.id === item.id ? { ...n, is_read: false } : n))
    })
  } else {
    uni.showToast({ title: "已打开通知详情", icon: "none" })
  }
  uni.navigateTo({ url: `/pages/notice/detail?id=${item.id}` })
}

onLoad(() => {
  loadData()
})
onPullDownRefresh(() => {
  loadData({ silent: true })
})
</script>

<template>
  <PageShell bottom-space="48rpx" max-width="430px">
    <!-- ===== 统计条：当前可见列表的三个数字 ===== -->
    <view class="stats-card" :class="{ 'is-error': Boolean(loadError) }">
      <view class="stat">
        <view v-if="showSkeleton" class="stat-skeleton" />
        <text v-else class="stat-number">{{ statDisplay.total }}</text>
        <text class="stat-label">全部通知</text>
      </view>
      <view class="stat stat--unread">
        <view v-if="showSkeleton" class="stat-skeleton" />
        <text v-else class="stat-number">{{ statDisplay.unread }}</text>
        <text class="stat-label">未读</text>
      </view>
      <view class="stat">
        <view v-if="showSkeleton" class="stat-skeleton" />
        <text v-else class="stat-number">{{ statDisplay.today }}</text>
        <text class="stat-label">今日新增</text>
      </view>
    </view>

    <!-- ===== 分类胶囊 ===== -->
    <view class="filter-row">
      <view
        v-for="tab in TABS"
        :key="tab.key"
        class="filter-pill"
        :class="{ 'is-active': activeTab === tab.key, 'is-locked': filtersLocked }"
        @click="selectTab(tab.key)"
      >
        <text class="filter-text">{{ tab.label }}</text>
      </view>
    </view>

    <!-- ===== 通知卡片流 ===== -->
    <view v-if="showList" class="notice-list">
      <view v-for="item in notices" :key="item.id" class="notice-card" @click="openNotice(item)">
        <view class="notice-icon" :class="{ 'is-unread': !item.is_read }">
          <text class="notice-icon-glyph">{{ categoryIcon(item.category) }}</text>
        </view>
        <view class="notice-body">
          <view class="notice-heading">
            <text class="notice-title" :class="{ 'is-unread': !item.is_read }">{{ item.title }}</text>
            <view v-if="!item.is_read" class="unread-dot" />
          </view>
          <text class="notice-preview">{{ previewOf(item) }}</text>
          <text class="notice-meta">{{ metaLine(item) }}</text>
        </view>
        <text class="notice-arrow">{{ ICON.chevron }}</text>
      </view>
    </view>

    <!-- ===== 筛选空态 ===== -->
    <view v-if="showFilterEmpty" class="state-card">
      <view class="state-orb">
        <text class="state-icon">{{ ICON.inbox }}</text>
      </view>
      <text class="state-title">该分类暂无通知</text>
      <view class="state-action state-action--ghost" @click="viewAll">
        <text>查看全部</text>
      </view>
    </view>

    <!-- ===== 全局空态 ===== -->
    <view v-if="showEmpty" class="state-card state-card--tall">
      <view class="state-orb">
        <text class="state-icon">{{ ICON.inbox }}</text>
      </view>
      <text class="state-title">暂无通知</text>
      <text class="state-desc">老师发布通知后会出现在这里</text>
      <view class="state-action" :class="{ 'is-busy': retrying }" @click="handleRefresh">
        <text>{{ retrying ? "刷新中…" : "刷新" }}</text>
      </view>
    </view>

    <!-- ===== 加载骨架 ===== -->
    <view v-if="showSkeleton" class="skeleton-list">
      <view v-for="i in 3" :key="i" class="skeleton-card">
        <view class="sk sk-icon" />
        <view class="sk-body">
          <view class="sk sk-title" />
          <view class="sk sk-line" />
          <view class="sk sk-meta" />
        </view>
      </view>
    </view>

    <!-- ===== 失败态 ===== -->
    <view v-if="showError" class="state-card state-card--tall">
      <view class="state-orb state-orb--danger">
        <text class="state-icon">{{ ICON.alert }}</text>
      </view>
      <text class="state-title">通知加载失败</text>
      <text class="state-desc">下拉刷新或点击重试</text>
      <view class="state-action" :class="{ 'is-busy': retrying }" @click="handleRetry">
        <text>{{ retrying ? "重试中…" : "重试" }}</text>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 安静收件箱：白卡 + 冷灰底，暖橙只落在未读圆点、选中胶囊与主按钮。
// ============================================================

// ---- 统计条 ------------------------------------------------
.stats-card {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  padding: 26rpx 0;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6rpx;
  min-width: 0;
  padding: 0 $spacing-xs;
  text-align: center;

  & + & {
    border-left: 1rpx solid $color-border-hairline;
  }
}

.stat-number {
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

// 未读数字用深一档的品牌色：白底上 --accent 的对比度只有 2.8:1（审查 P2-2）。
.stat--unread .stat-number {
  color: $color-brand-primary-dark;
}

.stats-card.is-error .stat-number {
  color: $color-text-hint;
}

.stat-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.stat-skeleton {
  width: 72rpx;
  height: 40rpx;
  border-radius: $border-radius-sm;
  background: $color-bg-page;
  animation: sk-breathe 1.3s $animation-easing-default infinite alternate;
}

// ---- 分类胶囊 ----------------------------------------------
.filter-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: $spacing-md;
  margin-top: 28rpx;
}

.filter-pill {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  height: $component-input-height;
  padding: 0 6rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  transition:
    background-color $animation-duration-fast $animation-easing-default,
    border-color $animation-duration-fast $animation-easing-default,
    color $animation-duration-fast $animation-easing-default,
    transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.98);
  }
}

.filter-pill.is-active {
  border-color: $color-brand-primary;
  background: $color-brand-primary-dark;
}

.filter-pill.is-locked {
  opacity: 0.5;
  pointer-events: none;
}

.filter-text {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  letter-spacing: 0.02em;
}

.filter-pill.is-active .filter-text {
  color: $color-text-on-brand;
  font-weight: $typography-font-weight-semibold;
}

// ---- 通知卡片 ----------------------------------------------
.notice-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  margin-top: 28rpx;
}

.notice-card {
  display: grid;
  grid-template-columns: 80rpx minmax(0, 1fr) 32rpx;
  align-items: center;
  gap: $spacing-lg;
  padding: 28rpx 24rpx 28rpx 28rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
  transition:
    transform $animation-duration-fast $animation-easing-default,
    background-color $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.99);
    background: $color-bg-page;
  }
}

.notice-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  transition:
    background-color $animation-duration-fast $animation-easing-default,
    color $animation-duration-fast $animation-easing-default;
}

.notice-icon.is-unread {
  background: $color-brand-primary-bg;
}

.notice-icon-glyph {
  color: $color-text-muted;
  font-family: remixicon;
  font-size: 44rpx;
  line-height: 1;
}

.notice-icon.is-unread .notice-icon-glyph {
  color: $color-brand-primary-dark;
}

.notice-body {
  min-width: 0;
}

.notice-heading {
  display: flex;
  align-items: flex-start;
  gap: 6rpx;
  min-width: 0;
}

.notice-title {
  flex: 0 1 auto;
  min-width: 0;
  display: -webkit-box;
  overflow: hidden;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.45;
  overflow-wrap: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.notice-title.is-unread {
  font-weight: $typography-font-weight-semibold;
}

.unread-dot {
  flex: none;
  width: 14rpx;
  height: 14rpx;
  margin-top: 14rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.notice-preview {
  display: -webkit-box;
  margin-top: 4rpx;
  overflow: hidden;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.55;
  overflow-wrap: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.notice-meta {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  letter-spacing: 0.01em;
  overflow-wrap: break-word;
}

.notice-arrow {
  align-self: center;
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

// ---- 空态 / 筛选空态 / 失败态 ------------------------------
.state-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 28rpx;
  padding: 64rpx 40rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
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
  width: 88rpx;
  height: 88rpx;
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
  font-size: 44rpx;
  line-height: 1;
}

.state-orb--danger .state-icon {
  color: $color-semantic-danger;
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
  font-size: $typography-font-size-sm;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.state-action {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 208rpx;
  height: $component-input-height;
  margin-top: $spacing-xl;
  padding: 0 40rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  letter-spacing: 0.02em;
  box-shadow: $shadow-md;
  transition:
    transform $animation-duration-fast $animation-easing-default,
    box-shadow $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.98);
    box-shadow: $shadow-sm;
  }
}

.state-action.is-busy {
  opacity: 0.82;
}

.state-action--ghost {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  box-shadow: $shadow-sm;
}

// ---- 加载骨架 ----------------------------------------------
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  margin-top: 28rpx;
}

.skeleton-card {
  display: grid;
  grid-template-columns: 80rpx minmax(0, 1fr);
  align-items: center;
  gap: $spacing-lg;
  padding: 28rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

.sk {
  background: $color-bg-page;
  animation: sk-breathe 1.3s $animation-easing-default infinite alternate;
}

.sk-icon {
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-full;
}

.sk-body {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  min-width: 0;
}

.sk-title,
.sk-line,
.sk-meta {
  border-radius: $border-radius-sm;
}

.sk-title {
  width: 64%;
  height: 24rpx;
}

.sk-line {
  width: 92%;
  height: 20rpx;
}

.sk-meta {
  width: 34%;
  height: 16rpx;
}

@keyframes sk-breathe {
  from {
    background-color: $color-bg-page;
  }
  to {
    background-color: $color-border-hairline;
  }
}

@media (prefers-reduced-motion: reduce) {
  .stat-skeleton,
  .sk {
    animation: none;
    background: $color-border-subtle;
  }

  .filter-pill,
  .notice-card,
  .state-action {
    transition: none;
  }
}
</style>
