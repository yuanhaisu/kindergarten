<template>
  <el-card shadow="never">
    <template #header><span>亲子任务</span></template>
    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="list as unknown as Record<string, unknown>[]"
      :loading="loading"
      :total="total"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column prop="title" label="任务" min-width="160" />
      <el-table-column prop="class_name" label="班级" />
      <el-table-column prop="frequency_label" label="频率" width="100" />
      <el-table-column prop="points" label="积分" width="70" />
      <el-table-column prop="completion_rate" label="完成率" width="100" />
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button link type="danger" @click="remove((row as { id: number }).id)">删除</el-button>
        </template>
      </el-table-column>
    </DataTable>
  </el-card>
</template>

<script setup lang="ts">
import type { TaskItem } from '~/types/api'
import { deleteTask, getTasks } from '~/api/tasks'

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<TaskItem[]>([])

async function load() {
  loading.value = true
  try {
    const res = await getTasks({ page: page.value, page_size: pageSize.value })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function remove(id: number) {
  try {
    await ElMessageBox.confirm('确定删除任务？', '提示', { type: 'warning' })
    await deleteTask(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* */
  }
}

onMounted(load)
</script>
