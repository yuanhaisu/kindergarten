<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchTeacherClassInfo, fetchTeacherClassStudents } from "@/api/teacher-class"
import { fetchTeacherPickupOverview } from "@/api/teacher-pickup"
import { fetchTeacherNotices } from "@/api/teacher-notices"
import { formatDateLabel, formatMonthDay, formatWeekdayShort, todayKey } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"
import BottomNav from "@/components/BottomNav.vue"

// 顺序与设计稿一致；计数与过滤共用同一份 key
const FILTERS = [
  { key: "all", label: "全部" },
  { key: "not_arrived", label: "未入园" },
  { key: "not_picked_up", label: "未接走" },
  { key: "picked_up", label: "已接走" }
]
const STATUS_TEXT = { not_arrived: "未入园", not_picked_up: "未接走", picked_up: "已接走" }

const info = ref({})
const students = ref([])
const overview = ref({})
const notices = ref([])
// 通知非关键数据、独立加载：首屏骨架不该等它，但也不能在它回来前先闪一下空位
const noticesLoading = ref(true)
const loading = ref(true)
// 首次成功返回前不把空列表当「班里没有孩子」——错误态必须在空态之前判定
const hasLoaded = ref(false)
const loadError = ref("")
const keyword = ref("")
const filter = ref("all")
const sheetOpen = ref(false)
const sheetStudent = ref(null)

const todayText = `${formatDateLabel(todayKey())} ${formatWeekdayShort(todayKey())}`

const viewState = computed(() => {
  if (loading.value && !hasLoaded.value) return "loading"
  if (loadError.value && !hasLoaded.value) return "error"
  if (hasLoaded.value && students.value.length === 0) return "empty"
  return "normal"
})

const className = computed(() => info.value.name || "我的班级")
const classInitial = computed(() => firstChar(className.value) || "班")
const gradeYearText = computed(() => {
  const grade = String(info.value.grade || "").trim()
  const year = String(info.value.academic_year || "").trim()
  return [grade, year ? `${year} 学年` : ""].filter(Boolean).join(" · ")
})
const teacherName = computed(() => String(info.value.teacher?.teacher_name || "").trim())
const teacherLine = computed(() =>
  [teacherName.value, String(info.value.teacher?.title_label || "").trim()].filter(Boolean).join(" · ")
)
const teacherInitial = computed(() => firstChar(teacherName.value))
const kindergartenName = computed(() => String(info.value.teacher?.kindergarten_name || "").trim())

// 区头以实际返回的名单为准；hero 人数优先用 class/info 的 student_count（两者口径可能不同）
const heroCount = computed(() => {
  const raw = info.value.student_count
  const n = Number(raw)
  return raw === null || raw === undefined || raw === "" || !Number.isFinite(n) ? students.value.length : n
})
const heroCountText = computed(() => `${viewState.value === "empty" ? 0 : heroCount.value} 人`)

const counts = computed(() => {
  const result = { all: students.value.length, not_arrived: 0, not_picked_up: 0, picked_up: 0 }
  students.value.forEach((student) => {
    const status = student?.pickup_status
    if (status === "not_arrived" || status === "not_picked_up" || status === "picked_up") {
      result[status] += 1
    }
  })
  return result
})

const stats = computed(() => {
  if (viewState.value === "empty") {
    return { arrived: 0, not_arrived: 0, picked_up: 0, not_picked_up: 0 }
  }
  return {
    arrived: countOf(overview.value.arrived),
    not_arrived: countOf(overview.value.not_arrived),
    picked_up: countOf(overview.value.picked_up),
    not_picked_up: countOf(overview.value.not_picked_up)
  }
})

const STAT_ROWS = [
  { key: "arrived", label: "已入园", tone: "success" },
  { key: "not_arrived", label: "未入园", tone: "danger" },
  { key: "picked_up", label: "已接走", tone: "info" },
  { key: "not_picked_up", label: "未接走", tone: "warning" }
]

