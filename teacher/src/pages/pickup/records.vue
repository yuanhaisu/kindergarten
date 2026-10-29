<script setup>
/**
 * 接送记录（老师端）
 *
 * 一次只查一天：请求始终带 date，page_size=100（后端上限）。不做分页 UI，
 * 超过 100 条是已知 P2。行详情只用列表行字段，不发详情请求。
 */
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchTeacherPickupRecords } from "@/api/teacher-pickup"
import { fetchTeacherClassInfo } from "@/api/teacher-class"
import { addDays, formatClock, formatDateLabel, formatWeekdayShort, todayKey } from "@/utils/time"
import { recordPersonLabel } from "@/utils/relation"
import PageShell from "@/components/PageShell.vue"
import AppButton from "@/components/AppButton.vue"

const PAGE_SIZE = 100

// 本地日期 key：selectedDate 一律由 todayKey()/addDays() 生成。
// 不能用 toISOString().slice(0, 10) —— 本地 00:00–07:59 会算成前一天。
const today = ref(todayKey())
const selectedDate = ref(todayKey())
const records = ref([])
const loading = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const className = ref("")
const showDateSheet = ref(false)
const showDetail = ref(false)
const detail = ref(null)
const dateCounts = ref({})
const countsLoading = ref(false)

let loadSeq = 0
let countSeq = 0

const dateLabel = computed(() => formatDateLabel(selectedDate.value))
const weekLabel = computed(() => formatWeekdayShort(selectedDate.value))
const isToday = computed(() => selectedDate.value === today.value)
const nextDisabled = computed(() => selectedDate.value >= today.value)
const isWeekend = computed(() => weekLabel.value === "周六" || weekLabel.value === "周日")

const summary = computed(() => {
  let pickup = 0
  let dropoff = 0
  for (const record of records.value) {
    const type = Number(record.type)
    if (type === 1) dropoff += 1
    if (type === 2) pickup += 1
  }
  return { total: records.value.length, pickup, dropoff }
})

// type / method 是数字枚举，映射表来自 server/internal/domain/pickup.go：
// type 1=送入 2=接走；method 1=扫码 2=手动。未知值显示 —，不猜。
const rows = computed(() =>
  records.value.map(record => {
    const type = Number(record.type)
    const method = Number(record.method)
    return {
      raw: record,
      id: record.id,
      typeKey: type === 1 ? "dropoff" : type === 2 ? "pickup" : "unknown",
      typeLabel: type === 1 ? "送入" : type === 2 ? "接走" : "—",
      methodKey: method === 1 ? "scan" : method === 2 ? "manual" : "unknown",
      methodLabel: method === 1 ? "扫码" : method === 2 ? "手动" : "—",
      time: formatClock(record.recorded_at),
      studentName: record.student?.name || "—",
      person: type === 2 ? recordPersonLabel(record.pickup_name, record.pickup_relation) : "",
      override: Boolean(record.is_authorized_override),
      abnormal: Boolean(record.is_abnormal),
      abnormalReason: String(record.abnormal_reason || ""),
      remark: String(record.remark || "")
    }
  })
)

const showError = computed(() => Boolean(loadError.value))
const showEmpty = computed(
  () => !loading.value && !showError.value && hasLoaded.value && rows.value.length === 0
)
const showNormal = computed(() => !loading.value && !showError.value && rows.value.length > 0)

const emptyDesc = computed(() =>
  isWeekend.value
    ? `${dateLabel.value} ${weekLabel.value} · 全园休息`
    : "换个日期看看，或下拉刷新"
)

const errorDesc = computed(
  () => `网络异常，没取到 ${dateLabel.value} 的接送记录，请重试`
)

const dateWindow = computed(() =>
  Array.from({ length: 7 }, (_, index) => {
    const iso = addDays(today.value, -index)
    return {
      iso,
      main: formatDateLabel(iso),
      week: formatWeekdayShort(iso),
      isToday: iso === today.value
    }
  })
)

const detailSub = computed(() => {
  if (!detail.value) return ""
  return `${dateLabel.value} ${weekLabel.value} · ${detail.value.time}`
})

