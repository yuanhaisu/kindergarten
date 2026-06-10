<template>
  <el-card shadow="never">
    <template #header><span>数据导出</span></template>
    <p class="text-slate-500 text-sm mb-4">提交异步导出任务后，轮询任务状态直至可下载。</p>
    <div class="flex flex-wrap gap-3">
      <el-button type="primary" :loading="busy === 'att'" @click="runAttendance">导出出勤报表</el-button>
      <el-button type="primary" :loading="busy === 'stu'" @click="runStudents">导出学生名册</el-button>
      <el-button type="primary" :loading="busy === 'gr'" @click="runGrowth">导出成长记录</el-button>
    </div>
    <el-card v-if="lastTask" class="mt-6" shadow="never">
      <div class="text-sm space-y-2">
        <div>任务 ID：{{ lastTask.task_id }}</div>
        <div>状态：{{ lastTask.status }} {{ lastTask.progress != null ? `(${lastTask.progress}%)` : '' }}</div>
        <div v-if="lastTask.status === 'completed' && lastTask.task_id">
          <el-button type="primary" link @click="downloadLast">下载文件</el-button>
        </div>
      </div>
    </el-card>
  </el-card>
</template>

<script setup lang="ts">
import type { ExportTaskStatus } from '~/types/api'
import {
  downloadExportBlob,
  exportAttendance,
  exportGrowthTask,
  exportStudentsTask,
  getExportTask,
} from '~/api/export'

const busy = ref('')
const lastTask = ref<ExportTaskStatus | null>(null)

async function poll(taskId: string) {
  for (let i = 0; i < 60; i++) {
    const t = await getExportTask(taskId)
    lastTask.value = t
    if (t.status === 'completed' || t.status === 'failed') return
    await new Promise((r) => setTimeout(r, 1500))
  }
}

async function runAttendance() {
  busy.value = 'att'
  try {
    const t = await exportAttendance({
      class_id: null,
      start_date: '2026-03-01',
      end_date: '2026-03-31',
      format: 'xlsx',
    })
    lastTask.value = t
    if (t.task_id) await poll(t.task_id)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  } finally {
    busy.value = ''
  }
}

async function runStudents() {
  busy.value = 'stu'
  try {
    const t = await exportStudentsTask({ class_id: null, status: 1, format: 'xlsx' })
    lastTask.value = t
    if (t.task_id) await poll(t.task_id)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  } finally {
    busy.value = ''
  }
}

async function runGrowth() {
  busy.value = 'gr'
  try {
    const t = await exportGrowthTask({
      class_id: 1,
      start_date: '2026-03-01',
      end_date: '2026-03-31',
      format: 'xlsx',
    })
    lastTask.value = t
    if (t.task_id) await poll(t.task_id)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  } finally {
    busy.value = ''
  }
}

async function downloadLast() {
  const t = lastTask.value
  if (!t?.task_id) return
  try {
    const blob = await downloadExportBlob(t.task_id)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = t.file_name ?? 'export.xlsx'
    a.click()
    URL.revokeObjectURL(url)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '下载失败')
  }
}
</script>
