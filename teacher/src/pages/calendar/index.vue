<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh, onShow } from "@dcloudio/uni-app"
import { fetchTeacherCalendarEvents } from "@/api/teacher-calendar"
import { fetchDict } from "@/api/dict"
import {
  daysInMonth,
  eventTimeText,
  formatMonthDay,
  formatMonthDayWeek,
  formatYmd,
  pad2,
  sortEvents,
  statusChipText,
  typeColorOf,
  typeLabelOf
} from "@/utils/calendar"
import PageShell from "@/components/PageShell.vue"

const WEEK_DAYS = ["日", "一", "二", "三", "四", "五", "六"]

const today = new Date()
const viewYear = ref(today.getFullYear())
const viewMonth = ref(today.getMonth() + 1)
const selectedDay = ref(today.getDate())
const events = ref([])
const loading = ref(true)
const loadError = ref("")
// 字典只决定图例与类型回退文案；事件自带的 label/color 已由服务端按字典解析
const typeMetas = ref([])
let loadSeq = 0
let skipNextShow = true

const isViewingCurrentMonth = computed(
  () => viewYear.value === today.getFullYear() && viewMonth.value === today.getMonth() + 1
)
const monthLabel = computed(() => `${viewYear.value}年${viewMonth.value}月`)

const monthKey = computed(() => `${viewYear.value}-${pad2(viewMonth.value)}`)
const monthEvents = computed(() =>
  events.value.filter((e) => String(e.event_date || "").startsWith(monthKey.value))
)
const monthCount = computed(() => monthEvents.value.length)
const isMonthEmpty = computed(() => monthCount.value === 0)

// 加载/失败时不报具体数量：数据还没到（或没拿到）时给数字会与骨架屏、错误块互相矛盾。
const monthSubText = computed(() => {
  if (loading.value) return "正在加载…"
  if (loadError.value) return "活动加载失败"
  return isMonthEmpty.value ? "本月暂无活动" : `本月 ${monthCount.value} 个活动`
})
// 失败态月历不可见，切月只会改标题而错误块原样，因此箭头禁用
const navDisabled = computed(() => Boolean(loadError.value))
const viewState = computed(() => {
  if (loading.value) return "loading"
  if (loadError.value) return "error"
  return "normal"
})

const dictMap = computed(() => {
  const map = {}
  typeMetas.value.forEach((item) => {
    map[item.key] = { label: item.label, color: item.color }
  })
  return map
})

// 图例默认跟字典走（内置 5 类，可改名/停用）；字典不可用时退化为本月实际出现的类型，
// label/color 仍只取接口/字典值，不做本地类型表。
const legendItems = computed(() => {
  if (typeMetas.value.length) return typeMetas.value
  const seen = new Map()
  monthEvents.value.forEach((e) => {
    if (!e.event_type || seen.has(e.event_type)) return
    seen.set(e.event_type, {
      key: e.event_type,
      label: e.event_type_label || e.event_type,
      color: e.color || ""
    })
  })
  return [...seen.values()]
})

const calendarCells = computed(() => {
  const y = viewYear.value
  const m = viewMonth.value
  const firstWeekday = new Date(y, m - 1, 1).getDay()
  const currentDays = daysInMonth(y, m)
  const prevDays = daysInMonth(y, m - 1)
  const cells = []
  for (let i = firstWeekday - 1; i >= 0; i -= 1) {
    cells.push({ type: "prev", day: prevDays - i, dots: [] })
  }
  for (let d = 1; d <= currentDays; d += 1) {
    const key = formatYmd(y, m, d)
    const dayEvents = sortEvents(events.value.filter((e) => e.event_date === key))
    cells.push({
      type: "current",
      day: d,
      dots: dayEvents.slice(0, 3).map((e) => typeColorOf(e, dictMap.value))
    })
  }
  const remaining = 42 - cells.length
  for (let d = 1; d <= remaining; d += 1) {
    cells.push({ type: "next", day: d, dots: [] })
  }
  return cells
})

function isToday(cell) {
  return cell.type === "current" && isViewingCurrentMonth.value && cell.day === today.getDate()
}
function isSelected(cell) {
  return cell.type === "current" && cell.day === selectedDay.value
}

