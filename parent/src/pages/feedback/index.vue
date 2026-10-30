<script setup>
/**
 * 意见反馈 · 一张便笺，回声可见
 *
 * 上半页是安静的填写便笺，下半页是同一条时间轴上的历史回声。
 * 两条错误通道严格分开：列表失败只由列表请求改写，提交失败只由提交请求改写，
 * 任一重试都不会清掉另一条的失败态（设计审查 P1-1）。
 * 提交成功给整页确认，而不是一闪而过的 toast。
 */
import { computed, nextTick, ref, watch } from "vue"
import { onLoad, onUnload } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchFeedbackList, submitFeedback } from "@/api/parent-feedback"
import { formatMonthDayWeekday, localDateKey } from "@/utils/time"
import {
  FEEDBACK_CATEGORIES,
  feedbackCategory,
  feedbackStatus
} from "@/utils/points"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形（App.vue 已全局注册字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  suggestion: "\u{ee59}", // information-line
  complaint: "\u{ed3b}", // flag-line
  bug: "\u{eca1}", // error-warning-line
  check: "\u{eb7b}", // check-line
  alert: "\u{eca1}", // error-warning-line
  inbox: "\u{ee4f}", // inbox-line
  chevron: "\u{ea6e}" // arrow-right-s-line（旋转 90° 作向下箭头）
}

const CATEGORY_ICONS = {
  suggestion: ICON.suggestion,
  complaint: ICON.complaint,
  bug: ICON.bug
}

const MAX_LENGTH = 500
const HIGHLIGHT_DURATION = 2400
// 三行截断约 72 个汉字；小程序量不到真实行数，用长度阈值决定是否给展开箭头。
const EXPAND_THRESHOLD = 72

const content = ref("")
const currentCategory = ref("suggestion")
const focusContent = ref(false)
const listState = ref("loading") // loading / ok / empty / failed
// 真实表单初始为空 = 必填未完成；设计稿的 normal 态带示例文案，不能照搬。
const submitState = ref("required") // idle / required / submitting / failed
const feedbacks = ref([])
const listTotal = ref(0)
const showSuccess = ref(false)
const successDraft = ref(null)
const lastSubmitted = ref(null)
const highlightId = ref("")
const expanded = ref({})

let listSeq = 0
let highlightTimer = null

const submitting = computed(() => submitState.value === "submitting")
const isRequired = computed(() => submitState.value === "required")
const submitFailed = computed(() => submitState.value === "failed")
const contentLength = computed(() => content.value.length)

// 计数位只表达数量；加载/失败交给骨架与错误面板（设计审查 P2-3）。
const countLabel = computed(() => {
  if (listState.value !== "ok" && listState.value !== "empty") return ""
  return `共 ${listState.value === "empty" ? 0 : listTotal.value} 条`
})

// 固定底栏高度：失败时多一条 64rpx 的失败条，内容区要同步让位。
const barSpace = computed(() => {
  if (showSuccess.value) return "48rpx"
  return submitFailed.value ? "400rpx" : "320rpx"
})

const successDateLabel = computed(() => {
  const key = localDateKey(successDraft.value?.created_at)
  return key ? formatMonthDayWeekday(key) : ""
})

function categoryOf(item) {
  return feedbackCategory(item?.category, item?.category_label)
}

function statusOf(item) {
  return feedbackStatus(item?.status, item?.status_label)
}

function recordDate(item) {
  const key = localDateKey(item?.created_at)
  return key ? formatMonthDayWeekday(key) : ""
}

function isExpandable(item) {
  return String(item?.content || "").length > EXPAND_THRESHOLD
}

function isExpanded(item) {
  return Boolean(expanded.value?.[item?.id])
}

function toggleRecord(item) {
  if (!isExpandable(item)) return
  expanded.value = { ...expanded.value, [item.id]: !expanded.value[item.id] }
}

function selectType(key) {
  if (submitting.value) return
  currentCategory.value = key
}

function onContentInput(event) {
  content.value = event?.detail?.value ?? ""
}

function focusComposer() {
  uni.pageScrollTo({ scrollTop: 0, duration: 200 })
  nextTick(() => {
    focusContent.value = true
  })
}

