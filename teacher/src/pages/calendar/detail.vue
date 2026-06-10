<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { fetchTeacherCalendarEvents } from "@/api/teacher-calendar"

const event = ref({})
const loading = ref(true)

const typeLabelMap = { notification: "通知", activity: "活动", meeting: "家长会", holiday: "假期", health: "体检" }
const typeColorMap = {
  notification: { bg: "#DBEAFE", fg: "#3B82F6" },
  activity: { bg: "#DCFCE7", fg: "#16A34A" },
  meeting: { bg: "#F3E8FF", fg: "#9333EA" },
  holiday: { bg: "#FFF7ED", fg: "#F97316" },
  health: { bg: "#FEE2E2", fg: "#EF4444" }
}

const readAvatars = Array.from({ length: 5 }, (_, i) => i)
const unreadCount = ref(6)
const readCount = ref(18)

function getTypeStyle(type) {
  return typeColorMap[type] || { bg: "#F1F5F9", fg: "#64748B" }
}

function goEdit() {
  uni.navigateTo({ url: `/pages/calendar/create?id=${event.value.id}` })
}

function notifyParents() {
  uni.showToast({ title: "已发送通知提醒", icon: "success" })
}

function goAllReaders() {
  uni.showToast({ title: "查看全部阅读状态", icon: "none" })
}

function downloadFile(file) {
  uni.showToast({ title: `下载 ${file.name}`, icon: "none" })
}

function handleShare() {
  uni.showToast({ title: "点击右上角分享", icon: "none" })
}

function handleMore() {
  uni.showActionSheet({
    itemList: ["编辑活动", "删除活动", "取消"],
    success(res) {
      if (res.tapIndex === 0) goEdit()
      else if (res.tapIndex === 1) {
        uni.showModal({
          title: "删除活动",
          content: "确定要删除该活动吗？",
          success(r) {
            if (r.confirm) {
              uni.showToast({ title: "已删除", icon: "success" })
              setTimeout(() => uni.navigateBack(), 1000)
            }
          }
        })
      }
    }
  })
}

