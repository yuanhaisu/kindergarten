<script setup>
import { computed, nextTick, ref } from "vue"
import { onLoad, onPageScroll, onPullDownRefresh, onReady } from "@dcloudio/uni-app"
import {
  approveTeacherLeave,
  fetchTeacherLeaveStats,
  fetchTeacherLeaves,
  rejectTeacherLeave
} from "@/api/teacher-leaves"
import { formatDateLabel, formatRelativeDay, formatRelativeStamp } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

const PAGE_SIZE = 20

const FILTERS = [
  { key: "all", label: "全部", status: "" },
  { key: "pending", label: "待审批", status: "pending" },
  { key: "approved", label: "已同意", status: "approved" },
  { key: "rejected", label: "已拒绝", status: "rejected" }
]
const FILTER_LABEL = { all: "全部", pending: "待审批", approved: "已同意", rejected: "已拒绝" }
const STATUS_TEXT = { 0: "待审批", 1: "已同意", 2: "已拒绝" }
const FILTER_EMPTY = {
  pending: {
    title: "暂无待审批申请",
    sub: "全部处理完了，新的提交会出现在这里",
    action: "查看全部"
  },
  approved: {
    title: "还没有已同意的申请",
    sub: "审批通过后会归档到这里",
    action: "查看全部"
  },
  rejected: {
    title: "还没有已拒绝的申请",
    sub: "驳回记录会连同理由保留在这里",
    action: "查看全部"
  }
}
const DEFAULT_EMPTY = {
  title: "暂无请假申请",
  sub: "家长提交后会出现在这里，下拉可刷新",
  action: "刷新列表"
}

const shellRef = ref(null)
const stats = ref({})
const leaves = ref([])
// 最近 20 条「全部」记录：分段计数与统计脚注同源，筛选时也保留它，避免计数跟着筛选塌成 0。
const allRecords = ref(null)
// 「全部」视图的服务端 total：全局空态只看它，不能拿当前筛选的列表或统计反推，
// 否则某个状态为 0 时会被误判成「一条请假都没有」。
const allTotal = ref(null)
const listTotal = ref(0)
const filter = ref("all")
// 当前列表真正对应的筛选。请求失败时回退到它，避免筛选高亮与列表内容不一致。
const appliedFilter = ref("all")
// 切换筛选的请求在途：列表区显示骨架而不是上一筛选的旧卡片。
const listLoading = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const busyId = ref(0)
const expanded = ref({})

const rejectOpen = ref(false)
const rejectTarget = ref(0)
const rejectReason = ref("")
const rejectSubmitting = ref(false)

// 固定吸顶条的测量状态
const filterFixed = ref(false)
// 测量完成前用一个不可能达到的下限：滚动事件先到也不会把固定条顶出来。
const filterTopPx = ref(99999)
let scrollTopPx = 0
let measureTries = 0
let loadSeq = 0

const isInitialLoading = computed(() => !hasLoaded.value && !loadError.value)
const hardError = computed(() => Boolean(loadError.value) && !hasLoaded.value)
// 全局空态只认「全部」视图的服务端 total：待审批为 0 但还有其他状态时是筛选空，不是全局空。
const isAllEmpty = computed(
  () =>
    hasLoaded.value &&
    !hardError.value &&
    !listLoading.value &&
    leaves.value.length === 0 &&
    allTotal.value === 0
)
const viewState = computed(() => {
  if (isInitialLoading.value) return "loading"
  if (hardError.value) return "error"
  if (isAllEmpty.value) return "empty"
  return "normal"
})
// 首次加载与切换筛选时都显示骨架；筛选在途时统计仍按上次成功值展示。
const showListSkeleton = computed(() => isInitialLoading.value || listLoading.value)
const showCounts = computed(() => viewState.value === "normal" && allRecords.value !== null)
const counts = computed(() => {
  const rows = allRecords.value || []
  const result = { all: rows.length, pending: 0, approved: 0, rejected: 0 }
  rows.forEach((item) => {
    const status = Number(item.status)
    if (status === 0) result.pending += 1
    else if (status === 1) result.approved += 1
    else result.rejected += 1
  })
  return result
})
const isFilteredEmpty = computed(
  () => viewState.value === "normal" && !listLoading.value && leaves.value.length === 0
)
const showListHead = computed(
  () => viewState.value === "normal" && !listLoading.value && !isFilteredEmpty.value
)
const listTitle = computed(() =>
  appliedFilter.value === "all" ? "最近 20 条" : FILTER_LABEL[appliedFilter.value] || "请假申请"
)
const listSub = computed(
  () =>
    `共 ${listTotal.value} 条 · ` +
    (appliedFilter.value === "all" ? "待审批优先，再按提交时间倒序" : "按提交时间倒序")
)
const filterEmptyCopy = computed(() => FILTER_EMPTY[appliedFilter.value] || DEFAULT_EMPTY)
const canReject = computed(() => !rejectSubmitting.value && rejectReason.value.trim().length > 0)

