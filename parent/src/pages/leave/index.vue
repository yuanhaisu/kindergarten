<script setup>
/**
 * 请假记录 · 一条请假一张进度卡
 *
 * 统计三格取接口 stats（全量），四个分段只在前端过滤已加载的 50 条：
 * 切换筛选不重新请求，数字也不会随筛选跳动。
 */
import { computed, ref } from "vue"
import { onLoad, onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchLeaves } from "@/api/parent-leave"
import { formatCommentTime, formatDateLabel, localDateKey } from "@/utils/time"
import { LEAVE_STATUS, leaveStatusLabel, leaveStatusTone, leaveTypeLabel } from "@/utils/leave"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  calendar: "", // calendar-line
  alert: "", // error-warning-line
  chevron: "" // arrow-right-s-line
}

const FILTERS = [
  { key: "all", label: "全部" },
  { key: "pending", label: "待审批", status: LEAVE_STATUS.PENDING },
  { key: "approved", label: "已同意", status: LEAVE_STATUS.APPROVED },
  { key: "rejected", label: "已拒绝", status: LEAVE_STATUS.REJECTED }
]

// 后端分页上限 50；请假单量远小于这个数，一次拉全量后前端筛选。
const PAGE_SIZE = 50

const leaves = ref([])
const stats = ref({ all: 0, pending: 0, approved: 0 })
const activeFilter = ref("all")
const loading = ref(false)
const retrying = ref(false)
const refreshing = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")

// 只允许最后一次请求写数据：下拉刷新与 onShow 回刷可能并发。
let loadSeq = 0

const isFirstLoading = computed(() => loading.value && !hasLoaded.value)
const showError = computed(() => Boolean(loadError.value) && !loading.value)
const pageLocked = computed(() => isFirstLoading.value || showError.value)
const hasList = computed(() => hasLoaded.value && !loadError.value && leaves.value.length > 0)
const showEmpty = computed(() => hasLoaded.value && !loadError.value && leaves.value.length === 0)

const filteredLeaves = computed(() => {
  const hit = FILTERS.find(f => f.key === activeFilter.value)
  if (!hit || hit.key === "all") return leaves.value
  return leaves.value.filter(item => Number(item.status) === hit.status)
})

// 筛选结果为空是页内分支，不能误判成「一条请假都没有」的整页空态。
const showFilterEmpty = computed(
  () => hasList.value && activeFilter.value !== "all" && filteredLeaves.value.length === 0
)

// 失败态没有可信数字，用「—」而不是 0，避免把「没拿到」读成「真的是 0」。
const statDisplay = computed(() => {
  if (loadError.value) return { all: "—", pending: "—", approved: "—" }
  return {
    all: String(stats.value.all),
    pending: String(stats.value.pending),
    approved: String(stats.value.approved)
  }
})

function avatarInitial(item) {
  const name = String(item?.student?.name || "")
  return name ? Array.from(name)[0] : "?"
}

function dateRangeText(item) {
  const start = formatDateLabel(item.start_date)
  const end = formatDateLabel(item.end_date)
  const days = Number(item.days) || 0
  const span = !end || start === end ? start : `${start} ~ ${end}`
  return [span, days > 0 ? `${days}天` : ""].filter(Boolean).join(" · ")
}

function approvalMeta(item) {
  const name = item?.reviewer?.nickname || ""
  const date = item?.reviewed_at ? formatDateLabel(localDateKey(item.reviewed_at)) : ""
  const who = [name, date].filter(Boolean).join(" ")
  return who ? `审批意见 · ${who}` : "审批意见"
}

function hasApproval(item) {
  const status = Number(item.status)
  if (status !== LEAVE_STATUS.APPROVED && status !== LEAVE_STATUS.REJECTED) return false
  return Boolean(item.reviewer?.nickname || item.reviewed_at || item.review_remark)
}