const selectedDate = computed(() => formatYmd(viewYear.value, viewMonth.value, selectedDay.value))
const dayEvents = computed(() =>
  sortEvents(events.value.filter((e) => e.event_date === selectedDate.value))
)
const dayDateText = computed(() => formatMonthDayWeek(selectedDate.value))
const dayCountText = computed(() =>
  dayEvents.value.length ? `${dayEvents.value.length} 个活动` : "暂无活动"
)
// 空态标题只带日期（不带周几）：上方 day-date 已经报过「10月26日 周一」，
// 同一张卡里再重复一次周几只是噪音（设计审查 P2-2 的取舍）。
const dayEmptyTitle = computed(() => `${formatMonthDay(selectedDate.value)}暂无活动安排`)

function typeChar(event) {
  return (typeLabelOf(event, dictMap.value) || "事").slice(0, 1)
}
function typeStyle(event) {
  const color = typeColorOf(event, dictMap.value)
  return color ? { backgroundColor: `${color}1F`, color } : {}
}
function typeTextStyle(event) {
  const color = typeColorOf(event, dictMap.value)
  return color ? { color } : {}
}
function dotStyle(color) {
  return color ? { backgroundColor: color } : {}
}
function hasFoot(event) {
  return Boolean(event.location) || event.scope !== "kindergarten"
}

function selectDay(cell) {
  if (cell.type !== "current") return
  selectedDay.value = cell.day
}

async function loadTypeDict() {
  try {
    const res = await fetchDict("calendar_event_type")
    typeMetas.value = (res.data?.items || []).map((item) => {
      let color = ""
      try {
        const extra = typeof item.extra === "string" ? JSON.parse(item.extra) : item.extra
        color = extra?.color || ""
      } catch {
        color = ""
      }
      return { key: item.item_key, label: item.item_label || item.item_key, color }
    })
  } catch {
    // 字典失败不阻塞日历主体；图例会退化为本月实际出现的类型（见 legendItems）
    typeMetas.value = []
  }
}

async function loadEvents({ retry = false } = {}) {
  const seq = ++loadSeq
  loading.value = true
  loadError.value = ""
  const start = `${monthKey.value}-01`
  const end = `${monthKey.value}-${pad2(daysInMonth(viewYear.value, viewMonth.value))}`
  try {
    const res = await fetchTeacherCalendarEvents({ start, end }, { silent: true })
    if (seq !== loadSeq) return
    events.value = res.data?.list || []
    if (retry) uni.showToast({ title: "已重新加载", icon: "none" })
  } catch {
    if (seq !== loadSeq) return
    events.value = []
    loadError.value = "活动加载失败"
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
  }
}

function shiftMonth(delta) {
  if (navDisabled.value) return
  const next = new Date(viewYear.value, viewMonth.value - 1 + delta, 1)
  viewYear.value = next.getFullYear()
  viewMonth.value = next.getMonth() + 1
  const total = daysInMonth(viewYear.value, viewMonth.value)
  if (isViewingCurrentMonth.value) selectedDay.value = today.getDate()
  else if (selectedDay.value > total) selectedDay.value = total
  loadEvents()
}

function retryLoad() {
  loadEvents({ retry: true })
}
function goAdd() {
  uni.navigateTo({ url: "/pages/calendar/create" })
}
function editEvent(event) {
  uni.navigateTo({ url: `/pages/calendar/detail?id=${event.id}` })
}

onLoad(() => {
  loadTypeDict()
  loadEvents()
})
// 从新建/详情返回时刷新：新建或删除后列表要立刻反映，否则会停留在旧数据
onShow(() => {
  if (skipNextShow) {
    skipNextShow = false
    return
  }
  loadEvents()
})
onPullDownRefresh(() => loadEvents())
</script>

