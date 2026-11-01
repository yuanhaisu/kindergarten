<script setup>
/**
 * 班级相册 · 老师发布的班级照片
 *
 * 只读。相册是否已发布、哪些照片包含我的孩子、班内共享撤回都由服务端过滤，
 * 列表只消费 visible_photo_count / cover_urls；端上不拿 photo_count 二次猜测，
 * 也不做任何前端二次过滤（openspec parent-class-album）。
 */
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchParentClassAlbums } from "@/api/parent-class-album"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { formatDateLabel } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  gallery: "\u{eda5}", // gallery-line
  alert: "\u{eca1}", // error-warning-line
  right: "\u{ea6e}" // arrow-right-s-line
}

const PAGE_SIZE = 20

// 服务端 category_label 是唯一文案来源；这张表只用于旧后端缺字段时兜底。
const CATEGORY_LABELS = {
  activity: "活动",
  daily: "日常",
  work: "作品",
  festival: "节日"
}

const CATEGORY_TONES = {
  activity: "tag--activity",
  daily: "tag--daily",
  work: "tag--work",
  festival: "tag--festival"
}

const albums = ref([])
const total = ref(0)
const loading = ref(true)
const loadingMore = ref(false)
// 首次成功返回前不把空列表当「还没有相册」——错误态必须先于空态判定。
const hasLoaded = ref(false)
const loadError = ref("")
const exhausted = ref(false)
const retrying = ref(false)
const studentId = ref(getCurrentStudentId())
// 全班视图开关只认列表响应的 classwide_enabled；地址/本地配置都不能替代它。
// 开关默认关闭，因此首次响应前不渲染切换。
const classwideEnabled = ref(false)
const scope = ref("mine")
// 下一次「加载更多」的页码：只在成功追加后自增，删除相册导致页大小变化时
// 不会因为按条数反推页码而漏页。
const nextPage = ref(2)

// 只允许最后一次请求写数据：下拉刷新与加载更多可能并发。
let loadSeq = 0

const viewState = computed(() => {
  if (loading.value && !hasLoaded.value) return "loading"
  if (loadError.value && !hasLoaded.value) return "error"
  if (hasLoaded.value && albums.value.length === 0) return "empty"
  return "normal"
})

// 全班视图下「空」意味着本班没有可共享给其他家长的照片，不是「老师还没发布」。
const emptyTitle = computed(() =>
  scope.value === "all" ? "本班暂无可见照片" : "还没有班级相册"
)

const emptyDesc = computed(() =>
  scope.value === "all"
    ? "老师发布新照片后会出现在这里"
    : "老师发布班级相册后会出现在这里"
)

const hasMore = computed(
  () => hasLoaded.value && !exhausted.value && albums.value.length < total.value
)

function normalizeCount(value) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? n : 0
}

// 封面只认 cover_urls（最多 3 张，且服务端只取可见照片）；少于 3 张时
// 按实际张数排布，没有封面就留一个占位格，不重复裁同一张图。
function coverTilesOf(album) {
  const urls = Array.isArray(album?.cover_urls)
    ? album.cover_urls.map(url => String(url || "").trim()).filter(Boolean)
    : []
  return urls.slice(0, 3)
}

function normalizeAlbum(album) {
  return {
    ...album,
    visible_photo_count: normalizeCount(album?.visible_photo_count),
    coverTiles: coverTilesOf(album)
  }
}

function categoryText(album) {
  return album?.category_label || CATEGORY_LABELS[album?.category] || "相册"
}

function tagClass(category) {
  return CATEGORY_TONES[category] || "tag--neutral"
}

// 存储无当前孩子时自愈一次（与成长相册/首页同入口），再显式传 student_id，
// 避免多孩家庭把另一个孩子的可见范围混进来。
async function ensureStudent() {
  if (studentId.value) return
  try {
    const res = await fetchParentHomeOverview(null)
    syncCurrentStudentFromOverview(res.data)
    studentId.value = getCurrentStudentId()
  } catch (err) {
    // 失败不阻断列表：服务端会回退到第一个孩子，只是多孩口径可能不精确。
    console.warn("[class-album] resolve student failed:", err?.message || err)
  }
}

function fetchPage(page) {
  return fetchParentClassAlbums({
    page,
    page_size: PAGE_SIZE,
    student_id: studentId.value || undefined,
    scope: scope.value
  })
}

