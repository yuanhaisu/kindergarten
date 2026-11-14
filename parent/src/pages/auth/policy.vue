<script setup>
/**
 * 政策全文：家长端非个人号，可用 web-view 打开官网。
 * 老师端个人号不支持 web-view，不使用此页面（改用摘要 + 复制链接）。
 */
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import config from "@/config/env"

const src = ref("")

onLoad((query) => {
  const type = query?.type === "terms" ? "terms" : "privacy"
  src.value = `${config.siteURL}/${type}`
})
</script>

<template>
  <web-view v-if="src" :src="src" />
</template>
