<template>
  <div class="space-y-6">
    <el-card shadow="never">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <span>基本参数设置</span>
          <el-button v-if="canManage" type="primary" data-testid="group-object-new" @click="openCreate">
            添加
          </el-button>
        </div>
      </template>

      <el-alert
        v-if="yearError"
        type="warning"
        :closable="false"
        show-icon
        title="学年列表加载失败"
        :description="yearError"
        class="mb-4"
      >
        <el-button size="small" @click="loadYears">重试</el-button>
      </el-alert>

      <SearchBar
        v-model:keyword="keyword"
        keyword-placeholder="集体对象名称"
        @search="onSearch"
        @reset="onSearch"
      >
        <div data-testid="group-object-year-filter">
          <el-select
            v-model="academicYearId"
            placeholder="学年"
            clearable
            class="!w-48"
            @change="onSearch"
          >
            <el-option
              v-for="year in years"
              :key="year.id"
              :label="`${year.year}${year.is_current ? '（当前学年）' : ''}`"
              :value="year.id"
            />
          </el-select>
        </div>
      </SearchBar>

      <el-alert
        v-if="listError"
        type="error"
        :closable="false"
        show-icon
        title="集体对象加载失败"
        :description="listError"
        class="mb-4"
        data-testid="group-object-list-error"
      >
        <el-button size="small" @click="load">重试</el-button>
      </el-alert>

      <DataTable
        v-model:page="page"
        v-model:page-size="pageSize"
        :data="list as unknown as Record<string, unknown>[]"
        :loading="loading"
        :total="total"
        empty-text="暂无集体对象"
        @page-change="load"
        @size-change="load"
      >
        <el-table-column prop="name" label="集体对象名称" min-width="130" fixed="left" />
        <el-table-column label="托管形式" width="90">
          <template #default="{ row }">
            {{ (row as DietGroupObjectListItem).care_type_label || '—' }}
          </template>
        </el-table-column>
        <el-table-column label="就餐对象" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">{{ classNames(row as DietGroupObjectListItem) }}</template>
        </el-table-column>
        <el-table-column label="餐次" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ mealNames(row as DietGroupObjectListItem) }}</template>
        </el-table-column>
        <el-table-column label="食物重量输入方式" width="130">
          <template #default="{ row }">
            {{ (row as DietGroupObjectListItem).food_input_type_label || '—' }}
          </template>
        </el-table-column>
        <el-table-column label="膳食营养素参考摄入量标准" width="170">
          <template #default="{ row }">{{ (row as DietGroupObjectListItem).policy_name || '—' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="190" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="canManage"
              link
              type="primary"
              :data-testid="`group-object-edit-${(row as DietGroupObjectListItem).id}`"
              @click="openEdit(row as DietGroupObjectListItem)"
            >
              修改
            </el-button>
            <el-button
              link
              type="primary"
              :data-testid="`group-object-detail-${(row as DietGroupObjectListItem).id}`"
              @click="openDetail((row as DietGroupObjectListItem).id)"
            >
              详情
            </el-button>
            <el-button
              v-if="canManage"
              link
              type="primary"
              :data-testid="`group-object-plan-${(row as DietGroupObjectListItem).id}`"
              @click="goCreatePlan(row as DietGroupObjectListItem)"
            >
              制定食谱
            </el-button>
          </template>
        </el-table-column>
      </DataTable>
    </el-card>

    <!-- 添加 / 修改：同一个对话框，修改时学年只读 -->
    <el-dialog
      v-model="editorVisible"
      :title="editingId == null ? '添加集体对象' : '修改集体对象'"
      width="880px"
      destroy-on-close
      @closed="resetEditor"
    >
      <el-alert
        v-if="formError"
        type="error"
        :closable="false"
        show-icon
        :title="formError"
        class="mb-4"
        data-testid="group-object-form-error"
      />

      <div v-loading="editorLoading" class="min-h-24">
        <el-form label-width="180px">
          <el-form-item label="集体对象名称" required>
            <div class="w-full max-w-md" data-testid="group-object-name">
              <el-input v-model="form.name" maxlength="20" show-word-limit placeholder="不超过20个字符" />
            </div>
          </el-form-item>

          <el-form-item label="学年" required>
            <div class="w-56" data-testid="group-object-year">
              <el-select
                v-if="editingId == null"
                v-model="form.academic_year_id"
                placeholder="请选择学年"
                class="w-full"
                @change="onEditorYearChange"
              >
                <el-option v-for="year in years" :key="year.id" :label="year.year" :value="year.id" />
              </el-select>
              <span v-else class="text-text-primary">{{ selectedYearLabel || '—' }}</span>
            </div>
            <span v-if="editingId != null" class="ml-3 text-xs text-text-muted">学年不可修改</span>
          </el-form-item>

          <el-form-item label="托管形式" required>
            <el-radio-group v-model="form.care_type">
              <el-radio value="day_care">日托</el-radio>
              <el-radio value="full_care">全托</el-radio>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="就餐班级" required>
            <div v-loading="classLoading" class="w-full min-h-12">
              <el-alert
                v-if="classError"
                type="error"
                :closable="false"
                show-icon
                title="班级列表加载失败"
                :description="classError"
              >
                <el-button size="small" @click="loadClasses">重试</el-button>
              </el-alert>
              <div
                v-else-if="!classLoading && gradeGroups.length === 0"
                class="text-sm text-text-muted"
                data-testid="group-object-class-empty"
              >
                所选学年暂无班级，请先在「班级管理」创建班级后再添加集体对象。
              </div>
              <div v-else class="w-full space-y-3">
                <div
                  v-for="group in gradeGroups"
                  :key="group.grade"
                  class="rounded-md border border-border-hairline p-3"
                >
                  <div class="mb-2 flex items-center gap-2">
                    <el-checkbox
                      :model-value="groupState(group).all"
                      :indeterminate="groupState(group).indeterminate"
                      @change="(checked: string | number | boolean) => toggleGrade(group, Boolean(checked))"
                    >
                      <span class="font-medium text-text-primary">{{ group.grade }}</span>
                    </el-checkbox>
                    <span class="text-xs text-text-muted">
                      {{ groupState(group).selected }}/{{ group.classes.length }}
                    </span>
                  </div>
                  <el-checkbox-group v-model="form.class_ids">
                    <el-checkbox
                      v-for="item in group.classes"
                      :key="item.id"
                      :value="item.id"
                      :data-testid="`group-object-class-${item.id}`"
                    >
                      {{ item.name }}
                    </el-checkbox>
                  </el-checkbox-group>
                </div>
                <div class="text-xs text-text-muted">已选 {{ form.class_ids.length }} 个班级</div>
              </div>
            </div>
          </el-form-item>

          <el-form-item label="餐次设置" required>
            <div class="w-full">
              <el-table :data="form.meals" row-key="meal_key" border class="w-full">
                <el-table-column prop="name" label="餐次" width="90" />
                <el-table-column label="就餐时间" width="170">
                  <template #default="{ row }">
                    <el-time-picker
                      v-model="(row as MealFormItem).time"
                      format="HH:mm"
                      value-format="HH:mm"
                      :clearable="false"
                      class="!w-32"
                    />
                  </template>
                </el-table-column>
                <el-table-column label="供能占比(%)" width="150">
                  <template #default="{ row }">
                    <el-input-number
                      v-model="(row as MealFormItem).energy_percent"
                      :min="0"
                      :max="100"
                      :controls="false"
                      :disabled="isStandard"
                      class="!w-24"
                    />
                  </template>
                </el-table-column>
                <el-table-column label="启用" width="80">
                  <template #default="{ row }">
                    <el-checkbox v-model="(row as MealFormItem).enabled" />
                  </template>
                </el-table-column>
              </el-table>
              <div class="mt-2 text-xs leading-5 text-text-muted">
                {{
                  isStandard
                    ? '标准比例固定为 早餐 25%、早点 5%、午餐 35%、午点 5%、晚餐 25%、晚点 5%，不可编辑。'
                    : customRatioHint
                }}
              </div>
            </div>
          </el-form-item>

          <el-form-item label="各餐热量及营养标准分配比">
            <el-radio-group v-model="form.ratio_mode" @change="onRatioModeChange">
              <el-radio value="standard">标准比例</el-radio>
              <el-radio value="custom">自定比例</el-radio>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="食物重量输入方式">
            <span class="text-text-primary">市品重量</span>
            <span class="ml-2 text-xs text-text-muted">本版本固定按市品重量录入，不可切换</span>
          </el-form-item>

          <el-form-item label="膳食营养素参考摄入量标准">
            <span class="text-text-primary">DRIs2000</span>
            <span class="ml-2 text-xs text-text-muted">固定标准，不可切换</span>
          </el-form-item>

          <el-form-item label="折算系数">
            <span class="font-medium text-text-primary">{{ formatCoefficient(coefficient) }}</span>
            <span class="ml-2 text-xs text-text-muted">= 已启用餐次供能占比合计 ÷ 100（只读）</span>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button
          type="primary"
          data-testid="group-object-submit"
          :loading="saving"
          :disabled="editorLoading || editorLoadFailed || gradeGroups.length === 0"
          @click="save"
        >
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 详情：全部字段 -->
    <el-drawer v-model="detailVisible" title="集体对象详情" size="640px" destroy-on-close>
      <div v-loading="detailLoading" class="min-h-24">
        <el-alert v-if="detailError" type="error" :closable="false" show-icon :title="detailError">
          <el-button size="small" @click="reloadDetail">重试</el-button>
        </el-alert>
        <template v-else-if="detail">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="集体对象名称">{{ detail.name }}</el-descriptions-item>
            <el-descriptions-item label="学年">
              {{ yearLabel(detail.academic_year_id) }}
            </el-descriptions-item>
            <el-descriptions-item label="托管形式">{{ detail.care_type_label || '—' }}</el-descriptions-item>
            <el-descriptions-item label="就餐班级">{{ classNames(detail) }}</el-descriptions-item>
            <el-descriptions-item label="餐次">{{ mealNames(detail) }}</el-descriptions-item>
            <el-descriptions-item label="比例模式">
              {{ detail.ratio_mode_label || '—' }}
            </el-descriptions-item>
            <el-descriptions-item label="食物重量输入方式">
              {{ detail.food_input_type_label || '—' }}
            </el-descriptions-item>
            <el-descriptions-item label="膳食营养素参考摄入量标准">
              {{ detail.policy_name || '—' }}
            </el-descriptions-item>
            <el-descriptions-item label="折算系数">
              {{ formatCoefficient(detail.coefficient) }}
            </el-descriptions-item>
          </el-descriptions>

          <div class="mt-4 mb-2 text-xs text-text-muted">餐次设置（含未启用餐次）</div>
          <el-table :data="detail.meals" row-key="meal_key" border class="w-full">
            <el-table-column prop="name" label="餐次" width="90" />
            <el-table-column prop="time" label="就餐时间" width="110" />
            <el-table-column label="供能占比(%)" width="120">
              <template #default="{ row }">{{ (row as DietMealConfigItem).energy_percent }}</template>
            </el-table-column>
            <el-table-column label="启用" width="80">
              <template #default="{ row }">
                {{ (row as DietMealConfigItem).enabled ? '是' : '否' }}
              </template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import type {
  DietApiError,
  DietAcademicYearOption,
  DietCareType,
  DietClassOption,
  DietGroupObjectDetail,
  DietGroupObjectInput,
  DietGroupObjectListItem,
  DietMealConfigInput,
  DietMealConfigItem,
  DietMealKey,
  DietRatioMode,
} from '~/types/api'
import {
  createDietGroupObject,
  getDietAcademicYears,
  getDietClasses,
  getDietGroupObject,
  getDietGroupObjects,
  updateDietGroupObject,
} from '~/api/diet'