const rosterCountText = computed(() =>
  viewState.value === "empty"
    ? "0 人"
    : `${students.value.length} 人（含 ${counts.value.not_arrived} 名未入园）`
)

// 只按姓名搜：接口不返回学号，搜它恒为空
const filteredStudents = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return students.value.filter((student) => {
    if (filter.value !== "all" && student?.pickup_status !== filter.value) return false
    if (!kw) return true
    return String(student?.name || "").toLowerCase().includes(kw)
  })
})

const noResults = computed(
  () =>
    viewState.value === "normal" &&
    students.value.length > 0 &&
    filteredStudents.value.length === 0
)

// 接送截止是班级级单一值（后端对每个学生写同一个值），不按人取值
const classDeadline = computed(() => {
  const row = students.value.find((student) => student?.pickup_deadline)
  return row ? row.pickup_deadline : "—"
})

function firstChar(text) {
  const name = String(text || "").trim()
  return name ? Array.from(name)[0] : ""
}

function countOf(value) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? n : 0
}

function statusText(student) {
  return STATUS_TEXT[student?.pickup_status] || "—"
}

function kidTime(student) {
  if (student?.arrive_time) return `${student.arrive_time} 入园`
  return student?.pickup_status === "not_arrived" ? "还没入园" : "—"
}

function arriveText(student) {
  return student?.arrive_time || "—"
}

function personsText(student) {
  return String(student?.authorized_persons || "").trim() || "—"
}

function initialOf(student) {
  return firstChar(student?.name) || "幼"
}

function noop() {}

async function loadNotices() {
  // 已有通知时下拉刷新保留旧列表，只有首屏/空列表才显示骨架
  if (!notices.value.length) noticesLoading.value = true
  try {
    const res = await fetchTeacherNotices({ page: 1, page_size: 2 })
    const list = Array.isArray(res.data?.list) ? res.data.list : []
    notices.value = list.slice(0, 2).map((notice) => {
      const rate = parseInt(notice.read_rate, 10)
      return {
        id: notice.id,
        title: notice.title || "未命名通知",
        // 后端 published_at/created_at 是 UTC，直接截字符串会差 8 小时，交给本地时区格式化
        date: formatMonthDay(notice.published_at || notice.created_at),
        readRate: Number.isFinite(rate) ? Math.min(100, Math.max(0, rate)) : 0
      }
    })
  } catch (err) {
    // 通知是非关键数据：失败不拖垮整页，保留上一次成功的结果
    console.warn("[class-index] load notices failed:", err?.message || err)
  } finally {
    noticesLoading.value = false
  }
}

async function loadClass() {
  loading.value = true
  loadError.value = ""
  closeSheet()
  try {
    const [infoRes, studentsRes, overviewRes] = await Promise.all([
      fetchTeacherClassInfo({ silent: true }),
      fetchTeacherClassStudents({}, { silent: true }),
      fetchTeacherPickupOverview({ silent: true })
    ])
    info.value = infoRes.data || {}
    students.value = Array.isArray(studentsRes.data?.list) ? studentsRes.data.list : []
    overview.value = overviewRes.data || {}
    hasLoaded.value = true
    // 关键数据成功后再取通知：整页失败态不该再叠一条通知接口的全局 toast
    await loadNotices()
  } catch (err) {
    if (!hasLoaded.value) {
      // 首次加载失败：整页错误态，不能让空列表冒充「班里没有孩子」
      info.value = {}
      students.value = []
      overview.value = {}
      loadError.value = "班级信息加载失败"
      noticesLoading.value = false
    } else {
      // 已有内容时保留旧数据，只提示本次刷新失败
      uni.showToast({ title: "刷新失败，请重试", icon: "none" })
    }
    console.warn("[class-index] load class failed:", err?.message || err)
  }
  loading.value = false
  uni.stopPullDownRefresh()
}

