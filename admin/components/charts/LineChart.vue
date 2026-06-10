<template>
  <ClientOnly>
    <VChart class="h-[280px] w-full" :option="chartOption" autoresize />
  </ClientOnly>
</template>

<script setup lang="ts">
defineProps<{
  chartOption: Record<string, unknown>
}>()
</script>
