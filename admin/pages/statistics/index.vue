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
      <template #header><span>成长统计</span></template>
      <el-descriptions v-if="growth" :column="2" border size="small">
        <el-descriptions-item label="日志总数">{{ growth.total_logs }}</el-descriptions-item>
        <el-descriptions-item label="媒体">{{ growth.total_media }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <span>任务统计（{{ taskStats?.period.start ?? '—' }} ~ {{ taskStats?.period.end ?? '—' }}）</span>
          <span class="text-xs text-text-hint">口径：提交 / 回应 / 参与，取代旧「完成率」</span>
        </div>
      </template>
      <el-descriptions v-if="taskStats" :column="4" border size="small">
        <el-descriptions-item label="任务数">{{ taskStats.total_tasks }}</el-descriptions-item>
        <el-descriptions-item label="进行中">{{ taskStats.active_tasks }}</el-descriptions-item>
        <el-descriptions-item label="已结束">{{ taskStats.completed_tasks }}</el-descriptions-item>
        <el-descriptions-item label="提交总数">{{ taskStats.total_checkins }}</el-descriptions-item>
        <el-descriptions-item label="回应率">{{ formatRate(taskStats.response_rate) }}</el-descriptions-item>
        <el-descriptions-item label="参与率">{{ formatRate(taskStats.participation_rate) }}</el-descriptions-item>
        <el-descriptions-item label="待回应">{{ taskStats.pending_count }}</el-descriptions-item>
        <el-descriptions-item label="超 48 小时未回应">{{ taskStats.pending_over_48h }}</el-descriptions-item>
        <el-descriptions-item label="回应中位数">
          {{ formatMetric(taskStats.median_response_hours, ' 小时') }}
        </el-descriptions-item>
        <el-descriptions-item label="反馈覆盖率">{{ formatRate(taskStats.feedback_coverage) }}</el-descriptions-item>
        <el-descriptions-item label="孩子开心占比">{{ formatRate(taskStats.child_happy_rate) }}</el-descriptions-item>
        <el-descriptions-item label="家长确认占比">{{ formatRate(taskStats.ack_rate) }}</el-descriptions-item>
      </el-descriptions>
      <p v-if="taskStats" class="mt-2 text-xs text-text-hint">
        「—」= 没有可计算的分母（例如此区间内没有回应），不是 0%。
        中位数只统计逐条回应；被剔除的负时长样本 {{ taskStats.median_response_negative_excluded }} 条。
        反馈类指标的分母是回应发出时的通知义务集，义务集未知的回应 {{ taskStats.notify_baseline_missing }} 条已从分母排除。
      </p>

      <div class="mt-6">
        <h4 class="mb-2 text-sm font-medium text-text-primary">按班级</h4>
        <el-table :data="taskStats?.by_class ?? []" size="small" border class="w-full">
          <el-table-column prop="class_name" label="班级" min-width="110" />
          <el-table-column prop="tasks" label="任务" width="70" />
          <el-table-column label="回应率" width="90">
            <template #default="{ row }">{{ formatRate(row.response_rate) }}</template>
          </el-table-column>
          <el-table-column label="参与率" width="90">
            <template #default="{ row }">{{ formatRate(row.participation_rate) }}</template>
          </el-table-column>
          <el-table-column prop="pending_count" label="待回应" width="90" />
          <el-table-column prop="pending_over_48h" label="超 48h" width="90" />
          <el-table-column label="回应中位数" width="110">
            <template #default="{ row }">{{ formatMetric(row.median_response_hours, ' 小时') }}</template>
          </el-table-column>
          <el-table-column label="反馈覆盖" width="100">
            <template #default="{ row }">{{ formatRate(row.feedback_coverage) }}</template>
          </el-table-column>
          <el-table-column label="孩子开心" width="100">
            <template #default="{ row }">{{ formatRate(row.child_happy_rate) }}</template>
          </el-table-column>
          <el-table-column label="家长确认" width="100">
            <template #default="{ row }">{{ formatRate(row.ack_rate) }}</template>
          </el-table-column>
        </el-table>
      </div>

      <div class="mt-6">
        <h4 class="mb-2 text-sm font-medium text-text-primary">按活动 / 独立任务</h4>
        <el-table :data="taskStats?.by_activity ?? []" size="small" border class="w-full">
          <el-table-column label="活动 / 任务" min-width="200" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="text-text-primary">{{ row.title }}</span>
              <el-tag v-if="row.activity_id" size="small" type="warning" class="ml-2">活动</el-tag>
              <el-tag v-else size="small" type="info" effect="plain" class="ml-2">独立任务</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="tasks" label="任务" width="70" />
          <el-table-column label="回应率" width="90">
            <template #default="{ row }">{{ formatRate(row.response_rate) }}</template>
          </el-table-column>
          <el-table-column label="参与率" width="90">
            <template #default="{ row }">{{ formatRate(row.participation_rate) }}</template>
          </el-table-column>
          <el-table-column prop="pending_count" label="待回应" width="90" />
          <el-table-column prop="pending_over_48h" label="超 48h" width="90" />
          <el-table-column label="回应中位数" width="110">
            <template #default="{ row }">{{ formatMetric(row.median_response_hours, ' 小时') }}</template>
          </el-table-column>
          <el-table-column label="反馈覆盖" width="100">
            <template #default="{ row }">{{ formatRate(row.feedback_coverage) }}</template>
          </el-table-column>
        </el-table>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import type { TaskStats } from '~/types/api'
import { getGrowthStats } from '~/api/growth'
import { getLeaveStats } from '~/api/leaves'
import { getTaskStats } from '~/api/tasks'
import { formatMetric, formatRate } from '~/utils/format'

const growth = ref<{ total_logs?: number; total_media?: number } | null>(null)
const taskStats = ref<TaskStats | null>(null)
const leavePieOption = ref<Record<string, unknown>>({})
const classBarOption = ref<Record<string, unknown>>({})

onMounted(async () => {
  try {
    const [ls, gr, ts] = await Promise.all([getLeaveStats(), getGrowthStats(), getTaskStats()])
    growth.value = gr as { total_logs?: number; total_media?: number }
    // 统计接口的比率字段可能是 null（无分母），展示层统一走 formatRate 显示「—」，
    // 不能 Number(null) 成 0%，也不要再对 "NN%" 乘 100。
    taskStats.value = ts
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
