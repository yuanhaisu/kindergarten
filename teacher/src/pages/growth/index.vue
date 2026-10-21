<script setup>
import { computed, ref } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchTeacherGrowthLogs } from "@/api/teacher-growth"
import { ensureLogin } from "@/utils/login"
import StateBlock from "@/components/StateBlock.vue"
import AppButton from "@/components/AppButton.vue"

const logs = ref([])
const loading = ref(false)
const loadError = ref("")
const searchQuery = ref("")

const filteredLogs = computed(() => {
  let result = logs.value
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    result = result.filter(item =>
      (item.content || "").toLowerCase().includes(q) ||
      (item.creator?.nickname || "").toLowerCase().includes(q)
    )
  }
  return result
})

const groupedLogs = computed(() => {
  const groups = new Map()
  for (const log of filteredLogs.value) {
    const key = (log.date || log.created_at || "").slice(0, 7) || "未知月份"
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key).push(log)
  }
  return Array.from(groups.entries()).map(([month, items]) => ({
    month,
    items,
    label: formatMonthLabel(month)
  }))
})

function formatMonthLabel(key) {
  if (!key || key === "未知月份") return "未知月份"
  const [y, m] = key.split("-")
  return `${y}年${parseInt(m)}月`
}

function formatDate(dateStr) {
  if (!dateStr) return ""
  const now = new Date()
  const d = new Date(dateStr)
  if (d.toDateString() === now.toDateString()) {
    return "今天 " + (dateStr.slice(11, 16) || "")
  }
  return dateStr.slice(0, 16)
}

function isClassRecord(item) {
  // target_type: 1=全班, 2=指定学生; 兼容旧数据（target_type 为 0 或 undefined 视为全班）
  if (item.target_type === 2) return false
  return true
}

function firstThumb(item) {
  return item.media_preview?.[0]?.thumb_url || ""
}

