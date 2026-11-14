<template>
  <el-dialog v-model="visible" title="同类替换" width="580px" destroy-on-close @open="reset">
    <div class="space-y-3">
      <div class="text-text-muted text-sm">
        正在替换「{{ foodName }}」。选择候选食物后保留原克数，只更换食物并按新食物的营养重算小计。
      </div>

      <div class="flex gap-2">
        <div class="flex-1" data-testid="replace-search">
          <el-input
            v-model="keyword"
            clearable
            placeholder="按名称搜索候选食物"
            @keyup.enter="search"
          />
        </div>
        <el-button type="primary" :loading="loading" @click="search">查询</el-button>
      </div>

      <el-alert v-if="error" type="error" :closable="false" show-icon :title="error" />

      <div v-loading="loading" class="max-h-80 overflow-auto">
        <el-empty
          v-if="!loading && candidates.length === 0"
          description="没有同类候选食物，可换个关键词"
          :image-size="72"
        />
        <div
          v-for="candidate in candidates"
          :key="candidate.id"
          class="flex items-center justify-between gap-2 border-b border-border-hairline py-2"
        >
          <div class="min-w-0">
            <div class="truncate text-sm text-text-primary">{{ candidate.name }}</div>
            <div class="text-xs text-text-muted">{{ candidate.category_name }}</div>
          </div>
          <el-button
            size="small"
            type="primary"
            plain
            :loading="pickingId === candidate.id"
            @click="pick(candidate)"
          >
            选择
          </el-button>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import type { DietFoodDetail, DietReplaceCandidate } from '~/types/api'
import { getDietFood, getDietFoodReplaceCandidates } from '~/api/diet'

const props = defineProps<{
  foodId: number | null
  foodName: string
}>()

const emit = defineEmits<{
  picked: [detail: DietFoodDetail]
}>()

const visible = defineModel<boolean>({ required: true })

const keyword = ref('')
const candidates = ref<DietReplaceCandidate[]>([])
const loading = ref(false)
const pickingId = ref<number | null>(null)
const error = ref('')

function reset() {
  keyword.value = ''
  candidates.value = []
  error.value = ''
  pickingId.value = null
  void search()
}

async function search() {
  if (props.foodId == null) {
    error.value = '未指定要替换的食物'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await getDietFoodReplaceCandidates(props.foodId, {
      keyword: keyword.value.trim() || undefined,
      limit: 20,
    })
    candidates.value = res.list ?? []
  } catch (e: unknown) {
    candidates.value = []
    error.value = e instanceof Error ? e.message : '候选食物加载失败'
  } finally {
    loading.value = false
  }
}

async function pick(candidate: DietReplaceCandidate) {
  pickingId.value = candidate.id
  error.value = ''
  try {
    const detail = await getDietFood(candidate.id)
    emit('picked', detail)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : '食物详情加载失败'
  } finally {
    pickingId.value = null
  }
}
</script>