function statText(key) {
  if (viewState.value === "loading" || viewState.value === "error") return "—"
  if (viewState.value === "empty") return 0
  return Number(stats.value[key] || 0)
}

function countFor(key) {
  if (viewState.value === "empty") return 0
  if (!showCounts.value) return ""
  return counts.value[key]
}

// 后端只按 created_at 倒序；「全部」视图需要把待审批留在第一屏，所以前端再排一次。
// 单一状态筛选下该比较器等价于 created_at 倒序，不会打乱服务端分页顺序。
function sortForDisplay(rows) {
  return rows.slice().sort((a, b) => {
    const aPending = Number(a.status) === 0 ? 0 : 1
    const bPending = Number(b.status) === 0 ? 0 : 1
    if (aPending !== bPending) return aPending - bPending
    return String(b.created_at || "").localeCompare(String(a.created_at || ""))
  })
}

async function loadLeaves() {
  const seq = ++loadSeq
  const requested = filter.value
  const status = FILTERS.find((item) => item.key === requested)?.status || ""
  // 切到另一个筛选：请求在途时列表区换成骨架，不让上一筛选的卡片被当成当前结果。
  if (requested !== appliedFilter.value) listLoading.value = true
  try {
    const statsPromise = fetchTeacherLeaveStats({ silent: true })
    const listPromise = fetchTeacherLeaves(
      { page: 1, page_size: PAGE_SIZE, ...(status ? { status } : {}) },
      { silent: true }
    )
    const allPromise =
      requested === "all"
        ? null
        : fetchTeacherLeaves({ page: 1, page_size: PAGE_SIZE }, { silent: true }).catch(() => null)
    const [statsRes, listRes, allRes] = await Promise.all([statsPromise, listPromise, allPromise])
    if (seq !== loadSeq) return

    stats.value = statsRes.data || {}
    const rows = Array.isArray(listRes.data?.list) ? listRes.data.list : []
    leaves.value = sortForDisplay(rows)
    const total = Number(listRes.data?.total)
    const viewTotal = Number.isFinite(total) && total >= rows.length ? total : rows.length
    listTotal.value = viewTotal
    if (requested === "all") {
      allRecords.value = rows
      allTotal.value = viewTotal
    } else if (Array.isArray(allRes?.data?.list)) {
      allRecords.value = allRes.data.list
      const allCount = Number(allRes.data?.total)
      allTotal.value =
        Number.isFinite(allCount) && allCount >= allRes.data.list.length
          ? allCount
          : allRes.data.list.length
    }
    appliedFilter.value = requested
    loadError.value = ""
    hasLoaded.value = true
  } catch (e) {
    if (seq !== loadSeq) return
    if (!hasLoaded.value) {
      // 首次加载失败：整页错误态，不能让空列表冒充「没有请假」。
      leaves.value = []
      allRecords.value = null
      loadError.value = "请假列表加载失败"
    } else {
      // 已有内容时保留旧列表，并把筛选回退到与列表一致的状态。
      filter.value = appliedFilter.value
      uni.showToast({ title: "加载失败，请重试", icon: "none" })
    }
  } finally {
    if (seq === loadSeq) {
      listLoading.value = false
      uni.stopPullDownRefresh()
    }
  }
  // 布局稳定后再量筛选条位置；固定条靠它判断何时接棒。
  await nextTick()
  measureFilter()
}

