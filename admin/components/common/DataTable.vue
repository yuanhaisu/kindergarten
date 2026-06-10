<template>
  <div>
    <el-table
      v-loading="loading"
      :data="data"
      stripe
      border
      class="w-full"
      :empty-text="emptyText"
      @selection-change="(s: unknown[]) => $emit('selection-change', s)"
    >
      <slot />
    </el-table>
    <div v-if="showPagination" class="flex justify-end mt-4">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="size"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next"
        background
        @current-change="$emit('page-change')"
        @size-change="$emit('size-change')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    data: Record<string, unknown>[]
    loading?: boolean
    total?: number
    emptyText?: string
    showPagination?: boolean
  }>(),
  {
    loading: false,
    total: 0,
    emptyText: '暂无数据',
    showPagination: true,
  },
)

defineEmits<{
  'selection-change': [rows: unknown[]]
  'page-change': []
  'size-change': []
}>()

const page = defineModel<number>('page', { default: 1 })
const size = defineModel<number>('pageSize', { default: 20 })
</script>
