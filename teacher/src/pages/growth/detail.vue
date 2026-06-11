<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { fetchTeacherGrowthLog, fetchTeacherGrowthComments, createTeacherGrowthComment } from "@/api/teacher-growth"
import { ensureLogin } from "@/utils/login"

const record = ref({})
const comments = ref([])
const commentText = ref("")
const loading = ref(true)
const loadError = ref("")
const submitting = ref(false)

async function loadDetail(id) {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const [logRes, commentRes] = await Promise.all([
      fetchTeacherGrowthLog(id),
      fetchTeacherGrowthComments(id)
    ])
    record.value = logRes.data || {}
    comments.value = commentRes.data?.list || commentRes.data || []
  } catch (err) {
    loadError.value = err?.message || "加载失败"
  } finally {
    loading.value = false
  }
}

function goBack() {
  uni.navigateBack()
}

function goEdit() {
  uni.navigateTo({ url: `/pages/growth/create?id=${record.value.id}` })
}

function handleShare() {
  // uni.shareAppMessage 由微信小程序 onShareAppMessage 钩子接管
  uni.showToast({ title: "点击右上角分享", icon: "none" })
}

function goAllComments() {
  uni.showToast({ title: "查看全部评论", icon: "none" })
}

function replyTo(parentId) {
  uni.showToast({ title: "回复功能开发中", icon: "none" })
}

async function submitComment() {
  const text = commentText.value.trim()
  if (!text) return

  submitting.value = true
  try {
    const res = await createTeacherGrowthComment(record.value.id, text)
    commentText.value = ""
    uni.showToast({ title: "评论成功", icon: "none" })
    // Reload comments
    const commentRes = await fetchTeacherGrowthComments(record.value.id)
    comments.value = commentRes.data?.list || commentRes.data || []
  } catch (err) {
    uni.showToast({ title: err?.message || "评论失败", icon: "none" })
  } finally {
    submitting.value = false
  }
}

function formatDate(dateStr) {
  if (!dateStr) return ""
  return dateStr.slice(0, 16)
}

function formatFullDate(dateStr) {
  if (!dateStr) return ""
  const d = new Date(dateStr)
  const y = d.getFullYear()
  const m = d.getMonth() + 1
  const day = d.getDate()
  const time = dateStr.slice(11, 16)
  return `${y}年${m}月${day}日 ${time}`
}

function formatTime(dateStr) {
  if (!dateStr) return ""
  return dateStr.slice(11, 16)
}

const visibleComments = ref(3)

function showAllComments() {
  visibleComments.value = comments.value.length
}

onLoad((options) => {
  const id = options?.id
  if (id) loadDetail(id)
})
</script>

