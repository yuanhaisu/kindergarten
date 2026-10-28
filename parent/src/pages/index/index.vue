<script setup>
import { computed, ref } from "vue"
import { onLoad, onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchParentHomeOverview, fetchHomeFeeds, switchStudent } from "@/api/parent-home"
import { fetchTasks } from "@/api/parent-task"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId, setCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { bindParentPhone } from "@/api/auth"
import { parseSceneTaskId } from "@/utils/taskScene"
import PageShell from "@/components/PageShell.vue"
import AppIcon from "@/components/AppIcon.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局加载。
// 沿用旧首页的字形映射，不为示意图新增位图；4 个入口仍走 AppIcon + PNG。
const ICON = {
  settings: "", // settings-line
  user: "", // user-line
  attendance: "", // checkbox-circle-line
  meal: "", // bowl-line
  sleep: "", // moon-line
  mood: "", // emotion-happy-line
  star: "", // star-fill
  book: "", // book-open-line
  chevron: "", // arrow-right-s-line
  leaf: "", // leaf-line
  error: "" // error-warning-line
}

const needBindPhone = ref(false)
const bindingPhone = ref(false)
const bindPhoneErr = ref("")

const loading = ref(false)
const retrying = ref(false)
const switching = ref(false)
const hasLoaded = ref(false)
const overview = ref({})
const feeds = ref([])
const todayTask = ref(null)
const loadError = ref("")
// 每次 loadHome 自增；切孩子与 onShow 可能并发，只允许最后一次请求写数据，
// 否则先发出的旧孩子响应晚到会把 feeds 覆盖回上一个孩子。
let loadSeq = 0

const currentStudent = computed(() => overview.value.current_student || {})
const students = computed(() => overview.value.students || [])
const summary = computed(() => overview.value.today_summary || {})
const heroInitial = computed(() => (currentStudent.value.name || "?").slice(0, 1))

// relation_label 由服务端映射（含 other 关系），页面不再维护本地 relation map。
const relationLabel = computed(() => {
  const stu = students.value.find(s => s.id === currentStudent.value.id)
  return stu?.relation_label || "家长"
})

const todayDateLabel = computed(() => {
  const m = String(overview.value.date || "").match(/^(\d{4})-(\d{2})-(\d{2})$/)
  const date = m ? `${m[1]}年${Number(m[2])}月${Number(m[3])}日` : ""
  const weekday = overview.value.weekday || ""
  return [date, weekday].filter(Boolean).join(" ")
})

// 缺字段显示「—」，不用假数据把空值伪装成正常结果。
const todayStatuses = computed(() => [
  {
    key: "attendance",
    label: "出勤",
    tone: "success",
    icon: ICON.attendance,
    value: summary.value.attendance_label || "—",
    sub: summary.value.arrive_time || ""
  },
  {
    key: "meal",
    label: "进餐",
    tone: "info",
    icon: ICON.meal,
    value: summary.value.meal_label || "—",
    sub: ""
  },
  {
    key: "sleep",
    label: "午睡",
    tone: "warning",
    icon: ICON.sleep,
    value: summary.value.sleep_label || "—",
    sub: summary.value.nap_duration ? summary.value.nap_duration + "分钟" : ""
  },
  {
    key: "mood",
    label: "情绪",
    tone: "accent",
    icon: ICON.mood,
    value: summary.value.mood_label || "—",
    sub: ""
  }
])

// 顺序与文案按设计稿 parent/01-home：通知公告 / 接送管理 / 亲子任务 / 我要请假。
// tab=true 的入口在 pages.json 的 tabBar.list 里，必须 switchTab；
// 其余是普通页面，用 navigateTo 保持页面栈语义。成长不放入口，只走空态 CTA。
const quickActions = computed(() => [
  {
    key: "notice",
    title: "通知公告",
    icon: "notice",
    path: "/pages/notice/index",
    tab: false,
    badge: Number(summary.value.new_notice_count) || 0
  },
  {
    key: "pickup",
    title: "接送管理",
    icon: "pickup",
    path: "/pages/pickup/index",
    tab: true,
    badge: 0
  },
  {
    key: "task",
    title: "亲子任务",
    icon: "task",
    path: "/pages/tasks/index",
    tab: false,
    badge: Number(summary.value.pending_tasks) || 0
  },
  {
    key: "leave",
    title: "我要请假",
    icon: "leave",
    path: "/pages/leave/create",
    tab: false,
    badge: 0
  }
])

