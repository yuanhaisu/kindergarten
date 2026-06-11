<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { fetchTaskDetail, checkinTask, fetchCheckinHistory } from "@/api/parent-task"

const detail = ref({})
const history = ref([])
const checkinContent = ref("")
const loading = ref(false)

async function loadData(id) {
  loading.value = true
  try {
    const [d, h] = await Promise.all([fetchTaskDetail(id), fetchCheckinHistory(id)])
    detail.value = d.data || {}
    history.value = h.data?.list || []
  } finally { loading.value = false }
}

async function handleCheckin() {
  const content = checkinContent.value.trim()
  if (!content) {
    uni.showToast({ title: "请填写打卡内容", icon: "none" })
    return
  }
  try {
    const res = await checkinTask(detail.value.id, { student_id: 1, content })
    checkinContent.value = ""
    detail.value.today_checked_in = true
    detail.value.streak_days = res.data?.streak_days
    uni.showToast({ title: res.data?.message || "打卡成功", icon: "success" })
    loadData(detail.value.id)
  } catch (err) {
    uni.showToast({ title: err?.message || "打卡失败", icon: "none" })
  }
}

onLoad((opts) => { if (opts?.id) loadData(opts.id) })
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>
    <template v-else>
      <view class="card">
        <text class="title">{{ detail.title }}</text>
        <text class="desc">{{ detail.description }}</text>
        <text class="intro">{{ detail.intro }}</text>
        <view v-if="detail.requirements?.length" class="reqs">
          <text v-for="(r, i) in detail.requirements" :key="i" class="req">{{ i + 1 }}. {{ r }}</text>
        </view>
        <view class="stats">
          <text>连续 {{ detail.streak_days }} 天</text>
          <text>完成率 {{ detail.my_completion_rate }}</text>
          <text>+{{ detail.points }}积分/次</text>
        </view>
      </view>

      <view v-if="!detail.today_checked_in" class="checkin-card">
        <text class="section-title">今日打卡</text>
        <textarea v-model="checkinContent" class="textarea" placeholder="记录今天的内容..." :maxlength="500" />
        <button class="checkin-btn" @click="handleCheckin">打卡</button>
      </view>

      <view class="history-section">
        <text class="section-title">打卡历史</text>
        <view v-for="h in history" :key="h.id" class="history-item">
          <text class="h-date">{{ h.date }}</text>
          <text class="h-content">{{ h.content }}</text>
          <text class="h-points">+{{ h.points }}</text>
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg 28rpx 48rpx; background: $color-bg-page; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }
.card { padding: $spacing-xl; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.title { display: block; color: $color-text-primary; font-size: $typography-font-size-2xl; font-weight: 800; }
.desc { display: block; margin-top: 12rpx; color: $color-text-primary; font-size: $typography-font-size-base; }
.intro { display: block; margin-top: 16rpx; color: $color-text-muted; font-size: $typography-font-size-sm; line-height: 1.6; }
.reqs { margin-top: 16rpx; }
.req { display: block; margin-top: 6rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
.stats { display: flex; gap: $spacing-lg; margin-top: 20rpx; color: $parent-blue; font-size: $typography-font-size-sm; }
.checkin-card { margin-top: 24rpx; padding: 28rpx; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.section-title { display: block; margin-bottom: 16rpx; color: $color-text-primary; font-size: $typography-font-size-md; font-weight: 700; }
.textarea { width: 100%; height: 160rpx; padding: 20rpx; border: 2rpx solid rgba(106,77,42,0.12); border-radius: $border-radius-lg; font-size: $typography-font-size-base; box-sizing: border-box; }
.checkin-btn { width: 100%; height: 80rpx; margin-top: 20rpx; border-radius: 999rpx; background: $color-brand-primary; color: #fff; font-size: $typography-font-size-md; font-weight: 700; line-height: 80rpx; }
.history-section { margin-top: 24rpx; }
.history-item { display: flex; align-items: center; gap: $spacing-md; padding: 20rpx; margin-bottom: 12rpx; border-radius: $border-radius-lg; background: $color-bg-card; }
.h-date { flex: 0 0 140rpx; color: $color-text-muted; font-size: $typography-font-size-xs; }
.h-content { flex: 1; color: $color-text-primary; font-size: $typography-font-size-sm; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.h-points { color: $color-brand-primary; font-size: $typography-font-size-sm; font-weight: 700; }
</style>
