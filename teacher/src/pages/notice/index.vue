<script setup>
import { computed, ref } from "vue"
import { onLoad, onShow, onPullDownRefresh, onReachBottom } from "@dcloudio/uni-app"
import StateBlock from "@/components/StateBlock.vue"
import {
  fetchTeacherNotices,
  fetchTeacherNotice,
  fetchTeacherNoticeReadStatus,
  remindTeacherNotice
} from "@/api/teacher-notices"

const statusTabs = [
  { key: "all", label: "全部" },
  { key: "2", label: "已发布" },
  { key: "1", label: "定时" },
  { key: "0", label: "草稿" },
  { key: "3", label: "已撤回" }
]

const categories = [
  { key: "all", label: "全部" },
  { key: "homework", label: "作业" },
  { key: "bring", label: "带物" },
  { key: "activity", label: "活动" },
  { key: "health", label: "健康" },
  { key: "other", label: "其他" }
]

const CATEGORY_LABELS = {
  homework: "作业",
  bring: "带物",
  activity: "活动",
  health: "健康",
  other: "其他"
}

const STATUS_META = {
  0: { label: "草稿", tone: "draft" },
  1: { label: "定时待发", tone: "scheduled" },
  2: { label: "已发布", tone: "published" },
  3: { label: "已撤回", tone: "revoked" }
}

const RELATION_LABELS = {
  mother: "妈妈",
  father: "爸爸",
  grandfather: "爷爷",
  grandmother: "奶奶",
  guardian: "监护人",
  other: "其他"
}

const activeStatus = ref("all")
const activeCategory = ref("all")
const notices = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const loadError = ref("")

const hasMore = computed(() => notices.value.length < total.value)
const avgReadRate = computed(() => {
  const published = notices.value.filter((item) => {
    const status = Number(item.status)
    return (status === 2 || status === 3) && Number(item.total_count) > 0
  })
  if (!published.length) return null
  const sum = published.reduce((acc, item) => acc + readRate(item), 0)
  return Math.round(sum / published.length)
})

function pad(value) {
  return String(value).padStart(2, "0")
}

