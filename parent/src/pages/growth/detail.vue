<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { fetchGrowthDetail, likeGrowth, unlikeGrowth, fetchGrowthComments, postGrowthComment } from "@/api/parent-growth"

const detail = ref({})
const comments = ref([])
const commentText = ref("")
const replyTo = ref(null)
const loading = ref(false)

async function loadData(id) {
  loading.value = true
  try {
    const [d, c] = await Promise.all([fetchGrowthDetail(id), fetchGrowthComments(id)])
    detail.value = d.data || {}
    comments.value = c.data?.list || []
  } finally { loading.value = false }
}

async function handleLike() {
  try {
    if (detail.value.is_liked) {
      const r = await unlikeGrowth(detail.value.id)
      detail.value.is_liked = r.data?.is_liked
      detail.value.likes_count = r.data?.likes_count
    } else {
      const r = await likeGrowth(detail.value.id)
      detail.value.is_liked = r.data?.is_liked
      detail.value.likes_count = r.data?.likes_count
    }
  } catch (err) {
    uni.showToast({ title: err?.message || "操作失败", icon: "none" })
  }
}

async function handleComment() {
  const text = commentText.value.trim()
  if (!text) return
  try {
    await postGrowthComment(detail.value.id, { content: text, parent_id: replyTo.value })
    commentText.value = ""
    replyTo.value = null
    uni.showToast({ title: "评论成功", icon: "success" })
    const c = await fetchGrowthComments(detail.value.id)
    comments.value = c.data?.list || []
  } catch (err) {
    uni.showToast({ title: err?.message || "评论失败", icon: "none" })
  }
}

function startReply(commentId, userName) {
  replyTo.value = commentId
  commentText.value = ""
  uni.showToast({ title: "回复 " + userName, icon: "none" })
}