function todayKey() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`
}

const todayLabel = computed(() => {
  const d = new Date()
  const week = ["日", "一", "二", "三", "四", "五", "六"]
  return `${d.getMonth() + 1}月${d.getDate()}日 星期${week[d.getDay()]}`
})

// 「今天」与「待处理」都基于当前加载的一页（page_size=20，按日期倒序）：
// 今天的记录在最顶部，所以今日条数准确；待处理数是这 20 条内的下界，不是全量。
const todayCount = computed(() => logs.value.filter(l => l.date === todayKey()).length)
const pendingCount = computed(() => logs.value.filter(l =>
  (l.pending_suggestions_count || 0) > 0 ||
  (!isClassRecord(l) && l.sync_to_feed === false)
).length)

async function loadLogs() {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const res = await fetchTeacherGrowthLogs({ page: 1, page_size: 20 })
    logs.value = res.data?.list || []
  } catch (err) {
    loadError.value = err?.message || "加载失败"
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

function goCreate() {
  uni.navigateTo({ url: "/pages/growth/create" })
}

function goDetail(id) {
  uni.navigateTo({ url: `/pages/growth/detail?id=${id}` })
}

// 必须用 onShow：create 页 navigateBack 回来后 onLoad 不再触发，
// 刚发布的那条记录不会出现在列表里（老师端无 tabBar，故与家长端理由不同、结论相同）
onShow(loadLogs)
onPullDownRefresh(loadLogs)
</script>

<template>
  <view class="page">
    <!-- 今天 · 工作台头部：首屏先回答「今天记了没、还差什么」 -->
    <view class="today-bar">
      <text class="today-date">{{ todayLabel }}</text>
      <view class="today-stats">
        <view class="today-stat">
          <text class="today-stat-num">{{ todayCount }}</text>
          <text class="today-stat-label">今日已记</text>
        </view>
        <view class="today-stat">
          <text class="today-stat-num">{{ pendingCount }}</text>
          <text class="today-stat-label">条待处理</text>
        </view>
      </view>
    </view>

    <!-- 快速记录 — 页面主操作（取代原 FAB）。入口式：进创建页，不在此搬表单 -->
    <view class="quick-entry">
      <AppButton type="primary" size="lg" @click="goCreate">快速记录</AppButton>
    </view>

    <!-- 搜索：明标真实范围（只过滤已加载的 20 条，不是全量检索） -->
    <view class="search-box">
      <text class="search-icon">搜</text>
      <input
        v-model="searchQuery"
        class="search-input"
        placeholder="搜索内容或老师（仅最近 20 条）"
        placeholder-style="color: #94A3B8"
      />
    </view>

    <!-- 状态：加载/错误/空三态合一。「错误先于空」的判定顺序由 StateBlock 内部固化 -->
    <StateBlock
      v-if="loading || loadError || filteredLogs.length === 0"
      :loading="loading"
      :error="loadError"
      loading-text="加载成长记录..."
      :empty-text="searchQuery.trim() ? '未找到匹配的记录' : '暂无成长记录'"
      @retry="loadLogs"
    />

    <!-- 按月分组的记录列表 -->
    <view v-else v-for="group in groupedLogs" :key="group.month" class="month-group">
      <view class="group-head">
        <text class="group-month">{{ group.label }}</text>
        <text class="group-count">（{{ group.items.length }}条记录）</text>
      </view>

      <view class="card-list">
        <view
          v-for="item in group.items"
          :key="item.id"
          class="card"
          @click="goDetail(item.id)"
        >
          <!-- 班级记录 — 顶部图片（media_preview 为后端返回的缩略图预览） -->
          <image
            v-if="firstThumb(item)"
            :src="firstThumb(item)"
            mode="aspectFill"
            class="card-image"
          />

          <view class="card-body" :class="{ 'card-body--flush': firstThumb(item) }">
            <!-- 卡片头部 -->
            <view class="card-head">
              <view class="card-head-left">
                <view class="author-row">
                  <!-- 头像 -->
                  <view class="author-avatar">
                    <text class="avatar-text">
                      {{ (item.creator?.nickname || "老师")[0] }}
                    </text>
                  </view>
                  <text class="author-name">
                    {{ item.creator?.nickname || "老师" }}
                  </text>
                </view>
                <!-- 个体记录二级标题：表现标签 -->
                <text v-if="!isClassRecord(item) && item.performance_tags?.length" class="card-subtitle">
                  {{ item.performance_tags.join(" · ") }}
                </text>
              </view>
              <text class="card-date">{{ formatDate(item.date || item.created_at) }}</text>
            </view>

            <!-- 内容 -->
            <text class="card-content">{{ item.content }}</text>

            <!-- 底部 -->
            <view class="card-foot">
              <view class="card-foot-left">
                <template v-if="isClassRecord(item)">
                  <text v-if="item.related_students_count" class="student-count">
                    相关 {{ item.related_students_count }} 位小朋友
                  </text>
                </template>
                <template v-else>
                  <!-- 个体记录 — 同步状态（由 sync_to_feed 驱动） -->
                  <view class="status-tag" :class="{ synced: item.sync_to_feed }">
                    <text>{{ item.sync_to_feed ? '已同步给家长' : '未同步给家长' }}</text>
                  </view>
                </template>
                <!-- 待确认建议入口（D10）：不打开详情也知道哪条待处理 -->
                <view v-if="item.pending_suggestions_count" class="sugg-badge">
                  <text>AI建议 {{ item.pending_suggestions_count }} 待确认</text>
                </view>
              </view>

              <view class="card-stats">
                <view class="stat-item">
                  <text class="stat-icon">赞</text>
                  <text class="stat-num">{{ item.likes_count || 0 }}</text>
                </view>
                <view class="stat-item">
                  <text class="stat-icon">评</text>
                  <text class="stat-num">{{ item.comments_count || 0 }}</text>
                </view>
              </view>
            </view>
          </view>
        </view>
      </view>
    </view>

    <!-- 底部安全区 -->
    <view class="bottom-safe" />
  </view>
</template>

<style scoped lang="scss">
$slate-600: #475569;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: $spacing-lg 28rpx;
  background: $color-bg-page;
}

// ===== 今天 · 工作台头部 =====
.today-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-lg $spacing-xl;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.today-date {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
}

.today-stats {
  display: flex;
  align-items: baseline;
  gap: $spacing-lg;
}

.today-stat {
  display: flex;
  align-items: baseline;
  gap: $spacing-xs;
}

.today-stat-num {
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: $color-brand-primary;
}

.today-stat-label {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.quick-entry {
  margin-bottom: $spacing-lg;
}

// ===== 搜索框 =====
.search-box {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 32rpx;
  border: 2rpx solid #E2E8F0;
  border-radius: $border-radius-xl;
  background: #fff;
  margin-bottom: 24rpx;
}

.search-icon {
  flex-shrink: 0;
  font-size: 34rpx;
  font-weight: 700;
  color: $color-text-hint;
  margin-right: 24rpx;
}

.search-input {
  flex: 1;
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

// ===== 月份分组 =====
.month-group {
  margin-bottom: 32rpx;
}

.group-head {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: 24rpx;
}

.group-month {
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

.group-count {
  font-size: $typography-font-size-md;
  color: $color-text-hint;
}

// ===== 卡片列表 =====
.card-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.card {
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-md;
  overflow: hidden;
}

.card-image {
  width: 100%;
  height: 320rpx;
}

.card-body {
  padding: $spacing-xl;
}

// 有顶部缩略图时，正文不再重复上间距
.card-body--flush {
  padding-top: 0;
}

.card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.card-head-left {
  flex: 1;
  min-width: 0;
}

.author-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  flex-wrap: nowrap;
}

.author-avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 56rpx;
  height: 56rpx;
  border-radius: 999rpx;
  background: $color-semantic-info-bg;
}

.avatar-text {
  font-size: $typography-font-size-base;
  font-weight: 700;
  color: $color-semantic-info;
}

.author-name {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
}

.card-subtitle {
  display: block;
  margin-top: 8rpx;
  font-size: $typography-font-size-base;
  color: $color-text-muted;
}

.card-date {
  flex-shrink: 0;
  font-size: $typography-font-size-base;
  color: $color-text-hint;
  margin-top: 2rpx;
}

.card-content {
  display: block;
  margin-top: 16rpx;
  font-size: $typography-font-size-md;
  color: $slate-600;
  line-height: 1.6;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  overflow: hidden;
}

// ===== 卡片底部 =====
.card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 24rpx;
}

.card-foot-left {
  display: flex;
  align-items: center;
}

.student-count {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.status-tag {
  padding: $spacing-xs 24rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-warning-bg;
  font-size: $typography-font-size-base;
  font-weight: 600;
  color: $color-semantic-warning;

  &.synced {
    background: $color-semantic-success-bg;
    color: $color-semantic-success;
  }
}

.card-stats {
  display: flex;
  gap: $spacing-xl;
}

/* 待确认 AI 建议角标（D10 入口）：与左侧状态标签同排 */
.sugg-badge {
  margin-left: $spacing-md;
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-full;
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
  font-size: $typography-font-size-sm;
  font-weight: 600;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.stat-icon {
  font-size: $typography-font-size-md;
  font-weight: 700;
  color: $color-text-hint;
}

.stat-num {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.bottom-safe {
  height: calc(32rpx + env(safe-area-inset-bottom));
}
</style>
