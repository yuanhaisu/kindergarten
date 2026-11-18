<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-text-primary">运营看板</h1>
        <p class="mt-1 text-xs text-text-muted">
          园所、收款与佣金的全局概览；数字为打开页面时的一次聚合快照。
        </p>
      </div>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </div>

    <el-alert v-if="error" type="error" :closable="false" show-icon>
      <template #title>{{ error }}</template>
      <el-button link type="primary" @click="load">重试</el-button>
    </el-alert>

    <div v-loading="loading && !data">
      <template v-if="data">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <StatCard
            title="园所总数"
            :value="data.kindergarten_total"
            :icon="OfficeBuilding"
            icon-bg="bg-blue-50"
            icon-color="var(--color-brand-primary)"
          />
          <StatCard
            title="在营园所"
            :value="data.kindergarten_active"
            :icon="CircleCheck"
            icon-bg="bg-emerald-50"
            icon-color="var(--color-semantic-success)"
          />
          <StatCard
            title="30 天内到期"
            :value="data.kindergarten_expiring_30d"
            :icon="AlarmClock"
            icon-bg="bg-amber-50"
            icon-color="var(--color-semantic-warning)"
          />
          <StatCard
            title="本月实收"
            :value="formatMoney(data.received_this_month_cents)"
            :icon="Wallet"
            icon-bg="bg-violet-50"
            icon-color="var(--color-accent-purple)"
          />
          <StatCard
            title="未收余额"
            :value="formatMoney(data.outstanding_cents)"
            :icon="Money"
            icon-bg="bg-rose-50"
            icon-color="var(--color-semantic-danger)"
          />
          <StatCard
            title="待结算佣金"
            :value="formatMoney(data.commission_settleable_cents)"
            :icon="Coin"
            icon-bg="bg-cyan-50"
            icon-color="var(--color-semantic-info)"
          />
        </div>

        <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div class="rounded-lg bg-bg-card p-4 shadow-sm">
            <div class="flex items-start justify-between">
              <div>
                <div class="text-sm font-medium text-text-primary">7 天内到期</div>
                <div class="mt-0.5 text-xs text-text-muted">需要联系续费的在营园所</div>
              </div>
              <NuxtLink to="/platform/kindergartens" class="text-xs text-brand-primaryDark hover:underline">
                园所列表
              </NuxtLink>
            </div>
            <div
              v-if="!data.expiring_kindergartens.length"
              class="py-10 text-center text-xs text-text-hint"
            >
              近 7 天没有到期园所
            </div>
            <ul v-else class="mt-2 divide-y divide-border-subtle">
              <li
                v-for="kg in data.expiring_kindergartens"
                :key="kg.id"
                class="flex items-center justify-between gap-3 py-2.5"
              >
                <div class="min-w-0">
                  <div class="truncate text-sm text-text-primary">{{ kg.name }}</div>
                  <div class="mt-0.5 text-xs text-text-muted">到期 {{ formatDate(kg.expires_at) }}</div>
                </div>
                <el-tag :type="daysLeft(kg.expires_at) <= 3 ? 'danger' : 'warning'" size="small">
                  {{ daysLeftLabel(kg.expires_at) }}
                </el-tag>
              </li>
            </ul>
          </div>

          <div class="rounded-lg bg-bg-card p-4 shadow-sm">
            <div class="flex items-start justify-between">
              <div>
                <div class="text-sm font-medium text-text-primary">可结算佣金</div>
                <div class="mt-0.5 text-xs text-text-muted">按分销商聚合的当前可结算净额</div>
              </div>
              <NuxtLink to="/platform/commissions" class="text-xs text-brand-primaryDark hover:underline">
                佣金台账
              </NuxtLink>
            </div>
            <div
              v-if="!data.settleable_commissions.length"
              class="py-10 text-center text-xs text-text-hint"
            >
              当前没有可结算佣金
            </div>
            <ul v-else class="mt-2 divide-y divide-border-subtle">
              <li
                v-for="row in data.settleable_commissions"
                :key="row.distributor_id"
                class="flex items-center justify-between gap-3 py-2.5"
              >
                <div class="min-w-0">
                  <div class="truncate text-sm text-text-primary">{{ row.distributor_name || '—' }}</div>
                  <div class="mt-0.5 text-xs text-text-muted">{{ row.commission_count }} 笔佣金</div>
                </div>
                <div class="shrink-0 text-sm font-medium text-text-primary tabular-nums">
                  {{ formatMoney(row.amount_cents) }}
                </div>
              </li>
            </ul>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { AlarmClock, CircleCheck, Coin, Money, OfficeBuilding, Wallet } from '@element-plus/icons-vue'
import { getDashboard, type PlatformDashboard } from '~/api/platform/dashboard'
import { formatDate, formatMoney } from '~/utils/format'

definePageMeta({ layout: 'platform' })

const data = ref<PlatformDashboard | null>(null)
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    data.value = await getDashboard()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载看板失败'
  } finally {
    loading.value = false
  }
}

/** 待办窗口最多 7 天，按剩余整天展示；当天到期显示 0。 */
function daysLeft(iso: string) {
  const diff = new Date(iso).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / 86_400_000))
}

function daysLeftLabel(iso: string) {
  const days = daysLeft(iso)
  return days === 0 ? '今天到期' : `${days} 天后到期`
}

onMounted(load)
</script>
