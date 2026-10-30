<script setup>
/**
 * 消息 · 门口的信箱
 *
 * 口径行的两个数字回答「几条没看、一共几条」；列表按「今天 / 昨天 / 更早」
 * 分堆。未读是刚到的信（品牌色小圆点 + 彩色类型位），读过的整行降灰，
 * 但卡片仍是白底 —— 不铺灰底、不整行染色、不加左边框。
 *
 * 分页：page_size=20，触底加载下一页。「共 N 条」取接口 total，不按已渲染
 * 卡片数算，否则永远停在 20，第 21 条以后的消息会被永久藏掉。
 */
import { computed, ref } from "vue"
import { onLoad, onShow, onPullDownRefresh, onReachBottom } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getToken } from "@/utils/auth"
import { rememberPendingRoute } from "@/utils/pendingRoute"
import { fetchMessages, fetchUnreadCount, markMessageRead } from "@/api/parent-messages"
import {
  groupMessagesByDay,
  messageTimeText,
  messageTypeLabel,
  messageTypeMeta
} from "@/utils/messages"
import PageShell from "@/components/PageShell.vue"

const PAGE_SIZE = 20
// 设计稿要求「重试中…」至少停留 1 秒：请求太快时闪一下反而像没点上。
const RETRY_MIN_MS = 1000

// remixicon 字形（App.vue 已全局注册字体）；WXML 不允许内联 SVG。
const ICON = {
  inbox: "\u{ee4f}", // inbox-line
  alert: "\u{eca1}", // error-warning-line
  chevron: "\u{ea6e}" // arrow-right-s-line
}

// 家长端 tabBar 只有这四页（pages.json → tabBar.list）。跳到 tabBar 页必须 switchTab，
// navigateTo 会直接失败——这个差异只在真实点击时暴露。
const TAB_PAGES = [
  "/pages/index/index",
  "/pages/growth/index",
  "/pages/pickup/index",
  "/pages/profile/index"
]

const messages = ref([])
const total = ref(0)
const unread = ref(0)
const loading = ref(false)
const retrying = ref(false)
const refreshingEmpty = ref(false)
const loadingMore = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const pageNo = ref(1)

// 只允许最后一次请求写数据：下拉刷新/重试可能和触底加载并发。
let loadSeq = 0

const hasMore = computed(() => messages.value.length < total.value)
const groups = computed(() => groupMessagesByDay(messages.value))
const showSummary = computed(() => messages.value.length > 0)
const showGroups = computed(() => messages.value.length > 0)
const showSkeleton = computed(() => loading.value && messages.value.length === 0)
// 错误卡叠在列表上方：已有列表仍可读可点，不清空也不锁点击。
// retrying 也算「卡还在」：重试成功时 loadError 立刻清空，但按钮要停留满 1 秒
// 显示「重试中…」，否则用户看不到任何反馈。
const showErrorCard = computed(
  () => (Boolean(loadError.value) || retrying.value) && !loading.value
)
const showEmpty = computed(
  () => hasLoaded.value && !loading.value && !retrying.value &&
    !loadError.value && messages.value.length === 0
)
const showNoMore = computed(() => showGroups.value && !hasMore.value && !loadingMore.value)
const unreadText = computed(() =>
  unread.value > 0 ? `${unread.value} 条未读` : "没有未读消息"
)
const totalText = computed(() => `共 ${total.value} 条`)

// action_path 由服务端写入，只允许小程序内部 `/pages/` 路径。
// 这里再挡一次 `://`：任何外部 URL 都不该被站内消息点开。
function isInternalPath(path) {
  return typeof path === "string" &&
    path.indexOf("/pages/") === 0 &&
    path.indexOf("://") === -1
}

function typeMetaOf(item) {
  return messageTypeMeta(item && item.type)
}

function typeLabelOf(item) {
  return messageTypeLabel(item && item.type)
}

// 点开即已读：先本地落状态（圆点立刻消失），再发请求；失败只记日志，
// 不回滚 UI —— 回滚会让刚点过的消息重新亮起来，看起来像点击没生效。
// 没有 action_path 的消息同样算“读过”，否则这些消息永远无法被标记。
function markRead(item) {
  if (!item || item.is_read) return
  item.is_read = true
  unread.value = Math.max(0, unread.value - 1)
  markMessageRead(item.id).catch((err) => {
    console.warn("[messages] 标记已读失败", err?.message || err)
  })
}

