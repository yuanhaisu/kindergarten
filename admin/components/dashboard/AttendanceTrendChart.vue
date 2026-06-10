<template>
  <div class="h-[280px] w-full">
    <ClientOnly>
      <VChart v-if="data.length" class="chart" :option="option" autoresize />
      <el-empty v-else description="暂无趋势数据" :image-size="80" />
    </ClientOnly>
  </div>
</template>

<script setup lang="ts">
import type { AttendanceTrendPoint } from '~/types/api'

const props = defineProps<{
  data: AttendanceTrendPoint[]
}>()

const option = computed(() => ({
  tooltip: { trigger: 'axis' as const },
  legend: {
    data: ['出勤', '请假', '缺勤'],
    bottom: 0,
  },
  grid: { left: '3%', right: '4%', top: '8%', bottom: '12%', containLabel: true },
  xAxis: {
    type: 'category' as const,
    boundaryGap: false,
    data: props.data.map((d) => d.date.slice(5)),
    axisLabel: { fontSize: 11, color: '#94A3B8' },
  },
  yAxis: {
    type: 'value' as const,
    axisLabel: { fontSize: 11, color: '#94A3B8' },
  },
  series: [
    {
      name: '出勤',
      type: 'line',
      smooth: true,
      data: props.data.map((d) => d.present),
      itemStyle: { color: '#3B82F6' },
      lineStyle: { width: 2 },
      symbol: 'none',
    },
    {
      name: '请假',
      type: 'line',
      smooth: true,
      data: props.data.map((d) => d.leave),
      itemStyle: { color: '#F97316' },
      lineStyle: { width: 2 },
      symbol: 'none',
    },
    {
      name: '缺勤',
      type: 'line',
      smooth: true,
      data: props.data.map((d) => d.absent),
      itemStyle: { color: '#94A3B8' },
      lineStyle: { width: 2 },
      symbol: 'none',
    },
  ],
}))
</script>

<style scoped>
.chart {
  height: 280px;
  width: 100%;
}
</style>
