<template>
  <view class="app-card" :style="{ borderRadius: radius + 'rpx' }">
    <slot />
  </view>
</template>

<script setup>
defineProps({
  radius: { type: Number, default: 16 }
})
</script>

<style scoped>
.app-card {
  padding: $spacing-xl;
  background: #FFFFFF;
  border-radius: $border-radius-lg;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}
</style>