const showTodaySkeleton = computed(
  () => !hasLoaded.value || (loading.value && !overview.value.date)
)
const showFeedSkeleton = computed(
  () => (!hasLoaded.value || loading.value) && feeds.value.length === 0
)
const isEmptyFeed = computed(
  () =>
    hasLoaded.value &&
    !needBindPhone.value &&
    !loading.value &&
    !loadError.value &&
    feeds.value.length === 0
)
// 空态按 brief 决策隐藏「今晚一件事」；正常态只由今日任务是否存在决定。
const showTonight = computed(
  () => !needBindPhone.value && !loading.value && !loadError.value && !isEmptyFeed.value && Boolean(todayTask.value)
)

async function loadHome(options) {
  const preserveError = Boolean(options && options.preserveError)
  const seq = ++loadSeq
  loading.value = true
  if (!preserveError) loadError.value = ""

  try {
    await ensureLogin()
    // 任务接口要求 student_id 必填且必须属于当前家长，不再回退到第一个孩子。
    // 有缓存孩子时 overview / feeds / 任务三路并发；overview 解析出的孩子变了、
    // 或早期请求失败，再用新 sid 重取，避免 overview 串行挡住首屏。
    const cachedSid = getCurrentStudentId()
    const overviewP = fetchParentHomeOverview(cachedSid)
    const earlyFeedsP = cachedSid
      ? fetchHomeFeeds({ page: 1, page_size: 5, student_id: cachedSid }).catch(() => null)
      : null
    const earlyTasksP = cachedSid
      ? fetchTasks({ student_id: cachedSid, page: 1, page_size: 20 }).catch(() => null)
      : null

    const overviewRes = await overviewP
    if (seq !== loadSeq) return
    overview.value = overviewRes.data || {}
    syncCurrentStudentFromOverview(overviewRes.data)
    const sid = getCurrentStudentId()

    let feedsRes = null
    let tasksRes = null
    if (cachedSid && sid === cachedSid) {
      [feedsRes, tasksRes] = await Promise.all([earlyFeedsP, earlyTasksP])
    }
    if (!feedsRes || !tasksRes) {
      [feedsRes, tasksRes] = await Promise.all([
        fetchHomeFeeds({ page: 1, page_size: 5, student_id: sid || undefined }),
        fetchTasks({ student_id: sid || undefined, page: 1, page_size: 20 })
      ])
    }
    if (seq !== loadSeq) return
    todayTask.value = (tasksRes.data?.list || []).find(t => !t.submitted_current_period) || null
    feeds.value = feedsRes.data?.list || []
    loadError.value = ""
    needBindPhone.value = false
    hasLoaded.value = true
  } catch (err) {
    if (seq !== loadSeq) return
    const msg = err?.message || ""
    // 不把原始错误直接丢给家长；同时保留 console 线索，失败不是静默的。
    console.warn("[parent-home] load failed:", msg || err)
    if (msg.includes("未绑定") || msg.includes("孩子信息") || msg.includes("联系管理员")) {
      needBindPhone.value = true
      loadError.value = ""
    } else {
      loadError.value = "首页数据加载失败"
    }
  } finally {
    // 旧请求收尾时不能把新请求的 loading 关掉
    if (seq === loadSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
  }
}

async function handleRetry() {
  if (loading.value) return
  retrying.value = true
  await loadHome({ preserveError: true })
  retrying.value = false
  if (loadError.value) {
    uni.showToast({ title: "仍无法加载，请稍后再试", icon: "none" })
  } else {
    uni.showToast({ title: "已重新加载", icon: "none" })
  }
}

async function handleGetPhoneNumber(e) {
  if (!e.detail.code) {
    bindPhoneErr.value = "获取手机号失败，请重试"
    return
  }
  bindingPhone.value = true
  bindPhoneErr.value = ""
  try {
    await bindParentPhone(e.detail.code)
    needBindPhone.value = false
    await loadHome()
  } catch (err) {
    bindPhoneErr.value = err?.message || "手机号绑定失败，请确认管理员已为您创建家长账号"
  } finally {
    bindingPhone.value = false
  }
}

async function handleSwitchStudent(student) {
  if (switching.value || !student || student.id === currentStudent.value.id) return
  const prevId = currentStudent.value.id
  switching.value = true
  // 先落盘：服务端的 switch-student 只校验归属后原样返回，不写任何存储，
  // 当前孩子完全由本地维护。接口失败也不该阻止本地切换。
  setCurrentStudentId(student.id)
  try {
    await switchStudent(student.id)
  } catch (err) {
    console.warn("[parent-home] switch-student failed:", err?.message || err)
  }

  await loadHome()
  // 失败有两种形态：请求报错，或后端把不属于该家长的 id 静默回退成第一个孩子。
  // 两种都不能让「选中 A、内容却是 B」留在页面上。
  if (loadError.value || currentStudent.value.id !== student.id) {
    setCurrentStudentId(prevId)
    uni.showToast({ title: "切换孩子失败，请重试", icon: "none" })
    await loadHome()
  } else {
    uni.showToast({ title: `已切换到 ${student.name || "该孩子"}`, icon: "none" })
  }
  switching.value = false
}

function handleAction(action) {
  if (action.tab) {
    uni.switchTab({ url: action.path })
  } else {
    uni.navigateTo({ url: action.path })
  }
}

function goSettings() {
  uni.navigateTo({ url: "/pages/settings/index" })
}

function goTask(id) {
  uni.navigateTo({ url: `/pages/tasks/detail?id=${id}` })
}

function goGrowth() {
  uni.switchTab({ url: "/pages/growth/index" })
}

function goFeed(feed) {
  if (feed.type === "notice") {
    uni.navigateTo({ url: `/pages/notice/detail?id=${feed.id}` })
  } else {
    uni.navigateTo({ url: `/pages/growth/detail?id=${feed.id}` })
  }
}

function badgeText(value) {
  const n = Number(value) || 0
  return n > 99 ? "99+" : String(n)
}

function formatTime(ts) {
  if (!ts) return ""
  const d = new Date(ts)
  if (Number.isNaN(d.getTime())) return String(ts).slice(0, 10)
  const diff = Date.now() - d.getTime()
  if (diff >= 0 && diff < 3600000) return Math.max(1, Math.floor(diff / 60000)) + "分钟前"
  if (diff >= 0 && diff < 86400000) return Math.floor(diff / 3600000) + "小时前"
  // 早于 24 小时按本地日期显示；直接用 ISO 串切片会取到 UTC 日期。
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const day = String(d.getDate()).padStart(2, "0")
  return `${d.getFullYear()}-${m}-${day}`
}

// onShow：tabBar 页 switchTab 切回来不会重跑 onLoad，从详情返回也要刷新。
onShow(() => loadHome())

// 扫任务分享码落到首页时（码的 page 可能是首页），scene 里带 t=<task_id>。
onLoad((options) => {
  const taskId = parseSceneTaskId(options?.scene)
  if (taskId) uni.navigateTo({ url: `/pages/tasks/detail?id=${taskId}` })
})

onPullDownRefresh(() => loadHome())
</script>

<template>
  <PageShell bottom-space="180rpx" max-width="430px" flush-top>
    <!-- ===== 孩子 hero：整页唯一的暖色块 ===== -->
    <view class="hero">
      <view class="hero-avatar">
        <image v-if="currentStudent.avatar" class="hero-avatar-img" :src="currentStudent.avatar" mode="aspectFill" />
        <text v-else-if="currentStudent.name" class="hero-avatar-initial">{{ heroInitial }}</text>
        <text v-else class="hero-avatar-icon">{{ ICON.user }}</text>
      </view>
      <view class="hero-body">
        <view class="hero-name-row">
          <text class="hero-name">{{ currentStudent.name || "孩子" }}</text>
          <text v-if="currentStudent.class_name" class="class-pill">{{ currentStudent.class_name }}</text>
        </view>
        <text class="hero-greeting">{{ relationLabel }} 您好~</text>
      </view>
      <view class="settings-btn" aria-label="设置" @click="goSettings">
        <text class="settings-icon">{{ ICON.settings }}</text>
      </view>
    </view>

    <!-- ===== 未绑定手机号：保留原有内联绑定流程，只做视觉适配 ===== -->
    <view v-if="needBindPhone" class="bind-guide">
      <view class="bind-card">
        <text class="bind-icon">{{ ICON.user }}</text>
        <text class="bind-title">绑定手机号以关联孩子信息</text>
        <text class="bind-desc">请输入或授权微信绑定的手机号，系统将自动匹配管理员为您创建的家长账号</text>
        <text class="bind-consent">授权手机号仅用于匹配园所为您创建的家长账号，不会对外公开</text>
        <button
          class="bind-phone-btn"
          open-type="getPhoneNumber"
          :disabled="bindingPhone"
          :loading="bindingPhone"
          @getphonenumber="handleGetPhoneNumber"
        >
          <text v-if="!bindingPhone">微信手机号一键授权</text>
          <text v-else>绑定中...</text>
        </button>
        <text v-if="bindPhoneErr" class="bind-err">{{ bindPhoneErr }}</text>
      </view>
    </view>

    <template v-if="!needBindPhone">
      <!-- ===== 错误态：替换今日区域，重试真实重新请求 ===== -->
      <view v-if="loadError" class="error-card">
        <view class="error-orb">
          <text class="error-icon">{{ ICON.error }}</text>
        </view>
        <text class="error-text">{{ loadError }}</text>
        <view
          class="retry-btn"
          :class="{ 'retry-btn--disabled': retrying }"
          @click="handleRetry"
        >
          <text>{{ retrying ? "重试中…" : "重试" }}</text>
        </view>
      </view>

      <!-- ===== 今日在校情况（视觉锚点） ===== -->
      <view v-else class="today-card">
        <view class="today-head">
          <text class="today-title">今日在校情况</text>
          <text class="today-date">{{ todayDateLabel }}</text>
        </view>

        <view v-if="showTodaySkeleton" class="today-grid">
          <view v-for="i in 4" :key="i" class="today-col">
            <view class="sk sk-orb" />
            <view class="sk sk-line" />
            <view class="sk sk-line sk-line--short" />
          </view>
        </view>

        <view v-else class="today-grid">
          <view v-for="st in todayStatuses" :key="st.key" class="today-col" :aria-label="st.label">
            <view :class="['stat-orb', 'stat-orb--' + st.tone]">
              <text class="stat-orb-icon">{{ st.icon }}</text>
            </view>
            <text class="stat-value">{{ st.value }}</text>
            <text class="stat-sub">{{ st.sub }}</text>
          </view>
        </view>

        <view v-if="!showTodaySkeleton && summary.performance_tag" class="perf-bar">
          <text class="perf-icon">{{ ICON.star }}</text>
          <text class="perf-text">今日获得「{{ summary.performance_tag }}」奖章</text>
        </view>
      </view>

      <!-- ===== 多孩切换：只在多孩家庭出现 ===== -->
      <view v-if="students.length > 1" class="child-switch">
        <view
          v-for="stu in students"
          :key="stu.id"
          class="child-pill"
          :class="{ 'is-active': stu.id === currentStudent.id }"
          @click="handleSwitchStudent(stu)"
        >
          <text class="child-pill-name">{{ stu.name }}</text>
        </view>
      </view>

      <!-- ===== 4 个入口：一行 ===== -->
      <view class="quick-grid">
        <view
          v-for="action in quickActions"
          :key="action.key"
          class="quick-item"
          @click="handleAction(action)"
        >
          <view class="quick-icon">
            <AppIcon :name="action.icon" :size="64" :label="action.title" />
            <view v-if="action.badge > 0" class="quick-badge">
              <text class="quick-badge-text">{{ badgeText(action.badge) }}</text>
            </view>
          </view>
          <text class="quick-label">{{ action.title }}</text>
        </view>
      </view>

      <!-- ===== 今晚一件事：空态隐藏 ===== -->
      <view v-if="showTonight" class="tonight-section">
        <text class="block-title">今晚一件事</text>
        <view class="tonight-card" @click="goTask(todayTask.id)">
          <view class="tonight-row">
            <view class="tonight-icon">
              <text class="tonight-icon-glyph">{{ ICON.book }}</text>
            </view>
            <text class="tonight-name">{{ todayTask.title || "今日小任务" }}</text>
          </view>
          <text v-if="todayTask.description || todayTask.learning_link" class="tonight-desc">
            {{ todayTask.description || todayTask.learning_link }}
          </text>
          <view class="tonight-foot">
            <view class="cta">
              <text class="cta-text">去完成</text>
              <text class="cta-chevron">{{ ICON.chevron }}</text>
            </view>
          </view>
        </view>
      </view>

      <!-- ===== 最新动态：通知 + 成长混合流 ===== -->
      <view v-if="!loadError" class="feed-section">
        <text class="block-title">最新动态</text>

        <view v-if="showFeedSkeleton" class="feed-skeleton">
          <view v-for="i in 2" :key="i" class="skeleton-card">
            <view class="sk sk-chip" />
            <view class="sk sk-title" />
            <view class="sk sk-text" />
            <view class="sk sk-text sk-text--short" />
          </view>
        </view>

        <view v-else-if="isEmptyFeed" class="feed-empty">
          <view class="empty-orb">
            <text class="empty-icon">{{ ICON.leaf }}</text>
          </view>
          <text class="empty-text">暂无最新动态</text>
          <view class="btn-primary" @click="goGrowth">
            <text>看看成长时光</text>
          </view>
        </view>

        <view v-else class="feed-list">
          <view
            v-for="feed in feeds"
            :key="feed.type + '-' + feed.id"
            class="feed-card"
            @click="goFeed(feed)"
          >
            <view class="feed-top">
              <text :class="['type-chip', feed.type === 'notice' ? 'type-chip--notice' : 'type-chip--growth']">
                {{ feed.type === "notice" ? "通知" : "成长" }}
              </text>
            </view>

            <view v-if="feed.type === 'notice'" class="feed-title-row">
              <text class="feed-title">{{ feed.notice?.title || "" }}</text>
              <view v-if="feed.notice?.is_read === false" class="unread-dot" />
            </view>

            <text
              v-if="feed.type === 'notice'"
              class="feed-text feed-text--notice"
            >{{ feed.notice?.content_preview || "" }}</text>
            <text
              v-else
              class="feed-text"
            >{{ feed.growth?.content || "" }}</text>

            <text class="feed-time">{{ formatTime(feed.created_at) }}</text>
          </view>
        </view>
      </view>
    </template>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 家长首页 ·「清晨第一眼」—— 孩子 → 今天 → 该做什么
// 冷雾白底；暖色只留在 hero 与 wash 图标底上。
// ============================================================

// ---- 孩子 hero -------------------------------------------------
.hero {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  // flushTop 去掉了 PageShell 顶部 padding，这里补回设计稿的 page-pad-y（12px）
  margin-top: $spacing-lg;
  padding: $spacing-xl;
  border-radius: $border-radius-2xl;
  background: $gradient-brand-soft;
  box-shadow: $shadow-sm;
}

.hero-avatar {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  // 设计稿 52px；token 最接近的是 avatar-lg（64px），这里按稿取值
  width: 104rpx;
  height: 104rpx;
  border: 4rpx solid $color-bg-card;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  overflow: hidden;
}

.hero-avatar-img {
  width: 100%;
  height: 100%;
  border-radius: $border-radius-full;
}

.hero-avatar-initial {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  line-height: 1;
}

.hero-avatar-icon {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.hero-body {
  flex: 1 1 auto;
  min-width: 0;
}

.hero-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-md;
  min-width: 0;
}

.hero-name {
  max-width: 100%;
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  line-height: 1.25;
  overflow-wrap: anywhere;
}

.class-pill {
  display: inline-block;
  max-width: 100%;
  // 设计稿 2px 垂直内边距，token 没有 4rpx 档
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1.5;
}

.hero-greeting {
  display: block;
  margin-top: 4rpx;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.settings-btn {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: $component-input-height;
  height: $component-input-height;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast $animation-easing-default,
    background $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.94);
    background: $color-brand-primary-bg;
  }
}

