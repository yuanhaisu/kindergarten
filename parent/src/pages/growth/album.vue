<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchGrowthAlbum } from "@/api/parent-growth"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { formatDateLabel, formatDateRange, formatMonthLabel } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  left: "", // arrow-left-s-line
  right: "", // arrow-right-s-line
  close: "", // close-line
  calendar: "", // calendar-line
  play: "", // play-circle-line
  error: "" // error-warning-line
}

function pad(n) {
  return String(n).padStart(2, "0")
}

function keyOf(y, m) {
  return `${y}-${pad(m)}`
}

function currentMonthKey() {
  const now = new Date()
  return keyOf(now.getFullYear(), now.getMonth() + 1)
}

const now = new Date()
const year = ref(now.getFullYear())
const month = ref(now.getMonth() + 1)
const groups = ref([])
const total = ref(0)
const loadedKey = ref("")
// 首帧先按加载态渲染，避免 onLoad 前的空白闪一下
const loading = ref(true)
const loadError = ref("")
const retrying = ref(false)
const studentId = ref(getCurrentStudentId())

// 每次 loadData 自增，快速连点换月时只让最后一次请求写数据。
let loadSeq = 0

const key = computed(() => keyOf(year.value, month.value))
const monthLabel = computed(() => formatMonthLabel(key.value))

// 只有「这个月份的数据已经回来」才渲染内容，否则换月失败会把上个月的照片
// 挂在新月份的标题下面。
const hasMonthData = computed(() => loadedKey.value === key.value)
const showError = computed(() => Boolean(loadError.value) && !hasMonthData.value)
// 2026-10-29 决策：只有月内空态。契约 total 只描述当月，无法判定「从未有过照片」，
// 不把某个月的空伪装成整页空。
const showMonthEmpty = computed(
  () => !showError.value && hasMonthData.value && groups.value.length === 0
)
const showList = computed(
  () => !showError.value && hasMonthData.value && groups.value.length > 0
)

const locked = computed(() => loading.value || showError.value)
const prevDisabled = computed(() => locked.value)
const nextDisabled = computed(() => locked.value || key.value >= currentMonthKey())

const dates = computed(() => groups.value.map(g => g.date).filter(Boolean))
// groups 由服务端按日期倒序返回；范围文案要升序，翻转后求首尾
const rangeText = computed(() => formatDateRange([...dates.value].reverse()))
const showMeta = computed(() => hasMonthData.value && total.value > 0)

function mediaOf(group) {
  return Array.isArray(group.media) ? group.media : []
}

function isVideo(m) {
  return m && m.file_type === "video"
}

// 存储无当前孩子时自愈一次（与首页/成长列表同入口），再显式传 student_id，
// 避免多孩家庭把另一个孩子的照片混进相册。
async function ensureStudent() {
  if (studentId.value) return
  try {
    const res = await fetchParentHomeOverview(null)
    syncCurrentStudentFromOverview(res.data)
    studentId.value = getCurrentStudentId()
  } catch (err) {
    console.warn("[growth-album] resolve student failed:", err?.message || err)
  }
}

async function loadData() {
  const seq = ++loadSeq
  const requestKey = key.value
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    await ensureStudent()
    const res = await fetchGrowthAlbum({
      student_id: studentId.value || undefined,
      year: year.value,
      month: month.value
    })
    if (seq !== loadSeq) return
    const data = res.data || {}
    groups.value = Array.isArray(data.groups) ? data.groups : []
    total.value = Number(data.total) || 0
    loadedKey.value = requestKey
  } catch (err) {
    if (seq !== loadSeq) return
    // http.js 已弹过全局 toast，这里只负责页面级错误态
    console.warn("[growth-album] load failed:", err?.message || err)
    loadError.value = "加载相册失败"
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
  }
}

function goMonth(delta) {
  if (locked.value) return
  const d = new Date(year.value, month.value - 1 + delta, 1)
  year.value = d.getFullYear()
  month.value = d.getMonth() + 1
  loadData()
}

