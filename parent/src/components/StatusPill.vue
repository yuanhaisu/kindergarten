<template>
  <view class="status-pill" :class="'is-' + tone">
    <text class="status-text"><slot>{{ label }}</slot></text>
  </view>
</template>

<script setup>
/**
 * StatusPill — 状态标签
 *
 * tone 对应 design token 的语义色（含浅底），不接受自定义色值。
 */
defineProps({
  tone: { type: String, default: "info" }, // success | warning | danger | info
  label: { type: String, default: "" }
})
</script>

<style scoped lang="scss">
.status-pill {
  display: inline-flex;
  align-items: center;
  padding: $component-badge-padding-x;
  border-radius: $component-badge-radius;
  font-size: $component-badge-font-size;
  font-weight: 700;
  line-height: 1.4;
}

.is-success { background: $color-semantic-success-bg; color: $color-semantic-success; }
.is-warning { background: $color-semantic-warning-bg; color: $color-semantic-warning; }
.is-danger  { background: $color-semantic-danger-bg;  color: $color-semantic-danger; }
.is-info    { background: $color-semantic-info-bg;    color: $color-semantic-info; }
</style>