function reload() {
  hasLoaded.value = false
  loadClass()
}

function selectFilter(key) {
  if (viewState.value === "loading" || filter.value === key) return
  filter.value = key
}

function onSearchInput(event) {
  if (viewState.value === "loading") return
  keyword.value = event?.detail?.value ?? ""
}

function clearFilters() {
  filter.value = "all"
  keyword.value = ""
}

function openStudent(student) {
  if (viewState.value !== "normal") return
  // 搜索框是原生组件，键盘未收起时可能盖住弹层（见 CSS-COMPAT §3）
  if (typeof uni.hideKeyboard === "function") uni.hideKeyboard()
  sheetStudent.value = student
  sheetOpen.value = true
}

function closeSheet() {
  sheetOpen.value = false
  sheetStudent.value = null
}

function onBell() {
  uni.showToast({ title: "打开消息", icon: "none" })
}

function onNotice(notice) {
  uni.showToast({ title: notice?.title ? `打开通知「${notice.title}」` : "打开通知", icon: "none" })
}

onLoad(loadClass)
onPullDownRefresh(loadClass)
</script>

<template>
  <PageShell bottom-space="180rpx" max-width="430px">
    <view class="class-page">
      <!-- 失败：整页错误态 -->
      <view v-if="viewState === 'error'" class="error-panel">
        <view class="state-icon is-danger"><text class="icon icon--lg">&#xf2c2;</text></view>
        <text class="error-title">班级信息加载失败</text>
        <text class="error-text">网络好像开小差了</text>
        <view class="btn-primary" @click="reload">重试</view>
      </view>

      <template v-else>
        <!-- 班级 hero：全页唯一渐变 -->
        <view class="hero">
          <view v-if="viewState === 'loading'" class="hero-sk">
            <view class="hero-sk-top">
              <view class="sk sk--brand sk-badge" />
              <view class="hero-sk-copy">
                <view class="sk sk--brand sk-line sk-line--a" />
                <view class="sk sk--brand sk-line sk-line--b" />
              </view>
              <view class="sk sk--brand sk-bell" />
            </view>
            <view class="sk sk--brand sk-capsule" />
          </view>
          <template v-else>
            <view class="hero-main">
              <view class="hero-badge"><text>{{ classInitial }}</text></view>
              <view class="hero-copy">
                <text v-if="gradeYearText" class="hero-eyebrow">{{ gradeYearText }}</text>
                <text class="hero-title">{{ className }}</text>
                <text v-if="kindergartenName" class="hero-school">{{ kindergartenName }}</text>
              </view>
              <view class="hero-bell" @click="onBell">
                <text class="icon bell-icon">&#xef94;</text>
              </view>
            </view>
            <view class="hero-foot">
              <view v-if="teacherLine" class="teacher-avatar"><text>{{ teacherInitial }}</text></view>
              <text v-if="teacherLine" class="teacher-name">{{ teacherLine }}</text>
              <view v-if="teacherLine" class="hero-sep" />
              <view class="hero-count"><text>{{ heroCountText }}</text></view>
            </view>
          </template>
        </view>

        <!-- 今日接送概览 -->
        <view class="overview">
          <view class="overview-head">
            <text class="section-title">今日接送</text>
            <text class="overview-date">{{ todayText }}</text>
          </view>
          <view class="stats">
            <view v-for="row in STAT_ROWS" :key="row.key" class="stat">
              <view class="stat-label">
                <view class="dot" :class="'dot--' + row.tone" />
                <text>{{ row.label }}</text>
              </view>
              <view v-if="viewState === 'loading'" class="sk sk-num" />
              <text v-else class="stat-value">{{ stats[row.key] }}</text>
            </view>
          </view>
        </view>

        <!-- 最新通知：非关键数据，失败或没有通知时整块不渲染 -->
        <view v-if="viewState === 'loading' || noticesLoading || notices.length" class="notices">
          <view class="section-head"><text class="section-title">最新通知</text></view>
          <view v-if="viewState === 'loading' || noticesLoading" class="notice-list">
            <view v-for="i in 2" :key="'notice-sk-' + i" class="notice notice--sk">
              <view class="sk sk-notice-icon" />
              <view class="notice-body">
                <view class="sk sk-notice-title" />
                <view class="sk sk-notice-time" />
              </view>
              <view class="sk sk-notice-rate" />
            </view>
          </view>
          <view v-else class="notice-list">
            <view v-for="notice in notices" :key="notice.id" class="notice" @click="onNotice(notice)">
              <view class="notice-icon"><text class="icon">&#xef94;</text></view>
              <view class="notice-body">
                <text class="notice-title">{{ notice.title }}</text>
                <text class="notice-time">{{ notice.date }}</text>
              </view>
              <view class="notice-rate">
                <view class="rate-bar"><view class="rate-fill" :style="{ width: notice.readRate + '%' }" /></view>
                <text class="rate-text">已读 {{ notice.readRate }}%</text>
              </view>
            </view>
          </view>
        </view>

        <!-- 幼儿名单 -->
        <view class="roster">
          <view class="roster-head">
            <text class="section-title">幼儿名单</text>
            <view v-if="viewState === 'loading'" class="sk sk-count" />
            <text v-else class="roster-count">{{ rosterCountText }}</text>
          </view>

          <!-- 信息卡打开时移除原生 input，避免老基础库上盖层压不住原生组件（CSS-COMPAT §3） -->
          <view
            v-if="viewState !== 'empty' && !sheetOpen"
            class="toolbar"
            :class="{ 'is-disabled': viewState === 'loading' }"
          >
            <view class="search">
              <text class="icon search-icon">&#xf0d1;</text>
              <input
                class="search-input"
                placeholder="搜索学生姓名"
                confirm-type="search"
                :value="keyword"
                :disabled="viewState === 'loading'"
                @input="onSearchInput"
              />
            </view>
            <view class="filters">
              <view
                v-for="item in FILTERS"
                :key="item.key"
                class="filter"
                :class="{ 'is-active': filter === item.key }"
                @click="selectFilter(item.key)"
              >
                <text class="filter-label">{{ item.label }}</text>
                <text v-if="viewState !== 'loading'" class="filter-count">{{ counts[item.key] }}</text>
              </view>
            </view>
          </view>

          <view v-if="viewState === 'loading'" class="kid-grid">
            <view v-for="i in 6" :key="'kid-sk-' + i" class="kid kid--sk">
              <view class="sk sk-av" />
              <view class="sk sk-name" />
              <view class="sk sk-status" />
              <view class="sk sk-time" />
            </view>
          </view>

          <view v-else-if="viewState === 'empty'" class="empty">
            <view class="state-icon"><text class="icon icon--lg">&#xf1ee;</text></view>
            <text class="empty-title">班里还没有幼儿档案</text>
            <view class="btn-primary" @click="reload">刷新名单</view>
          </view>

          <view v-else-if="noResults" class="no-results">
            <view class="state-icon state-icon--sm"><text class="icon icon--sm">&#xf0d1;</text></view>
            <text class="no-results-text">没有找到符合条件的孩子</text>
            <view class="link-btn" @click="clearFilters">清除筛选</view>
          </view>

          <view v-else class="kid-grid">
            <view
              v-for="student in filteredStudents"
              :key="student.student_id"
              class="kid"
              :class="'kid--' + (student.pickup_status || 'unknown')"
              @click="openStudent(student)"
            >
              <view class="kid-av"><text>{{ initialOf(student) }}</text></view>
              <text class="kid-name">{{ student.name || "未命名" }}</text>
              <view class="kid-status">
                <view class="dot" />
                <text>{{ statusText(student) }}</text>
              </view>
              <text class="kid-time">{{ kidTime(student) }}</text>
            </view>
          </view>
        </view>
      </template>
    </view>

    <!-- 幼儿简要信息卡：只读，没有任何写操作 -->
    <view v-if="sheetOpen && sheetStudent" class="sheet-layer">
      <view class="sheet-scrim" @click="closeSheet" @touchmove.stop.prevent="noop" />
      <view class="sheet" @touchmove.stop.prevent="noop">
        <view class="sheet-grab" />
        <view class="sheet-head">
          <view class="sheet-av"><text>{{ initialOf(sheetStudent) }}</text></view>
          <view class="sheet-id">
            <text class="sheet-name">{{ sheetStudent.name || "未命名" }}</text>
            <view class="sheet-pill" :class="'is-' + (sheetStudent.pickup_status || 'unknown')">
              <view class="dot" />
              <text>{{ statusText(sheetStudent) }}</text>
            </view>
          </view>
        </view>
        <view class="sheet-rows">
          <view class="sheet-row">
            <text class="sheet-label">入园时间</text>
            <text class="sheet-value">{{ arriveText(sheetStudent) }}</text>
          </view>
          <view class="sheet-row">
            <text class="sheet-label">接送状态</text>
            <text class="sheet-value">{{ statusText(sheetStudent) }}</text>
          </view>
          <view class="sheet-row">
            <text class="sheet-label">授权接送人</text>
            <text class="sheet-value">{{ personsText(sheetStudent) }}</text>
          </view>
          <view class="sheet-row">
            <text class="sheet-label">接送截止</text>
            <text class="sheet-value">{{ classDeadline }}</text>
          </view>
        </view>
        <view class="sheet-close" @click="closeSheet">关闭</view>
      </view>
    </view>

    <template #bottom>
      <BottomNav :active="1" />
    </template>
  </PageShell>
