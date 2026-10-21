<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap gap-2 items-center">
        <span>内容审核</span>
        <el-select v-model="atype" placeholder="类型" clearable class="!w-32" @change="load">
          <el-option label="通知" value="notice" />
          <el-option label="成长" value="growth" />
          <el-option label="评论" value="comment" />
        </el-select>
        <el-select v-model="astatus" placeholder="状态" clearable class="!w-32" @change="load">
          <el-option label="待审核" value="pending" />
          <el-option label="已通过" value="passed" />
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
      <el-table-column prop="type_label" label="类型" width="90" />
      <el-table-column prop="title" label="标题" min-width="160" show-overflow-tooltip />
      <el-table-column prop="status_label" label="状态" width="100" />
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="doAudit((row as { id: number }).id, 'pass')">通过</el-button>
          <el-button link type="danger" @click="doAudit((row as { id: number }).id, 'reject')">驳回</el-button>
        </template>
      </el-table-column>
    </DataTable>
  </el-card>
</template>

<script setup lang="ts">
import type { AuditListItem } from '~/types/api'
import { auditContent, getAuditList } from '~/api/audit'

const route = useRoute()
// 看板待办深链预填（/audit?type=growth&status=pending）
const atype = ref(typeof route.query.type === 'string' ? route.query.type : undefined)
const astatus = ref(typeof route.query.status === 'string' ? route.query.status : undefined)
const { page, pageSize, total, resetPage } = usePagination()

// 同页再次深链时组件复用，须跟随 query 更新并重查
watch(
  () => [route.query.type, route.query.status],
  ([t, s]) => {
    const nextType = typeof t === 'string' ? t : undefined
    const nextStatus = typeof s === 'string' ? s : undefined
    if (nextType !== atype.value || nextStatus !== astatus.value) {
      atype.value = nextType
      astatus.value = nextStatus
      resetPage()
      load()
    }
  },
)
const loading = ref(false)
const list = ref<AuditListItem[]>([])

async function load() {
  loading.value = true
  try {
    const res = await getAuditList({
      page: page.value,
      page_size: pageSize.value,
      type: atype.value,
      status: astatus.value,
    })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function doAudit(id: number, action: 'pass' | 'reject') {
  try {
    await auditContent(id, { action, remark: action === 'reject' ? '内容不合规' : '' })
    ElMessage.success('已处理')
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  }
}

onMounted(load)
</script>