async function loadList(options = {}) {
  const seq = ++listSeq
  listState.value = "loading"
  try {
    await ensureLogin()
    const res = await fetchFeedbackList({ page: 1, page_size: 50 })
    if (seq !== listSeq) return
    const list = Array.isArray(res?.data?.list) ? res.data.list : []
    feedbacks.value = list
    const total = Number(res?.data?.total)
    listTotal.value = Number.isFinite(total) ? total : list.length
    listState.value = list.length ? "ok" : "empty"
    if (options.retry) {
      uni.showToast({ title: "已刷新", icon: "none" })
    }
  } catch (err) {
    if (seq !== listSeq) return
    console.warn("[feedback-index] list failed:", err?.message || err)
    listState.value = "failed"
  }
}

function retryList() {
  if (listState.value === "failed") loadList({ retry: true })
}

function retrySubmit() {
  if (!content.value.trim()) {
    submitState.value = "required"
    focusComposer()
    return
  }
  doSubmit()
}

function handleSubmit() {
  if (submitting.value) return
  if (!content.value.trim()) {
    submitState.value = "required"
    focusComposer()
    return
  }
  doSubmit()
}

async function doSubmit() {
  const text = content.value.trim()
  if (!text) {
    submitState.value = "required"
    focusComposer()
    return
  }
  submitState.value = "submitting"
  try {
    await ensureLogin()
    const res = await submitFeedback({ category: currentCategory.value, content: text })
    const created = res?.data || {}
    const draft = {
      id: created.id || `local-${Date.now()}`,
      category: currentCategory.value,
      category_label: created.category_label || feedbackCategory(currentCategory.value).label,
      content: text,
      status: Number.isFinite(Number(created.status)) ? Number(created.status) : 0,
      status_label: created.status_label || "",
      reply: "",
      replied_at: null,
      created_at: created.created_at || new Date().toISOString()
    }
    lastSubmitted.value = draft
    successDraft.value = draft
    // 只有列表已可信（ok / empty）时才本地插入；列表失败时保持失败，
    // 等它自己的重试把服务端记录拉回来，不假装已加载。
    if (listState.value === "ok" || listState.value === "empty") {
      feedbacks.value = [draft, ...feedbacks.value]
      listTotal.value += 1
      listState.value = "ok"
    }
    content.value = ""
    submitState.value = "required"
    showSuccess.value = true
    highlightId.value = ""
    uni.pageScrollTo({ scrollTop: 0, duration: 0 })
  } catch (err) {
    console.warn("[feedback-index] submit failed:", err?.message || err)
    submitState.value = "failed"
  }
}

function viewRecords() {
  showSuccess.value = false
  const target = lastSubmitted.value
  nextTick(() => {
    uni.pageScrollTo({ selector: ".records", duration: 200 })
    if (!target) return
    highlightId.value = target.id
    if (highlightTimer) clearTimeout(highlightTimer)
    highlightTimer = setTimeout(() => {
      highlightId.value = ""
      highlightTimer = null
    }, HIGHLIGHT_DURATION)
  })
}

function writeAnother() {
  showSuccess.value = false
  content.value = ""
  submitState.value = "required"
  nextTick(() => {
    uni.pageScrollTo({ scrollTop: 0, duration: 0 })
    focusContent.value = true
  })
}

watch(content, (value) => {
  const empty = !String(value || "").trim()
  if (!empty && submitState.value === "required") {
    submitState.value = "idle"
  } else if (empty && submitState.value === "idle") {
    submitState.value = "required"
  }
})

onLoad(() => {
  loadList()
})

onUnload(() => {
  if (highlightTimer) clearTimeout(highlightTimer)
  highlightTimer = null
})
</script>