<template>
  <PageShell max-width="430px">
    <view class="calendar-page">
      <view class="month-bar">
        <view
          class="month-nav"
          :class="{ 'is-disabled': navDisabled }"
          aria-label="上一月"
          @click="shiftMonth(-1)"
        >
          <text class="icon">&#xea64;</text>
        </view>
        <view class="month-core">
          <text class="month-label">{{ monthLabel }}</text>
          <text class="month-sub">{{ monthSubText }}</text>
        </view>
        <view
          class="month-nav"
          :class="{ 'is-disabled': navDisabled }"
          aria-label="下一月"
          @click="shiftMonth(1)"
        >
          <text class="icon">&#xea6e;</text>
        </view>
        <view class="add-btn" aria-label="添加活动" @click="goAdd">
          <text class="icon add-icon">&#xea13;</text>
          <text class="add-text">添加</text>
        </view>
      </view>

      <template v-if="viewState === 'normal'">
        <view class="cal-card">
          <view class="week-row">
            <text
              v-for="(day, idx) in WEEK_DAYS"
              :key="day"
              class="week-day"
              :class="{ 'is-weekend': idx === 0 || idx === 6 }"
            >{{ day }}</text>
          </view>
          <view class="cal-grid" :class="{ 'is-empty': isMonthEmpty }">
            <view
              v-for="(cell, idx) in calendarCells"
              :key="idx"
              class="cell"
              :class="{
                'is-outside': cell.type !== 'current',
                'is-today': isToday(cell),
                'is-selected': isSelected(cell)
              }"
              @click="selectDay(cell)"
            >
              <text class="cell-num">{{ cell.day }}</text>
              <view class="cell-dots">
                <view
                  v-for="(color, di) in cell.dots"
                  :key="di"
                  class="cell-dot"
                  :class="{ 'is-unknown': !color }"
                  :style="dotStyle(color)"
                />
              </view>
            </view>
          </view>
          <view v-if="legendItems.length" class="cal-legend">
            <view v-for="item in legendItems" :key="item.key" class="legend-item">
              <view
                class="legend-dot"
                :class="{ 'is-unknown': !item.color }"
                :style="dotStyle(item.color)"
              />
              <text class="legend-label">{{ item.label }}</text>
            </view>
          </view>
        </view>

        <view class="day-section">
          <view class="day-head">
            <view class="day-head-main">
              <text class="day-title">当日安排</text>
              <text class="day-date">{{ dayDateText }}</text>
            </view>
            <text class="day-count">{{ dayCountText }}</text>
          </view>

          <view v-if="dayEvents.length" class="event-list">
            <view
              v-for="event in dayEvents"
              :key="event.id"
              class="event-card"
              :class="{ 'has-no-foot': !hasFoot(event) }"
            >
              <view class="event-top">
                <view
                  class="event-icon"
                  :class="{ 'is-unknown': !typeColorOf(event, dictMap) }"
                  :style="typeStyle(event)"
                >
                  <text>{{ typeChar(event) }}</text>
                </view>
                <view class="event-body">
                  <view class="event-title-row">
                    <text class="event-title">{{ event.title }}</text>
                    <text v-if="statusChipText(event.status)" class="chip is-draft">{{ statusChipText(event.status) }}</text>
                    <text v-if="event.scope === 'kindergarten'" class="chip is-garden">全园</text>
                  </view>
                  <view class="event-meta">
                    <text
                      class="event-type"
                      :class="{ 'is-unknown': !typeColorOf(event, dictMap) }"
                      :style="typeTextStyle(event)"
                    >{{ typeLabelOf(event, dictMap) }}</text>
                    <template v-if="eventTimeText(event)">
                      <text class="meta-sep">·</text>
                      <text class="event-time">{{ eventTimeText(event) }}</text>
                    </template>
                  </view>
                </view>
              </view>

              <text v-if="event.description" class="event-desc">{{ event.description }}</text>

              <view v-if="hasFoot(event)" class="event-foot">
                <view v-if="event.location" class="foot-item">
                  <text class="icon foot-icon">&#xef14;</text>
                  <text class="foot-text">{{ event.location }}</text>
                </view>
                <view
                  v-if="event.scope !== 'kindergarten'"
                  class="event-edit"
                  aria-label="编辑活动"
                  @click="editEvent(event)"
                >
                  <text>编辑</text>
                  <text class="icon edit-icon">&#xea6e;</text>
                </view>
              </view>
            </view>
          </view>

          <view v-else class="day-empty">
            <view class="empty-icon"><text class="icon">&#xeb27;</text></view>
            <text class="empty-title">{{ dayEmptyTitle }}</text>
            <text class="empty-desc">可以添加一场活动，家长端会同步看到。</text>
            <view class="empty-btn" @click="goAdd"><text>添加活动</text></view>
          </view>
        </view>
      </template>

      <template v-else-if="viewState === 'loading'">
        <view class="cal-skeleton">
          <view class="sk-week">
            <view v-for="i in 7" :key="i" class="sk-week-cell" />
          </view>
          <view class="sk-grid">
            <view v-for="i in 42" :key="i" class="sk-cell" />
          </view>
          <view class="sk-legend">
            <view v-for="i in 4" :key="i" class="sk-legend-item" />
          </view>
        </view>
        <view class="day-skeleton">
          <view class="sk-head">
            <view class="sk-bar is-title" />
            <view class="sk-bar is-count" />
          </view>
          <view v-for="i in 2" :key="i" class="sk-card">
            <view class="sk-line w60" />
            <view class="sk-line w40" />
            <view class="sk-line w80" />
          </view>
        </view>
      </template>

      <view v-else class="error-block">
        <view class="error-icon"><text class="icon">&#xf2c2;</text></view>
        <text class="error-title">日历加载失败</text>
        <text class="error-desc">网络异常，未获取到本月活动。</text>
        <view class="error-btn" @click="retryLoad"><text>重新加载</text></view>
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