</template>

<style scoped lang="scss">
.icon {
  font-family: remixicon;
  font-style: normal;
  line-height: 1;
}

.class-page {
  padding-bottom: $spacing-xl;
}

/* ---------- 班级 hero：全页唯一渐变 ---------- */
.hero {
  position: relative;
  overflow: hidden;
  padding: 28rpx $spacing-xl $spacing-lg;
  border-radius: $border-radius-2xl;
  background: $gradient-brand;
  box-shadow: $shadow-md;
}

/* 装饰圆：设计稿用 on-brand 9% 叠一层，token 派生（同色相，不新建色值） */
.hero::after {
  content: "";
  position: absolute;
  top: -96rpx;
  right: -72rpx;
  width: 256rpx;
  height: 256rpx;
  border-radius: $border-radius-full;
  background: rgba($color-text-on-brand, 0.09);
}

.hero-main,
.hero-foot,
.hero-sk {
  position: relative;
  z-index: 1;
}

.hero-main {
  display: flex;
  align-items: center;
}

.hero-badge {
  flex: none;
  width: 96rpx;
  height: 96rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: rgba($color-text-on-brand, 0.2);
  color: $color-text-on-brand;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-bold;
  line-height: 1;
}

.hero-copy {
  flex: 1;
  min-width: 0;
  margin-left: $spacing-md;
}

