<template>
  <image
    v-if="showImage"
    class="app-icon"
    :src="src"
    :style="boxStyle"
    mode="aspectFit"
    :aria-label="label || undefined"
    @error="onError"
  />
  <view
    v-else
    class="app-icon app-icon--fallback"
    :style="boxStyle"
    :aria-label="label || undefined"
  >
    <view class="app-icon__dot" />
  </view>
</template>

<script setup>
import { computed, ref, watch } from "vue"

/**
 * AppIcon — 统一图标入口
 *
 * 约定：name 直接映射到 /static/icons/<name>.png，两端各自持有同名资产。
 * 未知/非法 name 与加载失败都降级成中性占位块，不抛错 —— 页面不需要为
 * 「这个端还没有这张图」写分支，也不会因为拼错名字而白屏。
 */
const props = defineProps({
  name: { type: String, default: "" },
  // 数字按 rpx 处理；字符串可写 "72rpx" / "24px"
  size: { type: [String, Number], default: 48 },
  // 有 label 时才声明语义；空 label 视为装饰性图标
  label: { type: String, default: "" }
})

const SAFE_NAME = /^[a-z0-9][a-z0-9_-]*$/i
const failed = ref(false)

const iconName = computed(() => String(props.name || "").trim().replace(/\.png$/i, ""))
const src = computed(() => {
  if (!SAFE_NAME.test(iconName.value)) return ""
  return `/static/icons/${iconName.value}.png`
})
const showImage = computed(() => Boolean(src.value) && !failed.value)

const boxStyle = computed(() => {
  const raw = typeof props.size === "number" ? `${props.size}rpx` : String(props.size || "").trim()
  const size = !raw ? "48rpx" : /^\d+(\.\d+)?$/.test(raw) ? `${raw}rpx` : raw
  return { width: size, height: size }
})

function onError() {
  failed.value = true
}

// 换 name 时重新给资源一次机会，避免一次失败永久占位
watch(iconName, () => {
  failed.value = false
})
</script>

<style scoped lang="scss">
.app-icon {
  display: block;
  flex: none;
}

.app-icon--fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.app-icon__dot {
  width: 28%;
  height: 28%;
  border-radius: $border-radius-full;
  background: $color-text-hint;
}
</style>
