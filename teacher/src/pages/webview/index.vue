<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"

const src = ref("")

onLoad((query = {}) => {
  src.value = decodeURIComponent(query.url || "")
})
</script>

<template>
  <web-view v-if="src" :src="src" />
  <view v-else class="empty">
    <text>未提供页面地址</text>
  </view>
</template>

<style scoped lang="scss">
.empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: $teacher-bg;
  color: $teacher-muted;
  font-size: 28rpx;
}
</style>
