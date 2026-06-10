<script setup>
import { computed } from "vue"

const props = defineProps({
  active: { type: Number, default: 0 },
  unreadCount: { type: Number, default: 0 },
})

const tabs = [
  { label: "首页", icon: "首", path: "/pages/index/index" },
  { label: "班级", icon: "班", path: "/pages/class/index" },
  { label: "消息", icon: "消", path: "/pages/messages/index" },
  { label: "我的", icon: "我", path: "/pages/profile/index" },
]

const showBadge = computed(() => props.unreadCount > 0)

function onTab(tab, index) {
  if (index === props.active) {
    uni.pageScrollTo({ scrollTop: 0, duration: 300 })
    return
  }
  uni.navigateTo({ url: tab.path })
}
</script>

<template>
  <view class="bottom-nav">
    <view
      v-for="(tab, index) in tabs"
      :key="tab.path"
      class="nav-item"
      :class="{ active: active === index }"
      @click="onTab(tab, index)"
    >
      <view v-if="index === 2" class="msg-icon-wrap">
        <text class="nav-icon" :class="{ 'active-icon': active === index }">{{ tab.icon }}</text>
        <view v-if="showBadge" class="msg-badge" />
      </view>
      <text v-else class="nav-icon" :class="{ 'active-icon': active === index }">{{ tab.icon }}</text>
      <text class="nav-label" :class="{ 'active-label': active === index }">{{ tab.label }}</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
$blue-500: #3B82F6;

.bottom-nav {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  display: flex;
  justify-content: space-between;
  max-width: 430px;
  margin: 0 auto;
  padding: 24rpx 46rpx calc(24rpx + env(safe-area-inset-bottom));
  background: #fff;
  box-shadow: 0 -2rpx 20rpx rgba(0, 0, 0, 0.06);
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
  padding: 8rpx 0;
}

.nav-icon {
  font-size: 48rpx;
  font-weight: 600;
  color: #94A3B8;
}

.active-icon {
  color: $blue-500;
}

.nav-label {
  font-size: 22rpx;
  color: #94A3B8;
}

.active-label {
  color: $blue-500;
  font-weight: 600;
}

.msg-icon-wrap {
  position: relative;
}

.msg-badge {
  position: absolute;
  top: -4rpx;
  right: -8rpx;
  width: 18rpx;
  height: 16rpx;
  background: #EF4444;
  border-radius: 9999rpx;
}
</style>
