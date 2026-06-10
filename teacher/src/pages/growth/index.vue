<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchTeacherGrowthLogs } from "@/api/teacher-growth"
import { ensureLogin } from "@/utils/login"

const logs = ref([]) 
const loading = ref(false)
const loadError = ref("")
const searchQuery = ref("")
const activeCategory = ref("全部")

const categories = ["全部", "今日", "学习", "体能", "艺术", "其他"]

const categoryStyleMap = {
  "全部": { bg: "#EFF6FF", fg: "#3B82F6", border: "#BFDBFE" },
  "今日": { bg: "#DCFCE7", fg: "#16A34A", border: "#BBF7D0" },
  "学习": { bg: "#DBEAFE", fg: "#2563EB", border: "#BFDBFE" },
  "体能": { bg: "#FFF7ED", fg: "#F97316", border: "#FED7AA" },
  "艺术": { bg: "#F3E8FF", fg: "#9333EA", border: "#E9D5FF" },
  "其他": { bg: "#F1F5F9", fg: "#64748B", border: "#E2E8F0" }
}

const filteredLogs = computed(() => {
  let result = logs.value
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    result = result.filter(item =>
      (item.content || "").toLowerCase().includes(q) ||
      (item.student_name || "").toLowerCase().includes(q) ||
      (item.title || "").toLowerCase().includes(q) ||
      (item.creator?.nickname || "").toLowerCase().includes(q)
    )
  }
  if (activeCategory.value !== "全部") {
    result = result.filter(item =>
      (item.category === activeCategory.value) ||
      (item.performance_tags || []).includes(activeCategory.value)
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

const pendingTasks = computed(() => {
  return logs.value.filter(item => item.status === "pending" || item.status === "draft").slice(0, 3)
})

function getCategoryStyle(cat) {
  const active = activeCategory.value === cat
  const s = categoryStyleMap[cat] || categoryStyleMap["其他"]
  if (active) return { backgroundColor: s.bg, color: s.fg, borderColor: s.border }
  return { backgroundColor: "#FFFFFF", color: "#64748B", borderColor: "#E2E8F0" }
}

function getTagStyle(cat) {
  const s = categoryStyleMap[cat] || categoryStyleMap["其他"]
  return { backgroundColor: s.bg, color: s.fg }
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

function selectCategory(cat) {
  activeCategory.value = cat
}

onLoad(loadLogs)
onPullDownRefresh(loadLogs)
</script>

<template>
  <view class="page">
    <!-- 今日待记录卡片 -->
    <view v-if="pendingTasks.length" class="pending-card">
      <view class="pending-head">
        <view class="pending-head-left">
          <view class="pending-icon-box">
            <text class="pending-icon">待</text>
          </view>
          <text class="pending-title">今日待记录</text>
        </view>
        <view class="pending-badge">
          <text>{{ pendingTasks.length }}条待完成</text>
        </view>
      </view>
      <view class="pending-list">
        <view
          v-for="task in pendingTasks"
          :key="task.id"
          class="pending-item"
          @click="goDetail(task.id)"
        >
          <view class="pending-dot" />
          <text class="pending-item-text">{{ task.content || task.title || "待记录事项" }}</text>
          <view class="pending-go-btn" @click.stop="goCreate">
            <text>去记录</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 搜索框 -->
    <view class="search-box">
      <text class="search-icon">搜</text>
      <input
        v-model="searchQuery"
        class="search-input"
        placeholder="搜索学生姓名或记录内容"
        placeholder-style="color: #94A3B8; font-size: 28rpx"
      />
    </view>

    <!-- 分类标签 -->
    <scroll-view scroll-x class="category-scroll" :show-scrollbar="false">
      <view class="category-row">
        <view
          v-for="cat in categories"
          :key="cat"
          class="category-pill"
          :class="{ active: activeCategory === cat }"
          :style="getCategoryStyle(cat)"
          @click="selectCategory(cat)"
        >
          <text>{{ cat }}</text>
        </view>
      </view>
    </scroll-view>

    <!-- 状态 -->
    <view v-if="loading" class="state-wrap">
      <text class="state-text">加载中...</text>
    </view>
    <view v-else-if="loadError" class="state-wrap">
      <text class="state-text">{{ loadError }}</text>
      <button class="retry-btn" @click="loadLogs">重试</button>
    </view>
    <view v-else-if="filteredLogs.length === 0" class="state-wrap">
      <text class="state-text">暂无成长记录</text>
    </view>

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
          :class="{ 'has-image': item.media?.length && isClassRecord(item) }"
          @click="goDetail(item.id)"
        >
          <!-- 班级记录 — 顶部图片 -->
          <image
            v-if="item.media?.length && isClassRecord(item)"
            :src="item.media[0].thumb_url || item.media[0].url"
            mode="aspectFill"
            class="card-image"
          />

          <view class="card-body" :style="{ paddingTop: (item.media?.length && isClassRecord(item)) ? '0' : '32rpx' }">
            <!-- 卡片头部 -->
            <view class="card-head">
              <view class="card-head-left">
                <view class="author-row">
                  <!-- 头像 -->
                  <view class="author-avatar" :class="{ 'is-student': !isClassRecord(item) }">
                    <text class="avatar-text">
                      {{ isClassRecord(item) ? (item.creator?.nickname || "老")[0] : (item.student_name || "生")[0] }}
                    </text>
                  </view>
                  <text class="author-name">
                    {{ isClassRecord(item) ? (item.creator?.nickname || "老师") : (item.student_name || "学生") }}
                  </text>
                  <!-- 分类标签 -->
                  <view
                    v-if="item.category"
                    class="cat-tag"
                    :style="getTagStyle(item.category)"
                  >
                    <text>{{ item.category }}</text>
                  </view>
                </view>
                <!-- 标题（仅班级记录有独立标题） -->
                <text v-if="isClassRecord(item) && item.title" class="card-title">{{ item.title }}</text>
                <!-- 个体记录二级标题 -->
                <text v-if="!isClassRecord(item) && item.performance_tags" class="card-subtitle">
                  {{ Array.isArray(item.performance_tags) ? item.performance_tags.join(" · ") : item.performance_tags }}
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
                  <!-- 班级记录 — 学生头像组 -->
                  <view class="student-avatars">
                    <view class="mini-avatar" v-for="i in Math.min(3, item.student_count || 1)" :key="i" />
                  </view>
                  <text v-if="item.student_count" class="student-count">{{ item.student_count }}位小朋友</text>
                </template>
                <template v-else>
                  <!-- 个体记录 — 同步状态 -->
                  <view v-if="item.status" class="status-tag" :class="{ synced: item.status === 'synced' }">
                    <text>{{ item.status === 'synced' ? '已同步给家长' : '待同步' }}</text>
                  </view>
                </template>
              </view>

              <view class="card-stats">
                <view class="stat-item">
                  <text class="stat-icon">看</text>
                  <text class="stat-num">{{ item.views_count || 0 }}</text>
                </view>
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

    <!-- 悬浮添加按钮 -->
    <view class="fab-btn" @click="goCreate">
      <text class="fab-icon">＋</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
$blue-500: #3B82F6;
$blue-50: #EFF6FF;
$slate-50: #F8FAFC;
$slate-400: #94A3B8;
$slate-500: #64748B;
$slate-600: #475569;
$slate-800: #1E293B;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 24rpx 28rpx;
  background: $slate-50;
}

// ===== 今日待记录卡片 =====
.pending-card {
  margin-bottom: 24rpx;
  padding: 32rpx;
  border-radius: 24rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
}

.pending-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.pending-head-left {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.pending-icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  border-radius: 16rpx;
  background: $blue-50;
}

.pending-icon {
  font-size: 34rpx;
  font-weight: 700;
  color: $blue-500;
}

.pending-title {
  font-size: 30rpx;
  font-weight: 700;
  color: $slate-800;
}

.pending-badge {
  padding: 8rpx 24rpx;
  border-radius: 999rpx;
  background: #FEF2F2;
}

.pending-badge text {
  font-size: 24rpx;
  font-weight: 500;
  color: #EF4444;
}

.pending-list {
  margin-top: 24rpx;
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.pending-item {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.pending-dot {
  flex-shrink: 0;
  width: 16rpx;
  height: 16rpx;
  border-radius: 999rpx;
  background: #EF4444;
}

.pending-item-text {
  flex: 1;
  font-size: 28rpx;
  color: $slate-600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pending-go-btn {
  flex-shrink: 0;
  padding: 8rpx 24rpx;
  border-radius: 16rpx;
  background: $blue-50;
}

.pending-go-btn text {
  font-size: 24rpx;
  font-weight: 500;
  color: $blue-500;
}

// ===== 搜索框 =====
.search-box {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 32rpx;
  border: 2rpx solid #E2E8F0;
  border-radius: 24rpx;
  background: #fff;
  margin-bottom: 24rpx;
}

.search-icon {
  flex-shrink: 0;
  font-size: 34rpx;
  font-weight: 700;
  color: $slate-400;
  margin-right: 24rpx;
}

.search-input {
  flex: 1;
  font-size: 28rpx;
  color: $slate-800;
}

// ===== 分类标签 =====
.category-scroll {
  margin-bottom: 32rpx;
  white-space: nowrap;
}

.category-row {
  display: flex;
  gap: 16rpx;
  flex-wrap: nowrap;
}

.category-pill {
  flex-shrink: 0;
  padding: 14rpx 40rpx;
  border-radius: 999rpx;
  border: 2rpx solid #E2E8F0;
  background: #fff;
  font-size: 28rpx;
  color: $slate-500;
  transition: all 0.2s;
}

.category-pill.active {
  font-weight: 600;
}

// ===== 状态 =====
.state-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 50vh;
}

.state-text {
  font-size: 28rpx;
  color: $slate-400;
}

.retry-btn {
  width: 180rpx;
  height: 72rpx;
  margin-top: 28rpx;
  border-radius: 999rpx;
  background: $blue-500;
  color: #fff;
  font-size: 26rpx;
}

// ===== 月份分组 =====
.month-group {
  margin-bottom: 32rpx;
}

.group-head {
  display: flex;
  align-items: center;
  gap: 16rpx;
  margin-bottom: 24rpx;
}

.group-month {
  font-size: 32rpx;
  font-weight: 700;
  color: $slate-800;
}

.group-count {
  font-size: 28rpx;
  color: $slate-400;
}

// ===== 卡片列表 =====
.card-list {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
}

.card {
  border-radius: 24rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  overflow: hidden;
}

.card-image {
  width: 100%;
  height: 320rpx;
}

.card-body {
  padding: 32rpx;
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
  gap: 16rpx;
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
  background: $blue-50;
}

.author-avatar.is-student {
  background: #FEF3C7;
}

.avatar-text {
  font-size: 26rpx;
  font-weight: 700;
  color: $blue-500;
}

.author-avatar.is-student .avatar-text {
  color: #D97706;
}

.author-name {
  font-size: 30rpx;
  font-weight: 600;
  color: $slate-800;
}

.cat-tag {
  flex-shrink: 0;
  padding: 4rpx 16rpx;
  border-radius: 8rpx;
  font-size: 24rpx;
  font-weight: 500;
}

.card-title {
  display: block;
  margin-top: 8rpx;
  font-size: 30rpx;
  font-weight: 700;
  color: $slate-800;
}

.card-subtitle {
  display: block;
  margin-top: 8rpx;
  font-size: 26rpx;
  color: $slate-500;
}

.card-date {
  flex-shrink: 0;
  font-size: 26rpx;
  color: $slate-400;
  margin-top: 2rpx;
}

.card-content {
  display: block;
  margin-top: 16rpx;
  font-size: 28rpx;
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

.student-avatars {
  display: flex;
}

.mini-avatar {
  width: 50rpx;
  height: 50rpx;
  border: 4rpx solid #fff;
  border-radius: 999rpx;
  background: #F1F5F9;

  &:not(:first-child) {
    margin-left: -18rpx;
  }
}

.student-count {
  margin-left: 16rpx;
  font-size: 24rpx;
  color: $slate-500;
}

.status-tag {
  padding: 8rpx 24rpx;
  border-radius: 999rpx;
  background: #FEF3C7;
  font-size: 24rpx;
  font-weight: 500;
  color: #D97706;

  &.synced {
    background: #F0FDF4;
    color: #16A34A;
  }
}

.card-stats {
  display: flex;
  gap: 32rpx;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.stat-icon {
  font-size: 28rpx;
  font-weight: 700;
  color: $slate-400;
}

.stat-num {
  font-size: 24rpx;
  color: $slate-500;
}

.bottom-safe {
  height: calc(120rpx + env(safe-area-inset-bottom));
}

// ===== 悬浮添加按钮 =====
.fab-btn {
  position: fixed;
  right: 48rpx;
  bottom: calc(64rpx + env(safe-area-inset-bottom));
  width: 112rpx;
  height: 112rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: $blue-500;
  box-shadow: 0 8rpx 24rpx rgba(59, 130, 246, 0.4);
  z-index: 100;

  &:active {
    opacity: 0.85;
    transform: scale(0.95);
  }
}

.fab-icon {
  font-size: 52rpx;
  font-weight: 400;
  color: #fff;
  line-height: 1;
}
</style>
