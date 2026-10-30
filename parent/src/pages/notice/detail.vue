<script setup>
/**
 * 通知详情 · 一页正式通知
 *
 * 标题、发布人、正文、附件、已读回执收在同一张纸面里，用一条细分隔线代替
 * 层层嵌套的卡片。暖橙只出现在类型胶囊、头像底色和底部确认按钮三处。
 * 需要回执（need_confirm）与仅告知两类通知共用这一页，底栏按条件出现。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { fetchNoticeDetail, confirmNotice, markNoticeRead } from "@/api/parent-notice"
import { formatCommentTime } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形（App.vue 已全局注册字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  image: "", // image-line
  check: "", // check-line
  alert: "" // error-warning-line
}

const detail = ref(null)
const noticeId = ref("")
const loading = ref(true)
const hasLoaded = ref(false)
const loadError = ref("")
const confirming = ref(false)
const flashingAttId = ref(null)

// 只允许最后一次请求写数据：重试与首次加载可能并发。
let loadSeq = 0
let flashTimer = null

const showSkeleton = computed(() => loading.value || (!hasLoaded.value && !loadError.value))
const showError = computed(() => Boolean(loadError.value) && !loading.value)
const showSheet = computed(
  () => !loading.value && hasLoaded.value && !loadError.value && Boolean(detail.value)
)
const needConfirmBar = computed(
  () => Boolean(detail.value?.need_confirm) && !detail.value?.is_confirmed
)
// 设计稿：有底栏时正文底部留 96px，无底栏收到 48px。
const bottomSpace = computed(() => (showSheet.value && needConfirmBar.value ? "192rpx" : "96rpx"))

const paragraphs = computed(() => {
  const content = String(detail.value?.content || "")
  // content 是带换行的纯文本：按空行/换行统一拆段，否则长文会挤成一段。
  return content
    .split(/\n+/)
    .map(line => line.trim())
    .filter(Boolean)
})

const attachments = computed(() =>
  Array.isArray(detail.value?.attachments) ? detail.value.attachments : []
)

const categoryText = computed(
  () => detail.value?.category_label || (detail.value?.category === "task" ? "任务" : "通知")
)

const authorName = computed(
  () => detail.value?.creator?.nickname || "园所通知"
)

const authorInitial = computed(() => {
  const name = String(detail.value?.creator?.nickname || "").trim()
  return name ? name.slice(0, 1) : "园"
})

const publishedLabel = computed(() => formatCommentTime(detail.value?.published_at))
const readLabel = computed(() => formatCommentTime(detail.value?.read_at))
const confirmedLabel = computed(() => formatCommentTime(detail.value?.confirmed_at))

// 已读条一行说清两件事；未读时整块不渲染。
const readLine = computed(() => {
  if (!detail.value?.is_read) return ""
  const parts = [readLabel.value ? `已读 ${readLabel.value}` : "已读"]
  if (detail.value.is_confirmed) {
    parts.push(confirmedLabel.value ? `已确认 ${confirmedLabel.value}` : "已确认")
  }
  return parts.join(" · ")
})

async function markRead(id, seq) {
  try {
    await markNoticeRead(id)
    if (seq !== loadSeq || !detail.value) return
    // 打开即已读：读记录时间由服务端落库，这里补到页面上，已读条立刻出现。
    if (!detail.value.is_read) {
      detail.value = {
        ...detail.value,
        is_read: true,
        read_at: detail.value.read_at || new Date().toISOString()
      }
    }
  } catch (err) {
    // 已读记录失败不影响正文展示；http 层已 toast，这里留排查线索，不静默丢。
    console.warn("[notice-detail] mark read failed:", err?.message || err)
  }
}

async function loadData(id) {
  const seq = ++loadSeq
  loading.value = true
  loadError.value = ""
  try {
    const res = await fetchNoticeDetail(id)
    if (seq !== loadSeq) return
    const data = res?.data || null
    if (!data) {
      detail.value = null
      loadError.value = "通知加载失败"
      return
    }
    detail.value = data
    hasLoaded.value = true
    markRead(id, seq)
  } catch (err) {
    if (seq !== loadSeq) return
    console.warn("[notice-detail] load failed:", err?.message || err)
    detail.value = null
    loadError.value = "通知加载失败"
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

async function handleRetry() {
  if (loading.value || !noticeId.value) return
  await loadData(noticeId.value)
  if (!loadError.value) uni.showToast({ title: "已重新加载", icon: "none" })
}

async function handleConfirm() {
  if (confirming.value || !needConfirmBar.value) return
  confirming.value = true
  try {
    const res = await confirmNotice(detail.value.id)
    const confirmedAt = res?.data?.confirmed_at || new Date().toISOString()
    detail.value = {
      ...detail.value,
      is_confirmed: true,
      confirmed_at: confirmedAt,
      // 确认接口同时落 read_at：未读通知确认后，已读条应与横幅一起出现。
      is_read: true,
      read_at: detail.value.read_at || confirmedAt
    }
    uni.showToast({ title: "已确认收到", icon: "none" })
  } catch (err) {
    // 失败保持按钮可点，允许家长重试。
    console.warn("[notice-detail] confirm failed:", err?.message || err)
    uni.showToast({ title: err?.message || "操作失败", icon: "none" })
  } finally {
    confirming.value = false
  }
}

// 按设计不做原图预览（个人主体/未配置业务域名时 previewImage 会白屏），
// 只给描边反馈 + 轻提示，明确「这里是图片」。
function tapAttachment(att) {
  flashingAttId.value = att.id
  uni.showToast({ title: "图片预览暂未开放", icon: "none" })
  if (flashTimer) clearTimeout(flashTimer)
  flashTimer = setTimeout(() => {
    if (flashingAttId.value === att.id) flashingAttId.value = null
    flashTimer = null
  }, 520)
}

onLoad((options) => {
  noticeId.value = options?.id ? String(options.id) : ""
  if (!noticeId.value) {
    loading.value = false
    loadError.value = "通知加载失败"
    return
  }
  loadData(noticeId.value)
})
</script>

<template>
  <PageShell :bottom-space="bottomSpace" max-width="430px">
    <!-- ===== 通知纸面 ===== -->
    <view v-if="showSheet" class="sheet">
      <!-- 已回执横幅：仅需确认且已确认时出现 -->
      <view v-if="detail.need_confirm && detail.is_confirmed" class="confirm-banner">
        <text class="banner-icon">{{ ICON.check }}</text>
        <text class="banner-label">已确认收到</text>
        <text v-if="confirmedLabel" class="banner-time">{{ confirmedLabel }}</text>
      </view>

      <view class="notice-meta">
        <text class="notice-cat">{{ categoryText }}</text>
        <text v-if="publishedLabel" class="notice-published">{{ publishedLabel }}</text>
      </view>

      <text class="notice-title">{{ detail.title }}</text>

      <view class="notice-author">
        <view class="author-avatar">
          <text class="author-initial">{{ authorInitial }}</text>
        </view>
        <text class="author-name">{{ authorName }}</text>
      </view>

      <view class="sheet-rule" />

      <view class="notice-body">
        <text v-for="(para, index) in paragraphs" :key="index" class="notice-para">{{ para }}</text>
      </view>

      <!-- 附件：有图才出现 -->
      <view v-if="attachments.length" class="attachments">
        <view class="attachments-head">
          <text class="attachments-label">附件</text>
          <text class="attachments-count">{{ attachments.length }} 张</text>
        </view>
        <view class="attachments-row">
          <view v-for="(att, index) in attachments" :key="att.id" class="attachment">
            <view
              class="att-thumb"
              :class="{ 'is-flash': flashingAttId === att.id }"
              @click="tapAttachment(att)"
            >
              <text class="att-icon">{{ ICON.image }}</text>
            </view>
            <text class="att-caption">{{ att.file_name || `图片 ${index + 1}` }}</text>
          </view>
        </view>
      </view>

      <!-- 已读条：未读整块隐藏；已确认转成功色 -->
      <view
        v-if="detail.is_read"
        class="read-block"
        :class="{ 'is-confirmed': detail.is_confirmed }"
      >
        <text class="read-icon">{{ ICON.check }}</text>
        <text class="read-line">{{ readLine }}</text>
      </view>
    </view>

    <!-- ===== 加载骨架 ===== -->
    <view v-if="showSkeleton" class="skeleton-card">
      <view class="sk-row">
        <view class="sk sk-pill" />
        <view class="sk sk-stamp" />
      </view>
      <view class="sk-heading">
        <view class="sk sk-title-line sk-w-86" />
        <view class="sk sk-title-line sk-w-58" />
      </view>
      <view class="sk-author">
        <view class="sk sk-avatar" />
        <view class="sk sk-name" />
      </view>
      <view class="sk-body">
        <view class="sk sk-text sk-w-100" />
        <view class="sk sk-text sk-w-94" />
        <view class="sk sk-text sk-w-88" />
        <view class="sk sk-text sk-w-62" />
      </view>
      <view class="sk-thumbs">
        <view class="sk sk-thumb" />
        <view class="sk sk-thumb" />
      </view>
    </view>

    <!-- ===== 加载失败 ===== -->
    <view v-if="showError" class="error-card">
      <view class="error-orb">
        <text class="error-icon">{{ ICON.alert }}</text>
      </view>
      <text class="error-title">通知加载失败</text>
      <text class="error-desc">网络异常或这条通知已被撤回</text>
      <view v-if="noticeId" class="retry-btn" @click="handleRetry">
        <text>重试</text>
      </view>
    </view>

    <!-- ===== 固定确认栏：仅需确认且未回执 ===== -->
    <template #bottom>
      <view v-if="showSheet && needConfirmBar" class="confirm-bar">
        <button
          class="confirm-btn"
          :class="{ 'is-busy': confirming }"
          :disabled="confirming"
          @click="handleConfirm"
        >
          <text class="confirm-icon">{{ ICON.check }}</text>
          <text class="confirm-label">{{ confirming ? "确认中…" : "确认收到" }}</text>
        </button>
      </view>
    </template>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 正式通知纸面：白纸、留白、舒展的正文；暖色只落在三处。
// ============================================================

// ---- 纸面 --------------------------------------------------
.sheet {
  padding: 40rpx 36rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

// ---- 已回执横幅 --------------------------------------------
.confirm-banner {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin: 0 0 36rpx;
  padding: $spacing-lg 28rpx;
  border-radius: $border-radius-xl;
  background: $color-semantic-success-bg;
  // 深一档的绿：浅绿底上 --success 只有约 3:1（审查 P2-4）。
  color: $color-semantic-success-strong;
}

.banner-icon {
  flex: none;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.banner-label {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.banner-time {
  margin-left: auto;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

// ---- 顶行：类型 + 发布时间 ---------------------------------
.notice-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  margin-bottom: 28rpx;
}

.notice-cat {
  flex: none;
  display: inline-flex;
  align-items: center;
  height: 48rpx;
  padding: 0 $component-badge-padding-x;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  letter-spacing: 0.02em;
}

.notice-published {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
  text-align: right;
}

// ---- 标题 --------------------------------------------------
.notice-title {
  display: block;
  max-width: 22em;
  margin-bottom: 28rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.45;
  overflow-wrap: break-word;
}

// ---- 发布人 ------------------------------------------------
.notice-author {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.author-avatar {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.author-initial {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.author-name {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

// ---- 分隔线 ------------------------------------------------
.sheet-rule {
  height: 1rpx;
  margin: 36rpx 0;
  background: $color-border-hairline;
}

// ---- 正文 --------------------------------------------------
.notice-body {
  max-width: 34em;
  color: $color-text-body;
  font-size: $typography-font-size-lg;
  line-height: 1.85;
}

.notice-para {
  display: block;
  overflow-wrap: break-word;

  & + & {
    margin-top: 28rpx;
  }
}

// ---- 附件 --------------------------------------------------
.attachments {
  margin-top: 44rpx;
}

.attachments-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: 20rpx;
}

.attachments-label {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  letter-spacing: 0.02em;
}

.attachments-count {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.attachments-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-lg;
}

.attachment {
  width: 192rpx;
}

.att-thumb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 192rpx;
  height: 192rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  transition:
    background-color $animation-duration-fast $animation-easing-default,
    border-color $animation-duration-fast $animation-easing-default,
    transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.99);
    background: $color-brand-primary-bg;
    border-color: $color-brand-primary-light;
  }
}

// 点击后的描边反馈：用边框换色而不是 outline，避免小程序不支持 outline。
.att-thumb.is-flash {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
}

.att-icon {
  color: $color-text-hint;
  font-family: remixicon;
  font-size: 44rpx;
  line-height: 1;
}

.att-caption {
  display: block;
  width: 192rpx;
  margin-top: 20rpx;
  overflow: hidden;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

// ---- 已读条 ------------------------------------------------
.read-block {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: 40rpx;
  padding: $spacing-lg 28rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  color: $color-text-muted;
}

.read-block.is-confirmed {
  background: $color-semantic-success-bg-subtle;
  color: $color-semantic-success-strong;
}

.read-icon {
  flex: none;
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

.read-line {
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
  overflow-wrap: break-word;
}

// ---- 固定确认栏 --------------------------------------------
.confirm-bar {
  padding: $spacing-lg 28rpx calc(24rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl $border-radius-xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.confirm-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  width: 100%;
  height: 96rpx;
  padding: 0 40rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  box-shadow: $shadow-md;
  transition:
    transform $animation-duration-fast $animation-easing-default,
    opacity $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.99);
    box-shadow: $shadow-sm;
  }

  &.is-busy,
  &[disabled] {
    opacity: 0.6;
    box-shadow: none;
  }
}

.confirm-icon {
  flex: none;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

// ---- 加载骨架 ----------------------------------------------
.skeleton-card {
  padding: 40rpx 36rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

.sk {
  background: $color-border-hairline;
  border-radius: $border-radius-md;
  animation: sk-breathe 1.8s $animation-easing-default infinite;
}

.sk-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  margin-bottom: 28rpx;
}

.sk-pill {
  width: 112rpx;
  height: 48rpx;
  border-radius: $border-radius-full;
}

.sk-stamp {
  width: 136rpx;
  height: 24rpx;
  border-radius: $border-radius-sm;
}

.sk-heading {
  display: flex;
  flex-direction: column;
  gap: 18rpx;
}

.sk-title-line {
  height: 40rpx;
  border-radius: $border-radius-sm;
}

.sk-w-100 {
  width: 100%;
}

.sk-w-94 {
  width: 94%;
}

.sk-w-88 {
  width: 88%;
}

.sk-w-86 {
  width: 86%;
}

.sk-w-62 {
  width: 62%;
}

.sk-w-58 {
  width: 58%;
}

.sk-author {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-xl;
}

.sk-avatar {
  width: 64rpx;
  height: 64rpx;
  border-radius: $border-radius-full;
}

.sk-name {
  width: 128rpx;
  height: 24rpx;
  border-radius: $border-radius-sm;
}

.sk-body {
  display: flex;
  flex-direction: column;
  gap: 28rpx;
  margin-top: 48rpx;
}

.sk-text {
  height: 24rpx;
}

.sk-thumbs {
  display: flex;
  gap: $spacing-lg;
  margin-top: 48rpx;
}

.sk-thumb {
  width: 192rpx;
  height: 192rpx;
  border-radius: $border-radius-xl;
}

@keyframes sk-breathe {
  0%,
  100% {
    opacity: 0.45;
  }
  50% {
    opacity: 0.9;
  }
}

// ---- 失败态 ------------------------------------------------
.error-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 70vh;
  padding: 88rpx 48rpx 80rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
  text-align: center;
}

.error-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
}

.error-icon {
  color: $color-semantic-danger;
  font-family: remixicon;
  font-size: 52rpx;
  line-height: 1;
}

.error-title {
  margin-top: $spacing-xl;
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.error-desc {
  margin-top: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.retry-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 240rpx;
  height: $component-input-height;
  margin-top: 48rpx;
  padding: 0 48rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  transition:
    background-color $animation-duration-fast $animation-easing-default,
    transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.99);
    background: $color-bg-page;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
    opacity: 0.6;
  }

  .att-thumb,
  .confirm-btn,
  .retry-btn {
    transition: none;
  }
}
</style>