function formatTime(value) {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value).slice(0, 16).replace("T", " ")
  return `${date.getMonth() + 1}月${date.getDate()}日 ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function statusMeta(item) {
  return STATUS_META[Number(item?.status)] || { label: item?.status_label || "未知", tone: "draft" }
}

function statusTone(item) {
  return statusMeta(item).tone
}

function categoryLabel(item) {
  return item?.category_label || CATEGORY_LABELS[item?.category] || "通知"
}

function classNameList(item) {
  return Array.isArray(item?.class_names) ? item.class_names : []
}

function readRate(item) {
  const raw = Number(String(item?.read_rate || "").replace("%", ""))
  if (!Number.isNaN(raw) && item?.read_rate) return Math.max(0, Math.min(100, raw))
  const totalCount = Number(item?.total_count) || 0
  if (!totalCount) return 0
  return Math.max(0, Math.min(100, Math.round((Number(item?.read_count) || 0) / totalCount * 100)))
}

function showRead(item) {
  const status = Number(item?.status)
  return (status === 2 || status === 3) && Number(item?.total_count) > 0
}

function statusNote(item) {
  const status = Number(item?.status)
  if (status === 0) return "仅自己可见 · 发布后开始统计已读"
  if (status === 1) return "定时发送 · 到点后开始统计已读"
  if (status === 3) return "家长已不可见"
  return ""
}

async function loadNotices(reset = true) {
  if (loading.value && !reset) return
  loading.value = true
  if (reset) {
    page.value = 1
    loadError.value = ""
  }

  try {
    const params = { page: page.value, page_size: pageSize }
    if (activeStatus.value !== "all") params.status = Number(activeStatus.value)
    if (activeCategory.value !== "all") params.category = activeCategory.value

    const res = await fetchTeacherNotices(params)
    const data = res.data || res || {}
    const list = Array.isArray(data.list) ? data.list : []

    notices.value = reset ? list : notices.value.concat(list)
    total.value = Number(data.total ?? notices.value.length)
  } catch (err) {
    if (reset) {
      notices.value = []
      total.value = 0
      loadError.value = err?.message || "通知加载失败"
    } else {
      page.value = Math.max(1, page.value - 1)
      uni.showToast({ title: err?.message || "加载更多失败", icon: "none" })
    }
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

function changeStatus(key) {
  if (activeStatus.value === key) return
  activeStatus.value = key
  loadNotices(true)
}

function changeCategory(key) {
  if (activeCategory.value === key) return
  activeCategory.value = key
  loadNotices(true)
}

function loadMore() {
  if (loading.value || !hasMore.value) return
  page.value += 1
  loadNotices(false)
}

function goCreate() {
  uni.navigateTo({ url: "/pages/notice/create" })
}

// ---- 详情弹层 ----

const detailVisible = ref(false)
const detailLoading = ref(false)
const detailError = ref("")
const detail = ref(null)

const detailStats = computed(() => {
  const source = detail.value || {}
  const stats = source.read_stats || {}
  return {
    read: Number(stats.read_count ?? source.read_count ?? 0),
    unread: Number(stats.unread_count ?? source.unread_count ?? 0),
    confirmed: Number(stats.confirmed_count ?? source.confirmed_count ?? 0),
    total: Number(stats.total_count ?? source.total_count ?? 0),
    rate: stats.read_rate || source.read_rate || ""
  }
})

const detailRate = computed(() => {
  if (detailStats.value.rate) {
    const raw = Number(String(detailStats.value.rate).replace("%", ""))
    if (!Number.isNaN(raw)) return Math.max(0, Math.min(100, raw))
  }
  const totalCount = detailStats.value.total
  return totalCount ? Math.round(detailStats.value.read / totalCount * 100) : 0
})

const detailStatus = computed(() => statusMeta(detail.value))
const detailClasses = computed(() => classNameList(detail.value).join("、") || "未指定班级")
const detailTime = computed(() => formatTime(detail.value?.published_at || detail.value?.created_at))
const detailAttachments = computed(() => Array.isArray(detail.value?.attachments) ? detail.value.attachments : [])
const detailNote = computed(() => {
  const status = Number(detail.value?.status)
  if (status === 0) return "草稿仅自己可见，发布后开始统计已读。"
  if (status === 1) return `定时发送：${formatTime(detail.value?.scheduled_at)} 自动发送，发送后开始统计已读。`
  if (status === 3) return "已撤回：家长端不再展示，以上为撤回前数据。"
  return ""
})
const showDetailStats = computed(() => {
  const status = Number(detail.value?.status)
  return (status === 2 || status === 3) && detailStats.value.total > 0
})
const canReadStatus = computed(() => Number(detail.value?.status) === 2)
const canRemind = computed(() => Number(detail.value?.status) === 2)
const isDraft = computed(() => Number(detail.value?.status) === 0)
const isScheduled = computed(() => Number(detail.value?.status) === 1)

const unreadVisible = ref(false)
const unreadLoading = ref(false)
const unreadError = ref("")
const unreadList = ref([])
const unreadTotal = ref(0)
const reminding = ref(false)

async function openDetail(item) {
  if (!item?.id) return
  detailVisible.value = true
  detailLoading.value = true
  detailError.value = ""
  detail.value = item
  unreadVisible.value = false
  unreadList.value = []
  unreadError.value = ""

  try {
    const res = await fetchTeacherNotice(item.id)
    detail.value = res.data || res || item
  } catch (err) {
    detailError.value = err?.message || "通知详情加载失败"
  } finally {
    detailLoading.value = false
  }
}

function closeDetail() {
  detailVisible.value = false
  detailLoading.value = false
  detailError.value = ""
  detail.value = null
  unreadVisible.value = false
  unreadList.value = []
  unreadError.value = ""
}

function editNotice() {
  const id = detail.value?.id
  if (!id) return
  closeDetail()
  uni.navigateTo({ url: `/pages/notice/create?id=${id}` })
}

async function reloadDetail() {
  if (!detail.value?.id) return
  const id = detail.value.id
  detailLoading.value = true
  detailError.value = ""
  try {
    const res = await fetchTeacherNotice(id)
    detail.value = res.data || res
  } catch (err) {
    detailError.value = err?.message || "通知详情加载失败"
  } finally {
    detailLoading.value = false
  }
}

async function toggleUnread() {
  if (!detail.value?.id) return
  if (unreadVisible.value) {
    unreadVisible.value = false
    return
  }
  unreadVisible.value = true
  if (unreadList.value.length || unreadLoading.value) return

  unreadLoading.value = true
  unreadError.value = ""
  try {
    const res = await fetchTeacherNoticeReadStatus(detail.value.id, {
      status: "unread",
      page: 1,
      page_size: 50
    })
    const data = res.data || res || {}
    unreadList.value = Array.isArray(data.list) ? data.list : []
    unreadTotal.value = Number(data.unread_count ?? data.total ?? unreadList.value.length)
  } catch (err) {
    unreadError.value = err?.message || "未读名单加载失败"
  } finally {
    unreadLoading.value = false
  }
}

async function remind() {
  if (!detail.value?.id || reminding.value) return
  reminding.value = true
  try {
    const res = await remindTeacherNotice(detail.value.id)
    const data = res.data || res || {}
    uni.showToast({ title: data.message || "已提醒未读家长", icon: "none" })
  } catch (err) {
    uni.showToast({ title: err?.message || "提醒失败", icon: "none" })
  } finally {
    reminding.value = false
  }
}

function relationLabel(value) {
  return RELATION_LABELS[value] || value || ""
}

let firstShow = true

onLoad(() => loadNotices(true))
onShow(() => {
  // 从「发布通知」返回时刷新列表；首次 onShow 与 onLoad 同时触发，跳过避免重复请求。
  if (firstShow) {
    firstShow = false
    return
  }
  loadNotices(true)
})
onPullDownRefresh(() => loadNotices(true))
onReachBottom(loadMore)
</script>

<template>
  <view class="page">
    <!-- 概要 -->
    <view class="hero">
      <view class="hero-top">
        <view class="hero-copy">
          <text class="hero-title">通知管理</text>
          <text class="hero-sub">发布通知，跟进家长已读与确认</text>
        </view>
        <view class="publish-btn" @click="goCreate">
          <text>发布通知</text>
        </view>
      </view>
      <view class="hero-stats">
        <view class="hero-stat">
          <text class="hero-num">{{ total }}</text>
          <text class="hero-label">通知数</text>
        </view>
        <view class="hero-stat">
          <text class="hero-num">{{ notices.length }}</text>
          <text class="hero-label">已加载</text>
        </view>
        <view class="hero-stat">
          <text class="hero-num">{{ avgReadRate === null ? "—" : avgReadRate + "%" }}</text>
          <text class="hero-label">平均已读</text>
        </view>
      </view>
    </view>

    <!-- 筛选 -->
    <view class="filters">
      <view class="status-row">
        <view
          v-for="tab in statusTabs"
          :key="tab.key"
          class="filter-pill"
          :class="{ active: activeStatus === tab.key }"
          @click="changeStatus(tab.key)"
        >
          <text>{{ tab.label }}</text>
        </view>
      </view>
      <view class="category-row">
        <view
          v-for="item in categories"
          :key="item.key"
          class="category-pill"
          :class="{ active: activeCategory === item.key }"
          @click="changeCategory(item.key)"
        >
          <text>{{ item.label }}</text>
        </view>
      </view>
    </view>

    <view class="list-head">
      <text class="list-count">共 {{ total }} 条</text>
      <text v-if="loading && notices.length" class="list-loading">加载中...</text>
    </view>

    <!-- 首屏状态 -->
    <StateBlock
      v-if="loading && notices.length === 0"
      loading
      loading-text="通知加载中..."
    />
    <StateBlock
      v-else-if="loadError && notices.length === 0"
      :error="loadError"
      @retry="loadNotices(true)"
    />
    <StateBlock
      v-else-if="!loading && notices.length === 0"
      empty-text="暂无通知"
    />

    <!-- 列表 -->
    <view v-else class="notice-list">
      <view
        v-for="item in notices"
        :key="item.id"
        class="notice-card"
        @click="openDetail(item)"
      >
        <view class="card-top">
          <view class="category-tag" :class="'cat-' + (item.category || 'other')">
            <text>{{ categoryLabel(item) }}</text>
          </view>
          <view class="status-tag" :class="'status-' + statusTone(item)">
            <text>{{ statusMeta(item).label }}</text>
          </view>
          <text class="card-time">{{ formatTime(item.published_at || item.created_at) }}</text>
        </view>

        <text class="card-title">{{ item.title || "未命名通知" }}</text>
        <text class="card-preview">{{ item.content || "" }}</text>

        <view class="card-meta">
          <text
            v-for="name in classNameList(item).slice(0, 2)"
            :key="name"
            class="meta-chip"
          >{{ name }}</text>
          <text v-if="classNameList(item).length > 2" class="meta-chip">
            +{{ classNameList(item).length - 2 }}
          </text>
          <text v-if="item.attachments_count" class="meta-chip">
            {{ item.attachments_count }} 个附件
          </text>
        </view>

        <view v-if="showRead(item)" class="read-row">
          <view class="read-bar">
            <view class="read-fill" :style="{ width: readRate(item) + '%' }" />
          </view>
          <text class="read-text">
            已读 {{ item.read_count || 0 }}/{{ item.total_count || 0 }} · {{ readRate(item) }}%
          </text>
        </view>
        <text v-else class="draft-note">{{ statusNote(item) }}</text>

        <view class="card-foot">
          <text v-if="item.need_confirm" class="confirm-chip">需家长确认</text>
          <text v-else class="confirm-chip is-muted">无需确认</text>
          <text class="open-link">查看详情 ›</text>
        </view>
      </view>

      <view v-if="loading && notices.length" class="load-more">
        <text>加载中...</text>
      </view>
      <view v-else-if="!hasMore && notices.length" class="load-more">
        <text>没有更多了</text>
      </view>
    </view>

    <!-- 详情 -->
    <view v-if="detailVisible" class="sheet-mask" @click="closeDetail">
      <view class="sheet" @click.stop>
        <view class="sheet-grab" />
        <scroll-view scroll-y class="sheet-body">
          <view class="sheet-head">
            <view class="sheet-tags">
              <view class="category-tag" :class="'cat-' + (detail?.category || 'other')">
                <text>{{ categoryLabel(detail) }}</text>
              </view>
              <view class="status-tag" :class="'status-' + detailStatus.tone">
                <text>{{ detailStatus.label }}</text>
              </view>
            </view>
            <text class="sheet-title">{{ detail?.title || "通知详情" }}</text>
            <text class="sheet-meta">{{ detailClasses }} · {{ detailTime }}</text>
          </view>

          <view v-if="detailLoading" class="sheet-state">
            <text>加载中...</text>
          </view>
          <view v-else-if="detailError" class="sheet-state">
            <text class="sheet-error">{{ detailError }}</text>
            <view class="sheet-retry" @click="reloadDetail">
              <text>重试</text>
            </view>
          </view>
          <template v-else>
            <text class="sheet-content">{{ detail?.content || "" }}</text>

            <view v-if="showDetailStats" class="sheet-stats">
              <view class="sheet-stat">
                <text class="sheet-stat-num">{{ detailStats.read }}</text>
                <text class="sheet-stat-label">已读</text>
              </view>
              <view class="sheet-stat">
                <text class="sheet-stat-num">{{ detailStats.unread }}</text>
                <text class="sheet-stat-label">未读</text>
              </view>
              <view class="sheet-stat">
                <text class="sheet-stat-num">{{ detailStats.confirmed }}</text>
                <text class="sheet-stat-label">已确认</text>
              </view>
            </view>
            <view v-if="showDetailStats" class="sheet-progress">
              <view class="sheet-progress-fill" :style="{ width: detailRate + '%' }" />
            </view>

            <view v-if="detailAttachments.length" class="attachment-list">
              <text class="attachment-title">附件（{{ detailAttachments.length }}）</text>
              <view
                v-for="file in detailAttachments"
                :key="file.id || file.file_url"
                class="attachment-item"
              >
                <text class="attachment-icon">附</text>
                <text class="attachment-name">{{ file.file_name || "附件" }}</text>
              </view>
            </view>

            <view v-if="isDraft || isScheduled || canReadStatus || canRemind" class="sheet-actions">
              <view v-if="isDraft" class="sheet-btn is-primary" @click="editNotice">
                <text>继续编辑</text>
              </view>
              <view v-if="isScheduled" class="sheet-btn is-primary" @click="editNotice">
                <text>编辑定时</text>
              </view>
              <view v-if="canReadStatus" class="sheet-btn is-soft" @click="toggleUnread">
                <text>{{ unreadVisible ? "收起未读家长" : "查看未读家长" }}</text>
              </view>
              <view
                v-if="canRemind"
                class="sheet-btn is-primary"
                :class="{ 'is-disabled': reminding }"
                @click="remind"
              >
                <text>{{ reminding ? "提醒中..." : "再次提醒" }}</text>
              </view>
            </view>

            <view v-if="unreadVisible" class="unread-panel">
              <text class="unread-title">未读家长（{{ unreadTotal }}）</text>
              <view v-if="unreadLoading" class="unread-state">
                <text>加载中...</text>
              </view>
              <view v-else-if="unreadError" class="unread-state">
                <text class="sheet-error">{{ unreadError }}</text>
                <view class="sheet-retry" @click="toggleUnread">
                  <text>重试</text>
                </view>
              </view>
              <view v-else-if="unreadList.length === 0" class="unread-state">
                <text>家长都读完啦</text>
              </view>
              <view v-else>
                <view
                  v-for="row in unreadList"
                  :key="row.user_id || row.parent_name"
                  class="unread-row"
                >
                  <view class="unread-avatar">
                    <text>{{ (row.parent_name || "家")[0] }}</text>
                  </view>
                  <view class="unread-body">
                    <text class="unread-name">{{ row.parent_name || "家长" }}</text>
                    <text class="unread-sub">
                      {{ (row.student_name || "") + (relationLabel(row.relation) ? " · " + relationLabel(row.relation) : "") }}
                    </text>
                  </view>
                </view>
              </view>
            </view>

            <text v-if="detailNote" class="sheet-note">{{ detailNote }}</text>
          </template>
        </scroll-view>
        <view class="sheet-close" @click="closeDetail">
          <text>关闭</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  padding: $spacing-lg 28rpx calc(48rpx + env(safe-area-inset-bottom));
  background: $color-bg-page;
  box-sizing: border-box;
}

/* ===== 概要 ===== */
.hero {
  padding: 28rpx;
  border-radius: $border-radius-xl;
  background: $gradient-brand;
  box-shadow: $shadow-md;
}

.hero-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: $spacing-lg;
}

.hero-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.hero-title {
  font-size: $typography-font-size-3xl;
  font-weight: 700;
  line-height: 1.25;
  color: $color-text-on-brand;
}

.hero-sub {
  margin-top: 4rpx;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  color: rgba(255, 255, 255, 0.82);
}

.publish-btn {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 72rpx;
  padding: 0 24rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;

  text {
    font-size: $typography-font-size-md;
    font-weight: 600;
    color: $color-brand-primary-dark;
  }
}

.hero-stats {
  display: flex;
  margin-top: 24rpx;
  padding-top: 20rpx;
  border-top: 1rpx solid rgba(255, 255, 255, 0.24);
}

.hero-stat {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 0;
}

.hero-num {
  font-size: $typography-font-size-3xl;
  font-weight: 700;
  line-height: 1.2;
  color: $color-text-on-brand;
}

.hero-label {
  margin-top: 2rpx;
  font-size: $typography-font-size-xs;
  color: rgba(255, 255, 255, 0.78);
}

/* ===== 筛选 ===== */
.filters {
  margin-top: 22rpx;
}

.status-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.filter-pill {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 64rpx;
  padding: 0 8rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;

  text {
    font-size: $typography-font-size-sm;
    color: $color-text-muted;
  }

  &.active {
    border-color: $color-brand-primary;
    background: $color-brand-primary-bg;

    text {
      color: $color-brand-primary-dark;
      font-weight: 600;
    }
  }
}

.category-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
  margin-top: $spacing-sm;
}

.category-pill {
  display: flex;
  align-items: center;
  min-height: 56rpx;
  padding: 0 22rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;

  text {
    font-size: $typography-font-size-sm;
    color: $color-text-body;
  }

  &.active {
    border-color: $color-brand-primary;
    background: $color-brand-primary-bg;

    text {
      color: $color-brand-primary-dark;
      font-weight: 600;
    }
  }
}

.list-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 72rpx;
  margin-top: 4rpx;
}

.list-count {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.list-loading {
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

/* ===== 列表 ===== */
.notice-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.notice-card {
  padding: 24rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.card-top {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  flex-wrap: wrap;
}

.category-tag,
.status-tag {
  display: flex;
  align-items: center;
  min-height: 40rpx;
  padding: 0 14rpx;
  border-radius: $border-radius-full;

  text {
    font-size: $typography-font-size-xs;
    font-weight: 600;
    line-height: 1;
  }
}

.cat-homework {
  background: $color-brand-primary-bg;
  text { color: $color-brand-primary-dark; }
}
.cat-bring {
  background: $color-semantic-info-bg;
  text { color: $color-semantic-info; }
}
.cat-activity {
  background: $color-semantic-success-bg;
  text { color: $color-semantic-success; }
}
.cat-health {
  background: $color-semantic-warning-bg;
  text { color: $color-semantic-warning; }
}
.cat-other {
  background: $color-bg-page;
  text { color: $color-text-muted; }
}

.status-published {
  background: $color-semantic-success-bg;
  text { color: $color-semantic-success; }
}
.status-scheduled {
  background: $color-semantic-info-bg;
  text { color: $color-semantic-info; }
}
.status-draft {
  background: $color-bg-page;
  text { color: $color-text-muted; }
}
.status-revoked {
  background: $color-semantic-danger-bg;
  text { color: $color-semantic-danger; }
}

.card-time {
  margin-left: auto;
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.card-title {
  display: block;
  margin-top: 14rpx;
  font-size: $typography-font-size-lg;
  font-weight: 600;
  line-height: 1.4;
  color: $color-text-primary;
}

.card-preview {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  margin-top: 6rpx;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-text-muted;
}

.card-meta {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
  margin-top: 12rpx;
}

.meta-chip {
  display: flex;
  align-items: center;
  min-height: 38rpx;
  padding: 0 14rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  font-size: $typography-font-size-xs;
  color: $color-text-body;
}

.read-row {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  margin-top: 14rpx;
}

.read-bar {
  flex: 1;
  height: 10rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  overflow: hidden;
}

.read-fill {
  height: 100%;
  border-radius: $border-radius-full;
  background: $color-semantic-success;
}

.read-text {
  flex-shrink: 0;
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

.draft-note {
  display: block;
  margin-top: 12rpx;
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-sm;
  margin-top: 14rpx;
  padding-top: 12rpx;
  border-top: 1rpx solid $color-border-hairline;
}

.confirm-chip {
  display: flex;
  align-items: center;
  min-height: 38rpx;
  padding: 0 14rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  font-size: $typography-font-size-xs;
  color: $color-brand-primary-dark;

  &.is-muted {
    background: $color-bg-page;
    color: $color-text-hint;
  }
}

.open-link {
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

.load-more {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 72rpx;
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

/* ===== 详情弹层 ===== */
.sheet-mask {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 100;
  display: flex;
  align-items: flex-end;
  background: rgba(15, 23, 42, 0.42);
}

.sheet {
  width: 100%;
  max-height: 86vh;
  padding: 12rpx 32rpx calc(24rpx + env(safe-area-inset-bottom));
  border-radius: 32rpx 32rpx 0 0;
  background: $color-bg-card;
  box-sizing: border-box;
}

.sheet-grab {
  width: 72rpx;
  height: 8rpx;
  margin: 0 auto 18rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
}

.sheet-body {
  max-height: 62vh;
}

.sheet-head {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.sheet-tags {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.sheet-title {
  margin-top: 8rpx;
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  line-height: 1.4;
  color: $color-text-primary;
}

.sheet-meta {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.sheet-content {
  display: block;
  margin-top: 18rpx;
  font-size: $typography-font-size-md;
  line-height: 1.65;
  color: $color-text-body;
  white-space: pre-wrap;
}

.sheet-stats {
  display: flex;
  margin-top: 18rpx;
  padding: 18rpx 8rpx;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
}

.sheet-stat {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.sheet-stat-num {
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

.sheet-stat-label {
  margin-top: 2rpx;
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

.sheet-progress {
  height: 10rpx;
  margin-top: 10rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  overflow: hidden;
}

.sheet-progress-fill {
  height: 100%;
  border-radius: $border-radius-full;
  background: $color-semantic-success;
}

.attachment-list {
  margin-top: 18rpx;
}

.attachment-title {
  display: block;
  margin-bottom: $spacing-sm;
  font-size: $typography-font-size-sm;
  font-weight: 600;
  color: $color-text-primary;
}

.attachment-item {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  min-height: 64rpx;
  padding: 0 $spacing-md;
  margin-bottom: $spacing-sm;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.attachment-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36rpx;
  height: 36rpx;
  border-radius: $border-radius-sm;
  background: $color-brand-primary-bg;
  font-size: $typography-font-size-xs;
  color: $color-brand-primary-dark;
}

.attachment-name {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-sm;
  color: $color-text-body;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sheet-actions {
  display: flex;
  gap: $spacing-sm;
  margin-top: 18rpx;
}

.sheet-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 80rpx;
  padding: 0 8rpx;
  border-radius: $border-radius-full;

  text {
    font-size: $typography-font-size-md;
    font-weight: 600;
    line-height: 1.2;
    text-align: center;
  }

  &.is-soft {
    background: $color-brand-primary-bg;
    text { color: $color-brand-primary-dark; }
  }

  &.is-primary {
    background: $color-brand-primary-dark;
    text { color: $color-text-on-brand; }
  }

  &.is-disabled {
    opacity: 0.6;
  }
}

.unread-panel {
  margin-top: 18rpx;
  padding-top: 16rpx;
  border-top: 1rpx solid $color-border-hairline;
}

.unread-title {
  display: block;
  margin-bottom: $spacing-sm;
  font-size: $typography-font-size-sm;
  font-weight: 600;
  color: $color-text-primary;
}

.unread-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
  padding: $spacing-lg 0;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.unread-row {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  min-height: 76rpx;
}

.unread-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 52rpx;
  height: 52rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  font-size: $typography-font-size-xs;
  color: $color-brand-primary-dark;
}

.unread-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.unread-name {
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

.unread-sub {
  margin-top: 2rpx;
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

.sheet-note {
  display: block;
  margin-top: 14rpx;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  color: $color-text-hint;
}

.sheet-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
  padding: 48rpx 0;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.sheet-error {
  color: $color-semantic-danger;
}

.sheet-retry {
  padding: 12rpx 28rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;

  text {
    font-size: $typography-font-size-sm;
    color: $color-brand-primary-dark;
  }
}

.sheet-close {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 80rpx;
  margin-top: 12rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;

  text {
    font-size: $typography-font-size-md;
    color: $color-text-body;
  }
}
</style>
