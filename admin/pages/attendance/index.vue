<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap items-center gap-3">
        <span>出勤管理</span>
        <el-date-picker v-model="date" value-format="YYYY-MM-DD" @change="loadOverview" />
      </div>
    </template>
    <div v-if="overview" class="grid grid-cols-2 md:grid-cols-5 gap-3 mb-6">
      <StatCard title="应到" :value="overview.total_students" />
      <StatCard title="出勤" :value="overview.present" />
      <StatCard title="请假" :value="overview.leave" />
      <StatCard title="缺勤" :value="overview.absent" />
      <StatCard title="出勤率" :value="overview.attendance_rate" />
    </div>
    <el-table v-loading="loading" :data="overview?.by_class || []" stripe border>
      <el-table-column prop="class_name" label="班级" />
      <el-table-column prop="total" label="应到" width="80" />
      <el-table-column prop="present" label="出勤" width="80" />
      <el-table-column prop="leave" label="请假" width="80" />
      <el-table-column prop="absent" label="缺勤" width="80" />
      <el-table-column prop="rate" label="出勤率" width="100" />
    </el-table>
    <div class="mt-6">
      <div class="font-medium mb-2">出勤明细</div>
      <DataTable
        v-model:page="page"
        v-model:page-size="pageSize"
        :data="records as unknown as Record<string, unknown>[]"
        :loading="recLoading"
        :total="total"
        @page-change="loadRecords"
        @size-change="loadRecords"
      >
        <el-table-column label="学生" min-width="140">
          <template #default="{ row }">
            {{ (row as { student?: { name: string } }).student?.name }}
          </template>
        </el-table-column>
        <el-table-column prop="date" label="日期" width="120" />
        <el-table-column prop="status_label" label="状态" width="100" />
      </DataTable>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import type { AttendanceOverview } from '~/types/api'
import { getAttendanceOverview, getAttendanceRecords } from '~/api/attendance'

const date = ref(new Date().toISOString().slice(0, 10))
const overview = ref<AttendanceOverview | null>(null)
const loading = ref(false)

async function loadOverview() {
  loading.value = true
  try {
    overview.value = await getAttendanceOverview({ date: date.value })
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const { page, pageSize, total } = usePagination()
const recLoading = ref(false)
const records = ref<Record<string, unknown>[]>([])

async function loadRecords() {
  recLoading.value = true
  try {
    const res = await getAttendanceRecords({
      page: page.value,
      page_size: pageSize.value,
      date: date.value,
    })
    records.value = res.list as Record<string, unknown>[]
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    recLoading.value = false
  }
}

onMounted(async () => {
  await loadOverview()
  loadRecords()
})
</script>