onLoad(async (options) => {
  const id = options?.id
  if (id) {
    try {
      const res = await fetchTeacherCalendarEvents()
      const list = res.data?.list || []
      const match = list.find(function(e) { return e.id === Number(id) || String(e.id) === String(id) })
      event.value = match || {}
    } catch { /* */ }
  }
  loading.value = false
})
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state-wrap">
      <text class="state-text">加载中...</text>
    </view>

    <template v-else>
      <!-- 活动封面图 -->
      <view v-if="event.cover" class="cover-section">
        <image :src="event.cover" mode="aspectFill" class="cover-img" />
      </view>

      <!-- 活动基本信息卡片 -->
      <view class="info-card">
        <!-- 头部标签行 -->
        <view class="tag-row">
          <view class="type-tag" :style="{ backgroundColor: getTypeStyle(event.event_type).bg }">
            <text class="type-icon" :style="{ color: getTypeStyle(event.event_type).fg }">{{ (typeLabelMap[event.event_type] || "活动")[0] }}</text>
            <text class="type-text" :style="{ color: getTypeStyle(event.event_type).fg }">{{ typeLabelMap[event.event_type] || "活动" }}</text>
          </view>
          <view class="status-tag">
            <view class="status-dot" />
            <text>进行中</text>
          </view>
        </view>

        <!-- 标题 -->
        <text class="event-title">{{ event.title || "活动标题" }}</text>

        <!-- 描述 -->
        <text class="event-desc">{{ event.description || event.content || "暂无描述" }}</text>

        <!-- 信息列表 -->
        <view class="info-list">
          <!-- 日期 -->
          <view class="info-row">
            <view class="info-icon-box is-blue">
              <text class="info-icon">日</text>
            </view>
            <view class="info-content">
              <text class="info-label">活动日期</text>
              <text class="info-value">{{ event.event_date || event.date || "未设置" }}</text>
            </view>
          </view>

          <!-- 时间 -->
          <view class="info-row">
            <view class="info-icon-box is-orange">
              <text class="info-icon">时</text>
            </view>
            <view class="info-content">
              <text class="info-label">活动时间</text>
              <text class="info-value">{{ event.start_time || "未设置" }}{{ event.end_time ? " - " + event.end_time : "" }}</text>
            </view>
          </view>

          <!-- 地点 -->
          <view class="info-row">
            <view class="info-icon-box is-green">
              <text class="info-icon">点</text>
            </view>
            <view class="info-content">
              <text class="info-label">活动地点</text>
              <text class="info-value">{{ event.location || "未设置" }}</text>
            </view>
          </view>

          <!-- 班级 -->
          <view class="info-row">
            <view class="info-icon-box is-purple">
              <text class="info-icon">班</text>
            </view>
            <view class="info-content">
              <text class="info-label">参与班级</text>
              <view class="class-tags">
                <view v-for="cls in (event.classes || [])" :key="cls" class="class-tag">
                  <text>{{ cls }}</text>
                </view>
                <text v-if="!(event.classes || []).length" class="info-value">未设置</text>
              </view>
            </view>
          </view>

          <!-- 重要程度 -->
          <view class="info-row">
            <view class="info-icon-box is-amber">
              <text class="info-icon">重</text>
            </view>
            <view class="info-content">
              <text class="info-label">重要程度</text>
              <text class="info-value importance">{{ event.importance || "普通" }}</text>
            </view>
          </view>
        </view>
      </view>

      <!-- 发布者信息卡片 -->
      <view class="publisher-card">
        <view class="section-head">
          <view class="indicator is-blue" />
          <text class="section-title">发布信息</text>
        </view>
        <view class="publisher-row">
          <view class="publisher-avatar">
            <image v-if="event.creator?.avatar" :src="event.creator.avatar" mode="aspectFill" class="avatar-img" />
            <text v-else class="avatar-text">{{ (event.creator?.nickname || "老")[0] }}</text>
          </view>
          <view class="publisher-info">
            <text class="publisher-name">{{ event.creator?.nickname || "老师" }}</text>
            <text class="publisher-role">{{ event.creator?.role_label || "班主任" }}</text>
          </view>
          <view class="publish-time">
            <text class="time-label">发布于</text>
            <text class="time-value">{{ (event.created_at || "").slice(0, 10) }} {{ (event.created_at || "").slice(11, 16) }}</text>
          </view>
        </view>
      </view>

      <!-- 附件 -->
      <view v-if="(event.attachments || []).length" class="attachment-card">
        <view class="section-head">
          <view class="indicator is-green" />
          <text class="section-title">附件</text>
          <text class="section-extra">（{{ event.attachments.length }}个）</text>
        </view>
        <view
          v-for="(file, idx) in event.attachments"
          :key="idx"
          class="file-item"
          :class="{ 'is-alt': idx % 2 === 1 }"
        >
          <view class="file-icon-box" :class="{ 'is-alt': idx % 2 === 1 }">
            <text class="file-icon">文</text>
          </view>
          <view class="file-info">
            <text class="file-name">{{ file.name || "文件" }}</text>
            <text class="file-meta">{{ file.type || "未知" }} · {{ file.size || "未知" }}</text>
          </view>
          <view class="download-btn" :class="{ 'is-alt': idx % 2 === 1 }" @click="downloadFile(file)">
            <text class="download-icon">↓</text>
          </view>
        </view>
      </view>

      <!-- 家长阅读状态 -->
      <view class="read-card">
        <view class="section-head">
          <view class="indicator is-purple" />
          <text class="section-title">家长阅读状态</text>
          <text class="view-all" @click="goAllReaders">查看全部 ▸</text>
        </view>

        <!-- 统计 -->
        <view class="read-stats">
          <view class="read-stat">
            <view class="stat-dot is-green" />
            <text class="stat-label">已读</text>
            <text class="stat-num is-green-num">{{ readCount }}</text>
          </view>
          <view class="stat-divider" />
          <view class="read-stat">
            <view class="stat-dot is-amber" />
            <text class="stat-label">未读</text>
            <text class="stat-num is-amber-num">{{ unreadCount }}</text>
          </view>
        </view>

        <!-- 已读头像 -->
        <view class="read-avatars">
          <view v-for="i in readAvatars" :key="i" class="read-avatar" />
          <view class="read-more">
            <text>+13</text>
          </view>
        </view>
      </view>

      <!-- 底部操作 -->
      <view class="bottom-actions">
        <view class="edit-btn" @click="goEdit">
          <text class="edit-icon">编</text>
          <text class="edit-text">编辑活动</text>
        </view>
        <view class="notify-btn" @click="notifyParents">
          <text class="notify-icon">铃</text>
          <text class="notify-text">再次通知家长</text>
        </view>
      </view>

      <view class="bottom-safe" />
    </template>
  </view>
