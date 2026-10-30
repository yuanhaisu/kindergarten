<script setup>
/**
 * 班级墙 · 日期当钉子，照片当砖
 *
 * 只展示老师确认过的本班分享。分组按 date（提交日，不是 created_at）。
 * 项目按「不支持吸顶定位」处理（见 _shared/CSS-COMPAT.md），
 * 日期吸顶改为页面顶部 fixed 条：滚过某个日期标题后，固定条显示那一组。
 */
import { computed, nextTick, ref } from "vue"
import { onLoad, onPageScroll, onPullDownRefresh, onReachBottom } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchPublicWall, withStudentScope } from "@/api/parent-task"
import { formatClock, formatDayGroupLabel } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  lock: "", // lock-line
  image: "", // image-line
  alert: "", // error-warning-line
  refresh: "" // refresh-line
}

const PAGE_SIZE = 20

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loadingMore = ref(false)
const retrying = ref(false)
const noMore = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const unbound = ref(false)
// PageShell 是小程序自定义组件，slot 内容位于它的 shadow tree 里；
// 页面级 selectorQuery 跨不过组件边界，必须 .in(PageShell) 在组件内查询。
const shellRef = ref(null)

// 固定日期条：只在某个日期标题滚出视口后出现，避免和行内标题重复。
const fixedVisible = ref(false)
const fixedIndex = ref(0)
let scrollTopPx = 0
let groupTops = []
let loadSeq = 0
let measureTries = 0

const groups = computed(() => {
  const out = []
  let current = ""
  for (const item of list.value) {
    const key = String(item.date || "")
    if (key !== current) {
      current = key
      out.push({ key: key || String(out.length), label: formatDayGroupLabel(key), items: [] })
    }
    out[out.length - 1].items.push(item)
  }
  return out
})

const fixedGroup = computed(() => groups.value[fixedIndex.value] || null)

const isFirstLoading = computed(() => loading.value && list.value.length === 0)
const showFullError = computed(() => !loading.value && Boolean(loadError.value) && list.value.length === 0)
const showStale = computed(() => Boolean(loadError.value) && list.value.length > 0)
const showEmpty = computed(
  () => hasLoaded.value && !loading.value && !loadError.value && list.value.length === 0
)
const showFeed = computed(() => list.value.length > 0)

const errorTitle = computed(() => (unbound.value ? "未绑定孩子信息" : "班级墙暂时打不开"))
const errorDesc = computed(() =>
  unbound.value ? "请先在首页绑定孩子，再查看班级墙" : "网络好像不太稳定，请稍后再试。"
)

function initial(item) {
  const name = String(item?.student?.name || "")
  return name ? Array.from(name)[0] : "同"
}

function childName(item) {
  return item?.student?.name || "同班家长"
}

function previewPhoto(item) {
  if (!item?.media_url) return
  uni.previewImage({ urls: [item.media_url], current: item.media_url })
}

function goTasks() {
  // 正常路径是从任务列表 navigateTo 进来，返回即可；
  // 深链进入时上一页不一定是任务列表，直接重启到列表，保证按钮语义成立。
  const pages = getCurrentPages()
  const prev = pages[pages.length - 2]
  if (prev && prev.route === "pages/tasks/index") {
    uni.navigateBack()
  } else {
    uni.reLaunch({ url: "/pages/tasks/index" })
  }
}

function updateFixedBar() {
  if (!showFeed.value || groupTops.length === 0) {
    fixedVisible.value = false
    return
  }
  let idx = -1
  for (let i = 0; i < groupTops.length; i += 1) {
    if (scrollTopPx >= groupTops[i]) idx = i
    else break
  }
  if (idx < 0) {
    fixedVisible.value = false
    return
  }
  fixedIndex.value = idx
  fixedVisible.value = true
}

