<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap justify-between gap-2">
        <span>菜谱库</span>
        <div v-if="canManage" class="flex gap-2">
          <el-button @click="openMerge">合并重复菜谱</el-button>
        </div>
      </div>
    </template>

    <SearchBar
      v-model:keyword="keyword"
      keyword-placeholder="菜谱名称或拼音码"
      @search="onSearch"
      @reset="onSearch"
    />

    <el-alert
      v-if="listError"
      type="error"
      :closable="false"
      show-icon
      title="菜谱列表加载失败"
      :description="listError"
      class="mb-4"
    >
      <el-button size="small" @click="load">重试</el-button>
    </el-alert>

    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="list as unknown as Record<string, unknown>[]"
      :loading="loading"
      :total="total"
      empty-text="暂无菜谱"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column prop="name" label="菜谱名称" min-width="220" />
      <el-table-column prop="spell_code" label="拼音码" width="160" />
      <el-table-column prop="item_count" label="配料数" width="100" />
      <el-table-column label="操作" width="170" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail((row as DietRecipeListItem).id)">详情</el-button>
          <el-button v-if="canManage" link @click="openEditor((row as DietRecipeListItem).id)">编辑</el-button>
          <el-button v-if="canManage" link type="danger" @click="remove(row as DietRecipeListItem)">删除</el-button>
        </template>
      </el-table-column>
    </DataTable>

    <!-- 详情抽屉：配料 + 当前营养 -->
    <el-drawer v-model="detailVisible" title="菜谱详情" size="720px" destroy-on-close>
      <div v-loading="detailLoading" class="min-h-24">
        <el-alert v-if="detailError" type="error" :closable="false" show-icon :title="detailError" />
        <template v-else-if="detail">
          <div class="text-lg font-semibold text-text-primary">{{ detail.name }}</div>
          <div class="text-text-muted text-sm mt-1">
            拼音码 {{ detail.spell_code || '—' }} · 配料 {{ detail.items.length }} 种
          </div>

          <el-table :data="detail.items" border class="mt-4 w-full" empty-text="暂无配料">
            <el-table-column type="expand">
              <template #default="{ row }">
                <div class="grid grid-cols-3 gap-x-4 gap-y-1 p-2">
                  <div
                    v-for="n in ALL_NUTRIENTS"
                    :key="n.key"
                    class="flex justify-between border-b border-border-hairline py-1"
                  >
                    <span class="text-text-muted text-xs">
                      {{ n.label }}<span v-if="n.unit">（{{ n.unit }}）</span>
                    </span>
                    <span class="text-text-primary text-xs">{{ formatNutrient(row.nutrients[n.key]) }}</span>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="food_name" label="食物名称" min-width="150" />
            <el-table-column label="一人均量(克)" width="110">
              <template #default="{ row }">{{ row.grams }}</template>
            </el-table-column>
            <el-table-column label="当前营养（每 100g 可食部）" min-width="260">
              <template #default="{ row }">
                能量 {{ formatNutrient(row.nutrients.energy) }} 千卡 · 蛋白质
                {{ formatNutrient(row.nutrients.protein) }} 克 · 脂肪
                {{ formatNutrient(row.nutrients.fats) }} 克 · 碳水
                {{ formatNutrient(row.nutrients.carb) }} 克
              </template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <!-- 改名 / 改配方 -->
    <el-dialog
      v-model="editorVisible"
      title="编辑菜谱"
      width="760px"
      destroy-on-close
      @closed="resetEditor"
    >
      <div v-loading="editorLoading" class="min-h-24">
        <el-alert
          v-if="editorError"
          type="error"
          :closable="false"
          show-icon
          title="操作失败"
          :description="editorError"
          class="mb-4"
        />
        <el-form v-if="editorReady" :model="editorForm" label-width="90px">
          <el-form-item label="菜谱名称" required>
            <el-input v-model="editorForm.name" maxlength="40" placeholder="请输入菜谱名称" />
          </el-form-item>
          <el-form-item label="配料" required>
            <div class="w-full space-y-2">
              <div v-for="(item, idx) in editorForm.items" :key="idx" class="flex items-center gap-2">
                <el-select
                  v-model="item.food_id"
                  filterable
                  remote
                  clearable
                  :remote-method="searchFoods"
                  :loading="foodSearching"
                  placeholder="搜索食物"
                  class="flex-1"
                  @change="() => onFoodPicked(item)"
                >
                  <el-option
                    v-for="opt in foodOptionsForForm"
                    :key="opt.id"
                    :label="opt.name"
                    :value="opt.id"
                  />
                </el-select>
                <el-input-number
                  v-model="item.grams"
                  :min="0"
                  :controls="false"
                  placeholder="克数"
                  class="!w-32"
                />
                <el-button link type="danger" @click="removeItem(idx)">移除</el-button>
              </div>
              <el-button size="small" @click="addItem">添加食物</el-button>
            </div>
          </el-form-item>
          <div class="text-text-muted text-xs">
            保存后配方整体替换；同一食物多行会合并克数。改名撞上其他菜谱名称会被拒绝（409）。
          </div>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="editorSaving" :disabled="!editorReady" @click="submitEditor">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 合并重复 -->
    <el-dialog
      v-model="mergeVisible"
      title="合并重复菜谱"
      width="620px"
      destroy-on-close
      @closed="resetMerge"
    >
      <el-alert
        v-if="mergeError"
        type="error"
        :closable="false"
        show-icon
        title="合并失败"
        :description="mergeError"
        class="mb-4"
      />
      <el-form :model="mergeForm" label-width="100px">
        <el-form-item label="源菜谱" required>
          <el-select
            v-model="mergeForm.source_id"
            filterable
            remote
            clearable
            :remote-method="searchMergeSource"
            :loading="mergeLoadingSource"
            placeholder="搜索要合并掉的菜谱"
            class="w-full"
          >
            <el-option
              v-for="r in mergeSourceOptions"
              :key="r.id"
              :label="`${r.name}（${r.item_count} 项配料）`"
              :value="r.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="目标菜谱" required>
          <el-select
            v-model="mergeForm.target_id"
            filterable
            remote
            clearable
            :remote-method="searchMergeTarget"
            :loading="mergeLoadingTarget"
            placeholder="搜索保留的菜谱"
            class="w-full"
          >
            <el-option
              v-for="r in mergeTargetOptions"
              :key="r.id"
              :label="`${r.name}（${r.item_count} 项配料）`"
              :value="r.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <div class="text-text-muted text-xs leading-5">
        合并会把源菜谱的同种食物克数累加到目标菜谱，然后删除源菜谱；历史周食谱保持原菜名与营养快照，不受影响。
      </div>
      <template #footer>
        <el-button @click="mergeVisible = false">取消</el-button>
        <el-button type="primary" :loading="mergeSaving" @click="submitMerge">确认合并</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type {
  DietApiError,
  DietRecipeDetail,
  DietRecipeListItem,
  DietRecipeSaveItem,
} from '~/types/api'
import {
  deleteDietRecipe,
  getDietFoods,
  getDietRecipe,
  getDietRecipes,
  mergeDietRecipes,
  updateDietRecipe,
} from '~/api/diet'
import { ALL_NUTRIENTS, formatNutrient } from '../components/nutrients'