.hero-eyebrow,
.hero-school {
  display: block;
  font-size: $typography-font-size-sm;
  line-height: 1.35;
  color: rgba($color-text-on-brand, 0.78);
}

.hero-title {
  display: block;
  margin: 4rpx 0;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-bold;
  line-height: 1.2;
  color: $color-text-on-brand;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.hero-bell {
  flex: none;
  width: $component-input-height;
  height: $component-input-height;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-left: $spacing-md;
  border-radius: $border-radius-full;
  background: rgba($color-text-on-brand, 0.22);
}

.hero-bell:active {
  background: rgba($color-text-on-brand, 0.32);
}

.bell-icon {
  font-size: $typography-font-size-3xl;
  color: $color-text-on-brand;
}

.hero-foot {
  display: flex;
  align-items: center;
  margin-top: 20rpx;
}

.teacher-avatar {
  flex: none;
  width: 48rpx;
  height: 48rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: $spacing-md;
  border-radius: $border-radius-full;
  background: rgba($color-text-on-brand, 0.24);
  color: $color-text-on-brand;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.teacher-name {
  flex: 0 1 auto;
  min-width: 0;
  margin-right: $spacing-md;
  font-size: $typography-font-size-sm;
  color: rgba($color-text-on-brand, 0.9);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.hero-sep {
  flex: none;
  width: 6rpx;
  height: 6rpx;
  margin-right: $spacing-md;
  border-radius: $border-radius-full;
  background: rgba($color-text-on-brand, 0.55);
}

.hero-count {
  flex: none;
  margin-left: auto;
  padding: 6rpx 20rpx;
  border-radius: $border-radius-full;
  background: rgba($color-text-on-brand, 0.18);
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  color: $color-text-on-brand;
}

/* ---------- 今日接送 ---------- */
.overview {
  margin-top: $spacing-lg;
  padding: $spacing-lg 28rpx 28rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.overview-head,
.section-head,
.roster-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.section-title {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
}

.overview-date {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.stats {
  display: flex;
  margin-top: 20rpx;
}

.stat {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  margin-right: $spacing-xs;
}

.stat:last-child {
  margin-right: 0;
}

.stat-label {
  display: flex;
  align-items: center;
  font-size: $typography-font-size-xs;
  line-height: 1.3;
  color: $color-text-muted;
}

.stat-value {
  margin-top: 6rpx;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.1;
  color: $color-text-primary;
}

.dot {
  flex: none;
  width: 12rpx;
  height: 12rpx;
  margin-right: $spacing-xs;
  border-radius: $border-radius-full;
  background: $color-text-hint;
}

.dot--success { background: $color-semantic-success; }
.dot--danger { background: $color-semantic-danger; }
.dot--info { background: $color-semantic-info; }
.dot--warning { background: $color-semantic-warning; }

/* ---------- 最新通知 ---------- */
.notices {
  margin-top: $spacing-lg;
}

.section-head {
  margin-bottom: 20rpx;
}

.notice-list {
  display: flex;
  flex-direction: column;
}

.notice {
  display: flex;
  align-items: center;
  padding: 20rpx $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.notice + .notice {
  margin-top: $spacing-sm;
}

.notice:active {
  transform: scale(0.99);
}

.notice--sk {
  box-shadow: none;
}

.notice-icon {
  flex: none;
  width: 72rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 20rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.notice-icon .icon {
  font-size: $typography-font-size-2xl;
}

.notice-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.notice-title {
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  color: $color-text-primary;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.notice-time {
  margin-top: 4rpx;
  font-size: $typography-font-size-xs;
  line-height: 1.3;
  color: $color-text-hint;
}

.notice-rate {
  flex: none;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  margin-left: 20rpx;
}

.rate-bar {
  width: 112rpx;
  height: 10rpx;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  overflow: hidden;
}

.rate-fill {
  height: 100%;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.rate-text {
  margin-top: 6rpx;
  font-size: $typography-font-size-xs;
  line-height: 1.2;
  color: $color-text-muted;
}

/* 骨架：与真实卡片同尺寸，切态时不跳动 */
.sk-notice-icon {
  flex: none;
  width: 72rpx;
  height: 72rpx;
  margin-right: 20rpx;
  border-radius: $border-radius-full;
}

.sk-notice-title {
  width: 72%;
  height: 20rpx;
  border-radius: $border-radius-full;
}

.sk-notice-time {
  width: 34%;
  height: 16rpx;
  margin-top: 4rpx;
  border-radius: $border-radius-full;
}

.sk-notice-rate {
  flex: none;
  width: 112rpx;
  height: 10rpx;
  margin-left: 20rpx;
  border-radius: $border-radius-full;
}

/* ---------- 幼儿名单 ---------- */
.roster {
  margin-top: $spacing-lg;
}

.roster-count {
  flex: none;
  margin-left: $spacing-md;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  text-align: right;
  color: $color-text-muted;
}

.toolbar {
  margin-bottom: $spacing-md;
}

.toolbar.is-disabled {
  opacity: 0.42;
}

.search {
  display: flex;
  align-items: center;
  height: $component-input-height;
  padding: 0 $spacing-lg;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.search-icon {
  flex: none;
  margin-right: $spacing-md;
  font-size: $typography-font-size-2xl;
  color: $color-text-hint;
}

.search-input {
  flex: 1;
  min-width: 0;
  height: 100%;
  font-size: $typography-font-size-base;
  color: $color-text-primary;
}

.filters {
  display: flex;
  margin-top: $spacing-sm;
}

.filter {
  flex: 1;
  min-width: 0;
  min-height: $component-input-height;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: $spacing-xs;
  padding: 0 4rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
}

.filter:last-child {
  margin-right: 0;
}

.filter-label {
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
  color: $color-text-body;
}

.filter-count {
  margin-left: 8rpx;
  font-size: $typography-font-size-xs;
  line-height: 1;
  color: $color-text-muted;
}

.filter.is-active {
  border-color: $color-brand-primary;
  background: $color-brand-primary-dark;
}

.filter.is-active .filter-label {
  color: $color-text-on-brand;
  font-weight: $typography-font-weight-semibold;
}

.filter.is-active .filter-count {
  color: rgba($color-text-on-brand, 0.82);
}

/* 3 列网格：用 margin 而非 gap，老基础库 gap 会静默变 0（见 CSS-COMPAT §1） */
.kid-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin: 0 (-$spacing-xs);
}

.kid {
  min-width: 0;
  min-height: 224rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  margin: 0 $spacing-xs $spacing-md;
  padding: 20rpx $spacing-xs 18rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.kid:active {
  transform: scale(0.98);
}

.kid-av {
  flex: none;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 6rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.kid-name {
  max-width: 100%;
  margin-bottom: 6rpx;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
  color: $color-text-primary;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.kid-status {
  display: flex;
  align-items: center;
  font-size: $typography-font-size-xs;
  line-height: 1.3;
  color: $color-text-muted;
}

.kid-time {
  max-width: 100%;
  margin-top: 6rpx;
  font-size: $typography-font-size-xs;
  line-height: 1.2;
  color: $color-text-muted;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.kid--picked_up .kid-status .dot { background: $color-semantic-info; }
.kid--not_picked_up .kid-status .dot { background: $color-semantic-warning; }
.kid--not_arrived .kid-status .dot { background: $color-semantic-danger; }

/* ---------- 空结果 / 空名单 / 错误 ---------- */
.no-results {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 56rpx $spacing-lg;
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
}

.state-icon--sm {
  width: 88rpx;
  height: 88rpx;
}

.state-icon.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.icon--lg {
  font-size: 60rpx;
}

.icon--sm {
  font-size: 44rpx;
}

.no-results-text {
  margin-top: $spacing-md;
  font-size: $typography-font-size-base;
  color: $color-text-muted;
}

.link-btn {
  min-height: $component-input-height;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 12rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  color: $color-brand-primary-dark;
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 72rpx $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.empty-title {
  margin-top: 20rpx;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
}

.btn-primary {
  min-height: $component-input-height;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 20rpx;
  padding: 0 40rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.btn-primary:active {
  opacity: 0.9;
}

.error-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 112rpx 40rpx;
}

.error-title {
  margin-top: 20rpx;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  color: $color-text-primary;
}

.error-text {
  margin-top: $spacing-md;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  color: $color-text-muted;
}

/* ---------- 幼儿简要信息卡（只读） ---------- */
.sheet-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 60;
}

.sheet-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba($color-text-primary, 0.36);
}

.sheet {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  max-width: 430px;
  margin: 0 auto;
  padding: $spacing-md 40rpx calc(40rpx + env(safe-area-inset-bottom));
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.sheet-grab {
  width: 72rpx;
  height: 8rpx;
  margin: 0 auto 28rpx;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
}

.sheet-head {
  display: flex;
  align-items: center;
  min-width: 0;
}

.sheet-av {
  flex: none;
  width: $component-avatar-md;
  height: $component-avatar-md;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.sheet-id {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.sheet-name {
  max-width: 100%;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
  color: $color-text-primary;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.sheet-pill {
  display: inline-flex;
  align-items: center;
  margin-top: 8rpx;
  padding: 6rpx 18rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.sheet-pill .dot {
  width: 12rpx;
  height: 12rpx;
  margin-right: 10rpx;
  background: currentColor;
}

.sheet-pill.is-picked_up {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.sheet-pill.is-not_picked_up {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}

.sheet-pill.is-not_arrived {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.sheet-rows {
  margin: $spacing-xl 0 36rpx;
  padding: 0 $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-page;
}

.sheet-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: $spacing-lg 0;
  border-bottom: 1rpx solid $color-border-subtle;
}

.sheet-row:last-child {
  border-bottom: 0;
}

.sheet-label {
  flex: none;
  margin-right: $spacing-xl;
  font-size: $typography-font-size-base;
  line-height: 1.4;
  color: $color-text-muted;
}

.sheet-value {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  text-align: right;
  color: $color-text-primary;
  word-break: break-all;
}

.sheet-close {
  min-height: $component-input-height;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

/* ---------- 骨架 ---------- */
.sk {
  display: block;
  border-radius: $border-radius-sm;
  background-image: linear-gradient(
    90deg,
    $color-bg-page 0%,
    $color-brand-primary-bg 50%,
    $color-bg-page 100%
  );
  background-size: 200% 100%;
  animation: class-shimmer 1.6s linear infinite;
}

.sk--brand {
  background-image: linear-gradient(
    90deg,
    rgba($color-text-on-brand, 0.18) 0%,
    rgba($color-text-on-brand, 0.32) 50%,
    rgba($color-text-on-brand, 0.18) 100%
  );
}

.hero-sk-top {
  display: flex;
  align-items: center;
}

.hero-sk-copy {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  margin-left: $spacing-md;
}

.sk-badge {
  flex: none;
  width: 96rpx;
  height: 96rpx;
  border-radius: $border-radius-full;
}

.sk-bell {
  flex: none;
  width: $component-input-height;
  height: $component-input-height;
  margin-left: $spacing-md;
  border-radius: $border-radius-full;
}

.sk-line {
  height: 20rpx;
  border-radius: $border-radius-full;
}

.sk-line--a {
  width: 62%;
}

.sk-line--b {
  width: 42%;
  margin-top: $spacing-sm;
}

.sk-capsule {
  width: 46%;
  height: 40rpx;
  margin-top: $spacing-lg;
  border-radius: $border-radius-full;
}

.sk-num {
  width: 52rpx;
  height: 40rpx;
  margin-top: 6rpx;
}

.sk-count {
  width: 68rpx;
  height: 24rpx;
}

.kid--sk {
  box-shadow: none;
}

.sk-av {
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  margin-bottom: 6rpx;
  border-radius: $border-radius-full;
}

.sk-name {
  width: 64%;
  height: 20rpx;
  margin-bottom: 6rpx;
  border-radius: $border-radius-full;
}

.sk-status {
  width: 52%;
  height: 16rpx;
  margin-bottom: 6rpx;
  border-radius: $border-radius-full;
}

.sk-time {
  width: 44%;
  height: 16rpx;
  border-radius: $border-radius-full;
}

@keyframes class-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