<template>
  <view class="page">
    <!-- 加载/错误状态 -->
    <view v-if="loading" class="state-wrap">
      <text class="state-text">加载中...</text>
    </view>
    <view v-else-if="loadError" class="state-wrap">
      <text class="state-text">{{ loadError }}</text>
      <button class="retry-btn" @click="loadDetail(record.id || '')">重试</button>
    </view>

    <template v-else>
      <!-- 记录头部 -->
      <view class="header-section">
        <view class="author-row">
          <view class="author-avatar">
            <text class="avatar-icon">师</text>
          </view>
          <view class="author-info">
            <text class="author-name">{{ record.creator?.nickname || "老师" }}</text>
            <text class="author-date">发布于 {{ formatFullDate(record.date || record.created_at) }}</text>
          </view>
        </view>
        <text class="record-title">{{ record.title || "成长记录" }}</text>
        <text class="record-content">{{ record.content }}</text>
      </view>

      <!-- 照片区域 -->
      <view v-if="record.media?.length" class="photo-section">
        <view class="photo-grid">
          <image
            v-for="(m, idx) in record.media"
            :key="idx"
            :src="m.thumb_url || m.url"
            mode="aspectFill"
            class="photo-item"
            @click="uni.previewImage({ current: idx, urls: record.media.map(m => m.thumb_url || m.url) })"
          />
        </view>
      </view>

      <!-- 相关孩子 -->
      <view v-if="record.students?.length" class="student-section">
        <text class="section-label">相关孩子（{{ record.students.length }}人）</text>
        <view class="student-chips">
          <view
            v-for="(student, idx) in record.students.slice(0, 6)"
            :key="student.id || idx"
            class="student-chip"
            :class="{ primary: idx === 0 }"
          >
            <image
              v-if="student.avatar"
              :src="student.avatar"
              mode="aspectFill"
              class="chip-avatar"
            />
            <view v-else class="chip-avatar-placeholder" />
            <text class="chip-name">{{ student.name }}</text>
          </view>
          <view v-if="record.students.length > 6" class="student-chip more">
            <text>+{{ record.students.length - 6 }}人</text>
          </view>
        </view>
      </view>

      <!-- 互动统计 & 操作 -->
      <view class="stats-bar">
        <view class="stats-left">
          <view class="stat-item">
            <text class="stat-icon">看</text>
            <text class="stat-text">{{ record.views_count || 0 }}次浏览</text>
          </view>
          <view class="stat-item">
            <text class="stat-icon">赞</text>
            <text class="stat-text">{{ record.likes_count || 0 }}个点赞</text>
          </view>
          <view class="stat-item">
            <text class="stat-icon">评</text>
            <text class="stat-text">{{ record.comments_count || 0 }}条评论</text>
          </view>
        </view>
        <view class="stats-right">
          <view class="action-btn edit-btn" @click="goEdit">
            <text>编辑</text>
          </view>
          <view class="action-btn share-btn" @click="handleShare">
            <text>分享</text>
          </view>
        </view>
      </view>

      <!-- 评论区域 -->
      <view v-if="comments.length" class="comment-section">
        <view class="comment-head">
          <text class="section-label">家长评论（{{ record.comments_count || comments.length }}条）</text>
          <text class="view-all" @click="goAllComments">查看全部</text>
        </view>

        <view class="comment-list">
          <view
            v-for="comment in comments.slice(0, visibleComments)"
            :key="comment.id"
            class="comment-card"
          >
            <!-- 主评论 -->
            <view class="comment-main">
              <image
                v-if="comment.user?.avatar"
                :src="comment.user.avatar"
                mode="aspectFill"
                class="comment-avatar"
              />
              <view v-else class="comment-avatar-placeholder">
                <text>{{ (comment.user?.nickname || "家")[0] }}</text>
              </view>
              <view class="comment-body">
                <view class="comment-meta">
                  <text class="comment-author">{{ comment.user?.nickname || "家长" }}</text>
                  <view v-if="comment.user?.role_tag" class="comment-role-tag">
                    <text>{{ comment.user.role_tag }}</text>
                  </view>
                  <text class="comment-time">{{ formatTime(comment.created_at) }}</text>
                </view>
                <text class="comment-text">{{ comment.content }}</text>
                <view class="comment-actions">
                  <view class="comment-act" @click="replyTo(comment.id)">
                    <text class="comment-act-icon">赞</text>
                    <text class="comment-act-num">{{ comment.likes_count || 0 }}</text>
                  </view>
                  <view class="comment-act" @click="replyTo(comment.id)">
                    <text class="comment-act-icon">回</text>
                    <text>回复</text>
                  </view>
                </view>
              </view>
            </view>

            <!-- 老师回复 -->
            <view v-if="comment.replies?.length" class="teacher-reply">
              <view v-for="reply in comment.replies" :key="reply.id" class="reply-item">
                <text class="reply-author">{{ reply.user?.nickname || "老师" }}</text>
                <text class="reply-time">{{ formatTime(reply.created_at) }}</text>
                <text class="reply-text">{{ reply.content }}</text>
              </view>
            </view>
          </view>

          <view
            v-if="comments.length > visibleComments"
            class="show-more"
            @click="showAllComments"
          >
            <text>展开全部 {{ comments.length }} 条评论</text>
          </view>
        </view>
      </view>

      <!-- 底部留白 -->
      <view class="bottom-safe" />
    </template>

    <!-- 底部评论输入栏 -->
    <view class="bottom-bar">
      <view class="input-box">
        <input
          v-model="commentText"
          class="comment-input"
          placeholder="请输入评论..."
          placeholder-style="color: #94A3B8; font-size: $typography-font-size-md"
          :disabled="submitting"
          confirm-type="send"
          @confirm="submitComment"
        />
      </view>
      <view class="bottom-btn" @click="submitComment">
        <text class="bottom-btn-icon">发</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
$slate-200: #E2E8F0;
$slate-300: #CBD5E1;
$slate-600: #475569;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  background: #fff;
}

// ===== 状态 =====
.state-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}

.state-text {
  font-size: $typography-font-size-md;
  color: $color-text-hint;
}

.retry-btn {
  width: 180rpx;
  height: 72rpx;
  margin-top: 28rpx;
  border-radius: 999rpx;
  background: $color-semantic-info;
  color: #fff;
  font-size: $typography-font-size-base;
}

// ===== 记录头部 =====
.header-section {
  padding: $spacing-xl 48rpx 32rpx;
  border-bottom: 2rpx solid $slate-200;
}

.author-row {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
}

.author-avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 96rpx;
  height: 96rpx;
  border-radius: 999rpx;
  background: #DBEAFE;
}

.avatar-icon {
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: $color-semantic-info;
}

