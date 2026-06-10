<template>
  <el-tag :type="tagType" size="small" effect="light">{{ label }}</el-tag>
</template>

<script setup lang="ts">
const props = defineProps<{
  label: string
  type?: 'success' | 'warning' | 'info' | 'danger' | ''
}>()

const tagType = computed(() => props.type || 'info')
</script>