<template>
  <PageShell :bottom-space="barSpace" max-width="430px">
    <template v-if="!showSuccess">
      <!-- ===== 反馈便笺 ===== -->
      <view class="note-card" :class="{ 'is-locked': submitting }">
        <view class="field">
          <view class="field-head">
            <text class="field-label">反馈类型</text>
            <text class="field-tag">必选</text>
          </view>
          <view class="type-grid">
            <view
              v-for="item in FEEDBACK_CATEGORIES"
              :key="item.key"
              class="type-pill"
              :class="{ 'is-active': currentCategory === item.key }"
              @click="selectType(item.key)"
            >
              <text class="type-glyph">{{ CATEGORY_ICONS[item.key] }}</text>
              <text class="type-text">{{ item.label }}</text>
              <text class="type-check">{{ ICON.check }}</text>
            </view>
          </view>
        </view>

        <view class="field">
          <view class="field-head">
            <text class="field-label">反馈内容</text>
            <text class="field-tag">必填</text>
          </view>
          <view class="textarea-wrap" :class="{ 'is-invalid': isRequired }">
            <textarea
              class="feedback-textarea"
              :value="content"
              :maxlength="MAX_LENGTH"
              :disabled="submitting"
              :focus="focusContent"
              placeholder="请描述您的意见或问题..."
              placeholder-class="textarea-placeholder"
              @input="onContentInput"
              @blur="focusContent = false"
            />
            <text class="counter">{{ contentLength }} / {{ MAX_LENGTH }}</text>
          </view>
          <text class="field-hint">写明时间、地点或页面，园所处理更快</text>
        </view>
      </view>

      <!-- ===== 我的反馈 ===== -->
      <view class="records">
        <view class="records-head">
          <text class="section-title">我的反馈</text>
          <text v-if="countLabel" class="records-count">{{ countLabel }}</text>
        </view>

        <!-- 加载：两段时间轴骨架，表单仍可用 -->
        <view v-if="listState === 'loading'" class="records-skeleton">
          <view v-for="i in 2" :key="i" class="skeleton-item">
            <view class="skeleton-dot" />
            <view class="skeleton-card">
              <view class="skeleton-meta">
                <view class="skeleton-pill" />
                <view class="skeleton-date" />
              </view>
              <view class="skeleton-line" />
              <view class="skeleton-line skeleton-line--short" />
            </view>
          </view>
        </view>

        <!-- 列表失败：独立错误面板 -->
        <view v-else-if="listState === 'failed'" class="records-state">
          <view class="records-icon records-icon--danger">
            <text class="records-icon-glyph">{{ ICON.alert }}</text>
          </view>
          <text class="state-title">反馈记录加载失败</text>
          <text class="state-desc">请稍后重试，或检查网络连接</text>
          <view class="state-btn" @click="retryList">
            <text class="state-btn-text">重试</text>
          </view>
        </view>

        <!-- 空态 -->
        <view v-else-if="listState === 'empty'" class="records-state">
          <view class="records-icon">
            <text class="records-icon-glyph">{{ ICON.inbox }}</text>
          </view>
          <text class="state-title">还没有反馈记录</text>
          <text class="state-desc">提交后可以在这里查看园所回复</text>
          <view class="state-btn" @click="focusComposer">
            <text class="state-btn-text">写下第一条</text>
          </view>
        </view>

        <!-- 时间轴 -->
        <view v-else class="timeline">
          <view
            v-for="(item, index) in feedbacks"
            :key="item.id"
            class="record"
            :class="{ 'is-new': item.id === highlightId }"
          >
            <view class="record-dot" :class="'record-dot--' + statusOf(item).tone">
              <view class="dot-core" />
            </view>
            <view v-if="index < feedbacks.length - 1" class="record-line" />
            <view
              class="record-card"
              :class="{ 'is-open': isExpanded(item), 'is-tappable': isExpandable(item) }"
              @click="toggleRecord(item)"
            >
              <view class="record-meta">
                <text class="badge" :class="'badge--' + categoryOf(item).tone">{{ categoryOf(item).label }}</text>
                <view class="record-status">
                  <view class="status-dot" :class="'status-dot--' + statusOf(item).tone" />
                  <text class="record-status-text">{{ statusOf(item).label }}</text>
                </view>
                <text class="record-date">{{ recordDate(item) }}</text>
                <view v-if="isExpandable(item)" class="record-chevron" :class="{ 'is-open': isExpanded(item) }">
                  <text class="record-chevron-glyph">{{ ICON.chevron }}</text>
                </view>
              </view>
              <text class="record-text" :class="{ 'is-open': isExpanded(item) }">{{ item.content }}</text>
              <view v-if="item.reply" class="record-reply">
                <text class="record-reply-title">园所回复</text>
                <text class="record-reply-text">{{ item.reply }}</text>
              </view>
            </view>
          </view>
        </view>
      </view>
    </template>

    <!-- ===== 提交成功：整页确认 ===== -->
    <view v-else class="success-view">
      <view class="success-mark">
        <view class="success-ring success-ring--outer" />
        <view class="success-ring success-ring--mid" />
        <view class="success-ring success-ring--inner">
          <text class="success-check">{{ ICON.check }}</text>
        </view>
      </view>
      <text class="success-title">反馈已提交</text>
      <text class="success-desc">园所的回复会出现在「我的反馈」里</text>
      <view v-if="successDraft" class="success-card">
        <view class="success-card-head">
          <text class="badge" :class="'badge--' + categoryOf(successDraft).tone">
            {{ categoryOf(successDraft).label }}
          </text>
          <text class="success-card-date">{{ successDateLabel }}</text>
        </view>
        <text class="success-card-content">{{ successDraft.content }}</text>
      </view>
      <view class="success-actions">
        <view class="success-btn success-btn--primary" @click="viewRecords">
          <text class="success-btn-text">查看我的反馈</text>
        </view>
        <view class="success-btn success-btn--secondary" @click="writeAnother">
          <text class="success-btn-text success-btn-text--quiet">再提一条</text>
        </view>
      </view>
    </view>

    <!-- ===== 吸底提交栏 ===== -->
    <template #bottom>
      <view v-if="!showSuccess" class="submit-bar">
        <view v-if="submitFailed" class="bar-error">
          <text class="bar-error-glyph">{{ ICON.alert }}</text>
          <text class="bar-error-text">提交失败，请重试</text>
          <view class="bar-error-retry" @click="retrySubmit">
            <text class="bar-error-retry-text">重试</text>
          </view>
        </view>
        <view v-if="!submitFailed" class="bar-hint">
          <text v-if="isRequired" class="hint-required">
            <text class="hint-required-glyph">{{ ICON.alert }}</text>
            还需填写：反馈内容
          </text>
          <text v-else-if="submitting" class="hint-submitting">正在提交，请稍候</text>
          <text v-else class="hint-default">提交后可在记录里查看进展</text>
        </view>
        <view
          class="submit-btn"
          :class="{ 'is-disabled': isRequired, 'is-busy': submitting }"
          @click="handleSubmit"
        >
          <view v-if="submitting" class="btn-spinner" />
          <text class="submit-btn-text">{{ submitting ? "提交中…" : "提交" }}</text>
        </view>
      </view>
    </template>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 一张便笺，回声可见：白卡 + 冷雾底，暖橙只落在选中类型、
