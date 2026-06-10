<template>
  <div class="space-y-4">
    <el-card shadow="never">
      <template #header>
        <div class="flex items-center gap-3">
          <span>接送概览</span>
          <el-date-picker v-model="date" value-format="YYYY-MM-DD" @change="loadOv" />
        </div>
      </template>
      <div v-if="ov" class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <StatCard title="已到园" :value="ov.arrived" />
        <StatCard title="已接走" :value="ov.picked_up" />
        <StatCard title="未接走" :value="ov.not_picked_up" />
        <StatCard title="异常" :value="ov.abnormal_count" />
      </div>
    </el-card>
    <el-card shadow="never">
      <template #header><span>接送记录</span></template>
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
        <el-table-column prop="type_label" label="类型" width="90" />
        <el-table-column prop="pickup_name" label="接送人" />
        <el-table-column prop="method_label" label="方式" width="90" />
        <el-table-column prop="recorded_at" label="时间" width="170" />
      </DataTable>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import type { PickupOverview, PickupRecordItem } from '~/types/api'
import { getPickupOverview, getPickupRecords } from '~/api/pickup'

const date = ref(new Date().toISOString().slice(0, 10))
const ov = ref<PickupOverview | null>(null)
async function loadOv() {
  try {
    ov.value = await getPickupOverview({ date: date.value })
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<PickupRecordItem[]>([])

async function load() {
  loading.value = true
  try {
    const res = await getPickupRecords({
      page: page.value,
      page_size: pageSize.value,
      date: date.value,
    })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadOv()
  load()
})
</script>
