<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchSettings, updateSettings } from "@/api/parent-settings"
import { clearAuth } from "@/utils/auth"

const settings = ref({ notification: {}, privacy: {} })
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchSettings()
    settings.value = res.data || { notification: {}, privacy: {} }
  } finally { loading.value = false }
}

async function toggleNotification(key) {
  const updated = { ...settings.value.notification, [key]: !settings.value.notification[key] }
  try {
    await updateSettings({ notification: updated, privacy: settings.value.privacy })
    settings.value.notification = updated
    uni.showToast({ title: "已更新", icon: "success" })
  } catch (err) {
    uni.showToast({ title: err?.message || "更新失败", icon: "none" })
  }
}

function handleLogout() {
  uni.showModal({
    title: "退出登录",
    content: "确定要退出登录吗?",
    success(res) {
      if (res.confirm) {
        clearAuth()
        uni.reLaunch({ url: "/pages/auth/phone" })
      }
    }
  })
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>
    <template v-else>
      <view class="card">
        <text class="section-title">消息通知</text>
        <view v-for="(val, key) in settings.notification" :key="key" class="toggle-row">
          <text class="label">{{ { notice_push: '通知推送', leave_push: '请假通知', pickup_push: '接送通知', growth_push: '成长记录', task_push: '任务提醒', announcement_push: '公告通知' }[key] || key }}</text>
          <switch :checked="val" @change="toggleNotification(key)" color="#F97316" />
        </view>
      </view>
      <button class="logout-btn" @click="handleLogout">退出登录</button>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg 28rpx; background: $color-bg-page; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }
.card { padding: 28rpx; margin-bottom: 18rpx; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.section-title { display: block; margin-bottom: 18rpx; color: $color-text-primary; font-size: $typography-font-size-md; font-weight: 700; }
.toggle-row { display: flex; justify-content: space-between; align-items: center; padding: 20rpx 0; border-bottom: 1rpx solid rgba(106,77,42,0.06); }
.label { color: $color-text-primary; font-size: $typography-font-size-base; }
.logout-btn { width: 100%; height: 88rpx; margin-top: 24rpx; border-radius: 999rpx; background: #fff0ef; color: #e6534f; font-size: $typography-font-size-md; font-weight: 700; line-height: 88rpx; }
</style>
