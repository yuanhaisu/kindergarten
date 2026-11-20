<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-platform-textPrimary">运营看板</h1>
        <p class="mt-1 text-sm text-platform-textMuted">
          园所、收款与佣金的全局概览；数字为打开页面时的一次聚合快照。
        </p>
      </div>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </div>

    <el-alert v-if="error" type="error" :closable="false" show-icon>
      <template #title>{{ error }}</template>
      <el-button link type="primary" @click="load">重试</el-button>
    </el-alert>

    <div v-loading="loading && !data" class="min-h-[240px]">
      <template v-if="data">
        <div class="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-3">
          <PlatformStatCard
            title="园所总数"
            :value="data.kindergarten_total"
            :icon="OfficeBuilding"
            tone="brand"
            unit="所"
          />
          <PlatformStatCard
            title="在营园所"
            :value="data.kindergarten_active"
            :icon="CircleCheck"
            tone="success"
            unit="所"
          />
          <PlatformStatCard
            title="30 天内到期"
            :value="data.kindergarten_expiring_30d"
            :icon="AlarmClock"
            tone="warning"
            unit="所"
          />
          <PlatformStatCard
            title="本月实收"
            :value="formatMoney(data.received_this_month_cents)"
            :icon="Wallet"
            tone="brand"
          />
          <PlatformStatCard
            title="未收余额"
            :value="formatMoney(data.outstanding_cents)"
            :icon="Money"
            tone="danger"
          />
          <PlatformStatCard
            title="待结算佣金"
            :value="formatMoney(data.commission_settleable_cents)"
            :icon="Coin"
            tone="info"
          />
        </div>

        <div class="mt-6 grid grid-cols-1 gap-4 lg:grid-cols-2">
          <section class="overflow-hidden rounded-xl border border-platform-border bg-platform-card shadow-platformXs">
            <header class="flex items-start justify-between gap-3 border-b border-platform-borderLight px-5 py-4">
              <div>
                <h2 class="text-base font-semibold text-platform-textPrimary">7 天内到期</h2>
                <p class="mt-0.5 text-sm text-platform-textMuted">需要联系续费的在营园所</p>
              </div>
              <NuxtLink
                to="/platform/kindergartens"
                class="shrink-0 text-sm font-medium text-platform-brand hover:underline"
              >
                园所列表
              </NuxtLink>
            </header>
            <div
              v-if="!data.expiring_kindergartens.length"
              class="px-5 py-12 text-center text-sm text-platform-textMuted"
            >
              近 7 天没有到期园所
            </div>
            <ul v-else class="divide-y divide-platform-borderLight">
              <li
                v-for="kg in data.expiring_kindergartens"
                :key="kg.id"
                class="flex items-center justify-between gap-3 px-5 py-3"
              >
                <div class="min-w-0">
                  <div class="truncate text-sm font-medium text-platform-textPrimary">{{ kg.name }}</div>
                  <div class="mt-0.5 text-xs text-platform-textMuted">到期 {{ formatDate(kg.expires_at) }}</div>
                </div>
                <el-tag :type="daysLeft(kg.expires_at) <= 3 ? 'danger' : 'warning'">
                  {{ daysLeftLabel(kg.expires_at) }}
                </el-tag>
              </li>
            </ul>
          </section>

          <section class="overflow-hidden rounded-xl border border-platform-border bg-platform-card shadow-platformXs">
            <header class="flex items-start justify-between gap-3 border-b border-platform-borderLight px-5 py-4">
              <div>
                <h2 class="text-base font-semibold text-platform-textPrimary">可结算佣金</h2>
                <p class="mt-0.5 text-sm text-platform-textMuted">按分销商聚合的当前可结算净额</p>
              </div>
              <NuxtLink
                to="/platform/commissions"
                class="shrink-0 text-sm font-medium text-platform-brand hover:underline"
              >
                佣金台账
              </NuxtLink>
            </header>
            <div
              v-if="!data.settleable_commissions.length"
              class="px-5 py-12 text-center text-sm text-platform-textMuted"
            >
              当前没有可结算佣金
            </div>
            <ul v-else class="divide-y divide-platform-borderLight">
              <li
                v-for="row in data.settleable_commissions"
                :key="row.distributor_id"
                class="flex items-center justify-between gap-3 px-5 py-3"
              >
                <div class="min-w-0">
                  <div class="truncate text-sm font-medium text-platform-textPrimary">
                    {{ row.distributor_name || '—' }}
                  </div>
                  <div class="mt-0.5 text-xs text-platform-textMuted">{{ row.commission_count }} 笔佣金</div>
                </div>
                <div class="shrink-0 text-sm font-semibold tabular-nums text-platform-textPrimary">
                  {{ formatMoney(row.amount_cents) }}
                </div>
              </li>
            </ul>
          </section>
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