const { can } = usePermission()
const canManage = computed(() => can('diet:manage'))

const { page, pageSize, total, resetPage } = usePagination()
const keyword = ref('')
const list = ref<DietRecipeListItem[]>([])
const loading = ref(false)
const listError = ref('')

onMounted(load)

function onSearch() {
  resetPage()
  load()
}

async function load() {
  loading.value = true
  listError.value = ''
  try {
    const res = await getDietRecipes({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    list.value = res.list ?? []
    total.value = res.total ?? 0
  } catch (e: unknown) {
    list.value = []
    total.value = 0
    listError.value = e instanceof Error ? e.message : '网络错误'
  } finally {
    loading.value = false
  }
}

// ===== 详情 =====

const detailVisible = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const detail = ref<DietRecipeDetail | null>(null)

async function openDetail(id: number) {
  detail.value = null
  detailError.value = ''
  detailVisible.value = true
  await loadDetail(id)
}

async function loadDetail(id: number) {
  detailLoading.value = true
  detailError.value = ''
  try {
    detail.value = await getDietRecipe(id)
  } catch (e: unknown) {
    detailError.value = e instanceof Error ? e.message : '菜谱详情加载失败'
  } finally {
    detailLoading.value = false
  }
}

// ===== 编辑 =====

interface RecipeItemDraft {
  /** el-select 的 v-model 类型较宽，运行时为食物 id，本地按 unknown 收窄 */
  food_id?: unknown
  food_name: string
  grams: number | null | undefined
}

interface FoodOption {
  id: number
  name: string
}

const editorVisible = ref(false)
const editorLoading = ref(false)
const editorSaving = ref(false)
const editorReady = ref(false)
const editorError = ref('')
const editingId = ref<number | null>(null)

const editorForm = reactive<{ name: string; items: RecipeItemDraft[] }>({
  name: '',
  items: [],
})

const foodOptions = ref<FoodOption[]>([])
const foodSearching = ref(false)

const foodOptionsForForm = computed(() => {
  const map = new Map<number, FoodOption>()
  for (const opt of foodOptions.value) map.set(opt.id, opt)
  // 已选食物的名称始终并入选项，避免远程搜索结果变化后选中项退回显示 id
  for (const item of editorForm.items) {
    if (typeof item.food_id === 'number' && item.food_name) {
      map.set(item.food_id, { id: item.food_id, name: item.food_name })
    }
  }
  return Array.from(map.values())
})

function resetEditor() {
  editingId.value = null
  editorError.value = ''
  editorReady.value = false
  editorLoading.value = false
  editorSaving.value = false
  editorForm.name = ''
  editorForm.items = []
  foodOptions.value = []
}

async function openEditor(id: number) {
  resetEditor()
  editingId.value = id
  editorVisible.value = true
  editorLoading.value = true
  try {
    const res = await getDietRecipe(id)
    editorForm.name = res.name
    editorForm.items = (res.items ?? []).map((item) => ({
      food_id: item.food_id,
      food_name: item.food_name,
      grams: item.grams,
    }))
    const picked: FoodOption[] = []
    for (const item of editorForm.items) {
      if (typeof item.food_id === 'number') picked.push({ id: item.food_id, name: item.food_name })
    }
    foodOptions.value = picked
    editorReady.value = true
  } catch (e: unknown) {
    editorError.value = e instanceof Error ? e.message : '菜谱详情加载失败'
  } finally {
    editorLoading.value = false
  }
}

async function searchFoods(query: string) {
  foodSearching.value = true
  try {
    const res = await getDietFoods({
      keyword: query.trim() || undefined,
      page: 1,
      page_size: 20,
    })
    foodOptions.value = (res.list ?? []).map((food) => ({ id: food.id, name: food.name }))
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '食物搜索失败')
  } finally {
    foodSearching.value = false
  }
}

function onFoodPicked(item: RecipeItemDraft) {
  const picked = foodOptionsForForm.value.find((opt) => opt.id === item.food_id)
  item.food_name = picked?.name ?? ''
}

function addItem() {
  editorForm.items.push({ food_id: undefined, food_name: '', grams: null })
}

function removeItem(index: number) {
  editorForm.items.splice(index, 1)
}

async function submitEditor() {
  editorError.value = ''
  if (!editingId.value) return
  // 关闭弹窗后 @closed 会清空 editingId，保存前先捕获目标 id
  const targetId = editingId.value

  const name = editorForm.name.trim()
  if (!name) {
    editorError.value = '请输入菜谱名称'
    return
  }
  if (editorForm.items.length === 0) {
    editorError.value = '请至少添加一种食物'
    return
  }

  const items: DietRecipeSaveItem[] = []
  for (const item of editorForm.items) {
    if (typeof item.food_id !== 'number') {
      editorError.value = '每一行都必须选择食物'
      return
    }
    if (item.grams == null || item.grams <= 0) {
      editorError.value = '每一行的克数都必须大于 0'
      return
    }
    items.push({ food_id: item.food_id, grams: item.grams })
  }

  editorSaving.value = true
  try {
    await updateDietRecipe(targetId, { name, items })
    ElMessage.success('已保存')
    editorVisible.value = false
    await load()
    if (detail.value?.id === targetId) await loadDetail(targetId)
  } catch (e: unknown) {
    const err = e as DietApiError
    editorError.value = err.message || '保存失败'
    if (err.status !== 422) ElMessage.error(editorError.value)
  } finally {
    editorSaving.value = false
  }
}

// ===== 删除 =====

async function remove(row: DietRecipeListItem) {
  try {
    await ElMessageBox.confirm(`确定删除菜谱「${row.name}」？历史周食谱不受影响。`, '提示', {
      type: 'warning',
    })
    await deleteDietRecipe(row.id)
    ElMessage.success('已删除')
    if (detail.value?.id === row.id) detailVisible.value = false
    await load()
  } catch (e: unknown) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e instanceof Error ? e.message : '删除失败')
    }
  }
}

