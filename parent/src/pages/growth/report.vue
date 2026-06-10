<script setup>
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchGrowthReport } from "@/api/parent-growth"

const report = ref({})
const loading = ref(false)
const reportType = ref("weekly")

const types = [
  { key: "weekly", label: "周报" },
  { key: "monthly", label: "月报" },
  { key: "term", label: "学期" }
]

const periodLabel = computed(() => {
  const p = report.value.period
  if (!p) return ""
  return `${p.start_date} ~ ${p.end_date}`
})

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchGrowthReport({ type: reportType.value })
    report.value = res.data || {}
  } finally { loading.value = false }
}

function switchType(t) {
  if (reportType.value === t.key) return
  reportType.value = t.key
  loadData()
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <!-- 报告类型切换 -->
    <view class="type-tabs">
      <view v-for="t in types" :key="t.key"
        class="type-tab" :class="{ active: reportType === t.key }"
        @click="switchType(t)">
        <text>{{ t.label }}</text>
      </view>
    </view>

    <view v-if="loading" class="state">加载中...</view>
    <template v-else>
      <!-- 学生信息 -->
      <view class="header-card">
        <image v-if="report.student?.avatar" :src="report.student.avatar" class="header-avatar" mode="aspectFill" />
        <view v-else class="header-avatar-placeholder"><text class="hap-icon"></text></view>
        <view class="spacer-w12" />
        <view>
          <text class="header-name">{{ report.student?.name || "" }}</text>
          <text class="header-period">{{ periodLabel }}</text>
        </view>
      </view>

      <!-- 统计概览 -->
      <view class="stats-row">
        <view class="stat-box">
          <text class="stat-num">{{ report.attendance?.rate || "-" }}</text>
          <text class="stat-label">出勤率</text>
        </view>
        <view class="stat-box">
          <text class="stat-num">{{ report.points_earned || 0 }}</text>
          <text class="stat-label">获得积分</text>
        </view>
        <view class="stat-box">
          <text class="stat-num">{{ report.tasks_completed || 0 }}/{{ report.tasks_total || 0 }}</text>
          <text class="stat-label">完成任务</text>
        </view>
        <view class="stat-box">
          <text class="stat-num">{{ report.growth_media_count || 0 }}</text>
          <text class="stat-label">成长记录</text>
        </view>
      </view>

      <!-- 雷达图维度 -->
      <view v-if="report.dimensions?.length" class="card">
        <text class="card-title">能力评估</text>
        <view class="spacer-h16" />
        <view class="dim-list">
          <view v-for="d in report.dimensions" :key="d.name" class="dim-row">
            <text class="dim-name">{{ d.name }}</text>
            <view class="dim-bar-wrap">
              <view class="dim-bar" :style="{ width: (d.score / d.max_score * 100) + '%' }" />
            </view>
            <text class="dim-score">{{ d.score }}</text>
          </view>
        </view>
      </view>

      <!-- 亮点 -->
      <view v-if="report.highlights?.length" class="card" style="margin-top:24rpx">
        <text class="card-title">成长亮点</text>
        <view class="spacer-h12" />
        <view v-for="(h, i) in report.highlights" :key="i" class="hl-item">
          <view class="hl-dot" />
          <text class="hl-text">{{ h }}</text>
        </view>
      </view>

      <!-- 关注 -->
      <view v-if="report.concerns?.length" class="card" style="margin-top:24rpx">
        <text class="card-title">需要关注</text>
        <view class="spacer-h12" />
        <view v-for="(c, i) in report.concerns" :key="i" class="hl-item">
          <view class="hl-dot concern" />
          <text class="hl-text">{{ c }}</text>
        </view>
      </view>

      <!-- 老师评语 -->
      <view v-if="report.teacher_comment" class="card" style="margin-top:24rpx">
        <text class="card-title">老师评语</text>
        <view class="spacer-h12" />
        <view class="teacher-row">
          <view class="teacher-avatar"><text class="ta-icon"></text></view>
          <view class="spacer-w12" />
          <text class="comment-text">{{ report.teacher_comment }}</text>
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx 160rpx; background: $color-bg-page; }
.spacer-w12 { width: 24rpx; flex-shrink: 0; }
.spacer-h12 { height: 24rpx; }
.spacer-h16 { height: 32rpx; }

.type-tabs { display: flex; gap: 16rpx; margin-bottom: 24rpx; }
.type-tab { flex: 1; padding: 20rpx 0; border-radius: 9999rpx; background: $color-bg-card; text-align: center; font-size: 26rpx; color: $color-text-muted; }
.type-tab.active { background: $color-brand-primary; color: #fff; font-weight: 600; }

.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-hint; font-size: 28rpx; }

.header-card { display: flex; align-items: center; padding: 32rpx; border-radius: $border-radius-lg; background: $color-bg-card; box-shadow: $shadow-sm; margin-bottom: 24rpx; }
.header-avatar { width: 128rpx; height: 128rpx; border-radius: 9999rpx; }
.header-avatar-placeholder { width: 128rpx; height: 128rpx; border-radius: 9999rpx; background: #DBEAFE; display: flex; align-items: center; justify-content: center; }
.hap-icon { font-family: remixicon; font-size: 48rpx; color: #2563EB; }
.header-name { display: block; font-size: 36rpx; font-weight: 700; color: $color-text-primary; }
.header-period { display: block; margin-top: 8rpx; font-size: 24rpx; color: $color-text-hint; }

.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12rpx; margin-bottom: 24rpx; }
.stat-box { padding: 24rpx 8rpx; border-radius: $border-radius-md; background: $color-bg-card; text-align: center; box-shadow: $shadow-sm; }
.stat-num { display: block; font-size: 32rpx; font-weight: 700; color: $color-text-primary; }
.stat-label { display: block; margin-top: 8rpx; font-size: 22rpx; color: $color-text-hint; }

.card { padding: 32rpx; border-radius: $border-radius-lg; background: $color-bg-card; box-shadow: $shadow-sm; }
.card-title { font-size: 28rpx; font-weight: 600; color: $color-text-primary; }

.dim-list { display: flex; flex-direction: column; gap: 16rpx; }
.dim-row { display: flex; align-items: center; }
.dim-name { width: 80rpx; font-size: 24rpx; color: $color-text-primary; flex-shrink: 0; }
.dim-bar-wrap { flex: 1; height: 16rpx; margin: 0 16rpx; border-radius: 9999rpx; background: #F1F5F9; overflow: hidden; }
.dim-bar { height: 100%; border-radius: 9999rpx; background: linear-gradient(90deg, #FB923C, #F97316); }
.dim-score { width: 60rpx; font-size: 24rpx; font-weight: 600; color: $color-brand-primary; text-align: right; }

.hl-item { display: flex; align-items: flex-start; margin-bottom: 12rpx; }
.hl-dot { width: 16rpx; height: 16rpx; border-radius: 9999rpx; background: #16A34A; margin-top: 8rpx; flex-shrink: 0; margin-right: 12rpx; }
.hl-dot.concern { background: #F59E0B; }
.hl-text { font-size: 26rpx; color: $color-text-body; line-height: 1.5; flex: 1; }

.teacher-row { display: flex; }
.teacher-avatar { width: 80rpx; height: 80rpx; border-radius: 9999rpx; background: #DBEAFE; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.ta-icon { font-family: remixicon; font-size: 36rpx; color: #2563EB; }
.comment-text { font-size: 26rpx; color: $color-text-body; line-height: 1.6; flex: 1; }
</style>
