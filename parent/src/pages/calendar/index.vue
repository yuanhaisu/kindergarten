<script setup>
/**
 * 班级日历 · 家长端
 *
 * 月历回答「哪几天有事」，当日清单回答「这天要做什么」，顶部变更卡回答
 * 「哪些安排变了」。三者分工明确，所以变更卡不做月份过滤：服务端只在
 * 「近 7 天修订且尚未发生」窗口内下发 change_summary，若再按视图月份筛一次，
 * 隔壁月的调整会被静默吞掉，而家长可能正按旧时间行动。
 *
 * 视图月份与「今天」不一定同月：请求区间取
 * min(月首, 今天) ~ max(月末, 今天+7 天)，本月活动与全局变更一次拿回，
 * 不需要新接口。区间始终带 start/end —— 不传就只能拿到默认窗口，翻月会是空的。
 */
import { computed, nextTick, ref } from "vue"
import { onLoad, onShow, onUnload } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchClassCalendar } from "@/api/parent-class"
import { withStudentScope } from "@/api/parent-task"
import {
  formatDateLabel,
  formatMonthDayWeekday,
  formatMonthTitle,
  formatYmd,
  localDateKey,
  monthKeyOf,
  parseDateOnly,
  shiftDateKey
} from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  bell: "\u{ef94}", // notification-3-line
  left: "\u{ea64}", // arrow-left-s-line
  right: "\u{ea6e}", // arrow-right-s-line
  down: "\u{ea4e}", // arrow-down-s-line
  calendar: "\u{eb27}", // calendar-line
  calendarEvent: "\u{eb25}", // calendar-event-line
  pin: "\u{ef14}", // map-pin-line
  refresh: "\u{f064}", // refresh-line
  cloudOff: "\u{eb9f}" // cloud-off-line
}

// 类型色固定映射到既有 token（通知/活动/假期/家长会/体检）。
// 不使用后端下发的 color：字典里的紫色在本项目 token 中不存在，
// 同一份数据在老师端与家长端也不该出现两种配色。
const TYPE_INFO = {
  notice: { label: "通知", char: "通" },
  activity: { label: "活动", char: "活" },
  holiday: { label: "假期", char: "假" },
  meeting: { label: "家长会", char: "会" },
  checkup: { label: "体检", char: "检" }
}
const TYPE_ORDER = ["notice", "activity", "holiday", "meeting", "checkup"]

const WEEK_DAYS = ["日", "一", "二", "三", "四", "五", "六"]
const HIGHLIGHT_MS = 2400
const CHANGE_PREVIEW = 3

const daysInMonth = (y, m) => new Date(y, m + 1, 0).getDate()
const dateKeyOf = e => String(e?.event_date || "").slice(0, 10)

const now = new Date()
const todayKey = localDateKey(now)
const todayView = { y: now.getFullYear(), m: now.getMonth() }

const view = ref({ ...todayView })
const selected = ref(todayKey)
const events = ref([])
const loading = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const changesExpanded = ref(false)
const expandedMap = ref({})
const highlightId = ref("")

let loadSeq = 0
let highlightTimer = null

// ---------- 数据 ----------

function compareEvents(a, b) {
  const da = dateKeyOf(a)
  const db = dateKeyOf(b)
  if (da !== db) return da < db ? -1 : 1
  if (Boolean(a.is_all_day) !== Boolean(b.is_all_day)) return a.is_all_day ? -1 : 1
  const ta = a.start_time || "00:00"
  const tb = b.start_time || "00:00"
  if (ta !== tb) return ta < tb ? -1 : 1
  return String(a.title || "").localeCompare(String(b.title || ""), "zh-CN")
}

const sortedEvents = computed(() => [...events.value].sort(compareEvents))
const monthPrefix = computed(() => monthKeyOf(formatYmd(new Date(view.value.y, view.value.m, 1))))
const monthEvents = computed(() =>
  sortedEvents.value.filter(e => dateKeyOf(e).slice(0, 7) === monthPrefix.value)
)
const dayEvents = computed(() =>
  sortedEvents.value.filter(e => dateKeyOf(e) === selected.value)
)

const changes = computed(() => sortedEvents.value.filter(e => e.change_summary))
const visibleChanges = computed(() =>
  changesExpanded.value ? changes.value : changes.value.slice(0, CHANGE_PREVIEW)
)
const hiddenChangeCount = computed(() => Math.max(0, changes.value.length - CHANGE_PREVIEW))

