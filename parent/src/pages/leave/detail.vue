<script setup>
/**
 * 请假详情 · 一次审批的完整经过
 *
 * 状态头只给结论，提交/审批时间只由时间线承担一次；
 * 审批卡只放审批意见正文，避免同一段话在首屏出现两次。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { fetchLeaveDetail, cancelLeave } from "@/api/parent-leave"
import { formatCommentTime, formatMonthDayWeekday, inclusiveDays } from "@/utils/time"
import { LEAVE_STATUS, leaveStatusTitle, leaveStatusTone } from "@/utils/leave"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG。
const ICON = {
  check: "", // check-line
  close: "", // close-line
  time: "", // time-line
  undo: "", // arrow-go-back-line
  lock: "", // lock-line
  alert: "" // error-warning-line
}

const STATUS_ICON = {
  pending: ICON.time,
  approved: ICON.check,
  rejected: ICON.close,
  cancelled: ICON.undo
}

const leaveId = ref(0)
const detail = ref({})
const loading = ref(true)
const loadError = ref("")
const retrying = ref(false)
const cancelling = ref(false)

const status = computed(() => Number(detail.value.status))
const statusTone = computed(() => leaveStatusTone(status.value))
const statusTitle = computed(() => leaveStatusTitle(status.value, detail.value.status_label))
const statusIcon = computed(() => STATUS_ICON[statusTone.value] || ICON.time)
const statusMeta = computed(() => {
  if (status.value === LEAVE_STATUS.PENDING) return "申请已提交，正在等待老师处理"
  if (status.value === LEAVE_STATUS.CANCELLED) return "你已撤回这次请假申请"
  return ""
})

const reviewerName = computed(() => detail.value.reviewer?.nickname || "老师")
const submittedTime = computed(() => formatCommentTime(detail.value.created_at))
const reviewedTime = computed(() =>
  status.value === LEAVE_STATUS.APPROVED || status.value === LEAVE_STATUS.REJECTED
    ? formatCommentTime(detail.value.reviewed_at)
    : ""
)

// 审批人与时间只在这里出现一次。
const approvalNodeText = computed(() => {
  if (status.value === LEAVE_STATUS.APPROVED) return `${reviewerName.value}已同意`
  if (status.value === LEAVE_STATUS.REJECTED) return `${reviewerName.value}已拒绝`
  if (status.value === LEAVE_STATUS.PENDING) return `等待${reviewerName.value}审批`
  return "审批已终止"
})
const firstDotTone = computed(() =>
  status.value === LEAVE_STATUS.APPROVED ? "success" : "done"
)
const secondDotTone = computed(() => {
  if (status.value === LEAVE_STATUS.APPROVED) return "success"
  if (status.value === LEAVE_STATUS.REJECTED) return "danger"
  if (status.value === LEAVE_STATUS.PENDING) return "waiting"
  return "stopped"
})
const timelineDone = computed(() => status.value === LEAVE_STATUS.APPROVED)

const studentInitial = computed(() => {
  const name = String(detail.value.student?.name || "")
  return name ? Array.from(name)[0] : "?"
})

const dateRangeText = computed(() => {
  const start = formatMonthDayWeekday(detail.value.start_date)
  if (!start) return ""
  if (!detail.value.end_date || detail.value.start_date === detail.value.end_date) return start
  return `${start} → ${formatMonthDayWeekday(detail.value.end_date)}`
})

const daysValue = computed(() => {
  const fromApi = Number(detail.value.days) || 0
  if (fromApi > 0) return fromApi
  return inclusiveDays(detail.value.start_date, detail.value.end_date)
})
const daysText = computed(() => `${daysValue.value} 天`)

const contactName = computed(() => String(detail.value.emergency_contact?.name || ""))
const contactPhone = computed(() => String(detail.value.emergency_contact?.phone || ""))
const hasContact = computed(() => Boolean(contactName.value || contactPhone.value))

const approvalText = computed(() => detail.value.review_remark || "—")
const canRevoke = computed(() => status.value === LEAVE_STATUS.PENDING)

async function loadData(id, options = {}) {
  const keepView = Boolean(options.keepView)
  if (!keepView) {
    loading.value = true
    loadError.value = ""
  }
  try {
    const res = await fetchLeaveDetail(id)
    detail.value = res.data || {}
    loadError.value = ""
  } catch (err) {
    console.warn("[leave-detail] load failed:", err?.message || err)
    detail.value = {}
    loadError.value = "请假详情加载失败，请检查网络后重试。"
  } finally {
    if (!keepView) loading.value = false
  }
}

async function handleRetry() {
  if (retrying.value) return
  retrying.value = true
  await loadData(leaveId.value, { keepView: true })
  retrying.value = false
  if (!loadError.value) uni.showToast({ title: "请假详情已重新加载", icon: "none" })
}

function handleCancel() {
  if (cancelling.value || !canRevoke.value) return
  uni.showModal({
    title: "确认撤销",
    content: "确定要撤销此请假申请吗？",
    success: async res => {
      if (!res.confirm) return
      cancelling.value = true
      try {
        await cancelLeave(leaveId.value)
        // 只改本地状态：接口未返回 reviewer 快照，这里也不需要伪造审批信息。
        detail.value = { ...detail.value, status: LEAVE_STATUS.CANCELLED, status_label: "已取消" }
        uni.showToast({ title: "已撤销", icon: "success" })
      } catch (err) {
        // 失败保持待审批；http 层已把后端原因 toast 出来，按钮恢复可点。
        console.warn("[leave-detail] cancel failed:", err?.message || err)
      } finally {
        cancelling.value = false
      }
    }
  })
}

onLoad(opts => {
  const id = Number(opts?.id || 0)
  if (!id) {
    loading.value = false
    loadError.value = "请假详情加载失败，请检查网络后重试。"
    return
  }
  leaveId.value = id
  loadData(id)
})
</script>

<template>
  <PageShell bottom-space="48rpx" max-width="430px">
    <!-- ===== 加载骨架 ===== -->
    <view v-if="loading" class="skeleton-card">
      <view class="skeleton-row">
        <view class="skeleton-dot" />
        <view class="sk skeleton-line skeleton-line--wide" />
      </view>
      <view class="skeleton-row">
        <view class="sk skeleton-line skeleton-line--label" />
        <view class="sk skeleton-line" />
      </view>
      <view class="skeleton-row">
        <view class="sk skeleton-line skeleton-line--label" />
        <view class="sk skeleton-line skeleton-line--medium" />
      </view>
    </view>

    <!-- ===== 加载失败 ===== -->
    <view v-else-if="loadError" class="error-card">
      <view class="error-orb">
        <text class="error-icon">{{ ICON.alert }}</text>
      </view>
      <text class="error-text">{{ loadError }}</text>
      <view class="retry-btn" :class="{ 'is-busy': retrying }" @click="handleRetry">
        <text>{{ retrying ? "重试中…" : "重试" }}</text>
      </view>
    </view>

    <view v-else class="page-detail">
      <!-- ===== 状态头 + 审批时间线 ===== -->
      <view class="status-card">
        <view class="status-head">
          <view class="status-orb" :class="'orb--' + statusTone">
            <text class="status-orb-icon">{{ statusIcon }}</text>
          </view>
          <view class="status-copy">
            <text class="status-title">{{ statusTitle }}</text>
            <text v-if="statusMeta" class="status-meta">{{ statusMeta }}</text>
          </view>
        </view>

        <view class="timeline" :class="{ 'timeline--success': timelineDone }">
          <view class="timeline-step timeline-step--first">
            <view class="timeline-dot" :class="'dot--' + firstDotTone" />
            <view class="timeline-copy">
              <text class="timeline-title">提交申请</text>
              <text v-if="submittedTime" class="timeline-meta">{{ submittedTime }}</text>
            </view>
          </view>
          <view class="timeline-step timeline-step--last">
            <view class="timeline-dot" :class="'dot--' + secondDotTone" />
            <view class="timeline-copy">
              <text class="timeline-title" :class="{ 'is-muted': status === 0 || status === 3 }">{{ approvalNodeText }}<text v-if="reviewedTime" class="timeline-time"> · {{ reviewedTime }}</text></text>
            </view>
          </view>
        </view>
      </view>

      <!-- ===== 请假信息 ===== -->
      <view class="sheet-card">
        <text class="sheet-title">请假信息</text>
        <view class="info-list">
          <view class="info-row">
            <text class="info-label">学生</text>
            <view class="info-value student-line">
              <view class="mini-avatar">
                <text>{{ studentInitial }}</text>
              </view>
              <text class="student-name">{{ detail.student?.name || "孩子" }}</text>
              <text v-if="detail.student?.class_name" class="student-sep">·</text>
              <text v-if="detail.student?.class_name" class="student-class">{{ detail.student.class_name }}</text>
            </view>
          </view>
          <view class="info-row">
            <text class="info-label">日期</text>
            <view class="info-value"><text>{{ dateRangeText }}</text></view>
          </view>
          <view class="info-row">
            <text class="info-label">天数</text>
            <text class="info-value info-em">{{ daysText }}</text>
          </view>
          <view v-if="hasContact" class="info-row">
            <text class="info-label">接送人</text>
            <view class="info-value value-stack">
              <text v-if="contactName">{{ contactName }}</text>
              <text v-if="contactPhone" class="value-sub">{{ contactPhone }}</text>
            </view>
          </view>
          <view class="info-row">
            <text class="info-label">事由</text>
            <text class="info-value reason-text">{{ detail.reason || "" }}</text>
          </view>
        </view>
      </view>

      <!-- ===== 审批信息：只放意见正文，审批人/时间只出现一次 ===== -->
      <view class="sheet-card">
        <text class="sheet-title">审批信息</text>
        <view v-if="status === 0" class="approval-empty">
          <view class="empty-orb">
            <text class="empty-icon">{{ ICON.time }}</text>
          </view>
          <text class="empty-title">尚未审批</text>
        </view>
        <view v-else-if="status === 3" class="approval-empty">
          <view class="empty-orb">
            <text class="empty-icon">{{ ICON.undo }}</text>
          </view>
          <text class="empty-title">无审批记录</text>
        </view>
        <view v-else class="note" :class="{ 'note--danger': status === 2 }">
          <view class="note-label">
            <view class="note-dot" :class="{ 'note-dot--danger': status === 2 }" />
            <text>审批意见</text>
          </view>
          <text class="note-text">{{ approvalText }}</text>
        </view>
      </view>

      <!-- ===== 撤销：仅待审批 ===== -->
      <view v-if="canRevoke" class="revoke-block">
        <view class="revoke-btn" :class="{ 'is-busy': cancelling }" @click="handleCancel">
          <text class="revoke-icon">{{ ICON.undo }}</text>
          <text class="revoke-label">{{ cancelling ? "撤销中…" : "撤销申请" }}</text>
        </view>
        <text class="revoke-hint">撤销后审批将同时终止，且无法恢复</text>
      </view>

      <!-- ===== 终态说明 ===== -->
      <view v-else class="terminal-note">
        <text class="terminal-icon">{{ ICON.lock }}</text>
        <text class="terminal-text">已结束的申请不可撤销，如需再次请假请重新提交</text>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 请假详情 · 状态与经过在头部交代，下面是信息与意见。
// ============================================================

.page-detail {
  padding-top: $spacing-lg;
}

// ---- 状态头 ---------------------------------------------------
.status-card {
  padding: $spacing-xl;
  margin-bottom: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.status-head {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-width: 0;
}

.status-orb {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  box-shadow: $shadow-sm;
}

.orb--pending {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.orb--approved {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.orb--rejected {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger-strong;
}

.orb--cancelled {
  background: $color-bg-page;
  color: $color-text-muted;
  box-shadow: none;
}

.status-orb-icon {
  font-family: remixicon;
  font-size: 44rpx;
  line-height: 1;
}

.status-copy {
  flex: 1;
  min-width: 0;
}

.status-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.status-meta {
  display: block;
  margin-top: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

// ---- 审批时间线 -----------------------------------------------
.timeline {
  margin-top: $spacing-lg;
  padding-top: $spacing-lg;
  border-top: 1rpx solid $color-border-subtle;
}

.timeline-step {
  position: relative;
  padding-left: 48rpx;
  padding-bottom: $spacing-lg;
}

// 两个节点之间只连一条线；第一节点带 --first，避免依赖 :not() 选择器。
.timeline-step--first::after {
  content: "";
  position: absolute;
  top: 30rpx;
  bottom: 0;
  left: 11rpx;
  width: 2rpx;
  background: $color-border-subtle;
}

.timeline-step--last {
  padding-bottom: 0;
}

.timeline--success .timeline-step--first::after {
  background: $color-semantic-success;
}

.timeline-dot {
  position: absolute;
  top: 10rpx;
  left: 0;
  z-index: 1;
  width: 24rpx;
  height: 24rpx;
  border: 4rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
}

.dot--done {
  border-color: $color-text-body;
  background: $color-text-body;
  box-shadow: 0 0 0 6rpx $color-bg-page;
}

.dot--success {
  border-color: $color-semantic-success;
  background: $color-semantic-success;
  box-shadow: 0 0 0 6rpx $color-semantic-success-bg;
}

.dot--waiting {
  border-color: $color-brand-primary;
  background: $color-bg-card;
  box-shadow: 0 0 0 6rpx $color-brand-primary-bg;
}

.dot--danger {
  border-color: $color-semantic-danger;
  background: $color-semantic-danger;
  box-shadow: 0 0 0 6rpx $color-semantic-danger-bg;
}

.dot--stopped {
  border-color: $color-text-muted;
  background: $color-bg-card;
  box-shadow: 0 0 0 6rpx $color-bg-page;
}

.timeline-copy {
  min-width: 0;
}

.timeline-title {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.timeline-title.is-muted {
  color: $color-text-muted;
}

.timeline-time {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-normal;
}

.timeline-meta {
  display: block;
  margin-top: 2rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

// ---- 信息卡 ---------------------------------------------------
.sheet-card {
  padding: $spacing-xl;
  margin-bottom: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.sheet-title {
  display: block;
  margin-bottom: $spacing-lg;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.info-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.info-row {
  display: grid;
  grid-template-columns: 96rpx minmax(0, 1fr);
  column-gap: $spacing-lg;
  align-items: start;
  min-width: 0;
}

.info-label {
  padding-top: 2rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.6;
}

.info-value {
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.student-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4rpx 10rpx;
}

.mini-avatar {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  margin-right: 4rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.student-name {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.6;
}

.student-sep {
  color: $color-text-hint;
}

.student-class {
  display: inline-block;
  padding: 0 $component-badge-padding-x;
  border-radius: $component-badge-radius;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.8;
}

.info-em {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

.value-stack {
  display: flex;
  flex-direction: column;
  gap: 2rpx;
}

.value-sub {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-variant-numeric: tabular-nums;
}

.reason-text {
  color: $color-text-primary;
}

// ---- 审批信息 -------------------------------------------------
.approval-empty {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.empty-orb {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 60rpx;
  height: 60rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-muted;
  box-shadow: $shadow-sm;
}

.empty-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.empty-title {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.note {
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.note--danger {
  background: $color-semantic-danger-bg;
}

.note-label {
  display: flex;
  align-items: center;
  gap: 12rpx;
  margin-bottom: 8rpx;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.note-dot {
  width: 10rpx;
  height: 10rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-success;
}

.note-dot--danger {
  background: $color-semantic-danger;
}

.note-text {
  display: block;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

// ---- 撤销与终态说明 -------------------------------------------
.revoke-block {
  margin-top: $spacing-xs;
}

.revoke-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  width: 100%;
  min-height: $component-input-height;
  padding: 20rpx $spacing-xl;
  border: 2rpx solid $color-semantic-danger;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.985);
  }
}

.revoke-btn.is-busy {
  opacity: 0.7;
}

.revoke-icon {
  color: $color-semantic-danger-strong;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.revoke-hint {
  display: block;
  margin-top: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  text-align: center;
}

.terminal-note {
  display: flex;
  align-items: flex-start;
  gap: 12rpx;
  margin: 0 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.terminal-icon {
  flex: none;
  margin-top: 2rpx;
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-lg;
  line-height: 1;
}

.terminal-text {
  min-width: 0;
}

// ---- 加载骨架 -------------------------------------------------
.skeleton-card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.skeleton-row {
  display: grid;
  grid-template-columns: 64rpx minmax(0, 1fr);
  gap: $spacing-lg;
  align-items: center;
  min-height: 48rpx;

  & + & {
    margin-top: $spacing-xl;
  }
}

.skeleton-dot,
.skeleton-line {
  display: block;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  animation: skeleton-pulse 1.5s $animation-easing-default infinite;
}

.skeleton-dot {
  width: 64rpx;
  height: 64rpx;
}

.skeleton-line {
  width: 100%;
  height: 20rpx;
}

.skeleton-line--label {
  width: 92rpx;
}

.skeleton-line--wide {
  width: 78%;
}

.skeleton-line--medium {
  width: 58%;
}

// ---- 加载失败 -------------------------------------------------
.error-card {
  padding: 64rpx $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
  text-align: center;
}

.error-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 96rpx;
  height: 96rpx;
  margin: 0 auto;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.error-icon {
  color: $color-text-muted;
  font-family: remixicon;
  font-size: 48rpx;
  line-height: 1;
}

.error-text {
  display: block;
  max-width: 460rpx;
  margin: $spacing-lg auto $spacing-xl;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.6;
}

.retry-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 224rpx;
  min-height: $component-input-height;
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.97);
  }
}

.retry-btn.is-busy {
  opacity: 0.55;
  pointer-events: none;
}

@keyframes skeleton-pulse {
  0%,
  100% {
    opacity: 0.32;
  }
  50% {
    opacity: 0.62;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton-dot,
  .skeleton-line {
    animation: none;
  }

  .revoke-btn,
  .retry-btn {
    transition: none;
  }
}
</style>
