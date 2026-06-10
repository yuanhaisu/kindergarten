<template>
  <div>
    <!-- 表头 -->
    <div
      class="flex items-center gap-4 px-3 py-2 text-xs font-semibold text-slate-500 border-b border-slate-100"
    >
      <span class="w-8">#</span>
      <span class="flex-1">班级名称</span>
      <span class="w-32 hidden sm:block">活跃度</span>
      <span class="w-12 text-right">趋势</span>
    </div>

    <!-- 数据行 -->
    <div v-if="list.length" class="divide-y divide-slate-50">
      <div
        v-for="(item, idx) in list"
        :key="item.class_id || idx"
        class="flex items-center gap-4 px-3 py-3 text-sm hover:bg-slate-50 transition-colors"
      >
        <!-- 排名 -->
        <span
          class="w-8 shrink-0 font-semibold text-center"
          :class="idx < 3 ? 'text-blue-500' : 'text-slate-400'"
        >
          {{ idx + 1 }}
        </span>

        <!-- 班级名称 -->
        <span class="flex-1 font-medium text-slate-800 truncate">{{ item.class_name }}</span>

        <!-- 活跃度进度条 -->
        <div class="w-32 hidden sm:flex items-center gap-2">
          <el-progress
            :percentage="item.activity_rate"
            :stroke-width="6"
            :show-text="false"
            :color="progressColor(item.activity_rate)"
          />
          <span class="text-xs text-slate-500 tabular-nums w-10">{{ item.activity_rate }}%</span>
        </div>

        <!-- 移动端 -->
        <span class="sm:hidden text-xs text-slate-500 tabular-nums">{{ item.activity_rate }}%</span>

        <!-- 趋势 -->
        <span class="w-12 text-right shrink-0">
          <span
            v-if="item.trend === 'up'"
            class="text-green-500 font-semibold"
          >↑</span>
          <span
            v-else-if="item.trend === 'down'"
            class="text-red-500 font-semibold"
          >↓</span>
          <span v-else class="text-slate-400">—</span>
        </span>
      </div>
    </div>

    <el-empty v-else description="暂无数据" :image-size="80" />
  </div>
</template>

<script setup lang="ts">
import type { ClassActivityItem } from '~/types/api'

defineProps<{
  list: ClassActivityItem[]
}>()

function progressColor(rate: number): string {
  if (rate >= 80) return '#22C55E'
  if (rate >= 60) return '#3B82F6'
  if (rate >= 40) return '#F97316'
  return '#EF4444'
}
</script>