/**
 * 契约默认值：餐次名称、顺序与标准占比由服务端固定写入（23-diet.md §一），
 * 前端只在新建时给出同值初值，避免空表。
 */
const MEAL_DEFAULTS: { key: DietMealKey; name: string; time: string; percent: number; enabled: boolean }[] = [
  { key: 'breakfast', name: '早餐', time: '07:00', percent: 25, enabled: true },
  { key: 'morning_snack', name: '早点', time: '09:00', percent: 5, enabled: true },
  { key: 'lunch', name: '午餐', time: '11:00', percent: 35, enabled: true },
  { key: 'afternoon_snack', name: '午点', time: '14:30', percent: 5, enabled: true },
  { key: 'dinner', name: '晚餐', time: '17:00', percent: 25, enabled: true },
  { key: 'evening_snack', name: '晚点', time: '20:00', percent: 5, enabled: false },
]

const GRADE_ORDER = ['托班', '小班', '中班', '大班']

interface MealFormItem {
  meal_key: DietMealKey
  name: string
  time: string
  /** el-input-number 清空时给 null，提交前统一校验 */
  energy_percent: number | null | undefined
  sort: number
  enabled: boolean
}

interface EditorForm {
  academic_year_id: number | undefined
  name: string
  care_type: DietCareType
  ratio_mode: DietRatioMode
  class_ids: number[]
  meals: MealFormItem[]
}

