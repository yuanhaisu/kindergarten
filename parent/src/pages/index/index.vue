<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchParentHomeOverview, fetchHomeFeeds, fetchAnnouncements, switchStudent } from "@/api/parent-home"
import { ensureLogin } from "@/utils/login"
import { bindParentPhone } from "@/api/auth"
import BottomNav from "@/components/BottomNav.vue"

const needBindPhone = ref(false)
const bindingPhone = ref(false)
const bindPhoneErr = ref("")

const loading = ref(false)
const overview = ref({})
const feeds = ref([])
const loadError = ref("")

const currentStudent = computed(() => overview.value.current_student || {})
const students = computed(() => overview.value.students || [])
const summary = computed(() => overview.value.today_summary || {})
const relationLabel = computed(() => {
  const stu = students.value.find(s => s.id === currentStudent.value.id)
  if (!stu) return "家长"
  const map = { father: "爸爸", mother: "妈妈", grandfather: "爷爷", grandmother: "奶奶" }
  return map[stu.relation] || "家长"
})

/** 今日在校情况 4 状态 */
const todayStatuses = computed(() => [
  { key: "attendance", icon: "", label: summary.value.attendance_label || "--", sub: summary.value.arrive_time || "", color: "green" },
  { key: "meal", icon: "", label: summary.value.meal_label || "--", sub: "", color: "blue" },
  { key: "sleep", icon: "", label: summary.value.sleep_label || "--", sub: summary.value.nap_duration ? summary.value.nap_duration + "分钟" : "", color: "purple" },
  { key: "mood", icon: "", label: summary.value.mood_label || "--", sub: "", color: "orange" }
])

/** 4 功能入口 — 高频优先，不与Tab重复 */
const quickActions = [
  { title: "通知公告", icon: "", color: "blue", path: "/pages/notice/index", badge: summary.value.new_notice_count },
  { title: "接送管理", icon: "", color: "orange", path: "/pages/pickup/index" },
  { title: "亲子任务", icon: "", color: "purple", path: "/pages/tasks/index" },
  { title: "我要请假", icon: "", color: "yellow", path: "/pages/leave/create" },
]

async function loadHome() {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const [overviewRes, feedsRes] = await Promise.all([
      fetchParentHomeOverview(),
      fetchHomeFeeds({ page: 1, page_size: 5, student_id: currentStudent.value.id || undefined })
    ])
    overview.value = overviewRes.data || {}
    feeds.value = feedsRes.data?.list || []
    needBindPhone.value = false
  } catch (err) {
    const msg = err?.message || ""
    if (msg.includes("未绑定") || msg.includes("孩子信息") || msg.includes("联系管理员")) {
      needBindPhone.value = true
      loadError.value = ""
    } else {
      loadError.value = msg || "首页数据加载失败"
    }
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
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
  if (student.id === currentStudent.value.id) return
  try {
    await switchStudent(student.id)
    await loadHome()
  } catch (err) {
    uni.showToast({ title: err?.message || "切换失败", icon: "none" })
  }
}

const tabBarPaths = ["/pages/growth/index", "/pages/pickup/index", "/pages/index/index", "/pages/profile/index"]
function handleAction(action) {
  if (tabBarPaths.includes(action.path)) {
    uni.switchTab({ url: action.path })
  } else {
    uni.navigateTo({ url: action.path })
  }
}

function goNoticeDetail(id) { uni.navigateTo({ url: `/pages/notice/detail?id=${id}` }) }
function goGrowthDetail(id) { uni.navigateTo({ url: `/pages/growth/detail?id=${id}` }) }

function formatTime(ts) {
  if (!ts) return ""
  const d = new Date(ts)
  const now = new Date()
  const diff = now - d
  if (diff < 3600000) return Math.floor(diff / 60000) + "分钟前"
  if (diff < 86400000) return Math.floor(diff / 3600000) + "小时前"
  return ts.slice(0, 10)
}

function feedTypeLabel(type) {
  return type === "growth" ? "成长记录" : "通知"
}

onLoad(loadHome)
onPullDownRefresh(loadHome)
</script>

