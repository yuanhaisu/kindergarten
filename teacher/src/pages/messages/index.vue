<script setup>
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import BottomNav from "@/components/BottomNav.vue"
import PageShell from "@/components/PageShell.vue"
import {
  fetchTeacherMessages,
  markAllTeacherMessagesRead,
  markTeacherMessageRead
} from "@/api/messages"
import {
  MESSAGE_SEGMENTS,
  normalizeMessage,
  segmentCount,
  segmentUnread
} from "@/utils/messages"

const PAGE_SIZE = 100
// 契约单页上限 100。分段计数与「全部已读」都要全量口径，所以一次取完而不是只取第一页。
// 硬上限 10×100=1000 条；超出后分段计数/底栏角标可能不完整（见 self-review known_gaps）。
const MAX_PAGES = 10

const activeSegment = ref("all")
const allMessages = ref([])
const unreadTotal = ref(0)
const loading = ref(true)
const loadError = ref("")
const refreshing = ref(false)
const markingAll = ref(false)
let loadSeq = 0

const viewState = computed(() => {
  if (loading.value) return "loading"
  if (loadError.value) return "error"
  if (allMessages.value.length === 0) return "empty"
  return "normal"
})

const segments = computed(() =>
  MESSAGE_SEGMENTS.map((segment) => ({
    ...segment,
    count: segmentCount(allMessages.value, segment.key)
  }))
)

const filteredMessages = computed(() => {
  if (activeSegment.value === "all") return allMessages.value
  return allMessages.value.filter((item) => item.type === activeSegment.value)
})

const filteredUnread = computed(() => segmentUnread(allMessages.value, activeSegment.value))
// 口径行与当前分类一致；「全部已读」的可用性却按全量未读判定。
// brief B.2 的「M=0 禁用」若也按当前分类执行，会出现「按钮可点但静默无反应」
//（design review teacher-18 P1-1）；这里坚持契约的全量作用域，不改行为。
const summaryText = computed(
  () => `共 ${filteredMessages.value.length} 条 · ${filteredUnread.value} 条未读`
)
const canMarkAll = computed(() => unreadTotal.value > 0)

async function fetchAllMessages() {
  const collected = []
  let page = 1
  let total = Infinity
  while (page <= MAX_PAGES && collected.length < total) {
    const res = await fetchTeacherMessages(
      { page, page_size: PAGE_SIZE },
      { silent: true }
    )
    const data = res.data || {}
    const list = Array.isArray(data.list) ? data.list : []
    collected.push(...list)
    total = Number(data.total) || collected.length
    unreadTotal.value = Number(data.unread_count) || 0
    if (list.length === 0) break
    page += 1
  }
  return collected
}

async function loadMessages({ refresh = false } = {}) {
  const seq = ++loadSeq
  // 下拉刷新时保留现有列表，避免整页闪回骨架；首次加载与重试才显示骨架
  if (!refresh) loading.value = true
  loadError.value = ""
  try {
    const list = await fetchAllMessages()
    if (seq !== loadSeq) return
    allMessages.value = list.map(normalizeMessage)
  } catch (err) {
    if (seq !== loadSeq) return
    if (refresh && allMessages.value.length) {
      // 刷新失败不清空已加载内容：旧列表仍然可用，只提示本次刷新失败
      uni.showToast({ title: "刷新失败，请重试", icon: "none" })
    } else {
      allMessages.value = []
      loadError.value = "消息加载失败"
    }
    console.warn("[messages] load failed:", err?.message || err)
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      refreshing.value = false
    }
  }
}

function onRefresh() {
  if (refreshing.value) return
  refreshing.value = true
  loadMessages({ refresh: true })
}

function retryLoad() {
  loadMessages()
}

function switchSegment(key) {
  if (viewState.value !== "normal" || activeSegment.value === key) return
  activeSegment.value = key
}