</template>

<style scoped lang="scss">
$blue-500: #3B82F6;
$blue-50: #EFF6FF;
$slate-50: #F8FAFC;
$slate-200: #E2E8F0;
$slate-400: #94A3B8;
$slate-500: #64748B;
$slate-800: #1E293B;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 24rpx 28rpx;
  background: $slate-50;
}

.state-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}

.state-text {
  font-size: 28rpx;
  color: $slate-400;
}

// ===== 封面 =====
.cover-section {
  margin-bottom: 24rpx;
}

.cover-img {
  width: 100%;
  height: 352rpx;
  border-radius: 32rpx;
}

// ===== 信息卡片 =====
.info-card {
  padding: 40rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.tag-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.type-tag {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 8rpx 24rpx;
  border-radius: 16rpx;
}

.type-icon {
  font-size: 26rpx;
  font-weight: 700;
}

.type-text {
  font-size: 24rpx;
  font-weight: 500;
}

.status-tag {
  display: flex;
  align-items: center;
  gap: 8rpx;
  padding: 8rpx 24rpx;
  border-radius: 16rpx;
  background: $blue-50;
}

.status-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: 999rpx;
  background: $blue-500;
}

.status-tag text {
  font-size: 24rpx;
  font-weight: 500;
  color: $blue-500;
}

.event-title {
  display: block;
  font-size: 40rpx;
  font-weight: 800;
  color: $slate-800;
  margin-bottom: 24rpx;
}

.event-desc {
  display: block;
  font-size: 28rpx;
  color: $slate-500;
  line-height: 1.6;
  margin-bottom: 32rpx;
}

// ===== 信息列表 =====
.info-list {
  display: flex;
  flex-direction: column;
  gap: 24rpx;
  padding-top: 32rpx;
  border-top: 2rpx solid #F1F5F9;
}

.info-row {
  display: flex;
  align-items: flex-start;
  gap: 24rpx;
}

.info-icon-box {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border-radius: 24rpx;
}

