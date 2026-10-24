<template>
  <div>
    <el-table
      v-loading="loading"
      :data="data"
      :row-key="rowKey"
      :tree-props="treeProps"
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
    /** 传 row-key 时启用 el-table 的树形数据（如任务列表按活动折叠） */
    rowKey?: string
    treeProps?: { children?: string; hasChildren?: string }
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