// ===== 合并重复 =====

const mergeVisible = ref(false)
const mergeSaving = ref(false)
const mergeError = ref('')
const mergeForm = reactive<{ source_id?: unknown; target_id?: unknown }>({
  source_id: undefined,
  target_id: undefined,
})
const mergeSourceOptions = ref<DietRecipeListItem[]>([])
const mergeTargetOptions = ref<DietRecipeListItem[]>([])
const mergeLoadingSource = ref(false)
const mergeLoadingTarget = ref(false)

function openMerge() {
  resetMerge()
  mergeVisible.value = true
  searchMergeSource('')
  searchMergeTarget('')
}

function resetMerge() {
  mergeError.value = ''
  mergeSaving.value = false
  mergeForm.source_id = undefined
  mergeForm.target_id = undefined
  mergeSourceOptions.value = []
  mergeTargetOptions.value = []
}

async function searchMergeSource(query: string) {
  mergeLoadingSource.value = true
  try {
    const res = await getDietRecipes({ keyword: query.trim() || undefined, page: 1, page_size: 20 })
    mergeSourceOptions.value = res.list ?? []
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '菜谱搜索失败')
  } finally {
    mergeLoadingSource.value = false
  }
}

async function searchMergeTarget(query: string) {
  mergeLoadingTarget.value = true
  try {
    const res = await getDietRecipes({ keyword: query.trim() || undefined, page: 1, page_size: 20 })
    mergeTargetOptions.value = res.list ?? []
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '菜谱搜索失败')
  } finally {
    mergeLoadingTarget.value = false
  }
}

async function submitMerge() {
  mergeError.value = ''
  const sourceId = mergeForm.source_id
  const targetId = mergeForm.target_id
  if (typeof sourceId !== 'number' || typeof targetId !== 'number') {
    mergeError.value = '请选择源菜谱与目标菜谱'
    return
  }
  if (sourceId === targetId) {
    mergeError.value = '源菜谱与目标菜谱不能相同'
    return
  }

  mergeSaving.value = true
  try {
    const res = await mergeDietRecipes({ source_id: sourceId, target_id: targetId })
    ElMessage.success(`已合并，目标菜谱现有 ${res.merged_item_count} 条配料`)
    mergeVisible.value = false
    if (detail.value?.id === sourceId) detailVisible.value = false
    await load()
    if (detail.value?.id === res.target_id) await loadDetail(res.target_id)
  } catch (e: unknown) {
    const err = e as DietApiError
    mergeError.value = err.message || '合并失败'
    if (err.status !== 422) ElMessage.error(mergeError.value)
  } finally {
    mergeSaving.value = false
  }
}
</script>
