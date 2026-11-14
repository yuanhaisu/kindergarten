<template>
  <!-- 文件路由会把 [id]/analysis.vue 生成为本页的子路由，父级必须提供出口；子路由自己渲染整页 -->
  <NuxtPage v-if="isChildRoute" />
  <div
    v-else
    ref="editorRoot"
    v-loading="loading"
    class="min-h-32 space-y-4"
    :class="{ 'overflow-auto bg-bg-page p-4': isFullscreen }"
  >
    <el-alert
      v-if="loadError"
      type="error"
      :closable="false"
      show-icon
      :title="loadError"
      data-testid="plan-load-error"
    >
      <div class="mt-2 flex gap-2">
        <el-button size="small" @click="loadDetail">重试</el-button>
        <el-button size="small" @click="goList">返回列表</el-button>
      </div>
    </el-alert>

    <template v-if="plan">
      <el-card shadow="never">
        <template #header>
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <el-button link type="primary" @click="goList">← 返回列表</el-button>
              <span class="text-lg font-semibold text-text-primary">
                {{ detailMode ? '周食谱详情' : '周食谱编辑' }}
              </span>
              <el-tag v-if="detailMode" size="small" type="info">只读</el-tag>
            </div>
            <div class="flex flex-wrap gap-2">
              <el-button data-testid="plan-fullscreen" @click="toggleFullscreen">
                {{ isFullscreen ? '退出全屏' : '全屏' }}
              </el-button>
              <el-button
                data-testid="plan-analysis"
                title="营养分析使用已保存的明细"
                @click="openAnalysis"
              >
                查看分析
              </el-button>
              <el-button
                v-if="detailMode && canManage"
                type="primary"
                data-testid="plan-switch-edit"
                @click="switchToEdit"
              >
                编辑
              </el-button>
              <el-button
                v-if="canEdit"
                type="primary"
                data-testid="plan-save"
                :loading="saving"
                @click="savePlan"
              >
                保存
              </el-button>
              <el-button v-if="canEdit" data-testid="plan-clear" @click="clearPlan">
                清空
              </el-button>
              <el-button
                v-if="canEdit"
                type="danger"
                plain
                data-testid="plan-delete"
                @click="removePlan"
              >
                删除
              </el-button>
            </div>
          </div>
        </template>

        <div class="space-y-3">
          <el-form label-width="88px">
            <el-form-item label="食谱名称" required>
              <div class="w-full max-w-xl" data-testid="plan-name">
                <el-input
                  v-model="planName"
                  maxlength="128"
                  :disabled="!canEdit"
                  placeholder="请输入食谱名称"
                />
              </div>
            </el-form-item>
          </el-form>

          <el-descriptions :column="3" size="small" border>
            <el-descriptions-item label="学年">
              {{ academicYearLabel || '—' }}
            </el-descriptions-item>
            <el-descriptions-item label="集体对象">
              <span data-testid="plan-group-object-name">{{ plan.group_object_name || '—' }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="周期">
              {{ plan.start_date }} 至 {{ plan.end_date }}
            </el-descriptions-item>
            <el-descriptions-item label="有明细天数">{{ plan.day_num }}</el-descriptions-item>
            <el-descriptions-item label="周次">
              {{ plan.week_of_year || '—' }}
            </el-descriptions-item>
            <el-descriptions-item label="创建方式">
              {{ plan.created_method_label }}
            </el-descriptions-item>
            <el-descriptions-item label="折算系数">
              {{ plan.config_snapshot.coefficient }}
            </el-descriptions-item>
            <el-descriptions-item label="已启用餐次" :span="3">
              {{ enabledMealNames || '无' }}
            </el-descriptions-item>
          </el-descriptions>

          <div class="text-xs text-text-muted">
            总明细 {{ totalItemCount }} 条；营养小计按「克数 × 食部% × 每 100g 浓度 / 100」实时计算，
            缺测项不按 0 计入。
          </div>
          <div v-if="!canEdit" class="text-sm text-text-muted">
            当前账号无 diet:manage 权限，仅可查看。
          </div>
        </div>
      </el-card>

      <el-alert
        v-if="saveError"
        type="error"
        :closable="true"
        show-icon
        title="保存失败"
        :description="saveError"
        data-testid="plan-save-error"
        @close="saveError = ''"
      />

      <el-card v-if="mealRows.length === 0" shadow="never">
        <el-empty description="所属集体对象没有启用任何餐次，无法编辑周食谱">
          <el-button type="primary" @click="router.push('/diet/config')">前往基本参数设置</el-button>
        </el-empty>
      </el-card>

      <template v-else>
        <el-card shadow="never" v-loading="loading">
          <template #header>
            <div class="flex flex-wrap items-center justify-between gap-2">
              <span>7 天 × 餐次网格</span>
              <span class="text-xs text-text-muted">菜与食物的上下顺序即保存顺序</span>
            </div>
          </template>

          <el-alert
            v-if="totalItemCount === 0"
            type="info"
            :closable="false"
            show-icon
            title="还没有明细"
            description="点击任意餐次单元格里的「添加菜」开始录入；保存后按服务端计算的天数展示。"
            class="mb-3"
            data-testid="plan-empty-hint"
          />

          <el-table :data="mealRows" border class="w-full" row-key="meal_key">
            <el-table-column label="餐次" width="96" fixed>
              <template #default="{ row }">
                <div class="text-sm font-medium text-text-primary">{{ (row as MealRow).name }}</div>
                <div class="text-xs text-text-muted">{{ (row as MealRow).time }}</div>
              </template>
            </el-table-column>

            <el-table-column
              v-for="day in days"
              :key="day.date"
              :label="`${day.label} ${day.date.slice(5)}`"
              :min-width="isFullscreen ? 200 : 280"
            >
              <template #default="{ row }">
                <div
                  :data-testid="`meal-cell-${day.weekday}-${(row as MealRow).meal_key}`"
                  class="flex min-h-24 flex-col gap-2"
                >
                  <div
                    v-for="(dish, dishIndex) in dishesOf(day.date, (row as MealRow).meal_key)"
                    :key="dish.key"
                    class="rounded-md border border-border-hairline p-2"
                    :class="{
                      'bg-semantic-dangerBgSubtle ring-2 ring-semantic-danger':
                        highlightKey === dish.key,
                    }"
                    :data-highlight-key="dish.key"
                  >
                    <div class="text-sm font-medium text-text-primary break-all">{{ dish.name }}</div>

                    <div
                      v-if="canEdit"
                      class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5"
                    >
                      <el-button
                        link
                        size="small"
                        @click="openDishEditor(day.date, (row as MealRow).meal_key, dish)"
                      >
                        编辑
                      </el-button>
                      <el-button
                        link
                        size="small"
                        @click="openDishEditor(day.date, (row as MealRow).meal_key, dish, true)"
                      >
                        加食物
                      </el-button>
                      <el-button
                        link
                        size="small"
                        data-testid="dish-save-recipe"
                        @click="saveDishRecipe(dish)"
                      >
                        存菜谱
                      </el-button>
                      <el-button
                        link
                        size="small"
                        :disabled="dishIndex === 0"
                        @click="moveDish(day.date, (row as MealRow).meal_key, dishIndex, -1)"
                      >
                        上移
                      </el-button>
                      <el-button
                        link
                        size="small"
                        :disabled="
                          dishIndex ===
                          dishesOf(day.date, (row as MealRow).meal_key).length - 1
                        "
                        @click="moveDish(day.date, (row as MealRow).meal_key, dishIndex, 1)"
                      >
                        下移
                      </el-button>
                      <el-button
                        link
                        size="small"
                        type="danger"
                        @click="removeDish(day.date, (row as MealRow).meal_key, dishIndex)"
                      >
                        移除
                      </el-button>
                    </div>

                    <div
                      v-for="(food, foodIndex) in dish.foods"
                      :key="food.key"
                      class="mt-1 rounded border border-border-hairline p-1"
                      :class="{
                        'bg-semantic-dangerBgSubtle ring-2 ring-semantic-danger':
                          highlightKey === food.key,
                      }"
                      :data-highlight-key="food.key"
                    >
                      <div class="flex items-center gap-2">
                        <span class="min-w-0 flex-1 text-xs text-text-body break-all">
                          {{ food.food_name || '未选择食物' }}
                        </span>
                        <el-input-number
                          v-model="food.grams"
                          :min="0"
                          :controls="false"
                          size="small"
                          class="!w-20"
                          data-testid="plan-food-grams"
                          :disabled="!canEdit"
                          @input="markFoodDirty(food)"
                        />
                        <span class="text-xs text-text-muted">克</span>
                      </div>

                      <div v-if="canEdit" class="mt-0.5 flex flex-wrap items-center gap-x-2">
                        <el-button
                          link
                          size="small"
                          :disabled="foodIndex === 0"
                          @click="moveFood(dish, foodIndex, -1)"
                        >
                          上移
                        </el-button>
                        <el-button
                          link
                          size="small"
                          :disabled="foodIndex === dish.foods.length - 1"
                          @click="moveFood(dish, foodIndex, 1)"
                        >
                          下移
                        </el-button>
                        <el-button link size="small" @click="openReplace(food)">同类替换</el-button>
                        <el-button
                          link
                          size="small"
                          type="danger"
                          @click="removeFood(dish, foodIndex)"
                        >
                          移除
                        </el-button>
                      </div>
                    </div>

                    <div v-if="dish.foods.length === 0" class="mt-1 text-xs text-semantic-danger">
                      该菜还没有食物
                    </div>
                  </div>

                  <el-button
                    v-if="canEdit"
                    size="small"
                    class="w-full"
                    :data-testid="`add-dish-${day.weekday}-${(row as MealRow).meal_key}`"
                    @click="openDishEditor(day.date, (row as MealRow).meal_key)"
                  >
                    添加菜
                  </el-button>

                  <div class="mt-1 border-t border-border-hairline pt-1">
                    <div class="mb-0.5 text-xs text-text-muted">本餐小计</div>
                    <NutritionTotals
                      :totals="cellTotals(day.date, (row as MealRow).meal_key)"
                      :test-id="`meal-subtotal-${day.weekday}-${(row as MealRow).meal_key}`"
                    />
                  </div>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <el-card shadow="never">
          <template #header>
            <div class="flex flex-wrap items-center justify-between gap-2">
              <span>每日营养合计</span>
              <span class="text-xs text-text-muted">全部已启用餐次相加；缺测项显示「—」</span>
            </div>
          </template>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
            <div
              v-for="day in days"
              :key="day.date"
              class="rounded-md border border-border-hairline p-3"
            >
              <div class="mb-1 text-sm font-medium text-text-primary">
                {{ day.label }} {{ day.date }}
              </div>
              <NutritionTotals
                :totals="dayTotals(day.date)"
                :test-id="`day-total-${day.weekday}`"
              />
            </div>
          </div>
        </el-card>
      </template>
    </template>

    <DishEditorDialog
      v-model="dishDialogVisible"
      :mode="dishDialogMode"
      :context-label="dishDialogLabel"
      :initial="dishDialogInitial"
      :append-empty-row="dishDialogAppendRow"
      @submit="onDishSubmit"
    />
    <ReplaceFoodDialog
      v-model="replaceVisible"
      :food-id="replaceTarget?.food_id ?? null"
      :food-name="replaceTarget?.food_name ?? ''"
      @picked="onFoodReplaced"
    />
    <RecipeConflictDialog
      v-model="conflictVisible"
      :recipe-name="conflictDish?.name ?? ''"
      :saving="recipeSaving"
      @replace="onConflictReplace"
      @apply-once="onConflictOnce"
      @cancel="onConflictCancel"
    />
  </div>
