<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { getUser } from "@/utils/auth"
import { ensureLogin } from "@/utils/login"

const user = ref(null)

const menuItems = [
  { title: "我的孩子", icon: "child", path: "/pages/student/index" },
  { title: "班级信息", icon: "class", path: "/pages/class/index" },
  { title: "通知公告", icon: "notice", path: "/pages/notice/index" },
  { title: "请假记录", icon: "leave", path: "/pages/leave/index" },
  { title: "接送管理", icon: "pickup", path: "/pages/pickup/index" },
  { title: "亲子任务", icon: "task", path: "/pages/tasks/index" },
  { title: "积分成就", icon: "points", path: "/pages/points/index" },
  { title: "意见反馈", icon: "feedback", path: "/pages/feedback/index" },
  { title: "设置", icon: "settings", path: "/pages/settings/index" }
]

onLoad(async () => {
  await ensureLogin()
  user.value = getUser()
})

function goPage(path) {
  uni.navigateTo({ url: path })
}
</script>

<template>
  <view class="page">
    <view class="user-card">
      <view class="avatar-placeholder">
        <text class="avatar-text">{{ (user?.nickname || '家')[0] }}</text>
      </view>
      <text class="name">{{ user?.nickname || "家长" }}</text>
      <text class="phone">{{ user?.phone || "微信用户" }}</text>
    </view>

    <view class="menu-list">
      <view v-for="item in menuItems" :key="item.path" class="menu-item" @click="goPage(item.path)">
        <text class="menu-title">{{ item.title }}</text>
        <text class="arrow">›</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg 28rpx; background: $color-bg-page; }
.user-card { display: flex; flex-direction: column; align-items: center; padding: 40rpx; margin-bottom: 24rpx; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: $shadow-sm; }
.avatar-placeholder { display: flex; align-items: center; justify-content: center; width: 120rpx; height: 120rpx; border-radius: 999rpx; background: linear-gradient(135deg, #FB923C, #F97316); }
.avatar-text { color: #fff; font-size: 48rpx; font-weight: 700; }
.name { margin-top: 18rpx; color: $color-text-primary; font-size: $typography-font-size-xl; font-weight: 700; }
.phone { margin-top: 8rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
.menu-list { display: flex; flex-direction: column; border-radius: $border-radius-xl; overflow: hidden; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.menu-item { display: flex; justify-content: space-between; align-items: center; padding: 28rpx; border-bottom: 1rpx solid rgba(106,77,42,0.06); }
.menu-item:last-child { border-bottom: 0; }
.menu-title { color: $color-text-primary; font-size: $typography-font-size-md; }
.arrow { color: $color-text-muted; font-size: $typography-font-size-2xl; }
</style>
