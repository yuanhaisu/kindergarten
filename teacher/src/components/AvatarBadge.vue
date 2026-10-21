<template>
  <view class="avatar-badge" :style="boxStyle">
    <image
      v-if="src"
      :src="src"
      class="avatar-img"
      mode="aspectFill"
    />
    <text v-else class="avatar-placeholder">{{ placeholder }}</text>
    <view v-if="dot" class="avatar-dot" :class="'dot-' + dotColor" />
  </view>
</template>

<script setup>
/**
 * AvatarBadge — 头像（可带角标）
 *
 * size 单位 px（与 tokens.json 一致），组件内部换算 rpx。
 * 角标颜色走语义色 token。
 */
import { computed } from "vue"

const props = defineProps({
  src: { type: String, default: "" },
  size: { type: Number, default: 48 },      // px，构建时 ×2 = rpx
  placeholder: { type: String, default: "?" },
  dot: { type: Boolean, default: false },
  dotColor: { type: String, default: "success" } // success | warning | danger | info
})

const boxStyle = computed(() => {
  const rpx = props.size * 2
  return {
    width: rpx + "rpx",
    height: rpx + "rpx",
    borderRadius: rpx + "rpx",
    fontSize: props.size * 0.8 + "rpx"
  }
})
</script>

<style scoped lang="scss">
.avatar-badge {
  position: relative;
  flex-shrink: 0;
  overflow: visible;
  background: $gradient-brand-soft;
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-img {
  width: 100%;
  height: 100%;
  border-radius: $border-radius-full;
}

.avatar-placeholder {
  color: $color-brand-primary-dark;
  font-weight: 700;
}

.avatar-dot {
  position: absolute;
  top: 0;
  right: 0;
  width: 18rpx;
  height: 18rpx;
  border-radius: $border-radius-full;
  border: 3rpx solid $color-bg-card;
}
.dot-success { background: $color-semantic-success; }
.dot-warning { background: $color-semantic-warning; }
.dot-danger  { background: $color-semantic-danger; }
.dot-info    { background: $color-semantic-info; }
</style>