const { can } = usePermission()
const canManage = computed(() => can('diet:manage'))
const router = useRouter()

const { page, pageSize, total, resetPage } = usePagination()
const keyword = ref('')
const academicYearId = ref<number | undefined>(undefined)
const years = ref<DietAcademicYearOption[]>([])
const yearError = ref('')
const list = ref<DietGroupObjectListItem[]>([])
const loading = ref(false)
const listError = ref('')

const editorVisible = ref(false)
const editorLoading = ref(false)
/** 修改时详情没取到就不允许保存：否则会用默认餐次覆盖原对象 */
const editorLoadFailed = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)
const formError = ref('')
const form = reactive<EditorForm>(blankForm())

const classOptions = ref<DietClassOption[]>([])
const classLoading = ref(false)
const classError = ref('')

const detailVisible = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const detail = ref<DietGroupObjectDetail | null>(null)
let detailId: number | null = null

const isStandard = computed(() => form.ratio_mode === 'standard')

const coefficient = computed(() => {
  const sum = form.meals
    .filter((meal) => meal.enabled)
    .reduce((acc, meal) => acc + (Number(meal.energy_percent) || 0), 0)
  return sum / 100
})

const customRatioHint = computed(() => {
  // 相邻餐次共享上限（契约 §一 校验 13）；日托的早餐+早点比全托少 1
  const firstPair = form.care_type === 'full_care' ? 30 : 29
  return `自定比例下相邻餐次共享上限：早餐+早点 ≤ ${firstPair}%、午餐+午点 ≤ 39%、晚餐+晚点 ≤ 30%`
})

