<template>
  <el-dialog
    v-model="visible"
    :title="mode === 'edit' ? '编辑菜' : '添加菜'"
    width="780px"
    destroy-on-close
    @open="reset"
  >
    <div class="space-y-3">
      <div class="text-text-muted text-sm">{{ contextLabel }}</div>

      <el-alert
        v-if="error"
        type="error"
        :closable="false"
        show-icon
        title="无法完成操作"
        :description="error"
        data-testid="dish-error"
      />

      <el-form label-width="80px">
        <el-form-item label="菜名" required>
          <div class="w-full" data-testid="dish-name">
            <el-autocomplete
              v-model="name"
              :fetch-suggestions="fetchSuggestions"
              placeholder="输入菜名联想菜谱库；新名称也可直接使用"
              clearable
              class="w-full"
              :disabled="loadingRecipe"
              @select="onRecipePicked"
            >
              <template #default="{ item }">
                <div class="flex w-full items-center justify-between gap-2">
                  <span>{{ item.value }}</span>
                  <span class="text-text-muted text-xs">{{ item.itemCount }} 项配料</span>
                </div>
              </template>
            </el-autocomplete>
          </div>
          <div class="text-text-muted text-xs leading-5 mt-1">
            选中联想菜谱会替换下方食物明细；手动输入的新菜名不会自动写入菜谱库。
          </div>
          <div v-if="loadingRecipe" class="text-text-muted text-xs mt-1">正在载入菜谱配方…</div>
        </el-form-item>
      </el-form>

      <div>
        <div class="mb-2 flex items-center justify-between">
          <span class="text-text-body text-sm font-medium">食物与一人均量</span>
          <el-button size="small" data-testid="dish-add-food" @click="addFoodRow">添加食物</el-button>
        </div>

        <div
          v-if="rows.length === 0"
          class="rounded border border-border-hairline p-4 text-center text-sm text-text-muted"
        >
          暂无食物。输入菜名联想菜谱，或点击「添加食物」手动选择。
        </div>

        <div
          v-for="(row, index) in rows"
          :key="row.key"
          class="mb-2 rounded border border-border-hairline p-2"
        >
          <div class="flex items-center gap-2">
            <div class="min-w-0 flex-1" data-testid="dish-food-select">
              <el-select
                v-model="row.food_id"
                filterable
                remote
                clearable
                placeholder="搜索食物"
                :remote-method="searchFoods"
                :loading="foodSearching"
                class="w-full"
                @change="onFoodPicked(row)"
              >
                <el-option
                  v-for="option in optionsFor(row)"
                  :key="option.id"
                  :label="option.name"
                  :value="option.id"
                />
              </el-select>
            </div>
            <el-input-number
              v-model="row.grams"
              :min="0"
              :controls="false"
              placeholder="克数"
              class="!w-28"
              data-testid="dish-grams"
              @input="markRowDirty(row)"
            />
            <span class="text-text-muted text-xs">克</span>
            <el-button link type="danger" @click="removeRow(index)">移除</el-button>
          </div>
          <div v-if="row.loadingDetail" class="text-text-muted text-xs mt-1">
            正在读取「{{ row.food_name }}」的营养数据…
          </div>
          <div
            v-else-if="typeof row.food_id === 'number' && !row.detail && !row.snapshot"
            class="text-semantic-danger text-xs mt-1"
          >
            未取得营养数据，请重新选择食物
          </div>
          <div v-else-if="row.detail" class="text-text-muted text-xs mt-1">
            食部 {{ row.detail.edible_percent == null ? '—' : `${row.detail.edible_percent}%` }} ·
            {{ row.detail.category_name }}
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" data-testid="dish-confirm" :disabled="loadingRecipe" @click="submit">
        确定
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import type { DietFoodDetail, DietNutritionSnapshot } from '~/types/api'
import { getDietFood, getDietFoods, getDietRecipe, suggestDietRecipes } from '~/api/diet'
import type { DishDraftPayload, FoodDraft } from './plan-types'
import { nextKey } from './plan-types'

interface RecipeSuggestion {
  value: string
  id: number
  itemCount: number
  [key: string]: unknown
}

interface FoodRow {
  key: string
  food_id: number | null | undefined
  food_name: string
  grams: number | null | undefined
  snapshot: DietNutritionSnapshot | null
  detail: DietFoodDetail | null
  dirty: boolean
  loadingDetail: boolean
  loadingToken: number
}

interface FoodOption {
  id: number
  name: string
}

const props = defineProps<{
  mode: 'create' | 'edit'
  contextLabel: string
  initial: DishDraftPayload | null
  /** 从网格的「加食物」进入时，在已有食物后追加一行空行 */
  appendEmptyRow?: boolean
}>()

const emit = defineEmits<{
  submit: [dish: DishDraftPayload]
}>()

const visible = defineModel<boolean>({ required: true })

const name = ref('')
const recipeId = ref<number | null>(null)
const rows = ref<FoodRow[]>([])
const error = ref('')
const loadingRecipe = ref(false)
const foodOptions = ref<FoodOption[]>([])
const foodSearching = ref(false)

function reset() {
  name.value = props.initial?.name ?? ''
  recipeId.value = props.initial?.recipe_id ?? null
  rows.value = (props.initial?.foods ?? []).map((food) => ({ ...food, loadingToken: 0 }))
  if (props.appendEmptyRow) addFoodRow()
  error.value = ''
  loadingRecipe.value = false
  foodSearching.value = false
  foodOptions.value = []
}