.settings-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

// ---- 今日在校情况 ----------------------------------------------
.today-card {
  margin-top: $spacing-lg;
  padding: $spacing-xl;
  border-radius: $border-radius-xl; // radius-card 12px
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.today-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  min-width: 0;
}

.today-title {
  flex: 1 1 auto;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: 600;
  line-height: 1.3;
}

.today-date {
  flex: 0 1 auto;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  text-align: right;
}

.today-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: $spacing-xs;
  margin-top: $spacing-lg;
}

.today-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 0;
  text-align: center;
}

.stat-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  // 设计稿 38px 圆底
  width: 76rpx;
  height: 76rpx;
  border-radius: $border-radius-full;
}

.stat-orb--success {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.stat-orb--info {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.stat-orb--warning {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}

.stat-orb--accent {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.stat-orb-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.stat-value {
  max-width: 100%;
  margin-top: $spacing-sm;
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  font-weight: 600;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.stat-sub {
  max-width: 100%;
  // 占位行高，四列副行有无内容都不会让卡片高度跳动
  min-height: 32rpx;
  margin-top: 2rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.perf-bar {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  margin-top: $spacing-lg;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-brand-primary-bg;
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  line-height: 1.5;
}

.perf-icon {
  flex: none;
  margin-top: 2rpx;
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

.perf-text {
  color: $color-text-primary;
}

// ---- 骨架屏 ----------------------------------------------------
.sk {
  background: $color-border-hairline;
  opacity: 0.42;
  animation: sk-breathe 1.5s $animation-easing-default infinite;
}

.sk-orb {
  width: 76rpx;
  height: 76rpx;
  border-radius: $border-radius-full;
}

.sk-line {
  width: 64rpx;
  height: 12rpx;
  margin-top: $spacing-sm;
  border-radius: $border-radius-full;
}

.sk-line--short {
  width: 40rpx;
}

@keyframes sk-breathe {
  0%,
  100% {
    opacity: 0.3;
  }
  50% {
    opacity: 0.62;
  }
}

// ---- 错误态 ----------------------------------------------------
.error-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-lg;
  // 设计稿 24px 上下内边距
  padding: 48rpx $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-md;
  text-align: center;
}

.error-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  // 设计稿 44px
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.error-icon {
  font-family: remixicon;
  font-size: $typography-font-size-3xl;
  line-height: 1;
}

.error-text {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: 600;
  line-height: 1.5;
}

.retry-btn,
.btn-primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  // 设计稿 22px 水平内边距
  padding: 0 44rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-base;
  font-weight: 600;
  line-height: 1.4;
  box-shadow: $shadow-md;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.99);
    box-shadow: $shadow-sm;
  }
}

.retry-btn--disabled {
  background: $color-border-hairline;
  color: $color-text-hint;
  box-shadow: none;
  pointer-events: none;
}

// ---- 多孩胶囊 --------------------------------------------------
.child-switch {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;
  margin-top: $spacing-xl;
}

.child-pill {
  display: inline-flex;
  align-items: center;
  min-height: $component-input-height;
  padding: 0 $spacing-xl;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: 500;
  line-height: 1.4;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast $animation-easing-default,
    background $animation-duration-fast $animation-easing-default;

  &::before {
    content: "";
    flex: none;
    width: 12rpx;
    height: 12rpx;
    margin-right: $spacing-sm;
    border-radius: $border-radius-full;
    background: $color-border-hairline;
  }

  &:active {
    transform: scale(0.97);
  }
}

.child-pill.is-active {
  border-color: $color-brand-primary-light;
  background: $color-brand-primary-bg;
  color: $color-text-primary;
  box-shadow: none;

  &::before {
    background: $color-brand-primary;
  }

  &:active {
    transform: none;
  }
}

.child-pill-name {
  max-width: 100%;
  overflow-wrap: anywhere;
}

// ---- 4 个入口 --------------------------------------------------
.quick-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: $spacing-xs;
  margin-top: $spacing-xl;
}

.quick-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
  // 设计稿 80px 触控高度
  min-height: 160rpx;
  padding: $spacing-xs 4rpx;
  border-radius: $border-radius-xl;

  &:active .quick-icon {
    transform: scale(0.92);
  }
}

