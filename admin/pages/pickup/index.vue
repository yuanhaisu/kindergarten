<template>
  <div class="space-y-4">
    <!-- 接送概览 -->
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

    <!-- Tab 切换 -->
    <el-card shadow="never">
      <el-tabs v-model="activeTab" @tab-change="onTabChange">
        <!-- Tab 1: 接送记录 -->
        <el-tab-pane label="接送记录" name="records">
          <DataTable
            v-model:page="recPage"
            v-model:page-size="recPageSize"
            :data="recList as unknown as Record<string, unknown>[]"
            :loading="recLoading"
            :total="recTotal"
            @page-change="loadRecords"
            @size-change="loadRecords"
          >
            <el-table-column label="学生" width="120">
              <template #default="{ row }">
                {{ (row as any).student?.name }}
              </template>
            </el-table-column>
            <el-table-column prop="type_label" label="类型" width="90" />
            <el-table-column prop="pickup_name" label="接送人" />
            <el-table-column prop="method_label" label="方式" width="90" />
            <el-table-column prop="recorded_at" label="时间" width="170" />
          </DataTable>
        </el-tab-pane>

        <!-- Tab 2: 异常记录 -->
        <el-tab-pane label="异常记录" name="abnormal">
          <DataTable
            v-model:page="abPage"
            v-model:page-size="abPageSize"
            :data="(abList as any)"
            :loading="abLoading"
            :total="abTotal"
            @page-change="loadAbnormal"
            @size-change="loadAbnormal"
          >
            <el-table-column label="学生" width="120">
              <template #default="{ row }">
                {{ (row as any).student?.name }}
              </template>
            </el-table-column>
            <el-table-column label="班级" width="120">
              <template #default="{ row }">
                {{ (row as any).student?.class_name }}
              </template>
            </el-table-column>
            <el-table-column prop="type_label" label="类型" width="90" />
            <el-table-column prop="pickup_name" label="接送人" />
            <el-table-column label="授权状态" width="100">
              <template #default="{ row }">
                <el-tag
                  :type="(row as any).is_authorized ? 'success' : 'danger'"
                  size="small"
                >
                  {{ (row as any).is_authorized ? "已授权" : "未授权" }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="abnormal_reason" label="异常原因" min-width="150" show-overflow-tooltip />
            <el-table-column prop="recorded_at" label="时间" width="170" />
          </DataTable>
        </el-tab-pane>

        <!-- Tab 3: 授权审核 -->
        <el-tab-pane label="授权审核" name="audit">
          <div class="mb-3 flex items-center gap-2">
            <span class="text-sm text-gray-500">筛选：</span>
            <el-select v-model="auditStatusFilter" placeholder="审核状态" class="w-32" @change="loadAuditList">
              <el-option label="待审核" value="pending" />
              <el-option label="全部" value="" />
            </el-select>
          </div>
          <el-table :data="auditList" v-loading="auditLoading" stripe>
            <el-table-column label="学生" width="100">
              <template #default="{ row }">
                {{ row.student_name }}
              </template>
            </el-table-column>
            <el-table-column label="班级" width="100">
              <template #default="{ row }">
                {{ row.class_name }}
              </template>
            </el-table-column>
            <el-table-column prop="name" label="接送人姓名" width="120" />
            <el-table-column prop="relation" label="关系" width="80" />
            <el-table-column prop="phone" label="电话" width="130" />
            <el-table-column label="授权类型" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.auth_type === 'permanent' ? 'success' : 'warning'">
                  {{ row.auth_type === 'permanent' ? '长期' : '临时' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="审核状态" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.review_status === 0 ? 'warning' : row.review_status === 1 ? 'success' : 'danger'">
                  {{ row.review_status === 0 ? '待审核' : row.review_status === 1 ? '已通过' : '已拒绝' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="申请时间" width="170" />
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="{ row }">
                <template v-if="row.review_status === 0">
                  <el-button type="success" size="small" @click="handleAudit(row, 'approve')">通过</el-button>
                  <el-button type="danger" size="small" @click="handleAudit(row, 'reject')">拒绝</el-button>
                </template>
                <span v-else class="text-xs text-gray-400">
                  {{ row.audited_at?.slice(0, 10) }} {{ row.audit_remark || "—" }}
                </span>
              </template>
            </el-table-column>
          </el-table>
          <div class="mt-3 flex justify-end">
            <el-pagination
              v-model:current-page="auditPage"
              v-model:page-size="auditPageSize"
              :total="auditTotal"
              layout="total, prev, pager, next"
              @current-change="loadAuditList"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import type { PickupOverview, PickupRecordItem } from '~/types/api'
import { getPickupOverview, getPickupRecords, getPickupAbnormal, auditPickupAuthorizedPerson, getAuthorizedPersons } from '~/api/pickup'

// ---- 概览 ----
const date = ref(new Date().toISOString().slice(0, 10))
const ov = ref<PickupOverview | null>(null)
async function loadOv() {
  try {
    ov.value = await getPickupOverview({ date: date.value })
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}

// ---- Tab ----
const activeTab = ref('records')

// ---- 接送记录 ----
const { page: recPage, pageSize: recPageSize, total: recTotal } = usePagination()
const recLoading = ref(false)
const recList = ref<PickupRecordItem[]>([])

async function loadRecords() {
  recLoading.value = true
  try {
    const res = await getPickupRecords({
      page: recPage.value,
      page_size: recPageSize.value,
      date: date.value,
    })
    recList.value = res.list
    recTotal.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    recLoading.value = false
  }
}

// ---- 异常记录 ----
const { page: abPage, pageSize: abPageSize, total: abTotal } = usePagination()
const abLoading = ref(false)
const abList = ref<Record<string, unknown>[]>([])

async function loadAbnormal() {
  abLoading.value = true
  try {
    const res = await getPickupAbnormal({
      page: abPage.value,
      page_size: abPageSize.value,
      date: date.value,
    })
    abList.value = res.list as unknown as Record<string, unknown>[]
    abTotal.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    abLoading.value = false
  }
}

// ---- 授权审核 ----
interface AuditRow {
  id: number
  student_id: number
  student_name: string
  class_name: string
  name: string
  relation: string
  phone: string
  auth_type: string
  review_status: number
  audited_at: string | null
  audit_remark: string
  created_at: string
}

const auditStatusFilter = ref('pending')
const auditList = ref<AuditRow[]>([])
const auditLoading = ref(false)
const auditPage = ref(1)
const auditPageSize = ref(15)
const auditTotal = ref(0)

async function loadAuditList() {
  auditLoading.value = true
  try {
    const params: Record<string, unknown> = {
      page: auditPage.value,
      page_size: auditPageSize.value,
    }
    if (auditStatusFilter.value === 'pending') {
      params.status = 'pending'
    }
    const res = await getAuthorizedPersons(params as { page: number; page_size: number; status?: string })
    auditList.value = (res.list as unknown as AuditRow[]) || []
    auditTotal.value = res.total || 0
  } catch (e: unknown) {
    auditList.value = []
    auditTotal.value = 0
    ElMessage.error(e instanceof Error ? e.message : '加载授权列表失败')
  } finally {
    auditLoading.value = false
  }
}

async function handleAudit(row: AuditRow, action: 'approve' | 'reject') {
  try {
    const label = action === 'approve' ? '通过' : '拒绝'
    await ElMessageBox.confirm(`确定${label}该授权申请吗？`, '审核确认', {
      confirmButtonText: label,
      cancelButtonText: '取消',
      type: 'warning',
    })
    await auditPickupAuthorizedPerson(row.id, { action })
    ElMessage.success(`已${label}`)
    loadAuditList()
  } catch (e: unknown) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e instanceof Error ? e.message : '审核失败')
    }
  }
}

function onTabChange(tabName: string | number) {
  if (tabName === 'records') loadRecords()
  else if (tabName === 'abnormal') loadAbnormal()
  else if (tabName === 'audit') loadAuditList()
}

// ---- Init ----
onMounted(async () => {
  await loadOv()
  loadRecords()
})
</script>
