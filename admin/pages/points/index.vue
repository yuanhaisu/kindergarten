<template>
  <div class="space-y-4">
    <el-card v-if="overview" shadow="never">
      <template #header><span>积分概览</span></template>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatCard title="流通积分" :value="overview.total_points_in_circulation" />
        <StatCard title="本月发放" :value="overview.total_points_awarded_this_month" />
        <StatCard title="本月消耗" :value="overview.total_points_spent_this_month" />
      </div>
    </el-card>
    <el-card shadow="never">
      <template #header>
        <div class="flex justify-between">
          <span>积分流水</span>
          <el-button type="primary" @click="adjustVisible = true">手动调整</el-button>
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
        <el-table-column prop="source_label" label="来源" />
        <el-table-column prop="amount" label="变动" width="80" />
        <el-table-column prop="description" label="说明" min-width="160" show-overflow-tooltip />
        <el-table-column prop="created_at" label="时间" width="170" />
      </DataTable>
    </el-card>
    <el-dialog v-model="adjustVisible" title="调整积分" width="400px">
      <el-form :model="adj" label-width="80px">
        <el-form-item label="学生ID"><el-input-number v-model="adj.student_id" :min="1" class="!w-full" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="adj.type" class="w-full">
            <el-option label="增加" value="earn" />
            <el-option label="扣减" value="spend" />
          </el-select>
        </el-form-item>
        <el-form-item label="数量"><el-input-number v-model="adj.amount" :min="1" class="!w-full" /></el-form-item>
        <el-form-item label="原因"><el-input v-model="adj.reason" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="adjustVisible = false">取消</el-button>
        <el-button type="primary" @click="doAdjust">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import type { PointRecordItem, PointsOverview } from '~/types/api'
import { adjustPoints, getPointRecords, getPointsOverview } from '~/api/points'

const overview = ref<PointsOverview | null>(null)

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<PointRecordItem[]>([])

async function load() {
  loading.value = true
  try {
    const res = await getPointRecords({ page: page.value, page_size: pageSize.value })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const adjustVisible = ref(false)
const adj = reactive({
  student_id: 1,
  type: 'earn' as 'earn' | 'spend',
  amount: 10,
  reason: '',
})

async function doAdjust() {
  try {
    await adjustPoints(adj)
    ElMessage.success('已调整')
    adjustVisible.value = false
    load()
    overview.value = await getPointsOverview()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  }
}

onMounted(async () => {
  try {
    overview.value = await getPointsOverview()
  } catch {
    /* */
  }
  load()
})
</script>