.quick-icon {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  // 设计稿 46px
  width: 92rpx;
  height: 92rpx;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
  transition: transform $animation-duration-fast $animation-easing-default;
}

.quick-label {
  max-width: 100%;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  line-height: 1.4;
  text-align: center;
  overflow-wrap: anywhere;
}

.quick-badge {
  position: absolute;
  top: -10rpx;
  right: -12rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  // 设计稿 17px 数字胶囊
  min-width: 34rpx;
  height: 34rpx;
  padding: 0 8rpx;
  border: 4rpx solid $color-bg-page;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
  color: $color-text-on-brand;
  line-height: 1;
}

.quick-badge-text {
  font-size: $typography-font-size-xs;
  font-weight: 600;
  line-height: 1;
}

// ---- 区块标题 --------------------------------------------------
.block-title {
  display: block;
  margin: 0 0 $spacing-md 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  line-height: 1.4;
}

// ---- 今晚一件事 ------------------------------------------------
.tonight-section,
.feed-section {
  // 设计稿 24px 区块间距；token 最大间距是 32rpx（16px）
  margin-top: 48rpx;
}

.tonight-card {
  display: flex;
  flex-direction: column;
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-md;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.99);
    box-shadow: $shadow-sm;
  }
}

.tonight-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-width: 0;
}

