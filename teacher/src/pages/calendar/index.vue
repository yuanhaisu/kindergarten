<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchTeacherCalendarEvents } from "@/api/teacher-calendar"

const events = ref([])
const loading = ref(false)
const currentDate = ref(new Date())

const year = computed(() => currentDate.value.getFullYear())
const month = computed(() => currentDate.value.getMonth() + 1)
const today = computed(() => new Date().getDate())
const isCurrentMonth = computed(() => { const now = new Date(); return year.value === now.getFullYear() && month.value === now.getMonth() + 1 })
const monthLabel = computed(() => `${year.value}年${month.value}月`)
const weekDays = ["日", "一", "二", "三", "四", "五", "六"]

const calendarDays = computed(() => {
  const y = year.value
  const m = month.value - 1
  const firstDay = new Date(y, m, 1).getDay()
  const daysInMonth = new Date(y, m + 1, 0).getDate()
  const daysInPrevMonth = new Date(y, m, 0).getDate()
  const days = []
  for (let i = firstDay - 1; i >= 0; i--) { days.push({ day: daysInPrevMonth - i, type: "prev" }) }
  for (let i = 1; i <= daysInMonth; i++) { days.push({ day: i, type: "current" }) }
  const remaining = 42 - days.length
  for (let i = 1; i <= remaining && i <= 14; i++) { days.push({ day: i, type: "next" }) }
  return days
})

const selectedDay = ref(new Date().getDate())
function selectDay(day) { selectedDay.value = day }
function isSelected(day) {
  if (day === selectedDay.value) return true
  if (selectedDay.value === today.value && day === today.value && isCurrentMonth.value) return true
  return false
}

const todayEvents = computed(() => {
  const m = String(month.value).padStart(2, "0")
  const d = String(selectedDay.value).padStart(2, "0")
  return events.value.filter(e => e.event_date === `${year.value}-${m}-${d}`)
})

const eventDotColors = { notification: "#3B82F6", activity: "#22C55E", holiday: "#F97316", meeting: "#A855F7", health: "#EF4444" }
const eventTypeIcons = { notification: "通", activity: "活", holiday: "假", meeting: "会", health: "检" }
const eventTypeBgs = { notification: "#DBEAFE", activity: "#DCFCE7", holiday: "#FFF7ED", meeting: "#F3E8FF", health: "#FEE2E2" }

const legendItems = [
  { label: "通知", color: "#3B82F6" }, { label: "活动", color: "#22C55E" },
  { label: "假期", color: "#F97316" }, { label: "家长会", color: "#A855F7" }, { label: "体检", color: "#EF4444" }
]

function getEventsForDay(day) {
  if (day <= 0) return []
  const dateStr = `${year.value}-${String(month.value).padStart(2, "0")}-${String(day).padStart(2, "0")}`
  return events.value.filter(e => e.event_date === dateStr)
}

function getDayDotColors(day) { return getEventsForDay(day).map(e => eventDotColors[e.event_type] || "#3B82F6") }
function isToday(day) { if (!isCurrentMonth.value) return false; return day === today.value }

function prevMonth() { const d = new Date(currentDate.value); d.setMonth(d.getMonth() - 1); currentDate.value = d; loadEvents() }
function nextMonth() { const d = new Date(currentDate.value); d.setMonth(d.getMonth() + 1); currentDate.value = d; loadEvents() }

function goAdd() { uni.navigateTo({ url: "/pages/calendar/create" }) }
function editEvent(event) { uni.navigateTo({ url: `/pages/calendar/detail?id=${event.id}` }) }

async function loadEvents() {
  loading.value = true
  try {
    const y = year.value
    const m = String(month.value).padStart(2, "0")
    const start = `${y}-${m}-01`
    const lastDay = new Date(y, month.value, 0).getDate()
    const end = `${y}-${m}-${String(lastDay).padStart(2, "0")}`
    const res = await fetchTeacherCalendarEvents({ start, end })
    events.value = res.data?.list || []
  } catch (err) {
    events.value = []
    uni.showToast({ title: "加载日历失败", icon: "none" })
  } finally { loading.value = false; uni.stopPullDownRefresh() }
}