<template>
  <view class="page">
    <!-- ===== 顶部导航栏 (橙色渐变) ===== -->
    <view class="nav-header">
      <view class="nav-inner">
        <!-- 孩子信息 -->
        <view class="child-info">
          <image
            v-if="currentStudent.avatar"
            :src="currentStudent.avatar"
            class="child-avatar"
            mode="aspectFill"
          />
          <view v-else class="child-avatar-placeholder">
            <text class="placeholder-icon"></text>
          </view>
          <view class="spacer-w12" />
          <view class="child-meta">
            <view class="child-name-row">
              <text class="child-name">{{ currentStudent.name || "孩子" }}</text>
              <view class="spacer-w8" />
              <view class="class-tag">
                <text class="class-tag-text">{{ currentStudent.class_name || "" }}</text>
              </view>
            </view>
            <view class="spacer-h4" />
            <text class="greeting">{{ relationLabel }} 您好~</text>
          </view>
        </view>
        <view class="spacer-flex" />
        <!-- 设置按钮 -->
        <view class="settings-btn" @click="uni.navigateTo({ url: '/pages/settings/index' })">
          <text class="settings-icon"></text>
        </view>
      </view>
      <!-- 间距 -->
      <view class="nav-spacer" />
      <!-- ===== 今日在校情况卡片 ===== -->
      <view class="today-card">
        <view class="today-head">
          <text class="today-title">今日在校情况</text>
          <text class="today-date">2026年3月19日</text>
        </view>
        <view class="spacer-h16" />
        <!-- 4 状态 -->
        <view class="status-row">
          <view v-for="st in todayStatuses" :key="st.key" class="status-item">
            <view :class="['status-icon-box', 'bg-' + st.color]">
              <text :class="['status-icon', 'ic-' + st.color]">{{ st.icon }}</text>
            </view>
            <view class="spacer-h8" />
            <text class="status-label">{{ st.label }}</text>
            <text v-if="st.sub" class="status-sub">{{ st.sub }}</text>
          </view>
        </view>
        <!-- 今日表现 -->
        <view v-if="summary.performance_tag" class="spacer-h16" />
        <view v-if="summary.performance_tag" class="performance-bar">
          <text class="perf-icon"></text>
          <view class="spacer-w8" />
          <text class="perf-text">今日获得："{{ summary.performance_tag }}" 奖章</text>
        </view>
      </view>
    </view>

    <!-- ===== 手机号绑定引导 ===== -->
    <view v-if="needBindPhone" class="bind-guide">
      <view class="bind-card">
        <text class="bind-icon"></text>
        <text class="bind-title">绑定手机号以关联孩子信息</text>
        <text class="bind-desc">请输入或授权微信绑定的手机号，系统将自动匹配管理员为您创建的家长账号</text>
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

    <!-- ===== 多孩切换 ===== -->
    <view v-if="!needBindPhone && students.length > 1" class="student-switch">
      <view
        v-for="stu in students"
        :key="stu.id"
        class="switch-chip"
        :class="{ active: stu.id === currentStudent.id }"
        @click="handleSwitchStudent(stu)"
      >
        {{ stu.name }}
      </view>
    </view>

    <!-- ===== 常规内容区（仅非绑定状态） ===== -->
    <template v-if="!needBindPhone">

    <!-- ===== 功能入口 2×4 网格 ===== -->
    <view class="action-panel">
      <view class="action-grid">
        <view
          v-for="action in quickActions"
          :key="action.title"
          class="action-item"
          @click="handleAction(action)"
        >
          <view :class="['action-icon-box', 'bg-' + action.color]">
            <text :class="['action-icon', 'ic-' + action.color]">{{ action.icon }}</text>
            <view v-if="action.badge" class="action-badge" />
          </view>
          <view class="spacer-h8" />
          <text class="action-text">{{ action.title }}</text>
        </view>
      </view>
    </view>

    <!-- ===== 最新动态（通知 + 成长 混合时间线）===== -->
    <view class="section">
      <view class="section-head">
        <text class="section-title">最新动态</text>
      </view>
      <view class="spacer-h16" />
      <view v-if="feeds.length === 0" class="empty-card">
        <text class="empty-text">暂无最新动态</text>
      </view>
      <view v-else class="feed-list">
        <view
          v-for="feed in feeds"
          :key="feed.id"
          class="feed-card"
          @click="feed.type === 'notice' ? goNoticeDetail(feed.id) : goGrowthDetail(feed.id)"
        >
          <view :class="['feed-type-tag', feed.type === 'notice' ? 'tag-notice' : 'tag-growth']">
            <text>{{ feed.type === 'notice' ? '通知' : '成长' }}</text>
          </view>
          <template v-if="feed.type === 'notice'">
            <view class="feed-title-row">
              <text class="feed-title">{{ feed.notice?.title || "" }}</text>
              <view v-if="!feed.notice?.is_read" class="unread-dot" />
            </view>
            <text class="feed-preview">{{ feed.notice?.content_preview || "" }}</text>
          </template>
          <template v-else>
            <text class="feed-preview">{{ feed.growth?.content || "" }}</text>
          </template>
          <view class="feed-footer">
            <text class="feed-time">{{ formatTime(feed.created_at) }}</text>
          </view>
        </view>
      </view>
    </view>

    </template>

    <!-- 错误 / 加载中 -->
    <view v-if="loadError" class="error-card">
      <text>{{ loadError }}</text>
      <view class="retry-btn" @click="loadHome">重试</view>
    </view>
    <view v-if="loading" class="loading-hint">加载中...</view>

    <!-- 底部导航 -->
    <BottomNav :active="0" />
  </view>
