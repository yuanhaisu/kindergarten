<script setup>
import { computed } from "vue"
import { goTab } from "@/utils/nav"

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
const badgeText = computed(() => (props.unreadCount > 99 ? "99+" : String(props.unreadCount)))

function onTab(tab, index) {
  // 点当前 tab 时 goTab 内部会滚动置顶，与原来的行为一致
  if (index === props.active) {
    uni.pageScrollTo({ scrollTop: 0, duration: 300 })
    return
  }
  goTab(tab.path)
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
        <view v-if="showBadge" class="msg-badge">
          <text class="msg-badge-text">{{ badgeText }}</text>
        </view>
      </view>
      <text v-else class="nav-icon" :class="{ 'active-icon': active === index }">{{ tab.icon }}</text>
      <text class="nav-label" :class="{ 'active-label': active === index }">{{ tab.label }}</text>
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
  justify-content: space-between;
  max-width: 430px;
  margin: 0 auto;
  padding: $spacing-lg 46rpx calc($spacing-lg + env(safe-area-inset-bottom));
  background: $color-bg-card;
  box-shadow: 0 -2rpx 20rpx rgba(120, 53, 15, 0.06);
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
  padding: $spacing-xs 0;
}

.nav-icon {
  font-size: 48rpx;
  font-weight: 600;
  color: $color-text-hint;
}

// 选中态用品牌色。此前误用 semantic-info，导致老师端（暖橙）的底部导航
// 出现蓝色选中，违反 DESIGN.md 的 Front-Back Rule；家长端同位置是橙色。
.active-icon {
  color: $color-brand-primary;
}

.nav-label {
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.active-label {
  color: $color-brand-primary;
  font-weight: 600;
}

.msg-icon-wrap {
  position: relative;
}

.msg-badge {
  position: absolute;
  top: -8rpx;
  right: -14rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 28rpx;
  height: 28rpx;
  padding: 0 6rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
  box-sizing: border-box;
}

.msg-badge-text {
  font-size: 18rpx;
  font-weight: 700;
  line-height: 1;
  color: $color-text-on-brand;
}
</style>