const selectedYearLabel = computed(
  () => years.value.find((year) => year.id === form.academic_year_id)?.year ?? '',
)

/** 班级由服务端按学年过滤（DGO-S7），不再拉全量后前端筛选 */
const gradeGroups = computed(() => {
  const groups = new Map<string, DietClassOption[]>()
  for (const item of classOptions.value) {
    const grade = item.grade || '未分年级'
    const bucket = groups.get(grade)
    if (bucket) bucket.push(item)
    else groups.set(grade, [item])
  }
  return [...groups.entries()]
    .map(([grade, classes]) => ({
      grade,
      classes: classes.slice().sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN')),
    }))
    .sort((a, b) => {
      const ai = GRADE_ORDER.indexOf(a.grade)
      const bi = GRADE_ORDER.indexOf(b.grade)
      return (ai === -1 ? GRADE_ORDER.length : ai) - (bi === -1 ? GRADE_ORDER.length : bi)
    })
})

onMounted(async () => {
  await loadYears()
  await load()
})

function blankForm(): EditorForm {
  return {
    academic_year_id: undefined,
    name: '',
    care_type: 'day_care',
    ratio_mode: 'standard',
    class_ids: [],
    meals: MEAL_DEFAULTS.map((meal, index) => ({
      meal_key: meal.key,
      name: meal.name,
      time: meal.time,
      energy_percent: meal.percent,
      sort: index + 1,
      enabled: meal.enabled,
    })),
  }
}

