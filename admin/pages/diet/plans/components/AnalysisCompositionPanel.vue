<template>
  <section data-testid="analysis-composition">
    <h3 class="mb-1 text-center text-base font-semibold text-text-primary">膳食构成</h3>
    <p class="mb-2 text-center text-xs text-text-muted">
      接口返回 {{ allGroups.length }} 类，本页只渲染后端标记 displayed=true 的
      {{ groups.length }} 类；其余类别不参与报表展示。
    </p>
    <div class="overflow-x-auto">
      <el-table :data="rows" border size="small" class="w-full">
        <el-table-column prop="label" label="类别" width="120" />
        <el-table-column
          v-for="group in groups"
          :key="group.key"
          :label="group.label"
          align="center"
          min-width="86"
        >
          <template #default>
            {{ formatNumber(group.grams) }}
          </template>
        </el-table-column>
      </el-table>
    </div>
    <p class="mt-1 text-xs text-text-muted">摄入量单位为克，按平均每人每日计算。</p>
  </section>
</template>

<script setup lang="ts">
import type { DietAnalysisComposition } from '~/types/api'
import { formatNumber } from './analysis-format'

const props = defineProps<{
  composition: DietAnalysisComposition
}>()

const rows = [{ label: '摄入量（克）' }]

const allGroups = computed(() => props.composition.groups ?? [])
const groups = computed(() => allGroups.value.filter((group) => group.displayed))
</script>