// 首帧（onLoad 之前的第一次渲染）也走骨架：此时既没有数据也没有错误。
const showSkeleton = computed(() => loading.value || (!hasLoaded.value && !loadError.value))
const showError = computed(() => Boolean(loadError.value) && !loading.value)
const showChanges = computed(
  () => !showSkeleton.value && !showError.value && changes.value.length > 0
)

const isCurrentMonth = computed(
  () => view.value.y === todayView.y && view.value.m === todayView.m
)
const monthTitle = computed(() => formatMonthTitle(view.value.y, view.value.m))
const todayDisabled = computed(() => isCurrentMonth.value && selected.value === todayKey)

const monthSub = computed(() => {
  if (showSkeleton.value) return "正在加载本月活动"
  if (showError.value) return "本月活动未加载"
  const n = monthEvents.value.length
  if (!n) return isCurrentMonth.value ? "本月暂无活动" : `${view.value.m + 1}月暂无活动`
  // 非当前月写「本月」会和「2026年11月」的标题自相矛盾。
  return isCurrentMonth.value ? `本月 ${n} 个活动` : `${view.value.m + 1}月 ${n} 个活动`
})

const selectedDateLabel = computed(
  () => formatMonthDayWeekday(selected.value) || formatDateLabel(selected.value)
)

const monthEmpty = computed(
  () => !showSkeleton.value && !showError.value && monthEvents.value.length === 0
)
const dayEmpty = computed(
  () => !showSkeleton.value && !showError.value && !monthEmpty.value && dayEvents.value.length === 0
)
const monthEmptyTitle = computed(() => `${view.value.m + 1}月暂无活动安排`)
const dayEmptyTitle = computed(() => `${formatDateLabel(selected.value)}暂无活动安排`)
const dayCountText = computed(() => {
  if (showSkeleton.value) return "加载中"
  return dayEvents.value.length ? `${dayEvents.value.length} 个活动` : "暂无活动"
})

const legendTypes = computed(() =>
  TYPE_ORDER.filter(t => monthEvents.value.some(e => e.event_type === t))
)

// 固定 6 行 × 7 列：不因月份长短跳高，翻月时整块高度稳定。
const gridCells = computed(() => {
  const { y, m } = view.value
  const firstWeekday = new Date(y, m, 1).getDay()
  const dotsByDate = {}
  for (const e of monthEvents.value) {
    const key = dateKeyOf(e)
    if (!dotsByDate[key]) dotsByDate[key] = []
    if (dotsByDate[key].length < 3) dotsByDate[key].push(dotTypeOf(e))
  }
  const cells = []
  for (let i = 0; i < 42; i += 1) {
    const d = new Date(y, m, 1 - firstWeekday + i)
    const cy = d.getFullYear()
    const cm = d.getMonth()
    const cd = d.getDate()
    const iso = formatYmd(d)
    const inMonth = cy === y && cm === m
    cells.push({
      key: iso,
      iso,
      day: cd,
      inMonth,
      // 补格不参与「今天」的强调：邻月里的同一天不是当前视图的今天。
      isToday: inMonth && iso === todayKey,
      isSelected: inMonth && iso === selected.value,
      dots: inMonth ? dotsByDate[iso] || [] : []
    })
  }
  return cells
})

function dotTypeOf(e) {
  return TYPE_INFO[e?.event_type] ? e.event_type : "other"
}
function typeOf(e) {
  return TYPE_INFO[e?.event_type] || null
}
function typeKeyOf(e) {
  return typeOf(e) ? e.event_type : "other"
}
function typeLabelOf(e) {
  return e?.event_type_label || typeOf(e)?.label || "活动"
}
function typeCharOf(e) {
  const info = typeOf(e)
  if (info) return info.char
  return Array.from(String(typeLabelOf(e)))[0] || "事"
}
function timeLabelOf(e) {
  if (e?.is_all_day) return "全天"
  const start = e?.start_time || ""
  const end = e?.end_time || ""
  if (start && end) return `${start}–${end}`
  return start || end || "全天"
}
function isExpanded(id) {
  return expandedMap.value[String(id)] === true
}

// ---------- 请求 ----------

