<script setup>
defineProps({
  active: { type: Number, default: 0 },
})

const tabs = [
  { label: "首页", icon: "&#xee1c;", path: "/pages/index/index" },
  { label: "成长", icon: "&#xeda5;", path: "/pages/growth/index" },
  { label: "任务", icon: "&#xf1e8;", path: "/pages/tasks/index" },
  { label: "我的", icon: "&#xf264;", path: "/pages/profile/index" },
]

function onTab(tab, index) {
  if (index === tab.active) return
  uni.switchTab({ url: tab.path })
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
      <text class="nav-icon" v-html="tab.icon" />
      <text class="nav-label">{{ tab.label }}</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.bottom-nav {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  display: flex;
  justify-content: space-around;
  padding: $spacing-lg 48rpx calc(24rpx + env(safe-area-inset-bottom));
  background: #fff;
  box-shadow: 0 -2rpx 20rpx rgba(0, 0, 0, 0.06);
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
}

.nav-icon {
  font-size: 48rpx;
  color: #94A3B8;
}

.nav-label {
  font-size: $typography-font-size-xs;
  color: #94A3B8;
}

.nav-item.active {
  .nav-icon {
    color: $color-brand-primary;
  }
  .nav-label {
    color: $color-brand-primary;
    font-weight: 500;
  }
}
</style>
