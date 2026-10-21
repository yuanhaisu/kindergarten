<template>
  <view class="brand-block">
    <image class="brand-logo" src="/static/logo.png" mode="aspectFit" />
    <text class="brand-name">{{ name }}</text>
    <text class="brand-tagline">{{ tagline }}</text>
  </view>
</template>

<script setup>
/**
 * BrandBlock — 登录页品牌区
 *
 * 两端共用同一个标识、同一句 tagline、同一字阶，
 * 否则家长和老师并排看手机会以为是两个 App（2026-10-08 登录页评审 P1）。
 *
 * logo 由各端 /static/logo.png 提供（shared-ui 不带静态资源）。
 */
defineProps({
  name: { type: String, default: "养成星球" },
  tagline: { type: String, default: "幼儿园数字化管理平台" }
})
</script>

<style scoped lang="scss">
.brand-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: $spacing-xl;
}

.brand-logo {
  width: 160rpx;
  height: 160rpx;
  border-radius: $border-radius-full;
  box-shadow: $shadow-md;
}

.brand-name {
  font-size: $typography-font-size-3xl;
  font-weight: 700;
  color: $color-text-primary;
  letter-spacing: 0.08em;
}

.brand-tagline {
  font-size: $typography-font-size-md;
  color: $color-text-muted;
}
</style>
