<template>
  <section data-testid="analysis-conclusion">
    <h3 class="mb-2 text-center text-base font-semibold text-text-primary">定性分析结论</h3>
    <div class="space-y-1 text-sm leading-6 text-text-body">
      <p>
        1、本周期内食品种类：{{ conclusion.food_kind_count }}种，其中调味品种：{{
          conclusion.condiment_kind_count
        }}种，食物品种：{{ conclusion.non_condiment_kind_count }}种。
      </p>

      <p>2、本周期DRIs占比分析：</p>
      <p class="pl-4">
        1）热量占比：{{ formatPercent(dri.energy) }}，蛋白质占比：{{
          formatPercent(dri.protein)
        }}，脂肪占比：{{ formatPercent(dri.fats) }}，碳水化合物占比：{{
          formatPercent(dri.carb)
        }}
      </p>
      <p class="pl-4">
        2）维生素占比分析：维生素A：{{ formatPercent(dri.vitamin_a) }}，维生素B1：{{
          formatPercent(dri.vitamin_b1)
        }}，维生素B2：{{ formatPercent(dri.vitamin_b2) }}，维生素C：{{
          formatPercent(dri.vitamin_c)
        }}
      </p>
      <p class="pl-4">
        3）矿物质占比分析：钙：{{ formatPercent(dri.calcium) }}，铁：{{
          formatPercent(dri.iron)
        }}，锌：{{ formatPercent(dri.zinc) }}
      </p>

      <p>3、三大营养素占比分析：</p>
      <p class="pl-4">
        1）蛋白质：{{ formatPercent(macro.protein) }}，脂肪：{{ formatPercent(macro.fats) }}，
        碳水化合物：{{ formatPercent(macro.carb) }}
      </p>
      <p class="pl-4">
        2）【蛋白质 ：脂肪 ： 碳水化合物】的比值：【{{ conclusion.macro_ratio }}】
      </p>

      <p>
        4、【动物蛋白质 + 豆类蛋白质】占总蛋白质百分比：{{
          formatPercent(conclusion.animal_bean_protein_percent)
        }}
      </p>

      <template v-if="suggestions.length > 0">
        <p>5、营养摄入量分析：</p>
        <p
          v-for="(suggestion, index) in suggestions"
          :key="`${suggestion.nutrient_key}-${suggestion.level}`"
          class="pl-4"
        >
          {{ index + 1 }}）{{ suggestion.text }}
        </p>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { DietConclusion } from '~/types/api'
import { formatPercent } from './analysis-format'

const props = defineProps<{
  conclusion: DietConclusion
}>()

const dri = computed(() => props.conclusion.dri_percent ?? ({} as Record<string, number>))
const macro = computed(() => props.conclusion.macro_percent)
const suggestions = computed(() => props.conclusion.suggestions ?? [])
</script>
