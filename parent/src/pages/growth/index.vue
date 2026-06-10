<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchGrowthFeed } from "@/api/parent-growth"
import BottomNav from "@/components/BottomNav.vue"

const feeds = ref([])
const loading = ref(false)
const loadError = ref("")
const activeCategory = ref("全部")

const categories = ["全部", "校园生活", "活动照片", "作品展示", "获奖记录"]

// Mock child info stats (from design)
const childInfo = ref({
  name: "王小明",
  class: "小一班",
  age: "4岁半",
  avatar: "",
  stats: { records: 236, photos: 1286, medals: 18, attendance: "98%" },
})

// Mock timeline data matching Calicat design
const mockRecords = ref([
  {
    id: 1, month: "2026年3月",
    teacher: "李老师", teacherAvatarBg: "#DBEAFE", teacherAvatarColor: "#3B82F6",
    time: "3月19日 11:30", category: "美术课",
    badge: { text: "小小艺术家", bg: "#FEF9C3", color: "#D97706", icon: "&#xea8a;" },
    content: "今天的美术课上，小明发挥了自己的想象力，画出了自己心中的春天，色彩搭配非常棒，还主动帮助其他小朋友，表现特别优秀！",
    photos: 2, isLiked: true, likes: 28, comments: 2,
  },
  {
    id: 2, month: "",
    teacher: "张老师", teacherAvatarBg: "#DCFCE7", teacherAvatarColor: "#22C55E",
    time: "3月18日 15:30", category: "户外活动",
    badge: null,
    content: "今天户外活动玩接力赛跑，小明跑得特别快，还主动为其他小朋友加油，团队意识很强哦~",
    photos: 1, isLiked: false, likes: 15, comments: 0,
  },
  {
    id: 3, month: "2026年2月",
    teacher: "王老师", teacherAvatarBg: "#F3E8FF", teacherAvatarColor: "#A855F7",
    time: "2月28日 10:00", category: "开学典礼",
    badge: { text: "全勤小明星", bg: "#DBEAFE", color: "#2563EB", icon: "&#xf1c6;" },
    content: "恭喜小明在上个月获得全勤小明星称号，整个2月份没有请假，每天都准时到园，特别棒！继续保持哦~",
    photos: 3, isLiked: true, likes: 36, comments: 5,
  },
])

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
    const res = await fetchGrowthFeed({ page: 1, page_size: 20 })
    const list = res.data?.list || []
    if (list.length > 0) {
      feeds.value = list
    } else {
      feeds.value = mockRecords.value
    }
  } catch (err) {
    feeds.value = mockRecords.value
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

onLoad(loadData)
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
$orange-500: #F97316;
$orange-400: #FB923C;
$orange-50: #FFF7ED;
$orange-100: #FFEDD5;
$slate-50: #F8FAFC;
$slate-100: #F1F5F9;
$slate-200: #E2E8F0;
$slate-400: #94A3B8;
$slate-500: #64748B;
$slate-600: #475569;
$slate-800: #1E293B;
$red-500: #EF4444;
$blue-500: #3B82F6;

.page {
  min-height: 100vh;
  background: $slate-50;
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
  background: linear-gradient(135deg, $orange-400, $orange-500);
  border-radius: 32rpx;
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
  gap: 24rpx;
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
  gap: 8rpx;
}

.child-name {
  font-size: 48rpx;
  font-weight: 800;
  color: #fff;
}

.child-sub {
  font-size: 28rpx;
  color: $orange-100;
}

.export-btn {
  padding: 12rpx 24rpx;
  border-radius: 16rpx;
  background: rgba(255,255,255,0.2);
  text {
    font-size: 26rpx;
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
  font-size: 40rpx;
  font-weight: 800;
  color: #fff;
}

.stat-label {
  font-size: 24rpx;
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
  gap: 16rpx;
}

.tl-title {
  font-size: 36rpx;
  font-weight: 600;
  color: $slate-800;
}

.tl-year-badge {
  padding: 4rpx 16rpx;
  border-radius: 9999rpx;
  background: $orange-50;
  text {
    font-size: 24rpx;
    font-weight: 500;
    color: $orange-500;
  }
}

.tl-filter {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.tl-filter-text, .tl-filter-icon {
  font-size: 28rpx;
  color: $blue-500;
}

.tl-cat-scroll {
  white-space: nowrap;
}

.tl-cat-row {
  display: flex;
  gap: 16rpx;
}

.tl-cat-pill {
  flex-shrink: 0;
  padding: 16rpx 32rpx;
  border-radius: 9999rpx;
  background: $slate-100;
  text {
    font-size: 28rpx;
    font-weight: 500;
    color: $slate-500;
  }
}

.tl-cat-pill.active {
  border: 2rpx solid $orange-100;
  background: $orange-50;
  text {
    color: $orange-500;
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
  font-size: 28rpx;
  color: $slate-400;
}

// ===== 记录列表 =====
.record-list {
  display: flex;
  flex-direction: column;
}

.month-marker {
  display: flex;
  align-items: center;
  gap: 16rpx;
  padding: 24rpx 0 20rpx;
}

.month-dot {
  width: 18rpx;
  height: 16rpx;
  border-radius: 9999rpx;
  background: $orange-500;
}

.month-text {
  font-size: 32rpx;
  font-weight: 600;
  color: $slate-800;
}

.record-card {
  background: #fff;
  border-radius: 24rpx;
  padding: 32rpx;
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
  font-size: 40rpx;
}

.rc-teacher-meta {
  flex: 1;
  margin-left: 20rpx;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.rc-teacher-name {
  font-size: 30rpx;
  font-weight: 600;
  color: $slate-800;
}

.rc-time {
  font-size: 24rpx;
  color: $slate-400;
}

.rc-badge {
  display: flex;
  align-items: center;
  gap: 8rpx;
  padding: 4rpx 16rpx;
  border-radius: 8rpx;
  flex-shrink: 0;
  margin-top: 4rpx;
}

.rc-badge-icon {
  font-size: 28rpx;
}

.rc-badge-text {
  font-size: 24rpx;
  font-weight: 500;
}

// ===== 内容 =====
.rc-content {
  display: block;
  margin-top: 24rpx;
  font-size: 28rpx;
  color: $slate-600;
  line-height: 1.65;
}

// ===== 照片 =====
.rc-photos {
  display: flex;
  gap: 24rpx;
  margin-top: 24rpx;
}

.rc-photo-box {
  width: 200rpx;
  height: 200rpx;
  border-radius: 24rpx;
  overflow: hidden;
  flex-shrink: 0;
}

.rc-photo-full {
  width: 200rpx;
}

.rc-photo-placeholder {
  width: 100%;
  height: 100%;
  background: $slate-100;
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
  gap: 32rpx;
}

.rca-item {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.rca-icon {
  font-size: 36rpx;
  color: $slate-400;
}

.rca-num {
  font-size: 26rpx;
  color: $slate-400;
}

.rca-icon.liked, .rca-num.liked {
  color: $red-500;
}

.bottom-safe {
  height: calc(120rpx + env(safe-area-inset-bottom));
}
</style>
