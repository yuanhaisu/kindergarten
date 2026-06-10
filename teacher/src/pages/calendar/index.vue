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
const isCurrentMonth = computed(() => {
  const now = new Date()
  return year.value === now.getFullYear() && month.value === now.getMonth() + 1
})

const monthLabel = computed(() => `${year.value}年${month.value}月`)

const weekDays = ["日", "一", "二", "三", "四", "五", "六"]

const calendarDays = computed(() => {
  const y = year.value
  const m = month.value - 1
  const firstDay = new Date(y, m, 1).getDay()
  const daysInMonth = new Date(y, m + 1, 0).getDate()
  const daysInPrevMonth = new Date(y, m, 0).getDate()

  const days = []
  // Previous month tail
  for (let i = firstDay - 1; i >= 0; i--) {
    days.push({ day: daysInPrevMonth - i, type: "prev" })
  }
  // Current month
  for (let i = 1; i <= daysInMonth; i++) {
    days.push({ day: i, type: "current" })
  }
  // Next month head
  const remaining = 42 - days.length
  for (let i = 1; i <= remaining && i <= 14; i++) {
    days.push({ day: i, type: "next" })
  }
  return days
})

const selectedDay = ref(new Date().getDate())

function selectDay(day) {
  selectedDay.value = day
}

function isSelected(day) {
  if (day === selectedDay.value) return true
  if (selectedDay.value === today.value && day === today.value && isCurrentMonth.value) return true
  return false
}

const todayEvents = computed(() => {
  const m = String(month.value).padStart(2, "0")
  const d = String(selectedDay.value).padStart(2, "0")
  const dateStr = `${year.value}-${m}-${d}`
  return events.value.filter(e => e.event_date === dateStr)
})

const eventDotColors = {
  notification: "#3B82F6",
  activity: "#22C55E",
  holiday: "#F97316",
  meeting: "#A855F7",
  health: "#EF4444"
}

const eventTypeIcons = {
  notification: "通",
  activity: "活",
  holiday: "假",
  meeting: "会",
  health: "检"
}

const eventTypeBgs = {
  notification: "#DBEAFE",
  activity: "#DCFCE7",
  holiday: "#FFF7ED",
  meeting: "#F3E8FF",
  health: "#FEE2E2"
}

const legendItems = [
  { label: "通知", color: "#3B82F6" },
  { label: "活动", color: "#22C55E" },
  { label: "假期", color: "#F97316" },
  { label: "家长会", color: "#A855F7" },
  { label: "体检", color: "#EF4444" }
]

function getEventsForDay(day) {
  if (day <= 0) return []
  const dateStr = `${year.value}-${String(month.value).padStart(2, "0")}-${String(day).padStart(2, "0")}`
  return events.value.filter(e => e.event_date === dateStr)
}

function getDayDotColors(day) {
  const dayEvents = getEventsForDay(day)
  return dayEvents.map(e => eventDotColors[e.event_type] || "#3B82F6")
}

function isToday(day) {
  if (!isCurrentMonth.value) return false
  return day === today.value
}

function prevMonth() {
  const d = new Date(currentDate.value)
  d.setMonth(d.getMonth() - 1)
  currentDate.value = d
  loadEvents()
}
function nextMonth() {
  const d = new Date(currentDate.value)
  d.setMonth(d.getMonth() + 1)
  currentDate.value = d
  loadEvents()
}

function goAdd() {
  uni.navigateTo({ url: "/pages/calendar/create" })
}

function editEvent(event) {
  uni.navigateTo({ url: `/pages/calendar/detail?id=${event.id}` })
}

