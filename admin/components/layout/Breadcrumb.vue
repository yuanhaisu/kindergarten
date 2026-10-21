<template>
  <!-- 当前位置 = 分组 / 页面。祖先弱化、当前页加重；绝不把裸 URL 亮给用户 -->
  <nav class="flex items-center gap-2 min-w-0" aria-label="面包屑">
    <template v-if="current">
      <span class="text-sm text-slate-500 hidden sm:inline">{{ groupLabel }}</span>
      <span class="text-slate-400 hidden sm:inline" aria-hidden="true">/</span>
      <span class="text-xl font-bold text-slate-800 truncate">{{ current.title }}</span>
    </template>
    <span v-else class="text-xl font-bold text-slate-800">管理后台</span>
  </nav>
</template>

<script setup lang="ts">
import { findMenuGroup, matchMenuByPath } from '~/constants/menus'

const route = useRoute()

const current = computed(() => matchMenuByPath(route.path))
const groupLabel = computed(() => findMenuGroup(route.path)?.label ?? '')
</script>
