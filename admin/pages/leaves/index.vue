<template>
  <el-card shadow="never">
    <template #header><span>请假管理</span></template>
    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="list as unknown as Record<string, unknown>[]"
      :loading="loading"
      :total="total"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column label="学生" width="120">
        <template #default="{ row }">
          {{ (row as { student?: { name: string } }).student?.name }}
        </template>
      </el-table-column>
      <el-table-column prop="reason" label="原因" min-width="120" />
      <el-table-column prop="start_date" label="开始" width="120" />
      <el-table-column prop="end_date" label="结束" width="120" />
      <el-table-column prop="days" label="天数" width="70" />
      <el-table-column prop="status_label" label="状态" width="100" />
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <template v-if="(row as LeaveItem).status === 0">
            <el-button type="primary" link size="small" @click="onAudit(row as LeaveItem, 1)">同意</el-button>
            <el-button type="danger" link size="small" @click="onAudit(row as LeaveItem, 2)">驳回</el-button>
          </template>
          <span v-else class="text-gray-400">—</span>
        </template>
      </el-table-column>
    </DataTable>
  </el-card>
</template>

<script setup lang="ts">
import type { LeaveItem } from '~/types/api'
import { auditLeave, getLeaves } from '~/api/leaves'

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<LeaveItem[]>([])

async function load() {
  loading.value = true
  try {
    const res = await getLeaves({ page: page.value, page_size: pageSize.value })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function onAudit(row: LeaveItem, status: number) {
  const tip = status === 1 ? '确认同意该请假？' : '确认驳回该请假？'
  try {
    await ElMessageBox.confirm(tip, '审批', { type: 'warning' })
  } catch {
    return
  }
  loading.value = true
  try {
    await auditLeave(row.id, { status })
    ElMessage.success('已处理')
    await load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
