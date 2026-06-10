<script setup>
import { ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchTasks } from "@/api/parent-task"
import BottomNav from "@/components/BottomNav.vue"

const tasks = ref([])
const loading = ref(false)
const activeTab = ref("进行中")

const tabs = ["进行中", "已打卡", "全部"]

// Mock points data
const points = ref({ total: 520, earned: 1280, monthEarned: 85, rank: 12, streak: 7 })

// Mock badge wall
const badges = ref([
  { name: "全勤达人", iconColor: "#F59E0B", bgColor: "#FEF3C7", locked: false },
  { name: "阅读之星", iconColor: "#3B82F6", bgColor: "#DBEAFE", locked: false },
  { name: "运动健将", iconColor: "#10B981", bgColor: "#D1FAE5", locked: false },
  { name: "小小画家", iconColor: "#EC4899", bgColor: "#FCE7F3", locked: false },
  { name: "劳动能手", iconColor: "#8B5CF6", bgColor: "#EDE9FE", locked: false },
  { name: "爱心天使", iconColor: "#EF4444", bgColor: "#FEE2E2", locked: false },
  { name: "???", iconColor: "#94A3B8", bgColor: "#F1F5F9", locked: true },
  { name: "???", iconColor: "#94A3B8", bgColor: "#F1F5F9", locked: true },
])

// Mock task data
const mockTasks = ref([
  {
    id: 1, title: "亲子阅读打卡", points: 5, iconBg: "#DBEAFE", iconColor: "#3B82F6",
    desc: "每天陪伴孩子阅读15分钟，培养孩子良好的阅读习惯",
    progress: 0.75, progressLabel: "已打卡15/20天", infoLabel: "每完成1次获得5积分",
    status: "doing",
  },
  {
    id: 2, title: "家务小能手", points: 3, iconBg: "#D1FAE5", iconColor: "#10B981",
    desc: "鼓励孩子参与力所能及的家务劳动，培养独立能力",
    progress: 0.3, progressLabel: "已打卡3/10次", infoLabel: "每完成1次获得3积分",
    status: "doing",
  },
  {
    id: 3, title: "户外运动挑战", points: 10, iconBg: "#FEF3C7", iconColor: "#F59E0B",
    desc: "每天户外运动30分钟，强健体魄，释放活力，记录运动瞬间",
    progress: 0.6, progressLabel: "已打卡12/20天", infoLabel: "每完成1次获得10积分",
    status: "doing",
  },
])