</template>

<style scoped lang="scss">
// ============================================================
// 首页 — 对齐 Calicat 设计稿
// ============================================================

.page {
  min-height: 100vh;
  box-sizing: border-box;
  width: 100%;
  min-width: 320px;
  max-width: 430px;
  margin: 0 auto;
  background: $color-bg-page;
  padding-bottom: calc(120rpx + env(safe-area-inset-bottom));
}

// ---- helpers ----
.spacer-w4  { width: 8rpx;  flex-shrink: 0; }
.spacer-w8  { width: 16rpx; flex-shrink: 0; }
.spacer-w12 { width: 24rpx; flex-shrink: 0; }
.spacer-w24 { width: 48rpx; flex-shrink: 0; }
.spacer-h4  { height: 8rpx;  }
.spacer-h8  { height: 16rpx; }
.spacer-h12 { height: 24rpx; }
.spacer-h16 { height: 32rpx; }
.spacer-flex { flex: 1; }

// ---- 顶部导航栏 ----
.nav-header {
  background: linear-gradient(135deg, #FB923C, #F97316);
  padding: 48rpx 24rpx 24rpx;
  border-radius: 0 0 32rpx 32rpx;
}

.nav-inner {
  display: flex;
  align-items: center;
}

.child-info {
  display: flex;
  align-items: center;
}

.child-avatar {
  width: 128rpx;
  height: 128rpx;
  border-radius: 9999rpx;
  border: 4rpx solid rgba(255,255,255,0.4);
}

.child-avatar-placeholder {
  width: 128rpx;
  height: 128rpx;
  border-radius: 9999rpx;
  background: rgba(255,255,255,0.25);
  display: flex;
  align-items: center;
  justify-content: center;
}

.placeholder-icon {
  font-family: remixicon;
  font-size: 48rpx;
  color: rgba(255,255,255,0.7);
}

.child-meta {
  display: flex;
  flex-direction: column;
}

.child-name-row {
  display: flex;
  align-items: center;
}

.child-name {
  color: #FFFFFF;
  font-size: $typography-font-size-3xl;
  font-weight: 700;
}

.class-tag {
  padding: 4rpx 16rpx;
  border-radius: 999rpx;
  background: rgba(255,255,255,0.2);
}

.class-tag-text {
  color: #FFFFFF;
  font-size: $typography-font-size-sm;
}

.greeting {
  color: #FFEDD5;
  font-size: $typography-font-size-md;
}

.settings-btn {
  width: 80rpx;
  height: 80rpx;
  border-radius: 9999rpx;
  background: rgba(255,255,255,0.2);
  display: flex;
  align-items: center;
  justify-content: center;
}

.settings-icon {
  font-family: remixicon;
  font-size: 44rpx;
  color: #FFFFFF;
}

.nav-spacer {
  height: 48rpx;
}

// ---- 今日在校情况卡片 ----
.today-card {
  background: #FFFFFF;
  border-radius: $border-radius-lg;
  padding: $spacing-xl;
  box-shadow: $shadow-sm;
}

.today-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.today-title {
  font-size: $typography-font-size-xl;
  font-weight: 600;
  color: $color-text-primary;
}

.today-date {
  font-size: $typography-font-size-md;
  color: $color-text-muted;
}

.status-row {
  display: flex;
  justify-content: space-around;
}

.status-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 96rpx;
}