</template>

<script setup lang="ts">
import type {
  DietApiError,
  DietFoodDetail,
  DietMealKey,
  DietPlanDetail,
  DietPlanItemInput,
  DietRecipeSaveItem,
} from '~/types/api'
import {
  clearDietPlan,
  createDietRecipe,
  deleteDietPlan,
  getDietAcademicYears,
  getDietFood,
  getDietPlan,
  updateDietPlan,
} from '~/api/diet'
import NutritionTotals from './components/NutritionTotals.vue'
import DishEditorDialog from './components/DishEditorDialog.vue'
import ReplaceFoodDialog from './components/ReplaceFoodDialog.vue'
import RecipeConflictDialog from './components/RecipeConflictDialog.vue'
import {
  cellKey,
  nextKey,
  type CoreTotals,
  type DishDraft,
  type DishDraftPayload,
  type FoodDraft,
  type MealRow,
  type WeekDay,
} from './components/plan-types'
import {
  buildWeekDays,
  dishesCoreTotals,
  sumCoreTotals,
} from './components/plan-utils'

const route = useRoute()
const router = useRouter()
const { can } = usePermission()
const canManage = computed(() => can('diet:manage'))
const detailMode = computed(() => route.query.mode === 'detail')
const canEdit = computed(() => canManage.value && !detailMode.value)
const isChildRoute = computed(() => route.path.endsWith('/analysis'))