function openMessage(item) {
  if (!item.isRead) {
    // 先本地置读再跳转：返回列表时不应再出现未读点。
    // 契约要求「点开即读」，失败也不回滚（设计稿明确），只记录日志。
    item.isRead = true
    unreadTotal.value = Math.max(0, unreadTotal.value - 1)
    markTeacherMessageRead(item.id, item.type, { silent: true }).catch((err) => {
      console.warn("[messages] mark read failed:", err?.message || err)
    })
  }
  if (item.type === "notice") {
    uni.navigateTo({ url: "/pages/notice/index" })
    return
  }
  if (item.type === "leave") {
    uni.navigateTo({ url: "/pages/leave/index" })
    return
  }
  // 系统消息没有详情页可跳（契约里 system 无业务对象），保持 toast
  uni.showToast({ title: `查看：${item.title}`, icon: "none" })
}

async function markAllRead() {
  if (!canMarkAll.value || markingAll.value) return
  markingAll.value = true
  try {
    await markAllTeacherMessagesRead({ silent: true })
    allMessages.value.forEach((item) => {
      item.isRead = true
    })
    unreadTotal.value = 0
    uni.showToast({ title: "已全部标为已读", icon: "none" })
  } catch (err) {
    uni.showToast({ title: "操作失败，请重试", icon: "none" })
    console.warn("[messages] mark all read failed:", err?.message || err)
  } finally {
    markingAll.value = false
  }
}

onLoad(() => loadMessages())
</script>

<template>
  <PageShell max-width="430px" :padded="false" :safe-bottom="false">
    <view class="messages-page">
      <!-- 分段条放在 scroll-view 外：小程序不支持 sticky，外置才真正吸顶 -->
      <view v-if="viewState === 'normal'" class="seg-bar">
        <view class="segmented">
          <view
            v-for="segment in segments"
            :key="segment.key"
            class="seg-btn"
            :class="{ 'is-active': activeSegment === segment.key }"
            @click="switchSegment(segment.key)"
          >
            <text class="seg-label">{{ segment.label }}</text>
            <text class="seg-count">{{ segment.count }}</text>
          </view>
        </view>
      </view>

      <scroll-view
        class="messages-scroll"
        scroll-y
        :refresher-enabled="true"
        :refresher-triggered="refreshing"
        @refresherrefresh="onRefresh"
      >
        <view class="messages-body">
          <template v-if="viewState === 'normal'">
            <view class="inbox-meta">
              <text class="inbox-summary">{{ summaryText }}</text>
              <view
                class="mark-all"
                :class="{ 'is-disabled': !canMarkAll }"
                @click="markAllRead"
              >
                <text>全部已读</text>
              </view>
            </view>

            <view v-if="filteredMessages.length" class="inbox-card">
              <view
                v-for="msg in filteredMessages"
                :key="msg.type + '-' + msg.id"
                class="message-cell"
              >
                <view
                  class="message-row"
                  :class="msg.isRead ? 'is-read' : 'is-unread'"
                  @click="openMessage(msg)"
                >
                  <view class="message-icon" :class="'type-icon--' + msg.iconTone">
                    <text class="icon message-glyph">{{ msg.icon }}</text>
                    <view v-if="!msg.isRead" class="unread-dot" />
                  </view>
                  <view class="message-body">
                    <view class="message-head">
                      <text class="message-title">{{ msg.title }}</text>
                      <text v-if="msg.timeText" class="message-time">{{ msg.timeText }}</text>
                    </view>
                    <text v-if="msg.content" class="message-summary">{{ msg.content }}</text>
                    <view class="message-meta">
                      <text class="type-badge">{{ msg.badge }}</text>
                      <text
                        v-if="msg.detail.text"
                        class="meta-detail"
                        :class="msg.detail.tone ? 'status-text status-text--' + msg.detail.tone : ''"
                      >{{ msg.detail.text }}</text>
                    </view>
                  </view>
                  <text class="icon message-chevron">&#xea6e;</text>
                </view>
              </view>
            </view>

            <view v-else class="category-empty">
              <text class="icon category-empty-glyph">&#xee4f;</text>
              <text class="category-empty-text">该分类暂无消息</text>
            </view>
          </template>

          <template v-else-if="viewState === 'loading'">
            <text class="sr-only">正在加载消息</text>
            <view class="inbox-card skeleton-card">
              <view v-for="row in 5" :key="'sk-' + row" class="skeleton-row">
                <view class="sk-icon" />
                <view class="sk-copy">
                  <view class="sk-line sk-title" />
                  <view class="sk-line sk-body" />
                </view>
                <view class="sk-line sk-time" />
              </view>
            </view>
          </template>

          <view v-else-if="viewState === 'empty'" class="inbox-state">
            <view class="state-icon state-icon--empty">
              <text class="icon state-glyph">&#xee4f;</text>
            </view>
            <text class="state-title">暂无消息</text>
            <text class="state-copy">收到的通知、审批与系统消息会汇总到这里</text>
            <view class="state-btn" @click="retryLoad">
              <text class="icon state-btn-glyph">&#xf064;</text>
              <text>刷新消息</text>
            </view>
          </view>

          <view v-else class="inbox-state" role="alert">
            <view class="state-icon state-icon--error">
              <text class="icon state-glyph">&#xeca1;</text>
            </view>
            <text class="state-title">消息加载失败</text>
            <text class="state-copy">网络异常，请稍后重试</text>
            <view class="state-btn" @click="retryLoad"><text>重试</text></view>
          </view>

          <view class="bottom-safe" />
        </view>
      </scroll-view>
    </view>

    <template #bottom>
      <BottomNav :active="2" :unread-count="unreadTotal" />
    </template>
  </PageShell>