function measureGroups() {
  // 等布局稳定后再量：卡片是固定高度（照片区用 padding-top 撑 4:3），
  // 但图片加载与字体渲染仍可能让首帧高度不同。
  setTimeout(() => {
    // uni 的 SelectorQuery.in(component) 会自己取 component.$scope；
    // 传 Vue 组件代理即可，传原生实例反而会被当成 mock 处理。
    const scope = shellRef.value
    const finish = (rects) => {
      // 首帧可能还没渲染出分组（或 ref 尚未就绪）：最多重试两次再接受空结果。
      if (rects.length === 0 && showFeed.value && measureTries < 2) {
        measureTries += 1
        measureGroups()
        return
      }
      measureTries = 0
      groupTops = rects.map((rect) => rect.top + scrollTopPx)
      updateFixedBar()
    }
    let query = null
    try {
      query = scope ? uni.createSelectorQuery().in(scope) : uni.createSelectorQuery()
    } catch (err) {
      // 组件实例不可用时退回页面级查询，不能让吸顶条因为一次测量异常整页报错。
      console.warn("[tasks-wall] measure scope failed:", err?.message || err)
      query = uni.createSelectorQuery()
    }
    query
      .selectAll(".wall-day")
      .boundingClientRect((rects) => {
        const arr = Array.isArray(rects) ? rects : []
        if (arr.length || !scope) {
          finish(arr)
          return
        }
        // 兜底：部分基础库把 slot 节点归到页面树，组件内查不到时回退页面级。
        uni.createSelectorQuery()
          .selectAll(".wall-day")
          .boundingClientRect((pageRects) => finish(Array.isArray(pageRects) ? pageRects : []))
          .exec()
      })
      .exec()
  }, 60)
}

async function load(reset, options = {}) {
  const keepError = Boolean(options.keepError)
  if (!reset && !showFeed.value) return
  if (reset) {
    page.value = 1
    noMore.value = false
    groupTops = []
    fixedVisible.value = false
    if (list.value.length === 0 && !keepError) loading.value = true
  } else {
    if (loadingMore.value || noMore.value || loading.value) return
    loadingMore.value = true
  }
  const seq = ++loadSeq
  loadError.value = ""
  try {
    await ensureLogin()
    const scoped = await withStudentScope((sid) => fetchPublicWall({
      student_id: sid,
      page: page.value,
      page_size: PAGE_SIZE
    }))
    if (seq !== loadSeq) return
    if (!scoped) {
      list.value = []
      total.value = 0
      unbound.value = true
      loadError.value = "未绑定孩子信息"
      return
    }
    unbound.value = false
    const data = scoped.result?.data || {}
    const rows = Array.isArray(data.list) ? data.list : []
    list.value = reset ? rows : list.value.concat(rows)
    total.value = Number(data.total || list.value.length)
    hasLoaded.value = true
    noMore.value = rows.length < PAGE_SIZE || list.value.length >= total.value
    if (!noMore.value) page.value += 1
  } catch (err) {
    if (seq !== loadSeq) return
    console.warn("[tasks-wall] load failed:", err?.message || err)
    // 已有内容时保留列表：清空会把「刷新失败」伪装成「本班没有分享」。
    if (reset && list.value.length === 0) {
      list.value = []
      total.value = 0
      unbound.value = false
      loadError.value = "班级墙加载失败"
    } else {
      loadError.value = "网络开小差了"
    }
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      loadingMore.value = false
      uni.stopPullDownRefresh()
    }
  }
  await nextTick()
  measureGroups()
}

async function handleRetry() {
  if (retrying.value) return
  retrying.value = true
  await load(true, { keepError: true })
  retrying.value = false
  if (!loadError.value) {
    uni.showToast({ title: "已重新加载", icon: "none" })
  } else {
    uni.showToast({ title: "仍无法加载，请稍后再试", icon: "none" })
  }
}

async function handleRefresh() {
  if (loading.value || retrying.value) {
    uni.stopPullDownRefresh()
    return
  }
  await load(true)
}

