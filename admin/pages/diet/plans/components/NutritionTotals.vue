<template>
  <div :data-testid="testId" class="grid grid-cols-2 gap-x-3 gap-y-1 text-xs leading-5">
    <span v-for="nutrient in CORE_NUTRIENTS" :key="nutrient.key" class="whitespace-nowrap">
      <span class="text-text-muted">{{ nutrient.label }}</span>
      <span class="text-text-primary ml-1 font-medium">{{ formatCore(totals[nutrient.key]) }}</span>
      <span class="text-text-muted ml-0.5">{{ nutrient.unit }}</span>
    </span>
  </div>
</template>

<script setup lang="ts">
import { CORE_NUTRIENTS, type CoreTotals } from './plan-types'
import { formatCore } from './plan-utils'

withDefaults(defineProps<{ totals: CoreTotals; testId?: string }>(), {
  testId: '',
})
</script>