.tonight-icon {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  // 设计稿 32px
  width: 64rpx;
  height: 64rpx;
  border-radius: $border-radius-lg;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.tonight-icon-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.tonight-name {
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: 600;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.tonight-desc {
  // 与 32px 图标 + 8px 间距对齐
  padding-left: 80rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.65;
  overflow-wrap: anywhere;
}

.tonight-foot {
  display: flex;
  justify-content: flex-end;
  padding-left: 80rpx;
  margin-top: $spacing-sm;
}

.cta {
  display: inline-flex;
  align-items: center;
  // 设计稿 30px 高度
  min-height: 60rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-text-primary;
}

.cta-text {
  font-size: $typography-font-size-sm;
  font-weight: 600;
  line-height: 1.4;
}

.cta-chevron {
  margin-left: 4rpx;
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

// ---- 最新动态 --------------------------------------------------
.feed-list {
  display: flex;
  flex-direction: column;
}

.feed-card {
  display: flex;
  flex-direction: column;
  padding: $spacing-lg $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.99);
  }

  // 卡片列表用相邻 margin 而不是 gap，老基础库 gap 塌陷时不会粘成一块
  & + & {
    margin-top: $spacing-md;
  }
}

.feed-top {
  display: flex;
  align-items: center;
  margin-bottom: $spacing-sm;
}

.type-chip {
  display: inline-flex;
  align-items: center;
  // 设计稿 18px 胶囊
  min-height: 36rpx;
  // 设计稿 1px 垂直内边距
  padding: 2rpx $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1.45;
}

.type-chip--notice {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.type-chip--growth {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.feed-title-row {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  min-width: 0;
}

.feed-title {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: 600;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.unread-dot {
  flex: none;
  // 设计稿 7px
  width: 14rpx;
  height: 14rpx;
  margin-top: 10rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
}

.feed-text {
  display: -webkit-box;
  margin-top: $spacing-sm;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.65;
  overflow: hidden;
  overflow-wrap: anywhere;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.feed-text--notice {
  color: $color-text-muted;
}

.feed-time {
  align-self: flex-end;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  text-align: right;
}

// ---- 空态 ------------------------------------------------------
.feed-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-md;
  // 设计稿 24px 上下内边距
  padding: 48rpx $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.empty-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  // 设计稿 48px
  width: 96rpx;
  height: 96rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.empty-icon {
  font-family: remixicon;
  font-size: $typography-font-size-3xl;
  line-height: 1;
}

.empty-text {
  // 设计稿 230px 限宽
  max-width: 460rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.65;
}

// ---- 动态骨架 --------------------------------------------------
.feed-skeleton {
  display: flex;
  flex-direction: column;
}

.skeleton-card {
  display: flex;
  flex-direction: column;
  padding: $spacing-lg $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;

  & + & {
    margin-top: $spacing-md;
  }
}

.sk-chip {
  // 设计稿 40x18px
  width: 80rpx;
  height: 36rpx;
  border-radius: $border-radius-full;
}

.sk-title {
  width: 64%;
  height: 24rpx;
  margin-top: $spacing-sm;
  border-radius: $border-radius-sm;
}

.sk-text {
  width: 100%;
  height: 20rpx;
  margin-top: $spacing-sm;
  border-radius: $border-radius-sm;
}

.sk-text--short {
  width: 58%;
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
  }
}

// ---- 未绑定手机号（保留原流程，仅视觉适配） --------------------
.bind-guide {
  margin-top: $spacing-xl;
}

.bind-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48rpx;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.bind-icon {
  color: $color-text-hint;
  font-family: remixicon;
  font-size: 88rpx;
  line-height: 1;
}

.bind-title {
  margin-top: $spacing-xl;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: 600;
  line-height: 1.4;
}

.bind-desc {
  margin-top: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  text-align: center;
}

.bind-consent {
  margin-top: $spacing-md;
  color: $color-text-hint;
  font-size: $typography-font-size-xs;
  line-height: 1.6;
  text-align: center;
}

.bind-phone-btn {
  width: 100%;
  // 设计稿沿用原按钮高度 48px
  height: 96rpx;
  margin-top: 48rpx;
  padding: 0;
  border: none;
  border-radius: $border-radius-2xl;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-xl;
  font-weight: 600;
  line-height: 96rpx;
  text-align: center;

  &[disabled] {
    opacity: 0.6;
  }
}

.bind-err {
  margin-top: $spacing-lg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  text-align: center;
}
</style>
