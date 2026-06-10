<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchMessages, fetchUnreadCount } from "@/api/parent-messages"

const messages = ref([])
const unread = ref(0)
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const [m, u] = await Promise.all([fetchMessages(), fetchUnreadCount()])
    messages.value = m.data?.list || m.data || []
    unread.value = u.data?.count || 0
  } catch (err) {
    uni.showToast({ title: err?.message || "加载失败", icon: "none" })
  } finally { loading.value = false }
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <view class="header">
      <text class="page-title">消息</text>
      <text v-if="unread" class="badge">{{ unread }}条未读</text>
    </view>
    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="messages.length === 0" class="state">暂无消息</view>
    <view v-else class="list">
      <view v-for="item in messages" :key="item.id" class="card">
        <text class="title">{{ item.title }}</text>
        <text class="content">{{ item.content }}</text>
        <text class="date">{{ item.created_at?.slice(0, 16) }}</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx; background: $color-bg-page; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24rpx; }
.page-title { color: $color-text-primary; font-size: 36rpx; font-weight: 800; }
.badge { padding: 8rpx 20rpx; border-radius: 999rpx; background: #ff6b6b; color: #fff; font-size: 22rpx; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: 28rpx; }
.list { display: flex; flex-direction: column; gap: 18rpx; }
.card { padding: 26rpx; border-radius: 24rpx; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.title { display: block; color: $color-text-primary; font-size: 28rpx; font-weight: 700; }
.content { display: block; margin-top: 8rpx; color: $color-text-muted; font-size: 24rpx; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.date { display: block; margin-top: 12rpx; color: $color-text-muted; font-size: 20rpx; }
</style>