function openMessage(item) {
  if (!item) return
  const path = item.action_path
  // 没有 token 时先把目标路径存下来：http.js 的 401 会把当前页（消息页）记成回跳目标，
  // 那是「从哪掉线回哪恢复」，不是用户想去的任务详情。
  if (!getToken()) {
    if (isInternalPath(path)) rememberPendingRoute(path)
    uni.reLaunch({ url: "/pages/auth/login" })
    return
  }
  markRead(item)
  if (!isInternalPath(path)) return

  // 深链失效（任务被删、页面未注册）不能静默：点了没反应会被当成卡死。
  const fail = () => uni.showToast({ title: "该内容已不存在", icon: "none" })
  const pathname = path.split("?")[0]
  if (TAB_PAGES.indexOf(pathname) !== -1) {
    // tabBar 页不能带 query，switchTab 只认 path。
    uni.switchTab({ url: pathname, fail })
    return
  }
  uni.navigateTo({ url: path, fail })
}

function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * 拉第一页（重置分页）。keepView=true 时不清错误、不切骨架：
 * 重试按钮要能显示「重试中…」，下拉刷新也不该把正在读的列表换成骨架。
 */
async function loadFirstPage(options = {}) {
  const keepView = Boolean(options.keepView)
  const seq = ++loadSeq
  loadingMore.value = false
  if (!keepView) {
    loading.value = true
    loadError.value = ""
  }
  try {
    await ensureLogin()
    const [messageRes, unreadRes] = await Promise.all([
      fetchMessages({ page: 1, page_size: PAGE_SIZE }),
      // 未读数只是口径行的增强信息：它失败不该把整个列表一起判死。
      fetchUnreadCount().catch((err) => {
        console.warn("[messages] 未读数加载失败", err?.message || err)
        return null
      })
    ])
    if (seq !== loadSeq) return false
    const list = Array.isArray(messageRes?.data?.list)
      ? messageRes.data.list
      : Array.isArray(messageRes?.data)
        ? messageRes.data
        : []
    messages.value = list
    pageNo.value = 1
    // total 是接口口径；接口缺字段时才退回当前条数，避免显示 0 条。
    total.value = Number(messageRes?.data?.total) || list.length
    if (unreadRes) unread.value = Number(unreadRes?.data?.total) || 0
    hasLoaded.value = true
    loadError.value = ""
    return true
  } catch (err) {
    if (seq !== loadSeq) return false
    console.warn("[messages] 加载失败", err?.message || err)
    // 不清空已有列表：清空会把「刷新失败」伪装成「没有消息」。
    if (!keepView || messages.value.length === 0) {
      messages.value = []
      total.value = 0
    }
    loadError.value = "消息加载失败"
    return false
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
  }
}

async function loadMore() {
  if (!hasLoaded.value || loading.value || loadingMore.value || retrying.value) return
  if (loadError.value || !hasMore.value) return
  const seq = loadSeq
  loadingMore.value = true
  try {
    const next = pageNo.value + 1
    const res = await fetchMessages({ page: next, page_size: PAGE_SIZE })
    if (seq !== loadSeq) return
    const list = Array.isArray(res?.data?.list) ? res.data.list : []
    // 翻页期间若有新消息插入，offset 分页会重复返回同一条；按 id 去重。
    const seen = new Set(messages.value.map(item => item.id))
    messages.value = messages.value.concat(list.filter(item => !seen.has(item.id)))
    pageNo.value = next
    const nextTotal = Number(res?.data?.total)
    if (list.length === 0) {
      // 空页 = 已经到底。若此前 total 偏大（消息被删），不能让它触发无限触底。
      total.value = messages.value.length
    } else if (!Number.isNaN(nextTotal)) {
      total.value = Math.max(nextTotal, messages.value.length)
    }
  } catch (err) {
    if (seq !== loadSeq) return
    // http 层已 toast；保留已加载内容，下一次触底还能再试。
    console.warn("[messages] 加载更多失败", err?.message || err)
  } finally {
    loadingMore.value = false
  }
}