const detailTimeText = computed(() => {
  if (!detail.value) return ""
  return `${dateLabel.value} ${weekLabel.value} ${detail.value.time}`
})

const detailAuthorizedText = computed(() => {
  if (!detail.value) return ""
  return detail.value.override ? "例外放行 · 非授权接送人" : "有效授权接送人"
})

async function loadClassName() {
  if (className.value) return
  try {
    const res = await fetchTeacherClassInfo()
    className.value = res.data?.name || ""
  } catch (err) {
    // 班级名只做组头/详情副标题；失败不阻断记录本身
    console.warn("[pickup-records] load class failed:", err?.message || err)
  }
}

async function loadRecords() {
  const seq = ++loadSeq
  const nowKey = todayKey()
  today.value = nowKey
  if (selectedDate.value > nowKey) selectedDate.value = nowKey
  const date = selectedDate.value
  loading.value = true
  loadError.value = ""

  try {
    const res = await fetchTeacherPickupRecords(
      { date, page: 1, page_size: PAGE_SIZE },
      { silent: true }
    )
    if (seq !== loadSeq) return
    records.value = Array.isArray(res.data?.list) ? res.data.list : []
    hasLoaded.value = true
    if (!className.value) loadClassName()
  } catch (err) {
    if (seq !== loadSeq) return
    // 失败态不能保留旧条数：统计条会显示「—」，由模板按 loadError 切换。
    console.warn("[pickup-records] load failed:", err?.message || err)
    loadError.value = "接送记录加载失败"
    records.value = []
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
  }
}

function prevDay() {
  selectedDate.value = addDays(selectedDate.value, -1)
  loadRecords()
}

function nextDay() {
  if (nextDisabled.value) return
  selectedDate.value = addDays(selectedDate.value, 1)
  if (selectedDate.value > today.value) selectedDate.value = today.value
  loadRecords()
}

function goToday() {
  if (isToday.value) {
    loadRecords()
    return
  }
  selectedDate.value = today.value
  loadRecords()
}

function countText(iso) {
  const value = dateCounts.value[iso]
  if (value === null) return "—"
  if (typeof value === "number") return value > 0 ? `${value} 条` : "无记录"
  return countsLoading.value ? "…" : "—"
}

async function loadDateCounts() {
  const seq = ++countSeq
  const dates = dateWindow.value.map(item => item.iso)
  // 选中日的条数可能刚变过，每次开弹层都重算；其余日期用缓存。
  const missing = dates.filter(iso => !(iso in dateCounts.value) || iso === selectedDate.value)
  if (!missing.length) return

  countsLoading.value = true
  const results = await Promise.allSettled(
    missing.map(iso =>
      fetchTeacherPickupRecords(
        { date: iso, page: 1, page_size: PAGE_SIZE },
        { silent: true }
      )
    )
  )
  if (seq !== countSeq) return

  const next = { ...dateCounts.value }
  results.forEach((result, index) => {
    if (result.status !== "fulfilled") {
      // 取数失败显示 —，不能写成「无记录」——那是在替不存在的结论背书。
      next[missing[index]] = null
      return
    }
    const total = Number(result.value?.data?.total)
    if (Number.isFinite(total)) {
      next[missing[index]] = total
    } else {
      const list = result.value?.data?.list
      next[missing[index]] = Array.isArray(list) ? list.length : null
    }
  })
  dateCounts.value = next
  countsLoading.value = false
}

function openDateSheet() {
  showDateSheet.value = true
  loadDateCounts()
}

function closeDateSheet() {
  showDateSheet.value = false
}

function selectDate(iso) {
  showDateSheet.value = false
  if (iso === selectedDate.value) {
    loadRecords()
    return
  }
  selectedDate.value = iso
  loadRecords()
}

function openDetail(row) {
  detail.value = row
  showDetail.value = true
}

function closeDetail() {
  showDetail.value = false
}

function noop() {}

onLoad(() => {
  today.value = todayKey()
  selectedDate.value = today.value
  loadRecords()
})

onPullDownRefresh(loadRecords)
</script>

