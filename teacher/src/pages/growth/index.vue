<script setup>
import { computed, ref } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchTeacherGrowthLogs } from "@/api/teacher-growth"
import { fetchTeacherClassInfo } from "@/api/teacher-class"
import { ensureLogin } from "@/utils/login"
import { formatCardTime, formatHeadlineDate, formatMonthLabel, monthKeyOf, todayKey } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

const logs = ref([])
const loading = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const searchQuery = ref("")
const className = ref("")
const timeFilter = ref("all")
const sheetOpen = ref(false)
const todayLabel = ref(formatHeadlineDate(todayKey()))
// 时间选项跟着「今天」走：跨零点后刷新列表应换成新的本月/上月，而不是沿用 setup 时的月份。
const nowRef = ref(new Date())

// 「今天 / 已加载」是同一口径：只覆盖当前加载的一页（page_size=20），
// 所以页面把「仅统计已加载」写出来，避免被读成全量。
const isInitialLoading = computed(() => loading.value && !hasLoaded.value)
const showError = computed(() => Boolean(loadError.value) && !hasLoaded.value)
const locked = computed(() => isInitialLoading.value || showError.value)

const filterActive = computed(() => timeFilter.value !== "all" || searchQuery.value.trim() !== "")
const scopeNote = computed(() =>
  filterActive.value ? "仅统计已加载 · 不受筛选影响" : "仅统计已加载"
)

const todayCount = computed(() => logs.value.filter(item => item.date === todayKey()).length)
const pendingCount = computed(() =>
  logs.value.filter(item =>
    (item.pending_suggestions_count || 0) > 0 ||
    (!isClassRecord(item) && item.sync_to_feed === false)
  ).length
)
const statToday = computed(() => (locked.value ? "—" : todayCount.value))
const statPending = computed(() => (locked.value ? "—" : pendingCount.value))