onLoad(loadEvents)
onPullDownRefresh(loadEvents)
</script>

<template>
  <view class="page">
    <view class="month-bar">
      <view class="month-arrow" @click="prevMonth"><text class="arrow-text">◂</text></view>
      <text class="month-text">{{ monthLabel }}</text>
      <view class="month-arrow" @click="nextMonth"><text class="arrow-text">▸</text></view>
      <view class="add-btn" @click="goAdd"><text class="add-icon">+</text><text class="add-text">添加</text></view>
    </view>

    <view class="calendar-card">
      <view class="week-header">
        <text v-for="(day, idx) in weekDays" :key="idx" class="week-day" :class="{ weekend: idx === 0 || idx === 6 }">{{ day }}</text>
      </view>
      <view class="date-grid">
        <view v-for="(cell, idx) in calendarDays" :key="idx" class="date-cell" :class="{ dimmed: cell.type !== 'current' }">
          <view class="date-num" :class="{ 'is-today': isToday(cell.day) && cell.type === 'current', 'is-selected': isSelected(cell.day) && cell.type === 'current' }" @click="cell.type === 'current' && selectDay(cell.day)"><text>{{ cell.day }}</text></view>
          <view v-if="cell.type === 'current'" class="date-dots">
            <view v-for="(color, di) in getDayDotColors(cell.day).slice(0, 3)" :key="di" class="date-dot" :style="{ backgroundColor: color }" />
          </view>
        </view>
      </view>
      <view class="legend">
        <view v-for="item in legendItems" :key="item.label" class="legend-item">
          <view class="legend-dot" :style="{ backgroundColor: item.color }" />
          <text class="legend-label">{{ item.label }}</text>
        </view>
      </view>
    </view>

    <view class="events-section">
      <text class="events-title">{{ month.value }}月{{ selectedDay }}日活动</text>
      <view v-if="todayEvents.length === 0" class="events-empty"><text>{{ month.value }}月{{ selectedDay }}日暂无活动安排</text></view>
      <view v-else class="events-list">
        <view v-for="event in todayEvents" :key="event.id" class="event-card">
          <view class="event-left">
            <view class="event-icon-box" :style="{ backgroundColor: eventTypeBgs[event.event_type] || '#DBEAFE' }">
              <text class="event-icon" :style="{ color: eventDotColors[event.event_type] || '#3B82F6' }">{{ eventTypeIcons[event.event_type] || "事" }}</text>
            </view>
            <view class="event-info">
              <text class="event-title">{{ event.title }}</text>
              <text class="event-desc">{{ event.description || event.content || "" }}</text>
              <view v-if="event.start_time || event.end_time" class="event-time-row">
                <text class="event-time-icon">时</text>
                <text class="event-time">{{ event.start_time || "" }}{{ event.end_time ? " - " + event.end_time : "" }}</text>
              </view>
            </view>
          </view>
          <view class="event-edit" @click="editEvent(event)"><text>编辑</text></view>
        </view>
      </view>
    </view>
    <view class="bottom-safe" />
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; box-sizing: border-box; padding: $spacing-lg 28rpx; background: $color-bg-page; }

