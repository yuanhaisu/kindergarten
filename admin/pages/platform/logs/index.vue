<template>
  <div class="space-y-4">
    <div>
      <h1 class="text-xl font-semibold text-text-primary">审计</h1>
      <p class="mt-1 text-xs text-text-muted">
        平台操作留痕：只记录成功操作。可按时间范围、操作人、模块与目标园所检索。
      </p>
    </div>

    <SearchBar :show-keyword="false" @search="search" @reset="resetFilters">
      <el-date-picker
        v-model="range"
        type="daterange"
        value-format="YYYY-MM-DD"
        unlink-panels
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        range-separator="至"
        class="!w-80"
        @change="search"
      />
      <el-input
        v-model="operator"
        placeholder="操作人（精确匹配）"
        clearable
        class="!w-44"
        @keyup.enter="search"
        @clear="search"
      />
      <el-select v-model="moduleFilter" clearable placeholder="全部模块" class="!w-36" @change="search">
        <el-option v-for="item in PLATFORM_LOG_MODULES" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
      <el-select
        v-model="kindergartenID"
        filterable
        clearable
        placeholder="全部园所"
        class="!w-48"
        @change="search"
      >
        <el-option v-for="kg in kindergartens" :key="kg.id" :label="kg.name" :value="kg.id" />
      </el-select>
    </SearchBar>

    <el-alert v-if="errorMessage" type="error" show-icon :closable="false" :title="`加载审计记录失败：${errorMessage}`">
      <template #default>
        <el-button size="small" @click="load">重试</el-button>
      </template>
    </el-alert>

    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="rows"
      :loading="loading"
      :total="total"
      :empty-text="emptyText"
      @page-change="load"
      @size-change="onSizeChange"
    >
      <el-table-column label="时间" width="170">
        <template #default="{ row }">{{ formatLogTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column prop="operator" label="操作人" width="120" />
      <el-table-column label="模块" width="90">
        <template #default="{ row }">{{ moduleLabel(row.module) }}</template>
      </el-table-column>
      <el-table-column label="动作" width="110">
        <template #default="{ row }">
          <span :title="row.action">{{ actionLabel(row.action) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="目标园所" min-width="140">
        <template #default="{ row }">{{ kindergartenName(row.kindergarten_id) }}</template>
      </el-table-column>
      <el-table-column label="摘要" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ row.detail || '—' }}</template>
      </el-table-column>
      <el-table-column label="请求链路" min-width="230" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="font-mono text-xs text-text-muted">{{ row.request_id || '—' }}</span>
        </template>
      </el-table-column>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { listPlatformLogs, type PlatformLogRow } from '~/api/platform/logs'
import { listKindergartens, type PlatformKindergartenRow } from '~/api/platform/kindergartens'
import {
  PLATFORM_LOG_MODULES,
  actionLabel,
  buildPlatformLogQuery,
  formatLogTime,
  moduleLabel,
} from '~/utils/platformLogs'

definePageMeta({ layout: 'platform' })

const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rows = ref<PlatformLogRow[]>([])
const loading = ref(false)
const errorMessage = ref('')

const range = ref<[string, string] | null>(null)
const operator = ref('')
const moduleFilter = ref('')
const kindergartenID = ref<number | null>(null)

const kindergartens = ref<PlatformKindergartenRow[]>([])
const kindergartenNames = computed(() => new Map(kindergartens.value.map((kg) => [kg.id, kg.name])))

const hasFilters = computed(
  () => Boolean(range.value || operator.value.trim() || moduleFilter.value || kindergartenID.value),
)
const emptyText = computed(() => {
  if (errorMessage.value) return '加载失败，请点击上方「重试」'
  return hasFilters.value ? '没有符合筛选条件的审计记录' : '暂无审计记录'
})

async function load() {
  loading.value = true
  try {
    const data = await listPlatformLogs(
      buildPlatformLogQuery({
        page: page.value,
        pageSize: pageSize.value,
        operator: operator.value,
        module: moduleFilter.value,
        kindergartenID: kindergartenID.value,
        range: range.value,
      }),
    )
    rows.value = data.list || []
    total.value = data.total || 0
    errorMessage.value = ''
  } catch (e) {
    // 请求失败时清空旧数据：保留上一次结果会与当前筛选条件对不上，容易被误读。
    rows.value = []
    total.value = 0
    errorMessage.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

/** 筛选条件变化一律回到第 1 页，避免落在不存在的页码上看到空列表。 */
function search() {
  page.value = 1
  void load()
}

function resetFilters() {
  range.value = null
  operator.value = ''
  moduleFilter.value = ''
  kindergartenID.value = null
  // SearchBar 重置后还会发一次 search，由它负责回到第 1 页并重新加载。
}

function onSizeChange() {
  page.value = 1
  void load()
}

async function loadKindergartens() {
  try {
    const data = await listKindergartens({ page: 1, page_size: 100 })
    kindergartens.value = data.list || []
  } catch (e) {
    // 选项失败不阻断列表，但必须让用户知道园所筛选当前不可用。
    console.error('[platform-logs] 园所选项加载失败', e)
    ElMessage.warning('园所选项加载失败，园所筛选暂不可用')
  }
}

function kindergartenName(id: number) {
  if (!id) return '—'
  return kindergartenNames.value.get(id) || `#${id}`
}

onMounted(() => {
  void load()
  void loadKindergartens()
})
</script>