onLoad((opts) => { if (opts?.id) loadData(opts.id) })
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>
    <template v-else>
      <!-- 发布者信息 -->
      <view class="card">
        <view class="author-row">
          <view class="author-avatar">
            <text class="avatar-icon"></text>
          </view>
          <view class="spacer-w12" />
          <view>
            <text class="author-name">{{ detail.creator?.nickname || "老师" }}</text>
            <text class="author-time">{{ detail.date }} · {{ detail.created_at?.slice(11, 16) || "" }}</text>
          </view>
        </view>
      </view>

      <!-- 当日状态标签（餐食/睡眠/情绪） -->
      <view v-if="detail.breakfast_rating" class="status-tags">
        <view class="stag"><text class="stag-label">早餐</text><text class="stag-val">{{ ['','差','中','好'][detail.breakfast_rating] || '-' }}</text></view>
        <view class="stag"><text class="stag-label">午餐</text><text class="stag-val">{{ ['','差','中','好'][detail.lunch_rating] || '-' }}</text></view>
        <view class="stag"><text class="stag-label">午睡</text><text class="stag-val">{{ detail.nap_duration }}分钟</text></view>
        <view class="stag"><text class="stag-label">情绪</text><text class="stag-val">{{ ['','差','中','好'][detail.morning_mood] || '-' }}</text></view>
      </view>

      <!-- 正文 -->
      <view class="card" style="margin-top:24rpx">
        <text class="content">{{ detail.content }}</text>

        <!-- 表现标签 -->
        <view v-if="detail.performance_tags?.length" class="tag-row">
          <view v-for="tag in detail.performance_tags" :key="tag" class="perf-tag">
            <text>{{ tag }}</text>
          </view>
        </view>

        <!-- 照片网格 -->
        <view v-if="detail.media?.length" class="media-grid">
          <image v-for="m in detail.media" :key="m.id"
            :src="m.thumb_url || m.file_url" mode="aspectFill" class="media-item"
            @click="uni.previewImage({ urls: detail.media.map(x => x.file_url), current: m.file_url })" />
        </view>

        <!-- 关联学生 -->
        <view v-if="detail.related_students?.length" class="related-row">
          <text class="related-label">关联学生：</text>
          <text v-for="s in detail.related_students" :key="s.id" class="related-name">{{ s.name }}</text>
        </view>

        <!-- 互动栏 -->
        <view class="action-bar">
          <view class="ab-left">
            <view class="ab-item" :class="{ liked: detail.is_liked }" @click="handleLike">
              <text class="ab-icon">{{ detail.is_liked ? '' : '' }}</text>
              <text class="ab-num">{{ detail.likes_count || 0 }}</text>
            </view>
            <view class="ab-item">
              <text class="ab-icon"></text>
              <text class="ab-num">{{ detail.comments_count || 0 }}</text>
            </view>
          </view>
          <view class="ab-right">
            <text class="ab-icon"></text>
            <text class="ab-num">评论</text>
          </view>
        </view>
      </view>

      <!-- 评论列表 -->
      <view class="comments-section">
        <text class="section-title">评论 ({{ comments.length }})</text>
        <view v-for="c in comments" :key="c.id" class="comment-card">
          <view class="comment-head">
            <text class="comment-user">{{ c.user?.nickname }}</text>
            <text v-if="c.user?.relation" class="comment-relation">{{ c.user.relation === 'mother' ? '妈妈' : c.user.relation === 'father' ? '爸爸' : '' }}</text>
            <text class="comment-time">{{ c.created_at?.slice(0, 16) }}</text>
          </view>
          <text class="comment-content">{{ c.content }}</text>
          <text class="comment-reply-btn" @click="startReply(c.id, c.user?.nickname)">回复</text>
          <!-- 子回复 -->
          <view v-if="c.replies?.length" class="replies">
            <view v-for="r in c.replies" :key="r.id" class="reply-card">
              <text class="reply-user">{{ r.user?.nickname }}</text>
              <text class="reply-content">{{ r.content }}</text>
            </view>
          </view>
        </view>
      </view>

      <!-- 评论输入栏 -->
      <view class="input-bar">
        <view v-if="replyTo" class="reply-hint">
          <text>回复中</text>
          <text class="cancel-reply" @click="replyTo = null">取消</text>
        </view>
        <view class="input-row">
          <input v-model="commentText" class="comment-input" placeholder="写评论..." />
          <button class="send-btn" @click="handleComment">发送</button>
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg 28rpx 180rpx; background: $color-bg-page; }
.spacer-w12 { width: 24rpx; flex-shrink: 0; }

.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-hint; font-size: $typography-font-size-md; }
.card { padding: $spacing-xl; border-radius: $border-radius-lg; background: $color-bg-card; box-shadow: $shadow-sm; }

