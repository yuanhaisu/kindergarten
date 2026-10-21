<template>
  <button
    class="app-button"
    :class="['is-' + type, 'is-' + size, { 'is-block': block, 'is-disabled': disabled }]"
    :disabled="disabled"
    @click="$emit('click', $event)"
  >
    <slot />
  </button>
</template>

<script setup>
/**
 * AppButton — 统一按钮
 *
 * primary 用品牌渐变（两端各自品牌色，由 $gradient-brand 提供），
 * ghost 为浅色底 + 品牌文字。
 */
defineProps({
  type: { type: String, default: "primary" }, // primary | ghost
  size: { type: String, default: "md" },      // sm | md | lg
  block: { type: Boolean, default: true },
  disabled: { type: Boolean, default: false }
})
defineEmits(["click"])
</script>

<style scoped lang="scss">
.app-button {
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  font-weight: 700;
  letter-spacing: 0.02em;
  transition: opacity $animation-duration-fast $animation-easing-default;
}

.app-button.is-block { width: 100%; }

/* 尺寸 */
.is-sm { height: 64rpx; padding: 0 $spacing-xl; border-radius: $border-radius-full; font-size: $typography-font-size-sm; }
.is-md { height: 88rpx; padding: 0 $spacing-xl; border-radius: $border-radius-full; font-size: $typography-font-size-base; }
.is-lg { height: 96rpx; padding: 0 $spacing-xl; border-radius: $border-radius-full; font-size: $typography-font-size-lg; }

/* 类型 */
.is-primary {
  background: $gradient-brand;
  color: $color-text-on-brand;
  box-shadow: $shadow-md;
}
.is-ghost {
  background: $gradient-brand-soft;
  color: $color-brand-primary-dark;
  box-shadow: $shadow-sm;
}

.is-disabled { opacity: 0.55; }
</style>