async function loadData(options = {}) {
  const keepView = Boolean(options.keepView)
  const seq = ++loadSeq
  if (!keepView) {
    loading.value = true
    loadError.value = ""
  }
  try {
    await ensureLogin()
    const res = await fetchLeaves({ page: 1, page_size: PAGE_SIZE })
    if (seq !== loadSeq) return
    const list = Array.isArray(res?.data?.list) ? res.data.list : []
    const rawStats = res?.data?.stats || {}
    leaves.value = list
    stats.value = {
      all: Number(rawStats.all) || 0,
      pending: Number(rawStats.pending) || 0,
      approved: Number(rawStats.approved) || 0
    }
    hasLoaded.value = true
    // 下拉刷新也能从失败态恢复：成功就清掉错误，不只靠重试按钮。
    loadError.value = ""
  } catch (err) {
    if (seq !== loadSeq) return
    console.warn("[leave-index] load failed:", err?.message || err)
    // 刷新失败时保留已加载内容；只有「一句都还没拿到」才切整页错误态。
    if (keepView && leaves.value.length > 0) return
    leaves.value = []
    stats.value = { all: 0, pending: 0, approved: 0 }
    loadError.value = "请假记录加载失败"
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      refreshing.value = false
      uni.stopPullDownRefresh()
    }
  }
}

function switchFilter(key) {
  if (pageLocked.value || activeFilter.value === key) return
  // 筛选只改视图，不重新请求，统计口径保持全量。
  activeFilter.value = key
}

function viewAll() {
  switchFilter("all")
}

async function handleRetry() {
  if (retrying.value || isFirstLoading.value) return
  retrying.value = true
  await loadData({ keepView: true })
  retrying.value = false
  uni.showToast({
    title: loadError.value ? "仍无法加载，请稍后再试" : "已重新加载",
    icon: "none"
  })
}

async function handleRefresh() {
  if (refreshing.value || isFirstLoading.value) {
    uni.stopPullDownRefresh()
    return
  }
  refreshing.value = true
  await loadData({ keepView: true })
}

function goCreate() {
  // 失败态没有可展示的列表，入口隐藏；加载中仍允许先发起请假（设计稿要求 FAB 常驻）。
  if (showError.value) return
  uni.navigateTo({ url: "/pages/leave/create" })
}

function goDetail(id) {
  if (pageLocked.value) return
  uni.navigateTo({ url: `/pages/leave/detail?id=${id}` })
}

onLoad(() => {
  loadData()
})
// 从创建 / 详情返回时刷新：撤销或新提交后列表必须立刻反映最新状态。
onShow(() => {
  if (hasLoaded.value && !loading.value && !retrying.value) loadData({ keepView: true })
})
onPullDownRefresh(handleRefresh)
</script>