const plan = ref<DietPlanDetail | null>(null)
const planName = ref('')
const loading = ref(false)
const loadError = ref('')
const academicYearLabel = ref('')
const cells = ref<Record<string, DishDraft[]>>({})
const loadedItemCount = ref(0)

const saving = ref(false)
const saveError = ref('')
const highlightKey = ref('')
const editorRoot = ref<HTMLElement | null>(null)
const isFullscreen = ref(false)

function syncFullscreen() {
  isFullscreen.value = !!document.fullscreenElement
}

async function toggleFullscreen() {
  try {
    if (document.fullscreenElement) {
      await document.exitFullscreen()
    } else if (editorRoot.value) {
      await editorRoot.value.requestFullscreen()
    }
  } catch {
    ElMessage.warning('当前浏览器不支持全屏')
  }
}
let highlightTimer: ReturnType<typeof setTimeout> | null = null

const mealRows = computed<MealRow[]>(() => {
  const meals = plan.value?.config_snapshot?.meals ?? []
  return meals
    .filter((meal) => meal.enabled)
    .slice()
    .sort((a, b) => a.sort - b.sort)
    .map((meal) => ({
      meal_key: meal.meal_key,
      name: meal.name,
      time: meal.time,
      sort: meal.sort,
    }))
})

const days = computed<WeekDay[]>(() =>
  plan.value ? buildWeekDays(plan.value.start_date) : [],
)