// 时间轴状态点与主按钮；失败/阻断用 danger 深色阶保证对比度。
// ============================================================

// ---- 便笺 ---------------------------------------------------
.note-card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
  transition: opacity $animation-duration-base $animation-easing-default;
}

// 提交中整张便笺不可交互，只留下「正在提交」这一条事实。
.note-card.is-locked {
  opacity: 0.58;
  pointer-events: none;
}

.field + .field {
  margin-top: $spacing-xl;
}

.field-head {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
}

.field-label {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.field-tag {
  display: inline-flex;
  align-items: center;
  height: 38rpx;
  padding: 0 14rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

// ---- 类型三选 -----------------------------------------------
.type-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-md;
}

.type-pill {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  min-width: 0;
  min-height: $component-input-height;
  padding: 0 $spacing-sm;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;

  &:active {
    transform: scale(0.96);
  }
}

.type-pill.is-active {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
}

.type-glyph {
  flex: none;
  color: $color-text-muted;
  font-family: remixicon;
  font-size: $typography-font-size-base;
  line-height: 1;
}

.type-pill.is-active .type-glyph {
  color: $color-brand-primary-dark;
}

.type-text {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.type-pill.is-active .type-text {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

.type-check {
  flex: none;
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-xs;
  line-height: 1;
  opacity: 0;
}

.type-pill.is-active .type-check {
  opacity: 1;
}

// ---- 输入 ---------------------------------------------------
.textarea-wrap {
  position: relative;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  transition: border-color $animation-duration-fast $animation-easing-default;
}

.textarea-wrap.is-invalid {
  border-color: $color-semantic-danger;
}

.feedback-textarea {
  display: block;
  width: 100%;
  min-height: 264rpx;
  padding: $spacing-lg $spacing-lg 56rpx;
  background: transparent;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.7;
}

.textarea-placeholder {
  color: $color-text-hint;
}

.counter {
  position: absolute;
  right: 22rpx;
  bottom: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.field-hint {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

// ---- 我的反馈 -----------------------------------------------
.records {
  margin-top: $spacing-xl;
}

.records-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
}

.section-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.records-count {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

// ---- 时间轴 -------------------------------------------------
.record {
  position: relative;
  padding-left: 40rpx;
  padding-bottom: $spacing-lg;

  &:last-child {
    padding-bottom: 0;
  }
}

.record-dot {
  position: absolute;
  left: 0;
  top: 50rpx;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30rpx;
  height: 30rpx;
  border-radius: $border-radius-full;
}

.record-dot--replied {
  background: $color-brand-primary-bg;
}

.dot-core {
  width: 14rpx;
  height: 14rpx;
  border-radius: $border-radius-full;
  background: $color-text-hint;
}

.record-dot--replied .dot-core {
  background: $color-brand-primary;
}

.record-dot--submitted .dot-core {
  border: 3rpx solid $color-brand-primary;
  background: $color-bg-card;
}

.record-dot--closed .dot-core {
  background: $color-text-hint;
}

.record-line {
  position: absolute;
  left: 14rpx;
  top: 90rpx;
  bottom: 0;
  width: 2rpx;
  background: $color-border-subtle;
}

.record-card {
  position: relative;
  padding: 20rpx $spacing-lg 28rpx;
  border: 1rpx solid transparent;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  transition: border-color $animation-duration-base $animation-easing-default;
}

.record-card.is-tappable:active {
  transform: scale(0.995);
}

// 新记录置顶高亮 2.4s（viewRecords 里启动计时）
.record.is-new .record-card {
  border-color: $color-brand-primary;
}

.record-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-sm $spacing-md;
  min-width: 0;
  min-height: 88rpx;
}

.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 40rpx;
  padding: 0 $component-badge-padding-x;
  border-radius: $component-badge-radius;
  color: $color-text-primary;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.badge--info {
  background: $color-semantic-info-bg;
}

.badge--warning {
  background: $color-semantic-warning-bg;
}

.badge--danger {
  background: $color-semantic-danger-bg;
}

.record-status {
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
}

.record-status-text {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.status-dot {
  flex: none;
  width: 10rpx;
  height: 10rpx;
  border-radius: $border-radius-full;
}

.status-dot--submitted {
  background: $color-brand-primary;
}

.status-dot--replied {
  background: $color-semantic-success;
}

.status-dot--closed {
  background: $color-text-hint;
}

.record-date {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.record-chevron {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  margin: -26rpx -8rpx -26rpx auto;
  transform: rotate(90deg);
  transition: transform $animation-duration-base $animation-easing-default;
}

.record-chevron-glyph {
  color: $color-text-muted;
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

.record-chevron.is-open {
  transform: rotate(270deg);
}

.record-text {
  display: -webkit-box;
  margin-top: 4rpx;
  overflow: hidden;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.7;
  white-space: pre-wrap;
  overflow-wrap: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.record-text.is-open {
  display: block;
  overflow: visible;
}

.record-reply {
  margin-top: $spacing-lg;
  padding: 20rpx $spacing-lg;
  border-left: 4rpx solid $color-brand-primary-light;
  border-radius: $border-radius-lg;
  background: $color-brand-primary-bg;
}

.record-reply-title {
  display: block;
  margin-bottom: 4rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.record-reply-text {
  display: block;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.7;
  white-space: pre-wrap;
  overflow-wrap: break-word;
}

// ---- 空态 / 列表错误 ----------------------------------------
.records-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48rpx $spacing-xl 40rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.records-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 128rpx;
  height: 128rpx;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.records-icon--danger {
  background: $color-semantic-danger-bg;
}

.records-icon-glyph {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: 52rpx;
  line-height: 1;
}

.records-icon--danger .records-icon-glyph {
  color: $color-semantic-danger-strong;
}

.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.state-desc {
  max-width: 480rpx;
  margin-top: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.state-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: $spacing-lg;
  padding: 0 $spacing-xl;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;

  &:active {
    background: $color-bg-page;
  }
}

.state-btn-text {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

// ---- 加载骨架 -----------------------------------------------
.records-skeleton {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.skeleton-item {
  position: relative;
  padding-left: 40rpx;
}

.skeleton-dot {
  position: absolute;
  left: 8rpx;
  top: 64rpx;
  width: 14rpx;
  height: 14rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
  opacity: 0.6;
}

.skeleton-card {
  padding: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.skeleton-meta {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-height: 64rpx;
  margin-bottom: $spacing-md;
}

.skeleton-pill,
.skeleton-date,
.skeleton-line {
  display: block;
  background: $color-border-subtle;
  animation: skeleton-breathe 1.8s $animation-easing-default infinite;
}

.skeleton-pill {
  width: 104rpx;
  height: 40rpx;
  border-radius: $border-radius-full;
}

.skeleton-date {
  width: 144rpx;
  height: 20rpx;
  border-radius: $border-radius-sm;
}

.skeleton-line {
  height: 20rpx;
  border-radius: $border-radius-sm;
}

.skeleton-line--short {
  width: 64%;
}

.skeleton-line + .skeleton-line {
  margin-top: $spacing-md;
}

@keyframes skeleton-breathe {
  0%,
  100% {
    opacity: 0.24;
  }
  50% {
    opacity: 0.5;
  }
}

// ---- 吸底提交栏 ---------------------------------------------
.submit-bar {
  padding: $spacing-md $component-page-padding-x calc($spacing-md + env(safe-area-inset-bottom));
  border-top: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
}

.bar-error {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  min-height: 64rpx;
  margin-bottom: $spacing-sm;
  padding: 0 $spacing-md;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
}

.bar-error-glyph {
  flex: none;
  color: $color-semantic-danger-strong;
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  line-height: 1;
}

.bar-error-text {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.bar-error-retry {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  min-width: 88rpx;
  height: 88rpx;
  margin: -12rpx 0;
}

.bar-error-retry-text {
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

.bar-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 36rpx;
  margin-bottom: $spacing-sm;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  text-align: center;
}

.hint-default {
  color: $color-text-muted;
}

.hint-required {
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  color: $color-semantic-danger-strong;
  font-weight: $typography-font-weight-medium;
}

.hint-required-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-xs;
  line-height: 1;
}

.hint-submitting {
  color: $color-text-muted;
}

.submit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  min-height: 96rpx;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.985);
  }
}

// 阻断态：灰底灰字，点击只把焦点送回输入框（设计稿 required 态）。
.submit-btn.is-disabled {
  border-color: $color-border-subtle;
  background: $color-bg-page;
  box-shadow: none;

  &:active {
    transform: none;
  }
}

.submit-btn-text {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.submit-btn.is-disabled .submit-btn-text {
  color: $color-text-muted;
}

.btn-spinner {
  width: 28rpx;
  height: 28rpx;
  border: 4rpx solid $color-text-primary;
  border-right-color: transparent;
  border-radius: $border-radius-full;
  animation: spinner-rotate 0.75s linear infinite;
}

@keyframes spinner-rotate {
  to {
    transform: rotate(360deg);
  }
}

// ---- 成功整页 -----------------------------------------------
.success-view {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: calc(100vh - 120rpx);
  padding: 48rpx 0 $spacing-xl;
  text-align: center;
}

.success-mark {
  position: relative;
  width: 224rpx;
  height: 224rpx;
  margin-bottom: $spacing-xl;
}

.success-ring {
  position: absolute;
  border-radius: $border-radius-full;
}

.success-ring--outer {
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: $color-brand-primary-bg;
}

.success-ring--mid {
  top: 32rpx;
  right: 32rpx;
  bottom: 32rpx;
  left: 32rpx;
  border: 1rpx solid $color-brand-primary-light;
  background: $color-bg-card;
}

.success-ring--inner {
  top: 60rpx;
  right: 60rpx;
  bottom: 60rpx;
  left: 60rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background: $color-brand-primary;
}

.success-check {
  color: $color-bg-card;
  font-family: remixicon;
  font-size: 52rpx;
  line-height: 1;
}

.success-title {
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.success-desc {
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.success-card {
  width: 100%;
  max-width: 660rpx;
  margin-top: $spacing-xl;
  padding: $spacing-lg;
  border: 1rpx solid $color-border-subtle;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: left;
}

.success-card-head {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.success-card-date {
  margin-left: auto;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.success-card-content {
  display: -webkit-box;
  margin-top: $spacing-md;
  overflow: hidden;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.7;
  white-space: pre-wrap;
  overflow-wrap: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.success-actions {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  width: 100%;
  max-width: 660rpx;
  margin-top: $spacing-xl;
}

.success-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 96rpx;
  padding: 0 $spacing-lg;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;

  &:active {
    transform: scale(0.985);
  }
}

.success-btn--primary {
  background: $gradient-brand;
  box-shadow: $shadow-md;
}

.success-btn--secondary {
  border-color: $color-border-hairline;
  background: $color-bg-card;
}

.success-btn-text {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
}

.success-btn-text--quiet {
  color: $color-text-body;
}

@media (prefers-reduced-motion: reduce) {
  .skeleton-pill,
  .skeleton-date,
  .skeleton-line,
  .btn-spinner {
    animation: none;
  }

  .note-card,
  .type-pill,
  .textarea-wrap,
  .record-card,
  .record-chevron,
  .submit-btn,
  .success-btn {
    transition: none;
  }
}
</style>
