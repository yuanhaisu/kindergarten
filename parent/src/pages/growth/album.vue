<script setup>
import { ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchGrowthAlbum } from "@/api/parent-growth"

const groups = ref([])
const total = ref(0)
const loading = ref(false)
const year = ref(new Date().getFullYear())
const month = ref(new Date().getMonth() + 1)

const months = Array.from({ length: 12 }, (_, i) => ({ value: i + 1, label: (i + 1) + "月" }))

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchGrowthAlbum({ year: year.value, month: month.value })
    groups.value = res.data?.groups || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

function prevMonth() {
  if (month.value === 1) { year.value--; month.value = 12 }
  else month.value--
  loadData()
}
function nextMonth() {
  if (month.value === 12) { year.value++; month.value = 1 }
  else month.value++
  loadData()
}

function preview(urls, current) {
  uni.previewImage({ urls, current })
}

onLoad(loadData)
onPullDownRefresh(loadData)
</script>

<template>
  <view class="page">
    <!-- 月份切换 -->
    <view class="month-bar">
      <view class="month-arrow" @click="prevMonth"><text class="arrow-icon"></text></view>
      <text class="month-text">{{ year }}年{{ month }}月</text>
      <view class="month-arrow" @click="nextMonth"><text class="arrow-icon"></text></view>
      <view class="spacer-flex" />
      <text class="count-text">{{ total }}张照片</text>
    </view>

    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="groups.length === 0" class="state">本月暂无照片</view>
    <view v-else>
      <view v-for="group in groups" :key="group.date" class="group">
        <text class="group-date">{{ group.date }}</text>
        <view class="photo-grid">
          <image v-for="m in group.media" :key="m.id"
            :src="m.thumb_url" mode="aspectFill" class="photo"
            @click="preview(group.media.map(x => x.file_url), m.file_url)" />
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx calc(150rpx + env(safe-area-inset-bottom)); background: $color-bg-page; }
.spacer-flex { flex: 1; }

.month-bar { display: flex; align-items: center; margin-bottom: 24rpx; }
.month-arrow { width: 64rpx; height: 64rpx; border-radius: 9999rpx; background: $color-bg-card; display: flex; align-items: center; justify-content: center; }
.arrow-icon { font-family: remixicon; font-size: 32rpx; color: $color-text-primary; }
.month-text { margin: 0 24rpx; font-size: 32rpx; font-weight: 600; color: $color-text-primary; }
.count-text { font-size: 24rpx; color: $color-text-hint; }

.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-hint; font-size: 28rpx; }

.group { margin-bottom: 32rpx; }
.group-date { display: block; font-size: 28rpx; font-weight: 600; color: $color-text-primary; margin-bottom: 16rpx; }

.photo-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8rpx; }
.photo { width: 100%; aspect-ratio: 1; border-radius: 12rpx; }
</style>
