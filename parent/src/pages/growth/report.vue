<script setup>
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchGrowthReport } from "@/api/parent-growth"

const report = ref({})
const loading = ref(false)
const error = ref("")
const reportType = ref("weekly")

const types = [
  { key: "weekly", label: "周报" },
  { key: "monthly", label: "月报" },
  { key: "term", label: "学期" }
]
// 后端按 dim_name 的唯一索引返回，中文排序不稳定；这里固定产品口径的维度顺序。
const DIM_ORDER = ["语言", "认知", "社交", "运动", "艺术", "自理", "情绪"]

const typeLabel = computed(() => types.find((t) => t.key === reportType.value)?.label || "报告")
// 正文行与四格统计分属两层：即使没有报告正文，后端也会返回学生、周期与四格，
// 此时应展示纸面（四格有值、正文区块按空数组隐藏），而不是落回「这个周期还没有报告」。
const hasReport = computed(() =>
  Boolean(
    report.value.id ||
    report.value.report_type ||
    report.value.student?.name ||
    report.value.period?.start_date ||
    report.value.attendance
  )
)
const dimensions = computed(() => report.value.dimensions || [])
const highlights = computed(() => report.value.highlights || [])
const concerns = computed(() => report.value.concerns || [])
const teacherComment = computed(() => report.value.teacher_comment || "")
const periodLabel = computed(() => {
  const p = report.value.period || {}
  const start = formatDate(p.start_date)
  const end = formatDate(p.end_date)
  return start && end ? `${start} – ${end}` : "—"
})
const stats = computed(() => {
  const r = report.value
  return [
    { value: formatRate(r.attendance?.rate), label: "出勤率" },
    { value: numOrZero(r.points_earned), label: "获得积分" },
    { value: `${numOrZero(r.tasks_completed)}/${numOrZero(r.tasks_total)}`, label: "完成任务" },
    { value: numOrZero(r.growth_record_count), label: "成长记录" }
  ]
})
const emptyCta = computed(() => {
  if (reportType.value === "weekly") return "查看月报"
  if (reportType.value === "monthly") return "查看学期报告"
  return "返回成长时光"
})

function formatRate(value) {
  if (value === null || value === undefined || value === "") return "0%"
  return typeof value === "number" ? `${value}%` : value
}

// 四格统计缺失/null 一律显示 0：后端无数据时返回 0，旧版本接口缺字段也不能留下空白或「—」。
function numOrZero(value) {
  const n = Number(value)
  return Number.isFinite(n) ? n : 0
}