function requestRange() {
  const { y, m } = view.value
  const monthStart = formatYmd(new Date(y, m, 1))
  const monthEnd = formatYmd(new Date(y, m, daysInMonth(y, m)))
  const todayPlusWeek = shiftDateKey(todayKey, 7) || todayKey
  return {
    start: monthStart < todayKey ? monthStart : todayKey,
    end: monthEnd > todayPlusWeek ? monthEnd : todayPlusWeek
  }
}

async function loadEvents({ skeleton = true } = {}) {
  const seq = ++loadSeq
  if (skeleton) loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const { start, end } = requestRange()
    const scoped = await withStudentScope(sid =>
      fetchClassCalendar({ student_id: sid, start, end })
    )
    if (seq !== loadSeq) return
    if (!scoped) {
      events.value = []
      loadError.value = "未绑定孩子信息"
      return
    }
    const list = scoped.result?.data?.list
    events.value = Array.isArray(list) ? list : []
    hasLoaded.value = true
  } catch (err) {
    if (seq !== loadSeq) return
    console.warn("[calendar-index] load failed:", err?.message || err)
    loadError.value = "网络异常，未获取到本月活动"
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

// ---------- 交互 ----------

function resetTransient() {
  changesExpanded.value = false
  expandedMap.value = {}
  clearHighlight()
}

function selectDate(iso) {
  if (!iso || iso === selected.value) return
  selected.value = iso
  expandedMap.value = {}
  clearHighlight()
}

function onDayClick(cell) {
  if (!cell.inMonth) return
  selectDate(cell.iso)
}

function shiftMonth(delta) {
  if (loading.value) return
  const first = new Date(view.value.y, view.value.m + delta, 1)
  const y = first.getFullYear()
  const m = first.getMonth()
  const previousDay = parseDateOnly(selected.value)?.getDate() || 1
  view.value = { y, m }
  const current = y === todayView.y && m === todayView.m
  // 翻回当月自动落到今天；否则保留同号日，目标月没有该号就收敛到月末。
  selected.value = current ? todayKey : formatYmd(new Date(y, m, Math.min(previousDay, daysInMonth(y, m))))
  resetTransient()
  loadEvents()
}

function goToday() {
  const sameMonth = isCurrentMonth.value
  const alreadyHere = sameMonth && selected.value === todayKey
  if (alreadyHere && !loadError.value) return
  view.value = { ...todayView }
  selected.value = todayKey
  resetTransient()
  if (!sameMonth || !hasLoaded.value || loadError.value) loadEvents()
}

function onTodayClick() {
  if (todayDisabled.value) return
  goToday()
}

function retry() {
  if (loading.value) return
  loadEvents()
}

function toggleChanges() {
  changesExpanded.value = !changesExpanded.value
}

function toggleCard(id) {
  const key = String(id)
  expandedMap.value = { ...expandedMap.value, [key]: !expandedMap.value[key] }
}

function clearHighlight() {
  if (highlightTimer) {
    clearTimeout(highlightTimer)
    highlightTimer = null
  }
  highlightId.value = ""
}

function flashCard(id) {
  clearHighlight()
  highlightId.value = String(id)
  highlightTimer = setTimeout(() => {
    highlightId.value = ""
    highlightTimer = null
  }, HIGHLIGHT_MS)
}

function scrollToCard(id) {
  const query = uni.createSelectorQuery()
  query.select(`#evt-${id}`).boundingClientRect()
  query.selectViewport().scrollOffset()
  query.exec(res => {
    const rect = res && res[0]
    const viewport = res && res[1]
    if (!rect || !viewport) return
    uni.pageScrollTo({
      scrollTop: Math.max(0, viewport.scrollTop + rect.top - 120),
      duration: 300
    })
  })
}

// 变更条目可能指向隔壁月：先切视图并补拉该月，否则当日区是空的，
// 既定位不到卡片，也高亮不了「家长最该看到的那一条」。
async function jumpToChange(item) {
  const date = parseDateOnly(item?.event_date)
  if (!date) return
  const y = date.getFullYear()
  const m = date.getMonth()
  const monthChanged = y !== view.value.y || m !== view.value.m
  const loaded = events.value.some(e => String(e.id) === String(item.id))
  view.value = { y, m }
  selected.value = dateKeyOf(item)
  resetTransient()
  if (monthChanged || !loaded) {
    await loadEvents()
    if (loadError.value) return
  }
  await nextTick()
  flashCard(item.id)
  setTimeout(() => scrollToCard(item.id), 60)
}

let skipNextShow = true
onLoad(() => {
  loadEvents()
})
onShow(() => {
  // 从其他页返回时静默刷新；首次 onShow 紧随 onLoad，跳过。
  if (skipNextShow) {
    skipNextShow = false
    return
  }
  loadEvents({ skeleton: Boolean(loadError.value) || !hasLoaded.value })
})
onUnload(() => {
  clearHighlight()
})
</script>

<template>
  <PageShell max-width="430px" flush-top bottom-space="48rpx">
    <view class="page">
      <!-- ===== 变更卡：全局范围，不随翻月消失 ===== -->
      <view v-if="showChanges" class="changes-card">
        <view class="changes-head">
          <view class="changes-mark">
            <text class="changes-mark-glyph">{{ ICON.bell }}</text>
          </view>
          <text class="changes-title">安排有变更</text>
          <view class="changes-count">
            <text class="changes-count-text">{{ changes.length }} 条</text>
          </view>
        </view>
        <view class="change-list">
          <view
            v-for="item in visibleChanges"
            :key="item.id"
            class="change-item"
            @click="jumpToChange(item)"
          >
            <view class="change-dot" />
            <view class="change-copy">
              <text class="change-title">{{ item.title }}</text>
              <text class="change-diff">{{ formatDateLabel(item.event_date) }} · {{ item.change_summary }}</text>
            </view>
            <text class="change-chevron">{{ ICON.right }}</text>
          </view>
        </view>
        <view
          v-if="changesExpanded || hiddenChangeCount > 0"
          class="changes-toggle"
          @click="toggleChanges"
        >
          <text class="changes-toggle-text">{{ changesExpanded ? "收起" : `另有 ${hiddenChangeCount} 条` }}</text>
          <text class="changes-caret" :class="{ 'is-open': changesExpanded }">{{ ICON.down }}</text>
        </view>
      </view>

      <!-- ===== 月份条：加载/失败时保留，翻月与回今天始终可用 ===== -->
      <view class="month-bar">
        <view class="month-stepper">
          <view class="month-nav" @click="shiftMonth(-1)">
            <text class="month-nav-glyph">{{ ICON.left }}</text>
          </view>
          <view class="month-center">
            <text class="month-title">{{ monthTitle }}</text>
            <text class="month-sub">{{ monthSub }}</text>
          </view>
          <view class="month-nav" @click="shiftMonth(1)">
            <text class="month-nav-glyph">{{ ICON.right }}</text>
          </view>
        </view>
        <view class="today-pill" :class="{ 'is-disabled': todayDisabled }" @click="onTodayClick">
          <text class="today-pill-text">今天</text>
        </view>
      </view>

      <!-- ===== 失败态 ===== -->
      <view v-if="showError" class="error-panel">
        <view class="error-icon">
          <text class="error-glyph">{{ ICON.cloudOff }}</text>
        </view>
        <text class="error-title">日历加载失败</text>
        <text class="error-desc">{{ loadError }}</text>
        <view class="btn-primary" @click="retry">
          <text class="btn-icon">{{ ICON.refresh }}</text>
          <text class="btn-text">重新加载</text>
        </view>
      </view>

      <!-- ===== 加载态：月份条保留，月历与当日区换骨架 ===== -->
      <template v-else-if="showSkeleton">
        <view class="calendar-skeleton">
          <view class="week-head">
            <text
              v-for="(w, i) in WEEK_DAYS"
              :key="i"
              class="week-day"
              :class="{ 'is-weekend': i === 0 || i === 6 }"
            >{{ w }}</text>
          </view>
          <view class="skeleton-grid">
            <view v-for="i in 42" :key="i" class="sk-day"><view class="sk-orb" /></view>
          </view>
          <view class="skeleton-legend">
            <view v-for="i in 3" :key="i" class="sk-legend-bar" />
          </view>
        </view>

        <view class="events-section">
          <view class="events-head">
            <view class="events-heading">
              <text class="events-title">当日安排</text>
              <text class="events-date">{{ selectedDateLabel }}</text>
            </view>
            <view class="events-count">
              <text class="events-count-text">{{ dayCountText }}</text>
            </view>
          </view>
          <view class="event-skeletons">
            <view v-for="i in 3" :key="i" class="event-skeleton">
              <view class="skeleton-head">
                <view class="sk-icon" />
                <view class="sk-lines">
                  <view class="sk-line sk-title" />
                  <view class="sk-line sk-meta" />
                </view>
              </view>
              <view class="sk-line sk-copy" />
              <view class="sk-line sk-copy sk-copy-short" />
              <view class="sk-foot">
                <view class="sk-foot-line" />
                <view class="sk-foot-line" />
              </view>
            </view>
          </view>
        </view>
      </template>

      <!-- ===== 正常态 ===== -->
      <template v-else>
        <view class="calendar-card">
          <view class="week-head">
            <text
              v-for="(w, i) in WEEK_DAYS"
              :key="i"
              class="week-day"
              :class="{ 'is-weekend': i === 0 || i === 6 }"
            >{{ w }}</text>
          </view>
          <view class="day-grid">
            <view
              v-for="cell in gridCells"
              :key="cell.key"
              class="day"
              :class="{
                'is-outside': !cell.inMonth,
                'is-today': cell.isToday,
                'is-selected': cell.isSelected
              }"
              @click="onDayClick(cell)"
            >
              <text class="day-num">{{ cell.day }}</text>
              <view class="day-dots">
                <view
                  v-for="(t, di) in cell.dots"
                  :key="di"
                  class="type-dot"
                  :class="`type-dot--${t}`"
                />
              </view>
            </view>
          </view>
          <view v-if="legendTypes.length" class="calendar-legend">
            <view v-for="t in legendTypes" :key="t" class="legend-item">
              <view class="type-dot" :class="`type-dot--${t}`" />
              <text class="legend-label">{{ TYPE_INFO[t].label }}</text>
            </view>
          </view>
        </view>

        <view class="events-section">
          <view class="events-head">
            <view class="events-heading">
              <text class="events-title">当日安排</text>
              <text class="events-date">{{ selectedDateLabel }}</text>
            </view>
            <view class="events-count">
              <text class="events-count-text">{{ dayCountText }}</text>
            </view>
          </view>

          <!-- 整月空：先于单日空判断，避免「这一天暂无」盖住「这个月都没发」 -->
          <view v-if="monthEmpty" class="empty-card empty-card--month">
            <view class="empty-icon">
              <text class="empty-glyph">{{ ICON.calendarEvent }}</text>
            </view>
            <text class="empty-title">{{ monthEmptyTitle }}</text>
            <text class="empty-desc">园所还没发布这个月的安排</text>
            <view class="btn-primary" @click="goToday">
              <text class="btn-text">回到今天</text>
            </view>
          </view>

          <view v-else-if="dayEmpty" class="empty-card empty-card--day">
            <view class="empty-icon">
              <text class="empty-glyph">{{ ICON.calendar }}</text>
            </view>
            <text class="empty-title">{{ dayEmptyTitle }}</text>
            <text class="empty-desc">可以点选其他日期看看</text>
          </view>

          <view v-else class="event-list">
            <view
              v-for="item in dayEvents"
              :id="`evt-${item.id}`"
              :key="item.id"
              class="event-card"
              :class="[
                `evt-${typeKeyOf(item)}`,
                { 'is-expanded': isExpanded(item.id), 'is-flash': highlightId === String(item.id) }
              ]"
              @click="toggleCard(item.id)"
            >
              <view class="event-head">
                <view class="event-icon">
                  <text class="event-icon-text">{{ typeCharOf(item) }}</text>
                </view>
                <view class="event-summary">
                  <text class="event-title">{{ item.title }}</text>
                  <!-- 类型只在这里出现一次，标题行不再重复渲染 -->
                  <view class="event-meta">
                    <text class="meta-type">{{ typeLabelOf(item) }}</text>
                    <text class="meta-sep">·</text>
                    <text class="event-time">{{ timeLabelOf(item) }}</text>
                  </view>
                </view>
              </view>

              <text v-if="item.description" class="event-desc">{{ item.description }}</text>

              <view v-if="item.change_summary" class="event-change">
                <text class="change-label">安排调整</text>
                <text class="change-text">{{ item.change_summary }}</text>
              </view>

              <view v-if="item.location || item.description" class="event-foot">
                <view v-if="item.location" class="event-loc">
                  <text class="loc-glyph">{{ ICON.pin }}</text>
                  <text class="loc-text">{{ item.location }}</text>
                </view>
                <!-- 没有描述就没有可展开的内容，不留一个点了没反应的「展开全文」 -->
                <view v-if="item.description" class="expand-hint">
                  <text class="expand-text">{{ isExpanded(item.id) ? "收起" : "展开全文" }}</text>
                  <text class="expand-caret" :class="{ 'is-open': isExpanded(item.id) }">{{ ICON.down }}</text>
                </view>
              </view>
            </view>
          </view>
        </view>
      </template>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 班级日历：白卡承载数据，暖色只落在「今天 / 选中 / 变更 / 主按钮」上。