async function handleRetry() {
  if (retrying.value || loading.value) return
  retrying.value = true
  const startedAt = Date.now()
  const ok = await loadFirstPage({ keepView: true })
  const remain = RETRY_MIN_MS - (Date.now() - startedAt)
  if (remain > 0) await delay(remain)
  retrying.value = false
  uni.showToast({
    title: ok ? "已重新加载" : "仍无法加载，请稍后再试",
    icon: "none"
  })
}

async function handleEmptyRefresh() {
  if (refreshingEmpty.value || loading.value) return
  refreshingEmpty.value = true
  const ok = await loadFirstPage({ keepView: true })
  refreshingEmpty.value = false
  uni.showToast({
    title: ok ? "已刷新" : "仍无法加载，请稍后再试",
    icon: "none"
  })
}

onLoad(() => {
  loadFirstPage()
})
// 从详情返回后未读数可能变了；onLoad 不会再跑，静默回刷第一页。
onShow(() => {
  if (hasLoaded.value && !loading.value && !retrying.value) {
    loadFirstPage({ keepView: true })
  }
})
onPullDownRefresh(() => {
  loadFirstPage({ keepView: true })
})
onReachBottom(loadMore)
</script>

<template>
  <PageShell bottom-space="48rpx" max-width="430px" flush-top>
    <!-- ===== 口径行：未读 + 总数（total 取接口） ===== -->
    <view v-if="showSummary" class="inbox-bar">
      <view class="inbox-unread" :class="{ 'is-clear': unread === 0 }">
        <view v-if="unread > 0" class="inbox-dot" />
        <text class="inbox-unread-text">{{ unreadText }}</text>
      </view>
      <text class="inbox-total">{{ totalText }}</text>
    </view>
    <view v-if="showSummary" class="inbox-rule" />

    <!-- ===== 失败：错误卡叠在列表上方，列表照常可点 ===== -->
    <view v-if="showErrorCard" class="error-card">
      <view class="error-badge">
        <text class="error-glyph">{{ ICON.alert }}</text>
      </view>
      <view class="error-copy">
        <text class="error-title">消息加载失败</text>
        <text class="error-desc">下拉刷新或点击重试</text>
      </view>
      <view class="retry-btn" :class="{ 'is-busy': retrying }" @click="handleRetry">
        <text class="retry-text">{{ retrying ? "重试中…" : "重试" }}</text>
      </view>
    </view>

    <!-- ===== 消息列表：今天 / 昨天 / 更早 ===== -->
    <view v-if="showGroups" class="inbox-groups">
      <view v-for="group in groups" :key="group.key" class="msg-group">
        <text class="group-title">{{ group.label }}</text>
        <view class="msg-list">
          <view
            v-for="item in group.items"
            :key="item.id"
            class="msg-card"
            :class="{ 'is-read': item.is_read }"
            @click="openMessage(item)"
          >
            <view class="msg-tile" :class="'tone-' + typeMetaOf(item).tone">
              <text class="msg-tile-glyph">{{ typeMetaOf(item).icon }}</text>
            </view>
            <view class="msg-body">
              <view class="msg-heading">
                <text class="msg-title">{{ item.title }}</text>
                <view v-if="!item.is_read" class="msg-dot" />
              </view>
              <text class="msg-summary">{{ item.content }}</text>
              <view class="msg-meta">
                <text class="msg-type">{{ typeLabelOf(item) }}</text>
                <text class="msg-time">{{ messageTimeText(item.created_at) }}</text>
              </view>
            </view>
            <view class="msg-tail">
              <text v-if="isInternalPath(item.action_path)" class="msg-chevron">
                {{ ICON.chevron }}
              </text>
            </view>
          </view>
        </view>
      </view>

      <view v-if="loadingMore || showNoMore" class="list-foot">
        <text class="list-foot-text">{{ loadingMore ? "加载中…" : "没有更多了" }}</text>
      </view>
    </view>

    <!-- ===== 空态 ===== -->
    <view v-if="showEmpty" class="empty-card">
      <view class="empty-badge">
        <text class="empty-glyph">{{ ICON.inbox }}</text>
      </view>
      <text class="empty-title">暂无消息</text>
      <text class="empty-desc">老师发布通知、审批结果和接送提醒后会出现在这里</text>
      <view class="refresh-btn" :class="{ 'is-busy': refreshingEmpty }" @click="handleEmptyRefresh">
        <text class="refresh-btn-text">{{ refreshingEmpty ? "刷新中…" : "刷新" }}</text>
      </view>
    </view>

    <!-- ===== 加载：口径行短骨架 + 4 张同构骨架卡 ===== -->
    <view v-if="showSkeleton" class="skel-wrap">
      <view class="skel-head">
        <view class="skel-line skel-summary" />
      </view>
      <view class="skel-list">
        <view v-for="i in 4" :key="i" class="skel-card">
          <view class="skel-tile" />
          <view class="skel-lines">
            <view class="skel-line skel-title" />
            <view class="skel-line skel-body" />
            <view class="skel-line skel-meta" />
          </view>
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 门口的信箱：白卡 + 冷雾底；暖色只落在未读圆点、成长/任务图标位
// 与空态主按钮。已读降灰但卡片仍是白底，避免列表出现灰块。
// ============================================================

