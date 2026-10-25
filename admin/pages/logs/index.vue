<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap gap-2 items-center">
        <span>操作日志</span>
        <el-select v-model="module" placeholder="模块" clearable class="!w-40" @change="() => { resetPage(); load() }">
          <el-option label="学生" value="student" />
          <el-option label="老师" value="teacher" />
          <el-option label="家长" value="parent" />
          <el-option label="任务" value="task" />
          <el-option label="系统" value="system" />
        </el-select>
      </div>
    </template>
    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="list as unknown as Record<string, unknown>[]"
      :loading="loading"
      :total="total"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column prop="module_label" label="模块" width="120" />
      <el-table-column prop="action_label" label="动作" width="100" />
      <el-table-column prop="target" label="对象" min-width="160" show-overflow-tooltip />
      <el-table-column label="操作人" width="120">
        <template #default="{ row }">
          {{ (row as { operator?: { nickname: string } }).operator?.nickname }}
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="时间" width="170" />
    </DataTable>
  </el-card>
</template>

<script setup lang="ts">
import type { OperationLogItem } from '~/types/api'
import { getOperationLogs } from '~/api/logs'

const { page, pageSize, total, resetPage } = usePagination()
const loading = ref(false)
const list = ref<OperationLogItem[]>([])
const module = ref<string | undefined>()

async function load() {
  loading.value = true
  try {
    const res = await getOperationLogs({
      page: page.value,
      page_size: pageSize.value,
      module: module.value,
    })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
