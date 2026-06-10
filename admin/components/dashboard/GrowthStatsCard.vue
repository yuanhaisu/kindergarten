<template>
  <div class="grid grid-cols-2 gap-3 text-sm">
    <div class="rounded-lg bg-slate-50 p-3">
      <div class="text-slate-500">成长日志</div>
      <div class="text-lg font-semibold">{{ stats?.total_logs ?? '-' }}</div>
    </div>
    <div class="rounded-lg bg-slate-50 p-3">
      <div class="text-slate-500">媒体数</div>
      <div class="text-lg font-semibold">{{ stats?.total_media ?? '-' }}</div>
    </div>
    <div class="rounded-lg bg-slate-50 p-3">
      <div class="text-slate-500">点赞</div>
      <div class="text-lg font-semibold">{{ stats?.total_likes ?? '-' }}</div>
    </div>
    <div class="rounded-lg bg-slate-50 p-3">
      <div class="text-slate-500">评论</div>
      <div class="text-lg font-semibold">{{ stats?.total_comments ?? '-' }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { GrowthStatsRes } from '~/types/api'

defineProps<{
  stats: GrowthStatsRes | null
}>()
</script>
