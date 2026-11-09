<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-text-primary">园所</h1>
        <p class="mt-1 text-xs text-text-muted">一个园所 = 一个租户；开通后立即获得 30 天全功能试用。</p>
      </div>
      <div class="flex items-center gap-2">
        <el-input v-model="keyword" placeholder="搜索园所 / 园长" clearable class="!w-56" @keyup.enter="load" />
        <el-button @click="load">查询</el-button>
        <el-button type="primary" @click="openCreate">开通园所</el-button>
      </div>
    </div>

    <el-table v-loading="loading" :data="rows" border stripe>
      <el-table-column prop="name" label="园所" min-width="160" />
      <el-table-column prop="principal" label="园长" width="110" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
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
      <el-table-column label="操作" width="170" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail(row.id)">详情</el-button>
          <el-button link :type="row.status === 'active' ? 'danger' : 'success'" @click="toggleStatus(row)">
            {{ row.status === 'active' ? '停用' : '启用' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-model:current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="total, prev, pager, next"
      @current-change="load"
    />

    <el-dialog v-model="createVisible" title="开通园所" width="480px">
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="96px">
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
          <div class="text-lg font-semibold text-text-primary">{{ detail.name }}</div>
          <div class="mt-1 text-sm text-text-muted">
            园长 {{ detail.principal }} · 在园 {{ detail.student_count }} 人 ·
            {{ detail.status === 'active' ? '在营' : '停用' }}
          </div>
        </div>
        <div>
          <div class="mb-2 text-sm font-medium text-text-primary">功能授权</div>
          <el-table :data="detail?.entitlements || []" size="small" border>
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
          <div class="mb-2 text-sm font-medium text-text-primary">最近订单</div>
          <el-table :data="detail?.recent_orders || []" size="small" border>
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
} from '~/api/platform'

definePageMeta({ layout: 'platform' })

const keyword = ref('')
const page = ref(1)
const pageSize = 20
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
  phone: [{ required: true, message: '请输入手机号', trigger: 'blur' }],
  admin_password: [{ required: true, min: 8, message: '至少 8 位', trigger: 'blur' }],
}

const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<PlatformKindergartenDetail | null>(null)

async function load() {
  loading.value = true
  try {
    const data = await listKindergartens({ page: page.value, page_size: pageSize, keyword: keyword.value })
    rows.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { name: '', principal: '', phone: '', admin_password: '', address: '' })
  createVisible.value = true
}

async function submitCreate() {
  await formRef.value?.validate().catch(() => Promise.reject())
  submitting.value = true
  try {
    const res = await onboardKindergarten({ ...form })
    ElMessage.success(`已开通，园所 ID ${res.kindergarten_id}，请把初始密码告知园长`)
    createVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '开通失败')
  } finally {
    submitting.value = false
  }
}

async function openDetail(id: number) {
  detailVisible.value = true
  detailLoading.value = true
  try {
    detail.value = await getKindergarten(id)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function toggleStatus(row: PlatformKindergartenRow) {
  const next = row.status === 'active' ? 'suspended' : 'active'
  try {
    await setKindergartenStatus(row.id, next)
    ElMessage.success(next === 'active' ? '已启用' : '已停用')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

function formatMoney(cents: number) {
  return `¥${(cents / 100).toFixed(2)}`
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