<template>
  <PageShell>
    <view class="page">
      <!-- 日期卡：账页页眉 -->
      <view class="date-card">
        <view class="date-row">
          <view class="date-arrow" @click="prevDay">
            <text class="icon">&#xea64;</text>
          </view>
          <view class="date-current" @click="openDateSheet">
            <text class="date-icon icon">&#xeb21;</text>
            <text class="date-main">{{ dateLabel }}</text>
            <text class="date-week">{{ weekLabel }}</text>
            <text v-if="isToday" class="today-pill">今天</text>
            <text class="date-caret icon">&#xea4e;</text>
          </view>
          <view class="date-arrow" :class="{ 'is-disabled': nextDisabled }" @click="nextDay">
            <text class="icon">&#xea6e;</text>
          </view>
        </view>

        <!-- 统计条：失败时一律 —，避免「没取到」与具体条数同屏 -->
        <view class="summary">
          <view class="summary-cell">
            <text v-if="showError" class="summary-dash">—</text>
            <view v-else-if="loading" class="summary-skel is-s" />
            <view v-else class="summary-value">
              <text class="summary-num">{{ summary.total }}</text>
              <text class="summary-unit">条</text>
            </view>
            <text class="summary-label">记录</text>
          </view>
          <view class="summary-cell">
            <text v-if="showError" class="summary-dash">—</text>
            <view v-else-if="loading" class="summary-skel is-m" />
            <view v-else class="summary-value">
              <text class="summary-num">{{ summary.pickup }}</text>
              <text class="summary-unit">条</text>
            </view>
            <text class="summary-label">接走</text>
          </view>
          <view class="summary-cell">
            <text v-if="showError" class="summary-dash">—</text>
            <view v-else-if="loading" class="summary-skel is-l" />
            <view v-else class="summary-value">
              <text class="summary-num">{{ summary.dropoff }}</text>
              <text class="summary-unit">条</text>
            </view>
            <text class="summary-label">入园签到</text>
          </view>
        </view>
      </view>

      <!-- 组头 + 时间轴 -->
      <view v-if="showNormal" class="group-head">
        <view class="group-title">
          <text>{{ dateLabel }} {{ weekLabel }}</text>
          <text class="group-sep">·</text>
          <text class="group-count">共 {{ summary.total }} 条</text>
        </view>
        <text v-if="className" class="group-class">{{ className }}</text>
      </view>

      <view v-if="showNormal" class="timeline-card">
        <view
          v-for="row in rows"
          :key="row.id"
          class="tl-row"
          :class="{ 'is-danger': row.abnormal }"
          @click="openDetail(row)"
        >
          <text class="tl-time">{{ row.time }}</text>
          <view class="tl-rail">
            <view class="tl-dot" :class="'is-' + row.typeKey" />
          </view>
          <view class="tl-body">
            <view class="tl-top">
              <view class="tl-avatar"><text>{{ row.studentName.slice(0, 1) }}</text></view>
              <text class="tl-name">{{ row.studentName }}</text>
              <text v-if="row.methodKey === 'scan'" class="tl-method is-scan">
                <text class="icon">&#xf041;</text>
                <text>{{ row.methodLabel }}</text>
              </text>
              <text v-else-if="row.methodKey === 'manual'" class="tl-method is-manual">
                <text class="icon">&#xec86;</text>
                <text>{{ row.methodLabel }}</text>
              </text>
              <text v-else class="tl-method is-unknown">{{ row.methodLabel }}</text>
            </view>
            <view class="tl-meta">
              <text class="tl-type" :class="'is-' + row.typeKey">{{ row.typeLabel }}</text>
              <text v-if="row.person" class="tl-dotsep">·</text>
              <text v-if="row.person" class="tl-person">{{ row.person }}</text>
              <text v-if="row.override" class="tl-flag is-override">例外</text>
              <text v-if="row.abnormal" class="tl-flag is-danger">异常</text>
            </view>
            <text v-if="row.remark" class="tl-remark">
              <text class="tl-remark-label">理由</text>{{ row.remark }}
            </text>
          </view>
        </view>
      </view>

      <!-- 加载：统计骨架 + 4 行时间轴骨架 -->
      <view v-if="loading" class="state-loading">
        <view class="skeleton-card">
          <view v-for="i in 4" :key="i" class="skel-row">
            <view class="skel-time" />
            <view class="skel-rail"><view class="skel-dot" /></view>
            <view class="skel-body">
              <view class="skel-top">
                <view class="skel-avatar" />
                <view class="skel-line is-name" />
                <view class="skel-tag" />
              </view>
              <view class="skel-line is-meta" />
            </view>
          </view>
        </view>
      </view>

      <!-- 空态 -->
      <view v-else-if="showEmpty" class="state-view">
        <view class="state-icon"><text class="icon">&#xecef;</text></view>
        <text class="state-title">这天没有接送记录</text>
        <text class="state-desc">{{ emptyDesc }}</text>
        <view class="state-btn" @click="goToday">回到今天</view>
      </view>

      <!-- 失败态 -->
      <view v-else-if="showError" class="state-view">
        <view class="state-icon is-danger"><text class="icon">&#xeca1;</text></view>
        <text class="state-title">{{ loadError }}</text>
        <text class="state-desc">{{ errorDesc }}</text>
        <view class="state-btn" @click="loadRecords">重试</view>
      </view>
    </view>

    <!-- 日期选择 -->
    <view v-if="showDateSheet" class="sheet-layer">
      <view class="sheet-mask" @click="closeDateSheet" @touchmove.stop.prevent="noop" />
      <view class="sheet">
        <view class="sheet-grab" />
        <view class="sheet-head">
          <view class="sheet-head-main">
            <text class="sheet-title">选择日期</text>
            <text class="sheet-sub">最近 7 天</text>
          </view>
          <view class="sheet-close" @click="closeDateSheet"><text class="icon">&#xeb99;</text></view>
        </view>
        <scroll-view scroll-y class="sheet-body">
          <view
            v-for="item in dateWindow"
            :key="item.iso"
            class="date-option"
            :class="{ 'is-selected': item.iso === selectedDate }"
            @click="selectDate(item.iso)"
          >
            <view class="date-option-main">
              <text class="date-option-day">{{ item.main }}</text>
              <text class="date-option-week">{{ item.week }}</text>
              <text v-if="item.isToday" class="today-pill">今天</text>
            </view>
            <view class="date-option-side">
              <text class="date-option-count">{{ countText(item.iso) }}</text>
              <text v-if="item.iso === selectedDate" class="date-option-check icon">&#xeb7b;</text>
            </view>
          </view>
        </scroll-view>
      </view>
    </view>

    <!-- 记录详情 -->
    <view v-if="showDetail && detail" class="sheet-layer">
      <view class="sheet-mask" @click="closeDetail" @touchmove.stop.prevent="noop" />
      <view class="sheet">
        <view class="sheet-grab" />
        <view class="sheet-head">
          <view class="sheet-head-main">
            <text class="sheet-title">接送记录详情</text>
            <text class="sheet-sub">{{ detailSub }}</text>
          </view>
          <view class="sheet-close" @click="closeDetail"><text class="icon">&#xeb99;</text></view>
        </view>
        <scroll-view scroll-y class="sheet-body">
          <view class="detail-hero">
            <view class="detail-avatar"><text>{{ detail.studentName.slice(0, 1) }}</text></view>
            <view class="detail-hero-main">
              <view class="detail-name-row">
                <text class="detail-name">{{ detail.studentName }}</text>
                <text v-if="detail.override" class="tl-flag is-override">例外</text>
                <text v-if="detail.abnormal" class="tl-flag is-danger">异常</text>
              </view>
              <text class="detail-subline">{{ className || "本班" }} · {{ detail.typeLabel }}</text>
            </view>
            <text class="detail-time">{{ detail.time }}</text>
          </view>

          <view class="detail-list">
            <view class="detail-row">
              <text class="detail-label">接送类型</text>
              <text class="detail-value">{{ detail.typeLabel }}</text>
            </view>
            <view class="detail-row">
              <text class="detail-label">记录时间</text>
              <text class="detail-value">{{ detailTimeText }}</text>
            </view>
            <view v-if="detail.person" class="detail-row">
              <text class="detail-label">接送人</text>
              <text class="detail-value">{{ detail.person }}</text>
            </view>
            <view class="detail-row">
              <text class="detail-label">核验方式</text>
              <text class="detail-value">{{ detail.methodLabel }}</text>
            </view>
            <view v-if="detail.typeKey === 'pickup'" class="detail-row">
              <text class="detail-label">授权状态</text>
              <text class="detail-value">{{ detailAuthorizedText }}</text>
            </view>
            <view v-if="detail.abnormal && detail.abnormalReason" class="detail-row">
              <text class="detail-label">异常备注</text>
              <text class="detail-value">{{ detail.abnormalReason }}</text>
            </view>
            <view v-if="detail.remark" class="detail-row">
              <text class="detail-label">理由</text>
              <text class="detail-value">{{ detail.remark }}</text>
            </view>
          </view>
        </scroll-view>
        <view class="sheet-footer">
          <AppButton type="primary" block @click="closeDetail">关闭</AppButton>
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
.icon {
  font-family: remixicon;
  font-style: normal;
  line-height: 1;
}

