<template>
  <div class="space-y-4">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight text-platform-textPrimary">佣金台账</h1>
      <p class="mt-1 text-sm text-platform-textMuted">
        按实收到账计佣；净额 = 原始佣金 − 已冲回金额，到账满 30 天且净额大于 0 才可结算。
      </p>
    </div>

    <div class="flex flex-wrap items-center gap-2 rounded-xl border border-platform-border bg-platform-card p-4 shadow-platformXs">
      <el-select
        v-model="distributorId"
        clearable
        filterable
        placeholder="全部分销商"
        class="!w-40"
        @change="search"
      >
        <el-option v-for="d in distributors" :key="d.id" :label="d.name" :value="d.id" />
      </el-select>
      <el-date-picker v-model="month" type="month" value-format="YYYY-MM" placeholder="选择月份" class="!w-40" @change="search" />
      <el-select v-model="status" clearable placeholder="全部状态" class="!w-32" @change="search">
        <el-option label="待结算" value="pending" />
        <el-option label="可结算" value="settleable" />
        <el-option label="已结算" value="settled" />
        <el-option label="已冲回" value="reversed" />
      </el-select>
    </div>

    <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
      <div
        v-for="card in cards"
        :key="card.label"
        class="rounded-xl border border-platform-border bg-platform-card p-5 shadow-platformXs"
      >
        <div class="text-sm text-platform-textMuted">{{ card.label }}</div>
        <div class="mt-1.5 text-xl font-semibold tabular-nums text-platform-textPrimary">
          {{ formatMoney(card.value) }}
        </div>
      </div>
    </div>

    <el-table v-loading="loading" :data="rows" class="cursor-pointer" @row-click="openDetail">
      <el-table-column prop="order_no" label="订单号" width="160" />
      <el-table-column prop="kindergarten_name" label="园所" min-width="130" />
      <el-table-column prop="distributor_name" label="分销商" width="90" />
      <el-table-column label="计佣基数" width="110" align="right">
        <template #default="{ row }">{{ formatMoney(row.base_amount_cents) }}</template>
      </el-table-column>
      <el-table-column label="比例" width="70" align="right">
        <template #default="{ row }">{{ row.rate }}%</template>
      </el-table-column>
      <el-table-column label="原始佣金" width="110" align="right">
        <template #default="{ row }">{{ formatMoney(row.amount_cents) }}</template>
      </el-table-column>
      <el-table-column label="已冲回" width="110" align="right">
        <template #default="{ row }">
          <span v-if="row.reversed_amount_cents" class="text-platform-dangerStrong">
            -{{ formatMoney(row.reversed_amount_cents) }}
          </span>
          <span v-else class="text-platform-textHint">—</span>
        </template>
      </el-table-column>
      <el-table-column label="净额" width="110" align="right">
        <template #default="{ row }">
          <span :class="row.net_amount_cents < 0 ? 'text-platform-dangerStrong' : 'font-medium'">
            {{ formatMoney(row.net_amount_cents) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="95">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="可结算日" width="110">
        <template #default="{ row }">{{ formatDate(row.eligible_at) }}</template>
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

    <el-drawer v-model="detailVisible" title="佣金明细" size="520px">
      <div v-loading="detailLoading" class="space-y-4">
        <template v-if="detail">
          <div class="flex items-center justify-between">
            <div>
              <div class="text-base font-semibold text-platform-textPrimary">{{ detail.order_no }}</div>
              <div class="mt-1 text-xs text-platform-textMuted">
                {{ detail.kindergarten_name }} · {{ detail.distributor_name }} · 收款 #{{ detail.payment_id }}
              </div>
            </div>
            <el-tag :type="statusType(detail.status)">{{ statusLabel(detail.status) }}</el-tag>
          </div>

          <div class="grid grid-cols-3 gap-3 text-sm">
            <div class="rounded-lg bg-platform-bg p-3">
              <div class="text-xs text-platform-textMuted">计佣基数</div>
              <div class="mt-1 font-medium text-platform-textPrimary">{{ formatMoney(detail.base_amount_cents) }}</div>
            </div>
            <div class="rounded-lg bg-platform-bg p-3">
              <div class="text-xs text-platform-textMuted">比例</div>
              <div class="mt-1 font-medium text-platform-textPrimary">{{ detail.rate }}%</div>
            </div>
            <div class="rounded-lg bg-platform-bg p-3">
              <div class="text-xs text-platform-textMuted">原始佣金</div>
              <div class="mt-1 font-medium text-platform-textPrimary">{{ formatMoney(detail.amount_cents) }}</div>
            </div>
            <div class="rounded-lg bg-platform-bg p-3">
              <div class="text-xs text-platform-textMuted">已冲回</div>
              <div class="mt-1 font-medium" :class="detail.reversed_amount_cents ? 'text-platform-dangerStrong' : 'text-platform-textPrimary'">
                {{ detail.reversed_amount_cents ? `-${formatMoney(detail.reversed_amount_cents)}` : '—' }}
              </div>
            </div>
            <div class="rounded-lg bg-platform-bg p-3">
              <div class="text-xs text-platform-textMuted">净额</div>
              <div class="mt-1 font-medium" :class="detail.net_amount_cents < 0 ? 'text-platform-dangerStrong' : 'text-platform-textPrimary'">
                {{ formatMoney(detail.net_amount_cents) }}
              </div>
            </div>
            <div class="rounded-lg bg-platform-bg p-3">
              <div class="text-xs text-platform-textMuted">可结算日</div>
              <div class="mt-1 font-medium text-platform-textPrimary">{{ formatDate(detail.eligible_at) }}</div>
            </div>
          </div>

          <div class="text-xs text-platform-textMuted">到账时间：{{ formatDateTime(detail.received_at) }}</div>

          <div>
            <div class="mb-2 text-sm font-medium text-platform-textPrimary">冲回记录</div>
            <el-table :data="detail.reversals || []" size="small" empty-text="无冲回记录">
              <el-table-column label="退款收款" width="100">
                <template #default="{ row }">#{{ row.payment_id }}</template>
              </el-table-column>
              <el-table-column label="冲回基数" width="110" align="right">
                <template #default="{ row }">{{ formatMoney(row.base_amount_cents) }}</template>
              </el-table-column>
              <el-table-column label="冲回金额" width="110" align="right">
                <template #default="{ row }">
                  <span class="text-platform-dangerStrong">{{ formatSignedMoney(row.amount_cents) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="时间" min-width="110">
                <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { getCommission, listCommissions, type CommissionRow, type CommissionSummary } from '~/api/platform/commissions'
import { listDistributors, type DistributorRow } from '~/api/platform/distributors'
import { formatDateTime, formatMoney } from '~/utils/format'

definePageMeta({ layout: 'platform' })

const rows = ref<CommissionRow[]>([])
const month = ref('')
const status = ref('')
const distributorId = ref<number | ''>('')
const distributors = ref<DistributorRow[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const summary = ref<CommissionSummary>({ pending_cents: 0, settleable_cents: 0, settled_cents: 0, reversed_cents: 0 })

const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<CommissionRow | null>(null)

const cards = computed(() => [
  { label: '待结算', value: summary.value.pending_cents },
  { label: '可结算', value: summary.value.settleable_cents },
  { label: '已结算', value: summary.value.settled_cents },
  { label: '累计已冲回', value: summary.value.reversed_cents },
])

async function load() {
  loading.value = true
  try {
    const data = await listCommissions({
      page: page.value,
      page_size: pageSize.value,
      month: month.value || undefined,
      status: status.value || undefined,
      distributor_id: distributorId.value || undefined,
    })
    rows.value = data.list || []
    total.value = data.total || 0
    summary.value = data.summary || summary.value
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

/** 筛选条件变化时回到第 1 页，避免落在不存在的页码上看到空列表。 */
function search() {
  page.value = 1
  void load()
}

function onSizeChange() {
  page.value = 1
  void load()
}

async function loadDistributors() {
  try {
    // status 0 返回全部（含已停用），否则历史佣金无法按分销商回查。
    const data = await listDistributors({ status: 0 })
    distributors.value = data.list || []
  } catch {
    // 分销商选项加载失败不阻断台账列表
  }
}

async function openDetail(row: CommissionRow) {
  // 先清空上一条，避免加载期间抽屉里短暂显示错误的金额与冲回记录。
  detail.value = null
  detailVisible.value = true
  detailLoading.value = true
  try {
    detail.value = await getCommission(row.id)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载佣金明细失败')
  } finally {
    detailLoading.value = false
  }
}

function formatDate(value?: string | null) {
  if (!value) return '—'
  return String(value).slice(0, 10)
}
/** 冲回金额本身为负数：展示成「-¥40.00」而不是「¥-40.00」。 */
function formatSignedMoney(cents: number) {
  return cents < 0 ? `-${formatMoney(-cents)}` : formatMoney(cents)
}
function statusLabel(value: string) {
  return ({ pending: '待结算', settleable: '可结算', settled: '已结算', reversed: '已冲回' } as Record<string, string>)[value] || value
}
function statusType(value: string) {
  // 已冲回是终态，用中性灰；金额已用红色负数表达。
  return (
    { pending: 'info', settleable: 'warning', settled: 'success', reversed: '' } as Record<
      string,
      'success' | 'warning' | 'danger' | 'info' | ''
    >
  )[value] ?? 'info'
}

onMounted(() => {
  void load()
  void loadDistributors()
})
</script>