const mockEvents = [
  { id: 1, title: "春季出游活动通知", event_date: "2026-03-19", event_type: "notification", description: "请各位家长提醒孩子准备出游物品，明天早上8点在幼儿园门口集合", start_time: "08:00", end_time: "15:00", location: "幼儿园操场及周边公园", classes: ["中一班", "中二班"], importance: "important", student_count: 24 },
  { id: 2, title: "美术课：春天的小树", event_date: "2026-03-19", event_type: "activity", description: "带领小朋友们用画笔描绘春天的景色，发挥想象力", start_time: "09:00", end_time: "10:30", location: "中一班教室", classes: ["中一班"], importance: "normal", student_count: 12 },
  { id: 3, title: "音乐课：学习新儿歌", event_date: "2026-03-19", event_type: "activity", description: "教小朋友们唱《春天在哪里》，配合动作表演", start_time: "14:30", end_time: "15:30", location: "音乐教室", classes: ["中二班"], importance: "normal", student_count: 12 },
  { id: 4, title: "三八妇女节放假通知", event_date: "2026-03-08", event_type: "holiday", description: "三八妇女节全体放假一天，请各位家长安排好孩子的假期生活", start_time: "", end_time: "", location: "", classes: ["中一班", "中二班", "大一班"], importance: "important", student_count: 0 },
  { id: 5, title: "新学期家长会", event_date: "2026-03-15", event_type: "meeting", description: "召开新学期家长会，介绍本学期教学计划、活动安排及家园共育事项", start_time: "18:30", end_time: "20:00", location: "多功能厅", classes: ["中一班", "中二班"], importance: "important", student_count: 0 },
  { id: 6, title: "春季健康体检", event_date: "2026-03-20", event_type: "health", description: "幼儿园统一组织春季体检，请确保孩子当天到园，空腹参加", start_time: "08:00", end_time: "11:00", location: "保健室", classes: ["中一班", "中二班", "大一班", "大二班"], importance: "urgent", student_count: 36 },
  { id: 7, title: "亲子运动会", event_date: "2026-03-25", event_type: "activity", description: "举办春季亲子运动会，邀请家长与孩子一起参与趣味运动项目", start_time: "09:00", end_time: "12:00", location: "幼儿园操场", classes: ["中一班", "中二班", "大一班", "大二班", "小一班"], importance: "important", student_count: 48 },
  { id: 8, title: "消防安全演练通知", event_date: "2026-03-22", event_type: "notification", description: "全园进行消防安全疏散演练，请家长提前给孩子讲解基本消防知识", start_time: "10:00", end_time: "10:30", location: "全园", classes: ["中一班", "中二班", "大一班", "大二班"], importance: "urgent", student_count: 36 },
  { id: 9, title: "月度教学总结会议", event_date: "2026-03-29", event_type: "meeting", description: "全体教师参加月度教学总结，分享本月教学成果与下月计划", start_time: "16:00", end_time: "17:30", location: "会议室", classes: [], importance: "normal", student_count: 0 }
]

async function loadEvents() {
  loading.value = true
  try {
    const res = await fetchTeacherCalendarEvents()
    const list = res.data?.list || []
    events.value = list.length ? list : mockEvents
  } catch (err) {
    events.value = mockEvents
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

onLoad(loadEvents)
onPullDownRefresh(loadEvents)
</script>

<template>
  <view class="page">
    <!-- 月份选择栏 -->
    <view class="month-bar">
      <view class="month-arrow" @click="prevMonth">
        <text class="arrow-text">◂</text>
      </view>
      <text class="month-text">{{ monthLabel }}</text>
      <view class="month-arrow" @click="nextMonth">
        <text class="arrow-text">▸</text>
      </view>
      <view class="add-btn" @click="goAdd">
        <text class="add-icon">+</text>
        <text class="add-text">添加</text>
      </view>
    </view>

    <!-- 日历区域 -->
    <view class="calendar-card">
      <!-- 星期表头 -->
      <view class="week-header">
        <text
          v-for="(day, idx) in weekDays"
          :key="idx"
          class="week-day"
          :class="{ weekend: idx === 0 || idx === 6 }"
        >{{ day }}</text>
      </view>

      <!-- 日期网格 -->
      <view class="date-grid">
        <view
          v-for="(cell, idx) in calendarDays"
          :key="idx"
          class="date-cell"
          :class="{ dimmed: cell.type !== 'current' }"
        >
          <view
            class="date-num"
            :class="{
              'is-today': isToday(cell.day) && cell.type === 'current',
              'is-selected': isSelected(cell.day) && cell.type === 'current'
            }"
            @click="cell.type === 'current' && selectDay(cell.day)"
          >
            <text>{{ cell.day }}</text>
          </view>
          <!-- 事件彩色圆点 -->
          <view v-if="cell.type === 'current'" class="date-dots">
            <view
              v-for="(color, di) in getDayDotColors(cell.day).slice(0, 3)"
              :key="di"
              class="date-dot"
              :style="{ backgroundColor: color }"
            />
          </view>
        </view>
      </view>

      <!-- 图例 -->
      <view class="legend">
        <view v-for="item in legendItems" :key="item.label" class="legend-item">
          <view class="legend-dot" :style="{ backgroundColor: item.color }" />
          <text class="legend-label">{{ item.label }}</text>
        </view>
      </view>
    </view>

    <!-- 今日活动 -->
    <view class="events-section">
      <text class="events-title">
        {{ month.value }}月{{ selectedDay }}日活动
      </text>

      <view v-if="todayEvents.length === 0" class="events-empty">
        <text>{{ month.value }}月{{ selectedDay }}日暂无活动安排</text>
      </view>

      <view v-else class="events-list">
        <view v-for="event in todayEvents" :key="event.id" class="event-card">
          <view class="event-left">
            <view
              class="event-icon-box"
              :style="{ backgroundColor: eventTypeBgs[event.event_type] || '#DBEAFE' }"
            >
              <text
                class="event-icon"
                :style="{ color: eventDotColors[event.event_type] || '#3B82F6' }"
              >{{ eventTypeIcons[event.event_type] || "事" }}</text>
            </view>
            <view class="event-info">
              <text class="event-title">{{ event.title }}</text>
              <text class="event-desc">{{ event.description || event.content || "" }}</text>
              <view v-if="event.start_time || event.end_time" class="event-time-row">
                <text class="event-time-icon">时</text>
                <text class="event-time">
                  {{ event.start_time || "" }}{{ event.end_time ? " - " + event.end_time : "" }}
                </text>
              </view>
            </view>
          </view>
          <view class="event-edit" @click="editEvent(event)">
            <text>编辑</text>
          </view>
        </view>
      </view>
    </view>

    <view class="bottom-safe" />
  </view>
</template>

<style scoped lang="scss">
$blue-500: #3B82F6;
$blue-50: #EFF6FF;
$slate-50: #F8FAFC;
$slate-300: #CBD5E1;
$slate-400: #94A3B8;
$slate-500: #64748B;
$slate-800: #1E293B;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 24rpx 28rpx;
  background: $slate-50;
}