.info-icon-box.is-blue   { background: $blue-50; }
.info-icon-box.is-orange { background: #FFF7ED; }
.info-icon-box.is-green  { background: #DCFCE7; }
.info-icon-box.is-purple { background: #F3E8FF; }
.info-icon-box.is-amber  { background: #FFFBEB; }

.info-icon {
  font-size: 32rpx;
  font-weight: 700;

  .is-blue &   { color: $blue-500; }
  .is-orange & { color: #F97316; }
  .is-green &  { color: #16A34A; }
  .is-purple & { color: #9333EA; }
  .is-amber &  { color: #F59E0B; }
}

.info-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.info-label {
  font-size: 24rpx;
  color: $slate-400;
}

.info-value {
  font-size: 28rpx;
  font-weight: 500;
  color: $slate-800;
}

.info-value.importance {
  color: #F59E0B;
  font-weight: 600;
}

.class-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}

.class-tag {
  padding: 6rpx 20rpx;
  border-radius: 12rpx;
  background: #F1F5F9;

  text {
    font-size: 24rpx;
    color: $slate-500;
  }
}

// ===== 发布者卡片 =====
.publisher-card {
  padding: 40rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.section-head {
  display: flex;
  align-items: center;
  gap: 16rpx;
  margin-bottom: 32rpx;
}

.indicator {
  width: 10rpx;
  height: 40rpx;
  border-radius: 999rpx;
}

.indicator.is-blue   { background: $blue-500; }
.indicator.is-green  { background: #22C55E; }
.indicator.is-purple { background: #8B5CF6; }

.section-title {
  font-size: 30rpx;
  font-weight: 700;
  color: $slate-800;
  flex: 1;
}

.section-extra {
  font-size: 24rpx;
  color: $slate-400;
}

.view-all {
  font-size: 26rpx;
  color: $blue-500;
}

.publisher-row {
  display: flex;
  align-items: center;
  gap: 24rpx;
}

.publisher-avatar {
  flex-shrink: 0;
  width: 88rpx;
  height: 88rpx;
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
  font-size: 34rpx;
  font-weight: 700;
  color: $blue-500;
}

.publisher-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.publisher-name {
  font-size: 28rpx;
  font-weight: 600;
  color: $slate-800;
}

.publisher-role {
  font-size: 24rpx;
  color: $slate-400;
}

.publish-time {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4rpx;
}

.time-label, .time-value {
  font-size: 24rpx;
  color: $slate-400;
}

// ===== 附件 =====
.attachment-card {
  padding: 40rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 24rpx;
  padding: 24rpx;
  border-radius: 24rpx;
  background: $slate-50;
  margin-bottom: 16rpx;

  &:last-child { margin-bottom: 0; }
}

.file-icon-box {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: 24rpx;
  background: $blue-50;

  &.is-alt { background: #FEF3C7; }
}

.file-icon {
  font-size: 34rpx;
  font-weight: 700;

  .file-icon-box:not(.is-alt) & { color: $blue-500; }
  .file-icon-box.is-alt & { color: #F59E0B; }
}

.file-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}

.file-name {
  font-size: 26rpx;
  font-weight: 500;
  color: $slate-800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-meta {
  font-size: 22rpx;
  color: $slate-400;
}

.download-btn {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border-radius: 999rpx;
  background: $blue-50;

  &.is-alt { background: #FEF3C7; }
}

.download-icon {
  font-size: 30rpx;
  font-weight: 700;
  color: $blue-500;

  .download-btn.is-alt & { color: #F59E0B; }
}

// ===== 阅读状态 =====
.read-card {
  padding: 40rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.read-stats {
  display: flex;
  align-items: center;
  gap: 24rpx;
  margin-bottom: 32rpx;
}

.read-stat {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.stat-dot {
  width: 20rpx;
  height: 20rpx;
  border-radius: 999rpx;
}

.stat-dot.is-green { background: #16A34A; }
.stat-dot.is-amber { background: #F59E0B; }

.stat-label {
  font-size: 26rpx;
  font-weight: 500;
  color: $slate-800;
}

.stat-num {
  font-size: 26rpx;
  font-weight: 800;
}

.stat-num.is-green-num { color: #16A34A; }
.stat-num.is-amber-num { color: #F59E0B; }

.stat-divider {
  width: 4rpx;
  height: 32rpx;
  border-radius: 2rpx;
  background: $slate-200;
}

.read-avatars {
  display: flex;
  align-items: center;
}

.read-avatar {
  width: 72rpx;
  height: 72rpx;
  border-radius: 999rpx;
  background: #F1F5F9;

  &:not(:first-child) {
    margin-left: -20rpx;
  }
}

.read-more {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border-radius: 999rpx;
  background: #F1F5F9;
  margin-left: -20rpx;

  text {
    font-size: 22rpx;
    font-weight: 500;
    color: $slate-500;
  }
}

// ===== 底部操作 =====
.bottom-actions {
  display: flex;
  gap: 24rpx;
}

.edit-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  height: 96rpx;
  border: 2rpx solid $slate-200;
  border-radius: 24rpx;
  background: #fff;
}

.edit-icon {
  font-size: 32rpx;
  color: $slate-500;
}

.edit-text {
  font-size: 30rpx;
  font-weight: 500;
  color: $slate-500;
}

.notify-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  height: 96rpx;
  border-radius: 24rpx;
  background: $blue-500;
  box-shadow: 0 8rpx 24rpx rgba(59, 130, 246, 0.3);
}

.notify-icon {
  font-size: 32rpx;
  color: #fff;
}

.notify-text {
  font-size: 30rpx;
  font-weight: 700;
  color: #fff;
}

.bottom-safe {
  height: calc(60rpx + env(safe-area-inset-bottom));
}
</style>
