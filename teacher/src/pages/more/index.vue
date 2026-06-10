<template>
  <view class="page">
    <view class="card" @click="uni.navigateTo({ url: '/pages/notice/create' })">发布通知</view>
    <view class="card" @click="uni.navigateTo({ url: '/pages/growth/index' })">成长记录</view>
    <view class="card" @click="uni.navigateTo({ url: '/pages/pickup/index' })">接送管理</view>
    <view class="card" @click="uni.navigateTo({ url: '/pages/calendar/index' })">班级日历</view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 28rpx; background: $teacher-bg; }
.card { margin-bottom: 18rpx; padding: 30rpx; border-radius: 30rpx; background: $teacher-card; color: $teacher-text; font-size: 30rpx; font-weight: 800; box-shadow: $teacher-shadow; }
</style>
