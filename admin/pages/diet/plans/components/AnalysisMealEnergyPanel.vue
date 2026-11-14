<template>
  <section data-testid="analysis-meal-energy">
    <h3 class="mb-1 text-center text-base font-semibold text-text-primary">三餐供能比</h3>
    <p class="mb-2 text-center text-xs text-text-muted">
      各餐能量只除有食谱天数，不再除折算系数；比例为各餐能量 ÷ 全日 DRIs。
    </p>
    <div class="overflow-x-auto">
      <el-table :data="rows" border size="small" class="w-full">
        <el-table-column prop="label" label="类别" width="120" />
        <el-table-column
          v-for="meal in meals"
          :key="meal.meal_key"
          :label="meal.label"
          align="center"
          min-width="90"
        >
          <template #default="{ row }">
            {{
              row.key === 'energy'
                ? formatNumber(meal.energy)
                : formatPercent(meal.percent)
            }}
          </template>
        </el-table-column>
      </el-table>
    </div>
    <p class="mt-1 text-xs text-text-muted">
      合计：{{ formatPercent(mealEnergy.total_percent) }}（各餐比例之和，不归一化为 100%）
    </p>
  </section>
</template>

<script setup lang="ts">
import type { DietAnalysisMealEnergy } from '~/types/api'
import { formatNumber, formatPercent } from './analysis-format'

const props = defineProps<{
  mealEnergy: DietAnalysisMealEnergy
}>()

const rows = [
  { key: 'energy', label: '能量（千卡）' },
  { key: 'percent', label: '比例（%）' },
]

const meals = computed(() => props.mealEnergy.meals ?? [])
</script>