// ---- 口径行 -------------------------------------------------
.inbox-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  padding: 26rpx 0 22rpx;
}

.inbox-unread {
  display: flex;
  align-items: center;
  gap: 14rpx;
  min-width: 0;
  // 白底上 $color-brand-primary 只有 2.8:1，深一档才过 AA（与通知列表同口径）。
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.inbox-unread.is-clear {
  color: $color-text-muted;
  font-weight: $typography-font-weight-medium;
}

.inbox-unread-text {
  min-width: 0;
  overflow-wrap: break-word;
}

.inbox-dot {
  flex: none;
  width: 12rpx;
  height: 12rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.inbox-total {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.inbox-rule {
  height: 1rpx;
  background: $color-border-hairline;
}

// ---- 日期分组 ------------------------------------------------
.inbox-groups {
  display: flex;
  flex-direction: column;
  gap: 28rpx;
  padding-top: $spacing-lg;
}

.group-title {
  display: block;
  margin-bottom: 20rpx;
  // 与卡片正文左缘对齐：卡片内边距 24 + 图标位 80 + 间距 20。
  padding-left: 124rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  letter-spacing: 0.02em;
}

.msg-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

// ---- 消息卡 --------------------------------------------------
.msg-card {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  padding: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  transition:
    transform $animation-duration-fast $animation-easing-default,
    background-color $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.99);
    background: $color-bg-page;
  }
}

.msg-tile {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  transition:
    background-color $animation-duration-base $animation-easing-default,
    color $animation-duration-base $animation-easing-default;
}

.msg-tile-glyph {
  color: $color-text-hint;
  font-family: remixicon;
  font-size: 48rpx;
  line-height: 1;
  transition: color $animation-duration-base $animation-easing-default;
}

.tone-notice {
  background: $color-semantic-info-bg;

  .msg-tile-glyph {
    color: $color-semantic-info;
  }
}

.tone-leave {
  background: $color-semantic-warning-bg;

  .msg-tile-glyph {
    color: $color-semantic-warning;
  }
}

.tone-pickup {
  background: $color-semantic-success-bg;

  .msg-tile-glyph {
    color: $color-semantic-success;
  }
}

.tone-growth {
  background: $color-brand-primary-bg;

  .msg-tile-glyph {
    color: $color-brand-primary;
  }
}

.tone-task {
  background: $color-brand-primary-bg;

  .msg-tile-glyph {
    color: $color-brand-primary-dark;
  }
}

.tone-system {
  background: $color-bg-page;

  .msg-tile-glyph {
    color: $color-text-muted;
  }
}

.msg-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
  padding-top: 2rpx;
}

.msg-heading {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  min-width: 0;
}

.msg-title {
  flex: 0 1 auto;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.45;
  overflow-wrap: break-word;
}

