<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { approveTeacherLeave, fetchTeacherLeaveStats, fetchTeacherLeaves, rejectTeacherLeave } from "@/api/teacher-leaves"

const loading = ref(false)
const stats = ref({})
const leaves = ref([])
const activeTab = ref("pending")

async function loadLeaves() {
  loading.value = true
  try {
    const [statsRes, listRes] = await Promise.all([
      fetchTeacherLeaveStats(),
      fetchTeacherLeaves({ page: 1, page_size: 20 })
    ])
    stats.value = statsRes.data || {}
    const list = listRes.data?.list || []
    leaves.value = list
  } catch (err) {
    leaves.value = []
    uni.showToast({ title: "加载请假列表失败", icon: "none" })
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

const pendingLeaves = computed(() => leaves.value.filter(l => Number(l.status) === 0))
const approvedLeaves = computed(() => leaves.value.filter(l => Number(l.status) !== 0))

const displayStats = computed(() => ({
  pending: stats.value.pending || pendingLeaves.value.length,
  approved: stats.value.approved || leaves.value.filter(l => Number(l.status) === 1).length,
  rejected: stats.value.rejected || leaves.value.filter(l => Number(l.status) === 2).length,
  total: stats.value.total || leaves.value.length
}))

async function approve(id) {
  await approveTeacherLeave(id, "同意")
  uni.showToast({ title: "已同意", icon: "success" })
  loadLeaves()
}

function reject(id) {
  uni.showModal({
    title: "拒绝原因",
    editable: true,
    placeholderText: "请输入拒绝原因",
    success: async (res) => {
      if (!res.confirm) return
      if (!res.content?.trim()) {
        uni.showToast({ title: "请填写拒绝原因", icon: "none" })
        return
      }
      await rejectTeacherLeave(id, res.content.trim())
      uni.showToast({ title: "已拒绝", icon: "success" })
      loadLeaves()
    }
  })
}

function formatDateRange(item) {
  const start = item.start_date || ""
  const end = item.end_date || ""
  if (!start) return ""
  if (start === end || !end) return start
  const days = Math.ceil((new Date(end) - new Date(start)) / 86400000) + 1
  return `${start} 至 ${end}（共${days}天）`
}

function formatSubmitTime(dateStr) {
  if (!dateStr) return ""
  const d = new Date(dateStr)
  const now = new Date()
  const diff = Math.floor((now - d) / 86400000)
  const time = dateStr.slice(11, 16)
  if (diff === 0) return `今天 ${time}`
  if (diff === 1) return `昨天 ${time}`
  return dateStr.slice(0, 16)
}

onLoad(loadLeaves)
onPullDownRefresh(loadLeaves)
</script>

<template>
  <view class="page">
    <!-- 统计卡片 -->
    <view class="stats-card">
      <view class="stats-grid">
        <view class="stat-item">
          <text class="stat-num is-red">{{ displayStats.pending || 0 }}</text>
          <text class="stat-label">待审批</text>
        </view>
        <view class="stat-item">
          <text class="stat-num is-green">{{ displayStats.approved || 0 }}</text>
          <text class="stat-label">已同意</text>
        </view>
        <view class="stat-item">
          <text class="stat-num is-gray">{{ displayStats.rejected || 0 }}</text>
          <text class="stat-label">已拒绝</text>
        </view>
        <view class="stat-item">
          <text class="stat-num is-blue">{{ displayStats.total || 0 }}</text>
          <text class="stat-label">本月总计</text>
        </view>
      </view>
    </view>

    <!-- 待审批列表 -->
    <view class="section">
      <view class="section-head">
        <text class="section-title">待审批申请（{{ pendingLeaves.length }}条）</text>
      </view>

      <view v-if="!loading && pendingLeaves.length === 0" class="empty-state">
        <text>暂无待审批申请</text>
      </view>

      <view v-else class="leave-list">
        <view v-for="item in pendingLeaves" :key="item.id" class="leave-card">
          <!-- 头部信息 -->
          <view class="leave-header">
            <view class="leave-user">
              <view class="user-avatar">
                <image v-if="item.parent?.avatar" :src="item.parent.avatar" mode="aspectFill" class="avatar-img" />
                <text v-else class="avatar-text">{{ (item.parent?.nickname || item.student?.name || "家")[0] }}</text>
              </view>
              <view class="user-info">
                <view class="user-name-row">
                  <text class="user-name">{{ item.parent?.nickname || item.student?.name + "家长" || "家长" }}</text>
                  <view class="status-tag is-pending">
                    <text>待审批</text>
                  </view>
                </view>
                <text class="leave-reason">请假原因：{{ item.reason || "未填写" }}</text>
                <text class="leave-period">请假时间：{{ formatDateRange(item) }}</text>
                <view class="leave-meta">
                  <text class="meta-icon">时</text>
                  <text class="meta-text">提交时间：{{ formatSubmitTime(item.created_at) }}</text>
                </view>
              </view>
            </view>
          </view>

          <!-- 操作按钮 -->
          <view class="leave-actions">
            <view class="action-btn approve-btn" @click="approve(item.id)">
              <text>同意</text>
            </view>
            <view class="action-btn reject-btn" @click="reject(item.id)">
              <text>拒绝</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <!-- 已审批列表 -->
    <view class="section">
      <view class="section-head">
        <text class="section-title">最近已审批（{{ approvedLeaves.length }}条）</text>
      </view>

      <view v-if="approvedLeaves.length === 0" class="empty-state">
        <text>暂无审批记录</text>
      </view>

      <view v-else class="leave-list">
        <view v-for="item in approvedLeaves" :key="item.id" class="leave-card">
          <view class="leave-header">
            <view class="leave-user">
              <view class="user-avatar">
                <image v-if="item.parent?.avatar" :src="item.parent.avatar" mode="aspectFill" class="avatar-img" />
                <text v-else class="avatar-text">{{ (item.parent?.nickname || item.student?.name || "家")[0] }}</text>
              </view>
              <view class="user-info">
                <view class="user-name-row">
                  <text class="user-name">{{ item.parent?.nickname || item.student?.name + "家长" || "家长" }}</text>
                  <view class="status-tag" :class="Number(item.status) === 1 ? 'is-approved' : 'is-rejected'">
                    <text>{{ Number(item.status) === 1 ? "已同意" : "已拒绝" }}</text>
                  </view>
                </view>
                <text class="leave-reason">请假原因：{{ item.reason || "未填写" }}</text>
                <text class="leave-period">请假时间：{{ formatDateRange(item) }}</text>
                <view class="leave-meta">
                  <text class="meta-icon">时</text>
                  <text class="meta-text">审批时间：{{ formatSubmitTime(item.updated_at || item.created_at) }}</text>
                </view>
              </view>
            </view>
            <text v-if="Number(item.status) === 1" class="approved-icon">✓</text>
          </view>
        </view>
      </view>
    </view>

    <view class="bottom-safe" />
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: $spacing-lg 28rpx;
  background: $color-bg-page;
}

// ===== 统计卡片 =====
.stats-card {
  padding: 40rpx 32rpx;
  border-radius: $border-radius-xl;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 32rpx;
}

.stats-grid {
  display: flex;
  justify-content: space-around;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-xs;
}

.stat-num {
  font-size: 48rpx;
  font-weight: 800;
}

.stat-num.is-red   { color: #EF4444; }
.stat-num.is-green { color: #10B981; }
.stat-num.is-gray  { color: $color-text-muted; }
.stat-num.is-blue  { color: #3B82F6; }

.stat-label {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
}

// ===== 章节 =====
.section {
  margin-bottom: 32rpx;
}

.section-head {
  margin-bottom: 24rpx;
}

.section-title {
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

.empty-state {
  padding: 80rpx 0;
  text-align: center;

  text { font-size: $typography-font-size-md; color: $color-text-hint; }
}

// ===== 请假卡片 =====
.leave-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.leave-card {
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
}

.leave-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.leave-user {
  display: flex;
  gap: $spacing-lg;
  flex: 1;
  min-width: 0;
}

.user-avatar {
  flex-shrink: 0;
  width: 96rpx;
  height: 96rpx;
  border-radius: 999rpx;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #DBEAFE;
}

.avatar-img {
  width: 100%;
  height: 100%;
}

.avatar-text {
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: #3B82F6;
}

.user-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.user-name-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.user-name {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
}

.status-tag {
  padding: 4rpx 16rpx;
  border-radius: $border-radius-sm;
}

.status-tag.is-pending {
  background: #FEF2F2;
  text { font-size: $typography-font-size-sm; color: #EF4444; font-weight: 500; }
}

.status-tag.is-approved {
  background: #F0FDF4;
  text { font-size: $typography-font-size-sm; color: #10B981; font-weight: 500; }
}

.status-tag.is-rejected {
  background: #F1F5F9;
  text { font-size: $typography-font-size-sm; color: $color-text-muted; font-weight: 500; }
}

.leave-reason, .leave-period {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
}

.leave-meta {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.meta-icon {
  font-size: $typography-font-size-sm;
  font-weight: 700;
  color: $color-text-hint;
}

.meta-text {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.approved-icon {
  flex-shrink: 0;
  font-size: $typography-font-size-3xl;
  font-weight: 700;
  color: #22C55E;
}

// ===== 操作按钮 =====
.leave-actions {
  display: flex;
  gap: $spacing-lg;
  margin-top: 24rpx;
  padding-top: 24rpx;
  border-top: 2rpx solid #F1F5F9;
}

.action-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 80rpx;
  border-radius: $border-radius-lg;

  text { font-size: $typography-font-size-md; font-weight: 600; }
}

.approve-btn {
  background: #F0FDF4;
  border: 2rpx solid #BBF7D0;

  text { color: #10B981; }
}

.reject-btn {
  background: #FEF2F2;
  border: 2rpx solid #FECACA;

  text { color: #EF4444; }
}

.bottom-safe {
  height: calc(60rpx + env(safe-area-inset-bottom));
}
</style>
