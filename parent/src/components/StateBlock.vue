<template>
  <view class="state-block">
    <view v-if="loading" class="state-body">
      <view class="state-spinner" />
      <text class="state-text">{{ loadingText }}</text>
    </view>

    <view v-else-if="error" class="state-body">
      <view class="state-icon is-error">!</view>
      <text class="state-text">{{ error }}</text>
      <view v-if="retryable" class="state-retry" @click="$emit('retry')">重试</view>
    </view>

    <view v-else class="state-body">
      <view class="state-icon is-empty">·</view>
      <text class="state-text">{{ emptyText }}</text>
    </view>
  </view>
</template>

<script setup>
/**
 * StateBlock — 三态合一（加载 / 错误 / 空）
 *
 * 空态的语义往往危险（如接送页空列表 = 「孩子都接走了」），
 * 所以错误态必须**排在空态之前**判定 —— 本组件已固化这个顺序，
 * 调用方只需传 loading / error / emptyText，不会误用。
 */
defineProps({
  loading: { type: Boolean, default: false },
  error: { type: String, default: "" },     // 非空即错误态
  loadingText: { type: String, default: "加载中..." },
  emptyText: { type: String, default: "暂无数据" },
  retryable: { type: Boolean, default: true }
})
defineEmits(["retry"])
</script>

<style scoped lang="scss">
.state-block {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 320rpx;
  padding: $spacing-xl;
}

.state-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-lg;
  text-align: center;
}

.state-text {
  font-size: $typography-font-size-md;
  color: $color-text-hint;
  line-height: 1.5;
}

.state-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 96rpx;
  height: 96rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-3xl;
  font-weight: 700;
}
.state-icon.is-error { background: $color-semantic-danger-bg; color: $color-semantic-danger; }
.state-icon.is-empty { background: $color-bg-page; color: $color-text-hint; }

.state-spinner {
  width: 56rpx;
  height: 56rpx;
  border-radius: $border-radius-full;
  border: 5rpx solid $gradient-brand-soft;
  border-top-color: $color-brand-primary;
  animation: state-spin 0.9s linear infinite;
}

.state-retry {
  padding: $spacing-sm $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-semantic-info;
  color: $color-text-on-accent;
  font-size: $typography-font-size-base;
}

@keyframes state-spin {
  to { transform: rotate(360deg); }
}
</style>
