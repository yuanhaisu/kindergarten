<template>
  <section data-testid="analysis-calcium-iron">
    <h3 class="mb-1 text-center text-base font-semibold text-text-primary">钙铁来源</h3>
    <p class="mb-2 text-center text-xs text-text-muted">
      钙的动物性、植物性、乳类三项不互斥（乳类同时计入动物性），构成比合计不等于
      100%；铁的两项互斥且穷尽。
    </p>
    <div class="overflow-x-auto">
      <el-table :data="rows" border size="small" class="w-full">
        <el-table-column prop="label" label="来源" width="130" />
        <el-table-column label="钙" align="center">
          <el-table-column
            v-for="source in calciumSources"
            :key="`calcium-${source.key}`"
            :label="source.label"
            align="center"
            min-width="100"
          >
            <template #default="{ row }">
              {{ cell('calcium', source.key, row.key) }}
            </template>
          </el-table-column>
        </el-table-column>
        <el-table-column label="铁" align="center">
          <el-table-column
            v-for="source in ironSources"
            :key="`iron-${source.key}`"
            :label="source.label"
            align="center"
            min-width="100"
          >
            <template #default="{ row }">
              {{ cell('iron', source.key, row.key) }}
            </template>
          </el-table-column>
        </el-table-column>
      </el-table>
    </div>
    <p class="mt-1 text-xs text-text-muted">
      钙合计 {{ formatNumber(calciumIron.calcium.total) }} 毫克；铁合计
      {{ formatNumber(calciumIron.iron.total) }} 毫克。
    </p>
  </section>
</template>

<script setup lang="ts">
import type { DietAnalysisCalciumIron } from '~/types/api'
import { formatNumber, formatPercent } from './analysis-format'

const props = defineProps<{
  calciumIron: DietAnalysisCalciumIron
}>()

const rows = [
  { key: 'value', label: '重量（毫克）' },
  { key: 'percent', label: '构成比%' },
]

const calciumSources = computed(() => props.calciumIron.calcium.sources ?? [])
const ironSources = computed(() => props.calciumIron.iron.sources ?? [])

function cell(side: 'calcium' | 'iron', key: string, rowKey: string): string {
  const sources = side === 'calcium' ? calciumSources.value : ironSources.value
  const source = sources.find((item) => item.key === key)
  if (!source) return '—'
  return rowKey === 'value' ? formatNumber(source.value) : formatPercent(source.percent)
}
</script>
