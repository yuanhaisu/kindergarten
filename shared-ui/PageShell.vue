<template>
  <view class="page-shell" :style="rootStyle">
    <view
      class="page-shell__body"
      :class="{ 'is-padded': padded, 'is-flush-top': flushTop }"
      :style="bodyStyle"
    >
      <slot />
    </view>
    <view class="page-shell__bottom" :style="bottomStyle">
      <slot name="bottom" />
    </view>
  </view>
</template>

<script setup>
import { computed } from "vue"

/**
 * PageShell — 原生导航下的页面骨架
 *
 * 负责三件容易各页重复计算的事：页面底色 / 水平内边距、底部安全区、
 * 固定底栏的占位高度。垂直节奏仍由页面内容自己决定（flushTop 用于
 * 首块自带间距的页面）。
 *
 * 安全区只在本组件算一次：body 的 padding-bottom 由 bottomSpace +
 * env(safe-area-inset-bottom) 组成；bottom slot 里的底栏（如 BottomNav）
 * 自己画背景与安全区，页面不要再额外补 padding。
 */
const props = defineProps({
  // 是否套用页面级 padding（水平 page-padding-x + 顶部 page-padding-y）
  padded: { type: Boolean, default: true },
  // 首块内容自己带顶部间距时置 true，只去掉顶部 padding
  flushTop: { type: Boolean, default: false },
  // 覆盖页面底色；默认 $color-bg-page
  background: { type: String, default: "" },
  // 可选：PC / 平板上的内容宽度上限，如 "430px"
  maxWidth: { type: String, default: "" },
  // 固定底栏高度，如 "180rpx"；body 会预留该高度
  bottomSpace: { type: String, default: "" },
  // 是否在预留高度之外再让出底部安全区
  safeBottom: { type: Boolean, default: true }
})

const rootStyle = computed(() => {
  const style = {}
  if (props.background) style.background = props.background
  if (props.maxWidth) style.maxWidth = props.maxWidth
  return style
})

const bodyStyle = computed(() => {
  const space = props.bottomSpace || "0rpx"
  return {
    paddingBottom: props.safeBottom ? `calc(${space} + env(safe-area-inset-bottom))` : space
  }
})

const bottomStyle = computed(() => (props.maxWidth ? { maxWidth: props.maxWidth } : {}))
</script>

<style scoped lang="scss">
.page-shell {
  box-sizing: border-box;
  width: 100%;
  min-height: 100vh;
  margin: 0 auto;
  background: $color-bg-page;
}

.page-shell__body {
  box-sizing: border-box;
}

.page-shell__body.is-padded {
  padding-top: $component-page-padding-y;
  padding-right: $component-page-padding-x;
  padding-left: $component-page-padding-x;
}

.page-shell__body.is-padded.is-flush-top {
  padding-top: 0;
}

/*
 * bottom slot 只负责固定定位，不画底、不补安全区：
 * 底栏组件自己决定背景与 safe-area，页面用 bottom-space 预留高度。
 */
.page-shell__bottom {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  margin: 0 auto;
}
</style>
