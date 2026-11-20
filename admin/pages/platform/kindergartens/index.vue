<template>
  <div class="space-y-4">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-platform-textPrimary">园所</h1>
        <p class="mt-1 text-sm text-platform-textMuted">一个园所 = 一个租户；开通后立即获得 30 天全功能试用。</p>
      </div>
      <el-button type="primary" class="self-start" @click="openCreate">开通园所</el-button>
    </div>

    <div class="flex flex-wrap items-center gap-2 rounded-xl border border-platform-border bg-platform-card p-4 shadow-platformXs">
      <el-input
        v-model="keyword"
        placeholder="搜索园所 / 园长"
        clearable
        class="!w-56"
        @keyup.enter="search"
        @clear="search"
      />
      <el-select v-model="statusFilter" clearable placeholder="全部状态" class="!w-32" @change="search">
        <el-option label="在营" value="active" />
        <el-option label="停用" value="suspended" />
      </el-select>
      <el-button @click="search">查询</el-button>
    </div>

    <el-table v-loading="loading" :data="rows">
      <el-table-column prop="name" label="园所" min-width="160" />
      <el-table-column prop="principal" label="园长" width="110" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : ''" size="small">
            {{ row.status === 'active' ? '在营' : '停用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="幼儿数" width="130" align="right">
        <template #default="{ row }">
          <span>{{ row.student_count }}</span>
          <el-tag v-if="row.over_limit" type="warning" size="small" class="ml-1">超档</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="active_plan_name" label="当前套餐" width="130">
        <template #default="{ row }">{{ row.active_plan_name || '—' }}</template>
      </el-table-column>
      <el-table-column label="最早到期" width="120">
        <template #default="{ row }">{{ formatDate(row.soonest_expire_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="170">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail(row)">详情</el-button>
          <el-button link :type="row.status === 'active' ? 'danger' : 'success'" @click="toggleStatus(row)">
            {{ row.status === 'active' ? '停用' : '启用' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="flex justify-end">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50]"
        :total="total"
        layout="total, sizes, prev, pager, next"
        background
        @current-change="load"
        @size-change="onSizeChange"
      />
    </div>

    <el-dialog v-model="createVisible" title="开通园所" width="480px">
      <el-form ref="formRef" :model="form" :rules="formRules" label-position="top">
        <el-form-item label="园所名称" prop="name">
          <el-input v-model="form.name" placeholder="如：阳光幼儿园" />
        </el-form-item>
        <el-form-item label="园长姓名" prop="principal">
          <el-input v-model="form.principal" placeholder="园长称呼" />
        </el-form-item>
        <el-form-item label="园长手机号" prop="phone">
          <el-input v-model="form.phone" maxlength="11" placeholder="用于登录园所后台" />
        </el-form-item>
        <el-form-item label="初始密码" prop="admin_password">
          <el-input v-model="form.admin_password" type="password" show-password placeholder="含大写、小写与数字，至少 8 位" />
        </el-form-item>
        <el-form-item label="地址">
          <el-input v-model="form.address" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitCreate">开通</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="园所详情" size="520px">
      <div v-loading="detailLoading" class="space-y-5">
        <div v-if="detail">
          <div class="text-lg font-semibold text-platform-textPrimary">{{ detail.name }}</div>
          <div class="mt-1 text-sm text-platform-textMuted">
            园长 {{ detail.principal }} · 在园 {{ detail.student_count }} 人 ·
            {{ detail.status === 'active' ? '在营' : '停用' }}
          </div>
        </div>
        <div>
          <div class="mb-2 text-sm font-medium text-platform-textPrimary">功能授权</div>
          <el-table :data="detail?.entitlements || []" size="small">
            <el-table-column prop="module_name" label="模块" />
            <el-table-column prop="source" label="来源" width="80" />
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag size="small" :type="stateType(row.state)">{{ stateLabel(row.state) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="到期" width="110">
              <template #default="{ row }">{{ formatDate(row.expires_at) }}</template>
            </el-table-column>
          </el-table>
        </div>
        <div>
          <div class="mb-2 text-sm font-medium text-platform-textPrimary">最近订单</div>
          <el-table :data="detail?.recent_orders || []" size="small">
            <el-table-column prop="order_no" label="订单号" />
            <el-table-column label="应收" width="100" align="right">
              <template #default="{ row }">{{ formatMoney(row.amount_due_cents) }}</template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="90" />
          </el-table>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import {
  getKindergarten,
  listKindergartens,
  onboardKindergarten,
  setKindergartenStatus,
  type PlatformKindergartenDetail,
  type PlatformKindergartenRow,
} from '~/api/platform/kindergartens'
import {
  buildKindergartenListQuery,
  validateAdminPassword,
  validatePhone,
} from '~/utils/platformKindergarten'
import { formatMoney } from '~/utils/format'

definePageMeta({ layout: 'platform' })

const keyword = ref('')
const statusFilter = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rows = ref<PlatformKindergartenRow[]>([])
const loading = ref(false)

const createVisible = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({ name: '', principal: '', phone: '', admin_password: '', address: '' })
const formRules: FormRules = {
  name: [{ required: true, message: '请输入园所名称', trigger: 'blur' }],
  principal: [{ required: true, message: '请输入园长姓名', trigger: 'blur' }],
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    {
      validator: (_rule, value, callback) => {
        const error = validatePhone(String(value ?? ''))
        if (error) callback(new Error(error))
        else callback()
      },
      trigger: 'blur',
    },
  ],
  admin_password: [
    { required: true, message: '请输入初始口令', trigger: 'blur' },
    {
      validator: (_rule, value, callback) => {
        const error = validateAdminPassword(String(value ?? ''))
        if (error) callback(new Error(error))
        else callback()
      },
      trigger: 'blur',
    },
  ],
}

const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<PlatformKindergartenDetail | null>(null)
const detailId = ref<number | null>(null)

async function load() {
  loading.value = true
  try {
    const data = await listKindergartens(
      buildKindergartenListQuery({
        page: page.value,
        pageSize: pageSize.value,
        keyword: keyword.value,
        status: statusFilter.value,
      }),
    )
    rows.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

/** 关键词或状态筛选变化时回到第 1 页，避免落在不存在的页码上看到空列表。 */
function search() {
  page.value = 1
  void load()
}

function onSizeChange() {
  page.value = 1
  void load()
}

function openCreate() {
  Object.assign(form, { name: '', principal: '', phone: '', admin_password: '', address: '' })
  createVisible.value = true
}

async function submitCreate() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    const res = await onboardKindergarten({ ...form })
    ElMessage.success(`已开通，园所 ID ${res.kindergarten_id}，请把初始口令告知园长`)
    createVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '开通失败')
  } finally {
    submitting.value = false
  }
}

async function openDetail(row: PlatformKindergartenRow) {
  detailId.value = row.id
  // 先清空上一所园的数据，避免加载期间抽屉里短暂显示错误的授权/订单。
  detail.value = null
  detailVisible.value = true
  await refreshDetail()
}

async function refreshDetail() {
  if (detailId.value == null) return
  detailLoading.value = true
  try {
    detail.value = await getKindergarten(detailId.value)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function toggleStatus(row: PlatformKindergartenRow) {
  const next = row.status === 'active' ? 'suspended' : 'active'
  if (next === 'suspended') {
    try {
      await ElMessageBox.confirm(
        `停用「${row.name}」后，该园全部园所端会话立即不可用。确定停用？`,
        '停用园所',
        { type: 'warning', confirmButtonText: '停用', cancelButtonText: '取消' },
      )
    } catch {
      return
    }
  }
  try {
    await setKindergartenStatus(row.id, next)
    ElMessage.success(next === 'active' ? '已启用' : '已停用')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

function formatDate(value?: string | null) {
  if (!value) return '—'
  return String(value).slice(0, 10)
}
function stateLabel(state: string) {
  return state === 'active' ? '在期' : state === 'grace' ? '宽限' : '已降级'
}
function stateType(state: string) {
  return state === 'active' ? 'success' : state === 'grace' ? 'warning' : 'danger'
}

onMounted(load)
</script>