async function loadAlbums({ skeleton = true } = {}) {
  const seq = ++loadSeq
  if (skeleton) {
    loading.value = true
    loadError.value = ""
  }
  try {
    await ensureLogin()
    await ensureStudent()
    const res = await fetchPage(1)
    if (seq !== loadSeq) return
    const data = res.data || {}
    // 缺字段（旧后端）按关闭处理：只有明确的 true 才放出全班入口。
    const enabled = data.classwide_enabled === true
    classwideEnabled.value = enabled
    if (!enabled && scope.value === "all") {
      // 园所中途关闭全班视图：静默退回「我的孩子」，不停在不可用的视图上。
      scope.value = "mine"
      return loadAlbums({ skeleton })
    }
    albums.value = (Array.isArray(data.list) ? data.list : []).map(normalizeAlbum)
    total.value = normalizeCount(data.total)
    exhausted.value = false
    nextPage.value = 2
    loadError.value = ""
    hasLoaded.value = true
    return true
  } catch (err) {
    if (seq !== loadSeq) return
    // http.js 已弹过全局 toast，这里只负责页面级错误态。
    console.warn("[class-album] load failed:", err?.message || err)
    if (hasLoaded.value && !skeleton) {
      // 下拉刷新失败：保留已有列表，不把整页退回错误态。
      uni.showToast({ title: "刷新失败，请稍后再试", icon: "none" })
    } else {
      albums.value = []
      total.value = 0
      hasLoaded.value = false
      loadError.value = "班级相册加载失败"
    }
    return false
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
  const ok = await loadAlbums()
  retrying.value = false
  if (ok === false) {
    uni.showToast({ title: "仍无法加载，请稍后再试", icon: "none" })
  }
}

// 空态里的刷新：不进入骨架，原地换成「刷新中…」，没有新内容也要给个回执。
async function refreshEmpty() {
  if (retrying.value || loading.value) return
  retrying.value = true
  const ok = await loadAlbums({ skeleton: false })
  retrying.value = false
  if (ok === true) {
    uni.showToast({
      title: albums.value.length ? "已更新" : "暂时还没有新相册",
      icon: "none"
    })
  }
}

// 切换等于换一份数据集：先清空列表/分页/错误态再按新 scope 拉第一页，
// 否则 all 会沿用 mine 的 total 与「加载更多」页码。
function switchScope(next) {
  if (next === scope.value) return
  if (next === "all" && !classwideEnabled.value) return
  scope.value = next
  // 丢弃在途的「加载更多」：它拿的是旧 scope 的页码。
  loadSeq += 1
  albums.value = []
  total.value = 0
  exhausted.value = false
  nextPage.value = 2
  hasLoaded.value = false
  loadError.value = ""
  loadingMore.value = false
  loadAlbums()
}

async function loadMore() {
  if (loadingMore.value || loading.value || !hasMore.value) return
  loadingMore.value = true
  const seq = loadSeq
  try {
    const page = nextPage.value
    const res = await fetchPage(page)
    if (seq !== loadSeq) return
    const list = Array.isArray(res.data?.list) ? res.data.list : []
    // 契约按 album_date DESC, id DESC 排序；按 id 去重，防止边界页重复。
    const seen = new Set(albums.value.map(item => item.id))
    const fresh = list.filter(item => !seen.has(item.id)).map(normalizeAlbum)
    if (!fresh.length) exhausted.value = true
    albums.value = albums.value.concat(fresh)
    total.value = normalizeCount(res.data?.total) || total.value
    nextPage.value = page + 1
  } catch (err) {
    uni.showToast({ title: "加载更多失败，请重试", icon: "none" })
    console.warn("[class-album] load more failed:", err?.message || err)
  } finally {
    loadingMore.value = false
  }
}

// 详情页不是 tabBar 页（tabBar 只有首页/成长/接送/我的），用 navigateTo。
function openAlbum(album) {
  if (!album?.id) return
  const sid = studentId.value ? `&student_id=${studentId.value}` : ""
  // 详情必须按同一 scope 取照片，否则全班列表点进去只剩自己孩子的几张。
  uni.navigateTo({ url: `/pages/album/detail?id=${album.id}${sid}&scope=${scope.value}` })
}

onLoad(() => {
  loadAlbums()
})

onPullDownRefresh(() => {
  loadAlbums({ skeleton: false })
})
</script>

<template>
  <PageShell max-width="430px" bottom-space="48rpx">
    <!-- 全班视图切换：只有服务端明确下发 classwide_enabled=true 才出现；
         关闭时整个入口不渲染，避免点了才 403。 -->
    <view v-if="classwideEnabled" class="scope-switch">
      <view class="scope-seg">
        <view
          class="scope-btn"
          :class="{ 'is-active': scope === 'mine' }"
          @click="switchScope('mine')"
        >
          <text class="scope-label">我的孩子</text>
        </view>
        <view
          class="scope-btn"
          :class="{ 'is-active': scope === 'all' }"
          @click="switchScope('all')"
        >
          <text class="scope-label">全班</text>
        </view>
      </view>
    </view>

    <!-- 加载：与卡片同形状的骨架，不跳动 -->
    <view v-if="viewState === 'loading'" class="skeleton">
      <view v-for="i in 3" :key="i" class="sk-card">
        <view class="sk-covers">
          <view v-for="j in 3" :key="j" class="sk sk-square" />
        </view>
        <view class="sk-lines">
          <view class="sk sk-line sk-line--title" />
          <view class="sk sk-line sk-line--meta" />
        </view>
      </view>
    </view>

    <!-- 失败：独立错误态 + 重试，不把失败显示成「还没有相册」 -->
    <view v-else-if="viewState === 'error'" class="error-state">
      <view class="error-icon">
        <text class="error-glyph">{{ ICON.alert }}</text>
      </view>
      <text class="error-title">班级相册加载失败</text>
      <text class="error-desc">网络不太稳定，请稍后再试</text>
      <view
        class="state-action"
        :class="{ 'is-busy': retrying }"
        :aria-disabled="retrying ? 'true' : 'false'"
        @click="handleRetry"
      >
        <text class="state-action-text">{{ retrying ? "重试中…" : "重试" }}</text>
      </view>
    </view>

    <!-- 空：本班还没有已发布相册 -->
    <view v-else-if="viewState === 'empty'" class="empty-state">
      <view class="empty-art">
        <text class="empty-glyph">{{ ICON.gallery }}</text>
      </view>
      <text class="empty-title">{{ emptyTitle }}</text>
      <text class="empty-desc">{{ emptyDesc }}</text>
      <view class="link-btn" @click="refreshEmpty">
        <text>{{ retrying ? "刷新中…" : "刷新看看" }}</text>
      </view>
    </view>

    <!-- 正常：卡片列表 + 加载更多 -->
    <view v-else class="album-list">
      <view class="list-head">
        <text class="list-title">全部相册</text>
        <text class="list-meta">共 {{ total }} 本</text>
      </view>

      <view
        v-for="album in albums"
        :key="album.id"
        class="album-card"
        hover-class="album-card--active"
        @click="openAlbum(album)"
      >
        <view class="album-cover" :class="'album-cover--' + album.coverTiles.length">
          <view v-for="(cover, index) in album.coverTiles" :key="index" class="cover-cell">
            <image class="cover-img" :src="cover" mode="aspectFill" />
          </view>
          <view v-if="!album.coverTiles.length" class="cover-cell cover-cell--empty">
            <text class="cover-glyph">{{ ICON.gallery }}</text>
          </view>
        </view>

        <view class="album-info">
          <view class="album-line">
            <text class="album-name">{{ album.title || "班级相册" }}</text>
            <!-- 计数只用 visible_photo_count：photo_count 是老师端全量口径 -->
            <text class="album-count">{{ album.visible_photo_count }} 张照片</text>
          </view>
          <view class="album-line album-meta">
            <text class="album-date">{{ formatDateLabel(album.album_date) }}</text>
            <view class="tag" :class="tagClass(album.category)">
              <text>{{ categoryText(album) }}</text>
            </view>
          </view>
        </view>
      </view>

      <view v-if="hasMore" class="load-more">
        <text class="load-more-hint">已展示 {{ albums.length }} 本 · 共 {{ total }} 本</text>
        <view
          class="load-more-btn"
          :class="{ 'is-busy': loadingMore }"
          @click="loadMore"
        >
          <text>{{ loadingMore ? "加载中…" : "加载更多" }}</text>
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 班级相册 · 可扫读的卡片列表
// 封面承担主要识别；标题、可见张数、日期与分类依次降级。
// ============================================================

// ---- 全班视图切换 --------------------------------------------
// 复用成长页的胶囊分段：选中态用品牌浅底，不引入新色值。
.scope-switch {
  margin-bottom: $spacing-lg;
}

.scope-seg {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: $spacing-xs;
  padding: $spacing-xs;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
}

.scope-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  padding: $spacing-sm $spacing-xs;
  border: 1rpx solid $color-bg-card;
  border-radius: $border-radius-full;
  text-align: center;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.985);
  }
}

