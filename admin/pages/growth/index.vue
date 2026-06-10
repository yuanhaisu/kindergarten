<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex justify-between">
        <span>成长记录</span>
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
      <el-table-column prop="date" label="日期" width="120" />
      <el-table-column prop="class_name" label="班级" />
      <el-table-column prop="content_preview" label="摘要" min-width="200" show-overflow-tooltip />
      <el-table-column prop="media_count" label="媒体" width="70" />
      <el-table-column prop="likes_count" label="赞" width="60" />
      <el-table-column prop="audit_status" label="审核" width="90" />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="audit((row as { id: number }).id, 'pass')">通过</el-button>
          <el-button link type="danger" @click="audit((row as { id: number }).id, 'reject')">驳回</el-button>
          <el-button link type="danger" @click="remove((row as { id: number }).id)">删除</el-button>
        </template>
      </el-table-column>
    </DataTable>
  </el-card>
</template>

<script setup lang="ts">
import type { GrowthLogItem } from '~/types/api'
import { auditGrowthLog, deleteGrowthLog, getGrowthLogs } from '~/api/growth'

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<GrowthLogItem[]>([])

async function load() {
  loading.value = true
  try {
    const res = await getGrowthLogs({ page: page.value, page_size: pageSize.value })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function audit(id: number, action: 'pass' | 'reject') {
  try {
    await auditGrowthLog(id, { action, remark: action === 'reject' ? '不符合规范' : '通过' })
    ElMessage.success('已处理')
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  }
}

async function remove(id: number) {
  try {
    await ElMessageBox.confirm('确定删除？', '提示', { type: 'warning' })
    await deleteGrowthLog(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* */
  }
}

onMounted(load)
</script>
