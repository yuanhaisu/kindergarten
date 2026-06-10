<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchClassCalendar } from "@/api/parent-class"

const events = ref([])
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchClassCalendar()
    events.value = res.data?.list || []
  } finally { loading.value = false }
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="events.length === 0" class="state">暂无活动安排</view>
    <view v-else class="list">
      <view v-for="item in events" :key="item.id" class="card">
        <view class="dot" :style="{ background: item.color }"></view>
        <view class="info">
          <text class="title">{{ item.title }}</text>
          <text class="meta">{{ item.event_date }} {{ item.start_time }}-{{ item.end_time }}</text>
          <text v-if="item.location" class="location">{{ item.location }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx; background: $color-bg-page; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: 28rpx; }
.list { display: flex; flex-direction: column; gap: 18rpx; }
.card { display: flex; gap: 20rpx; padding: 26rpx; border-radius: 24rpx; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.dot { flex: 0 0 16rpx; width: 16rpx; height: 16rpx; margin-top: 8rpx; border-radius: 999rpx; }
.info { flex: 1; }
.title { display: block; color: $color-text-primary; font-size: 28rpx; font-weight: 700; }
.meta { display: block; margin-top: 8rpx; color: $color-text-muted; font-size: 24rpx; }
.location { display: block; margin-top: 4rpx; color: $parent-blue; font-size: 22rpx; }
</style>
