<template>
  <view class="app-card" :class="['is-' + variant, customClass]">
    <slot />
  </view>
</template>

<script setup>
/**
 * AppCard — 卡片容器
 *
 * 圆角/阴影/内边距全部走 design token，页面只管内容。
 * variant=soft 用于嵌套卡片或弱化层级（更轻的阴影）。
 */
defineProps({
  variant: { type: String, default: "default" }, // default | soft
  customClass: { type: String, default: "" }
})
</script>

<style scoped lang="scss">
.app-card {
  padding: $component-card-padding;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.app-card.is-soft {
  box-shadow: $shadow-sm;
}
</style>
