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

function getTypeStyle(type) { return typeColorMap[type] || { bg: "#F1F5F9", fg: "#64748B" } }

function goEdit() { uni.navigateTo({ url: `/pages/calendar/create?id=${event.value.id}` }) }
function notifyParents() { uni.showToast({ title: "已发送通知提醒", icon: "success" }) }
function goAllReaders() { uni.showToast({ title: "查看全部阅读状态", icon: "none" }) }
function downloadFile(file) { uni.showToast({ title: `下载 ${file.name}`, icon: "none" }) }

function handleShare() { uni.showToast({ title: "点击右上角分享", icon: "none" }) }

function handleMore() {
  uni.showActionSheet({
    itemList: ["编辑活动", "删除活动", "取消"],
    success(res) {
      if (res.tapIndex === 0) goEdit()
      else if (res.tapIndex === 1) {
        uni.showModal({ title: "删除活动", content: "确定要删除该活动吗？", success(r) { if (r.confirm) { uni.showToast({ title: "已删除", icon: "success" }); setTimeout(() => uni.navigateBack(), 1000) } } })
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
    <view v-if="loading" class="state-wrap"><text class="state-text">加载中...</text></view>
    <template v-else>
      <view v-if="event.cover" class="cover-section"><image :src="event.cover" mode="aspectFill" class="cover-img" /></view>

      <view class="info-card">
        <view class="tag-row">
          <view class="type-tag" :style="{ backgroundColor: getTypeStyle(event.event_type).bg }">
            <text class="type-icon" :style="{ color: getTypeStyle(event.event_type).fg }">{{ (typeLabelMap[event.event_type] || "活动")[0] }}</text>
            <text class="type-text" :style="{ color: getTypeStyle(event.event_type).fg }">{{ typeLabelMap[event.event_type] || "活动" }}</text>
          </view>
          <view class="status-tag"><view class="status-dot" /><text>进行中</text></view>
        </view>
        <text class="event-title">{{ event.title || "活动标题" }}</text>
        <text class="event-desc">{{ event.description || event.content || "暂无描述" }}</text>

        <view class="info-list">
          <view class="info-row">
            <view class="info-icon-box is-blue"><text class="info-icon">日</text></view>
            <view class="info-content"><text class="info-label">活动日期</text><text class="info-value">{{ event.event_date || event.date || "未设置" }}</text></view>
          </view>
          <view class="info-row">
            <view class="info-icon-box is-orange"><text class="info-icon">时</text></view>
            <view class="info-content"><text class="info-label">活动时间</text><text class="info-value">{{ event.start_time || "未设置" }}{{ event.end_time ? " - " + event.end_time : "" }}</text></view>
          </view>
          <view class="info-row">
            <view class="info-icon-box is-green"><text class="info-icon">点</text></view>
            <view class="info-content"><text class="info-label">活动地点</text><text class="info-value">{{ event.location || "未设置" }}</text></view>
          </view>
          <view class="info-row">
            <view class="info-icon-box is-purple"><text class="info-icon">班</text></view>
            <view class="info-content">
              <text class="info-label">参与班级</text>
              <view class="class-tags">
                <view v-for="cls in (event.classes || [])" :key="cls" class="class-tag"><text>{{ cls }}</text></view>
                <text v-if="!(event.classes || []).length" class="info-value">未设置</text>
              </view>
            </view>
          </view>
          <view class="info-row">
            <view class="info-icon-box is-amber"><text class="info-icon">重</text></view>
            <view class="info-content"><text class="info-label">重要程度</text><text class="info-value importance">{{ event.importance || "普通" }}</text></view>
          </view>
        </view>
      </view>

      <view class="publisher-card">
        <view class="section-head"><view class="indicator is-blue" /><text class="section-title">发布信息</text></view>
        <view class="publisher-row">
          <view class="publisher-avatar">
            <image v-if="event.creator?.avatar" :src="event.creator.avatar" mode="aspectFill" class="avatar-img" />
            <text v-else class="avatar-text">{{ (event.creator?.nickname || "老")[0] }}</text>
          </view>
          <view class="publisher-info"><text class="publisher-name">{{ event.creator?.nickname || "老师" }}</text><text class="publisher-role">{{ event.creator?.role_label || "班主任" }}</text></view>
          <view class="publish-time"><text class="time-label">发布于</text><text class="time-value">{{ (event.created_at || "").slice(0, 10) }} {{ (event.created_at || "").slice(11, 16) }}</text></view>
        </view>
      </view>

      <view v-if="(event.attachments || []).length" class="attachment-card">
        <view class="section-head"><view class="indicator is-green" /><text class="section-title">附件</text><text class="section-extra">（{{ event.attachments.length }}个）</text></view>
        <view v-for="(file, idx) in event.attachments" :key="idx" class="file-item" :class="{ 'is-alt': idx % 2 === 1 }">
          <view class="file-icon-box" :class="{ 'is-alt': idx % 2 === 1 }"><text class="file-icon">文</text></view>
          <view class="file-info"><text class="file-name">{{ file.name || "文件" }}</text><text class="file-meta">{{ file.type || "未知" }} · {{ file.size || "未知" }}</text></view>
          <view class="download-btn" :class="{ 'is-alt': idx % 2 === 1 }" @click="downloadFile(file)"><text class="download-icon">↓</text></view>
        </view>
      </view>

      <view class="read-card">
        <view class="section-head"><view class="indicator is-purple" /><text class="section-title">家长阅读状态</text><text class="view-all" @click="goAllReaders">查看全部 ▸</text></view>
        <view class="read-stats">
          <view class="read-stat"><view class="stat-dot is-green" /><text class="stat-label">已读</text><text class="stat-num is-green-num">{{ readCount }}</text></view>
          <view class="stat-divider" />
          <view class="read-stat"><view class="stat-dot is-amber" /><text class="stat-label">未读</text><text class="stat-num is-amber-num">{{ unreadCount }}</text></view>
        </view>
        <view class="read-avatars">
          <view v-for="i in readAvatars" :key="i" class="read-avatar" />
          <view class="read-more"><text>+13</text></view>
        </view>
      </view>

      <view class="bottom-actions">
        <view class="edit-btn" @click="goEdit"><text class="edit-icon">编</text><text class="edit-text">编辑活动</text></view>
        <view class="notify-btn" @click="notifyParents"><text class="notify-icon">铃</text><text class="notify-text">再次通知家长</text></view>
      </view>
      <view class="bottom-safe" />
    </template>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: $spacing-lg 28rpx;
  background: $color-bg-page;
}

.state-wrap { display: flex; align-items: center; justify-content: center; min-height: 60vh; }
.state-text { font-size: $typography-font-size-md; color: $color-text-hint; }

.cover-section { margin-bottom: $spacing-lg; }
.cover-img { width: 100%; height: 352rpx; border-radius: $border-radius-xl; }

.info-card { padding: $spacing-xl; border-radius: $border-radius-xl; background: #fff; box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05); margin-bottom: $spacing-lg; }
.tag-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: $spacing-lg; }

.type-tag { display: flex; align-items: center; gap: $spacing-sm; padding: $spacing-xs $spacing-lg; border-radius: $border-radius-md; }
.type-icon { font-size: $typography-font-size-base; font-weight: 700; }
.type-text { font-size: $typography-font-size-sm; font-weight: 500; }

.status-tag { display: flex; align-items: center; gap: $spacing-xs; padding: $spacing-xs $spacing-lg; border-radius: $border-radius-md; background: $color-semantic-info-bg; }
.status-dot { width: 16rpx; height: 16rpx; border-radius: $border-radius-full; background: $color-semantic-info; }
.status-tag text { font-size: $typography-font-size-sm; font-weight: 500; color: $color-semantic-info; }

.event-title { display: block; font-size: $typography-font-size-3xl; font-weight: 800; color: $color-text-primary; margin-bottom: $spacing-lg; }
.event-desc { display: block; font-size: $typography-font-size-md; color: $color-text-muted; line-height: 1.6; margin-bottom: $spacing-xl; }

.info-list { display: flex; flex-direction: column; gap: $spacing-lg; padding-top: $spacing-xl; border-top: 2rpx solid #F1F5F9; }
.info-row { display: flex; align-items: flex-start; gap: $spacing-lg; }
.info-icon-box { display: flex; flex-shrink: 0; align-items: center; justify-content: center; width: 72rpx; height: 72rpx; border-radius: $border-radius-xl; }
.info-icon-box.is-blue   { background: $color-semantic-info-bg; }
.info-icon-box.is-orange { background: #FFF7ED; }
.info-icon-box.is-green  { background: #DCFCE7; }
.info-icon-box.is-purple { background: #F3E8FF; }
.info-icon-box.is-amber  { background: #FFFBEB; }

.info-icon { font-size: $typography-font-size-xl; font-weight: 700; }
.info-icon-box.is-blue   .info-icon { color: $color-semantic-info; }
.info-icon-box.is-orange .info-icon { color: #F97316; }
.info-icon-box.is-green  .info-icon { color: #16A34A; }
.info-icon-box.is-purple .info-icon { color: #9333EA; }
.info-icon-box.is-amber  .info-icon { color: #F59E0B; }

.info-content { flex: 1; display: flex; flex-direction: column; gap: 4rpx; }
.info-label { font-size: $typography-font-size-sm; color: $color-text-hint; }
.info-value { font-size: $typography-font-size-md; font-weight: 500; color: $color-text-primary; }
.info-value.importance { color: #F59E0B; font-weight: 600; }

.class-tags { display: flex; flex-wrap: wrap; gap: $spacing-sm; }
.class-tag { padding: 6rpx 20rpx; border-radius: $border-radius-md; background: #F1F5F9; text { font-size: $typography-font-size-sm; color: $color-text-muted; } }

.publisher-card { padding: $spacing-xl; border-radius: $border-radius-xl; background: #fff; box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05); margin-bottom: $spacing-lg; }
.section-head { display: flex; align-items: center; gap: $spacing-md; margin-bottom: $spacing-xl; }
.indicator { width: 10rpx; height: 40rpx; border-radius: $border-radius-full; }
.indicator.is-blue   { background: $color-semantic-info; }
.indicator.is-green  { background: #22C55E; }
.indicator.is-purple { background: #8B5CF6; }

.section-title { font-size: $typography-font-size-lg; font-weight: 700; color: $color-text-primary; flex: 1; }
.section-extra { font-size: $typography-font-size-sm; color: $color-text-hint; }
.view-all { font-size: $typography-font-size-base; color: $color-semantic-info; }

.publisher-row { display: flex; align-items: center; gap: $spacing-lg; }
.publisher-avatar { flex-shrink: 0; width: 88rpx; height: 88rpx; border-radius: $border-radius-full; overflow: hidden; display: flex; align-items: center; justify-content: center; background: #DBEAFE; }
.avatar-img { width: 100%; height: 100%; }
.avatar-text { font-size: 34rpx; font-weight: 700; color: $color-semantic-info; }
.publisher-info { flex: 1; display: flex; flex-direction: column; gap: 4rpx; }
.publisher-name { font-size: $typography-font-size-md; font-weight: 600; color: $color-text-primary; }
.publisher-role { font-size: $typography-font-size-sm; color: $color-text-hint; }
.publish-time { display: flex; flex-direction: column; align-items: flex-end; gap: 4rpx; }
.time-label, .time-value { font-size: $typography-font-size-sm; color: $color-text-hint; }

.attachment-card { padding: $spacing-xl; border-radius: $border-radius-xl; background: #fff; box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05); margin-bottom: $spacing-lg; }
.file-item { display: flex; align-items: center; gap: $spacing-lg; padding: $spacing-lg; border-radius: $border-radius-xl; background: $color-bg-card; margin-bottom: $spacing-md; &:last-child { margin-bottom: 0; } }
.file-icon-box { display: flex; flex-shrink: 0; align-items: center; justify-content: center; width: 80rpx; height: 80rpx; border-radius: $border-radius-xl; background: $color-semantic-info-bg; &.is-alt { background: #FEF3C7; } }

.file-icon { font-size: 34rpx; font-weight: 700; }
.file-icon-box:not(.is-alt) .file-icon { color: $color-semantic-info; }
.file-icon-box.is-alt .file-icon { color: #F59E0B; }

.file-info { flex: 1; display: flex; flex-direction: column; gap: 4rpx; min-width: 0; }
.file-name { font-size: $typography-font-size-base; font-weight: 500; color: $color-text-primary; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-meta { font-size: $typography-font-size-xs; color: $color-text-hint; }
.download-btn { display: flex; flex-shrink: 0; align-items: center; justify-content: center; width: 72rpx; height: 72rpx; border-radius: $border-radius-full; background: $color-semantic-info-bg; &.is-alt { background: #FEF3C7; } }
.download-icon { font-size: $typography-font-size-lg; font-weight: 700; color: $color-semantic-info; .download-btn.is-alt & { color: #F59E0B; } }

.read-card { padding: $spacing-xl; border-radius: $border-radius-xl; background: #fff; box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05); margin-bottom: $spacing-lg; }
.read-stats { display: flex; align-items: center; gap: $spacing-lg; margin-bottom: $spacing-xl; }
.read-stat { display: flex; align-items: center; gap: $spacing-sm; }
.stat-dot { width: 20rpx; height: 20rpx; border-radius: $border-radius-full; }
.stat-dot.is-green { background: #16A34A; }
.stat-dot.is-amber { background: #F59E0B; }
.stat-label { font-size: $typography-font-size-base; font-weight: 500; color: $color-text-primary; }
.stat-num { font-size: $typography-font-size-base; font-weight: 800; }
.stat-num.is-green-num { color: #16A34A; }
.stat-num.is-amber-num { color: #F59E0B; }
.stat-divider { width: 4rpx; height: 32rpx; border-radius: 2rpx; background: #E2E8F0; }
.read-avatars { display: flex; align-items: center; }
.read-avatar { width: 72rpx; height: 72rpx; border-radius: $border-radius-full; background: #F1F5F9; &:not(:first-child) { margin-left: -20rpx; } }
.read-more { display: flex; align-items: center; justify-content: center; width: 72rpx; height: 72rpx; border-radius: $border-radius-full; background: #F1F5F9; margin-left: -20rpx; text { font-size: $typography-font-size-xs; font-weight: 500; color: $color-text-muted; } }

.bottom-actions { display: flex; gap: $spacing-lg; }
.edit-btn { flex: 1; display: flex; align-items: center; justify-content: center; gap: $spacing-sm; height: 96rpx; border: 2rpx solid #E2E8F0; border-radius: $border-radius-xl; background: #fff; }
.edit-icon { font-size: $typography-font-size-xl; color: $color-text-muted; }
.edit-text { font-size: $typography-font-size-lg; font-weight: 500; color: $color-text-muted; }
.notify-btn { flex: 1; display: flex; align-items: center; justify-content: center; gap: $spacing-sm; height: 96rpx; border-radius: $border-radius-xl; background: $color-semantic-info; box-shadow: 0 8rpx 24rpx rgba(59, 130, 246, 0.3); }
.notify-icon { font-size: $typography-font-size-xl; color: #fff; }
.notify-text { font-size: $typography-font-size-lg; font-weight: 700; color: #fff; }
.bottom-safe { height: calc(60rpx + env(safe-area-inset-bottom)); }
</style>
