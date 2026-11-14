<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-text-primary">佣金台账</h1>
        <p class="mt-1 text-xs text-text-muted">
          按实收到账计佣；净额 = 原始佣金 − 已冲回金额，到账满 30 天且净额大于 0 才可结算。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <el-date-picker v-model="month" type="month" value-format="YYYY-MM" placeholder="选择月份" class="!w-40" @change="load" />
        <el-select v-model="status" clearable placeholder="全部状态" class="!w-32" @change="load">
          <el-option label="待结算" value="pending" />
          <el-option label="可结算" value="settleable" />
          <el-option label="已结算" value="settled" />
          <el-option label="已冲回" value="reversed" />
        </el-select>
      </div>
    </div>

    <div class="grid grid-cols-4 gap-3">
      <div v-for="card in cards" :key="card.label" class="rounded-lg bg-bg-card p-4 shadow-sm">
        <div class="text-xs text-text-muted">{{ card.label }}</div>
        <div class="mt-1 text-lg font-semibold text-text-primary">{{ formatMoney(card.value) }}</div>
      </div>
    </div>

    <el-table v-loading="loading" :data="rows" border stripe>
      <el-table-column prop="order_no" label="订单号" width="180" />
      <el-table-column prop="kindergarten_name" label="园所" min-width="140" />
      <el-table-column prop="distributor_name" label="分销商" width="130" />
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
          <span v-if="row.reversed_amount_cents" class="text-semantic-danger">
            -{{ formatMoney(row.reversed_amount_cents) }}
          </span>
          <span v-else class="text-text-hint">—</span>
        </template>
      </el-table-column>
      <el-table-column label="净额" width="110" align="right">
        <template #default="{ row }">
          <span :class="row.net_amount_cents < 0 ? 'text-semantic-danger' : 'font-medium'">
            {{ formatMoney(row.net_amount_cents) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="可结算日" width="110">
        <template #default="{ row }">{{ formatDate(row.eligible_at) }}</template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-model:current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="total, prev, pager, next"
      @current-change="load"
    />
  </div>
</template>

<script setup lang="ts">
import { listCommissions, type CommissionRow, type CommissionSummary } from '~/api/platform'

definePageMeta({ layout: 'platform' })

const rows = ref<CommissionRow[]>([])
const month = ref('')
const status = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const loading = ref(false)
const summary = ref<CommissionSummary>({ pending_cents: 0, settleable_cents: 0, settled_cents: 0, reversed_cents: 0 })

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
      page: page.value, page_size: pageSize, month: month.value || undefined, status: status.value || undefined,
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

function formatMoney(cents: number) {
  return `¥${(cents / 100).toFixed(2)}`
}
function formatDate(value?: string | null) {
  if (!value) return '—'
  return String(value).slice(0, 10)
}
function statusLabel(value: string) {
  return ({ pending: '待结算', settleable: '可结算', settled: '已结算', reversed: '已冲回' } as Record<string, string>)[value] || value
}
function statusType(value: string) {
  return (
    { pending: 'info', settleable: 'warning', settled: 'success', reversed: 'danger' } as Record<
      string,
      'success' | 'warning' | 'danger' | 'info'
    >
  )[value] || 'info'
}

onMounted(load)
</script>