onLoad(() => {
  load(true)
})
onPageScroll((event) => {
  scrollTopPx = Number(event?.scrollTop) || 0
  updateFixedBar()
})
onPullDownRefresh(handleRefresh)
onReachBottom(() => {
  if (showFeed.value) load(false)
})
</script>

<template>
  <PageShell ref="shellRef" :padded="false" bottom-space="48rpx" max-width="430px">
    <!-- ===== 吸顶日期条：行内标题滚出视口后接棒 ===== -->
    <view v-if="showFeed && fixedVisible && fixedGroup" class="datebar-fixed">
      <view class="datebar-bar" />
      <text class="datebar-label">{{ fixedGroup.label }}</text>
      <text class="datebar-count">{{ fixedGroup.items.length }} 份</text>
      <view class="datebar-line" />
    </view>

    <view class="wall-page">
      <!-- ===== 刷新失败：列表原地保留 ===== -->
      <view v-if="showStale" class="stale-banner" @click="handleRetry">
        <text class="stale-icon">{{ ICON.alert }}</text>
        <text class="stale-text">{{ retrying ? "正在重试…" : "网络开小差了，点击重试" }}</text>
        <text class="stale-arrow">{{ ICON.refresh }}</text>
      </view>

      <!-- ===== 墙头：规则 + 总份数 ===== -->
      <view v-if="showFeed" class="wall-head">
        <view class="wall-rule">
          <text class="wall-lock">{{ ICON.lock }}</text>
          <text class="wall-rule-text">老师确认后才会出现在这里 · 仅本班可见</text>
        </view>
        <text class="wall-count">{{ total }} 份分享</text>
      </view>

      <!-- ===== 按日期分组 ===== -->
      <view v-if="showFeed" class="wall-feed">
        <view v-for="group in groups" :key="group.key" class="wall-day">
          <view class="wall-datebar">
            <view class="datebar-bar" />
            <text class="datebar-label">{{ group.label }}</text>
            <text class="datebar-count">{{ group.items.length }} 份</text>
            <view class="datebar-line" />
          </view>

          <view class="wall-stack">
            <view
              v-for="item in group.items"
              :key="item.checkin_id"
              class="share-card"
              :class="item.media_url ? 'share-card--photo' : 'share-card--note'"
            >
              <!-- 照片卡：照片 4:3（padding-top 撑高，替代不支持的宽高比属性） -->
              <view v-if="item.media_url" class="photo-hit" @click="previewPhoto(item)">
                <image class="wall-photo" :src="item.media_url" mode="aspectFill" />
                <text v-if="item.task_title" class="task-chip">{{ item.task_title }}</text>
              </view>

              <view class="share-body" :class="{ 'share-body--note': !item.media_url }">
                <!-- 一句话卡：暖底 + 引号 + 任务胶囊 -->
                <view v-if="!item.media_url" class="note-topline">
                  <text class="note-quote">“</text>
                  <text v-if="item.task_title" class="task-chip task-chip--note">{{ item.task_title }}</text>
                </view>
                <text
                  v-if="item.content"
                  class="share-copy"
                  :class="{ 'share-copy--note': !item.media_url }"
                >{{ item.content }}</text>
                <view class="share-meta">
                  <image
                    v-if="item.student && item.student.avatar"
                    class="avatar avatar--image"
                    :src="item.student.avatar"
                    mode="aspectFill"
                  />
                  <view v-else class="avatar">
                    <text class="avatar-initial">{{ initial(item) }}</text>
                  </view>
                  <text class="child-name">{{ childName(item) }}</text>
                  <text class="share-time">{{ formatClock(item.created_at) }}</text>
                </view>
              </view>
            </view>
          </view>
        </view>

        <view v-if="loadingMore" class="wall-foot">
          <text class="wall-foot-text">正在加载更多...</text>
        </view>
        <view v-else-if="noMore" class="wall-foot">
          <text class="wall-foot-text">没有更多了</text>
        </view>
      </view>

      <!-- ===== 加载骨架：3 张同构卡 ===== -->
      <view v-if="isFirstLoading" class="skeleton-list">
        <view v-for="i in 3" :key="i" class="skeleton-card">
          <view class="skel-art" />
          <view class="skel-body">
            <view class="skel-line" />
            <view class="skel-line skel-line--short" />
            <view class="skel-meta">
              <view class="skel-avatar" />
              <view class="skel-line skel-line--tiny" />
            </view>
          </view>
        </view>
      </view>

      <!-- ===== 空态 ===== -->
      <view v-if="showEmpty" class="wall-state">
        <view class="state-orb">
          <text class="state-glyph">{{ ICON.image }}</text>
        </view>
        <text class="state-title">本班还没有分享</text>
        <text class="state-desc">家长提交、老师确认之后，会出现在这里</text>
        <view class="state-action" @click="goTasks">
          <text>去任务列表</text>
        </view>
      </view>

      <!-- ===== 错误 / 未绑定 ===== -->
      <view v-if="showFullError" class="wall-state">
        <view class="state-orb state-orb--danger">
          <text class="state-glyph state-glyph--danger">{{ ICON.alert }}</text>
        </view>
        <text class="state-title">{{ errorTitle }}</text>
        <text class="state-desc">{{ errorDesc }}</text>
        <view class="state-action" :class="{ 'is-busy': retrying }" @click="handleRetry">
          <text>{{ retrying ? "正在重试…" : "重试" }}</text>
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 安静的墙面：白 + 冷雾底；暖色只落在规则胶囊底、日期竖条、
// 一句话卡底色与任务胶囊。照片区用 padding-top 撑 4:3。
// ============================================================