async function handleRetry() {
  await loadLeaves()
  if (!loadError.value) uni.showToast({ title: "已重新加载", icon: "none" })
}

async function handleRefresh() {
  await loadLeaves()
  if (!loadError.value) uni.showToast({ title: "已刷新", icon: "none" })
}

function selectFilter(key) {
  // 分段始终可点：空态/错误态下切换筛选本身就是一次按新状态的重试。
  if (key === filter.value) return
  filter.value = key
  loadLeaves()
}

function resetToAll() {
  if (filter.value === "all") {
    handleRefresh()
    return
  }
  filter.value = "all"
  loadLeaves()
}

function studentName(item) {
  return item?.student?.name || "学生"
}

function applicantName(item) {
  return item?.applicant?.nickname || "家长"
}

function initialOf(item) {
  const name = String(item?.student?.name || "").trim()
  return name ? Array.from(name)[0] : "家"
}

function statusOf(item) {
  return Number(item.status)
}

function statusText(item) {
  return STATUS_TEXT[statusOf(item)] || "已审批"
}

function statusClass(item) {
  const status = statusOf(item)
  return status === 0 ? "is-pending" : status === 1 ? "is-approved" : "is-rejected"
}

function cardTime(item) {
  if (statusOf(item) === 0) return formatRelativeStamp(item.created_at) || "—"
  const day = formatRelativeDay(item.reviewed_at)
  return day ? `审批 ${day}` : "已审批"
}

function leaveDays(item) {
  const days = Number(item.days)
  if (Number.isFinite(days) && days > 0) return days
  // 兜底：接口缺 days 时按起止日算含首尾天数。
  const start = String(item.start_date || "")
  const end = String(item.end_date || start)
  if (!start || !end) return 0
  const s = new Date(`${start}T00:00:00Z`)
  const e = new Date(`${end}T00:00:00Z`)
  if (Number.isNaN(s.getTime()) || Number.isNaN(e.getTime())) return 0
  return Math.max(1, Math.round((e.getTime() - s.getTime()) / 86400000) + 1)
}

function rangeText(item) {
  const start = formatDateLabel(item.start_date)
  const end = formatDateLabel(item.end_date)
  const days = leaveDays(item)
  const suffix = days ? ` · 共 ${days} 天` : ""
  if (!start && !end) return days ? `共 ${days} 天` : "—"
  const sameDay = !end || item.start_date === item.end_date
  const range = sameDay ? start || end : `${start || "—"}–${end || "—"}`
  return range + suffix
}

function reviewStamp(item) {
  return formatRelativeStamp(item.reviewed_at) || "—"
}

function toggleTrace(item) {
  if (statusOf(item) === 0) return
  const id = item.id
  expanded.value = { ...expanded.value, [id]: !expanded.value[id] }
}

function noop() {}

function statusKeyOf(status) {
  return status === 0 ? "pending" : status === 1 ? "approved" : "rejected"
}

// 审批成功后先本地同步状态：重拉完成前卡片也必须从「待审批」翻成已审批，
// 否则会出现一张仍显示「同意/驳回」的旧卡。
function applyLocalReview(id, status, remark) {
  const now = new Date().toISOString()
  const patch = (item) =>
    Number(item.id) === Number(id)
      ? { ...item, status, reviewed_at: now, review_remark: remark }
      : item
  const scopeStatus = FILTERS.find((item) => item.key === appliedFilter.value)?.status || ""
  if (scopeStatus && scopeStatus !== statusKeyOf(status)) {
    // 当前筛选已不包含这条（例如在「待审批」里同意）：先移除，不能让它挂在错误分段下。
    leaves.value = leaves.value.filter((item) => Number(item.id) !== Number(id))
  } else {
    leaves.value = sortForDisplay(leaves.value.map(patch))
  }
  if (allRecords.value) allRecords.value = allRecords.value.map(patch)
}