function makeRow(input: {
  food_id: number
  food_name: string
  grams: number | null | undefined
  detail?: DietFoodDetail | null
  snapshot?: DietNutritionSnapshot | null
  dirty?: boolean
}): FoodRow {
  return {
    key: nextKey('fd'),
    food_id: input.food_id,
    food_name: input.food_name,
    grams: input.grams,
    snapshot: input.snapshot ?? null,
    detail: input.detail ?? null,
    dirty: input.dirty ?? true,
    loadingDetail: false,
    loadingToken: 0,
  }
}

async function fetchSuggestions(query: string, callback: (items: RecipeSuggestion[]) => void) {
  const keyword = query.trim()
  if (!keyword) {
    callback([])
    return
  }
  try {
    const res = await suggestDietRecipes({ keyword, limit: 10 })
    callback(
      (res.list ?? []).map((recipe) => ({
        value: recipe.name,
        id: recipe.id,
        itemCount: recipe.item_count,
      })),
    )
  } catch (e: unknown) {
    // 联想失败不打断手动输入；真正的错误在选中后载入配方时反馈
    console.warn('[diet-dish-editor] 菜谱联想失败', e)
    callback([])
  }
}

async function onRecipePicked(item: Record<string, unknown>) {
  const id = Number(item.id)
  if (!Number.isFinite(id) || id <= 0) return
  error.value = ''
  loadingRecipe.value = true
  try {
    const recipe = await getDietRecipe(id)
    const nextRows = await Promise.all(
      (recipe.items ?? []).map(async (recipeItem) => {
        const detail = await getDietFood(recipeItem.food_id)
        return makeRow({
          food_id: recipeItem.food_id,
          food_name: recipeItem.food_name,
          grams: recipeItem.grams,
          detail,
        })
      }),
    )
    name.value = recipe.name
    recipeId.value = recipe.id
    rows.value = nextRows
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : '菜谱配方载入失败'
  } finally {
    loadingRecipe.value = false
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

function optionsFor(row: FoodRow): FoodOption[] {
  const list = foodOptions.value.slice()
  if (typeof row.food_id === 'number' && !list.some((option) => option.id === row.food_id)) {
    list.unshift({ id: row.food_id, name: row.food_name })
  }
  return list
}

function addFoodRow() {
  rows.value.push({
    key: nextKey('fd'),
    food_id: undefined,
    food_name: '',
    grams: null,
    snapshot: null,
    detail: null,
    dirty: true,
    loadingDetail: false,
    loadingToken: 0,
  })
}

function removeRow(index: number) {
  rows.value.splice(index, 1)
}

function onFoodPicked(row: FoodRow) {
  error.value = ''
  if (typeof row.food_id !== 'number') {
    row.food_name = ''
    row.detail = null
    row.snapshot = null
    row.dirty = true
    return
  }
  const picked = foodOptions.value.find((option) => option.id === row.food_id)
  row.food_name = picked?.name ?? row.food_name
  row.snapshot = null
  row.detail = null
  row.dirty = true
  void loadDetail(row)
}

function markRowDirty(row: FoodRow) {
  row.dirty = true
  if (!row.detail && typeof row.food_id === 'number') void loadDetail(row)
}

async function loadDetail(row: FoodRow) {
  if (typeof row.food_id !== 'number') return
  const target = row.food_id
  const token = row.loadingToken + 1
  row.loadingToken = token
  row.loadingDetail = true
  try {
    const detail = await getDietFood(target)
    if (row.loadingToken !== token || row.food_id !== target) return
    row.detail = detail
    row.food_name = detail.name
  } catch (e: unknown) {
    if (row.loadingToken === token) {
      error.value = `食物「${row.food_name || target}」营养加载失败：${
        e instanceof Error ? e.message : '网络错误'
      }`
    }
  } finally {
    if (row.loadingToken === token) row.loadingDetail = false
  }
}

function submit() {
  error.value = ''
  const dishName = name.value.trim()
  if (!dishName) {
    error.value = '请输入菜名'
    return
  }
  if (rows.value.length === 0) {
    error.value = '请至少添加一种食物'
    return
  }

  const foods: FoodDraft[] = []
  for (const row of rows.value) {
    if (typeof row.food_id !== 'number') {
      error.value = `「${dishName}」有食物未选择`
      return
    }
    if (row.loadingDetail) {
      error.value = `「${row.food_name || dishName}」营养数据加载中，请稍候再试`
      return
    }
    if (row.grams == null || !Number.isFinite(row.grams) || row.grams <= 0) {
      error.value = `「${dishName}」的「${row.food_name || row.food_id}」一人均量必须大于 0`
      return
    }
    if (!row.detail && !row.snapshot) {
      error.value = `「${row.food_name || row.food_id}」缺少营养数据，请重新选择该食物`
      return
    }
    foods.push({
      key: row.key,
      food_id: row.food_id,
      food_name: row.food_name,
      grams: row.grams,
      snapshot: row.snapshot,
      detail: row.detail,
      dirty: row.dirty,
      loadingDetail: false,
    })
  }

  emit('submit', { name: dishName, recipe_id: recipeId.value, foods })
}
</script>