.calendar-page {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  padding-bottom: $spacing-xl;
}

/* ---------- 月份切换条（本页唯一暖色区块） ---------- */
.month-bar {
  display: flex;
  align-items: center;
  gap: 4rpx;
  padding: 10rpx 16rpx 10rpx 8rpx;
  border-radius: $component-card-radius;
  background: $gradient-brand-soft;
  box-shadow: $shadow-sm;
}
.month-nav {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xl;
  color: $color-text-body;
}
.month-nav:active {
  background: $color-bg-card;
}
.month-nav.is-disabled {
  opacity: 0.4;
}
.month-core {
  flex: 1;
  min-width: 0;
  padding-left: 4rpx;
}
.month-label {
  display: block;
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  line-height: 1.25;
  color: $color-text-primary;
}
.month-sub {
  display: block;
  margin-top: 4rpx;
  font-size: $typography-font-size-xs;
  font-weight: 500;
  color: $color-text-muted;
}
.add-btn {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 6rpx;
  min-height: 88rpx;
  padding: 0 28rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-sm;
}
.add-btn:active {
  transform: scale(0.96);
}
.add-icon,
.add-text {
  color: $color-text-on-brand;
  font-size: $typography-font-size-base;
}
.add-text {
  font-weight: 500;
}

/* ---------- 月历卡片 ---------- */
.cal-card {
  padding: 14rpx 14rpx 20rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}
.week-row {
  display: flex;
  margin-bottom: 4rpx;
}
.week-day {
  flex: 1;
  padding: 6rpx 0 10rpx;
  text-align: center;
  font-size: $typography-font-size-xs;
  font-weight: 500;
  color: $color-text-muted;
}
.week-day.is-weekend {
  color: $color-semantic-danger;
}
.cal-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  row-gap: 4rpx;
}
.cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6rpx;
  min-height: 94rpx;
  padding: 6rpx 0 8rpx;
  border-radius: $border-radius-lg;
}
.cell:active {
  background: $color-brand-primary-bg;
}
.cell-num {
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  border: 3rpx solid transparent;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-base;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  color: $color-text-body;
  line-height: 1;
}
.cell-dots {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6rpx;
  height: 10rpx;
}
.cell-dot {
  width: 8rpx;
  height: 8rpx;
  border-radius: $border-radius-full;
}
.cell-dot.is-unknown {
  background: $color-text-hint;
}
.cell.is-outside {
  color: $color-text-hint;
}
.cell.is-outside .cell-num {
  opacity: 0.42;
}
.cell.is-outside .cell-dots {
  visibility: hidden;
}
/* 今天只用品牌色数字；选中才画实心圆——两者同时实心会让用户以为选了两天 */
.cell.is-today:not(.is-selected) .cell-num {
  color: $color-brand-primary-dark;
  font-weight: 700;
}
.cell.is-selected .cell-num {
  background: $color-brand-primary-dark;
  border-color: $color-brand-primary;
  color: $color-text-on-brand;
  font-weight: 700;
}
.cell.is-today.is-selected .cell-num {
  outline: 4rpx solid $color-brand-primary-light;
  outline-offset: 4rpx;
}
.cal-grid.is-empty .cell-dots {
  visibility: hidden;
}

.cal-legend {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10rpx 24rpx;
  margin-top: 16rpx;
  padding-top: 18rpx;
  border-top: 2rpx solid $color-border-subtle;
}
.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 10rpx;
}
.legend-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: $border-radius-full;
}
.legend-dot.is-unknown {
  background: $color-text-hint;
}
.legend-label {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

/* ---------- 当日安排 ---------- */
.day-section {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}
.day-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20rpx;
  padding: 0 4rpx;
}
.day-head-main {
  min-width: 0;
}
.day-title {
  display: block;
  font-size: $typography-font-size-xl;
  font-weight: 600;
  line-height: 1.3;
  color: $color-text-primary;
}
.day-date {
  display: block;
  margin-top: 6rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}
.day-count {
  flex: none;
  margin-top: 2rpx;
  padding: 6rpx 20rpx;
  border: 2rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
  white-space: nowrap;
}

