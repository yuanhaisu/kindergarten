<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchClassInfo } from "@/api/parent-class"

const info = ref({})
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchClassInfo(1)
    info.value = res.data || {}
  } finally { loading.value = false }
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>
    <template v-else>
      <view class="card">
        <text class="title">{{ info.name || "班级" }}</text>
        <text class="sub">{{ info.grade }} · {{ info.academic_year }} · {{ info.student_count }}人</text>
      </view>
      <view class="card">
        <text class="section-title">班级老师</text>
        <view v-for="t in info.teachers || []" :key="t.id" class="teacher">
          <text class="t-name">{{ t.nickname }}</text>
          <text class="t-role">{{ t.title_label }}</text>
          <text class="t-phone">{{ t.phone }}</text>
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg 28rpx; background: $color-bg-page; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }
.card { padding: 28rpx; margin-bottom: 18rpx; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.title { display: block; color: $color-text-primary; font-size: $typography-font-size-2xl; font-weight: 800; }
.sub { display: block; margin-top: 8rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
.section-title { display: block; margin-bottom: 18rpx; color: $color-text-primary; font-size: $typography-font-size-md; font-weight: 700; }
.teacher { padding: 18rpx 0; border-bottom: 1rpx solid rgba(106,77,42,0.06); }
.t-name { display: block; color: $color-text-primary; font-size: $typography-font-size-md; font-weight: 700; }
.t-role { display: block; margin-top: 4rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
.t-phone { display: block; margin-top: 4rpx; color: $parent-blue; font-size: $typography-font-size-sm; }
</style>
