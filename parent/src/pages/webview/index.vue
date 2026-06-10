<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"

const webviewUrl = ref("")

onLoad((options) => {
  if (options?.url) {
    webviewUrl.value = decodeURIComponent(options.url)
  }
})
</script>

<template>
  <view class="page">
    <web-view v-if="webviewUrl" :src="webviewUrl" />
    <view v-else class="empty">
      <text>缺少页面地址</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
}

.empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  color: $color-text-muted;
  font-size: 28rpx;
}
</style>
