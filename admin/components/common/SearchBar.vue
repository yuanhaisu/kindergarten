<template>
  <div class="flex flex-wrap items-center gap-2 mb-4">
    <el-input
      v-if="showKeyword"
      v-model="keyword"
      clearable
      :placeholder="keywordPlaceholder"
      class="!w-52"
      @keyup.enter="emitSearch"
    />
    <slot />
    <el-button type="primary" @click="emitSearch">查询</el-button>
    <el-button v-if="showReset" @click="onReset">重置</el-button>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    showKeyword?: boolean
    keywordPlaceholder?: string
    showReset?: boolean
  }>(),
  {
    showKeyword: true,
    keywordPlaceholder: '关键词',
    showReset: true,
  },
)

const emit = defineEmits<{
  search: [payload: { keyword: string }]
  reset: []
}>()

const keyword = defineModel<string>('keyword', { default: '' })

function emitSearch() {
  emit('search', { keyword: keyword.value || '' })
}

function onReset() {
  keyword.value = ''
  emit('reset')
  emit('search', { keyword: '' })
}
</script>