function goDetail(id) { uni.navigateTo({ url: `/pages/tasks/detail?id=${id}` }) }
function goCheckIn(id) { uni.navigateTo({ url: `/pages/tasks/detail?id=${id}` }) }
function goRule() { uni.showToast({ title: "积分规则", icon: "none" }) }
function goExchange() { uni.showToast({ title: "兑换奖励", icon: "none" }) }
function goBadges() { uni.navigateTo({ url: "/pages/points/index" }) }

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchTasks()
    const list = res.data?.list || []
    tasks.value = list.length > 0 ? list : mockTasks.value
  } catch (err) {
    tasks.value = mockTasks.value
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
      <!-- 积分卡片 -->
      <view class="points-card">
        <view class="pc-top">
          <view class="pc-left">
            <text class="pc-label">我的积分</text>
            <view class="pc-num-row">
              <text class="pc-star">&#xea8a;</text>
              <text class="pc-num">{{ points.total }}</text>
            </view>
          </view>
          <view class="pc-exchange" @click="goExchange">
            <text>兑换奖励</text>
          </view>
        </view>
        <view class="pc-stats">
          <view class="pcs-item">
            <text class="pcs-num">{{ points.earned }}</text>
            <text class="pcs-label">累计获得</text>
          </view>
          <view class="pcs-item">
            <text class="pcs-num">{{ points.monthEarned }}</text>
            <text class="pcs-label">本月获得</text>
          </view>
          <view class="pcs-item">
            <text class="pcs-num">{{ points.rank }}</text>
            <text class="pcs-label">排名</text>
          </view>
          <view class="pcs-item">
            <text class="pcs-num">{{ points.streak }}天</text>
            <text class="pcs-label">连续打卡</text>
          </view>
        </view>
      </view>

      <!-- 任务分类栏 -->
      <view class="tab-bar">
        <view
          v-for="tab in tabs"
          :key="tab"
          class="tab-item"
          :class="{ active: activeTab === tab }"
          @click="activeTab = tab"
        >
          <text>{{ tab }}</text>
        </view>
      </view>

      <!-- 任务列表 -->
      <view v-if="loading" class="state-wrap">
        <text class="state-text">加载中...</text>
      </view>
      <view v-else-if="tasks.length === 0" class="state-wrap">
        <text class="state-text">暂无任务</text>
      </view>
      <view v-else class="task-list">
        <view v-for="item in tasks" :key="item.id" class="task-card">
          <!-- 任务头部 -->
          <view class="tc-head">
            <view class="tc-icon-box" :style="{ backgroundColor: item.iconBg }">
              <text class="tc-icon" :style="{ color: item.iconColor }">&#xf1e8;</text>
            </view>
            <view class="tc-head-mid">
              <text class="tc-title">{{ item.title }}</text>
              <text class="tc-desc">{{ item.desc }}</text>
            </view>
            <view class="tc-points-badge">
              <text>+{{ item.points }}</text>
            </view>
          </view>

          <!-- 进度条 -->
          <view class="tc-progress">
            <view class="tcp-bg">
              <view class="tcp-fill" :style="{ width: (item.progress * 100) + '%' }" />
            </view>
            <text class="tcp-label">{{ item.progressLabel }}</text>
          </view>

          <!-- 积分说明 -->
          <view class="tc-info">
            <text class="tci-icon">&#xea8a;</text>
            <text class="tci-text">{{ item.infoLabel }}</text>
          </view>

          <!-- 操作按钮 -->
          <view class="tc-actions">
            <view class="tca-checkin" @click="goCheckIn(item.id)">
              <text>去打卡</text>
            </view>
            <view class="tca-detail" @click="goDetail(item.id)">
              <text>查看详情</text>
            </view>
          </view>
        </view>
      </view>

      <!-- 徽章墙 -->
      <view class="badge-section">
        <view class="bs-head">
          <text class="bs-title">徽章墙</text>
          <text class="bs-more" @click="goBadges">查看全部</text>
        </view>
        <view class="bs-grid">
          <view
            v-for="(b, i) in badges"
            :key="i"
            class="bs-item"
            :class="{ locked: b.locked }"
          >
            <view class="bs-icon-box" :style="{ backgroundColor: b.bgColor }">
              <text class="bs-icon" :style="{ color: b.iconColor }">&#xea8a;</text>
            </view>
            <text class="bs-name">{{ b.name }}</text>
          </view>
        </view>
      </view>

      <view class="bottom-safe" />
    </scroll-view>

    <BottomNav :active="2" />
  </view>
</template>

<style scoped lang="scss">
$orange-500: #F97316;
$orange-400: #FB923C;
$orange-100: #FFEDD5;
$slate-50: #F8FAFC;
$slate-100: #F1F5F9;
$slate-200: #E2E8F0;
$slate-400: #94A3B8;
$slate-500: #64748B;
$slate-600: #475569;
$slate-800: #1E293B;
$purple-500: #A855F7;
$purple-50: #F3E8FF;
$purple-100: #EDE9FE;

.page {
  min-height: 100vh;
  background: $slate-50;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow-x: hidden;
}

.content-scroll {
  flex: 1;
  padding: 24rpx 28rpx 0;
  height: 0;
  box-sizing: border-box;
  width: 100%;
}

// ===== 积分卡片 =====
.points-card {
  background: linear-gradient(135deg, $orange-400, $orange-500);
  border-radius: 32rpx;
  padding: 48rpx;
  margin-bottom: 24rpx;
}

.pc-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.pc-label {
  font-size: 28rpx;
  color: $orange-100;
}

.pc-num-row {
  display: flex;
  align-items: center;
  gap: 16rpx;
  margin-top: 8rpx;
}

.pc-star {
  font-size: 72rpx;
  color: #fff;
}

.pc-num {
  font-size: 88rpx;
  font-weight: 800;
  color: #fff;
}

.pc-exchange {
  padding: 14rpx 28rpx;
  border-radius: 16rpx;
  background: rgba(255,255,255,0.2);
  text {
    font-size: 28rpx;
    font-weight: 500;
    color: #fff;
  }
}

.pc-stats {
  display: flex;
  justify-content: space-between;
  margin-top: 32rpx;
}

.pcs-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
}