.scope-btn.is-active {
  border-color: $color-brand-primary-light;
  background: $color-brand-primary-bg;
  box-shadow: $shadow-sm;
}

.scope-label {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.scope-btn.is-active .scope-label {
  color: $color-text-primary;
}

.album-list {
  padding-bottom: $spacing-lg;
}

.list-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin: 0 4rpx $spacing-md;
}

.list-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.list-meta {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

// ---- 相册卡 --------------------------------------------------
.album-card {
  padding: 20rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast $animation-easing-default;
}

.album-card + .album-card {
  margin-top: $component-card-gap;
}

.album-card--active {
  transform: scale(0.99);
}

.album-cover {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
}

// 有几张封面画几格，不重复裁同一张图。
.album-cover--0,
.album-cover--1 {
  grid-template-columns: minmax(0, 1fr);
}

.album-cover--2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.cover-cell {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 210rpx;
  overflow: hidden;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.cover-img {
  width: 100%;
  height: 100%;
}

.cover-glyph {
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.album-info {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  padding: 20rpx 2rpx 2rpx;
}

.album-line {
  display: flex;
  align-items: flex-start;
  gap: $spacing-md;
  width: 100%;
  min-width: 0;
}

.album-name {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.album-count {
  flex: none;
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.album-meta {
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
}

.album-date {
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.3;
  font-variant-numeric: tabular-nums;
}

.tag {
  display: inline-flex;
  align-items: center;
  flex: none;
  padding: 6rpx $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.tag--activity {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.tag--daily {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.tag--work {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.tag--festival {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
}

.tag--neutral {
  background: $color-bg-page;
  color: $color-text-muted;
}

// ---- 加载更多 ------------------------------------------------
.load-more {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
  margin-top: $spacing-xl;
  text-align: center;
}

.load-more-hint {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.3;
  font-variant-numeric: tabular-nums;
}

.load-more-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  padding: 0 $spacing-lg;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  box-shadow: $shadow-sm;

  &:active {
    transform: scale(0.97);
  }
}

.load-more-btn.is-busy {
  opacity: 0.7;
}

// ---- 加载骨架 ------------------------------------------------
.skeleton {
  padding-bottom: $spacing-lg;
}

.sk-card {
  padding: 20rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sk-card + .sk-card {
  margin-top: $component-card-gap;
}

.sk-covers {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
}

.sk {
  display: block;
  background: $color-border-subtle;
  border-radius: $border-radius-lg;
  animation: sk-breathe 1.6s $animation-easing-default infinite;
}

.sk-square {
  width: 100%;
  height: 210rpx;
}

.sk-lines {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  padding: 20rpx 2rpx 2rpx;
}

.sk-line {
  height: 24rpx;
  border-radius: $border-radius-full;
}

.sk-line--title {
  width: 46%;
}

.sk-line--meta {
  width: 30%;
  height: 20rpx;
}

@keyframes sk-breathe {
  0%,
  100% {
    opacity: 0.5;
  }
  50% {
    opacity: 1;
  }
}

// ---- 空态 / 错误态 -------------------------------------------
.empty-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 70vh;
  padding: $spacing-xl 0 96rpx;
  text-align: center;
}

.empty-art {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 192rpx;
  height: 192rpx;
  margin-bottom: $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.empty-glyph,
.error-glyph {
  font-family: remixicon;
  font-size: 72rpx;
  line-height: 1;
}

.error-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 128rpx;
  height: 128rpx;
  margin-bottom: $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
}

.error-glyph {
  color: $color-semantic-danger;
  font-size: 56rpx;
}

.empty-title,
.error-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.empty-desc,
.error-desc {
  max-width: 520rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.state-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 224rpx;
  min-height: $component-input-height;
  margin-top: $spacing-xl;
  padding: 0 calc(#{$spacing-xl} + #{$spacing-xs});
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.98);
  }
}

// 白字压品牌橙不足 4.5:1，主按钮按设计稿用深色字。
.state-action-text {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.state-action.is-busy {
  background: $color-border-hairline;
  box-shadow: none;

  &:active {
    transform: none;
  }
}

.state-action.is-busy .state-action-text {
  color: $color-text-hint;
}

.link-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: $spacing-md;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  text-decoration: underline;
  text-decoration-color: $color-border-hairline;
  text-underline-offset: 10rpx;
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
    opacity: 0.7;
  }

  .album-card,
  .load-more-btn,
  .state-action {
    transition: none;
  }
}
</style>