.author-info {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.author-name {
  font-size: $typography-font-size-xl;
  font-weight: 600;
  color: $color-text-primary;
}

.author-date {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
}

.record-title {
  display: block;
  margin-top: 32rpx;
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

.record-content {
  display: block;
  margin-top: 16rpx;
  font-size: $typography-font-size-lg;
  color: $color-text-body;
  line-height: 1.7;
}

// ===== 照片区域 =====
.photo-section {
  padding: $spacing-xl 48rpx;
}

.photo-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: $spacing-xs;
}

.photo-item {
  width: 100%;
  aspect-ratio: 1;
  border-radius: $border-radius-xl;
}

// ===== 相关孩子 =====
.student-section {
  padding: 0 48rpx 32rpx;
}

.section-label {
  font-size: $typography-font-size-lg;
  font-weight: 700;
  color: $color-text-primary;
  margin-bottom: 24rpx;
}

.student-chips {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;
  margin-top: 24rpx;
}

.student-chip {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  padding: $spacing-sm 24rpx;
  border-radius: 999rpx;
  background: #F1F5F9;
}

.student-chip.primary {
  background: $color-semantic-info-bg;
}

.chip-avatar {
  width: 40rpx;
  height: 40rpx;
  border-radius: 999rpx;
}

.chip-avatar-placeholder {
  width: 40rpx;
  height: 40rpx;
  border-radius: 999rpx;
  background: $slate-300;
}

.chip-name {
  font-size: $typography-font-size-sm;
  font-weight: 500;

  .primary & {
    color: $color-semantic-info;
  }

  .student-chip:not(.primary):not(.more) & {
    color: $color-text-muted;
  }
}

.student-chip.more {
  background: #F1F5F9;

  text {
    font-size: $typography-font-size-sm;
    color: $color-text-muted;
  }
}

// ===== 互动统计 & 操作 =====
.stats-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-xl 48rpx;
  border-top: 2rpx solid $slate-200;
  border-bottom: 2rpx solid $slate-200;
}

.stats-left {
  display: flex;
  gap: 48rpx;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.stat-icon {
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: $color-text-hint;
}

.stat-text {
  font-size: $typography-font-size-md;
  color: $color-text-muted;
}

.stats-right {
  display: flex;
  gap: $spacing-xl;
}

.action-btn {
  padding: $spacing-sm 24rpx;
  border-radius: $border-radius-lg;

  text {
    font-size: $typography-font-size-base;
    font-weight: 500;
  }
}

.edit-btn {
  background: #F1F5F9;

  text { color: $color-text-muted; }
}

.share-btn {
  background: $color-semantic-info-bg;

  text { color: $color-semantic-info; }
}

// ===== 评论区域 =====
.comment-section {
  padding: $spacing-xl 48rpx;
}

.comment-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 32rpx;
}

.view-all {
  font-size: $typography-font-size-md;
  color: $color-semantic-info;
}

.comment-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-xl;
}

.comment-card {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.comment-main {
  display: flex;
  gap: $spacing-lg;
}

.comment-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 999rpx;
  flex-shrink: 0;
}

.comment-avatar-placeholder {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: 999rpx;
  background: #DBEAFE;

  text {
    font-size: $typography-font-size-xl;
    font-weight: 700;
    color: $color-semantic-info;
  }
}

.comment-body {
  flex: 1;
  min-width: 0;
}

.comment-meta {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  flex-wrap: wrap;
}

.comment-author {
  font-size: $typography-font-size-md;
  font-weight: 600;
  color: $color-text-primary;
}

.comment-role-tag {
  padding: 4rpx 16rpx;
  border-radius: $border-radius-sm;
  background: $color-semantic-info-bg;

  text {
    font-size: $typography-font-size-xs;
    font-weight: 500;
    color: $color-semantic-info;
  }
}

.comment-time {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.comment-text {
  display: block;
  margin-top: 8rpx;
  font-size: $typography-font-size-md;
  color: $color-text-body;
  line-height: 1.5;
}

.comment-actions {
  display: flex;
  gap: $spacing-xl;
  margin-top: 4rpx;
}

.comment-act {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.comment-act-icon {
  font-size: $typography-font-size-md;
  font-weight: 700;
  color: $color-text-hint;
}

.comment-act-num,
.comment-act text:last-child {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

// 老师回复
.teacher-reply {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  margin-left: 104rpx;
}

.reply-item {
  padding: $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
}

.reply-author {
  font-size: $typography-font-size-base;
  font-weight: 600;
  color: $color-semantic-info;
}

.reply-time {
  display: inline;
  margin-left: 16rpx;
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.reply-text {
  display: block;
  margin-top: 8rpx;
  font-size: $typography-font-size-base;
  color: $color-text-body;
  line-height: 1.5;
}

.show-more {
  padding: $spacing-lg 0;
  text-align: center;

  text {
    font-size: $typography-font-size-md;
    color: $color-text-hint;
  }
}

// ===== 底部留白 =====
.bottom-safe {
  height: calc(160rpx + env(safe-area-inset-bottom));
}

// ===== 底部评论输入栏 =====
.bottom-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: $spacing-md 48rpx calc(16rpx + env(safe-area-inset-bottom));
  background: #fff;
  box-shadow: 0 -2rpx 20rpx rgba(0, 0, 0, 0.04);
}

.input-box {
  flex: 1;
  min-width: 0;
  height: 80rpx;
  border-radius: 999rpx;
  background: #F1F5F9;
  padding: 0 32rpx;
  display: flex;
  align-items: center;
}

.comment-input {
  width: 100%;
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

.bottom-btn {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: 999rpx;
  background: $color-semantic-info;
}

.bottom-btn-icon {
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: #fff;
}
</style>