async function approve(item) {
  if (busyId.value || statusOf(item) !== 0) return
  busyId.value = item.id
  try {
    await approveTeacherLeave(item.id, "", { silent: true })
    applyLocalReview(item.id, 1, "")
    busyId.value = 0
    uni.showToast({ title: "已同意", icon: "success" })
    await loadLeaves()
  } catch (e) {
    busyId.value = 0
    uni.showToast({ title: "审批失败，请重试", icon: "none" })
  }
}

function openReject(item) {
  if (busyId.value || statusOf(item) !== 0) return
  rejectTarget.value = item.id
  rejectReason.value = ""
  rejectOpen.value = true
}

function closeReject() {
  if (rejectSubmitting.value) return
  rejectOpen.value = false
  rejectTarget.value = 0
}

async function submitReject() {
  if (!canReject.value) return
  const id = rejectTarget.value
  const reason = rejectReason.value.trim()
  rejectSubmitting.value = true
  busyId.value = id
  try {
    await rejectTeacherLeave(id, reason, { silent: true })
    applyLocalReview(id, 2, reason)
    rejectSubmitting.value = false
    busyId.value = 0
    rejectOpen.value = false
    rejectTarget.value = 0
    uni.showToast({ title: "已拒绝", icon: "success" })
    await loadLeaves()
  } catch (e) {
    // 失败保持弹窗：理由不丢，老师改完可以直接重试。
    rejectSubmitting.value = false
    busyId.value = 0
    uni.showToast({ title: "驳回失败，请重试", icon: "none" })
  }
}

// ---------- 吸顶分段 ----------
// 项目统一不依赖 sticky 吸顶（见 _shared/CSS-COMPAT.md）：行内分段滚出视口后，
// 由页面顶部的 fixed 副本接棒；行内条保留占位，避免内容跳动。

function updateFixedBar() {
  if (viewState.value !== "normal") {
    filterFixed.value = false
    return
  }
  filterFixed.value = scrollTopPx >= filterTopPx.value
}

function measureFilter() {
  setTimeout(() => {
    const scope = shellRef.value
    const finish = (rect, scrollTop) => {
      if (!rect && measureTries < 2) {
        measureTries += 1
        measureFilter()
        return
      }
      measureTries = 0
      if (!rect) return
      // 用实测滚动量而不是旧的 scrollTopPx：列表变短后滚动会被夹紧，
      // 旧值会让固定条的阈值偏高，表现为「滚回顶部了还挂着」。
      const offset = Number(scrollTop)
      if (Number.isFinite(offset)) scrollTopPx = offset
      filterTopPx.value = rect.top + scrollTopPx
      filterFixed.value = viewState.value === "normal" && rect.top <= 0
    }
    const readScroll = (rect) => {
      uni
        .createSelectorQuery()
        .selectViewport()
        .scrollOffset((viewport) => finish(rect, viewport ? viewport.scrollTop : scrollTopPx))
        .exec()
    }
    const runRectQuery = (query, allowFallback) => {
      query
        .select(".filter-sticky")
        .boundingClientRect((rect) => {
          if (!rect && allowFallback && scope) {
            // 兜底：部分基础库把 slot 节点归到页面树，组件内查不到时回退页面级。
            runRectQuery(uni.createSelectorQuery(), false)
            return
          }
          readScroll(rect)
        })
        .exec()
    }
    let query = null
    try {
      query = scope ? uni.createSelectorQuery().in(scope) : uni.createSelectorQuery()
    } catch (err) {
      console.warn("[leave-index] measure filter failed:", err?.message || err)
      query = uni.createSelectorQuery()
    }
    runRectQuery(query, Boolean(scope))
  }, 60)
}

onLoad(() => {
  loadLeaves()
})
onReady(() => {
  measureFilter()
})
onPageScroll((event) => {
  scrollTopPx = Number(event?.scrollTop) || 0
  updateFixedBar()
})
onPullDownRefresh(() => {
  loadLeaves()
})
</script>