function monthKeyWithOffset(offset) {
  const base = nowRef.value
  const d = new Date(base.getFullYear(), base.getMonth() + offset, 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`
}

const timeOptions = computed(() => {
  const base = nowRef.value
  const prev = new Date(base.getFullYear(), base.getMonth() - 1, 1)
  return [
    { value: "all", label: "全部时间", chip: "全部时间" },
    { value: monthKeyWithOffset(0), label: `本月（${base.getMonth() + 1}月）`, chip: `${base.getMonth() + 1}月` },
    { value: monthKeyWithOffset(-1), label: `上月（${prev.getMonth() + 1}月）`, chip: `上月（${prev.getMonth() + 1}月）` }
  ]
})

const activeTimeOption = computed(() =>
  timeOptions.value.find(option => option.value === timeFilter.value) || timeOptions.value[0]
)

const filteredLogs = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  return logs.value.filter(item => {
    const month = monthKeyOf(item.date || item.created_at)
    if (timeFilter.value !== "all" && month !== timeFilter.value) return false
    if (!query) return true
    return (item.content || "").toLowerCase().includes(query) ||
      (item.creator?.nickname || "").toLowerCase().includes(query)
  })
})

const monthGroups = computed(() => {
  const groups = new Map()
  for (const item of filteredLogs.value) {
    const key = monthKeyOf(item.date || item.created_at) || "未知月份"
    if (!groups.has(key)) {
      groups.set(key, { key, label: formatMonthLabel(key) || "未知月份", items: [] })
    }
    groups.get(key).items.push(item)
  }
  return Array.from(groups.values())
})

const isClassRecordSafe = item => isClassRecord(item)
const viewState = computed(() => {
  if (isInitialLoading.value) return "loading"
  if (showError.value) return "error"
  if (!hasLoaded.value) return "normal"
  if (logs.value.length === 0) return "empty"
  if (filteredLogs.value.length === 0 && filterActive.value) return "noresult"
  return "normal"
})

const emptyCopy = computed(() => {
  if (searchQuery.value.trim()) {
    return { title: "未找到匹配的记录", hint: "换一个关键词，或清除搜索查看全部记录。", action: "清除搜索" }
  }
  return { title: "未找到匹配的记录", hint: "当前筛选范围内没有记录，换一个时间试试。", action: "清除筛选" }
})

const listFootText = computed(() => {
  if (filterActive.value) {
    return `已加载 ${logs.value.length} 条 · 当前显示 ${filteredLogs.value.length} 条`
  }
  return `已显示最近 ${logs.value.length} 条 · 下拉刷新`
})

function isClassRecord(item) {
  // target_type: 1=全班, 2=指定幼儿；兼容旧数据（0/undefined）视为全班
  return item.target_type !== 2
}

function mediaOf(item) {
  return Array.isArray(item.media_preview) ? item.media_preview : []
}

function mediaLayout(item) {
  const count = mediaOf(item).length
  if (count <= 1) return "media-one"
  if (count <= 3) return `media-row media-row-${count}`
  return "media-grid"
}

function mediaTiles(item) {
  const items = mediaOf(item)
  return items.length > 4 ? items.slice(0, 4) : items
}

function mediaOverflow(item) {
  return Math.max(0, mediaOf(item).length - 4)
}

function creatorName(item) {
  return item.creator?.nickname || "老师"
}

function creatorInitial(item) {
  return String(creatorName(item)).slice(0, 1) || "师"
}

function recordTime(item) {
  return formatCardTime(item.date, item.created_at)
}

function tagLine(item) {
  return (item.performance_tags || []).filter(Boolean).join(" · ")
}

async function loadClassName() {
  try {
    const res = await fetchTeacherClassInfo()
    className.value = res.data?.name || ""
  } catch (err) {
    // 班级名只用于筛选条展示；失败时退回中性文案，不阻断列表
    console.warn("[growth-index] load class failed:", err?.message || err)
  }
}

async function loadLogs() {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const res = await fetchTeacherGrowthLogs({ page: 1, page_size: 20 })
    logs.value = res.data?.list || []
    hasLoaded.value = true
    nowRef.value = new Date()
    todayLabel.value = formatHeadlineDate(todayKey())
    // 跨月后「上月」会变成新的月份：旧的筛选值已不在选项里时回到全部时间，
    // 避免 chip 显示「全部时间」却仍在按旧月份过滤。
    if (timeFilter.value !== "all" && !timeOptions.value.some(option => option.value === timeFilter.value)) {
      timeFilter.value = "all"
    }
    if (!className.value) loadClassName()
  } catch (err) {
    // 刷新失败时保留已加载内容：只有第一次就失败才切整页错误态
    console.warn("[growth-index] load failed:", err?.message || err)
    loadError.value = err?.message || "加载失败"
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

async function handleRetry() {
  if (loading.value) return
  await loadLogs()
  if (!loadError.value) {
    uni.showToast({ title: "已重新加载", icon: "none" })
  }
}

function goCreate() {
  if (locked.value) return
  uni.navigateTo({ url: "/pages/growth/create" })
}

function goDetail(id) {
  uni.navigateTo({ url: `/pages/growth/detail?id=${id}` })
}

function openTimeSheet() {
  if (locked.value) return
  sheetOpen.value = true
}

function selectTime(value) {
  timeFilter.value = value
  sheetOpen.value = false
}

function clearSearch() {
  searchQuery.value = ""
}

function runEmptyAction() {
  if (searchQuery.value.trim()) {
    clearSearch()
    return
  }
  if (timeFilter.value !== "all") {
    timeFilter.value = "all"
    return
  }
  goCreate()
}

// 必须用 onShow：create 页 navigateBack 回来后 onLoad 不再触发，刚发布的记录不会出现。
onShow(loadLogs)
onPullDownRefresh(loadLogs)
</script>

<template>
  <PageShell>
    <view class="page" :class="'state-' + viewState">
      <!-- 头部：日期 + 唯一主操作「发布」 -->
      <view class="page-head">
        <view class="head-top">
          <view class="date-block">
            <text class="date-main">{{ todayLabel.main }}</text>
            <text class="date-week">{{ todayLabel.weekday }}</text>
          </view>
          <view
            v-if="viewState !== 'empty'"
            class="publish-btn"
            :class="{ 'is-disabled': locked }"
            @click="goCreate"
          >
            <text class="publish-plus icon">&#xea13;</text>
            <text>发布</text>
          </view>
        </view>
        <view class="stat-row">
          <view class="stat">
            <text class="stat-num">{{ statToday }}</text>
            <text class="stat-label">今日已记</text>
          </view>
          <view class="stat-divider" />
          <view class="stat">
            <text class="stat-num">{{ statPending }}</text>
            <text class="stat-label">条待处理</text>
          </view>
          <text class="stat-scope">{{ scopeNote }}</text>
        </view>
      </view>

      <!-- 工具区：静态班级名（不可点）+ 时间筛选 + 搜索 -->
      <view class="tools">
        <view class="filter-bar">
          <view class="chip chip-static">
            <text>{{ className || "当前班级" }}</text>
          </view>
          <view
            class="chip"
            :class="{ 'is-active': timeFilter !== 'all', 'is-disabled': locked }"
            @click="openTimeSheet"
          >
            <text>{{ activeTimeOption.chip }}</text>
            <text class="chevron icon">&#xea4e;</text>
          </view>
        </view>
        <view class="search-shell" :class="{ 'is-disabled': locked }">
          <text class="search-icon icon">&#xf0d1;</text>
          <input
            v-model="searchQuery"
            class="search-input"
            type="text"
            :disabled="locked"
            placeholder="搜索内容或老师 · 仅最近 20 条"
            confirm-type="search"
          />
          <view v-if="searchQuery && !locked" class="search-clear" @click="clearSearch">
            <text class="icon">&#xeb99;</text>
          </view>
        </view>
      </view>

      <!-- 加载：3 张骨架 -->
      <view v-if="viewState === 'loading'" class="state-loading">
        <view class="loading-note">
          <view class="spinner" />
          <text>加载成长记录...</text>
        </view>
        <view v-for="i in 3" :key="i" class="skeleton-card">
          <view class="sk-block sk-media" />
          <view class="sk-author">
            <view class="sk-block sk-avatar" />
            <view class="sk-lines">
              <view class="sk-block sk-line line-a" />
              <view class="sk-block sk-line line-b" />
            </view>
          </view>
        </view>
      </view>

      <!-- 错误 -->
      <view v-else-if="viewState === 'error'" class="state-error">
        <view class="error-icon"><text>!</text></view>
        <text class="error-title">加载失败，请检查网络</text>
        <view class="btn-ghost" @click="handleRetry">重试</view>
      </view>

      <!-- 服务端空态 -->
      <view v-else-if="viewState === 'empty'" class="state-empty">
        <view class="empty-art">
          <view class="empty-orbit" />
          <view class="empty-planet" />
          <view class="empty-orbit-dot" />
          <view class="empty-star s1" />
          <view class="empty-star s2" />
          <view class="empty-star s3" />
        </view>
        <text class="empty-text">还没有成长记录，从今天开始吧</text>
        <view class="btn-gradient" @click="goCreate">发布第一条</view>
      </view>

      <!-- 筛选/搜索无结果：保留发布与筛选条，统计不受影响 -->
      <view v-else-if="viewState === 'noresult'" class="state-empty state-noresult">
        <view class="empty-art">
          <view class="empty-orbit" />
          <view class="empty-planet" />
          <view class="empty-orbit-dot" />
          <view class="empty-star s1" />
          <view class="empty-star s2" />
          <view class="empty-star s3" />
        </view>
        <text class="empty-text">{{ emptyCopy.title }}</text>
        <text class="empty-hint">{{ emptyCopy.hint }}</text>
        <view class="btn-ghost" @click="runEmptyAction">{{ emptyCopy.action }}</view>
      </view>

      <!-- 正常列表：按月分组 -->
      <view v-else class="list-region">
        <view v-for="group in monthGroups" :key="group.key" class="month-group">
          <view class="month-head">
            <text class="month-title">{{ group.label }}</text>
            <text class="month-count">（{{ group.items.length }}条记录）</text>
          </view>
          <view class="card-stack">
            <view
              v-for="item in group.items"
              :key="item.id"
              class="record-card"
              @click="goDetail(item.id)"
            >
              <view class="card-inner">
                <!-- 媒体区：1 张 16:9；2–3 张等分横排；≥5 张 2×2 + 末格 +N -->
                <view
                  v-if="mediaOf(item).length"
                  class="card-media"
                  :class="mediaLayout(item)"
                >
                  <view
                    v-for="(media, index) in mediaTiles(item)"
                    :key="index"
                    class="media-tile"
                    :class="'ph-' + (index % 6 + 1)"
                  >
                    <image
                      v-if="media.thumb_url"
                      class="media-img"
                      :src="media.thumb_url"
                      mode="aspectFill"
                    />
                    <view v-if="mediaLayout(item) === 'media-grid' && index === 3 && mediaOverflow(item)" class="media-more">
                      <text>+{{ mediaOverflow(item) }}</text>
                    </view>
                  </view>
                  <text class="media-count">{{ mediaOf(item).length }} 张</text>
                </view>

                <view class="author-row">
                  <view class="avatar">
                    <text>{{ creatorInitial(item) }}</text>
                  </view>
                  <view class="author-main">
                    <text class="author-name">{{ creatorName(item) }}</text>
                  </view>
                  <text class="record-date">{{ recordTime(item) }}</text>
                </view>

                <text v-if="!isClassRecordSafe(item) && tagLine(item)" class="tag-line">
                  {{ tagLine(item) }}
                </text>

                <text class="card-content">{{ item.content }}</text>

                <view class="card-foot">
                  <view class="foot-left">
                    <text
                      v-if="!isClassRecordSafe(item) && item.related_students_count"
                      class="related-text"
                    >相关 {{ item.related_students_count }} 位小朋友</text>
                    <!-- 决策：全班记录只显示同步状态；related_students_count 对全班恒 0，不伪造人数 -->
                    <view
                      class="badge"
                      :class="item.sync_to_feed ? 'badge-success' : 'badge-warning'"
                    >
                      <view class="badge-dot" />
                      <text>{{ item.sync_to_feed ? "已同步给家长" : "未同步给家长" }}</text>
                    </view>
                    <view v-if="item.pending_suggestions_count" class="badge badge-info">
                      <view class="badge-dot" />
                      <text>AI建议 <text class="badge-strong">{{ item.pending_suggestions_count }}</text> 待确认</text>
                    </view>
                  </view>
                  <view class="foot-counts">
                    <view class="count-item">
                      <text class="count-icon icon">&#xee0f;</text>
                      <text>{{ item.likes_count || 0 }}</text>
                    </view>
                    <view class="count-item">
                      <text class="count-icon icon">&#xeb51;</text>
                      <text>{{ item.comments_count || 0 }}</text>
                    </view>
                  </view>
                </view>
              </view>
            </view>
          </view>
        </view>
        <text class="list-foot">{{ listFootText }}</text>
      </view>
    </view>

    <!-- 时间筛选弹层：口径只看已加载的一页 -->
    <view v-if="sheetOpen" class="sheet-layer">
      <view class="sheet-scrim" @click="sheetOpen = false" />
      <view class="sheet">
        <view class="sheet-grabber" />
        <view class="sheet-head">
          <text class="sheet-title">选择时间</text>
          <view class="sheet-close" @click="sheetOpen = false">
            <text class="icon">&#xeb99;</text>
          </view>
        </view>
        <view class="sheet-options">
          <view
            v-for="option in timeOptions"
            :key="option.value"
            class="sheet-option"
            :class="{ 'is-selected': timeFilter === option.value }"
            @click="selectTime(option.value)"
          >
            <text>{{ option.label }}</text>
            <text v-if="timeFilter === option.value" class="option-check icon">&#xeb7b;</text>
          </view>
        </view>
        <text class="sheet-note">筛选作用于已加载的最近 20 条</text>
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

/* ---------- 头部 ---------- */
.page-head {
  padding: 0 2rpx;
}

.head-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: $spacing-md $spacing-lg;
}

.date-block {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: $spacing-sm $spacing-md;
  min-width: 0;
}

.date-main {
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.date-week {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.publish-btn {
  flex: none;
  height: 76rpx;
  padding: 0 $spacing-xl;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.96);
  }
}

.publish-btn.is-disabled {
  background: $color-bg-page;
  border: 1rpx solid $color-border-hairline;
  color: $color-text-hint;
  box-shadow: none;
}

.publish-plus {
  font-size: $typography-font-size-base;
}

.stat-row {
  margin-top: $spacing-xl;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-md $spacing-lg;
}

.stat {
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
}

.stat-num {
  min-width: 60rpx;
  height: 60rpx;
  padding: 0 $spacing-xs;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.stat-label {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.stat-divider {
  width: 1rpx;
  height: 36rpx;
  background: $color-border-hairline;
}

.stat-scope {
  margin-left: auto;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

/* ---------- 工具区 ---------- */
.tools {
  margin-top: $spacing-xl;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.filter-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-md;
}

.chip {
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  box-shadow: $shadow-sm;
}

.chip.is-active {
  background: $color-brand-primary-bg;
  border-color: $color-brand-primary-light;
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

.chip.is-active .chevron {
  color: $color-brand-primary-dark;
}

.chip.is-disabled {
  background: $color-bg-page;
  border-color: $color-border-hairline;
  color: $color-text-hint;
  box-shadow: none;
}

.chip-static {
  color: $color-text-body;
}

.chevron {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
}

.search-shell {
  height: 88rpx;
  padding: 0 $spacing-md 0 $spacing-lg;
  display: flex;
  align-items: center;
  gap: $spacing-md;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.search-shell.is-disabled {
  background: $color-bg-page;
  box-shadow: none;
}

.search-icon {
  color: $color-text-body;
  font-size: $typography-font-size-lg;
}

.search-input {
  flex: 1;
  min-width: 0;
  height: 80rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
}

.search-shell.is-disabled .search-input,
.search-shell.is-disabled .search-icon {
  color: $color-text-hint;
}

.search-clear {
  flex: none;
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
}

/* ---------- 列表 ---------- */
.list-region {
  margin-top: $spacing-xl;
}

.month-group + .month-group {
  margin-top: $spacing-xl;
}

.month-head {
  padding: 0 2rpx $spacing-md;
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: $spacing-xs;
}

.month-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.month-count {
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.card-stack {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.record-card {
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.98);
    box-shadow: $shadow-sm;
  }
}

.card-inner {
  padding: $spacing-xl;
}

/* ---------- 媒体区 ---------- */
.card-media {
  position: relative;
  margin-bottom: $spacing-lg;
}

.media-one .media-tile {
  height: 348rpx;
}

.media-row {
  display: grid;
  gap: $spacing-xs;
  height: 192rpx;
}

.media-row .media-tile {
  height: 100%;
}

.media-row-2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.media-row-3 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.media-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: $spacing-xs;
}

.media-grid .media-tile {
  height: 184rpx;
}

.media-tile {
  position: relative;
  overflow: hidden;
  min-width: 0;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.media-img {
  width: 100%;
  height: 100%;
  display: block;
}

.ph-1 {
  background: linear-gradient(145deg, $color-brand-primary-bg 0%, $color-bg-page 58%, $color-semantic-warning-bg 100%);
}

.ph-2 {
  background: linear-gradient(150deg, $color-semantic-info-bg 0%, $color-bg-card 60%, $color-brand-primary-bg 100%);
}

.ph-3 {
  background: linear-gradient(140deg, $color-semantic-success-bg 0%, $color-bg-page 64%, $color-brand-primary-bg 100%);
}

.ph-4 {
  background: linear-gradient(155deg, $color-brand-primary-bg 0%, $color-semantic-info-bg 100%);
}

.ph-5 {
  background: linear-gradient(150deg, $color-bg-page 0%, $color-semantic-warning-bg 100%);
}

.ph-6 {
  background: linear-gradient(145deg, $color-semantic-info-bg 0%, $color-bg-page 58%, $color-semantic-success-bg 100%);
}

.media-count {
  position: absolute;
  top: $spacing-xs;
  right: $spacing-xs;
  z-index: 3;
  height: 44rpx;
  padding: 0 $spacing-xs;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  box-shadow: $shadow-sm;
}

.media-more {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  background: $color-text-primary;
  color: $color-text-on-brand;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
}

/* ---------- 卡片文字 ---------- */
.author-row {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.avatar {
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
}

.author-main {
  flex: 1;
  min-width: 0;
}

.author-name {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.record-date {
  flex: none;
  padding-left: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  text-align: right;
}

.tag-line {
  margin-top: $spacing-xs;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.6;
}

.card-content {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  margin-top: $spacing-md;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.card-foot {
  margin-top: $spacing-lg;
  padding-top: $spacing-md;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: $spacing-sm;
  border-top: 1rpx solid $color-border-subtle;
}

.foot-left {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-xs;
}

.related-text {
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.badge {
  height: 44rpx;
  padding: 0 $spacing-xs;
  display: inline-flex;
  align-items: center;
  gap: 4rpx;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.badge-dot {
  width: 10rpx;
  height: 10rpx;
  flex: none;
  border-radius: $border-radius-full;
}

.badge-success {
  background: $color-semantic-success-bg;
}

.badge-success .badge-dot {
  background: $color-semantic-success;
}

.badge-warning {
  background: $color-semantic-warning-bg;
}

.badge-warning .badge-dot {
  background: $color-semantic-warning;
}

.badge-info {
  background: $color-semantic-info-bg;
}

.badge-info .badge-dot {
  background: $color-semantic-info;
}

.badge-strong {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

.foot-counts {
  flex: none;
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.count-item {
  display: inline-flex;
  align-items: center;
  gap: 4rpx;
}

.count-icon {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
}

.list-foot {
  width: fit-content;
  max-width: 100%;
  margin: $spacing-lg auto 0;
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.6;
  text-align: center;
}

/* ---------- 空 / 无结果 / 错误 / 加载 ---------- */
.state-empty {
  padding: 92rpx $spacing-lg 56rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.state-noresult {
  padding-top: 72rpx;
}

.empty-art {
  position: relative;
  width: 264rpx;
  height: 264rpx;
  margin-bottom: $spacing-lg;
}

.empty-planet {
  position: absolute;
  top: 48rpx;
  left: 48rpx;
  width: 168rpx;
  height: 168rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand-soft;
  box-shadow: $shadow-md;
}

.empty-orbit {
  position: absolute;
  top: 68rpx;
  left: 6rpx;
  width: 252rpx;
  height: 132rpx;
  border: 2rpx dashed $color-brand-primary-light;
  border-radius: $border-radius-full;
  transform: rotate(-18deg);
}

.empty-orbit-dot {
  position: absolute;
  top: 46rpx;
  right: 26rpx;
  width: 18rpx;
  height: 18rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-light;
}

.empty-star {
  position: absolute;
  width: 8rpx;
  height: 8rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-light;
}

.empty-star.s1 {
  top: 40rpx;
  left: 36rpx;
}

.empty-star.s2 {
  bottom: 18rpx;
  left: 76rpx;
  width: 6rpx;
  height: 6rpx;
}

.empty-star.s3 {
  right: 40rpx;
  bottom: 54rpx;
  width: 6rpx;
  height: 6rpx;
}

.empty-text {
  max-width: 500rpx;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.8;
}

.empty-hint {
  max-width: 500rpx;
  margin-top: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.7;
}

.btn-gradient,
.btn-ghost {
  min-height: 88rpx;
  padding: 0 $spacing-xl;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.btn-gradient {
  margin-top: $spacing-lg;
  background: $gradient-brand;
  color: $color-text-primary;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.97);
  }
}

.btn-ghost {
  margin-top: $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  background: $color-brand-primary-bg;
  color: $color-text-primary;
  box-shadow: $shadow-sm;

  &:active {
    transform: scale(0.97);
  }
}

.state-error {
  padding: 84rpx $spacing-lg 48rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.error-icon {
  width: 128rpx;
  height: 128rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-bold;
  box-shadow: $shadow-sm;
}

.error-title {
  margin-top: $spacing-md;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.state-loading .loading-note {
  margin: 0 2rpx $spacing-lg;
  display: flex;
  align-items: center;
  gap: $spacing-md;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.spinner {
  width: 28rpx;
  height: 28rpx;
  flex: none;
  border: 4rpx solid $color-border-hairline;
  border-top-color: $color-brand-primary-dark;
  border-radius: $border-radius-full;
  animation: spin 900ms linear infinite;
}

.skeleton-card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.skeleton-card + .skeleton-card {
  margin-top: $spacing-lg;
}

.sk-block {
  background: $color-border-subtle;
  animation: skeleton-pulse 1500ms $animation-easing-default infinite;
}

.sk-media {
  width: 100%;
  height: 192rpx;
  border-radius: $border-radius-lg;
}

.sk-author {
  margin-top: $spacing-lg;
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.sk-avatar {
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  flex: none;
  border-radius: $border-radius-full;
}

.sk-lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.sk-line {
  height: 20rpx;
  border-radius: $border-radius-full;
}

.sk-line.line-a {
  width: 38%;
}

.sk-line.line-b {
  width: 88%;
}

/* ---------- 时间弹层 ---------- */
.sheet-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 80;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.sheet-scrim {
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
  width: 100%;
  padding: $spacing-md $component-page-padding-x calc(#{$spacing-xl} + env(safe-area-inset-bottom));
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.sheet-grabber {
  width: 72rpx;
  height: 8rpx;
  margin: 0 auto $spacing-md;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
}

.sheet-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.sheet-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
}

.sheet-close {
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  color: $color-text-body;
}

.sheet-options {
  margin-top: $spacing-md;
  display: flex;
  flex-direction: column;
}

.sheet-option {
  min-height: 104rpx;
  padding: 0 $spacing-md;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-radius: $border-radius-xl;
  color: $color-text-body;
  font-size: $typography-font-size-md;
}

.sheet-option.is-selected {
  background: $color-brand-primary-bg;
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

.option-check {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-lg;
}

.sheet-note {
  margin-top: $spacing-md;
  padding-top: $spacing-md;
  border-top: 1rpx solid $color-border-subtle;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  line-height: 1.6;
  text-align: center;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
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
