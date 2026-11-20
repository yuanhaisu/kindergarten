<template>
  <div class="flex min-h-screen items-center justify-center bg-platform-bg px-4 py-10">
    <div class="w-full max-w-[420px]">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
// 登录页没有侧栏，但 EP 的输入框/消息框同样走平台主题。
useHead({ bodyAttrs: { class: 'platform-surface' } })
</script>