async function handleRetry() {
  if (loading.value) return
  retrying.value = true
  await loadData()
  retrying.value = false
  if (loadError.value) {
    uni.showToast({ title: "仍无法加载，请稍后再试", icon: "none" })
  } else {
    uni.showToast({ title: "已重新加载", icon: "none" })
  }
}

// ---- 预览弹层：作用域是「当天这一组」，与源码的 previewImage 一致 ----
const preview = ref({ open: false, date: "", items: [], index: 0 })
let touchStartX = null

const previewCurrent = computed(() => preview.value.items[preview.value.index] || null)

function openPreview(group, idx) {
  if (locked.value) return
  const items = mediaOf(group)
  if (!items.length) return
  preview.value = {
    open: true,
    date: formatDateLabel(group.date),
    items,
    index: Math.min(Math.max(idx, 0), items.length - 1)
  }
}

function closePreview() {
  preview.value = { open: false, date: "", items: [], index: 0 }
}

function stepPreview(delta) {
  const totalItems = preview.value.items.length
  if (totalItems < 2) return
  preview.value = {
    ...preview.value,
    index: (preview.value.index + delta + totalItems) % totalItems
  }
}

function onStageTouchStart(e) {
  const touch = e?.touches && e.touches[0]
  touchStartX = touch ? touch.clientX : null
}

function onStageTouchEnd(e) {
  if (touchStartX === null) return
  const touch = e?.changedTouches && e.changedTouches[0]
  const dx = touch ? touch.clientX - touchStartX : 0
  touchStartX = null
  if (Math.abs(dx) < 36) return
  stepPreview(dx < 0 ? 1 : -1)
}

onLoad(() => loadData())
// pages.json 未对相册页开启下拉刷新；保留处理器，开启后无需改页面代码
onPullDownRefresh(loadData)
</script>

