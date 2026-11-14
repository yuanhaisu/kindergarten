<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap justify-between gap-2">
        <span>资源库</span>
        <div v-if="canManage" class="flex gap-2">
          <el-button type="danger" plain :disabled="selectedRows.length === 0" @click="batchRemove">
            批量删除{{ selectedRows.length ? `(${selectedRows.length})` : '' }}
          </el-button>
          <el-button type="primary" @click="openCreate">新增食物</el-button>
        </div>
      </div>
    </template>

    <SearchBar
      v-model:keyword="keyword"
      keyword-placeholder="食物名称"
      @search="onSearch"
      @reset="onSearch"
    >
      <el-cascader
        v-model="categoryId"
        :options="categoryOptions"
        :props="filterCategoryProps"
        placeholder="全部分类"
        clearable
        class="!w-52"
        @change="onSearch"
      />
    </SearchBar>

    <el-alert
      v-if="listError"
      type="error"
      :closable="false"
      show-icon
      title="食物列表加载失败"
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
      empty-text="暂无食物"
      @selection-change="onSelectionChange"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column v-if="canManage" type="selection" width="46" />
      <el-table-column prop="name" label="食物名称" min-width="150" fixed="left" />
      <el-table-column label="来源" width="100">
        <template #default="{ row }">
          <el-tag :type="(row as DietFoodListItem).source === 'system' ? 'info' : 'success'" size="small">
            {{ sourceLabel((row as DietFoodListItem).source) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="食部(%)" width="90">
        <template #default="{ row }">{{ formatNutrient((row as DietFoodListItem).edible_percent) }}</template>
      </el-table-column>
      <el-table-column
        v-for="col in LIST_NUTRIENTS"
        :key="col.prop"
        :label="`${col.label}(${col.unit})`"
        :prop="col.prop"
        min-width="112"
      >
        <template #default="{ row }">{{ nutrientValue(row as DietFoodListItem, col.prop) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="170" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail((row as DietFoodListItem).id)">详情</el-button>
          <el-button v-if="canManage" link @click="openEdit((row as DietFoodListItem).id)">编辑</el-button>
          <el-button v-if="canManage" link type="danger" @click="remove(row as DietFoodListItem)">删除</el-button>
        </template>
      </el-table-column>
    </DataTable>

    <!-- 详情：27 项全量 -->
    <el-dialog v-model="detailVisible" title="食物详情" width="720px" destroy-on-close>
      <div v-loading="detailLoading" class="min-h-24">
        <el-alert v-if="detailError" type="error" :closable="false" show-icon :title="detailError" />
        <template v-else-if="detail">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="食物名称">{{ detail.name }}</el-descriptions-item>
            <el-descriptions-item label="拼音码">{{ detail.spell_code || '—' }}</el-descriptions-item>
            <el-descriptions-item label="食物分类">{{ detail.category_name || '—' }}</el-descriptions-item>
            <el-descriptions-item label="来源">{{ sourceLabel(detail.source) }}</el-descriptions-item>
            <el-descriptions-item label="食部">
              {{ detail.edible_percent == null ? '—' : `${detail.edible_percent}%` }}
            </el-descriptions-item>
            <el-descriptions-item label="膳食构成">
              {{ detail.group16 ? GROUP16_LABELS[detail.group16] : '—' }}
            </el-descriptions-item>
            <el-descriptions-item label="采购分类">{{ detail.group12 ?? '—' }}</el-descriptions-item>
            <el-descriptions-item label="食物属性">
              {{ detail.source_flags?.length ? detail.source_flags.map((f: DietSourceFlag) => SOURCE_FLAG_LABELS[f]).join('、') : '—' }}
            </el-descriptions-item>
          </el-descriptions>

          <div class="text-text-muted text-xs mt-4 mb-2">营养（每 100g 可食部）</div>
          <div class="grid grid-cols-3 gap-x-4 gap-y-2">
            <div v-for="n in ALL_NUTRIENTS" :key="n.key" class="flex justify-between border-b border-border-hairline py-1">
              <span class="text-text-muted text-sm">{{ n.label }}<span v-if="n.unit">（{{ n.unit }}）</span></span>
              <span class="text-text-primary text-sm">{{ formatNutrient(detail.nutrients[n.key]) }}</span>
            </div>
          </div>
        </template>
      </div>
    </el-dialog>

    <!-- 新增 / 编辑 -->
    <el-dialog
      v-model="editorVisible"
      :title="editingId ? '编辑食物' : '新增食物'"
      width="900px"
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
        <el-scrollbar max-height="62vh">
          <el-form v-if="editorReady" :model="foodForm" label-width="100px" class="pr-4">
            <el-form-item label="食物名称" required>
              <el-input v-model="foodForm.name" maxlength="40" placeholder="请输入食物名称" />
            </el-form-item>
            <el-form-item label="拼音码">
              <span class="text-text-muted text-sm">
                {{ foodForm.spell_code || '保存后由系统按名称生成，无需填写' }}
              </span>
            </el-form-item>
            <el-form-item label="来源">
              <span class="text-text-primary">{{ sourceLabel(foodForm.source) }}</span>
            </el-form-item>
            <el-form-item label="食物分类" required>
              <el-cascader
                v-model="foodForm.category_id"
                :options="categoryOptions"
                :props="formCategoryProps"
                placeholder="请选择二级分类"
                clearable
                class="!w-full"
              />
            </el-form-item>
            <el-form-item label="食部(%)" required>
              <el-input-number
                v-model="foodForm.edible_percent"
                :min="0"
                :max="100"
                :controls="false"
                class="!w-40"
              />
            </el-form-item>

            <el-divider content-position="left">营养（每 100g 可食部，留空表示缺测）</el-divider>
            <div v-for="group in NUTRIENT_GROUPS" :key="group.title" class="mb-4">
              <div class="font-medium text-text-body mb-2">{{ group.title }}</div>
              <div class="grid grid-cols-3 gap-3">
                <div v-for="n in group.items" :key="n.key">
                  <div class="text-text-muted text-xs mb-1">
                    {{ n.label }}<span v-if="n.unit">（{{ n.unit }}）</span>
                  </div>
                  <el-input-number
                    v-model="nutrientForm[n.key]"
                    :min="0"
                    :controls="false"
                    placeholder="—"
                    class="!w-full"
                  />
                </div>
              </div>
            </div>
            <div class="text-text-muted text-xs">
              空值会保存为 NULL，不会写成 0；热量留空且能量有值时，由服务端按「能量 × 4.186」换算。
            </div>
          </el-form>
        </el-scrollbar>
      </div>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="editorSaving" :disabled="!editorReady" @click="submitFood">
          保存
        </el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type {
  DietApiError,
  DietFoodCategory,
  DietFoodCoreNutrients,
  DietFoodCreateBody,
  DietFoodDetail,
  DietFoodListItem,
  DietFoodSource,
  DietNutrients,
  DietSourceFlag,
} from '~/types/api'
import {
  batchDeleteDietFoods,
  createDietFood,
  deleteDietFood,
  getDietFood,
  getDietFoodCategories,
  getDietFoods,
  updateDietFood,
} from '~/api/diet'
import {
  ALL_NUTRIENTS,
  FOOD_SOURCE_LABELS,
  GROUP16_LABELS,
  NUTRIENT_GROUPS,
  SOURCE_FLAG_LABELS,
  formatNutrient,
} from '../components/nutrients'

const { can } = usePermission()
const canManage = computed(() => can('diet:manage'))

const { page, pageSize, total, resetPage } = usePagination()
const keyword = ref('')
const categoryId = ref<unknown>(undefined)
const list = ref<DietFoodListItem[]>([])
const loading = ref(false)
const listError = ref('')
const selectedRows = ref<DietFoodListItem[]>([])

const categories = ref<DietFoodCategory[]>([])

const filterCategoryProps = {
  value: 'id',
  label: 'name',
  children: 'children',
  emitPath: false,
  checkStrictly: true,
}
const formCategoryProps = {
  value: 'id',
  label: 'name',
  children: 'children',
  emitPath: false,
}

/** 节点键名必须与上方 props 的 value/label（id/name）保持一致 */
const categoryOptions = computed(() =>
  categories.value.map((c) => ({
    id: c.id,
    name: c.name,
    children: c.children?.length
      ? c.children.map((child) => ({ id: child.id, name: child.name }))
      : undefined,
  })),
)

const LIST_NUTRIENTS: { prop: keyof DietFoodCoreNutrients; label: string; unit: string }[] = [
  { prop: 'energy', label: '能量', unit: '千卡' },
  { prop: 'protein', label: '蛋白质', unit: '克' },
  { prop: 'fats', label: '脂肪', unit: '克' },
  { prop: 'carb', label: '碳水化合物', unit: '克' },
  { prop: 'retinol', label: '视黄醇', unit: '微克' },
  { prop: 'carotin', label: '胡萝卜素', unit: '微克' },
  { prop: 'calcium', label: '钙', unit: '毫克' },
  { prop: 'iron', label: '铁', unit: '毫克' },
  { prop: 'zinc', label: '锌', unit: '毫克' },
  { prop: 'phosphorus', label: '磷', unit: '毫克' },
  { prop: 'nicotinic_acid', label: '尼克酸', unit: '毫克' },
  { prop: 'vitamin_a', label: '维生素A', unit: '微克 RE' },
  { prop: 'vitamin_b1', label: '维生素B1', unit: '毫克' },
  { prop: 'vitamin_b2', label: '维生素B2', unit: '毫克' },
  { prop: 'vitamin_c', label: '维生素C', unit: '毫克' },
]

onMounted(() => {
  loadCategories()
  load()
})

function sourceLabel(source: DietFoodSource) {
  return FOOD_SOURCE_LABELS[source] ?? source
}

/** 列表列动态取值：避免模板里用宽类型索引强类型行 */
function nutrientValue(row: DietFoodListItem, key: keyof DietFoodCoreNutrients) {
  return formatNutrient(row[key])
}

function onSearch() {
  resetPage()
  load()
}

function onSelectionChange(rows: unknown[]) {
  selectedRows.value = rows as DietFoodListItem[]
}

async function load() {
  loading.value = true
  listError.value = ''
  try {
    const res = await getDietFoods({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      category_id: typeof categoryId.value === 'number' ? categoryId.value : undefined,
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

async function loadCategories() {
  try {
    const res = await getDietFoodCategories()
    categories.value = res.list ?? []
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '食物分类加载失败')
  }
}

// ===== 详情 =====

const detailVisible = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const detail = ref<DietFoodDetail | null>(null)

async function openDetail(id: number) {
  detail.value = null
  detailError.value = ''
  detailVisible.value = true
  detailLoading.value = true
  try {
    detail.value = await getDietFood(id)
  } catch (e: unknown) {
    detailError.value = e instanceof Error ? e.message : '详情加载失败'
  } finally {
    detailLoading.value = false
  }
}

// ===== 新增 / 编辑 =====

const editorVisible = ref(false)
const editorLoading = ref(false)
const editorSaving = ref(false)
const editorReady = ref(false)
const editorError = ref('')
const editingId = ref<number | null>(null)

const foodForm = reactive({
  name: '',
  /** el-cascader 的 emitPath=false 运行时给 number；本地按 unknown 收窄 */
  category_id: undefined as unknown,
  edible_percent: null as number | null | undefined,
  spell_code: '',
  source: 'custom' as DietFoodSource,
})

function emptyNutrients(): Record<keyof DietNutrients, number | null | undefined> {
  const out = {} as Record<keyof DietNutrients, number | null | undefined>
  for (const n of ALL_NUTRIENTS) out[n.key] = null
  return out
}

const nutrientForm = reactive<Record<keyof DietNutrients, number | null | undefined>>(emptyNutrients())

function resetEditor() {
  editingId.value = null
  editorError.value = ''
  editorReady.value = false
  editorLoading.value = false
  Object.assign(foodForm, {
    name: '',
    category_id: undefined,
    edible_percent: null,
    spell_code: '',
    source: 'custom',
  })
  Object.assign(nutrientForm, emptyNutrients())
}

function openCreate() {
  resetEditor()
  foodForm.edible_percent = 100
  editorReady.value = true
  editorVisible.value = true
}

async function openEdit(id: number) {
  resetEditor()
  editingId.value = id
  editorVisible.value = true
  editorLoading.value = true
  try {
    const res = await getDietFood(id)
    Object.assign(foodForm, {
      name: res.name,
      category_id: res.category_id,
      edible_percent: res.edible_percent,
      spell_code: res.spell_code,
      source: res.source,
    })
    Object.assign(nutrientForm, emptyNutrients(), res.nutrients)
    editorReady.value = true
  } catch (e: unknown) {
    editorError.value = e instanceof Error ? e.message : '食物详情加载失败'
  } finally {
    editorLoading.value = false
  }
}

/** 只提交有值的键：未提交的键由服务端写 NULL，绝不能把空值转成 0 */
function nutrientPayload(): Partial<DietNutrients> {
  const payload: Partial<DietNutrients> = {}
  for (const n of ALL_NUTRIENTS) {
    const value = nutrientForm[n.key]
    if (typeof value === 'number' && Number.isFinite(value)) payload[n.key] = value
  }
  return payload
}

async function submitFood() {
  editorError.value = ''
  // 关闭弹窗后 @closed 会清空 editingId，保存前先捕获目标 id
  const targetId = editingId.value
  const name = foodForm.name.trim()
  if (!name) {
    editorError.value = '请输入食物名称'
    return
  }
  const categoryId = foodForm.category_id
  if (typeof categoryId !== 'number') {
    editorError.value = '请选择食物分类'
    return
  }
  if (foodForm.edible_percent == null) {
    editorError.value = '请输入食部'
    return
  }

  const body: DietFoodCreateBody = {
    name,
    category_id: categoryId,
    edible_percent: foodForm.edible_percent,
    nutrients: nutrientPayload(),
  }

  editorSaving.value = true
  try {
    if (targetId) {
      await updateDietFood(targetId, body)
    } else {
      await createDietFood(body)
    }
    ElMessage.success('已保存')
    editorVisible.value = false
    await load()
  } catch (e: unknown) {
    const err = e as DietApiError
    editorError.value = err.message || '保存失败'
    if (err.status !== 422) ElMessage.error(editorError.value)
  } finally {
    editorSaving.value = false
  }
}

// ===== 删除 =====

async function remove(row: DietFoodListItem) {
  try {
    await ElMessageBox.confirm(`确定删除食物「${row.name}」？`, '提示', { type: 'warning' })
    await deleteDietFood(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e: unknown) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e instanceof Error ? e.message : '删除失败')
    }
  }
}

async function batchRemove() {
  if (selectedRows.value.length === 0) return
  try {
    await ElMessageBox.confirm(
      `确定删除选中的 ${selectedRows.value.length} 条食物？被菜谱引用的食物会自动跳过。`,
      '批量删除',
      { type: 'warning' },
    )
    const res = await batchDeleteDietFoods({ ids: selectedRows.value.map((r) => r.id) })
    ElMessage.success(`已删除 ${res.deleted} 条`)
    if (res.skipped?.length) {
      const detail = res.skipped
        .map((s) => `${s.name}（${s.reason === 'referenced_by_recipe' ? '仍被菜谱引用' : '不存在或跨园所'}）`)
        .join('、')
      ElMessage.warning(`跳过 ${res.skipped.length} 条：${detail}`)
    }
    selectedRows.value = []
    await load()
  } catch (e: unknown) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e instanceof Error ? e.message : '批量删除失败')
    }
  }
}
</script>