// 报告周期由 date 决定；页面没有日期选择器，传今天即当前周/月/学期。
function todayParam() {
  const d = new Date()
  const pad = (n) => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function formatDate(value) {
  if (!value) return ""
  const text = String(value).slice(0, 10)
  const [y, m, d] = text.split("-")
  if (!y || !m || !d) return text
  return `${Number(m)}月${Number(d)}日`
}

function toList(value) {
  if (Array.isArray(value)) {
    return value
      .map((item) => (typeof item === "string" ? item : item?.text || item?.content || ""))
      .filter(Boolean)
  }
  if (typeof value !== "string" || !value.trim()) return []
  try {
    const parsed = JSON.parse(value)
    if (Array.isArray(parsed)) return toList(parsed)
  } catch {
    // 旧数据是换行/分号分隔的纯文本，继续走下面的拆分。
  }
  return value.split(/\n|；|;/).map((item) => item.trim()).filter(Boolean)
}

function normalize(data) {
  const raw = data || {}
  // 后端把报告正文放在 report 里；四格统计、student、period 在顶层（契约 04-growth.md）。
  // 先铺顶层再铺正文：有正文时以正文为准，无正文时四格统计仍然可读。
  const r = raw.report || {}
  const period = raw.period || r.period || {
    type: r.report_type,
    start_date: r.period_start,
    end_date: r.period_end
  }
  const dims = raw.dimensions || r.dimensions || []
  return {
    ...raw,
    ...r,
    student: r.student || raw.student || {},
    period,
    dimensions: dims.map((d) => ({
      ...d,
      name: d.name || d.dim_name || "",
      score: Number(d.score) || 0,
      max_score: Number(d.max_score) || 100
    })).sort((a, b) => {
      const ia = DIM_ORDER.indexOf(a.name)
      const ib = DIM_ORDER.indexOf(b.name)
      return (ia < 0 ? 99 : ia) - (ib < 0 ? 99 : ib)
    }),
    highlights: toList(r.highlights),
    concerns: toList(r.concerns),
    teacher_comment: r.teacher_comment || r.content || ""
  }
}

async function loadData() {
  loading.value = true
  error.value = ""
  try {
    await ensureLogin()
    const res = await fetchGrowthReport({ type: reportType.value, date: todayParam() })
    report.value = normalize(res.data)
  } catch (e) {
    report.value = {}
    error.value = e?.message || "成长报告加载失败"
  } finally {
    loading.value = false
  }
}

function switchType(t) {
  if (loading.value || error.value || reportType.value === t.key) return
  reportType.value = t.key
  loadData()
}

function handleEmptyCta() {
  if (reportType.value === "weekly") {
    switchType({ key: "monthly" })
    return
  }
  if (reportType.value === "monthly") {
    switchType({ key: "term" })
    return
  }
  const pages = getCurrentPages()
  if (pages.length > 1) uni.navigateBack()
  else uni.reLaunch({ url: "/pages/growth/index" })
}

function dimPercent(d) {
  const max = Number(d.max_score) || 100
  return Math.max(0, Math.min(100, (Number(d.score) / max) * 100))
}

function dimName(d) {
  return d.name || d.dim_name || ""
}

onLoad(() => loadData(false))
</script>

<template>
  <view class="page">
    <!-- 周期切换 -->
    <view class="segmented">
      <view
        v-for="t in types"
        :key="t.key"
        class="seg-btn"
        :class="{ on: reportType === t.key, disabled: loading || error }"
        @click="switchType(t)"
      >{{ t.label }}</view>
    </view>

    <!-- 加载骨架 -->
    <view v-if="loading" class="report-card skeleton">
      <view class="skel-cover">
        <view class="sk skel-circle" />
        <view class="skel-stack">
          <view class="sk skel-name" />
          <view class="sk skel-period" />
        </view>
      </view>
      <view class="skel-stats">
        <view v-for="i in 4" :key="i" class="skel-stat">
          <view class="sk skel-value" />
          <view class="sk skel-label" />
        </view>
      </view>
      <view class="skeleton-body">
        <view class="sk skel-title" />
        <view class="skel-bars">
          <view v-for="i in 7" :key="i" class="sk skel-bar" />
        </view>
        <view class="sk skel-line" />
        <view class="sk skel-line skel-line--short" />
      </view>
    </view>

    <!-- 失败 -->
    <view v-else-if="error" class="state-panel">
      <view class="state-orb state-orb--danger"><text>!</text></view>
      <text class="state-title">成长报告加载失败</text>
      <text class="state-copy">网络断开，或报告还在生成中</text>
      <button class="state-btn" @click="loadData(false)">重试</button>
    </view>

    <!-- 空态 -->
    <view v-else-if="!hasReport" class="state-panel">
      <view class="state-orb"><text>页</text></view>
      <text class="state-title">这个周期还没有成长报告</text>
      <text class="state-copy">{{ reportType === "weekly" ? "老师通常周五整理周报，先看看月报或学期报告" : reportType === "monthly" ? "这个月的报告还没整理好，先看看学期报告吧" : "学期报告还在整理中，先回成长时光看看其他记录吧" }}</text>
      <button class="state-btn" @click="handleEmptyCta">{{ emptyCta }}</button>
    </view>

    <!-- 报告纸面 -->
    <view v-else class="report-card">
      <view class="sheet-head">
        <view class="head-avatar">
          <image v-if="report.student?.avatar" :src="report.student.avatar" class="head-avatar-img" mode="aspectFill" />
          <text v-else>{{ report.student?.name ? report.student.name[0] : "" }}</text>
        </view>
        <view class="head-copy">
          <view class="head-line">
            <text class="head-name">{{ report.student?.name || "孩子" }}</text>
            <text class="period-tag">{{ typeLabel }}</text>
          </view>
          <text class="head-period">{{ periodLabel }}</text>
        </view>
      </view>

      <view class="stats">
        <view v-for="s in stats" :key="s.label" class="stat">
          <text class="stat-value">{{ s.value }}</text>
          <text class="stat-label">{{ s.label }}</text>
        </view>
      </view>

      <view v-if="dimensions.length" class="rule" />
      <view v-if="dimensions.length" class="block">
        <view class="sec-head">
          <text class="sec-title">能力评估</text>
          <text class="sec-meta">满分 100</text>
        </view>
        <view class="dims">
          <view v-for="(d, i) in dimensions" :key="i" class="dim">
            <view class="dim-track">
              <view class="dim-bar" :style="{ height: dimPercent(d) + '%' }">
                <text class="dim-score">{{ d.score }}</text>
              </view>
            </view>
            <text class="dim-name">{{ dimName(d) }}</text>
          </view>
        </view>
        <text class="dims-note">条形越高表示该维度表现越好</text>
      </view>

      <view v-if="highlights.length" class="rule" />
      <view v-if="highlights.length" class="block">
        <view class="sec-head"><text class="sec-title">成长亮点</text></view>
        <view class="items">
          <view v-for="(item, i) in highlights" :key="i" class="item">
            <view class="dot dot--good" />
            <text class="item-text">{{ item }}</text>
          </view>
        </view>
      </view>

      <view v-if="concerns.length" class="rule" />
      <view v-if="concerns.length" class="block">
        <view class="sec-head"><text class="sec-title">需要关注</text></view>
        <view class="items">
          <view v-for="(item, i) in concerns" :key="i" class="item">
            <view class="dot dot--watch" />
            <text class="item-text">{{ item }}</text>
          </view>
        </view>
      </view>

      <view v-if="teacherComment" class="rule" />
      <view v-if="teacherComment" class="block">
        <view class="sec-head"><text class="sec-title">老师评语</text></view>
        <view class="comment-box">
          <view class="comment-avatar"><text>师</text></view>
          <text class="comment-text">{{ teacherComment }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  padding: $component-page-padding-y $component-page-padding-x calc(56rpx + env(safe-area-inset-bottom));
  background: $color-bg-page;
}

/* 周期切换 */
.segmented {
  display: flex;
  height: 88rpx;
  margin-bottom: $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  overflow: hidden;
}

.seg-btn {
  flex: 1 1 0;
  min-width: 0;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: 500;
}

.seg-btn.on {
  background: $color-brand-primary;
  color: $color-text-on-brand;
  font-weight: 600;
}

.seg-btn.disabled {
  color: $color-text-hint;
}

/* 纸面 */
.report-card {
  width: 100%;
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sheet-head {
  position: relative;
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: 36rpx 32rpx;
  background: $gradient-brand-soft;
  overflow: hidden;
}

.sheet-head::before {
  content: "";
  position: absolute;
  top: -84rpx;
  right: -60rpx;
  width: 264rpx;
  height: 264rpx;
  border: 1rpx solid $color-brand-primary-light;
  border-radius: $border-radius-full;
  opacity: 0.28;
}

.head-avatar {
  position: relative;
  z-index: 1;
  flex: 0 0 112rpx;
  width: 112rpx;
  height: 112rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-3xl;
  font-weight: 600;
  line-height: 1;
  overflow: hidden;
}

.head-avatar-img {
  width: 100%;
  height: 100%;
}

.head-copy {
  position: relative;
  z-index: 1;
  flex: 1;
  min-width: 0;
}

.head-line {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.head-name {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-3xl;
  font-weight: 600;
  line-height: 1.4;
  color: $color-text-primary;
}

.period-tag {
  flex: none;
  min-height: 48rpx;
  padding: 0 20rpx;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: 500;
}

.head-period {
  display: block;
  margin-top: 8rpx;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

/* 统计带 */
.stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  background: $color-bg-card;
}

.stat {
  min-height: 144rpx;
  padding: 28rpx 8rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6rpx;
  text-align: center;
}

.stat + .stat {
  border-left: 1rpx solid $color-border-hairline;
}

.stat-value {
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  line-height: 1.2;
}

.stat-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.35;
}

.rule {
  height: 1rpx;
  margin: 0 $spacing-xl;
  background: $color-border-hairline;
}

.block {
  padding: $spacing-xl;
}

.sec-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: 20rpx;
}

.sec-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: 600;
  line-height: 1.4;
}

