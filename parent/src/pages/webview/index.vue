<script setup>
import AppButton from "@/components/AppButton.vue"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  info: "\u{eca1}" // error-warning-line
}

// X00 决策：本页入口已隐藏，只保留边界空态。孤儿路由没有返回栈时
// 回落到首页；首页是 tabBar 页，必须用 switchTab。
function goBack() {
  if (getCurrentPages().length > 1) {
    uni.navigateBack()
    return
  }
  uni.switchTab({ url: "/pages/index/index" })
}
</script>

<template>
  <PageShell :padded="false" :safe-bottom="false" max-width="430px">
    <view class="webview-page">
      <view class="webview-empty">
        <view class="webview-empty__icon">
          <text class="webview-empty__glyph">{{ ICON.info }}</text>
        </view>
        <text class="webview-empty__title">该页面暂未开放</text>
        <text class="webview-empty__desc">园所配置业务域名后可访问</text>
        <view class="webview-empty__action">
          <AppButton size="md" @click="goBack">返回</AppButton>
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
.webview-page {
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: $spacing-xl $component-page-padding-x;
  padding-bottom: calc(#{$spacing-xl} + env(safe-area-inset-bottom));
}

.webview-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.webview-empty__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-semantic-info-bg;
}

.webview-empty__glyph {
  color: $color-semantic-info;
  font-family: remixicon;
  font-size: 52rpx;
  line-height: 1;
}

.webview-empty__title {
  max-width: 560rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.webview-empty__desc {
  max-width: 460rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.webview-empty__action {
  width: 320rpx;
  margin-top: $spacing-xl;
}
</style>
