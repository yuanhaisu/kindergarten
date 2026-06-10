<template>
  <div>
    <!-- 环形图 -->
    <ClientOnly>
      <VChart
        v-if="chartOption"
        :option="chartOption"
        style="height: 240px"
        autoresize
      />
      <div v-else style="height: 240px" class="flex items-center justify-center">
        <span class="text-slate-400 text-sm">暂无数据</span>
      </div>
    </ClientOnly>

    <!-- 图例列表 -->
    <div v-if="data.length" class="mt-4 space-y-3">
      <div
        v-for="(item, idx) in data"
        :key="item.name"
        class="flex items-center justify-between text-sm"
      >
        <div class="flex items-center gap-2">
          <span
            class="w-3 h-3 rounded-sm shrink-0"
            :style="{ backgroundColor: item.color || defaultColors[idx] }"
          />
          <span class="text-slate-700">{{ item.name }}</span>
        </div>
        <span class="text-slate-500 font-medium tabular-nums">
          {{ item.value }}人
          <span class="text-slate-400 ml-1">
            ({{ total > 0 ? Math.round((item.value / total) * 100) : 0 }}%)
          </span>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { LeaveReasonItem } from '~/types/api'
import { computed } from 'vue'

const defaultColors = ['#3B82F6', '#F97316', '#8B5CF6', '#64748B', '#EF4444']

const props = defineProps<{
  data: LeaveReasonItem[]
  total?: number
}>()

const total = computed(() => {
  if (props.total !== undefined) return props.total
  return props.data.reduce((sum, item) => sum + item.value, 0)
})

const chartOption = computed(() => {
  if (!props.data.length) return null

  return {
    tooltip: {
      trigger: 'item' as const,
      formatter: '{b}: {c}人 ({d}%)',
    },
    legend: { show: false },
    series: [
      {
        type: 'pie',
        radius: ['50%', '75%'],
        center: ['50%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 4,
          borderColor: '#fff',
          borderWidth: 2,
        },
        label: { show: false },
        emphasis: {
          label: { show: true, fontSize: 14, fontWeight: 'bold' },
        },
        data: props.data.map((item, idx) => ({
          name: item.name,
          value: item.value,
          itemStyle: {
            color: item.color || defaultColors[idx % defaultColors.length],
          },
        })),
      },
    ],
  }
})
</script>
