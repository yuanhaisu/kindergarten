<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-text-primary">订单</h1>
        <p class="mt-1 text-xs text-text-muted">
          订单确认时冻结套餐快照；收款状态由已收金额推导，退款以负数登记。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <el-button type="primary" @click="openCreate">新建订单</el-button>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-2 rounded-md bg-bg-card p-3 shadow-sm">
      <el-input
        v-model="orderNo"
        placeholder="订单号"
        clearable
        class="!w-48"
        @keyup.enter="search"
        @clear="search"
      />
      <el-select v-model="kindergartenFilter" clearable filterable placeholder="全部园所" class="!w-48" @change="search">
        <el-option v-for="kg in kindergartens" :key="kg.id" :label="kg.name" :value="kg.id" />
      </el-select>
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        value-format="YYYY-MM-DD"
        start-placeholder="创建开始"
        end-placeholder="创建结束"
        class="!w-72"
        @change="search"
      />
      <el-select v-model="status" clearable placeholder="全部状态" class="!w-32" @change="search">
        <el-option label="待收款" value="unpaid" />
        <el-option label="部分收款" value="partial" />
        <el-option label="已收清" value="paid" />
        <el-option label="已到期" value="expired" />
        <el-option label="已取消" value="cancelled" />
      </el-select>
      <el-checkbox v-model="outstandingOnly" @change="search">只看未收清</el-checkbox>
      <el-button @click="search">查询</el-button>
      <el-button text @click="resetFilters">重置</el-button>
    </div>

    <el-table v-loading="loading" :data="rows" border stripe>
      <el-table-column prop="order_no" label="订单号" width="180" />
      <el-table-column prop="kindergarten_name" label="园所" min-width="140" />
      <el-table-column prop="plan_name" label="套餐" width="120" />
      <el-table-column label="应收" width="110" align="right">
        <template #default="{ row }">{{ formatMoney(row.amount_due_cents) }}</template>
      </el-table-column>
      <el-table-column label="已收" width="110" align="right">
        <template #default="{ row }">{{ formatMoney(row.amount_paid_cents) }}</template>
      </el-table-column>
      <el-table-column label="未清" width="110" align="right">
        <template #default="{ row }">
          <span :class="outstandingCents(row) > 0 ? 'font-medium text-semantic-danger' : 'text-text-hint'">
            {{ outstandingCents(row) > 0 ? formatMoney(outstandingCents(row)) : '—' }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="标记" width="100">
        <template #default="{ row }">
          <el-tag v-if="row.credit_granted" type="warning" size="small">信用开通</el-tag>
          <span v-else class="text-text-hint">—</span>
        </template>
      </el-table-column>
      <el-table-column label="付款到期" width="116">
        <template #default="{ row }">
          <span :class="row.status === 'expired' ? 'text-semantic-danger' : ''">
            {{ formatDate(row.pay_due_at) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="服务期至" width="116">
        <template #default="{ row }">{{ formatDate(row.expires_at) }}</template>
      </el-table-column>
      <el-table-column prop="distributor_name" label="分销商" width="120">
        <template #default="{ row }">{{ row.distributor_name || '—' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="90" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail(row.id)">详情</el-button>
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

    <el-dialog v-model="createVisible" title="新建订单" width="480px">
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="90px">
        <el-form-item label="园所" prop="kindergarten_id">
          <el-select v-model="form.kindergarten_id" filterable placeholder="选择园所" class="w-full">
            <el-option v-for="kg in kindergartens" :key="kg.id" :label="kg.name" :value="kg.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="套餐" prop="plan_id">
          <el-select v-model="form.plan_id" placeholder="选择套餐" class="w-full">
            <el-option v-for="p in plans" :key="p.id" :label="`${p.name}（${formatMoney(p.price_cents)}）`" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="分销商">
          <el-select v-model="form.distributor_id" clearable placeholder="无" class="w-full">
            <el-option v-for="d in distributors" :key="d.id" :label="`${d.name}（${d.commission_rate}%）`" :value="d.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="订单详情" size="560px">
      <div v-loading="detailLoading" class="space-y-4">
        <template v-if="detail">
          <div class="flex items-center justify-between">
            <div>
              <div class="text-base font-semibold text-text-primary">{{ detail.order_no }}</div>
              <div class="mt-1 text-xs text-text-muted">
                {{ detail.kindergarten_name }} · {{ detail.plan_snapshot.plan_name }}
              </div>
            </div>
            <div class="flex items-center gap-2">
              <el-tag v-if="detail.credit_granted" type="warning">信用开通</el-tag>
              <el-tag :type="statusType(detail.status)">{{ statusLabel(detail.status) }}</el-tag>
            </div>
          </div>

          <div class="grid grid-cols-3 gap-3 text-sm">
            <div class="rounded-md bg-bg-page p-3">
              <div class="text-xs text-text-muted">应收</div>
              <div class="mt-1 font-medium text-text-primary">{{ formatMoney(detail.amount_due_cents) }}</div>
            </div>
            <div class="rounded-md bg-bg-page p-3">
              <div class="text-xs text-text-muted">已收</div>
              <div class="mt-1 font-medium text-text-primary">{{ formatMoney(detail.amount_paid_cents) }}</div>
            </div>
            <div class="rounded-md bg-bg-page p-3">
              <div class="text-xs text-text-muted">未清</div>
              <div class="mt-1 font-medium text-text-primary">
                {{ formatMoney(Math.max(0, detail.amount_due_cents - detail.amount_paid_cents)) }}
              </div>
            </div>
          </div>

          <div class="text-xs text-text-muted">
            服务期：{{ formatDate(detail.starts_at) }} — {{ formatDate(detail.expires_at) }}
            · 付款到期：<span :class="detail.status === 'expired' ? 'text-semantic-danger' : ''">{{ formatDate(detail.pay_due_at) }}</span>
            <template v-if="detail.distributor">
              · 分销商 {{ detail.distributor.name }}（{{ detail.distributor.commission_rate }}%）
            </template>
          </div>

          <div v-if="detail.credit_granted" class="rounded-md bg-bg-page p-3 text-sm">
            <div class="text-xs text-text-muted">信用开通理由</div>
            <div class="mt-1 text-text-primary">{{ detail.credit_reason || '—' }}</div>
          </div>

          <div class="flex flex-wrap gap-2">
            <el-button v-if="canConfirm(detail.status)" size="small" type="primary" @click="onConfirm">
              确认订单
            </el-button>
            <el-button v-if="!detail.status.startsWith('cancelled')" size="small" type="success" @click="openPayment">
              登记收款
            </el-button>
            <el-button v-if="!detail.status.startsWith('cancelled')" size="small" type="danger" @click="onCancel">
              取消订单
            </el-button>
          </div>

          <div>
            <div class="mb-2 text-sm font-medium text-text-primary">收款流水</div>
            <el-table :data="detail.payments" size="small" border>
              <el-table-column label="金额" width="110" align="right">
                <template #default="{ row }">
                  <span :class="row.amount_cents < 0 ? 'text-semantic-danger' : ''">{{ formatMoney(row.amount_cents) }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="method" label="方式" width="110" />
              <el-table-column label="关联" width="100">
                <template #default="{ row }">
                  <span v-if="row.refund_of_payment_id" class="text-xs text-text-muted">
                    退 #{{ row.refund_of_payment_id }}
                  </span>
                  <span v-else class="text-text-hint">—</span>
                </template>
              </el-table-column>
              <el-table-column prop="remark" label="备注" />
              <el-table-column label="到账" width="110">
                <template #default="{ row }">{{ formatDate(row.received_at) }}</template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </div>
    </el-drawer>

    <el-dialog v-model="paymentVisible" title="登记收款" width="420px">
      <el-form label-width="90px">
        <el-form-item label="金额（元）">
          <el-input-number v-model="paymentYuan" :precision="2" :step="100" class="w-full" />
          <div class="mt-1 text-xs text-text-muted">退款请填负数。</div>
        </el-form-item>
        <el-form-item v-if="paymentYuan < 0" label="被退收款" required>
          <el-select v-model="refundOfPaymentID" class="w-full" placeholder="选择被退的原收款">
            <el-option
              v-for="p in refundablePayments"
              :key="p.id"
              :label="`#${p.id} ${formatMoney(p.amount_cents)} ${formatDate(p.received_at)}`"
              :value="p.id"
            />
          </el-select>
          <div class="mt-1 text-xs text-text-muted">冲回该笔收款产生的佣金。</div>
        </el-form-item>
        <el-form-item label="方式">
          <el-select v-model="paymentMethod" class="w-full">
            <el-option label="对公转账" value="bank_transfer" />
            <el-option label="微信" value="wechat" />
            <el-option label="现金" value="cash" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="paymentRemark" placeholder="如：首付 / 尾款" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="paymentVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitPayment">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { addPayment, cancelOrder, confirmOrder, createOrder, getOrder, listOrders, type OrderDetail, type OrderRow } from '~/api/platform/orders'
import { listDistributors, type DistributorRow } from '~/api/platform/distributors'
import { listKindergartens, type PlatformKindergartenRow } from '~/api/platform/kindergartens'
import { listPlans, type PlanRow } from '~/api/platform/plans'
import { buildOrderListParams } from '~/utils/platformOrders'

definePageMeta({ layout: 'platform' })

const rows = ref<OrderRow[]>([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const status = ref('')
const orderNo = ref('')
const kindergartenFilter = ref<number>()
const dateRange = ref<[string, string] | null>(null)
const outstandingOnly = ref(false)
const loading = ref(false)
const submitting = ref(false)

const kindergartens = ref<PlatformKindergartenRow[]>([])
const plans = ref<PlanRow[]>([])
const distributors = ref<DistributorRow[]>([])

const createVisible = ref(false)
const formRef = ref<FormInstance>()
const form = reactive<{ kindergarten_id?: number; plan_id?: number; distributor_id?: number }>({})
const formRules: FormRules = {
  kindergarten_id: [{ required: true, message: '请选择园所', trigger: 'change' }],
  plan_id: [{ required: true, message: '请选择套餐', trigger: 'change' }],
}

const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<OrderDetail | null>(null)

const paymentVisible = ref(false)
const paymentYuan = ref(0)
const paymentMethod = ref('bank_transfer')
const paymentRemark = ref('')
const refundOfPaymentID = ref<number>()
// 每次打开弹窗生成一次，窗口内的重试复用同一个键：响应丢失后重试不会重复记账。
const paymentIdempotencyKey = ref('')

// 只有正数收款可以被退款；退款必须指明被退的那一笔。
const refundablePayments = computed(() =>
  (detail.value?.payments || []).filter((p) => p.amount_cents > 0),
)

async function load() {
  loading.value = true
  try {
    const data = await listOrders(
      buildOrderListParams({
        page: page.value,
        pageSize,
        orderNo: orderNo.value,
        dateRange: dateRange.value,
        kindergartenId: kindergartenFilter.value,
        status: status.value,
        outstandingOnly: outstandingOnly.value,
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

/** 筛选变化时回到第 1 页，避免落在不存在的页码上看到空列表。 */
function search() {
  page.value = 1
  void load()
}

function resetFilters() {
  orderNo.value = ''
  kindergartenFilter.value = undefined
  dateRange.value = null
  status.value = ''
  outstandingOnly.value = false
  search()
}

async function loadOptions() {
  try {
    const [kgs, ps, ds] = await Promise.all([
      listKindergartens({ page: 1, page_size: 100 }),
      listPlans(),
      listDistributors({ status: 1 }),
    ])
    kindergartens.value = kgs.list || []
    plans.value = (ps.list || []).filter((p) => p.status === 'active')
    distributors.value = ds.list || []
  } catch {
    // 选项加载失败不阻断列表
  }
}

function openCreate() {
  Object.assign(form, { kindergarten_id: undefined, plan_id: undefined, distributor_id: undefined })
  createVisible.value = true
}

async function submitCreate() {
  await formRef.value?.validate().catch(() => Promise.reject())
  submitting.value = true
  try {
    const res = await createOrder({ ...form })
    ElMessage.success(`已创建 ${res.order_no}`)
    createVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '创建失败')
  } finally {
    submitting.value = false
  }
}

async function openDetail(id: number) {
  detailVisible.value = true
  detailLoading.value = true
  try {
    detail.value = await getOrder(id)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function refreshDetail() {
  if (detail.value) await openDetail(detail.value.id)
}

/**
 * 确认按钮只在「尚未收清且未结束」的订单上出现：
 * 已收清（paid）与已过付款到期日（expired）不再提供确认，已取消的订单由调用处的
 * cancelled 判断隐藏全部操作（design D2 / D9）。
 */
function canConfirm(status: string) {
  return status !== 'paid' && status !== 'expired' && !status.startsWith('cancelled')
}

async function onConfirm() {
  if (!detail.value) return
  // 净收款为零的确认即信用开通：理由由后端强制要求（422），
  // 这里先收口，避免把一次必然失败的请求发出去。
  let reason: string | undefined
  if (detail.value.amount_paid_cents === 0) {
    try {
      const res = await ElMessageBox.prompt('零收款确认将标记为「信用开通」。请填写理由：', '确认订单', {
        inputPlaceholder: '如：合同已签署，对公转账到账后补录',
        inputValidator: (value: string) => (value?.trim() ? true : '必须填写信用开通理由'),
      })
      reason = res.value?.trim()
    } catch {
      return
    }
  }
  try {
    if (reason) {
      await confirmOrder(detail.value.id, reason)
    } else {
      await confirmOrder(detail.value.id)
    }
    ElMessage.success('订单已确认，授权已发放')
    await Promise.all([refreshDetail(), load()])
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '确认失败')
  }
}

async function onCancel() {
  if (!detail.value) return
  let reason = ''
  try {
    const res = await ElMessageBox.prompt('取消后该订单产生的授权会立即失效。请填写原因：', '取消订单', {
      inputPlaceholder: '如：园所暂缓采购',
    })
    reason = res.value ?? ''
  } catch {
    return
  }
  try {
    await cancelOrder(detail.value.id, reason)
    ElMessage.success('订单已取消')
    await Promise.all([refreshDetail(), load()])
  } catch (e) {
    // 有未退款收款时后端拒绝取消并回传待退金额，这里直接展示，避免运营反复试错。
    const unrefunded = (e as Error & { data?: { unrefunded_amount_cents?: number } }).data
      ?.unrefunded_amount_cents
    if (typeof unrefunded === 'number') {
      ElMessageBox.alert(
        `订单仍有未退款收款 ${formatMoney(unrefunded)}，请先在「登记收款」中以负数登记退款，再取消订单。`,
        '无法取消',
        { type: 'warning', confirmButtonText: '知道了' },
      ).catch(() => {})
    } else {
      ElMessage.error(e instanceof Error ? e.message : '取消失败')
    }
  }
}

function newIdempotencyKey() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `pay-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}

function openPayment() {
  paymentYuan.value = 0
  paymentMethod.value = 'bank_transfer'
  paymentRemark.value = ''
  refundOfPaymentID.value = undefined
  paymentIdempotencyKey.value = newIdempotencyKey()
  paymentVisible.value = true
}

async function submitPayment() {
  if (!detail.value) return
  const amountCents = Math.round(paymentYuan.value * 100)
  if (amountCents === 0) {
    ElMessage.warning('金额不能为 0')
    return
  }
  if (amountCents < 0 && !refundOfPaymentID.value) {
    ElMessage.warning('退款请选择被退的原收款')
    return
  }
  const body: Record<string, unknown> = {
    amount_cents: amountCents,
    method: paymentMethod.value,
    remark: paymentRemark.value,
  }
  if (amountCents < 0) body.refund_of_payment_id = refundOfPaymentID.value
  submitting.value = true
  try {
    await addPayment(detail.value.id, body, paymentIdempotencyKey.value)
    ElMessage.success('收款已登记')
    paymentVisible.value = false
    await Promise.all([refreshDetail(), load()])
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '登记失败')
  } finally {
    submitting.value = false
  }
}

function formatMoney(cents: number) {
  return `¥${(cents / 100).toFixed(2)}`
}
function outstandingCents(row: OrderRow) {
  return Math.max(0, row.amount_due_cents - row.amount_paid_cents)
}
function formatDate(value?: string | null) {
  if (!value) return '—'
  return String(value).slice(0, 10)
}
function statusLabel(value: string) {
  return (
    { unpaid: '待收款', partial: '部分收款', paid: '已收清', expired: '已到期', cancelled: '已取消' } as Record<string, string>
  )[value] || value
}
function statusType(value: string) {
  return (
    { unpaid: 'info', partial: 'warning', paid: 'success', expired: 'danger', cancelled: 'info' } as Record<
      string,
      'success' | 'warning' | 'danger' | 'info'
    >
  )[value] || 'info'
}

onMounted(() => {
  load()
  loadOptions()
})
</script>