.author-row { display: flex; align-items: center; }
.author-avatar { width: 80rpx; height: 80rpx; border-radius: 9999rpx; background: #DBEAFE; display: flex; align-items: center; justify-content: center; }
.avatar-icon { font-family: remixicon; font-size: $typography-font-size-2xl; color: #2563EB; }
.author-name { display: block; font-size: $typography-font-size-md; font-weight: 600; color: $color-text-primary; }
.author-time { display: block; font-size: $typography-font-size-sm; color: $color-text-hint; margin-top: 4rpx; }

.status-tags { display: flex; gap: $spacing-md; margin-top: 24rpx; }
.stag { flex: 1; padding: 20rpx 12rpx; border-radius: $border-radius-md; background: $color-bg-card; text-align: center; box-shadow: $shadow-sm; }
.stag-label { display: block; font-size: $typography-font-size-xs; color: $color-text-hint; }
.stag-val { display: block; margin-top: 8rpx; font-size: $typography-font-size-md; font-weight: 600; color: $color-text-primary; }

.content { display: block; font-size: $typography-font-size-md; color: $color-text-body; line-height: 1.8; white-space: pre-wrap; }

.tag-row { display: flex; flex-wrap: wrap; gap: $spacing-sm; margin-top: 20rpx; }
.perf-tag { padding: $spacing-xs 20rpx; border-radius: 9999rpx; background: #FFF7ED; font-size: $typography-font-size-sm; color: #92400E; }

.media-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: $spacing-sm; margin-top: 20rpx; }
.media-item { width: 100%; height: 200rpx; border-radius: $border-radius-md; }

.related-row { display: flex; align-items: center; gap: $spacing-sm; margin-top: 20rpx; }
.related-label { font-size: $typography-font-size-sm; color: $color-text-hint; }
.related-name { padding: 6rpx 16rpx; border-radius: 9999rpx; background: #DBEAFE; font-size: $typography-font-size-xs; color: #2563EB; }

.action-bar { display: flex; justify-content: space-between; align-items: center; margin-top: 24rpx; padding-top: 24rpx; border-top: 1rpx solid #F1F5F9; }
.ab-left { display: flex; gap: $spacing-xl; }
.ab-item { display: flex; align-items: center; }
.ab-icon { font-family: remixicon; font-size: $typography-font-size-2xl; color: $color-text-hint; }
.ab-num { font-size: $typography-font-size-base; color: $color-text-muted; margin-left: 8rpx; }
.ab-item.liked .ab-icon { color: #EF4444; }
.ab-item.liked .ab-num { color: #EF4444; }
.ab-right { display: flex; align-items: center; }

.comments-section { margin-top: 36rpx; }
.section-title { display: block; margin-bottom: 20rpx; font-size: $typography-font-size-xl; font-weight: 600; color: $color-text-primary; }

.comment-card { padding: $spacing-lg; margin-bottom: 16rpx; border-radius: $border-radius-md; background: $color-bg-card; box-shadow: $shadow-sm; }
.comment-head { display: flex; align-items: center; gap: $spacing-sm; margin-bottom: 12rpx; }
.comment-user { font-size: $typography-font-size-base; font-weight: 600; color: $color-text-primary; }
.comment-relation { font-size: $typography-font-size-xs; color: $color-brand-primary; }
.comment-time { font-size: $typography-font-size-xs; color: $color-text-hint; margin-left: auto; }
.comment-content { display: block; font-size: $typography-font-size-base; color: $color-text-body; line-height: 1.5; }
.comment-reply-btn { font-size: $typography-font-size-sm; color: $parent-blue; margin-top: 8rpx; display: inline-block; }

.replies { margin-top: 16rpx; padding-left: 24rpx; border-left: 3rpx solid #E2E8F0; }
.reply-card { padding: $spacing-md; margin-bottom: 8rpx; border-radius: $border-radius-md; background: $color-bg-page; }
.reply-user { font-size: $typography-font-size-sm; font-weight: 600; color: $color-text-primary; margin-right: 12rpx; }
.reply-content { font-size: $typography-font-size-sm; color: $color-text-body; }

.input-bar { position: fixed; right: 0; bottom: 0; left: 0; padding: $spacing-md 28rpx calc(16rpx + env(safe-area-inset-bottom)); background: #FFFFFF; box-shadow: 0 -2rpx 8rpx rgba(0,0,0,0.05); }
.reply-hint { display: flex; justify-content: space-between; font-size: $typography-font-size-sm; color: $color-brand-primary; margin-bottom: 12rpx; }
.cancel-reply { color: $color-text-hint; }
.input-row { display: flex; gap: $spacing-md; }
.comment-input { flex: 1; height: 72rpx; padding: 0 24rpx; border-radius: 9999rpx; background: $color-bg-page; font-size: $typography-font-size-base; }
.send-btn { padding: 0 32rpx; height: 72rpx; border-radius: 9999rpx; background: $color-brand-primary; color: #fff; font-size: $typography-font-size-base; font-weight: 600; line-height: 72rpx; }
</style>