.sec-meta {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

/* 7 柱能力评估 */
.dims {
  display: flex;
  align-items: flex-end;
  padding-top: 32rpx;
  gap: 4rpx;
}

.dim {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.dim-track {
  // 7 柱评估的固定轨道，设计稿 16×96px；token 无图表专用尺寸。
  width: 32rpx;
  height: 192rpx;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.dim-bar {
  position: relative;
  width: 100%;
  min-height: 8rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
}

.dim-score {
  position: absolute;
  bottom: calc(100% + 8rpx);
  left: 50%;
  transform: translateX(-50%);
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: 600;
  line-height: 1.2;
}

.dim-name {
  max-width: 100%;
  margin-top: 14rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.35;
  text-align: center;
}

.dims-note {
  display: block;
  margin-top: 24rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
}

/* 亮点 / 关注 */
.items {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.item {
  display: flex;
  align-items: flex-start;
  gap: $spacing-md;
}

.dot {
  flex: none;
  width: 12rpx;
  height: 12rpx;
  margin-top: 10rpx;
  border-radius: $border-radius-full;
}

.dot--good {
  background: $color-semantic-success;
}

.dot--watch {
  background: $color-semantic-warning;
}

.item-text {
  flex: 1;
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.65;
}

/* 老师评语 */
.comment-box {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: $spacing-md;
  padding: 28rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  border-left: 6rpx solid $color-brand-primary-light;
}

.comment-avatar {
  flex: 0 0 64rpx;
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: 600;
}

.comment-text {
  flex: 1;
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.75;
}

/* 状态 */
.state-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 120rpx 32rpx 80rpx;
  text-align: center;
}

.state-orb {
  width: 192rpx;
  height: 192rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 32rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: 56rpx;
  font-weight: 600;
}

.state-orb--danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: 600;
  line-height: 1.4;
}

.state-copy {
  max-width: 560rpx;
  margin-top: 16rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.state-btn {
  min-width: 280rpx;
  min-height: 96rpx;
  margin-top: 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-md;
  font-weight: 600;
  box-shadow: $shadow-md;
}

.state-btn::after {
  border: none;
}

/* 骨架 */
.skeleton {
  overflow: hidden;
}

.skel-cover {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: 36rpx 32rpx;
  background: $color-bg-page;
}

.skel-circle {
  flex: none;
  width: 112rpx;
  height: 112rpx;
  border-radius: $border-radius-full;
}

.skel-stack {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.sk {
  display: block;
  background: $color-border-hairline;
  border-radius: $border-radius-full;
  opacity: 0.45;
  animation: skPulse 1.6s $animation-easing-default infinite;
}

.skel-name {
  width: 36%;
  height: 28rpx;
}

.skel-period {
  width: 62%;
  height: 20rpx;
}

.skel-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.skel-stat {
  min-height: 144rpx;
  padding: 28rpx 8rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
}

.skel-stat + .skel-stat {
  border-left: 1rpx solid $color-border-hairline;
}

.skel-value {
  width: 44%;
  height: 24rpx;
}

.skel-label {
  width: 58%;
  height: 16rpx;
}

.skeleton-body {
  padding: 32rpx;
}

.skel-title {
  width: 26%;
  height: 24rpx;
  margin-bottom: 24rpx;
}

.skel-bars {
  height: 192rpx;
  display: flex;
  align-items: flex-end;
  gap: 8rpx;
  margin-bottom: 32rpx;
}

.skel-bar {
  flex: 1;
  height: 100%;
  border-radius: $border-radius-full;
}

.skel-bar:nth-child(2n) {
  height: 64%;
}

.skel-line {
  width: 100%;
  height: 18rpx;
  margin-top: 16rpx;
}

.skel-line--short {
  width: 64%;
}

@keyframes skPulse {
  0%, 100% { opacity: 0.45; }
  50% { opacity: 0.22; }
}

@media (prefers-reduced-motion: reduce) {
  .sk,
  .spinner {
    animation: none;
  }
}
</style>