<template>
  <PageShell max-width="430px" bottom-space="48rpx">
    <!-- 月份条始终可用：只有月内空态（2026-10-29 决策），空月也要能翻走 -->
    <view class="monthbar">
      <view class="mb-row">
        <view class="mb-arrow" :class="{ 'is-disabled': prevDisabled }" @click="goMonth(-1)">
          <text class="mb-icon">{{ ICON.left }}</text>
        </view>
        <text class="mb-title">{{ monthLabel }}</text>
        <view class="mb-arrow" :class="{ 'is-disabled': nextDisabled }" @click="goMonth(1)">
          <text class="mb-icon">{{ ICON.right }}</text>
        </view>
      </view>

      <view v-if="loading" class="mb-meta">
        <view class="sk sk-line" />
        <view class="sk sk-line sk-line--wide" />
      </view>
      <view v-else-if="showMeta" class="mb-meta">
        <text class="mb-count">{{ total }} 张照片</text>
        <text v-if="rangeText" class="mb-dot">·</text>
        <text v-if="rangeText" class="mb-range">{{ rangeText }}</text>
      </view>
    </view>

    <!-- 加载：骨架屏保持网格形状，不跳动 -->
    <view v-if="loading" class="skeleton">
      <view v-for="i in 2" :key="i" class="sk-day">
        <view class="sk sk-date" />
        <view class="sk-grid">
          <view v-for="j in 9" :key="j" class="sk sk-photo" />
        </view>
      </view>
    </view>

    <!-- 错误：独立卡片 + 重试，不把失败显示成「本月暂无照片」 -->
    <view v-else-if="showError" class="state state-error">
      <view class="error-card">
        <view class="error-orb">
          <text class="error-icon">{{ ICON.error }}</text>
        </view>
        <text class="state-title">加载相册失败</text>
        <text class="state-copy">网络不太稳定，请稍后再试</text>
        <view class="btn-primary" :class="{ 'is-busy': retrying }" @click="handleRetry">
          <text>{{ retrying ? "重试中…" : "重试" }}</text>
        </view>
      </view>
    </view>

    <!-- 月内空态：这张月历里没有照片，但月份条仍可翻 -->
    <view v-else-if="showMonthEmpty" class="state state-empty">
      <view class="empty-art">
        <text class="empty-icon">{{ ICON.calendar }}</text>
      </view>
      <text class="state-title">本月暂无照片</text>
      <text class="state-copy">换个月份看看，或等老师上传新的照片</text>
      <view class="link-btn" @click="goMonth(-1)">
        <text>看上个月</text>
      </view>
    </view>

    <!-- 正常：按记录日期分组，三列网格 -->
    <view v-else-if="showList" class="months">
      <view v-for="group in groups" :key="group.date" class="day">
        <view class="day-head">
          <text class="day-date">{{ formatDateLabel(group.date) }}</text>
          <text class="day-count">{{ mediaOf(group).length }} 张</text>
        </view>
        <view class="day-grid">
          <view
            v-for="(m, idx) in mediaOf(group)"
            :key="m.id"
            class="ph"
            @click="openPreview(group, idx)"
          >
            <image class="ph-img" :src="m.thumb_url || m.file_url" mode="aspectFill" />
            <view v-if="isVideo(m)" class="ph-video">
              <text class="ph-play">{{ ICON.play }}</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <!-- 预览弹层：标题带计数，脚注只说切换方式，不重复总数 -->
    <view v-if="preview.open" class="pv">
      <view class="pv-backdrop" @click="closePreview" />
      <view class="pv-sheet">
        <view class="pv-grab" />
        <text class="pv-title">
          {{ preview.date }} · 第 {{ preview.index + 1 }} / {{ preview.items.length }} 张
        </text>

        <view class="pv-stage" @touchstart="onStageTouchStart" @touchend="onStageTouchEnd">
          <view
            class="pv-nav"
            :class="{ 'is-disabled': preview.items.length < 2 }"
            @click="stepPreview(-1)"
          >
            <text class="pv-nav-icon">{{ ICON.left }}</text>
          </view>

          <view class="pv-art">
            <video
              v-if="previewCurrent && isVideo(previewCurrent)"
              class="pv-video"
              :src="previewCurrent.file_url"
              :poster="previewCurrent.thumb_url || ''"
              controls
              autoplay
              object-fit="contain"
            />
            <image
              v-else-if="previewCurrent"
              class="pv-img"
              :src="previewCurrent.file_url || previewCurrent.thumb_url"
              mode="aspectFit"
            />
          </view>

          <view
            class="pv-nav"
            :class="{ 'is-disabled': preview.items.length < 2 }"
            @click="stepPreview(1)"
          >
            <text class="pv-nav-icon">{{ ICON.right }}</text>
          </view>
        </view>

        <view class="pv-foot">
          <text v-if="preview.items.length > 1" class="pv-meta">左右切换查看</text>
          <view class="pv-close" @click="closePreview">
            <text class="pv-close-icon">{{ ICON.close }}</text>
            <text>关闭</text>
          </view>
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 成长相册 · 按月翻看的相册页
// 月份切换是唯一导航轴；日期分组把照片切成可扫读的段落。
// ============================================================

// ---- 月份条 ----------------------------------------------------
.monthbar {
  padding: $spacing-xs 0 $spacing-sm;
  text-align: center;
}

.mb-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
}

.mb-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.94);
  }
}

.mb-arrow.is-disabled {
  border-color: transparent;
  background: $color-border-subtle;
  color: $color-text-hint;
  box-shadow: none;
  pointer-events: none;
}

.mb-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.mb-title {
  min-width: 240rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

.mb-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: center;
  gap: $spacing-xs;
  min-height: 36rpx;
  margin-top: $spacing-sm;
  font-size: $typography-font-size-sm;
  line-height: 1.35;
}

.mb-count {
  color: $color-text-body;
  font-weight: $typography-font-weight-semibold;
  font-variant-numeric: tabular-nums;
}

.mb-dot {
  color: $color-text-hint;
}

.mb-range {
  color: $color-text-muted;
  font-weight: $typography-font-weight-medium;
  font-variant-numeric: tabular-nums;
}

// ---- 日期分组 --------------------------------------------------
.months {
  padding-top: $spacing-xs;
}

.day + .day {
  margin-top: 44rpx;
}