<template>
  <PageShell ref="shellRef" bottom-space="48rpx" max-width="430px">
    <!-- 固定分段条：行内条滚出视口后接棒（项目不支持 sticky，见 CSS-COMPAT） -->
    <view v-if="filterFixed" class="filter-fixed">
      <view class="filter-bar">
        <view
          v-for="option in FILTERS"
          :key="option.key"
          class="filter-btn"
          :class="{ 'is-active': filter === option.key }"
          @click="selectFilter(option.key)"
        >
          <text class="filter-label">{{ option.label }}</text>
          <text v-if="countFor(option.key) !== ''" class="filter-count">{{ countFor(option.key) }}</text>
        </view>
      </view>
    </view>

    <view class="leave-page" :class="'state-' + viewState">
      <!-- 统计：待审批/已同意/已拒绝为班级累计，本月总计按月提交 -->
      <view class="stats-card">
        <view class="stats-grid">
          <view class="stat">
            <text class="stat-num is-pending">{{ statText("pending") }}</text>
            <text class="stat-label">待审批</text>
          </view>
          <view class="stat">
            <text class="stat-num is-approved">{{ statText("approved") }}</text>
            <text class="stat-label">已同意</text>
          </view>
          <view class="stat">
            <text class="stat-num is-rejected">{{ statText("rejected") }}</text>
            <text class="stat-label">已拒绝</text>
          </view>
          <view class="stat">
            <text class="stat-num is-total">{{ statText("monthly_total") }}</text>
            <text class="stat-label">本月总计</text>
          </view>
        </view>
        <text class="stats-foot">待审批/已同意/已拒绝为班级累计 · 分段与列表取最近 20 条</text>
      </view>

      <!-- 行内分段（保留占位；滚出视口后由 fixed 副本接棒）。分段始终可点：
           空态/错误态下切换筛选本身就是一次按新状态的重试。 -->
      <view class="filter-sticky">
        <view class="filter-bar">
          <view
            v-for="option in FILTERS"
            :key="option.key"
            class="filter-btn"
            :class="{ 'is-active': filter === option.key }"
            @click="selectFilter(option.key)"
          >
            <text class="filter-label">{{ option.label }}</text>
            <text v-if="countFor(option.key) !== ''" class="filter-count">{{ countFor(option.key) }}</text>
          </view>
        </view>
      </view>

      <!-- 列表头：全部视图说明排序规则，筛选视图只留时间口径 -->
      <view v-if="showListHead" class="list-head">
        <text class="list-title">{{ listTitle }}</text>
        <text class="list-sub">{{ listSub }}</text>
      </view>

      <!-- 错误：整页错误态，重试可恢复；分段仍可点，可换状态重试 -->
      <view v-if="viewState === 'error'" class="state-panel">
        <view class="state-icon is-danger"><text class="icon">&#xeca1;</text></view>
        <text class="state-title">请假列表加载失败</text>
        <text class="state-desc">网络异常，请检查网络后重试</text>
        <view class="state-btn" @click="handleRetry">重试</view>
      </view>

      <!-- 全局空：仅「全部」也无数据时出现，统计归零、隐藏列表头 -->
      <view v-else-if="viewState === 'empty'" class="state-panel">
        <view class="state-icon"><text class="icon">&#xeb81;</text></view>
        <text class="state-title">暂无请假申请</text>
        <text class="state-desc">家长提交后会出现在这里，下拉可刷新</text>
        <view class="state-btn" @click="handleRefresh">刷新列表</view>
      </view>

      <!-- 切换筛选的请求在途：骨架占位，不展示上一筛选的旧卡片 -->
      <view v-else-if="showListSkeleton" class="skeleton-list">
        <view v-for="i in 3" :key="i" class="skeleton-card">
          <view class="skel-head">
            <view class="sk sk-avatar" />
            <view class="skel-lines">
              <view class="sk sk-line sk-w60" />
              <view class="sk sk-line sk-w40" />
            </view>
          </view>
          <view class="sk sk-line" />
          <view class="sk sk-line sk-w80" />
          <view class="skel-actions">
            <view class="sk sk-btn" />
            <view class="sk sk-btn" />
          </view>
        </view>
      </view>

      <!-- 筛选空：该状态下没有申请，但仍可回到全部 -->
      <view v-else-if="isFilteredEmpty" class="state-panel">
        <view class="state-icon"><text class="icon">&#xeb81;</text></view>
        <text class="state-title">{{ filterEmptyCopy.title }}</text>
        <text class="state-desc">{{ filterEmptyCopy.sub }}</text>
        <view class="state-btn" @click="resetToAll">{{ filterEmptyCopy.action }}</view>
      </view>

      <!-- 正常列表 -->
      <view v-else class="leave-list">
        <view
          v-for="item in leaves"
          :key="item.id"
          class="leave-card"
          :class="{ 'is-history': statusOf(item) !== 0, 'is-expanded': expanded[item.id] }"
          @click="toggleTrace(item)"
        >
          <view class="card-head">
            <view class="avatar">{{ initialOf(item) }}</view>
            <view class="who">
              <view class="name-row">
                <text class="student-name">{{ studentName(item) }}</text>
                <text class="status-pill" :class="statusClass(item)">{{ statusText(item) }}</text>
              </view>
              <text class="applicant">{{ applicantName(item) }} 提交</text>
            </view>
            <text class="card-time">{{ cardTime(item) }}</text>
          </view>

          <view class="facts">
            <view class="fact">
              <text class="fact-label">请假时间</text>
              <text class="fact-value">{{ rangeText(item) }}</text>
            </view>
            <view class="fact">
              <text class="fact-label">事由</text>
              <text class="fact-value">{{ item.reason || "—" }}</text>
            </view>
            <view v-if="statusOf(item) === 2" class="fact is-reject">
              <text class="fact-label">驳回理由</text>
              <text class="fact-value">{{ item.review_remark || "未填写理由" }}</text>
            </view>
          </view>

          <!-- 待审批：驳回 / 同意；处理中两键同时禁用 -->
          <view v-if="statusOf(item) === 0" class="card-actions">
            <view
              class="btn-reject"
              :class="{ 'is-busy': busyId === item.id }"
              @click.stop="openReject(item)"
            >驳回</view>
            <view
              class="btn-approve"
              :class="{ 'is-busy': busyId === item.id }"
              @click.stop="approve(item)"
            >
              <view v-if="busyId === item.id" class="btn-spinner" />
              <text>{{ busyId === item.id ? "处理中…" : "同意" }}</text>
            </view>
          </view>

          <!-- 已审批：展开审批留痕（箭头旋转） -->
          <view v-else class="trace-hint">
            <text>审批留痕</text>
            <view class="trace-chev"><text class="icon">&#xea6e;</text></view>
          </view>
          <view v-if="statusOf(item) !== 0 && expanded[item.id]" class="trace">
            <view class="trace-row">
              <text class="trace-label">审批时间</text>
              <text class="trace-value">{{ reviewStamp(item) }}</text>
            </view>
            <view class="trace-row">
              <text class="trace-label">审批意见</text>
              <text class="trace-value">{{ item.review_remark || "未填写审批意见" }}</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <!-- 拒绝原因：微信语汇的居中弹窗，理由必填、0/100 -->
    <view v-if="rejectOpen" class="dialog-layer">
      <view class="dialog-mask" @click="closeReject" @touchmove.stop.prevent="noop" />
      <view class="dialog">
        <text class="dialog-title">拒绝原因</text>
        <text class="dialog-sub">请填写拒绝原因，家长会收到这条说明</text>
        <textarea
          v-model="rejectReason"
          class="dialog-input"
          maxlength="100"
          :disabled="rejectSubmitting"
          placeholder="请输入拒绝原因"
        />
        <text class="dialog-count">{{ rejectReason.length }}/100</text>
        <view class="dialog-actions">
          <view class="dialog-btn dialog-cancel" @click="closeReject">取消</view>
          <view
            class="dialog-btn dialog-confirm"
            :class="{ 'is-disabled': !canReject }"
            @click="submitReject"
          >{{ rejectSubmitting ? "提交中…" : "确定驳回" }}</view>
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