.page {
  padding-bottom: $spacing-xl;
}

/* ---------- 日期卡 ---------- */
.date-card {
  padding: $spacing-sm $spacing-sm 0;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.date-row {
  display: flex;
  align-items: center;
  gap: 4rpx;
  height: 96rpx;
  min-width: 0;
}

.date-arrow {
  flex: none;
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-2xl;
}

.date-arrow.is-disabled {
  color: $color-text-hint;
  opacity: 0.4;
}

.date-current {
  flex: 1;
  min-width: 0;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  border-radius: $border-radius-full;
}

.date-icon {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
}

.date-main {
  min-width: 0;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
  overflow-wrap: anywhere;
}

.date-week {
  flex: none;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  line-height: 1;
}

.today-pill {
  flex: none;
  height: 36rpx;
  padding: 0 $spacing-sm;
  display: flex;
  align-items: center;
  border: 1rpx solid $color-brand-primary-light;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.date-caret {
  flex: none;
  color: $color-text-hint;
  font-size: $typography-font-size-sm;
}

/* ---------- 统计条 ---------- */
.summary {
  display: flex;
  padding: 20rpx 20rpx 24rpx;
  border-top: 1rpx solid $color-border-subtle;
}

.summary-cell {
  flex: 1;
  min-width: 0;
}

.summary-cell + .summary-cell {
  padding-left: 20rpx;
  border-left: 1rpx solid $color-border-subtle;
}

.summary-value {
  min-height: 48rpx;
  display: flex;
  align-items: baseline;
  gap: 4rpx;
}

.summary-num {
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-bold;
  color: $color-text-primary;
  line-height: 1;
}

.summary-unit {
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
  line-height: 1;
}

.summary-label {
  display: block;
  margin-top: 6rpx;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
  line-height: 1.3;
}

.summary-skel {
  height: 36rpx;
  margin: 6rpx 0;
  border-radius: $border-radius-sm;
  background: $color-border-subtle;
  animation: skeleton-pulse 1.5s $animation-easing-default infinite;
}

.summary-skel.is-s {
  width: 80rpx;
}

.summary-skel.is-m {
  width: 96rpx;
}

.summary-skel.is-l {
  width: 112rpx;
}

.summary-dash {
  display: block;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-bold;
  color: $color-text-hint;
  line-height: 1;
}

/* ---------- 组头 ---------- */
.group-head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-xs 20rpx;
  margin: 28rpx 4rpx 16rpx;
}

.group-title {
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  color: $color-text-body;
}

.group-sep {
  margin: 0 $spacing-xs;
  color: $color-text-muted;
}

.group-count {
  color: $color-text-muted;
  font-weight: $typography-font-weight-normal;
}

.group-class {
  flex: none;
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

/* ---------- 时间轴 ---------- */
.timeline-card {
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.tl-row {
  position: relative;
  display: grid;
  grid-template-columns: 80rpx 28rpx minmax(0, 1fr);
  column-gap: $spacing-md;
  padding: 0 $spacing-lg;
}

.tl-row + .tl-row {
  border-top: 1rpx solid $color-border-subtle;
}

.tl-row.is-danger {
  background: $color-semantic-danger-bg-subtle;
  box-shadow: inset 6rpx 0 0 $color-semantic-danger;
}

.tl-time {
  padding-top: 32rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  line-height: 1;
  text-align: right;
}

.tl-rail {
  position: relative;
  min-height: 100%;
}

.tl-rail::before {
  content: "";
  position: absolute;
  top: 0;
  bottom: 0;
  left: 50%;
  width: 1rpx;
  margin-left: -1rpx;
  background: $color-border-hairline;
  opacity: 0.6;
}

.tl-row:first-child .tl-rail::before {
  top: 36rpx;
}

.tl-row:last-child .tl-rail::before {
  bottom: calc(100% - 36rpx);
}

.tl-dot {
  position: absolute;
  top: 30rpx;
  left: 50%;
  z-index: 1;
  width: 16rpx;
  height: 16rpx;
  margin-left: -8rpx;
  border-radius: $border-radius-full;
  box-shadow: 0 0 0 6rpx $color-bg-card;
}

.tl-dot.is-pickup {
  background: $color-semantic-success;
}

.tl-dot.is-dropoff {
  background: $color-semantic-info;
}

.tl-dot.is-unknown {
  background: $color-text-hint;
}

.tl-body {
  min-width: 0;
  padding: $spacing-lg 0 26rpx;
}

.tl-top {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10rpx;
  min-width: 0;
}

.tl-avatar {
  flex: none;
  width: 60rpx;
  height: 60rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.tl-name {
  min-width: 0;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
  overflow-wrap: anywhere;
}

.tl-method {
  flex: none;
  margin-left: auto;
  height: 40rpx;
  padding: 0 14rpx;
  display: flex;
  align-items: center;
  gap: 4rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.tl-method .icon {
  font-size: $typography-font-size-xs;
}

.tl-method.is-scan {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.tl-method.is-manual {
  border: 1rpx solid $color-border-hairline;
  color: $color-text-muted;
}

.tl-method.is-unknown {
  border: 1rpx solid $color-border-hairline;
  color: $color-text-hint;
}

.tl-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10rpx;
  margin-top: 10rpx;
  font-size: $typography-font-size-sm;
  line-height: 1.45;
}

.tl-type {
  font-weight: $typography-font-weight-semibold;
}

.tl-type.is-pickup {
  color: $color-semantic-success;
}

.tl-type.is-dropoff {
  color: $color-semantic-info;
}

.tl-type.is-unknown {
  color: $color-text-hint;
}

.tl-dotsep {
  color: $color-text-hint;
}

.tl-person {
  min-width: 0;
  color: $color-text-body;
  overflow-wrap: anywhere;
}

.tl-flag {
  flex: none;
  height: 34rpx;
  padding: 0 $spacing-sm;
  display: flex;
  align-items: center;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.tl-flag.is-override {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
}

.tl-flag.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger-strong;
}

.tl-remark {
  display: block;
  margin-top: 10rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.tl-remark-label {
  margin-right: $spacing-xs;
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

/* ---------- 加载骨架 ---------- */
.state-loading {
  margin-top: 24rpx;
}

.skeleton-card {
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.skel-row {
  position: relative;
  display: grid;
  grid-template-columns: 80rpx 28rpx minmax(0, 1fr);
  column-gap: $spacing-md;
  padding: 0 $spacing-lg;
}

.skel-row + .skel-row {
  border-top: 1rpx solid $color-border-subtle;
}

.skel-time {
  width: 64rpx;
  height: 24rpx;
  justify-self: end;
  margin-top: 34rpx;
  border-radius: $border-radius-sm;
  background: $color-border-subtle;
  animation: skeleton-pulse 1.5s $animation-easing-default infinite;
}

.skel-rail {
  position: relative;
}

.skel-rail::before {
  content: "";
  position: absolute;
  top: 0;
  bottom: 0;
  left: 50%;
  width: 1rpx;
  margin-left: -1rpx;
  background: $color-border-hairline;
  opacity: 0.6;
}

.skel-dot {
  position: absolute;
  top: 30rpx;
  left: 50%;
  width: 16rpx;
  height: 16rpx;
  margin-left: -8rpx;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  box-shadow: 0 0 0 6rpx $color-bg-card;
}

.skel-body {
  min-width: 0;
  padding: $spacing-lg 0 26rpx;
}

.skel-top {
  display: flex;
  align-items: center;
  gap: 10rpx;
  min-width: 0;
}

.skel-avatar {
  flex: none;
  width: 60rpx;
  height: 60rpx;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  animation: skeleton-pulse 1.5s $animation-easing-default infinite;
}

.skel-line {
  height: 20rpx;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  animation: skeleton-pulse 1.5s $animation-easing-default infinite;
}

.skel-line.is-name {
  width: 128rpx;
  height: 22rpx;
}

.skel-line.is-meta {
  width: 300rpx;
  margin-top: 18rpx;
}

.skel-tag {
  flex: none;
  width: 84rpx;
  height: 36rpx;
  margin-left: auto;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  animation: skeleton-pulse 1.5s $animation-easing-default infinite;
}

/* ---------- 空 / 失败 ---------- */
.state-view {
  min-height: 48vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $spacing-lg;
  padding: 80rpx $spacing-xl;
  text-align: center;
}

.state-icon {
  width: 112rpx;
  height: 112rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-3xl;
}

.state-icon.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.state-title {
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
}

.state-desc {
  max-width: 540rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  line-height: 1.6;
}

.state-btn {
  min-height: 88rpx;
  padding: 0 $spacing-xl;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

/* ---------- 弹层 ---------- */
.sheet-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 100;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.sheet-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: $color-text-primary;
  opacity: 0.44;
}

.sheet {
  position: relative;
  z-index: 1;
  width: 100%;
  max-height: 82vh;
  padding-bottom: env(safe-area-inset-bottom);
  display: flex;
  flex-direction: column;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.sheet-grab {
  flex: none;
  width: 72rpx;
  height: 8rpx;
  margin: $spacing-md auto 0;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
  opacity: 0.7;
}

.sheet-head {
  flex: none;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: $spacing-md;
  padding: $spacing-lg $spacing-xl $spacing-md;
}

.sheet-head-main {
  min-width: 0;
}

.sheet-title {
  display: block;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
}

.sheet-sub {
  display: block;
  margin-top: 4rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  line-height: 1.35;
}

.sheet-close {
  flex: none;
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: $color-text-body;
  font-size: $typography-font-size-lg;
}

.sheet-body {
  flex: 1 1 auto;
  min-height: 0;
  max-height: 60vh;
  padding: 0 $spacing-xl $spacing-lg;
  box-sizing: border-box;
}

.sheet-footer {
  flex: none;
  padding: $spacing-md $spacing-xl $spacing-xl;
  border-top: 1rpx solid $color-border-hairline;
}

/* ---------- 日期弹层 ---------- */
.date-option {
  min-height: 96rpx;
  padding: $spacing-md 20rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  border-radius: $border-radius-lg;
}

.date-option + .date-option {
  border-top: 1rpx solid $color-border-subtle;
}

.date-option.is-selected {
  background: $color-brand-primary-bg;
}

.date-option-main {
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: $spacing-sm;
}

.date-option-day {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
}

.date-option.is-selected .date-option-day {
  color: $color-brand-primary-dark;
}

.date-option-week {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.date-option-side {
  flex: none;
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.date-option-count {
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

.date-option-check {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
}

/* ---------- 记录详情 ---------- */
.detail-hero {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  padding-bottom: 28rpx;
}

.detail-avatar {
  flex: none;
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.detail-hero-main {
  flex: 1;
  min-width: 0;
}

.detail-name-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-sm;
}

.detail-name {
  min-width: 0;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
  overflow-wrap: anywhere;
}

.detail-subline {
  display: block;
  margin-top: 6rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  line-height: 1.35;
}

.detail-time {
  flex: none;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-bold;
  color: $color-text-primary;
  line-height: 1.25;
}

.detail-list {
  border-top: 1rpx solid $color-border-subtle;
}

.detail-row {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  padding: 20rpx 0;
}

.detail-row + .detail-row {
  border-top: 1rpx solid $color-border-subtle;
}

.detail-label {
  flex: none;
  width: 152rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  line-height: 1.5;
}

.detail-value {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-base;
  color: $color-text-primary;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

@keyframes skeleton-pulse {
  0%,
  100% {
    opacity: 0.55;
  }
  50% {
    opacity: 1;
  }
}
</style>