.status-icon-box {
  width: 96rpx;
  height: 96rpx;
  border-radius: 9999rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.status-icon {
  font-family: remixicon;
  font-size: 44rpx;
}

.status-label {
  font-size: $typography-font-size-base;
  font-weight: 500;
  color: $color-text-primary;
}

.status-sub {
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
  margin-top: 4rpx;
}

.performance-bar {
  display: flex;
  align-items: center;
  background: #FFF7ED;
  border-radius: $border-radius-md;
  padding: $spacing-lg;
}

.perf-icon {
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  color: #F97316;
}

.perf-text {
  font-size: $typography-font-size-base;
  color: #92400E;
  font-weight: 500;
}

// ---- 状态色 ----
.bg-green  { background: #DCFCE7; }
.bg-blue   { background: #DBEAFE; }
.bg-purple { background: #F3E8FF; }
.bg-orange { background: #FFEDD5; }
.bg-cyan   { background: #CFFAFE; }
.bg-yellow { background: #FEF9C3; }
.bg-pink   { background: #FCE7F3; }
.bg-gray   { background: #F1F5F9; }

.ic-green  { color: #16A34A; }
.ic-blue   { color: #2563EB; }
.ic-purple { color: #9333EA; }
.ic-orange { color: #EA580C; }
.ic-cyan   { color: #06B6D4; }
.ic-yellow { color: #CA8A04; }
.ic-pink   { color: #EC4899; }
.ic-gray   { color: #64748B; }

// ---- 多孩切换 ----
.student-switch {
  display: flex;
  gap: $spacing-md;
  padding: 0 24rpx;
  margin-top: 24rpx;
}

.switch-chip {
  padding: 14rpx 28rpx;
  border-radius: 9999rpx;
  background: $color-bg-card;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.switch-chip.active {
  background: $color-brand-primary-bg;
  color: $color-brand-primary;
  font-weight: 700;
}

// ---- 功能入口 2×4 网格 ----
.action-panel {
  padding: 0 24rpx;
  margin-top: 24rpx;
}

.action-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20rpx;
  padding: $spacing-xl;
  background: $color-bg-card;
  border-radius: $border-radius-lg;
  box-shadow: $shadow-sm;
}

.action-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
}

.action-icon-box {
  width: 112rpx;
  height: 112rpx;
  border-radius: $border-radius-xl;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.action-icon {
  font-family: remixicon;
  font-size: 52rpx;
}

.action-badge {
  position: absolute;
  top: 8rpx;
  right: 8rpx;
  width: 18rpx;
  height: 18rpx;
  border-radius: 9999rpx;
  background: #EF4444;
}

.action-text {
  font-size: $typography-font-size-base;
  font-weight: 500;
  color: $color-text-body;
}

// ---- 通用 section ----
.section {
  padding: 0 24rpx;
  margin-top: 36rpx;
}

.section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.section-title {
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  color: $color-text-primary;
}

.section-link {
  font-size: $typography-font-size-md;
  color: #2563EB;
}

// ---- 通知列表 ----
.notice-card {
  display: flex;
  background: $color-bg-card;
  border-radius: $border-radius-md;
  padding: $spacing-xl;
  margin-bottom: 16rpx;
  box-shadow: $shadow-sm;
}

// ---- 统一动态列表 ----
.feed-list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.feed-card {
  background: $color-bg-card;
  border-radius: $border-radius-md;
  padding: $spacing-xl;
  box-shadow: $shadow-sm;
}

.feed-type-tag {
  display: inline-block;
  padding: 4rpx 16rpx;
  border-radius: 999rpx;
  font-size: 20rpx;
  font-weight: 500;
  margin-bottom: 12rpx;
}
.feed-type-tag.tag-notice { background: #DBEAFE; color: #2563EB; }
.feed-type-tag.tag-growth { background: #DCFCE7; color: #16A34A; }

.feed-title-row {
  display: flex;
  align-items: center;
  margin-bottom: 8rpx;
}

.feed-title {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.unread-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 9999rpx;
  background: #EF4444;
  flex-shrink: 0;
  margin-left: 8rpx;
}

.feed-preview {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  margin-bottom: 8rpx;
}

.feed-footer {
  display: flex;
  justify-content: flex-end;
}

.feed-time {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

// ---- 成长动态（保留旧样式，供其他页引用） ----

.growth-author {
  display: flex;
  align-items: center;
}

.author-avatar-box {
  width: 80rpx;
  height: 80rpx;
  border-radius: 9999rpx;
  background: #DBEAFE;
  display: flex;
  align-items: center;
  justify-content: center;
}

.author-avatar-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  color: #2563EB;
}

.author-meta {
  display: flex;
  flex-direction: column;
}

.author-name-text {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
}

.author-time {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.growth-content {
  font-size: $typography-font-size-md;
  color: $color-text-body;
  line-height: 1.6;
}

.growth-photos {
  display: flex;
  gap: $spacing-lg;
}

.growth-photo {
  width: 200rpx;
  height: 200rpx;
  border-radius: $border-radius-md;
}

.growth-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.ga-left {
  display: flex;
  align-items: center;
}

.ga-item {
  display: flex;
  align-items: center;
}

.ga-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  color: $color-text-hint;
}

.ga-num {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
}

.ga-like-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  color: $color-text-hint;
}

.ga-like-icon.liked {
  color: #EF4444;
}

// ---- 空状态 / 错误 ----
.empty-card {
  padding: 48rpx 28rpx;
  border-radius: $border-radius-md;
  background: $color-bg-card;
  text-align: center;
  box-shadow: $shadow-sm;
}

.empty-text {
  color: $color-text-hint;
  font-size: $typography-font-size-base;
}

.error-card {
  padding: 38rpx 28rpx;
  margin: $spacing-lg;
  border-radius: $border-radius-md;
  background: $color-bg-card;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  text-align: center;
  box-shadow: $shadow-sm;
}

.retry-btn {
  width: 180rpx;
  height: 72rpx;
  margin: $spacing-lg auto 0;
  border-radius: 9999rpx;
  background: $color-brand-primary;
  color: #fff;
  font-size: $typography-font-size-base;
  line-height: 72rpx;
}

.loading-hint {
  text-align: center;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  padding: $spacing-xl;
}

// ---- 手机号绑定引导 ----
.bind-guide {
  padding: 48rpx 48rpx 0;
}

.bind-card {
  background: #fff;
  border-radius: $border-radius-xl;
  padding: 64rpx 48rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
}

.bind-icon {
  font-size: 96rpx;
  color: #94A3B8;
}

.bind-title {
  font-size: 34rpx;
  font-weight: 700;
  color: #1E293B;
  margin-top: 32rpx;
}

.bind-desc {
  font-size: $typography-font-size-base;
  color: #94A3B8;
  text-align: center;
  line-height: 1.6;
  margin-top: 16rpx;
}

.bind-phone-btn {
  margin-top: 48rpx;
  width: 100%;
  height: 96rpx;
  line-height: 96rpx;
  background: linear-gradient(135deg, #FB923C, #F97316);
  color: #fff;
  font-size: $typography-font-size-xl;
  font-weight: 600;
  border-radius: 48rpx;
  border: none;
  text-align: center;
  padding: 0;
}

.bind-phone-btn[disabled] {
  opacity: 0.6;
}

.bind-err {
  margin-top: 24rpx;
  font-size: $typography-font-size-sm;
  color: #EF4444;
  text-align: center;
}
</style>