.msg-dot {
  flex: none;
  width: 12rpx;
  height: 12rpx;
  margin-top: 14rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.msg-summary {
  display: -webkit-box;
  overflow: hidden;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.65;
  overflow-wrap: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.msg-meta {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  letter-spacing: 0.01em;
}

.msg-type {
  min-width: 0;
}

.msg-time {
  flex: none;
  font-variant-numeric: tabular-nums;
}

.msg-tail {
  display: flex;
  flex: none;
  align-self: center;
  align-items: center;
  justify-content: center;
  width: 32rpx;
}

.msg-chevron {
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

// 拆过的信：整行降灰，但卡片仍是白底。
// 摘要保持 muted 而不是 hint：hint 压白底只有 2.57:1，旧消息会几乎读不清
// （设计审查 P2-3），降灰交给标题字重、图标位与 meta 行承担。
.msg-card.is-read .msg-title {
  color: $color-text-muted;
  font-weight: $typography-font-weight-medium;
}

.msg-card.is-read .msg-meta {
  color: $color-text-hint;
}

.msg-card.is-read .msg-tile {
  background: $color-bg-page;
}

.msg-card.is-read .msg-tile-glyph {
  color: $color-text-hint;
}

// ---- 失败卡 --------------------------------------------------
.error-card {
  display: flex;
  align-items: center;
  gap: 20rpx;
  margin-top: $spacing-lg;
  padding: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.error-badge {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
}

.error-glyph {
  color: $color-semantic-danger;
  font-family: remixicon;
  font-size: $typography-font-size-3xl;
  line-height: 1;
}

.error-copy {
  flex: 1;
  min-width: 0;
}

.error-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.error-desc {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.retry-btn {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  min-width: 160rpx;
  height: 88rpx;
  padding: 0 $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  transition:
    transform $animation-duration-fast $animation-easing-default,
    background-color $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.98);
  }
}

.retry-btn.is-busy {
  border-color: $color-bg-page;
  background: $color-bg-page;
  box-shadow: none;

  &:active {
    transform: none;
  }
}

.retry-text {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  letter-spacing: 0.02em;
}

.retry-btn.is-busy .retry-text {
  color: $color-text-hint;
  font-weight: $typography-font-weight-medium;
}

// ---- 空态 ----------------------------------------------------
.empty-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 40rpx;
  padding: 56rpx 40rpx 48rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.empty-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.empty-glyph {
  color: $color-text-hint;
  font-family: remixicon;
  font-size: 44rpx;
  line-height: 1;
}

.empty-title {
  margin-top: $spacing-lg;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.empty-desc {
  max-width: 480rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.75;
  overflow-wrap: break-word;
}

.refresh-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 208rpx;
  height: 88rpx;
  margin-top: 36rpx;
  padding: 0 52rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-sm;
  transition:
    transform $animation-duration-fast $animation-easing-default,
    box-shadow $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.98);
    box-shadow: $shadow-md;
  }
}

.refresh-btn.is-busy {
  opacity: 0.82;
}

.refresh-btn-text {
  // 白字压品牌橙不足 4.5:1；与亲子任务主按钮同口径用深色字。
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  letter-spacing: 0.02em;
}

// ---- 骨架 ----------------------------------------------------
.skel-wrap {
  padding-top: 28rpx;
}

.skel-head {
  padding-bottom: $spacing-xl;
}

.skel-line,
.skel-tile {
  background: $color-border-hairline;
  animation: skel-breathe 1.7s $animation-easing-default infinite;
}

.skel-line {
  display: block;
  border-radius: $border-radius-full;
}

.skel-summary {
  width: 192rpx;
  height: 24rpx;
}

.skel-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.skel-card {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  padding: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.skel-tile {
  flex: none;
  width: 80rpx;
  height: 80rpx;
  // 与真实类型位同形状：圆形→圆角会在加载完成时跳变（设计审查 P2-5）。
  border-radius: $border-radius-xl;
}

.skel-lines {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-md;
  min-width: 0;
  padding-top: 6rpx;
}

.skel-title {
  width: 68%;
  height: 24rpx;
}

.skel-body {
  width: 92%;
  height: 18rpx;
}

.skel-meta {
  width: 32%;
  height: 16rpx;
}

// ---- 列表脚注 ------------------------------------------------
.list-foot {
  padding-top: $spacing-lg;
  padding-bottom: $spacing-xs;
  text-align: center;
}

.list-foot-text {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
}

@keyframes skel-breathe {
  0%,
  100% {
    opacity: 0.45;
  }
  50% {
    opacity: 0.9;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skel-line,
  .skel-tile {
    animation: none;
    opacity: 0.6;
  }

  .msg-card,
  .msg-tile,
  .msg-tile-glyph,
  .retry-btn,
  .refresh-btn {
    transition: none;
  }
}
</style>