// 类型点与图标底色按类型固定映射，同一天最多 3 个点，不做 +N 溢出提示。
// ============================================================

.page {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  padding-top: $spacing-xs;
}

// ---- 安排有变更 ----------------------------------------------
.changes-card {
  padding: 22rpx $spacing-lg 0;
  border-radius: $component-card-radius;
  background: $color-semantic-warning-bg;
  box-shadow: $shadow-sm;
}

.changes-head {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-width: 0;
  padding: 0 4rpx 14rpx;
}

.changes-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 54rpx;
  height: 54rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
}

.changes-mark-glyph {
  color: $color-semantic-warning;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.changes-title {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.changes-count {
  display: inline-flex;
  align-items: center;
  flex: none;
  min-height: 44rpx;
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-full;
  background: $color-bg-card;
}

.changes-count-text {
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.change-list {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.change-item {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-width: 0;
  min-height: 88rpx;
  padding: $spacing-sm $spacing-xs;
  border-radius: $border-radius-lg;

  &:active {
    background: $color-bg-card;
  }
}

.change-dot {
  flex: none;
  width: 10rpx;
  height: 10rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-warning;
}

.change-copy {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 2rpx $spacing-sm;
  min-width: 0;
}

.change-title {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.45;
  overflow-wrap: break-word;
}

.change-diff {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
  overflow-wrap: break-word;
}

.change-chevron {
  flex: none;
  color: $color-text-body;
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  line-height: 1;
}

.changes-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  min-height: 88rpx;
  padding: 0 $spacing-xs;
  border-top: 1rpx solid $color-bg-card;

  &:active {
    background: $color-bg-card;
  }
}

.changes-toggle-text {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.changes-caret {
  color: $color-text-body;
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  line-height: 1;
  transition: transform $animation-duration-fast $animation-easing-default;

  &.is-open {
    transform: rotate(180deg);
  }
}

// ---- 月份切换 ------------------------------------------------
.month-bar {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-width: 0;
  padding-top: 4rpx;
}

.month-stepper {
  display: flex;
  flex: 1;
  align-items: center;
  min-width: 0;
}

.month-nav {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;

  &:active {
    background: $color-brand-primary-bg;
  }
}

.month-nav-glyph {
  color: $color-text-body;
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.month-center {
  flex: 1;
  min-width: 0;
  padding: 0 4rpx;
  text-align: center;
}

.month-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  font-variant-numeric: tabular-nums;
  overflow-wrap: break-word;
}

.month-sub {
  display: block;
  margin-top: 2rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.today-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  min-width: 108rpx;
  min-height: 88rpx;
  padding: 0 26rpx;
  border: 1rpx solid $color-brand-primary-light;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;

  &:active {
    transform: scale(0.96);
  }
}

.today-pill-text {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  white-space: nowrap;
}

// 只有「已在今天」才置灰；当前月选中其他日期时仍要能一键回到今天。
.today-pill.is-disabled {
  border-color: $color-border-hairline;
  background: $color-bg-card;

  &:active {
    transform: none;
  }
}

.today-pill.is-disabled .today-pill-text {
  color: $color-text-hint;
}

// ---- 月历 ----------------------------------------------------
.calendar-card {
  padding: $spacing-lg 20rpx 18rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.week-head {
  display: flex;
  align-items: center;
  height: 48rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  letter-spacing: 0.04em;
  text-align: center;
}

.week-day {
  flex: 1;
  min-width: 0;
}

.week-day.is-weekend {
  color: $color-semantic-danger;
  font-weight: $typography-font-weight-semibold;
}

.day-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
}

.day {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  min-width: 0;
  min-height: 88rpx;
  padding: $spacing-xs 0 4rpx;
  border-radius: $border-radius-lg;

  &:active {
    background: $color-bg-page;
  }
}

// 补格不可选：点了也没有数据，灰显并去掉按压反馈。
.day.is-outside {
  &:active {
    background: transparent;
  }
}

.day-num {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.day.is-outside .day-num {
  color: $color-text-hint;
  font-weight: $typography-font-weight-normal;
  opacity: 0.55;
}

.day.is-today .day-num {
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-bold;
}

.day.is-selected .day-num {
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
  font-weight: $typography-font-weight-semibold;
}

// 今天 + 选中共存时加一圈外环，两种语义不合并成一个颜色。
.day.is-today.is-selected .day-num {
  box-shadow: 0 0 0 4rpx $color-brand-primary-light;
}

.day-dots {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6rpx;
  width: 100%;
  height: 8rpx;
}

.type-dot {
  width: 8rpx;
  height: 8rpx;
  flex: none;
  border-radius: $border-radius-full;
}

.type-dot--notice {
  background: $color-semantic-info;
}

.type-dot--activity {
  background: $color-semantic-success;
}

.type-dot--holiday {
  background: $color-brand-primary;
}

.type-dot--meeting {
  background: $color-text-muted;
}

.type-dot--checkup {
  background: $color-semantic-danger;
}

.type-dot--other {
  background: $color-text-hint;
}

.calendar-legend {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10rpx $spacing-lg;
  margin-top: 14rpx;
  padding: 18rpx 4rpx 2rpx;
  border-top: 1rpx solid $color-border-hairline;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  min-width: 0;
}

.legend-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

// ---- 当日安排 ------------------------------------------------
.events-section {
  margin-top: 4rpx;
}

.events-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20rpx;
  min-width: 0;
  padding: 0 4rpx 20rpx;
}

.events-heading {
  min-width: 0;
}

.events-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.events-date {
  display: block;
  margin-top: 6rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
}

.events-count {
  display: inline-flex;
  align-items: center;
  flex: none;
  min-height: 48rpx;
  margin-top: 2rpx;
  padding: 0 18rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
}

.events-count-text {
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.event-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.event-card {
  padding: 26rpx 28rpx 24rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;

  &:active {
    background: $color-bg-page;
  }
}

.event-head {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  min-width: 0;
}

.event-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}

.event-icon-text {
  color: $color-text-body;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.evt-notice .event-icon {
  background: $color-semantic-info-bg;
}

.evt-activity .event-icon {
  background: $color-semantic-success-bg;
}

.evt-holiday .event-icon {
  background: $color-brand-primary-bg;
}

.evt-meeting .event-icon {
  background: $color-bg-page;
}

.evt-checkup .event-icon {
  background: $color-semantic-danger-bg;
}

.event-summary {
  flex: 1;
  min-width: 0;
}

.event-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.event-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2rpx 10rpx;
  margin-top: 10rpx;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
}

.meta-type {
  color: $color-text-body;
  font-weight: $typography-font-weight-medium;
}

.meta-sep {
  color: $color-text-hint;
}

.event-time {
  color: $color-text-muted;
}

.event-desc {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  margin-top: 18rpx;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.65;
  overflow-wrap: break-word;
}

.event-card.is-expanded .event-desc {
  display: block;
  overflow: visible;
}

.event-change {
  display: flex;
  flex-wrap: wrap;
  gap: 2rpx $spacing-sm;
  margin-top: 18rpx;
  padding: 14rpx 18rpx;
  border-radius: $border-radius-lg;
  background: $color-semantic-warning-bg;
  font-size: $typography-font-size-sm;
  line-height: 1.55;
  font-variant-numeric: tabular-nums;
  overflow-wrap: break-word;
}

.change-label {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

.change-text {
  color: $color-text-body;
}

.event-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  min-width: 0;
  margin-top: 20rpx;
}

.event-loc {
  display: flex;
  flex: 1;
  align-items: flex-start;
  gap: $spacing-xs;
  min-width: 0;
}

.loc-glyph {
  flex: none;
  margin-top: 4rpx;
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  line-height: 1;
}

.loc-text {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  overflow-wrap: break-word;
}

.expand-hint {
  display: inline-flex;
  align-items: center;
  gap: 4rpx;
  flex: none;
  margin-left: auto;
}

.expand-text {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.expand-caret {
  color: $color-text-body;
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  line-height: 1;
  transition: transform $animation-duration-fast $animation-easing-default;

  &.is-open {
    transform: rotate(180deg);
  }
}

// 从变更条目跳转后的定位高亮：2.4s 内退回常态，避免留下「一直选中」的错觉。
.event-card.is-flash {
  border-color: $color-brand-primary-light;
  background: $color-brand-primary-bg;
  animation: card-flash 2400ms $animation-easing-default;
}

@keyframes card-flash {
  0%,
  62% {
    background: $color-brand-primary-bg;
    border-color: $color-brand-primary-light;
  }

  100% {
    background: $color-bg-card;
    border-color: $color-border-hairline;
  }
}

// ---- 空态 ----------------------------------------------------
.empty-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 0;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.empty-card--day {
  padding: 52rpx 36rpx 48rpx;
}

.empty-card--month {
  padding: 60rpx 40rpx 52rpx;
}

.empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 108rpx;
  height: 108rpx;
  margin-bottom: 20rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.empty-glyph {
  color: $color-text-muted;
  font-family: remixicon;
  font-size: 50rpx;
  line-height: 1;
}

.empty-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.45;
  overflow-wrap: break-word;
}

.empty-desc {
  margin-top: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

// ---- 主按钮 --------------------------------------------------
.btn-primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  min-height: 88rpx;
  padding: 0 40rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-sm;

  &:active {
    transform: scale(0.96);
    box-shadow: none;
  }
}

.btn-text {
  color: $color-text-on-accent;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.btn-icon {
  color: $color-text-on-accent;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.empty-card .btn-primary,
.error-panel .btn-primary {
  margin-top: $spacing-xl;
}

// ---- 加载骨架 ------------------------------------------------
.calendar-skeleton {
  padding: $spacing-lg 20rpx 20rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
}

.sk-day {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 88rpx;
}

.sk-orb {
  width: 56rpx;
  height: 56rpx;
  border-radius: $border-radius-lg;
  background: $color-border-hairline;
}

.skeleton-legend {
  display: flex;
  justify-content: center;
  gap: $spacing-md;
  margin-top: 16rpx;
  padding-top: 20rpx;
  border-top: 1rpx solid $color-border-hairline;
}

.sk-legend-bar {
  width: 84rpx;
  height: 16rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
}

.event-skeletons {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.event-skeleton {
  padding: 28rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.skeleton-head {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
}

.sk-icon {
  flex: none;
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-xl;
  background: $color-border-hairline;
}

.sk-lines {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 14rpx;
  min-width: 0;
  padding-top: 6rpx;
}

.sk-line {
  display: block;
  height: 20rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
}

.sk-title {
  width: 54%;
  height: 26rpx;
}

.sk-meta {
  width: 38%;
  height: 18rpx;
}

.sk-copy {
  margin-top: 24rpx;
}

.sk-copy-short {
  width: 68%;
  margin-top: 12rpx;
}

.sk-foot {
  display: flex;
  justify-content: space-between;
  gap: $spacing-lg;
  margin-top: 24rpx;
  padding-top: 20rpx;
  border-top: 1rpx solid $color-border-hairline;
}

.sk-foot-line {
  width: 36%;
  height: 18rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
}

.sk-orb,
.sk-legend-bar,
.sk-icon,
.sk-line,
.sk-foot-line {
  animation: sk-breathe 1.5s $animation-easing-default infinite alternate;
}

@keyframes sk-breathe {
  from {
    opacity: 0.45;
  }

  to {
    opacity: 0.9;
  }
}

// ---- 失败态 --------------------------------------------------
.error-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 0;
  padding: 72rpx 40rpx 56rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
  text-align: center;
}

.error-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 112rpx;
  height: 112rpx;
  margin-bottom: 22rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
}

.error-glyph {
  color: $color-semantic-danger;
  font-family: remixicon;
  font-size: 56rpx;
  line-height: 1;
}

.error-title {
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.error-desc {
  margin-top: 10rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

@media (prefers-reduced-motion: reduce) {
  .changes-caret,
  .expand-caret {
    transition: none;
  }

  .event-card.is-flash,
  .sk-orb,
  .sk-legend-bar,
  .sk-icon,
  .sk-line,
  .sk-foot-line {
    animation: none;
  }
}
</style>
