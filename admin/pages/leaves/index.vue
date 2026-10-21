<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap gap-2 items-center">
        <span>请假管理</span>
        <el-select
          v-model="statusFilter"
          placeholder="状态"
          clearable
          class="!w-32"
          @change="onFilterChange"
        >
          <el-option label="待审核" value="pending" />
          <el-option label="已同意" value="approved" />
          <el-option label="已驳回" value="rejected" />
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

// 查询串用语义值（与审核页一致），接口要数字：0 待审核 / 1 同意 / 2 驳回
const STATUS_MAP: Record<string, number> = { pending: 0, approved: 1, rejected: 2 }

const route = useRoute()
// 看板待办深链预填（/leaves?status=pending）
const statusFilter = ref<string | undefined>(
  typeof route.query.status === 'string' && route.query.status in STATUS_MAP
    ? route.query.status
    : undefined,
)
const { page, pageSize, total, resetPage } = usePagination()
const loading = ref(false)
const list = ref<LeaveItem[]>([])

// 同页再次深链时组件复用，须跟随 query 更新并重查
watch(
  () => route.query.status,
  (s) => {
    const next = typeof s === 'string' && s in STATUS_MAP ? s : undefined
    if (next !== statusFilter.value) {
      statusFilter.value = next
      resetPage()
      load()
    }
  },
)

async function load() {
  loading.value = true
  try {
    const res = await getLeaves({
      page: page.value,
      page_size: pageSize.value,
      status: statusFilter.value ? STATUS_MAP[statusFilter.value] : undefined,
    })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function onFilterChange() {
  resetPage()
  load()
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
