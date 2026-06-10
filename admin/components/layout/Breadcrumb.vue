<template>
  <el-breadcrumb separator="/">
    <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
    <el-breadcrumb-item v-if="currentTitle">{{ currentTitle }}</el-breadcrumb-item>
  </el-breadcrumb>
</template>

<script setup lang="ts">
import { ADMIN_MENUS } from '~/constants/menus'

const route = useRoute()

const currentTitle = computed(() => {
  const m = ADMIN_MENUS.find((x) => x.path === route.path)
  return m?.title || (route.path !== '/' ? route.path : '')
})
</script>
