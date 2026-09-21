<script setup>
import { computed } from "vue"
import { getUser } from "@/utils/auth"

// 老师端没有 tabBar（见 src/pages.json），首页工作台只放得下 8 个磁贴，
// 这里是完整入口的聚合页。
//
// 晨检只对 doctor 角色开放：Casbin 策略里 teacher 角色**没有**
// /api/v1/doctor/* 权限（见 server/configs/rbac_policy.csv），
// 给普通老师显示这个入口只会让他们撞 403。
const isDoctor = computed(() => {
  const roles = (getUser() && getUser().roles) || []
  return roles.some((r) => r.role === "doctor")
})

const items = computed(() => {
  const list = [
    { title: "发布通知", desc: "向班级家长发送通知", path: "/pages/notice/create" },
    { title: "成长记录", desc: "查看与添加成长记录", path: "/pages/growth/index" },
    { title: "接送管理", desc: "扫码或手动登记接送", path: "/pages/pickup/index" },
    { title: "接送记录", desc: "查看历史接送明细", path: "/pages/pickup/records" },
    { title: "班级日历", desc: "班级活动与安排", path: "/pages/calendar/index" },
    { title: "班级相册", desc: "班级照片集", path: "/pages/album/index" },
    { title: "请假审批", desc: "处理家长请假申请", path: "/pages/leave/index" },
    { title: "亲子任务", desc: "发布与查看亲子任务", path: "/pages/tasks/index" },
    { title: "消息中心", desc: "通知、请假与系统消息", path: "/pages/messages/index" },
  ]
  if (isDoctor.value) {
    list.push({ title: "入园晨检", desc: "录入晨检结果", path: "/pages/doctor/checkin" })
  }
  return list
})

function go(path) {
  uni.navigateTo({ url: path })
}
</script>

<template>
  <view class="page">
    <view v-for="item in items" :key="item.path" class="card" @click="go(item.path)">
      <view class="card-text">
        <text class="card-title">{{ item.title }}</text>
        <text class="card-desc">{{ item.desc }}</text>
      </view>
      <text class="card-arrow">›</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 28rpx; background: $teacher-bg; }

.card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18rpx;
  padding: 30rpx;
  border-radius: 30rpx;
  background: $teacher-card;
  box-shadow: $teacher-shadow;
}

.card-text { display: flex; flex-direction: column; gap: $spacing-xs; }
.card-title { color: $teacher-text; font-size: $typography-font-size-lg; font-weight: 800; }
.card-desc { color: $teacher-muted; font-size: $typography-font-size-sm; }
.card-arrow { color: $color-text-hint; font-size: $typography-font-size-2xl; }
</style>
