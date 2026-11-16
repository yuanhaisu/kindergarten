<template>
  <div class="space-y-4">
    <div>
      <h1 class="text-xl font-semibold text-text-primary">我的套餐</h1>
      <p class="mt-1 text-xs text-text-muted">这里只展示本园当前可用的功能与到期时间；续费请联系平台方。</p>
    </div>

    <div v-loading="loading" class="space-y-4">
      <el-alert
        v-if="expiringNotice"
        type="warning"
        :closable="false"
        show-icon
        title="套餐即将到期"
      >
        <template #default>
          套餐将于 {{ expiringNotice.expiresOn }} 到期，请联系平台方续费。
        </template>
      </el-alert>

      <el-alert
        v-if="view && view.state !== 'active'"
        :type="view.state === 'grace' ? 'warning' : 'error'"
        :closable="false"
        show-icon
        :title="view.state === 'grace' ? '套餐已到期，处于宽限期' : '套餐已到期，部分功能已停用'"
      >
        <template #default>
          <span v-if="view.state === 'grace'">宽限期至 {{ formatDate(view.grace_until) }}，请尽快联系平台方续费。</span>
          <span v-else>历史内容仍可查看，新增功能已暂停。请联系平台方续费后恢复。</span>
        </template>
      </el-alert>

      <el-alert
        v-if="view?.over_limit"
        type="warning"
        :closable="false"
        show-icon
        title="在园幼儿数已超过所购档位上限"
      >
        <template #default>
          当前在园 {{ view.student_count }} 人，所购档位上限 {{ view.tier_max_students }} 人。
          现有考勤、成长、接送等功能不受影响；如需扩展规模，请联系平台方升级套餐。
        </template>
      </el-alert>

      <div class="grid grid-cols-3 gap-3">
        <div class="rounded-lg bg-bg-card p-4 shadow-sm">
          <div class="text-xs text-text-muted">当前套餐</div>
          <div class="mt-1 text-lg font-semibold text-text-primary">{{ view?.plan_name || '试用中' }}</div>
        </div>
        <div class="rounded-lg bg-bg-card p-4 shadow-sm">
          <div class="text-xs text-text-muted">最早到期</div>
          <div class="mt-1 text-lg font-semibold text-text-primary">{{ formatDate(view?.expires_at) }}</div>
        </div>
        <div class="rounded-lg bg-bg-card p-4 shadow-sm">
          <div class="text-xs text-text-muted">状态</div>
          <div class="mt-1">
            <el-tag :type="stateType(view?.state)">{{ stateLabel(view?.state) }}</el-tag>
          </div>
        </div>
      </div>

      <el-table :data="view?.modules || []" border stripe>
        <el-table-column prop="module_name" label="功能模块" min-width="160" />
        <el-table-column label="来源" width="100">
          <template #default="{ row }">{{ sourceLabel(row.source) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="stateType(row.state)">{{ stateLabel(row.state) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="到期时间" width="140">
          <template #default="{ row }">{{ formatDate(row.expires_at) }}</template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { getSubscription, type SubscriptionView } from '~/api/platform'
import { expiringSoon } from '~/utils/entitlement'

const loading = ref(false)
const view = ref<SubscriptionView | null>(null)

// 只在整体在期时提示；宽限期与已降级由下方既有的到期提示承担，避免叠加。
const expiringNotice = computed(() =>
  view.value?.state === 'active' ? expiringSoon(view.value.expires_at) : null,
)

async function load() {
  loading.value = true
  try {
    view.value = await getSubscription()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function formatDate(value?: string | null) {
  if (!value) return '—'
  return String(value).slice(0, 10)
}
function stateLabel(state?: string) {
  if (state === 'active') return '在期'
  if (state === 'grace') return '宽限期'
  if (state === 'expired') return '已降级'
  return '—'
}
function stateType(state?: string) {
  if (state === 'active') return 'success'
  if (state === 'grace') return 'warning'
  if (state === 'expired') return 'danger'
  return 'info'
}
function sourceLabel(source: string) {
  return ({ trial: '试用', subscription: '订阅', gift: '赠送' } as Record<string, string>)[source] || source
}

onMounted(load)
</script>
