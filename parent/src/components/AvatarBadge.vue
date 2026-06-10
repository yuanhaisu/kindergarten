<template>
  <view class="avatar-badge" :style="{ width: size + 'rpx', height: size + 'rpx' }">
    <image
      v-if="src"
      :src="src"
      class="avatar-img"
      :style="{ width: size + 'rpx', height: size + 'rpx' }"
      mode="aspectFill"
    />
    <text v-else class="avatar-placeholder" :style="{ fontSize: size * 0.4 + 'rpx' }">
      {{ placeholder }}
    </text>
    <view v-if="dot" class="avatar-dot" :class="'dot-' + dotColor" />
  </view>
</template>

<script setup>
defineProps({
  src: { type: String, default: "" },
  size: { type: Number, default: 64 },
  placeholder: { type: String, default: "?" },
  dot: { type: Boolean, default: false },
  dotColor: { type: String, default: "green" }
})
</script>

<style scoped>
.avatar-badge {
  position: relative;
  border-radius: 9999rpx;
  overflow: hidden;
  background: #DBEAFE;
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-img {
  border-radius: 9999rpx;
}

.avatar-placeholder {
  color: #2563EB;
  font-weight: 700;
}

.avatar-dot {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 16rpx;
  height: 16rpx;
  border-radius: 9999rpx;
  border: 3rpx solid #FFFFFF;
}

.dot-green { background: #16A34A; }
.dot-orange { background: #F97316; }
.dot-red { background: #EF4444; }
.dot-gray { background: #94A3B8; }
</style>
