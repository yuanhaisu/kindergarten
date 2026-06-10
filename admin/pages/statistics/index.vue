<template>
  <div class="space-y-4">
    <el-card shadow="never">
      <template #header><span>请假统计（本月示例）</span></template>
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <PieChart v-if="leavePieOption.series" :chart-option="leavePieOption" />
        <BarChart v-if="classBarOption.series" :chart-option="classBarOption" />
      </div>
    </el-card>
    <el-card shadow="never">
      <template #header><span>成长 / 任务统计</span></template>
      <el-descriptions v-if="growth" :column="2" border size="small" class="mb-4">
        <el-descriptions-item label="日志总数">{{ (growth as { total_logs?: number }).total_logs }}</el-descriptions-item>
        <el-descriptions-item label="媒体">{{ (growth as { total_media?: number }).total_media }}</el-descriptions-item>
      </el-descriptions>
      <el-descriptions v-if="taskStats" :column="2" border size="small">
        <el-descriptions-item label="任务数">{{ (taskStats as { total_tasks?: number }).total_tasks }}</el-descriptions-item>
        <el-descriptions-item label="平均完成率">{{ (taskStats as { avg_completion_rate?: string }).avg_completion_rate }}</el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { getGrowthStats } from '~/api/growth'
import { getLeaveStats } from '~/api/leaves'
import { getTaskStats } from '~/api/tasks'

const growth = ref<Record<string, unknown> | null>(null)
const taskStats = ref<Record<string, unknown> | null>(null)
const leavePieOption = ref<Record<string, unknown>>({})
const classBarOption = ref<Record<string, unknown>>({})

onMounted(async () => {
  try {
    const [ls, gr, ts] = await Promise.all([getLeaveStats(), getGrowthStats(), getTaskStats()])
    growth.value = gr as Record<string, unknown>
    taskStats.value = ts as Record<string, unknown>
    const byReason = (ls as { by_reason?: { reason: string; count: number }[] }).by_reason || []
    leavePieOption.value = {
      tooltip: { trigger: 'item' },
      series: [
        {
          type: 'pie',
          radius: '60%',
          data: byReason.map((x) => ({ name: x.reason, value: x.count })),
        },
      ],
    }
    const byClass = (ls as { by_class?: { class_name: string; count: number }[] }).by_class || []
    classBarOption.value = {
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: byClass.map((x) => x.class_name) },
      yAxis: { type: 'value' },
      series: [{ type: 'bar', data: byClass.map((x) => x.count) }],
    }
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载统计失败')
  }
})
</script>