function formatCoefficient(value: number | null | undefined) {
  if (value == null || !Number.isFinite(value)) return '—'
  return String(Number(value.toFixed(4)))
}

function yearLabel(id: number) {
  return years.value.find((year) => year.id === id)?.year ?? `#${id}`
}

function classNames(row: { classes?: { name: string }[] }) {
  const names = (row.classes ?? []).map((item) => item.name).filter(Boolean)
  return names.length ? names.join('、') : '—'
}

function mealNames(row: { meal_names?: string[] }) {
  const names = row.meal_names ?? []
  return names.length ? names.join('/') : '—'
}

async function loadYears() {
  yearError.value = ''
  try {
    const res = await getDietAcademicYears()
    years.value = res.list ?? []
    if (academicYearId.value == null) {
      const current = years.value.find((year) => year.is_current) ?? years.value[0]
      academicYearId.value = current?.id
    }
  } catch (e: unknown) {
    yearError.value = e instanceof Error ? e.message : '网络错误'
  }
}

function onSearch() {
  resetPage()
  load()
}

async function load() {
  loading.value = true
  listError.value = ''
  try {
    const res = await getDietGroupObjects({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      academic_year_id: academicYearId.value,
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

// ===== 添加 / 修改 =====

function resetEditor() {
  Object.assign(form, blankForm())
  editingId.value = null
  formError.value = ''
  editorLoading.value = false
  editorLoadFailed.value = false
  saving.value = false
  classOptions.value = []
  classError.value = ''
  classLoading.value = false
  classRequestSeq += 1
}

function openCreate() {
  resetEditor()
  form.academic_year_id =
    academicYearId.value ??
    years.value.find((year) => year.is_current)?.id ??
    years.value[0]?.id
  editorVisible.value = true
  void loadClasses()
}

async function openEdit(row: DietGroupObjectListItem) {
  resetEditor()
  editingId.value = row.id
  form.academic_year_id = row.academic_year_id
  form.name = row.name
  form.care_type = row.care_type
  editorVisible.value = true
  void loadClasses()
  editorLoading.value = true
  try {
    const res = await getDietGroupObject(row.id)
    // 对话框可能在请求返回前被关闭并换成另一个对象
    if (editingId.value === row.id) applyDetail(res)
  } catch (e: unknown) {
    if (editingId.value !== row.id) return
    const err = e as DietApiError
    editorLoadFailed.value = true
    formError.value = err.message || '集体对象加载失败'
  } finally {
    if (editingId.value === row.id) editorLoading.value = false
  }
}

function applyDetail(res: DietGroupObjectDetail) {
  editorLoadFailed.value = false
  form.name = res.name
  form.academic_year_id = res.academic_year_id
  form.care_type = res.care_type
  form.ratio_mode = res.ratio_mode
  form.class_ids = (res.classes ?? []).map((item) => item.id)
  form.meals = (res.meals ?? []).map((meal) => ({
    meal_key: meal.meal_key,
    name: meal.name,
    time: meal.time,
    energy_percent: meal.energy_percent,
    sort: meal.sort,
    enabled: meal.enabled,
  }))
}

/** 班级选项是动态数据：每次打开对话框重新拉取，新增班级后重开即可见 */
let classRequestSeq = 0
async function loadClasses() {
  const seq = ++classRequestSeq
  classLoading.value = true
  classError.value = ''
  try {
    const res = await getDietClasses(selectedYearLabel.value || undefined)
    if (seq !== classRequestSeq) return
    classOptions.value = res.list ?? []
  } catch (e: unknown) {
    if (seq !== classRequestSeq) return
    classOptions.value = []
    classError.value = e instanceof Error ? e.message : '网络错误'
  } finally {
    if (seq === classRequestSeq) classLoading.value = false
  }
}

function onEditorYearChange() {
  // 换学年后已选班级属于别的学年，全部清空并按新学年重新拉取
  form.class_ids = []
  void loadClasses()
}

function groupState(group: { grade: string; classes: DietClassOption[] }) {
  const selected = group.classes.filter((item) => form.class_ids.includes(item.id)).length
  return {
    selected,
    all: group.classes.length > 0 && selected === group.classes.length,
    indeterminate: selected > 0 && selected < group.classes.length,
  }
}

function toggleGrade(group: { grade: string; classes: DietClassOption[] }, checked: boolean) {
  const ids = group.classes.map((item) => item.id)
  const rest = form.class_ids.filter((id) => !ids.includes(id))
  form.class_ids = checked ? [...rest, ...ids] : rest
}

/** 切回标准模式时占比必须回到固定值，否则提交会被 422 拒绝 */
function onRatioModeChange(mode: string | number | boolean | undefined) {
  if (mode !== 'standard') return
  for (const meal of form.meals) {
    const fixed = MEAL_DEFAULTS.find((item) => item.key === meal.meal_key)
    if (fixed) meal.energy_percent = fixed.percent
  }
}

async function save() {
  if (saving.value || editorLoadFailed.value) return
  formError.value = ''

  const name = form.name.trim()
  if (!name) {
    formError.value = '请设置集体对象名称'
    ElMessage.error(formError.value)
    return
  }
  if (name.length > 20) {
    formError.value = '集体对象名称不能超过 20 个字符'
    ElMessage.error(formError.value)
    return
  }
  if (editingId.value == null && typeof form.academic_year_id !== 'number') {
    formError.value = '请选择学年'
    ElMessage.error(formError.value)
    return
  }
  if (gradeGroups.value.length === 0) {
    formError.value = '所选学年暂无班级，无法保存'
    ElMessage.error(formError.value)
    return
  }
  if (form.class_ids.length === 0) {
    formError.value = '请至少选择一个就餐班级'
    ElMessage.error(formError.value)
    return
  }

  const meals: DietMealConfigInput[] = []
  for (const meal of form.meals) {
    if (meal.energy_percent == null || !Number.isFinite(meal.energy_percent)) {
      formError.value = `「${meal.name}」供能占比不能为空`
      ElMessage.error(formError.value)
      return
    }
    meals.push({
      meal_key: meal.meal_key,
      time: meal.time,
      energy_percent: meal.energy_percent,
      enabled: meal.enabled,
    })
  }

  const body: DietGroupObjectInput = {
    academic_year_id: Number(form.academic_year_id),
    name,
    care_type: form.care_type,
    food_input_type: 'market_weight',
    ratio_mode: form.ratio_mode,
    class_ids: form.class_ids,
    meals,
  }

  saving.value = true
  try {
    if (editingId.value == null) {
      await createDietGroupObject(body)
      // 新对象可能建在别的学年：把筛选切过去，否则刚保存的行不在当前列表里
      academicYearId.value = body.academic_year_id
    } else {
      await updateDietGroupObject(editingId.value, body)
    }
    ElMessage.success('已保存')
    editorVisible.value = false
    await load()
  } catch (e: unknown) {
    const err = e as DietApiError
    // 422 的业务文案由服务端给出，原样展示（契约 §一 校验）
    formError.value = err.message || '保存失败'
    ElMessage.error(formError.value)
  } finally {
    saving.value = false
  }
}

// ===== 详情 =====

async function openDetail(id: number) {
  detailId = id
  detailVisible.value = true
  await reloadDetail()
}

async function reloadDetail() {
  if (detailId == null) return
  const target = detailId
  detailLoading.value = true
  detailError.value = ''
  detail.value = null
  try {
    const res = await getDietGroupObject(target)
    if (detailId === target) detail.value = res
  } catch (e: unknown) {
    if (detailId !== target) return
    const err = e as DietApiError
    detailError.value = err.status === 404 ? '集体对象不存在或已被删除' : err.message || '加载失败'
  } finally {
    if (detailId === target) detailLoading.value = false
  }
}

// ===== 制定食谱 =====

function goCreatePlan(row: DietGroupObjectListItem) {
  router.push({
    path: '/diet/plans',
    query: { create: '1', group_object_id: String(row.id) },
  })
}
</script>