<template>
  <PageShell bottom-space="200rpx" max-width="430px">
    <!-- ===== 统计三格：全量口径，不随分段变化 ===== -->
    <view class="stats-card" :class="{ 'is-error': Boolean(loadError) }">
      <view class="stat">
        <view v-if="isFirstLoading" class="stat-skeleton" />
        <text v-else class="stat-number" :class="{ 'is-dash': Boolean(loadError) }">{{ statDisplay.all }}</text>
        <text class="stat-label">全部请假</text>
      </view>
      <view class="stat">
        <view v-if="isFirstLoading" class="stat-skeleton" />
        <text
          v-else
          class="stat-number stat-number--accent"
          :class="{ 'is-dash': Boolean(loadError) }"
        >{{ statDisplay.pending }}</text>
        <text class="stat-label">待审批</text>
      </view>
      <view class="stat">
        <view v-if="isFirstLoading" class="stat-skeleton" />
        <text v-else class="stat-number" :class="{ 'is-dash': Boolean(loadError) }">{{ statDisplay.approved }}</text>
        <text class="stat-label">已同意</text>
      </view>
    </view>

    <!-- ===== 状态分段：加载 / 失败时整体禁用 ===== -->
    <view class="filter-row" :class="{ 'is-locked': pageLocked }">
      <view
        v-for="f in FILTERS"
        :key="f.key"
        class="filter-pill"
        :class="{ 'is-active': activeFilter === f.key }"
        @click="switchFilter(f.key)"
      >
        <text class="filter-text">{{ f.label }}</text>
      </view>
    </view>

    <!-- ===== 请假卡片流 ===== -->
    <view v-if="hasList && !showFilterEmpty" class="leave-list">
      <view
        v-for="item in filteredLeaves"
        :key="item.id"
        class="leave-card"
        :class="{ 'is-pending': Number(item.status) === 0 }"
        @click="goDetail(item.id)"
      >
        <view class="leave-avatar" :class="{ 'is-pending': Number(item.status) === 0 }">
          <text>{{ avatarInitial(item) }}</text>
        </view>
        <view class="leave-copy">
          <view class="leave-heading">
            <text class="leave-name">{{ item.student?.name || "孩子" }}</text>
            <text class="leave-badge" :class="'badge--' + leaveStatusTone(item.status)">{{ leaveStatusLabel(item.status, item.status_label) }}</text>
          </view>
          <view class="leave-meta">
            <text class="leave-type">{{ leaveTypeLabel(item) }}</text>
            <text class="leave-dates">{{ dateRangeText(item) }}</text>
          </view>
          <text class="leave-reason">{{ item.reason }}</text>
          <text v-if="Number(item.status) === 0" class="leave-submitted">提交于 {{ formatCommentTime(item.created_at) }}</text>
          <view
            v-else-if="hasApproval(item)"
            class="approval-note"
            :class="'approval-note--' + leaveStatusTone(item.status)"
          >
            <text class="approval-meta">{{ approvalMeta(item) }}</text>
            <text v-if="item.review_remark" class="approval-text">{{ item.review_remark }}</text>
          </view>
        </view>
        <text class="leave-chevron">{{ ICON.chevron }}</text>
      </view>
      <text class="leave-footer">共 {{ filteredLeaves.length }} 条请假记录</text>
    </view>

    <!-- ===== 筛选空态：不是整页空态 ===== -->
    <view v-if="showFilterEmpty" class="state-card">
      <text class="state-title">该状态下暂无请假记录</text>
      <view class="state-action state-action--ghost" @click="viewAll">
        <text>查看全部</text>
      </view>
    </view>

    <!-- ===== 整页空态 ===== -->
    <view v-if="showEmpty" class="state-card state-card--tall">
      <view class="state-orb">
        <text class="state-icon">{{ ICON.calendar }}</text>
      </view>
      <text class="state-title">暂无请假记录</text>
      <text class="state-desc">提交请假后，老师的审批结果会显示在这里</text>
      <view class="state-action" @click="goCreate">
        <text>发起请假</text>
      </view>
    </view>

    <!-- ===== 加载骨架 ===== -->
    <view v-if="isFirstLoading" class="skeleton-list">
      <view v-for="i in 3" :key="i" class="skeleton-card">
        <view class="sk sk-avatar" />
        <view class="sk-copy">
          <view class="sk sk-title" />
          <view class="sk sk-body" />
          <view class="sk sk-meta" />
        </view>
      </view>
    </view>

    <!-- ===== 失败态 ===== -->
    <view v-if="showError" class="state-card state-card--tall">
      <view class="state-orb state-orb--danger">
        <text class="state-icon">{{ ICON.alert }}</text>
      </view>
      <text class="state-title">请假记录加载失败</text>
      <text class="state-desc">网络不太稳定，请稍后再试</text>
      <view class="state-action" :class="{ 'is-busy': retrying }" @click="handleRetry">
        <text>{{ retrying ? "重试中…" : "重试" }}</text>
      </view>
    </view>

    <!-- ===== 右下新建入口：空态 / 失败态隐藏，加载中保留 ===== -->
    <view v-if="!showEmpty && !showError" class="float-create" @click="goCreate">
      <text class="fab-plus">＋</text>
      <text>请假</text>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 请假记录 · 白卡进度流；暖橙只落在待审批数字、选中分段与 FAB。
// ============================================================

// ---- 统计三格 ------------------------------------------------
.stats-card {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  padding: 28rpx 0;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

.stat {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  min-width: 0;
  min-height: 104rpx;
  padding: 0 $spacing-xs;
  text-align: center;

  // 发丝分隔只占 56rpx，不贯穿整格（设计稿 28px）。
  & + &::before {
    content: "";
    position: absolute;
    top: 50%;
    left: 0;
    width: 1rpx;
    height: 56rpx;
    background: $color-border-hairline;
    transform: translateY(-50%);
  }
}

.stat-number {
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

// 白底上 $color-brand-primary 只有 2.8:1，深一档才达到大字号 3:1（设计审查 P2-5）。
.stat-number--accent {
  color: $color-brand-primary-dark;
}

.stat-number.is-dash {
  color: $color-text-hint;
}

.stat-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.stat-skeleton {
  width: 56rpx;
  height: 40rpx;
  border-radius: $border-radius-sm;
  background: $color-bg-page;
  animation: sk-breathe 1.4s $animation-easing-default infinite alternate;
}

// ---- 状态分段 ------------------------------------------------
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
    transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.97);
  }
}