</template>

<style scoped lang="scss">
.icon {
  font-family: remixicon;
  font-style: normal;
  line-height: 1;
}

.messages-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

/* ---------- 分类分段（scroll-view 外置，滚动时保持可见） ---------- */
.seg-bar {
  flex: none;
  padding: $spacing-lg $component-page-padding-x $spacing-md;
  background: $color-bg-page;
}

.segmented {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 4rpx;
  padding: 6rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.seg-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  min-height: $component-input-height;
  padding: 0 8rpx;
  border-radius: $border-radius-full;
  background: transparent;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.3;
}

.seg-btn.is-active {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.seg-count {
  margin-left: 8rpx;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

/* ---------- 滚动区 ---------- */
.messages-scroll {
  flex: 1;
  min-height: 0;
  height: 0;
}

.messages-body {
  padding: 0 $component-page-padding-x;
}

/* ---------- 列表口径 ---------- */
.inbox-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: $component-input-height;
  margin: 4rpx 4rpx $spacing-md;
}

.inbox-summary {
  min-width: 0;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  color: $color-text-muted;
}

.mark-all {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  min-height: $component-input-height;
  margin-right: -$spacing-md;
  padding: 0 $spacing-md;
  border-radius: $border-radius-lg;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  color: $color-brand-primary-dark;
}

.mark-all.is-disabled {
  color: $color-text-hint;
  opacity: 0.55;
  pointer-events: none;
}

/* ---------- 收件箱 ---------- */
.inbox-card {
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.message-cell + .message-cell {
  border-top: 1rpx solid $color-border-hairline;
}

.message-row {
  display: grid;
  grid-template-columns: 80rpx minmax(0, 1fr) 24rpx;
  align-items: start;
  column-gap: $spacing-lg;
  min-height: $component-input-height;
  padding: 28rpx;
}

.message-row:active {
  background: $color-brand-primary-bg;
}

.message-icon {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  margin-top: 2rpx;
  border-radius: $border-radius-xl;
}

.type-icon--notice {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.type-icon--system {
  background: $color-bg-page;
  color: $color-text-body;
}

.type-icon--pending {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}

.type-icon--approved {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.type-icon--rejected {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.type-icon--muted {
  background: $color-bg-page;
  color: $color-text-muted;
}

.message-glyph {
  font-size: $typography-font-size-3xl;
}

.unread-dot {
  position: absolute;
  top: -4rpx;
  right: -4rpx;
  width: 16rpx;
  height: 16rpx;
  border: 4rpx solid $color-bg-card;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.message-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.message-head {
  display: flex;
  align-items: flex-start;
  min-width: 0;
}

.message-title {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-normal;
  line-height: 1.4;
  color: $color-text-muted;
  word-break: break-all;
}

.message-time {
  flex: none;
  margin-top: 2rpx;
  margin-left: $spacing-md;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  color: $color-text-hint;
  white-space: nowrap;
}

.message-summary {
  display: -webkit-box;
  margin-top: 6rpx;
  overflow: hidden;
  font-size: $typography-font-size-base;
  line-height: 1.55;
  color: $color-text-muted;
  word-break: break-all;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

/* 未读：标题提亮加粗；已读整行降一档灰度，不整行铺底色 */
.message-row.is-unread .message-title {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

.message-row.is-unread .message-summary {
  color: $color-text-body;
}

.message-row.is-read .message-icon {
  opacity: 0.62;
}

.message-row.is-read .type-badge {
  color: $color-text-muted;
}

.message-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  min-width: 0;
  margin-top: $spacing-sm;
}

.type-badge {
  display: flex;
  align-items: center;
  min-height: 36rpx;
  padding: 2rpx 14rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
  color: $color-text-body;
}

.meta-detail {
  min-width: 0;
  margin-left: $spacing-sm;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-normal;
  line-height: 1.4;
  color: $color-text-muted;
  word-break: break-all;
}

.status-text {
  font-weight: $typography-font-weight-medium;
}

.status-text--pending {
  color: $color-semantic-warning;
}

.status-text--approved {
  color: $color-semantic-success;
}

.status-text--rejected {
  color: $color-semantic-danger;
}

.status-text--muted {
  color: $color-text-muted;
}

.message-chevron {
  align-self: center;
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

/* ---------- 分类空态 ---------- */
.category-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 376rpx;
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.category-empty-glyph {
  font-size: 48rpx;
  color: $color-text-hint;
}

.category-empty-text {
  margin-top: $spacing-md;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  color: $color-text-muted;
}

/* ---------- 空态 / 失败 ---------- */
.inbox-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 880rpx;
  padding: 64rpx 48rpx;
  text-align: center;
}

.state-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 128rpx;
  height: 128rpx;
  border-radius: $border-radius-full;
}

.state-icon--empty {
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  color: $color-text-muted;
}

.state-icon--error {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.state-glyph {
  font-size: 60rpx;
}

.state-title {
  margin-top: $spacing-xl;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  color: $color-text-primary;
}

.state-copy {
  max-width: 540rpx;
  margin-top: $spacing-md;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  color: $color-text-muted;
}

.state-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  margin-top: 40rpx;
  padding: 0 44rpx;
  border-radius: $border-radius-full;
  background: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  color: $color-bg-card;
  box-shadow: $shadow-sm;
}

.state-btn-glyph {
  margin-right: 12rpx;
  font-size: $typography-font-size-lg;
}

/* ---------- 加载骨架 ---------- */
/* 设计稿把加载文案放在无障碍层（sr-only），屏幕只留 5 行骨架 */
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  margin: -1px;
  padding: 0;
  overflow: hidden;
  clip-path: inset(50%);
  border: 0;
}

.skeleton-card {
  box-shadow: $shadow-sm;
}

.skeleton-row {
  display: grid;
  grid-template-columns: 80rpx minmax(0, 1fr) 56rpx;
  align-items: center;
  column-gap: $spacing-lg;
  padding: 28rpx;
}

.skeleton-row + .skeleton-row {
  border-top: 1rpx solid $color-border-hairline;
}

.sk-icon {
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-xl;
  background: $color-border-subtle;
  opacity: 0.5;
  animation: messages-skeleton-pulse 1.7s $animation-easing-default infinite;
}

.sk-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.sk-line {
  display: block;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  opacity: 0.4;
  animation: messages-skeleton-pulse 1.7s $animation-easing-default infinite;
}

.sk-title {
  width: 62%;
  height: 20rpx;
}

.sk-body {
  width: 92%;
  height: 18rpx;
  margin-top: $spacing-md;
}

.sk-time {
  justify-self: end;
  width: 56rpx;
  height: 16rpx;
}

@keyframes messages-skeleton-pulse {
  0%,
  100% {
    opacity: 0.55;
  }

  50% {
    opacity: 1;
  }
}

/* ---------- 底部 ---------- */
.bottom-safe {
  height: calc(160rpx + env(safe-area-inset-bottom));
}
</style>