const enabledMealNames = computed(() => mealRows.value.map((meal) => meal.name).join('、'))

const totalItemCount = computed(() =>
  Object.values(cells.value).reduce(
    (sum, dishes) => sum + dishes.reduce((acc, dish) => acc + dish.foods.length, 0),
    0,
  ),
)

onMounted(() => {
  document.addEventListener('fullscreenchange', syncFullscreen)
  if (!isChildRoute.value) void loadDetail()
})
onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', syncFullscreen)
  if (document.fullscreenElement) void document.exitFullscreen()
})
watch(
  () => route.params.id,
  () => {
    if (!isChildRoute.value) void loadDetail()
  },
)
// 从营养分析子路由返回时父组件不会重新挂载：仅在尚未载入过详情时补载，
// 避免把编辑器里未保存的改动冲掉
watch(isChildRoute, (child) => {
  if (!child && !plan.value) void loadDetail()
})

async function loadDetail() {
  const id = Number(route.params.id)
  if (!Number.isFinite(id) || id <= 0) {
    plan.value = null
    loadError.value = '周食谱 ID 无效'
    return
  }
  loading.value = true
  loadError.value = ''
  clearHighlight()
  try {
    const detail = await getDietPlan(id)
    plan.value = detail
    planName.value = detail.name
    rebuildCells(detail)
    loadedItemCount.value = detail.items?.length ?? 0
    void loadYearLabel(detail.academic_year_id)
  } catch (e: unknown) {
    const err = e as DietApiError
    plan.value = null
    loadError.value =
      err.status === 404 ? '周食谱不存在或已被删除' : err.message || '周食谱加载失败'
  } finally {
    loading.value = false
  }
}

async function loadYearLabel(academicYearId: number) {
  try {
    const res = await getDietAcademicYears()
    academicYearLabel.value =
      res.list?.find((year) => year.id === academicYearId)?.year ?? `#${academicYearId}`
  } catch {
    academicYearLabel.value = `#${academicYearId}`
    console.warn('[diet-plan] 学年名称加载失败，回退显示 ID', academicYearId)
  }
}