.filter-pill.is-active {
  border-color: $color-brand-primary;
  background: $color-brand-primary-dark;
}

.filter-row.is-locked {
  opacity: 0.5;
  pointer-events: none;
}

.filter-text {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.filter-pill.is-active .filter-text {
  color: $color-text-on-brand;
  font-weight: $typography-font-weight-semibold;
}

// ---- 请假卡片 -------------------------------------------------
.leave-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  margin-top: 28rpx;
}

.leave-card {
  display: grid;
  grid-template-columns: 80rpx minmax(0, 1fr) 32rpx;
  align-items: start;
  column-gap: $spacing-lg;
  padding: 28rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.99);
  }
}

.leave-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.leave-avatar.is-pending {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.leave-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
}

.leave-heading {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}

.leave-name {
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.leave-badge {
  flex: none;
  display: inline-flex;
  align-items: center;
  padding: 8rpx $component-badge-padding-x;
  border-radius: $component-badge-radius;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
  white-space: normal;
}

// 徽标文字用 -strong 档：原色在浅底上低于 AA（设计审查 P2-5）。
.badge--pending {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
}

.badge--approved {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.badge--rejected {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger-strong;
}

.badge--cancelled {
  background: $color-bg-page;
  color: $color-text-muted;
}

.leave-meta {
  min-width: 0;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-sm $spacing-md;
}

.leave-type {
  flex: none;
  display: inline-flex;
  align-items: center;
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.leave-dates {
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  overflow-wrap: break-word;
}

// 长事由保留两行上下文；批次规范禁止 nowrap 单行截断。
.leave-reason {
  min-width: 0;
  display: -webkit-box;
  overflow: hidden;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.leave-submitted {
  display: block;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.approval-note {
  display: block;
  margin-top: 4rpx;
  padding: $spacing-md 20rpx;
  border-radius: $border-radius-sm;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.approval-note--approved {
  background: $color-bg-page;
}

.approval-note--rejected {
  background: $color-semantic-danger-bg;
}

.approval-meta {
  display: block;
  margin-bottom: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

// 浅红底上 muted 只有 3.9:1，审批元信息改用正文色保证 AA。
.approval-note--rejected .approval-meta {
  color: $color-text-body;
}

.approval-text {
  display: block;
  overflow-wrap: break-word;
}

.leave-chevron {
  align-self: center;
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.leave-footer {
  padding-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.6;
  text-align: center;
}

// ---- 空态 / 筛选空态 / 失败态 ---------------------------------
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
  box-shadow: $shadow-md;

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

// ---- 加载骨架 -------------------------------------------------
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  margin-top: 28rpx;
}

.skeleton-card {
  display: grid;
  grid-template-columns: 80rpx minmax(0, 1fr);
  align-items: start;
  column-gap: $spacing-lg;
  padding: 28rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

.sk {
  background: $color-bg-page;
  animation: sk-breathe 1.4s $animation-easing-default infinite alternate;
}

.sk-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-full;
}

.sk-copy {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  min-width: 0;
  padding-top: 4rpx;
}

.sk-title,
.sk-body,
.sk-meta {
  border-radius: $border-radius-sm;
}

.sk-title {
  width: 42%;
  height: 28rpx;
}

.sk-body {
  width: 86%;
  height: 24rpx;
}

.sk-meta {
  width: 58%;
  height: 20rpx;
}

// ---- 右下新建入口 ---------------------------------------------
.float-create {
  position: fixed;
  right: 28rpx;
  bottom: calc(32rpx + env(safe-area-inset-bottom));
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  height: 96rpx;
  padding: 0 36rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  box-shadow: $shadow-lg;
  transition:
    transform $animation-duration-fast $animation-easing-default,
    box-shadow $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.98);
    box-shadow: $shadow-md;
  }
}

.fab-plus {
  font-size: $typography-font-size-xl;
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
  .stat-skeleton,
  .sk {
    animation: none;
  }

  .filter-pill,
  .leave-card,
  .state-action,
  .float-create {
    transition: none;
  }
}
</style>