.event-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}
.event-card {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
  padding: 22rpx 28rpx 8rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.event-card.has-no-foot {
  padding-bottom: 26rpx;
}
.event-top {
  display: flex;
  gap: 20rpx;
  min-width: 0;
}
.event-icon {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 76rpx;
  height: 76rpx;
  border-radius: $border-radius-xl;
  font-size: $typography-font-size-xl;
  font-weight: 700;
  line-height: 1;
}
.event-icon.is-unknown {
  background: $color-bg-page;
  color: $color-text-muted;
}
.event-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
}
.event-title-row {
  display: flex;
  align-items: flex-start;
  gap: 10rpx;
  min-width: 0;
}
.event-title {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-md;
  font-weight: 600;
  line-height: 1.4;
  color: $color-text-primary;
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
}
.chip {
  flex: none;
  margin-top: 2rpx;
  padding: 2rpx 16rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1.5;
  white-space: nowrap;
}
.chip.is-draft {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}
.chip.is-garden {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}
.event-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6rpx 12rpx;
  min-width: 0;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}
.event-type {
  font-weight: 500;
}
.event-type.is-unknown {
  color: $color-text-muted;
}
.event-type.is-unknown::before {
  background: $color-text-hint;
}
.event-type::before {
  content: "";
  display: inline-block;
  width: 10rpx;
  height: 10rpx;
  margin-right: 8rpx;
  border-radius: $border-radius-full;
  background: currentColor;
  vertical-align: 1rpx;
}
.meta-sep {
  color: $color-text-hint;
}
.event-time {
  font-variant-numeric: tabular-nums;
}
.event-desc {
  font-size: $typography-font-size-base;
  line-height: 1.5;
  color: $color-text-muted;
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
}
.event-foot {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6rpx 24rpx;
  min-height: 88rpx;
}
.foot-item {
  display: inline-flex;
  align-items: center;
  gap: 6rpx;
  max-width: 100%;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  overflow-wrap: anywhere;
}
.foot-icon {
  flex: none;
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}
.foot-text {
  min-width: 0;
}
.event-edit {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4rpx;
  min-width: 88rpx;
  min-height: 88rpx;
  margin-left: auto;
  padding-left: 28rpx;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  color: $color-brand-primary-dark;
}
.event-edit:active {
  opacity: 0.55;
}
.edit-icon {
  font-size: $typography-font-size-sm;
}

/* ---------- 当日空态 ---------- */
.day-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14rpx;
  padding: 48rpx 36rpx 44rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}
.empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 104rpx;
  height: 104rpx;
  margin-bottom: 4rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-3xl;
}
.empty-title {
  font-size: $typography-font-size-md;
  font-weight: 500;
  color: $color-text-body;
}
.empty-desc {
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-text-muted;
}
.empty-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: 8rpx;
  padding: 0 44rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-md;
  font-weight: 500;
}
.empty-btn:active {
  transform: scale(0.96);
}

/* ---------- 骨架屏 ---------- */
.sk-week {
  display: flex;
  gap: 6rpx;
  margin-bottom: 20rpx;
}
.sk-week-cell {
  flex: 1;
  height: 16rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}
.sk-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 6rpx;
}
.sk-cell {
  height: 88rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  animation: sk-pulse 1.6s ease-in-out infinite;
}
.sk-legend {
  display: flex;
  justify-content: center;
  gap: 20rpx;
  margin-top: 28rpx;
  padding-top: 20rpx;
  border-top: 2rpx solid $color-border-subtle;
}
.sk-legend-item {
  width: 84rpx;
  height: 14rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}
.cal-skeleton {
  padding: 28rpx 24rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.day-skeleton {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}
.sk-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 4rpx;
}
.sk-bar {
  height: 24rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}
.sk-bar.is-title {
  width: 224rpx;
}
.sk-bar.is-count {
  width: 96rpx;
}
.sk-card {
  display: flex;
  flex-direction: column;
  gap: 18rpx;
  padding: 28rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.sk-line {
  height: 20rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  animation: sk-pulse 1.6s ease-in-out infinite;
}
.sk-line.w40 {
  width: 40%;
}
.sk-line.w60 {
  width: 60%;
}
.sk-line.w80 {
  width: 80%;
}
@keyframes sk-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.55;
  }
}

/* ---------- 失败态 ---------- */
.error-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14rpx;
  padding: 56rpx 40rpx 52rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}
.error-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 104rpx;
  height: 104rpx;
  margin-bottom: 4rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-3xl;
}
.error-title {
  font-size: $typography-font-size-md;
  font-weight: 500;
  color: $color-text-primary;
}
.error-desc {
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-text-muted;
}
.error-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: 8rpx;
  padding: 0 44rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-md;
  font-weight: 500;
}
.error-btn:active {
  transform: scale(0.96);
}
</style>