.pcs-num {
  font-size: 40rpx;
  font-weight: 800;
  color: #fff;
}

.pcs-label {
  font-size: 24rpx;
  color: $orange-100;
}

// ===== 分类标签 =====
.tab-bar {
  display: flex;
  gap: 24rpx;
  margin-bottom: 24rpx;
}

.tab-item {
  flex: 1;
  padding: 24rpx 0;
  border-radius: 20rpx;
  background: #fff;
  text-align: center;
  text {
    font-size: 28rpx;
    font-weight: 500;
    color: $slate-500;
  }
}

.tab-item.active {
  background: $purple-50;
  text {
    color: $purple-500;
    font-weight: 600;
  }
}

// ===== 状态 =====
.state-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 200rpx 0;
}
.state-text { font-size: 28rpx; color: $slate-400; }

// ===== 任务卡片 =====
.task-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.task-card {
  background: #fff;
  border-radius: 24rpx;
  padding: 32rpx;
  box-shadow: 0 2rpx 8rpx rgba(0,0,0,0.05);
}

.tc-head {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
}

.tc-icon-box {
  width: 96rpx;
  height: 96rpx;
  border-radius: 24rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tc-icon {
  font-size: 52rpx;
}

.tc-head-mid {
  flex: 1;
  overflow: hidden;
}

.tc-title {
  display: block;
  font-size: 32rpx;
  font-weight: 700;
  color: $slate-800;
}

.tc-desc {
  display: block;
  font-size: 26rpx;
  color: $slate-400;
  margin-top: 8rpx;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.tc-points-badge {
  padding: 6rpx 20rpx;
  border-radius: 12rpx;
  background: $purple-50;
  flex-shrink: 0;
  text {
    font-size: 36rpx;
    font-weight: 800;
    color: $purple-500;
  }
}

// ===== 进度条 =====
.tc-progress {
  display: flex;
  align-items: center;
  gap: 16rpx;
  margin-top: 24rpx;
}

.tcp-bg {
  flex: 1;
  height: 16rpx;
  border-radius: 9999rpx;
  background: $slate-100;
  overflow: hidden;
}

.tcp-fill {
  height: 100%;
  border-radius: 9999rpx;
  background: $purple-500;
}

.tcp-label {
  font-size: 24rpx;
  color: $slate-400;
  flex-shrink: 0;
}

// ===== 积分说明 =====
.tc-info {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-top: 16rpx;
}

.tci-icon {
  font-size: 28rpx;
  color: $slate-400;
}

.tci-text {
  font-size: 24rpx;
  color: $slate-500;
}

// ===== 操作按钮 =====
.tc-actions {
  display: flex;
  gap: 24rpx;
  margin-top: 24rpx;
}

.tca-checkin {
  flex: 1;
  padding: 24rpx 0;
  border-radius: 20rpx;
  background: $purple-500;
  text-align: center;
  text {
    font-size: 28rpx;
    font-weight: 600;
    color: #fff;
  }
}

.tca-detail {
  flex: 1;
  padding: 24rpx 0;
  border-radius: 20rpx;
  background: $purple-50;
  text-align: center;
  text {
    font-size: 28rpx;
    font-weight: 500;
    color: $purple-500;
  }
}

// ===== 徽章墙 =====
.badge-section {
  margin-top: 48rpx;
}

.bs-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.bs-title {
  font-size: 36rpx;
  font-weight: 600;
  color: $slate-800;
}

.bs-more {
  font-size: 28rpx;
  color: $purple-500;
}

.bs-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 20rpx;
}

.bs-item {
  width: calc(25% - 15rpx);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12rpx;
}

.bs-icon-box {
  width: 128rpx;
  height: 128rpx;
  border-radius: 32rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.bs-icon {
  font-size: 72rpx;
}

.bs-name {
  font-size: 24rpx;
  color: $slate-500;
  text-align: center;
}

.bs-item.locked {
  .bs-icon-box { opacity: 0.5; }
  .bs-name { color: $slate-400; }
}

.bottom-safe {
  height: calc(120rpx + env(safe-area-inset-bottom));
}
</style>
