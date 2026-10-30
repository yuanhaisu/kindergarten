<script setup>
/**
 * 积分与成就 · 星轨账本
 *
 * 英雄卡回答「我在哪里」：等级、总分、班级名次；
 * 三张账页回答「分从哪来、我在班里第几、点亮了哪些徽章」。
 *
 * 等级进度按 B16 契约降级：仓库没有等级模型，next_level_points 恒为 0，
 * 此时整块进度区不渲染 —— 不写「还差多少」，也不画一条没有来源的阈值。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId } from "@/utils/currentStudent"
import {
  fetchPointsOverview,
  fetchPointsHistory,
  fetchLeaderboard,
  fetchAchievements
} from "@/api/parent-points"
import { formatDateLabel, formatDateTimeLabel, localDateKey } from "@/utils/time"
import { badgeIconKey, historyTitle, historyTone, signedAmount } from "@/utils/points"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形（App.vue 已全局注册字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  star: "\u{f186}", // star-fill
  add: "\u{ea13}", // add-line
  medal: "\u{ef28}", // medal-line
  check: "\u{eb7b}", // check-line
  calendar: "\u{eb27}", // calendar-line
  book: "\u{eadb}", // book-open-line
  image: "\u{ee4b}", // image-line
  health: "\u{ee11}", // heart-pulse-line
  heart: "\u{ee0f}", // heart-line
  task: "\u{f1e8}", // task-line
  alert: "\u{eca1}", // error-warning-line
  close: "\u{eb99}" // close-line
}

const BADGE_ICONS = {
  medal: ICON.medal,
  calendar: ICON.calendar,
  book: ICON.book,
  image: ICON.image,
  health: ICON.health,
  heart: ICON.heart,
  task: ICON.task
}

const TABS = [
  { key: "points", label: "积分" },
  { key: "rank", label: "排行榜" },
  { key: "achievements", label: "成就" }
]

const EMPTY_COPY = {
  points: {
    title: "还没有积分记录",
    desc: "完成任务、坚持出勤都会获得积分",
    action: "查看亲子任务"
  },
  rank: {
    title: "班级还没有排名",
    desc: "老师开启积分活动后会显示本班排行",
    action: "回到积分"
  },
  achievements: {
    title: "还没有获得徽章",
    desc: "老师授予徽章后会出现在这里",
    action: "查看成长记录"
  }
}

// 与设计稿一致：首屏 9 条，之后每点一次「加载更多」再取 9 条。
const HISTORY_PAGE_SIZE = 9

const overview = ref({})
const history = ref([])
const historyTotal = ref(0)
const historyPage = ref(1)
const leaderboard = ref([])
const leaderboardClassTotal = ref(0)
const achievements = ref({ earned: [], not_earned: [], earned_count: 0, total_count: 0 })
const activeTab = ref("points")
// 首屏一律先骨架：onLoad 必定发请求，先渲染空面板会闪一帧「0 积分」。
const loading = ref(true)
const retrying = ref(false)
const loadingMore = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const sheetBadge = ref(null)

// 只允许最后一次请求写数据：重试与首屏加载可能并发。
let loadSeq = 0

const showSkeleton = computed(() => loading.value && !hasLoaded.value)
const showError = computed(() => !loading.value && Boolean(loadError.value) && !hasLoaded.value)

const nextLevel = computed(() => Number(overview.value.next_level_points) || 0)
const showProgress = computed(() => nextLevel.value > 0)
const progressPercent = computed(() => {
  if (!showProgress.value) return 0
  const total = Number(overview.value.total_points) || 0
  return Math.min(100, Math.round((total / nextLevel.value) * 100))
})
const progressGap = computed(() => {
  const total = Number(overview.value.total_points) || 0
  return Math.max(0, nextLevel.value - total)
})

const levelLabel = computed(() => {
  const o = overview.value
  if (!o.level && !o.level_name) return ""
  return [o.level ? `Lv.${o.level}` : "", o.level_name || ""].filter(Boolean).join(" ")
})

// 名次为 0 表示没有班级 / 不在读，此时宁可不显示，也不写「班级第 0 / 0」。
const rankLabel = computed(() => {
  const total = Number(overview.value.class_total) || 0
  const rank = Number(overview.value.rank_in_class) || 0
  if (total <= 0 || rank <= 0) return ""
  return `班级第 ${rank} / ${total}`
})

const activeEmpty = computed(() => {
  if (!hasLoaded.value || showError.value) return false
  if (activeTab.value === "points") return history.value.length === 0
  if (activeTab.value === "rank") return leaderboard.value.length === 0
  return (achievements.value.earned || []).length === 0
})
const showHero = computed(() => !showSkeleton.value && !showError.value && !activeEmpty.value)
const emptyCopy = computed(() => EMPTY_COPY[activeTab.value] || EMPTY_COPY.points)

const historyMeta = computed(() => (historyTotal.value > 0 ? `共 ${historyTotal.value} 条` : ""))
const canLoadMore = computed(() => history.value.length < historyTotal.value)
const allLoaded = computed(() => hasLoaded.value && !canLoadMore.value)

const rankRows = computed(() =>
  (leaderboard.value || []).map((item) => {
    const name = item.student?.name || "同学"
    return {
      key: `${item.rank}-${item.student?.id ?? name}`,
      rank: Number(item.rank) || 0,
      name,
      initial: Array.from(name)[0] || "幼",
      avatar: item.student?.avatar || "",
      points: Number(item.points) || 0,
      isSelf: Boolean(item.is_self)
    }
  })
)

const badgeList = computed(() => {
  const view = (item, earned) => {
    const earnedDate = earned && item.earned_at ? formatDateLabel(localDateKey(item.earned_at)) : ""
    return {
      key: `badge-${item.id ?? item.name}`,
      name: item.name || "成就徽章",
      description: item.description || "",
      iconUrl: item.icon_url || "",
      iconKey: badgeIconKey(item.category),
      points: Number(item.points) || 0,
      earned,
      earnedLabel: earnedDate ? `${earnedDate}获得` : ""
    }
  }
  return [
    ...(achievements.value.earned || []).map((item) => view(item, true)),
    ...(achievements.value.not_earned || []).map((item) => view(item, false))
  ]
})

const earnedCount = computed(() => {
  const n = Number(achievements.value.earned_count)
  return Number.isFinite(n) ? n : (achievements.value.earned || []).length
})
const badgeTotal = computed(() => {
  const n = Number(achievements.value.total_count)
  return Number.isFinite(n) ? n : badgeList.value.length
})

const sheetMeta = computed(() => {
  const badge = sheetBadge.value
  if (!badge) return ""
  if (!badge.earned) return "尚未获得"
  return [badge.earnedLabel, badge.points ? `+${badge.points} 积分` : ""].filter(Boolean).join(" · ")
})

function badgeGlyph(badge) {
  return BADGE_ICONS[badge?.iconKey] || ICON.medal
}

function emptyGlyph() {
  if (activeTab.value === "achievements") return ICON.medal
  return ICON.star
}

async function loadData(options = {}) {
  const keepView = Boolean(options.keepView)
  const seq = ++loadSeq
  // 重试时保留错误面板（retrying 状态就画在面板上），只在成功后清掉错误。
  if (!keepView) {
    loading.value = true
    loadError.value = ""
  }
  try {
    await ensureLogin()
    const sid = getCurrentStudentId()
    const [overviewRes, historyRes, leaderboardRes, achievementsRes] = await Promise.all([
      fetchPointsOverview(sid),
      fetchPointsHistory({ student_id: sid, page: 1, page_size: HISTORY_PAGE_SIZE }),
      fetchLeaderboard({ student_id: sid }),
      fetchAchievements(sid)
    ])
    if (seq !== loadSeq) return
    overview.value = overviewRes?.data || {}
    const list = Array.isArray(historyRes?.data?.list) ? historyRes.data.list : []
    history.value = list
    historyTotal.value = Number(historyRes?.data?.total)
    if (!Number.isFinite(historyTotal.value)) historyTotal.value = list.length
    historyPage.value = Number(historyRes?.data?.page) || 1
    leaderboard.value = Array.isArray(leaderboardRes?.data?.list) ? leaderboardRes.data.list : []
    leaderboardClassTotal.value = Number(leaderboardRes?.data?.class_total) || 0
    achievements.value = achievementsRes?.data || {
      earned: [],
      not_earned: [],
      earned_count: 0,
      total_count: 0
    }
    hasLoaded.value = true
    loadError.value = ""
  } catch (err) {
    if (seq !== loadSeq) return
    console.warn("[points-index] load failed:", err?.message || err)
    if (!keepView) {
      overview.value = {}
      history.value = []
      historyTotal.value = 0
      leaderboard.value = []
      achievements.value = { earned: [], not_earned: [], earned_count: 0, total_count: 0 }
      hasLoaded.value = false
    }
    loadError.value = "积分记录加载失败"
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

async function handleRetry() {
  if (retrying.value || loading.value) return
  retrying.value = true
  await loadData({ keepView: true })
  retrying.value = false
  uni.showToast({
    title: loadError.value ? "仍无法加载，请稍后再试" : "已重新加载",
    icon: "none"
  })
}

async function loadMore() {
  if (loadingMore.value || !canLoadMore.value) return
  loadingMore.value = true
  const page = historyPage.value + 1
  try {
    const res = await fetchPointsHistory({
      student_id: getCurrentStudentId(),
      page,
      page_size: HISTORY_PAGE_SIZE
    })
    const list = Array.isArray(res?.data?.list) ? res.data.list : []
    history.value = [...history.value, ...list]
    // 接口返回空页时收口，避免 total 与实际页数不一致时空转。
    historyTotal.value = list.length ? (Number(res?.data?.total) || history.value.length) : history.value.length
    historyPage.value = Number(res?.data?.page) || page
    uni.showToast({ title: "已加载更早记录", icon: "none" })
  } catch (err) {
    console.warn("[points-index] load more failed:", err?.message || err)
    uni.showToast({ title: "加载失败，请重试", icon: "none" })
  } finally {
    loadingMore.value = false
  }
}

function selectTab(key) {
  if (activeTab.value === key) return
  activeTab.value = key
}

function handleEmptyAction() {
  if (activeTab.value === "rank") {
    activeTab.value = "points"
    uni.showToast({ title: "已回到积分", icon: "none" })
    return
  }
  if (activeTab.value === "achievements") {
    // 成长时光是家长端 tabBar 页，只能 switchTab；亲子任务页不是 tabBar 页。
    uni.switchTab({ url: "/pages/growth/index" })
    return
  }
  uni.navigateTo({ url: "/pages/tasks/index" })
}

function handleLedgerTap(item) {
  uni.showToast({
    title: `${historyTitle(item)} · ${signedAmount(item.amount)} 积分`,
    icon: "none"
  })
}

function handleRankTap(row) {
  uni.showToast({ title: `${row.name} · 本周 ${row.points} 积分`, icon: "none" })
}

function openBadge(badge) {
  sheetBadge.value = badge
}

function closeBadge() {
  sheetBadge.value = null
}

onLoad(() => {
  loadData()
})
</script>

<template>
  <PageShell bottom-space="48rpx" max-width="430px">
    <!-- ===== 英雄卡：等级 / 总分 / 班级名次 ===== -->
    <view v-if="showHero" class="hero-card">
      <view class="hero-glow" />
      <view class="hero-top">
        <view v-if="levelLabel" class="level-chip">
          <text class="level-glyph">{{ ICON.star }}</text>
          <text class="level-text">{{ levelLabel }}</text>
        </view>
        <text v-if="rankLabel" class="rank-chip">{{ rankLabel }}</text>
      </view>
      <view class="points-line">
        <text class="points-value">{{ overview.total_points || 0 }}</text>
        <text class="points-unit">积分</text>
      </view>
      <!-- next_level_points 为 0（无等级模型）时整块不渲染，不编造阈值 -->
      <view v-if="showProgress" class="level-progress">
        <view class="progress-head">
          <text>距离下一等级</text>
          <text class="progress-gap">还差 {{ progressGap }} 分</text>
        </view>
        <view class="progress-track">
          <view class="progress-fill" :style="{ width: progressPercent + '%' }" />
        </view>
        <view class="progress-scale">
          <text>已获得 {{ overview.total_points || 0 }}</text>
          <text>目标 {{ nextLevel }}</text>
        </view>
      </view>
    </view>

    <!-- ===== 三段切换：加载/失败/空态下仍保留可辨 ===== -->
    <view class="tabs" :class="{ 'tabs--flush': !showHero }">
      <view
        v-for="tab in TABS"
        :key="tab.key"
        class="tab"
        :class="{ 'is-active': activeTab === tab.key }"
        @click="selectTab(tab.key)"
      >
        <text class="tab-text">{{ tab.label }}</text>
      </view>
    </view>

    <!-- ===== 加载：3 行骨架 ===== -->
    <view v-if="showSkeleton" class="state-card">
      <view class="skeleton-card">
        <view v-for="i in 3" :key="i" class="skeleton-row">
          <view class="sk sk-circle" />
          <view class="sk-lines">
            <view class="sk sk-line" :class="'sk-w' + i" />
            <view class="sk sk-line sk-line--short" />
          </view>
          <view class="sk sk-amount" />
        </view>
      </view>
      <text class="state-desc">正在加载积分记录…</text>
    </view>

    <template v-else>
      <!-- ===== 失败态 ===== -->
      <view v-if="showError" class="state-card state-card--tall">
        <view class="state-orb state-orb--danger">
          <text class="state-icon state-icon--danger">{{ ICON.alert }}</text>
        </view>
        <text class="state-title">积分记录加载失败</text>
        <text class="state-desc">网络开小差了，请稍后重试</text>
        <view class="state-action" :class="{ 'is-busy': retrying }" @click="handleRetry">
          <text class="state-action-text">{{ retrying ? "正在重试…" : "重试" }}</text>
        </view>
      </view>

      <!-- ===== 空态：文案随 tab 切换 ===== -->
      <view v-else-if="activeEmpty" class="state-card state-card--tall">
        <view class="state-orb">
          <text class="state-icon">{{ emptyGlyph() }}</text>
        </view>
        <text class="state-title">{{ emptyCopy.title }}</text>
        <text class="state-desc">{{ emptyCopy.desc }}</text>
        <view class="state-action" @click="handleEmptyAction">
          <text class="state-action-text">{{ emptyCopy.action }}</text>
        </view>
      </view>

      <!-- ===== 积分 ===== -->
      <view v-else-if="activeTab === 'points'" class="panel">
        <view class="stat-grid">
          <view class="stat-card">
            <text class="stat-label">本周积分</text>
            <text class="stat-value">{{ overview.this_week || 0 }}</text>
          </view>
          <view class="stat-card">
            <text class="stat-label">本月积分</text>
            <text class="stat-value">{{ overview.this_month || 0 }}</text>
          </view>
        </view>
        <view class="panel-head">
          <view class="panel-heading">
            <text class="panel-title">积分明细</text>
            <text class="panel-sub">最近获得与消耗</text>
          </view>
          <text v-if="historyMeta" class="panel-meta">{{ historyMeta }}</text>
        </view>
        <view class="ledger">
          <view
            v-for="item in history"
            :key="item.id"
            class="ledger-row"
            @click="handleLedgerTap(item)"
          >
            <view class="ledger-icon" :class="'is-' + historyTone(item)">
              <text v-if="historyTone(item) === 'earn'" class="ledger-glyph">{{ ICON.add }}</text>
              <view v-else class="ledger-minus" />
            </view>
            <view class="ledger-main">
              <text class="ledger-title">{{ historyTitle(item) }}</text>
              <text v-if="item.description" class="ledger-desc">{{ item.description }}</text>
              <text class="ledger-time">{{ formatDateTimeLabel(item.created_at) }}</text>
            </view>
            <view class="ledger-side">
              <text class="ledger-amount" :class="'is-' + historyTone(item)">{{ signedAmount(item.amount) }}</text>
              <text class="ledger-balance">余额 {{ item.balance }}</text>
            </view>
          </view>
        </view>
        <view class="load-more" :class="{ 'is-disabled': allLoaded || loadingMore }" @click="loadMore">
          <text class="load-more-text">{{ loadingMore ? "加载中…" : allLoaded ? "已经到底了" : "加载更多" }}</text>
        </view>
      </view>

      <!-- ===== 排行榜 ===== -->
      <view v-else-if="activeTab === 'rank'" class="panel">
        <view class="panel-head">
          <view class="panel-heading">
            <text class="panel-title">本周排行榜</text>
            <text class="panel-sub">按本周获得积分排序</text>
          </view>
        </view>
        <view class="rank-list">
          <view
            v-for="row in rankRows"
            :key="row.key"
            class="rank-row"
            :class="{ 'is-self': row.isSelf }"
            @click="handleRankTap(row)"
          >
            <text class="rank-num" :class="{ 'is-top': row.rank <= 3 }">{{ row.rank }}</text>
            <image v-if="row.avatar" class="rank-avatar rank-avatar--image" :src="row.avatar" mode="aspectFill" />
            <view v-else class="rank-avatar">
              <text class="rank-initial">{{ row.initial }}</text>
            </view>
            <view class="rank-name">
              <text class="rank-name-text">{{ row.name }}</text>
              <text v-if="row.isSelf" class="self-pill">我</text>
            </view>
            <view class="rank-points">
              <text class="rank-points-value">{{ row.points }}</text>
              <text class="rank-points-unit">分</text>
            </view>
          </view>
        </view>
        <text class="panel-foot">仅显示本班前 20 名（在园 {{ leaderboardClassTotal }} 人）</text>
      </view>

      <!-- ===== 成就 ===== -->
      <view v-else class="panel">
        <view class="panel-head">
          <view class="panel-heading">
            <text class="panel-title">成就徽章</text>
            <text class="panel-sub">已获得 {{ earnedCount }} / {{ badgeTotal }}</text>
          </view>
        </view>
        <view class="badge-grid">
          <view
            v-for="badge in badgeList"
            :key="badge.key"
            class="badge"
            :class="badge.earned ? 'is-earned' : 'is-locked'"
            @click="openBadge(badge)"
          >
            <view class="badge-icon">
              <image v-if="badge.iconUrl" class="badge-image" :src="badge.iconUrl" mode="aspectFit" />
              <text v-else class="badge-glyph">{{ badgeGlyph(badge) }}</text>
            </view>
            <text class="badge-name">{{ badge.name }}</text>
          </view>
        </view>
        <text class="panel-foot">灰色徽章尚未获得</text>
      </view>
    </template>

    <!-- ===== 徽章详情底部面板 ===== -->
    <view v-if="sheetBadge" class="sheet-mask" @click="closeBadge" @touchmove.stop.prevent />
    <view v-if="sheetBadge" class="badge-sheet">
      <view class="sheet-handle" />
      <view class="sheet-close" @click="closeBadge">
        <text class="sheet-close-glyph">{{ ICON.close }}</text>
      </view>
      <view class="sheet-icon" :class="sheetBadge.earned ? 'is-earned' : 'is-locked'">
        <image v-if="sheetBadge.iconUrl" class="sheet-image" :src="sheetBadge.iconUrl" mode="aspectFit" />
        <text v-else class="sheet-glyph">{{ badgeGlyph(sheetBadge) }}</text>
      </view>
      <text class="sheet-name">{{ sheetBadge.name }}</text>
      <text v-if="sheetBadge.description" class="sheet-desc">{{ sheetBadge.description }}</text>
      <text class="sheet-meta" :class="{ 'is-locked': !sheetBadge.earned }">{{ sheetMeta }}</text>
      <view class="sheet-btn" @click="closeBadge">
        <text class="sheet-btn-text">知道了</text>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 星轨账本：白卡 + 冷雾底，暖橙只落在等级、名次、徽章与主按钮。
// ============================================================

// ---- 英雄卡 -------------------------------------------------
.hero-card {
  position: relative;
  overflow: hidden;
  padding: 36rpx 36rpx 32rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.hero-glow {
  position: absolute;
  top: -108rpx;
  right: -88rpx;
  width: 296rpx;
  height: 296rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  opacity: 0.85;
  pointer-events: none;
}

.hero-top {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-xs;
}

.level-chip {
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  min-height: 52rpx;
  padding: 0 20rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.level-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-xs;
  line-height: 1;
}

.level-text {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
}

.rank-chip {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
}

.points-line {
  position: relative;
  display: flex;
  align-items: baseline;
  gap: $spacing-sm;
  margin-top: $spacing-lg;
}

.points-value {
  color: $color-text-primary;
  font-size: 80rpx;
  font-weight: $typography-font-weight-bold;
  line-height: 1.05;
  letter-spacing: -1rpx;
  font-variant-numeric: tabular-nums;
}

.points-unit {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

.level-progress {
  position: relative;
  margin-top: $spacing-xl;
}

.progress-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.progress-gap {
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

.progress-track {
  height: 12rpx;
  margin-top: $spacing-sm;
  overflow: hidden;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.progress-fill {
  display: block;
  height: 100%;
  border-radius: $border-radius-full;
  background: $gradient-brand;
}

.progress-scale {
  display: flex;
  justify-content: space-between;
  margin-top: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-variant-numeric: tabular-nums;
}

// ---- 三段切换 -----------------------------------------------
.tabs {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-xs;
  margin-top: $spacing-lg;
  padding: $spacing-xs;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.tab {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 88rpx;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;
  transition:
    background-color $animation-duration-fast $animation-easing-default,
    border-color $animation-duration-fast $animation-easing-default,
    transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.97);
  }
}

.tab.is-active {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
}

.tab-text {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.tab.is-active .tab-text {
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

// 英雄卡在加载/失败/空态隐藏时，tab 顶上不留下多余的 12px。
.tabs--flush {
  margin-top: 0;
}

// ---- 面板 ---------------------------------------------------
.panel {
  margin-top: $spacing-lg;
  overflow: hidden;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20rpx;
  padding: $spacing-xl $spacing-xl $spacing-md;
}

.panel-heading {
  min-width: 0;
}

.panel-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.panel-sub {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
}

.panel-meta {
  flex: none;
  max-width: 46%;
  padding: $spacing-xs 18rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  text-align: right;
}

.panel-foot {
  display: block;
  padding: 20rpx $spacing-xl $spacing-xl;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  text-align: center;
}

// ---- 本周 / 本月 --------------------------------------------
.stat-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: $spacing-sm;
  padding: $spacing-xl $spacing-xl 4rpx;
}

.stat-card {
  min-width: 0;
  padding: 20rpx $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-brand-primary-bg;
}

.stat-label {
  display: block;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
}

.stat-value {
  display: block;
  margin-top: 4rpx;
  color: $color-brand-primary-dark;
  font-size: 44rpx;
  font-weight: $typography-font-weight-bold;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

// ---- 积分明细 -----------------------------------------------
.ledger {
  padding: 0 $spacing-md;
}

.ledger-row {
  display: grid;
  grid-template-columns: 72rpx minmax(0, 1fr) auto;
  align-items: start;
  gap: 20rpx;
  min-height: 120rpx;
  padding: $spacing-lg $spacing-md;
  border-radius: $border-radius-xl;
  text-align: left;

  &:active {
    background: $color-brand-primary-bg;
  }
}

.ledger-row + .ledger-row {
  border-top: 1rpx solid $color-border-subtle;
}

.ledger-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  margin-top: 2rpx;
  border-radius: $border-radius-full;
}

.ledger-icon.is-earn {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.ledger-icon.is-spend {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger-strong;
}

.ledger-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.ledger-minus {
  width: 24rpx;
  height: 4rpx;
  border-radius: $border-radius-full;
  background: currentColor;
}

.ledger-main {
  min-width: 0;
}

.ledger-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  overflow-wrap: break-word;
}

.ledger-desc {
  display: block;
  margin-top: 2rpx;
  overflow: hidden;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ledger-time {
  display: block;
  margin-top: 6rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.ledger-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4rpx;
}

.ledger-amount {
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-bold;
  line-height: 1.3;
  font-variant-numeric: tabular-nums;
}

.ledger-amount.is-earn {
  color: $color-brand-primary-dark;
}

.ledger-amount.is-spend {
  color: $color-semantic-danger-strong;
}

.ledger-balance {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.load-more {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin: $spacing-md $spacing-xl $spacing-xl;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;

  &:active {
    background: $color-bg-page;
  }
}

.load-more.is-disabled {
  border-color: transparent;
  background: $color-brand-primary-bg;

  &:active {
    background: $color-brand-primary-bg;
  }
}

.load-more-text {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.load-more.is-disabled .load-more-text {
  color: $color-text-hint;
}

// ---- 排行榜 -------------------------------------------------
.rank-list {
  padding: 0 $spacing-md;
}

.rank-row {
  display: grid;
  grid-template-columns: 52rpx 72rpx minmax(0, 1fr) auto;
  align-items: center;
  gap: 20rpx;
  min-height: 112rpx;
  padding: $spacing-md;
  border-radius: $border-radius-xl;

  &:active {
    background: $color-bg-page;
  }
}

.rank-row + .rank-row {
  border-top: 1rpx solid $color-border-subtle;
}

.rank-row.is-self {
  background: $color-brand-primary-bg;
}

.rank-num {
  color: $color-text-muted;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-bold;
  line-height: 1;
  text-align: center;
  font-variant-numeric: tabular-nums;
}

.rank-num.is-top {
  color: $color-brand-primary-dark;
}

.rank-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.rank-avatar--image {
  display: block;
}

.rank-initial {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.rank-row.is-self .rank-avatar {
  background: $color-bg-card;
}

.rank-row.is-self .rank-initial {
  color: $color-brand-primary-dark;
}

.rank-name {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-sm;
  min-width: 0;
}

.rank-name-text {
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.self-pill {
  flex: none;
  padding: 2rpx 14rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
  font-size: 20rpx;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.6;
}

.rank-points {
  display: flex;
  align-items: baseline;
  gap: 2rpx;
}

.rank-points-value {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-bold;
  font-variant-numeric: tabular-nums;
}

.rank-points-unit {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

// ---- 成就徽章 -----------------------------------------------
.badge-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 28rpx $spacing-sm;
  padding: $spacing-xs 28rpx 20rpx;
}

.badge {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14rpx;
  min-width: 0;
  padding: $spacing-md 0 $spacing-sm;
  border-radius: $border-radius-xl;

  &:active {
    background: $color-brand-primary-bg;
  }
}

.badge-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 104rpx;
  height: 104rpx;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;
}

.badge.is-earned .badge-icon {
  border-color: $color-brand-primary-light;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.badge.is-locked .badge-icon {
  border-color: $color-border-subtle;
  background: $color-brand-primary-bg;
  color: $color-text-hint;
}

.badge-glyph {
  font-family: remixicon;
  font-size: 48rpx;
  line-height: 1;
}

.badge-image {
  width: 56rpx;
  height: 56rpx;
}

.badge-name {
  max-width: 100%;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
  text-align: center;
  overflow-wrap: break-word;
}

.badge.is-locked .badge-name {
  color: $color-text-muted;
}

// ---- 四态 ---------------------------------------------------
.state-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
  margin-top: $spacing-lg;
  padding: 80rpx 40rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-md;
  text-align: center;
}

.state-card--tall {
  justify-content: center;
  min-height: 46vh;
}

.state-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 128rpx;
  height: 128rpx;
  margin-bottom: 4rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.state-orb--danger {
  background: $color-semantic-danger-bg;
}

.state-icon {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: 56rpx;
  line-height: 1;
}

.state-icon--danger {
  color: $color-semantic-danger-strong;
}

.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.state-desc {
  max-width: 520rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: break-word;
}

// 白字压品牌橙不足 4.5:1，主按钮按设计审查改用深色字。
.state-action {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: $spacing-sm;
  padding: 0 48rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-sm;

  &:active {
    transform: scale(0.97);
  }
}

.state-action.is-busy {
  opacity: 0.7;
}

.state-action-text {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
}

// ---- 加载骨架 -----------------------------------------------
.skeleton-card {
  width: 100%;
  padding: $spacing-sm 0;
}

.skeleton-row {
  display: grid;
  grid-template-columns: 72rpx minmax(0, 1fr) 96rpx;
  align-items: center;
  gap: 20rpx;
  padding: $spacing-lg $spacing-xs;
}

.skeleton-row + .skeleton-row {
  border-top: 1rpx solid $color-border-subtle;
}

.sk {
  display: block;
  background: $color-bg-page;
  animation: sk-breathe 1.3s $animation-easing-default infinite alternate;
}

.sk-circle {
  width: 72rpx;
  height: 72rpx;
  border-radius: $border-radius-full;
}

.sk-lines {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  min-width: 0;
}

.sk-line {
  height: 18rpx;
  border-radius: $border-radius-full;
}

.sk-w1 {
  width: 60%;
}

.sk-w2 {
  width: 50%;
}

.sk-w3 {
  width: 65%;
}

.sk-line--short {
  width: 40%;
}

.sk-amount {
  width: 80rpx;
  height: 24rpx;
  border-radius: $border-radius-full;
}

@keyframes sk-breathe {
  0%,
  100% {
    opacity: 0.55;
  }
  50% {
    opacity: 1;
  }
}

// ---- 徽章详情底部面板 ---------------------------------------
.sheet-mask {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 30;
  background: $color-text-primary;
  opacity: 0.28;
}

.badge-sheet {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 31;
  display: flex;
  flex-direction: column;
  align-items: center;
  max-height: 82vh;
  padding: $spacing-md 40rpx calc(40rpx + env(safe-area-inset-bottom));
  overflow-y: auto;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
  text-align: center;
}

.sheet-handle {
  width: 72rpx;
  height: 8rpx;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
}

.sheet-close {
  position: absolute;
  top: 8rpx;
  right: 8rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
}

.sheet-close-glyph {
  color: $color-text-muted;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.sheet-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 152rpx;
  height: 152rpx;
  margin: $spacing-xs 0 20rpx;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;
}

.sheet-icon.is-earned {
  border-color: $color-brand-primary-light;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.sheet-icon.is-locked {
  border-color: $color-border-subtle;
  background: $color-brand-primary-bg;
  color: $color-text-hint;
}

.sheet-glyph {
  font-family: remixicon;
  font-size: 68rpx;
  line-height: 1;
}

.sheet-image {
  width: 88rpx;
  height: 88rpx;
}

.sheet-name {
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.sheet-desc {
  max-width: 560rpx;
  margin-top: 12rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: break-word;
}

.sheet-meta {
  margin-top: $spacing-md;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.sheet-meta.is-locked {
  color: $color-text-muted;
}

.sheet-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 96rpx;
  margin-top: 40rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;

  &:active {
    transform: scale(0.98);
  }
}

.sheet-btn-text {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
  }

  .tab,
  .badge,
  .load-more,
  .state-action,
  .sheet-btn {
    transition: none;
  }
}
</style>