function rebuildCells(detail: DietPlanDetail) {
  const next: Record<string, DishDraft[]> = {}
  for (const day of buildWeekDays(detail.start_date)) {
    for (const meal of mealRows.value) {
      next[cellKey(day.date, meal.meal_key)] = []
    }
  }

  let missingFood = 0
  for (const item of detail.items ?? []) {
    const key = cellKey(item.date, item.meal_key)
    if (!next[key]) next[key] = []
    if (!item.food_id) {
      missingFood += 1
      continue
    }
    const dishes = next[key]
    let dish = dishes.find((candidate) => candidate.name === item.recipe_name)
    if (!dish) {
      dish = {
        key: nextKey('dish'),
        name: item.recipe_name,
        recipe_id: item.recipe_id ?? null,
        foods: [],
      }
      dishes.push(dish)
    }
    dish.foods.push({
      key: nextKey('food'),
      food_id: item.food_id,
      food_name: item.food_name ?? '',
      grams: item.grams,
      snapshot: item.nutrition_snapshot ?? null,
      detail: null,
      dirty: false,
      loadingDetail: false,
    })
  }

  cells.value = next
  if (missingFood > 0) {
    ElMessage.warning(`有 ${missingFood} 条历史明细缺少食物，未载入编辑器`)
  }
}

function dishesOf(date: string, mealKey: DietMealKey): DishDraft[] {
  return cells.value[cellKey(date, mealKey)] ?? []
}

function cellTotals(date: string, mealKey: DietMealKey): CoreTotals {
  return dishesCoreTotals(dishesOf(date, mealKey))
}

function dayTotals(date: string): CoreTotals {
  return sumCoreTotals(
    mealRows.value.map((meal) => dishesCoreTotals(dishesOf(date, meal.meal_key))),
  )
}

// ===== 网格内编辑 =====

function moveDish(date: string, mealKey: DietMealKey, index: number, delta: number) {
  const dishes = cells.value[cellKey(date, mealKey)]
  if (!dishes) return
  const target = index + delta
  if (target < 0 || target >= dishes.length) return
  const [moved] = dishes.splice(index, 1)
  dishes.splice(target, 0, moved)
}

function removeDish(date: string, mealKey: DietMealKey, index: number) {
  cells.value[cellKey(date, mealKey)]?.splice(index, 1)
}

function moveFood(dish: DishDraft, index: number, delta: number) {
  const target = index + delta
  if (target < 0 || target >= dish.foods.length) return
  const [moved] = dish.foods.splice(index, 1)
  dish.foods.splice(target, 0, moved)
}

function removeFood(dish: DishDraft, index: number) {
  dish.foods.splice(index, 1)
}

function markFoodDirty(food: FoodDraft) {
  food.dirty = true
  if (!food.detail) void ensureFoodDetail(food)
}

async function ensureFoodDetail(food: FoodDraft) {
  if (food.detail || food.loadingDetail) return
  food.loadingDetail = true
  const target = food.food_id
  try {
    const detail = await getDietFood(target)
    if (food.food_id === target) {
      food.detail = detail
      food.food_name = detail.name
    }
  } catch (e: unknown) {
    ElMessage.error(
      `「${food.food_name}」营养数据加载失败：${e instanceof Error ? e.message : '网络错误'}`,
    )
  } finally {
    food.loadingDetail = false
  }
}

// ===== 添加 / 编辑菜 =====

const dishDialogVisible = ref(false)
const dishDialogMode = ref<'create' | 'edit'>('create')
const dishDialogLabel = ref('')
const dishDialogInitial = ref<DishDraftPayload | null>(null)
const dishDialogAppendRow = ref(false)
const dishDialogTarget = ref<{
  date: string
  mealKey: DietMealKey
  dishKey: string | null
} | null>(null)

function openDishEditor(
  date: string,
  mealKey: DietMealKey,
  dish?: DishDraft,
  appendEmptyRow = false,
) {
  if (!canEdit.value) return
  const day = days.value.find((candidate) => candidate.date === date)
  const meal = mealRows.value.find((candidate) => candidate.meal_key === mealKey)
  dishDialogMode.value = dish ? 'edit' : 'create'
  dishDialogLabel.value = `${day?.label ?? date} · ${meal?.name ?? mealKey}`
  dishDialogTarget.value = { date, mealKey, dishKey: dish?.key ?? null }
  dishDialogAppendRow.value = appendEmptyRow
  dishDialogInitial.value = dish
    ? {
        name: dish.name,
        recipe_id: dish.recipe_id,
        foods: dish.foods.map((food) => ({ ...food })),
      }
    : null
  dishDialogVisible.value = true
}