.month-bar { display: flex; align-items: center; justify-content: space-between; padding: $spacing-xl; border-radius: $border-radius-xl; background: #fff; box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05); margin-bottom: $spacing-lg; }
.month-arrow { display: flex; align-items: center; justify-content: center; width: 64rpx; height: 64rpx; border-radius: $border-radius-full; background: #F1F5F9; }
.arrow-text { font-size: $typography-font-size-md; color: $color-text-muted; }
.month-text { font-size: $typography-font-size-2xl; font-weight: 700; color: $color-text-primary; }
.add-btn { display: flex; align-items: center; gap: 6rpx; padding: $spacing-md $spacing-xl; border-radius: $border-radius-full; background: $color-semantic-info; }
.add-icon { font-size: $typography-font-size-lg; font-weight: 700; color: #fff; }
.add-text { font-size: $typography-font-size-base; font-weight: 500; color: #fff; }

.calendar-card { padding: $spacing-xl; border-radius: $border-radius-xl; background: #fff; box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05); margin-bottom: $spacing-lg; }
.week-header { display: flex; justify-content: space-between; margin-bottom: $spacing-lg; }
.week-day { width: 88rpx; text-align: center; font-size: $typography-font-size-md; font-weight: 500; color: $color-text-muted; }
.week-day.weekend { color: $color-semantic-danger; }
.date-grid { display: grid; grid-template-columns: repeat(7, 1fr); row-gap: $spacing-xs; }
.date-cell { display: flex; flex-direction: column; align-items: center; padding: 6rpx 0 10rpx; min-height: 90rpx; }
.date-cell.dimmed { opacity: 0.35; }
.date-num { display: flex; align-items: center; justify-content: center; width: 64rpx; height: 64rpx; border-radius: $border-radius-full; }
.date-num text { font-size: $typography-font-size-md; color: $color-text-muted; }
.date-num.is-today { background: $color-semantic-info; text { color: #fff; font-weight: 700; } }
.date-num.is-selected { background: $color-semantic-info-bg; text { color: $color-semantic-info; font-weight: 700; } }
.date-num.is-today.is-selected { background: $color-semantic-info; text { color: #fff; font-weight: 700; } }
.date-dots { display: flex; gap: 6rpx; margin-top: 4rpx; }
.date-dot { width: 14rpx; height: 14rpx; border-radius: $border-radius-full; }

.legend { display: flex; justify-content: center; gap: $spacing-xl; margin-top: $spacing-lg; padding-top: $spacing-lg; border-top: 2rpx solid #F1F5F9; }
.legend-item { display: flex; align-items: center; gap: $spacing-xs; }
.legend-dot { width: 16rpx; height: 16rpx; border-radius: $border-radius-full; }
.legend-label { font-size: $typography-font-size-sm; color: $color-text-muted; }

.events-section { margin-bottom: $spacing-lg; }
.events-title { display: block; margin-bottom: $spacing-lg; font-size: $typography-font-size-xl; font-weight: 700; color: $color-text-primary; }
.events-empty { padding: 80rpx 0; text-align: center; text { font-size: $typography-font-size-md; color: $color-text-hint; } }
.events-list { display: flex; flex-direction: column; gap: $spacing-lg; }
.event-card { display: flex; align-items: flex-start; justify-content: space-between; padding: $spacing-xl; border-radius: $border-radius-xl; background: #fff; box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05); }
.event-left { display: flex; gap: $spacing-lg; flex: 1; min-width: 0; }
.event-icon-box { display: flex; flex-shrink: 0; align-items: center; justify-content: center; width: 80rpx; height: 80rpx; border-radius: $border-radius-xl; }
.event-icon { font-size: 34rpx; font-weight: 700; }
.event-info { display: flex; flex-direction: column; gap: $spacing-xs; min-width: 0; }
.event-title { font-size: $typography-font-size-lg; font-weight: 600; color: $color-text-primary; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.event-desc { font-size: $typography-font-size-base; color: $color-text-muted; display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; line-height: 1.4; }
.event-time-row { display: flex; align-items: center; gap: $spacing-xs; margin-top: 4rpx; }
.event-time-icon { font-size: $typography-font-size-xs; font-weight: 700; color: $color-text-hint; }
.event-time { font-size: $typography-font-size-sm; color: $color-text-hint; }
.event-edit { flex-shrink: 0; padding: $spacing-sm $spacing-lg; border-radius: $border-radius-md; background: #F1F5F9; margin-top: 4rpx; margin-left: $spacing-lg; text { font-size: $typography-font-size-sm; color: $color-text-muted; } }
.bottom-safe { height: calc(60rpx + env(safe-area-inset-bottom)); }
</style>
