<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { fetchNoticeDetail, confirmNotice } from "@/api/parent-notice"

const detail = ref({})
const loading = ref(false)

async function loadData(id) {
  loading.value = true
  try {
    const res = await fetchNoticeDetail(id)
    detail.value = res.data || {}
  } finally {
    loading.value = false
  }
}

async function handleConfirm() {
  try {
    await confirmNotice(detail.value.id)
    detail.value.is_confirmed = true
    detail.value.confirmed_at = new Date().toISOString()
    uni.showToast({ title: "已确认", icon: "success" })
  } catch (err) {
    uni.showToast({ title: err?.message || "操作失败", icon: "none" })
  }
}

function previewImage(url) {
  uni.previewImage({ urls: [url] })
}

onLoad((opts) => { if (opts?.id) loadData(opts.id) })
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>
    <template v-else>
      <!-- 内容卡片 -->
      <view class="card">
        <view class="card-head">
          <view class="category-tag">{{ detail.category_label }}</view>
          <text class="meta-text">{{ detail.published_at?.slice(0, 16) }}</text>
        </view>
        <text class="title">{{ detail.title }}</text>
        <view class="spacer-h16" />
        <view class="author-row">
          <view class="author-avatar">
            <text class="avatar-icon"></text>
          </view>
          <view class="spacer-w12" />
          <text class="author-name">{{ detail.creator?.nickname || "" }}</text>
        </view>
        <view class="spacer-h24" />
        <text class="content">{{ detail.content }}</text>
        <!-- 附件 -->
        <template v-if="detail.attachments?.length">
          <view class="spacer-h24" />
          <text class="section-label">附件</text>
          <view class="spacer-h12" />
          <view class="attach-list">
            <image
              v-for="att in detail.attachments" :key="att.id"
              :src="att.file_url"
              class="attach-img"
              mode="aspectFill"
              @click="previewImage(att.file_url)"
            />
          </view>
        </template>
        <!-- 已读信息 -->
        <view v-if="detail.is_read" class="spacer-h24" />
        <view v-if="detail.is_read" class="read-info">
          <text class="read-text">
            已读 {{ detail.read_at?.slice(0, 16) }}
            <template v-if="detail.is_confirmed"> · 已确认 {{ detail.confirmed_at?.slice(0, 16) }}</template>
          </text>
        </view>
      </view>

      <!-- 确认按钮 -->
      <view v-if="detail.need_confirm && !detail.is_confirmed" class="confirm-bar">
        <button class="confirm-btn" @click="handleConfirm">确认收到</button>
      </view>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  padding: 24rpx 28rpx 160rpx;
  background: $color-bg-page;
}

.spacer-w4 { width: 8rpx; flex-shrink: 0; }
.spacer-w12 { width: 24rpx; flex-shrink: 0; }
.spacer-h12 { height: 24rpx; }
.spacer-h16 { height: 32rpx; }
.spacer-h24 { height: 48rpx; }

.state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  color: $color-text-hint;
  font-size: 28rpx;
}

.card {
  padding: 32rpx;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16rpx;
}

.category-tag {
  padding: 8rpx 24rpx;
  border-radius: 9999rpx;
  background: $parent-blue-bg;
  color: $parent-blue;
  font-size: 24rpx;
  font-weight: 500;
}

.meta-text {
  font-size: 24rpx;
  color: $color-text-hint;
}

.title {
  display: block;
  font-size: 36rpx;
  font-weight: 700;
  color: $color-text-primary;
  line-height: 1.4;
}

.author-row {
  display: flex;
  align-items: center;
}

.author-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 9999rpx;
  background: #DBEAFE;
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-icon {
  font-family: remixicon;
  font-size: 36rpx;
  color: #2563EB;
}

.author-name {
  font-size: 28rpx;
  color: $color-text-primary;
  font-weight: 500;
}

.content {
  display: block;
  font-size: 28rpx;
  color: $color-text-body;
  line-height: 1.8;
  white-space: pre-wrap;
}

.section-label {
  font-size: 26rpx;
  font-weight: 600;
  color: $color-text-primary;
}

.attach-list {
  display: flex;
  flex-wrap: wrap;
  gap: 16rpx;
}

.attach-img {
  width: 200rpx;
  height: 200rpx;
  border-radius: $border-radius-md;
}

.read-info {
  padding: 20rpx;
  border-radius: $border-radius-md;
  background: $color-bg-page;
}

.read-text {
  font-size: 24rpx;
  color: $color-text-hint;
}

.confirm-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  padding: 20rpx 28rpx calc(20rpx + env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.96);
}

.confirm-btn {
  width: 100%;
  height: 88rpx;
  border-radius: 9999rpx;
  background: $color-brand-primary;
  color: #fff;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 88rpx;
}
</style>