.leave-page {
  padding-bottom: $spacing-xl;
}

/* ---------- 统计卡 ---------- */
.stats-card {
  padding: 28rpx $spacing-lg 20rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.stats-grid {
  display: flex;
}

.stat {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
}

.stat-num {
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-bold;
  line-height: 1.2;
}

.stat-num.is-pending {
  color: $color-brand-primary-dark;
}

.stat-num.is-approved {
  color: $color-semantic-success;
}

.stat-num.is-rejected {
  color: $color-text-muted;
}

.stat-num.is-total {
  color: $color-text-primary;
}

.stat-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.stats-foot {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  text-align: center;
}

/* ---------- 状态分段 ---------- */
.filter-sticky {
  margin: 20rpx (-$component-page-padding-x) 0;
  padding: $spacing-sm $component-page-padding-x;
  background: $color-bg-page;
}

.filter-fixed {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 30;
  width: 100%;
  max-width: 430px;
  margin: 0 auto;
  padding: $spacing-sm $component-page-padding-x;
  background: $color-bg-page;
}

.filter-fixed .filter-bar {
  box-shadow: $shadow-md;
}

.filter-bar {
  display: flex;
  gap: $spacing-xs;
  padding: $spacing-xs;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.filter-btn {
  flex: 1;
  min-width: 0;
  min-height: 88rpx;
  padding: 0 6rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6rpx;
  border-radius: $border-radius-full;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.filter-btn.is-active {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.filter-btn:active {
  transform: scale(0.97);
}

.filter-count {
  color: inherit;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  opacity: 0.8;
}

/* ---------- 列表头 ---------- */
.list-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin: 28rpx 4rpx $spacing-md;
}

.list-title {
  flex: none;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.list-sub {
  flex: 1;
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  text-align: right;
  overflow-wrap: anywhere;
}

/* ---------- 请假卡 ---------- */
.leave-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.leave-card {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  padding: 28rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.leave-card.is-history {
  box-shadow: $shadow-sm;
}

.leave-card.is-history:active {
  transform: scale(0.995);
}

.card-head {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  min-width: 0;
}

.avatar {
  flex: none;
  width: 80rpx;
  height: 80rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $gradient-brand-soft;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.is-history .avatar {
  background: $color-bg-page;
  color: $color-text-muted;
}

.who {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-sm;
  min-width: 0;
}

.student-name {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  overflow-wrap: anywhere;
}

.status-pill {
  flex: none;
  min-height: 36rpx;
  padding: 2rpx $spacing-md;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.3;
}

.status-pill.is-pending {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
}

.status-pill.is-approved {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.status-pill.is-rejected {
  background: $color-bg-page;
  color: $color-text-muted;
}

.applicant {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.card-time {
  flex: none;
  max-width: 160rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  text-align: right;
  overflow-wrap: anywhere;
}

.facts {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  min-width: 0;
}

.fact {
  display: flex;
  align-items: baseline;
  gap: $spacing-md;
  min-width: 0;
}

.fact-label {
  flex: none;
  width: 120rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

.fact-value {
  flex: 1;
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.fact.is-reject .fact-label,
.fact.is-reject .fact-value {
  color: $color-semantic-danger;
}

/* ---------- 审批动作 ---------- */
.card-actions {
  display: flex;
  gap: $spacing-md;
  padding-top: 20rpx;
  border-top: 1rpx solid $color-border-subtle;
}

.btn-reject,
.btn-approve {
  flex: 1;
  min-width: 0;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
}

.btn-reject {
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-card;
  color: $color-semantic-danger;
}

.btn-approve {
  background: $gradient-brand;
  color: $color-text-primary;
  box-shadow: $shadow-sm;
}

.btn-reject:active,
.btn-approve:active {
  transform: scale(0.98);
}

.btn-reject.is-busy,
.btn-approve.is-busy {
  opacity: 0.6;
}

.btn-spinner {
  flex: none;
  width: 26rpx;
  height: 26rpx;
  border: 4rpx solid $color-text-primary;
  border-top-color: transparent;
  border-radius: $border-radius-full;
  animation: leave-spin 0.7s linear infinite;
}

/* ---------- 审批留痕 ---------- */
.trace-hint {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  min-height: 72rpx;
  padding-top: $spacing-md;
  border-top: 1rpx solid $color-border-subtle;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
}

/* 箭头必须是 view：text 节点上的 transform 在小程序里不生效 */
.trace-chev {
  flex: none;
  width: 28rpx;
  height: 28rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  transition: transform $animation-duration-base $animation-easing-default;
}

.leave-card.is-expanded .trace-chev {
  transform: rotate(90deg);
}

.trace {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  padding: 20rpx $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.trace-row {
  display: flex;
  align-items: baseline;
  gap: $spacing-md;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
}

.trace-label {
  flex: none;
  width: 120rpx;
  color: $color-text-muted;
}

.trace-value {
  flex: 1;
  min-width: 0;
  color: $color-text-body;
  overflow-wrap: anywhere;
}

/* ---------- 骨架 ---------- */
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.skeleton-card {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  padding: 28rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.skel-head {
  display: flex;
  align-items: center;
  gap: 20rpx;
}

.skel-lines {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
}

.skel-actions {
  display: flex;
  gap: $spacing-md;
}

.sk {
  display: block;
  height: 20rpx;
  border-radius: $border-radius-full;
  background-image: linear-gradient(
    90deg,
    $color-bg-page 0%,
    $color-brand-primary-bg 50%,
    $color-bg-page 100%
  );
  background-size: 200% 100%;
  animation: leave-shimmer 1.6s linear infinite;
}

.sk-avatar {
  flex: none;
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-full;
}

.sk-w40 {
  width: 40%;
}

.sk-w60 {
  width: 60%;
}

.sk-w80 {
  width: 80%;
}

.sk-btn {
  flex: 1;
  height: 88rpx;
  border-radius: $border-radius-full;
}

/* ---------- 空 / 错误 ---------- */
.state-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 72rpx 36rpx 56rpx;
  text-align: center;
}

.state-icon {
  width: 112rpx;
  height: 112rpx;
  margin-bottom: $spacing-md;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-muted;
  box-shadow: $shadow-sm;
  font-size: $typography-font-size-3xl;
}

.state-icon.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  box-shadow: none;
}

.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.state-desc {
  max-width: 520rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}

.state-btn {
  margin-top: $spacing-xl;
  height: 88rpx;
  padding: 0 44rpx;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.state-btn:active {
  transform: scale(0.98);
}

/* ---------- 拒绝原因弹窗 ---------- */
.dialog-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48rpx;
}

.dialog-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba($color-text-primary, 0.5);
}

.dialog {
  position: relative;
  width: 100%;
  max-width: 560rpx;
  display: flex;
  flex-direction: column;
  padding: 40rpx 36rpx 0;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
  overflow: hidden;
}

.dialog-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  text-align: center;
}

.dialog-sub {
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  text-align: center;
}

.dialog-input {
  box-sizing: border-box;
  width: 100%;
  min-height: 168rpx;
  margin-top: $spacing-lg;
  padding: 20rpx $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}

.dialog-count {
  align-self: flex-end;
  margin-top: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.dialog-actions {
  display: flex;
  margin: $spacing-lg (-36rpx) 0;
  border-top: 1rpx solid $color-border-subtle;
}

.dialog-btn {
  flex: 1;
  height: 96rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: $typography-font-size-xl;
  line-height: 1.2;
}

.dialog-cancel {
  border-right: 1rpx solid $color-border-subtle;
  color: $color-text-body;
}

.dialog-confirm {
  color: $color-semantic-danger;
  font-weight: $typography-font-weight-semibold;
}

.dialog-confirm.is-disabled {
  color: $color-text-hint;
}

@keyframes leave-spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes leave-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