.wall-page {
  width: 100%;
  padding-top: 20rpx;
}

// ---- 刷新失败提示条 ------------------------------------------
.stale-banner {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-height: 88rpx;
  margin: 0 $component-page-padding-x 20rpx;
  padding: $spacing-md $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-semantic-danger-bg;

  &:active {
    transform: scale(0.995);
  }
}

.stale-icon {
  flex: none;
  color: $color-semantic-danger-strong;
  font-family: remixicon;
  font-size: $typography-font-size-base;
  line-height: 1;
}

.stale-text {
  flex: 1;
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.45;
}

.stale-arrow {
  flex: none;
  color: $color-text-muted;
  font-family: remixicon;
  font-size: $typography-font-size-base;
  line-height: 1;
}

// ---- 墙头 ----------------------------------------------------
.wall-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  margin: 0 $component-page-padding-x;
  padding: 20rpx 0 $spacing-lg;
}

.wall-rule {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  flex: 1;
  min-width: 0;
}

.wall-lock {
  flex: none;
  color: $color-text-muted;
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  line-height: 1;
}

.wall-rule-text {
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  overflow-wrap: break-word;
}

// 计数用深色字压暖底：品牌深橙在浅底上只有约 3.4:1（审查 P2）。
.wall-count {
  flex: none;
  max-width: 100%;
  padding: $spacing-xs 20rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

// ---- 日期条（行内 + 固定） -----------------------------------
.datebar-fixed {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  gap: $spacing-md;
  width: 100%;
  max-width: 430px;
  margin: 0 auto;
  min-height: 68rpx;
  padding: $spacing-sm $component-page-padding-x;
  background: $color-bg-page;
  box-shadow: $shadow-sm;
}

.wall-datebar {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-height: 68rpx;
  padding: $spacing-sm $component-page-padding-x;
  background: $color-bg-page;
}

.datebar-bar {
  flex: none;
  width: 6rpx;
  height: 28rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.datebar-label {
  flex: none;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.datebar-count {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.datebar-line {
  flex: 1;
  min-width: 32rpx;
  height: 1rpx;
  background: $color-border-subtle;
}

// ---- 分组 ----------------------------------------------------
.wall-feed {
  display: flex;
  flex-direction: column;
  gap: 40rpx;
  width: 100%;
}

.wall-day {
  width: 100%;
}

.wall-stack {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  padding: 0 $component-page-padding-x;
}

// ---- 分享卡 --------------------------------------------------
.share-card {
  width: 100%;
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.share-card--note {
  background: $color-brand-primary-bg;
}

.photo-hit {
  position: relative;
  display: block;
  width: 100%;
  padding-top: 75%;
  background: $color-brand-primary-bg;

  &:active {
    opacity: 0.94;
  }
}

.wall-photo {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

.task-chip {
  display: inline-block;
  max-width: 100%;
  padding: $spacing-xs 20rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  overflow-wrap: break-word;
  box-shadow: $shadow-sm;
}

.photo-hit .task-chip {
  position: absolute;
  left: 20rpx;
  bottom: 20rpx;
  z-index: 1;
  max-width: calc(100% - 40rpx);
}

.task-chip--note {
  max-width: calc(100% - 34rpx);
}

.share-body {
  padding: 20rpx $spacing-lg $spacing-lg;
}

.share-body--note {
  padding: $spacing-lg;
}

.share-copy {
  display: block;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.65;
  overflow-wrap: break-word;
}

.share-copy--note {
  margin-top: $spacing-sm;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.7;
}

.share-meta {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-width: 0;
  margin-top: $spacing-sm;
}

.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
  background: $gradient-brand-soft;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.avatar--image {
  display: block;
  background: $color-bg-page;
}

.share-card--note .avatar {
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.avatar-initial {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.child-name {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
  overflow-wrap: break-word;
}

.share-time {
  flex: none;
  margin-left: auto;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

// ---- 一句话卡 ------------------------------------------------
.note-topline {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  min-width: 0;
}

.note-quote {
  flex: none;
  color: $color-brand-primary;
  font-size: 56rpx;
  font-weight: $typography-font-weight-bold;
  line-height: 0.7;
  transform: translateY(-4rpx);
}

// ---- 骨架 ----------------------------------------------------
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  padding: 0 $component-page-padding-x;
}

.skeleton-card {
  width: 100%;
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.skel-art {
  display: block;
  width: 100%;
  padding-top: 75%;
  background: $color-brand-primary-bg;
  animation: sk-breathe 1.3s $animation-easing-default infinite alternate;
}

.skel-body {
  padding: $spacing-lg;
}

.skel-line {
  display: block;
  width: 100%;
  height: 22rpx;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
}

.skel-line + .skel-line {
  margin-top: $spacing-sm;
}

.skel-line--short {
  width: 62%;
}

.skel-line--tiny {
  width: 92rpx;
  height: 18rpx;
  margin-top: 0;
}

.skel-meta {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-lg;
}

.skel-avatar {
  flex: none;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
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

// ---- 空 / 错误状态 -------------------------------------------
.wall-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  padding: 64rpx $component-page-padding-x 112rpx;
  text-align: center;
}

.state-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 152rpx;
  height: 152rpx;
  margin-bottom: $spacing-md;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.state-orb--danger {
  background: $color-semantic-danger-bg;
}

.state-glyph {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: 64rpx;
  line-height: 1;
}

.state-glyph--danger {
  color: $color-semantic-danger-strong;
}

.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.state-desc {
  max-width: 540rpx;
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
  min-width: 264rpx;
  min-height: 88rpx;
  margin-top: $spacing-xl;
  padding: 0 44rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.97);
  }
}

.state-action text {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
}

.state-action.is-busy {
  opacity: 0.7;
}

// ---- 墙脚 ----------------------------------------------------
.wall-foot {
  display: flex;
  justify-content: center;
  padding: 44rpx 0 60rpx;
}

.wall-foot-text {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  letter-spacing: 0.04em;
  text-align: center;
}

@media (prefers-reduced-motion: reduce) {
  .skel-art {
    animation: none;
  }

  .stale-banner,
  .photo-hit,
  .state-action {
    transition: none;
  }
}
</style>