.day-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-lg;
  margin-bottom: $spacing-xs;
  padding: 0 4rpx;
}

.day-date {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.day-count {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
  font-variant-numeric: tabular-nums;
}

.day-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-xs;
}

// aspect-ratio 小程序不支持：padding-top 撑正方形 + 图片绝对定位
.ph {
  position: relative;
  width: 100%;
  padding-top: 100%;
  overflow: hidden;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.985);
  }
}

.ph-img {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

// 视频角标：网格里先看到「这是视频」，点开在弹层里播放
.ph-video {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  // 视频封面上压一层半透明黑，tokens.json 无遮罩色，属允许的单处例外
  background: rgba(0, 0, 0, 0.28);
}

.ph-play {
  color: $color-text-on-brand;
  font-family: remixicon;
  font-size: 56rpx;
  line-height: 1;
}

// ---- 加载骨架 --------------------------------------------------
.skeleton {
  padding: 20rpx 0 $spacing-xl;
}

.sk {
  background: $color-border-subtle;
  animation: sk-breathe 1.6s $animation-easing-default infinite;
}

.sk-line {
  width: 116rpx;
  height: 16rpx;
  border-radius: $border-radius-full;
}

.sk-line--wide {
  width: 236rpx;
}

.sk-day + .sk-day {
  margin-top: 48rpx;
}

.sk-date {
  width: 152rpx;
  height: 24rpx;
  margin-bottom: $spacing-xs;
  border-radius: $border-radius-full;
}

.sk-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-xs;
}

.sk-photo {
  width: 100%;
  height: 216rpx;
  border-radius: $border-radius-lg;
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

// ---- 空态 / 错误态 ---------------------------------------------
.state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
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

.empty-icon {
  font-family: remixicon;
  font-size: 72rpx;
  line-height: 1;
}

.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.state-copy {
  max-width: 560rpx;
  margin-top: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.7;
}

.error-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: 640rpx;
  padding: 48rpx $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.error-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 96rpx;
  height: 96rpx;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.error-icon {
  font-family: remixicon;
  font-size: $typography-font-size-3xl;
  line-height: 1;
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  min-height: $component-input-height;
  margin-top: 40rpx;
  padding: 0 40rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.97);
    box-shadow: $shadow-sm;
  }
}

.btn-primary.is-busy {
  opacity: 0.82;
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
  // 下划线色取 border 语义：token 无 text-decoration-color，透明度过低时可读
  text-decoration-color: $color-border-hairline;
  text-underline-offset: 10rpx;
}

// ---- 预览弹层 --------------------------------------------------
.pv {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 60;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding: $spacing-sm $spacing-md calc(16rpx + env(safe-area-inset-bottom));
}

.pv-backdrop {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  // 遮罩半透明黑：tokens.json 无遮罩色，属允许的单处例外
  background: rgba(0, 0, 0, 0.45);
}

.pv-sheet {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 750rpx;
  padding: $spacing-xl;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.pv-grab {
  width: 72rpx;
  height: 8rpx;
  margin: 0 auto $spacing-md;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
}

.pv-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  text-align: center;
  font-variant-numeric: tabular-nums;
}

.pv-stage {
  display: grid;
  grid-template-columns: 88rpx minmax(0, 1fr) 88rpx;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-lg;
}

.pv-nav {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  box-shadow: $shadow-sm;

  &:active {
    transform: scale(0.94);
  }
}

.pv-nav.is-disabled {
  background: $color-bg-page;
  color: $color-text-hint;
  box-shadow: none;
  pointer-events: none;
}

.pv-nav-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.pv-art {
  position: relative;
  width: 100%;
  padding-top: 100%;
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-page;
}

.pv-img,
.pv-video {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

.pv-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  margin-top: $spacing-lg;
}

.pv-meta {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.pv-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  min-height: 88rpx;
  margin-left: auto;
  padding: 0 36rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;

  &:active {
    transform: scale(0.96);
  }
}

.pv-close-icon {
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
    opacity: 0.7;
  }
}
</style>
