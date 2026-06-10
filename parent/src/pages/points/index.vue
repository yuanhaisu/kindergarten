<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchPointsOverview, fetchLeaderboard, fetchAchievements } from "@/api/parent-points"

const overview = ref({})
const leaderboard = ref([])
const achievements = ref({ earned: [], earned_count: 0 })
const tab = ref("overview")

async function loadData() {
  try {
    await ensureLogin()
    const [o, l, a] = await Promise.all([
      fetchPointsOverview(1),
      fetchLeaderboard({ student_id: 1 }),
      fetchAchievements(1)
    ])
    overview.value = o.data || {}
    leaderboard.value = l.data?.list || []
    achievements.value = a.data || { earned: [], earned_count: 0 }
  } catch (err) {
    uni.showToast({ title: err?.message || "加载失败", icon: "none" })
  }
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <view class="hero-card">
      <text class="level">{{ overview.level_name || "新手上路" }}</text>
      <text class="points">{{ overview.total_points || 0 }} 积分</text>
      <text class="rank">班级排名 {{ overview.rank_in_class || "--" }}/{{ overview.class_total || "--" }}</text>
    </view>

    <view class="tabs">
      <button class="tab" :class="{ active: tab === 'overview' }" @click="tab = 'overview'">积分</button>
      <button class="tab" :class="{ active: tab === 'leaderboard' }" @click="tab = 'leaderboard'">排行榜</button>
      <button class="tab" :class="{ active: tab === 'achievements' }" @click="tab = 'achievements'">成就</button>
    </view>

    <view v-if="tab === 'overview'" class="card">
      <view class="stat-row"><text>本周积分</text><text class="val">{{ overview.this_week || 0 }}</text></view>
      <view class="stat-row"><text>本月积分</text><text class="val">{{ overview.this_month || 0 }}</text></view>
    </view>

    <view v-if="tab === 'leaderboard'" class="list">
      <view v-for="item in leaderboard" :key="item.rank" class="rank-item" :class="{ self: item.is_self }">
        <text class="rank-num">{{ item.rank }}</text>
        <text class="rank-name">{{ item.student?.name }}</text>
        <text class="rank-points">{{ item.points }}分</text>
      </view>
    </view>

    <view v-if="tab === 'achievements'" class="list">
      <text class="section-title">已获得 ({{ achievements.earned_count }})</text>
      <view v-for="a in achievements.earned" :key="a.id" class="ach-card earned">
        <text class="ach-name">{{ a.name }}</text>
        <text class="ach-desc">{{ a.description }}</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx; background: $color-bg-page; }
.hero-card { padding: 40rpx; border-radius: 32rpx; background: linear-gradient(135deg, #FB923C, #F97316); text-align: center; margin-bottom: 24rpx; }
.level { display: block; color: #FFEDD5; font-size: 24rpx; }
.points { display: block; margin-top: 12rpx; color: #fff; font-size: 56rpx; font-weight: 800; }
.rank { display: block; margin-top: 8rpx; color: #FFEDD5; font-size: 22rpx; }
.tabs { display: flex; gap: 16rpx; margin: 24rpx 0; }
.tab { flex: 1; height: 72rpx; border-radius: 999rpx; background: $color-bg-card; color: $color-text-muted; font-size: 26rpx; line-height: 72rpx; }
.tab.active { background: $color-brand-primary; color: #fff; font-weight: 700; }
.card { padding: 28rpx; border-radius: 24rpx; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.stat-row { display: flex; justify-content: space-between; padding: 20rpx 0; border-bottom: 1rpx solid rgba(106,77,42,0.06); color: $color-text-primary; font-size: 28rpx; }
.stat-row .val { color: $color-brand-primary; font-weight: 700; }
.list { display: flex; flex-direction: column; gap: 12rpx; }
.section-title { display: block; margin-bottom: 12rpx; color: $color-text-primary; font-size: 28rpx; font-weight: 700; }
.rank-item { display: flex; align-items: center; gap: 16rpx; padding: 20rpx 24rpx; border-radius: 16rpx; background: $color-bg-card; }
.rank-item.self { background: #fff4df; border: 2rpx solid $color-brand-primary; }
.rank-num { width: 48rpx; text-align: center; color: $color-text-muted; font-size: 26rpx; font-weight: 700; }
.rank-name { flex: 1; color: $color-text-primary; font-size: 26rpx; }
.rank-points { color: $color-brand-primary; font-size: 24rpx; font-weight: 700; }
.ach-card { padding: 24rpx; border-radius: 20rpx; background: $color-bg-card; }
.ach-card.earned { background: #fffaf0; border: 2rpx solid #ffda73; }
.ach-name { display: block; color: $color-text-primary; font-size: 26rpx; font-weight: 700; }
.ach-desc { display: block; margin-top: 6rpx; color: $color-text-muted; font-size: 22rpx; }
</style>
