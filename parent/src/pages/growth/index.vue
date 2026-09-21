<script setup>
import { computed, ref } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchGrowthFeed } from "@/api/parent-growth"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import BottomNav from "@/components/BottomNav.vue"

const feeds = ref([])
const loading = ref(false)
const loadError = ref("")
const activeCategory = ref("全部")

const categories = ["全部", "校园生活", "活动照片", "作品展示", "获奖记录"]

const childInfo = ref({
  name: "",
  class: "",
  age: "",
  avatar: "",
  stats: { records: 0, photos: 0, medals: 0, attendance: "" },
})

const categoryClassMap = {
  "全部": "cat-all",
  "校园生活": "cat-life",
  "活动照片": "cat-event",
  "作品展示": "cat-work",
  "获奖记录": "cat-award",
}

function selectCategory(cat) { activeCategory.value = cat }

function goDetail(id) { uni.navigateTo({ url: `/pages/growth/detail?id=${id}` }) }
function goAlbum() { uni.navigateTo({ url: "/pages/growth/album" }) }
function goReport() { uni.navigateTo({ url: "/pages/growth/report" }) }

function toggleLike(item) {
  item.isLiked = !item.isLiked
  item.likes += item.isLiked ? 1 : -1
}

async function loadData() {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const sid = getCurrentStudentId()
    const [overviewRes, feedRes] = await Promise.all([
      fetchParentHomeOverview(sid),
      fetchGrowthFeed({ page: 1, page_size: 20, student_id: sid || undefined }),
    ])
    // 与首页同样承担自愈：存储为空时把服务端认定的当前孩子落盘
    syncCurrentStudentFromOverview(overviewRes.data)
    const stu = overviewRes.data?.current_student
    if (stu) {
      childInfo.value = {
        name: stu.name || "",
        class: stu.class_name || "",
        age: "",
        avatar: stu.avatar || "",
        stats: childInfo.value.stats,
      }
    }
    feeds.value = feedRes.data?.list || []
  } catch (err) {
    feeds.value = []
    uni.showToast({ title: "加载成长时光失败", icon: "none" })
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

// tabBar 页必须用 onShow：switchTab 切回来不重跑 onLoad，切完孩子会看不到变化
onShow(loadData)
onPullDownRefresh(loadData)
</script>

<template>
  <view class="page">
    <scroll-view scroll-y class="content-scroll" :refresher-enabled="true" @refresherrefresh="loadData">
      <!-- 孩子信息卡片 -->
      <view class="child-card">
        <view class="child-top">
          <view class="child-left">
            <image
              v-if="childInfo.avatar"
              :src="childInfo.avatar"
              class="child-avatar"
              mode="aspectFill"
            />
            <view v-else class="child-avatar-placeholder">
              <text>王</text>
            </view>
            <view class="child-meta">
              <text class="child-name">{{ childInfo.name }}</text>
              <text class="child-sub">{{ childInfo.class }} · {{ childInfo.age }}</text>
            </view>
          </view>
          <view class="export-btn">
            <text>导出相册</text>
          </view>
        </view>
        <view class="child-stats">
          <view class="stat-item">
            <text class="stat-num">{{ childInfo.stats.records }}</text>
            <text class="stat-label">成长记录</text>
          </view>
          <view class="stat-item">
            <text class="stat-num">{{ childInfo.stats.photos }}</text>
            <text class="stat-label">精彩照片</text>
          </view>
          <view class="stat-item">
            <text class="stat-num">{{ childInfo.stats.medals }}</text>
            <text class="stat-label">获得奖章</text>
          </view>
          <view class="stat-item">
            <text class="stat-num">{{ childInfo.stats.attendance }}</text>
            <text class="stat-label">出勤率</text>
          </view>
        </view>
      </view>

      <!-- 时光轴筛选栏 -->
      <view class="timeline-bar">
        <view class="tl-head">
          <view class="tl-title-row">
            <text class="tl-title">时光轴</text>
            <view class="tl-year-badge">
              <text>2026年</text>
            </view>
          </view>
          <view class="tl-filter">
            <text class="tl-filter-text">筛选</text>
            <text class="tl-filter-icon">&#xed25;</text>
          </view>
        </view>
        <scroll-view scroll-x class="tl-cat-scroll" :show-scrollbar="false">
          <view class="tl-cat-row">
            <view
              v-for="cat in categories"
              :key="cat"
              class="tl-cat-pill"
              :class="[categoryClassMap[cat], { active: activeCategory === cat }]"
              @click="selectCategory(cat)"
            >
              <text>{{ cat }}</text>
            </view>
          </view>
        </scroll-view>
      </view>

      <!-- 成长记录列表 -->
      <view v-if="loading" class="state-wrap">
        <text class="state-text">加载中...</text>
      </view>
      <view v-else-if="loadError" class="state-wrap">
        <text class="state-text">{{ loadError }}</text>
      </view>
      <view v-else class="record-list">
        <template v-for="(item, index) in feeds" :key="item.id">
          <!-- 月份分隔 -->
          <view v-if="item.month" class="month-marker">
            <view class="month-dot" />
            <text class="month-text">{{ item.month }}</text>
          </view>

          <!-- 记录卡片 -->
          <view class="record-card" @click="goDetail(item.id)">
            <!-- 卡片头部 -->
            <view class="rc-head">
              <view class="rc-teacher-avatar" :style="{ backgroundColor: item.teacherAvatarBg }">
                <text class="rc-ta-text" :style="{ color: item.teacherAvatarColor }">&#xf256;</text>
              </view>
              <view class="rc-teacher-meta">
                <text class="rc-teacher-name">{{ item.teacher }}</text>
                <text class="rc-time">{{ item.time }} · {{ item.category }}</text>
              </view>
              <view v-if="item.badge" class="rc-badge" :style="{ backgroundColor: item.badge.bg }">
                <text class="rc-badge-icon" :style="{ color: item.badge.color }" v-html="item.badge.icon" />
                <text class="rc-badge-text" :style="{ color: item.badge.color }">{{ item.badge.text }}</text>
              </view>
            </view>

            <!-- 内容 -->
            <text class="rc-content">{{ item.content }}</text>

            <!-- 照片网格 -->
            <view v-if="item.photos > 0" class="rc-photos">
              <view
                v-for="p in item.photos"
                :key="p"
                class="rc-photo-box"
                :class="{ 'rc-photo-full': item.photos === 1 }"
              >
                <view class="rc-photo-placeholder" />
              </view>
            </view>

            <!-- 互动栏 -->
            <view class="rc-actions">
              <view class="rca-left">
                <view class="rca-item" @click.stop="toggleLike(item)">
                  <text class="rca-icon" :class="{ liked: item.isLiked }" :style="{ color: item.isLiked ? '#EF4444' : '#94A3B8' }">
                    {{ item.isLiked ? '&#xee0f;' : '&#xee0e;' }}
                  </text>
                  <text class="rca-num" :class="{ liked: item.isLiked }">
                    {{ item.isLiked ? '已点赞' : '点赞' }}
                  </text>
                </view>
                <view class="rca-item">
                  <text class="rca-icon">&#xec3a;</text>
                  <text class="rca-num">{{ item.comments > 0 ? item.comments + '条评论' : '评论' }}</text>
                </view>
              </view>
              <view class="rca-item">
                <text class="rca-icon">&#xf37d;</text>
                <text class="rca-num">分享</text>
              </view>
            </view>
          </view>
        </template>
      </view>

      <view class="bottom-safe" />
    </scroll-view>

    <!-- 底部导航 -->
    <BottomNav :active="1" />
  </view>
</template>

<style scoped lang="scss">
$orange-100: #FFEDD5;
$slate-200: #E2E8F0;
$slate-600: #475569;

.page {
  min-height: 100vh;
  background: $color-bg-page;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow-x: hidden;
}

// ===== 内容区 =====
.content-scroll {
  flex: 1;
  padding: 0 28rpx;
  height: 0;
  box-sizing: border-box;
  width: 100%;
}

// ===== 孩子信息卡片 =====
.child-card {
  background: linear-gradient(135deg, $color-brand-primary-light, $color-brand-primary);
  border-radius: $border-radius-xl;
  padding: 48rpx;
  margin-bottom: 24rpx;
  margin-top: 24rpx;
}

.child-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.child-left {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
}

.child-avatar,
.child-avatar-placeholder {
  width: 160rpx;
  height: 160rpx;
  border-radius: 9999rpx;
  border: 8rpx solid rgba(255,255,255,0.3);
}

.child-avatar-placeholder {
  background: rgba(255,255,255,0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  text {
    font-size: 64rpx;
    font-weight: 700;
    color: #fff;
  }
}

.child-meta {
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
}

.child-name {
  font-size: 48rpx;
  font-weight: 800;
  color: #fff;
}

.child-sub {
  font-size: $typography-font-size-md;
  color: $orange-100;
}

.export-btn {
  padding: $spacing-sm 24rpx;
  border-radius: $border-radius-lg;
  background: rgba(255,255,255,0.2);
  text {
    font-size: $typography-font-size-base;
    font-weight: 500;
    color: #fff;
  }
}

.child-stats {
  display: flex;
  justify-content: space-between;
  margin-top: 32rpx;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
}

.stat-num {
  font-size: $typography-font-size-3xl;
  font-weight: 800;
  color: #fff;
}

.stat-label {
  font-size: $typography-font-size-sm;
  color: $orange-100;
}

// ===== 时光轴筛选 =====
.timeline-bar {
  margin-bottom: 16rpx;
}

.tl-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.tl-title-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.tl-title {
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  color: $color-text-primary;
}

.tl-year-badge {
  padding: 4rpx 16rpx;
  border-radius: 9999rpx;
  background: $color-brand-primary-bg;
  text {
    font-size: $typography-font-size-sm;
    font-weight: 500;
    color: $color-brand-primary;
  }
}

.tl-filter {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.tl-filter-text, .tl-filter-icon {
  font-size: $typography-font-size-md;
  color: $color-semantic-info;
}

.tl-cat-scroll {
  white-space: nowrap;
}

.tl-cat-row {
  display: flex;
  gap: $spacing-md;
}

.tl-cat-pill {
  flex-shrink: 0;
  padding: $spacing-md 32rpx;
  border-radius: 9999rpx;
  background: $color-bg-card;
  text {
    font-size: $typography-font-size-md;
    font-weight: 500;
    color: $color-text-muted;
  }
}

.tl-cat-pill.active {
  border: 2rpx solid $orange-100;
  background: $color-brand-primary-bg;
  text {
    color: $color-brand-primary;
  }
}

// ===== 状态 =====
.state-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 200rpx 0;
}

.state-text {
  font-size: $typography-font-size-md;
  color: $color-text-hint;
}

// ===== 记录列表 =====
.record-list {
  display: flex;
  flex-direction: column;
}

.month-marker {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  padding: $spacing-lg 0 20rpx;
}

.month-dot {
  width: 18rpx;
  height: 16rpx;
  border-radius: 9999rpx;
  background: $color-brand-primary;
}

.month-text {
  font-size: $typography-font-size-xl;
  font-weight: 600;
  color: $color-text-primary;
}

.record-card {
  background: #fff;
  border-radius: $border-radius-xl;
  padding: $spacing-xl;
  box-shadow: 0 2rpx 8rpx rgba(0,0,0,0.05);
  margin-bottom: 24rpx;
}

// ===== 记录卡片头部 =====
.rc-head {
  display: flex;
  align-items: flex-start;
}

.rc-teacher-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 9999rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.rc-ta-text {
  font-size: $typography-font-size-3xl;
}

.rc-teacher-meta {
  flex: 1;
  margin-left: 20rpx;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.rc-teacher-name {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
}

.rc-time {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.rc-badge {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  padding: 4rpx 16rpx;
  border-radius: $border-radius-sm;
  flex-shrink: 0;
  margin-top: 4rpx;
}

.rc-badge-icon {
  font-size: $typography-font-size-md;
}

.rc-badge-text {
  font-size: $typography-font-size-sm;
  font-weight: 500;
}

// ===== 内容 =====
.rc-content {
  display: block;
  margin-top: 24rpx;
  font-size: $typography-font-size-md;
  color: $slate-600;
  line-height: 1.65;
}

// ===== 照片 =====
.rc-photos {
  display: flex;
  gap: $spacing-lg;
  margin-top: 24rpx;
}

.rc-photo-box {
  width: 200rpx;
  height: 200rpx;
  border-radius: $border-radius-xl;
  overflow: hidden;
  flex-shrink: 0;
}

.rc-photo-full {
  width: 200rpx;
}

.rc-photo-placeholder {
  width: 100%;
  height: 100%;
  background: $color-bg-card;
}

// ===== 互动栏 =====
.rc-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 24rpx;
}

.rca-left {
  display: flex;
  align-items: center;
  gap: $spacing-xl;
}

.rca-item {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.rca-icon {
  font-size: $typography-font-size-2xl;
  color: $color-text-hint;
}

.rca-num {
  font-size: $typography-font-size-base;
  color: $color-text-hint;
}

.rca-icon.liked, .rca-num.liked {
  color: $color-semantic-danger;
}

.bottom-safe {
  height: calc(120rpx + env(safe-area-inset-bottom));
}
</style>