// ===== 月份选择栏 =====
.month-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 32rpx 32rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.month-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  border-radius: 999rpx;
  background: #F1F5F9;
}

.arrow-text {
  font-size: 28rpx;
  color: $slate-500;
}

.month-text {
  font-size: 36rpx;
  font-weight: 700;
  color: $slate-800;
}

.add-btn {
  display: flex;
  align-items: center;
  gap: 6rpx;
  padding: 16rpx 32rpx;
  border-radius: 999rpx;
  background: $blue-500;
}

.add-icon {
  font-size: 30rpx;
  font-weight: 700;
  color: #fff;
}

.add-text {
  font-size: 26rpx;
  font-weight: 500;
  color: #fff;
}

// ===== 日历卡片 =====
.calendar-card {
  padding: 32rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.week-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.week-day {
  width: 88rpx;
  text-align: center;
  font-size: 28rpx;
  font-weight: 500;
  color: $slate-500;
}

.week-day.weekend {
  color: #EF4444;
}

.date-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  row-gap: 8rpx;
}

.date-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 6rpx 0 10rpx;
  min-height: 90rpx;
}

.date-cell.dimmed {
  opacity: 0.35;
}

.date-num {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  border-radius: 999rpx;
}

.date-num text {
  font-size: 28rpx;
  color: $slate-500;
}

.date-num.is-today {
  background: $blue-500;

  text { color: #fff; font-weight: 700; }
}

.date-num.is-selected {
  background: $blue-50;

  text { color: $blue-500; font-weight: 700; }
}

.date-num.is-today.is-selected {
  background: $blue-500;

  text { color: #fff; font-weight: 700; }
}

.date-dots {
  display: flex;
  gap: 6rpx;
  margin-top: 4rpx;
}

.date-dot {
  width: 14rpx;
  height: 14rpx;
  border-radius: 999rpx;
}

// ===== 图例 =====
.legend {
  display: flex;
  justify-content: center;
  gap: 32rpx;
  margin-top: 24rpx;
  padding-top: 24rpx;
  border-top: 2rpx solid #F1F5F9;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.legend-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 999rpx;
}

.legend-label {
  font-size: 24rpx;
  color: $slate-500;
}

// ===== 今日活动 =====
.events-section {
  margin-bottom: 24rpx;
}

.events-title {
  display: block;
  margin-bottom: 24rpx;
  font-size: 32rpx;
  font-weight: 700;
  color: $slate-800;
}

.events-empty {
  padding: 80rpx 0;
  text-align: center;

  text {
    font-size: 28rpx;
    color: $slate-400;
  }
}

.events-list {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.event-card {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 32rpx;
  border-radius: 24rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
}

.event-left {
  display: flex;
  gap: 24rpx;
  flex: 1;
  min-width: 0;
}

.event-icon-box {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: 24rpx;
}

.event-icon {
  font-size: 34rpx;
  font-weight: 700;
}

.event-info {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
  min-width: 0;
}

.event-title {
  font-size: 30rpx;
  font-weight: 600;
  color: $slate-800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.event-desc {
  font-size: 26rpx;
  color: $slate-500;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  line-height: 1.4;
}

.event-time-row {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-top: 4rpx;
}

.event-time-icon {
  font-size: 22rpx;
  font-weight: 700;
  color: $slate-400;
}

.event-time {
  font-size: 24rpx;
  color: $slate-400;
}

.event-edit {
  flex-shrink: 0;
  padding: 12rpx 24rpx;
  border-radius: 16rpx;
  background: #F1F5F9;
  margin-top: 4rpx;
  margin-left: 24rpx;

  text {
    font-size: 24rpx;
    color: $slate-500;
  }
}

.bottom-safe {
  height: calc(60rpx + env(safe-area-inset-bottom));
}
</style>
