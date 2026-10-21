<template>
  <div class="space-y-4">
    <!-- 接送概览 -->
    <el-card shadow="never">
      <template #header>
        <div class="flex items-center gap-3">
          <span>接送概览</span>
          <el-date-picker v-model="date" value-format="YYYY-MM-DD" @change="onDateChange" />
        </div>
      </template>
      <div v-if="ov" class="grid grid-cols-2 md:grid-cols-5 gap-3">
        <StatCard title="已到园" :value="ov.arrived" />
        <StatCard title="已接走" :value="ov.picked_up" />
        <StatCard title="未接走" :value="ov.not_picked_up" />
        <StatCard title="超时未接" :value="ov.overdue_not_picked ?? overdueCount" color="#EF4444" />
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
            <el-table-column label="核验人" width="110">
              <template #default="{ row }">
                {{ (row as any).operator?.nickname || "—" }}
              </template>
            </el-table-column>
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
            <span class="text-sm text-text-muted">筛选：</span>
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
                <span v-else class="text-xs text-text-hint">
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
        <!-- Tab 4: 滞留名单 -->
        <el-tab-pane label="滞留名单" name="overdue">
          <div class="mb-3 flex items-center gap-2">
            <el-button type="warning" size="small" :loading="notifyLoading" @click="handleNotifyAll">
              一键通知全部滞留家长
            </el-button>
            <span class="text-sm text-text-muted">共 {{ overdueList.length }} 人滞留</span>
          </div>
          <el-table :data="overdueList" v-loading="overdueLoading" stripe>
            <el-table-column prop="name" label="学生" width="120" />
            <el-table-column prop="class_name" label="班级" width="120" />
            <el-table-column prop="pickup_deadline" label="放学时间" width="100" />
            <el-table-column prop="arrive_time" label="到园时间" width="100" />
            <el-table-column label="滞留时长" width="120">
              <template #default="{ row }">
                <el-tag type="danger" size="small">{{ row.overdue_minutes }} 分钟</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120">
              <template #default="{ row }">
                <el-button type="warning" size="small" @click="handleNotifyOne(row)">通知家长</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div v-if="overdueByClass.length > 0" class="mt-4">
            <h4 class="text-sm font-medium text-text-body mb-2">各班滞留计数</h4>
            <div class="flex flex-wrap gap-2">
              <el-tag v-for="c in overdueByClass" :key="c.class_id" type="warning">
                {{ c.class_name }}：{{ c.count }}人
              </el-tag>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import type { PickupOverview, PickupRecordItem } from '~/types/api'
import { getPickupOverview, getPickupRecords, getPickupAbnormal, auditPickupAuthorizedPerson, getAuthorizedPersons, getPickupOverdue, notifyPickupOverdue } from '~/api/pickup'

// ---- 概览 ----
const route = useRoute()
const date = ref(new Date().toISOString().slice(0, 10))
const ov = ref<PickupOverview | null>(null)
const overdueCount = ref(0)
async function loadOv() {
  try {
    ov.value = await getPickupOverview({ date: date.value })
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}

// ---- Tab ----
// 壳层紧急横幅深链 /pickup?tab=overdue 直达滞留名单
const TAB_NAMES = ['records', 'abnormal', 'audit', 'overdue'] as const
const queryTab = typeof route.query.tab === 'string' ? route.query.tab : 'records'
const activeTab = ref<string>(
  (TAB_NAMES as readonly string[]).includes(queryTab) ? queryTab : 'records',
)

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

// ---- 滞留名单 ----
interface OverdueRow {
  student_id: number
  name: string
  class_name: string
  pickup_deadline: string
  arrive_time: string
  overdue_minutes: number
}
const overdueList = ref<OverdueRow[]>([])
const overdueByClass = ref<Array<{ class_id: number; class_name: string; count: number }>>([])
const overdueLoading = ref(false)
const notifyLoading = ref(false)

/** 滞留时长（分钟）：后端未返回该字段，按班级放学时刻 HH:MM 与当前时间推算 */
function overdueMinutesOf(deadlineHM: string): number {
  const [h, m] = String(deadlineHM || '').split(':').map(Number)
  if (Number.isNaN(h) || Number.isNaN(m)) return 0
  const deadline = new Date()
  deadline.setHours(h, m, 0, 0)
  return Math.max(0, Math.floor((Date.now() - deadline.getTime()) / 60_000))
}

async function loadOverdue() {
  overdueLoading.value = true
  try {
    const res = await getPickupOverdue({ date: date.value })
    // 后端把学生挂在 by_class[].students 下（无扁平 list），这里摊平供表格使用
    const rows: OverdueRow[] = []
    const byClass: Array<{ class_id: number; class_name: string; count: number }> = []
    for (const g of res.by_class || []) {
      byClass.push({ class_id: g.class_id, class_name: g.class_name, count: g.overdue_count })
      for (const s of g.students || []) {
        rows.push({
          student_id: s.student_id,
          name: s.name,
          class_name: g.class_name,
          pickup_deadline: s.pickup_deadline,
          arrive_time: s.arrive_time,
          overdue_minutes: overdueMinutesOf(s.pickup_deadline),
        })
      }
    }
    overdueList.value = rows
    overdueByClass.value = byClass
    overdueCount.value = res.overdue_total || 0
  } catch (e: unknown) {
    overdueList.value = []
    overdueByClass.value = []
    ElMessage.error(e instanceof Error ? e.message : '加载滞留名单失败')
  } finally {
    overdueLoading.value = false
  }
}

async function handleNotifyOne(row: OverdueRow) {
  try {
    await notifyPickupOverdue({ student_id: row.student_id })
    ElMessage.success(`已通知 ${row.name} 家长`)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '通知失败')
  }
}

async function handleNotifyAll() {
  try {
    await ElMessageBox.confirm(`确定通知全部 ${overdueList.value.length} 位滞留儿童的家长吗？`, '一键通知', {
      confirmButtonText: '通知',
      cancelButtonText: '取消',
      type: 'warning',
    })
    notifyLoading.value = true
    await notifyPickupOverdue({})
    ElMessage.success('已通知全部滞留家长')
  } catch (e: unknown) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e instanceof Error ? e.message : '通知失败')
    }
  } finally {
    notifyLoading.value = false
  }
}

function onTabChange(tabName: string | number) {
  if (tabName === 'records') loadRecords()
  else if (tabName === 'abnormal') loadAbnormal()
  else if (tabName === 'audit') loadAuditList()
  else if (tabName === 'overdue') loadOverdue()
}

function onDateChange() {
  loadOv()
  // 级联刷新当前可见表格，避免日期切换后其他 Tab 仍显示旧数据
  if (activeTab.value === 'records') loadRecords()
  else if (activeTab.value === 'abnormal') loadAbnormal()
  else if (activeTab.value === 'overdue') loadOverdue()
}

// ---- Init ----
onMounted(async () => {
  await loadOv()
  loadRecords()
  loadOverdue()
  // 深链到其他 tab 时补载该 tab 的数据，避免空表
  if (activeTab.value === 'abnormal') loadAbnormal()
  else if (activeTab.value === 'audit') loadAuditList()
})
</script>
