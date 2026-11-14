<template>
  <div class="space-y-4">
    <section data-testid="analysis-nutrition-table">
      <h3 class="mb-1 text-center text-base font-semibold text-text-primary">营养分析表</h3>
      <p class="mb-2 text-center text-xs text-text-muted">
        共 {{ columns.length }} 列：视黄醇当量列复用维生素A的值，胡萝卜素列按参考产品固定显示「-」；
        DRIs 行为全日原值，未按折算系数缩放。
      </p>
      <div class="overflow-x-auto">
        <el-table :data="metricRows" border size="small" class="w-full">
          <el-table-column prop="label" label="项目" width="110" />
          <el-table-column
            v-for="column in columns"
            :key="column.key"
            align="center"
            min-width="64"
          >
            <template #header>
              <div class="leading-tight">
                <div>{{ column.label }}</div>
                <div class="text-xs text-text-muted">（{{ column.unit }}）</div>
              </div>
            </template>
            <template #default="{ row }">
              <el-tag
                v-if="row.key === 'comment' && !column.placeholder"
                size="small"
                :type="commentTagType(cell(column, row.key))"
              >
                {{ cell(column, row.key) }}
              </el-tag>
              <span v-else>{{ cell(column, row.key) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </section>

    <section data-testid="analysis-macro-energy">
      <h3 class="mb-1 text-sm font-semibold text-text-primary">三大营养素供能</h3>
      <el-table :data="macroRows" border size="small" class="w-full">
        <el-table-column prop="label" label="热量来源" min-width="120" />
        <el-table-column label="脂肪" align="center" min-width="120">
          <template #default="{ row }">
            <div>{{ formatNumber((row as DietMacroEnergyRow).fats) }}</div>
            <div v-if="(row as DietMacroEnergyRow).recommended" class="text-xs text-text-muted">
              要求 {{ (row as DietMacroEnergyRow).recommended?.fats }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="蛋白质" align="center" min-width="120">
          <template #default="{ row }">
            <div>{{ formatNumber((row as DietMacroEnergyRow).protein) }}</div>
            <div v-if="(row as DietMacroEnergyRow).recommended" class="text-xs text-text-muted">
              要求 {{ (row as DietMacroEnergyRow).recommended?.protein }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="碳水化合物" align="center" min-width="120">
          <template #default="{ row }">
            <div>{{ formatNumber((row as DietMacroEnergyRow).carb) }}</div>
            <div v-if="(row as DietMacroEnergyRow).recommended" class="text-xs text-text-muted">
              要求 {{ (row as DietMacroEnergyRow).recommended?.carb }}
            </div>
          </template>
        </el-table-column>
      </el-table>
      <p class="mt-1 text-xs text-text-muted">
        占总热量% 的分母 = 蛋白质×4 + 脂肪×9 + 碳水化合物×4 =
        {{ formatNumber(macroEnergy.denominator) }} 千卡
      </p>
    </section>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
      <section data-testid="analysis-protein-source">
        <h3 class="mb-1 text-sm font-semibold text-text-primary">蛋白质来源</h3>
        <el-table :data="sourceRows" border size="small" class="w-full">
          <el-table-column prop="label" label="项目" min-width="110" />
          <el-table-column
            v-for="column in proteinColumns"
            :key="column.key"
            :label="column.label"
            align="center"
            min-width="100"
          >
            <template #default="{ row }">
              {{
                row.key === 'value'
                  ? formatNumber(proteinItem(column.key)?.value)
                  : formatPercent(proteinItem(column.key)?.percent)
              }}
            </template>
          </el-table-column>
        </el-table>
        <p class="mt-1 text-xs text-text-muted">
          动物蛋白质 + 豆类蛋白质占总蛋白质：
          {{ formatPercent(proteinSource.animal_bean_percent) }}；三项合计不等于总蛋白。
        </p>
      </section>

      <section data-testid="analysis-fat-source">
        <h3 class="mb-1 text-sm font-semibold text-text-primary">脂肪来源</h3>
        <el-table :data="sourceRows" border size="small" class="w-full">
          <el-table-column prop="label" label="项目" min-width="110" />
          <el-table-column
            v-for="item in displayedFatItems"
            :key="item.key"
            :label="item.label"
            align="center"
            min-width="100"
          >
            <template #default="{ row }">
              {{
                row.key === 'value'
                  ? formatNumber(item.value)
                  : formatPercent(item.percent)
              }}
            </template>
          </el-table-column>
        </el-table>
        <p class="mt-1 text-xs text-text-muted">
          总脂肪 {{ formatNumber(fatSource.total) }} 克；「其他」为残差，仅用于分析图，本页不展示。
        </p>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import type {
  DietAnalysisFatSource,
  DietAnalysisMacroEnergy,
  DietAnalysisNutrition,
  DietAnalysisProteinSource,
  DietMacroEnergyRow,
  DietNutritionRow,
} from '~/types/api'
import { formatNumber, formatPercent } from './analysis-format'

const props = defineProps<{
  nutrition: DietAnalysisNutrition
  macroEnergy: DietAnalysisMacroEnergy
  proteinSource: DietAnalysisProteinSource
  fatSource: DietAnalysisFatSource
}>()

const metricRows = [
  { key: 'intake', label: '平均每人每日' },
  { key: 'dri', label: 'DRIs' },
  { key: 'compare', label: '比较%' },
  { key: 'comment', label: '评价' },
]

const sourceRows = [
  { key: 'value', label: '摄入量（克）' },
  { key: 'percent', label: '构成比%' },
]

interface NutritionColumn {
  key: string
  label: string
  unit: string
  /** 复用另一项数值（视黄醇当量列 = 维生素A） */
  sourceKey?: string
  /** 固定占位（胡萝卜素列 = "-"） */
  placeholder?: string
}

const itemByKey = computed(() => {
  const map = new Map<string, DietNutritionRow>()
  for (const item of props.nutrition.items ?? []) {
    map.set(item.key, item)
  }
  return map
})

const columns = computed<NutritionColumn[]>(() => {
  const real = (key: string, label: string, unit: string): NutritionColumn => {
    const item = itemByKey.value.get(key)
    return { key, label: item?.label || label, unit: item?.unit || unit }
  }
  return [
    real('energy', '热量', '千卡'),
    real('heat', '热量', '千焦'),
    real('protein', '蛋白质', '克'),
    real('fats', '脂肪', '克'),
    real('carb', '碳水化合物', '克'),
    {
      key: 'retinol_equivalent',
      label: '视黄醇当量',
      unit: itemByKey.value.get('vitamin_a')?.unit || '微克',
      sourceKey: 'vitamin_a',
    },
    real('vitamin_a', '维生素A', '微克'),
    { key: 'carotin', label: '胡萝卜素', unit: '微克', placeholder: '-' },
    real('vitamin_b1', '维生素B1', '毫克'),
    real('vitamin_b2', '维生素B2', '毫克'),
    real('vitamin_c', '维生素C', '毫克'),
    real('calcium', '钙', '毫克'),
    real('zinc', '锌', '毫克'),
    real('iron', '铁', '毫克'),
  ]
})

function cell(column: NutritionColumn, rowKey: string): string {
  if (column.placeholder) return column.placeholder
  const item = itemByKey.value.get(column.sourceKey || column.key)
  if (!item) return '—'
  switch (rowKey) {
    case 'intake':
      return formatNumber(item.intake)
    case 'dri':
      return formatNumber(item.dri)
    case 'compare':
      return formatPercent(item.compare_percent)
    case 'comment':
      return item.comment || '—'
    default:
      return '—'
  }
}

type TagType = 'success' | 'warning' | 'danger' | 'info'

function commentTagType(comment: string): TagType {
  if (comment === '低') return 'danger'
  if (comment === '高') return 'warning'
  if (comment === '合格') return 'success'
  return 'info'
}

const macroRows = computed<DietMacroEnergyRow[]>(() => props.macroEnergy.rows ?? [])

const proteinColumns = [
  { key: 'required', label: '要求' },
  { key: 'animal', label: '动物性食物' },
  { key: 'bean', label: '豆类' },
]

function proteinItem(key: string) {
  return (props.proteinSource.items ?? []).find((item) => item.key === key)
}

const displayedFatItems = computed(() =>
  (props.fatSource.items ?? []).filter((item) => item.displayed),
)
</script>