function onDishSubmit(payload: DishDraftPayload) {
  const target = dishDialogTarget.value
  if (!target) return
  const key = cellKey(target.date, target.mealKey)
  if (!cells.value[key]) cells.value[key] = []
  const dishes = cells.value[key]

  if (target.dishKey) {
    const dish = dishes.find((candidate) => candidate.key === target.dishKey)
    if (dish) {
      dish.name = payload.name
      dish.recipe_id = payload.recipe_id
      dish.foods = payload.foods
    }
  } else {
    // 契约按「日期 + 餐次 + 菜名」分组：同名菜合并到同一块，避免界面出现两个同名块
    const sameName = dishes.find((candidate) => candidate.name === payload.name)
    if (sameName) {
      sameName.foods.push(...payload.foods)
      if (sameName.recipe_id == null) sameName.recipe_id = payload.recipe_id
    } else {
      dishes.push({
        key: nextKey('dish'),
        name: payload.name,
        recipe_id: payload.recipe_id,
        foods: payload.foods,
      })
    }
  }
  dishDialogVisible.value = false
}

// ===== 同类替换 =====

const replaceVisible = ref(false)
const replaceTarget = ref<FoodDraft | null>(null)

function openReplace(food: FoodDraft) {
  if (!canEdit.value) return
  replaceTarget.value = food
  replaceVisible.value = true
}

function onFoodReplaced(detail: DietFoodDetail) {
  const food = replaceTarget.value
  replaceVisible.value = false
  if (!food) return
  const grams = food.grams
  food.food_id = detail.id
  food.food_name = detail.name
  food.grams = grams
  food.detail = detail
  food.snapshot = null
  food.dirty = true
  food.loadingDetail = false
  ElMessage.success(`已替换为「${detail.name}」，克数保持不变`)
}

// ===== 保存菜谱（替换 / 仅本次） =====

const recipeSaving = ref(false)
const conflictVisible = ref(false)
const conflictDish = ref<DishDraft | null>(null)

function dishRecipeItems(dish: DishDraft): DietRecipeSaveItem[] | null {
  if (!dish.name.trim() || dish.foods.length === 0) return null
  const items: DietRecipeSaveItem[] = []
  for (const food of dish.foods) {
    if (!food.food_id) return null
    if (food.grams == null || !Number.isFinite(food.grams) || food.grams <= 0) return null
    items.push({ food_id: food.food_id, grams: food.grams })
  }
  return items
}

async function saveDishRecipe(dish: DishDraft) {
  const items = dishRecipeItems(dish)
  if (!items) {
    ElMessage.warning(`「${dish.name || '未命名菜'}」需要至少一种食物且克数大于 0 才能保存菜谱`)
    return
  }
  const name = dish.name.trim()
  recipeSaving.value = true
  try {
    const res = await createDietRecipe({ name, items })
    dish.recipe_id = res.id
    ElMessage.success(`菜谱「${name}」已保存到菜谱库`)
  } catch (e: unknown) {
    const err = e as DietApiError
    if (err.status === 409) {
      conflictDish.value = dish
      conflictVisible.value = true
    } else {
      ElMessage.error(err.message || '保存菜谱失败')
    }
  } finally {
    recipeSaving.value = false
  }
}

async function onConflictReplace() {
  const dish = conflictDish.value
  if (!dish) return
  const items = dishRecipeItems(dish)
  if (!items) {
    conflictVisible.value = false
    return
  }
  recipeSaving.value = true
  try {
    const res = await createDietRecipe({ name: dish.name.trim(), items, replace: true })
    dish.recipe_id = res.id
    ElMessage.success(`已替换原菜谱「${dish.name}」`)
    conflictVisible.value = false
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '替换菜谱失败')
  } finally {
    recipeSaving.value = false
  }
}

function onConflictOnce() {
  conflictVisible.value = false
  ElMessage.info('本次配方仅保留在当前周食谱中，未写入菜谱库')
}

function onConflictCancel() {
  conflictVisible.value = false
}

// ===== 整单保存 / 清空 / 删除 =====

function highlight(key: string) {
  highlightKey.value = key
  if (highlightTimer) clearTimeout(highlightTimer)
  highlightTimer = setTimeout(() => {
    highlightKey.value = ''
  }, 4000)
  void nextTick(() => {
    document
      .querySelector(`[data-highlight-key="${key}"]`)
      ?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  })
}

function clearHighlight() {
  highlightKey.value = ''
  if (highlightTimer) {
    clearTimeout(highlightTimer)
    highlightTimer = null
  }
}

function failLocal(message: string, target?: DishDraft | FoodDraft) {
  saveError.value = message
  if (target) highlight(target.key)
  ElMessage.error(message)
}

function locateSaveError(message: string) {
  for (const dishes of Object.values(cells.value)) {
    for (const dish of dishes) {
      if (!message.includes(dish.name)) continue
      const food = dish.foods.find((candidate) => message.includes(candidate.food_name))
      highlight((food ?? dish).key)
      return
    }
  }
}

async function savePlan() {
  if (!plan.value) return
  saveError.value = ''
  clearHighlight()

  const name = planName.value.trim()
  if (!name) {
    saveError.value = '请输入食谱名称'
    return
  }

  const items: DietPlanItemInput[] = []
  for (const day of days.value) {
    for (const meal of mealRows.value) {
      for (const dish of dishesOf(day.date, meal.meal_key)) {
        const dishName = dish.name.trim()
        if (!dishName) {
          failLocal(`「${dish.name || '未命名菜'}」的菜名不能为空`, dish)
          return
        }
        if (dish.foods.length === 0) {
          failLocal(`「${dishName}」没有任何食物`, dish)
          return
        }
        for (const food of dish.foods) {
          if (!food.food_id) {
            failLocal(`「${dishName}」有食物未选择`, dish)
            return
          }
          if (food.grams == null || !Number.isFinite(food.grams) || food.grams <= 0) {
            failLocal(`「${dishName}」的「${food.food_name}」一人均量必须大于 0`, food)
            return
          }
          items.push({
            date: day.date,
            meal_key: meal.meal_key,
            recipe_name: dishName,
            recipe_id: dish.recipe_id,
            food_id: food.food_id,
            food_name: food.food_name,
            grams: food.grams,
          })
        }
      }
    }
  }

  if (items.length === 0 && loadedItemCount.value > 0) {
    try {
      await ElMessageBox.confirm(
        '当前没有任何明细，保存将清空该食谱的全部明细。确认保存？',
        '保存确认',
        { type: 'warning', confirmButtonText: '确认清空并保存' },
      )
    } catch {
      return
    }
  }

  const target = plan.value
  saving.value = true
  try {
    const res = await updateDietPlan(target.id, { name, items })
    ElMessage.success(`已保存：${res.item_count} 条明细，覆盖 ${res.day_num} 天`)
    await loadDetail()
  } catch (e: unknown) {
    const err = e as DietApiError
    saveError.value = err.message || '保存失败'
    if (err.status === 422) {
      locateSaveError(err.message)
    } else {
      ElMessage.error(saveError.value)
    }
  } finally {
    saving.value = false
  }
}

async function clearPlan() {
  if (!plan.value) return
  try {
    await ElMessageBox.confirm(
      `将清空「${plan.value.name}」的全部明细（${totalItemCount.value} 条），食谱本身保留。`,
      '清空确认',
      { type: 'warning', confirmButtonText: '下一步' },
    )
    await ElMessageBox.confirm(
      '再次确认：清空后明细无法恢复，营养分析也会因没有明细而无法查看。',
      '最终确认',
      { type: 'warning', confirmButtonText: '确认清空' },
    )
  } catch {
    return
  }
  try {
    const res = await clearDietPlan(plan.value.id)
    ElMessage.success(`已清空，当前有明细 ${res.day_num} 天`)
    await loadDetail()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '清空失败')
  }
}

async function removePlan() {
  if (!plan.value) return
  try {
    await ElMessageBox.confirm(
      `确定删除周食谱「${plan.value.name}」？`,
      '删除确认',
      { type: 'warning', confirmButtonText: '下一步' },
    )
    await ElMessageBox.confirm(
      '再次确认：删除后该食谱与营养分析都将不再返回，且无法恢复。',
      '最终确认',
      { type: 'warning', confirmButtonText: '确认删除' },
    )
  } catch {
    return
  }
  try {
    await deleteDietPlan(plan.value.id)
    ElMessage.success('已删除')
    await goList()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}

function switchToEdit() {
  router.replace(`/diet/plans/${route.params.id}`)
}

function openAnalysis() {
  const id = Number(route.params.id)
  if (!Number.isFinite(id) || id <= 0) return
  return router.push(`/diet/plans/${id}/analysis`)
}

function goList() {
  return router.push('/diet/plans')
}
</script>
