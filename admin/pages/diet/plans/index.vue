<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span>膳食计划</span>
        <el-button
          v-if="canManage"
          type="primary"
          data-testid="plan-new"
          :disabled="years.length === 0"
          @click="openWizard"
        >
          新建周食谱
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
      keyword-placeholder="食谱名称"
      @search="onSearch"
      @reset="onSearch"
    >
      <div data-testid="plan-year-filter">
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
      title="周食谱列表加载失败"
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
      empty-text="暂无周食谱"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column label="起止日期" min-width="220">
        <template #default="{ row }">
          {{ (row as DietPlanListItem).start_date }} 至 {{ (row as DietPlanListItem).end_date }}
        </template>
      </el-table-column>
      <el-table-column prop="day_num" label="天数" width="80" />
      <el-table-column label="周次" width="80">
        <template #default="{ row }">
          {{ (row as DietPlanListItem).week_of_year || '—' }}
        </template>
      </el-table-column>
      <el-table-column label="创建方式" width="110">
        <template #default="{ row }">
          <el-tag size="small" type="info">
            {{ (row as DietPlanListItem).created_method_label }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEditor((row as DietPlanListItem).id)">
            编辑
          </el-button>
          <el-button
            v-if="canManage"
            link
            type="danger"
            @click="remove(row as DietPlanListItem)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </DataTable>

    <!-- 新建向导：三种方式三选一 -->
    <el-dialog
      v-model="wizardVisible"
      title="新建周食谱"
      width="720px"
      destroy-on-close
      @closed="resetWizard"
    >
      <el-alert
        v-if="createError"
        type="error"
        :closable="false"
        show-icon
        title="创建失败"
        :description="createError"
        class="mb-4"
        data-testid="plan-create-error"
      />

      <el-form :model="wizard" label-width="100px">
        <el-form-item label="学年" required>
          <div class="w-full" data-testid="plan-year">
            <el-select
              v-model="wizard.academic_year_id"
              placeholder="请选择学年"
              class="w-full"
              @change="onWizardYearChange"
            >
              <el-option
                v-for="year in years"
                :key="year.id"
                :label="year.year"
                :value="year.id"
              />
            </el-select>
          </div>
        </el-form-item>

        <el-form-item label="食谱名称" required>
          <div class="w-full" data-testid="plan-name">
            <el-input v-model="wizard.name" maxlength="128" placeholder="如：第九周" />
          </div>
        </el-form-item>

        <el-form-item label="开始日期" required>
          <div data-testid="plan-start">
            <el-date-picker
              v-model="wizard.start_date"
              type="date"
              value-format="YYYY-MM-DD"
              format="YYYY-MM-DD"
              :clearable="false"
              :disabled-date="notMonday"
              class="!w-52"
            />
          </div>
          <span class="ml-3 text-xs text-text-muted">仅可选周一</span>
        </el-form-item>

        <el-form-item label="结束日期">
          <span class="text-text-primary">{{ wizardEndDate }}</span>
          <span class="ml-2 text-xs text-text-muted">= 开始日期 + 6 天（整周 7 天）</span>
        </el-form-item>

        <el-form-item label="周次">
          <div data-testid="plan-week">
            <el-input-number
              v-model="wizard.week_of_year"
              :min="0"
              :max="53"
              :controls="false"
              placeholder="如 9"
              class="!w-32"
            />
          </div>
          <span class="ml-3 text-xs text-text-muted">仅展示，不参与计算</span>
        </el-form-item>

        <el-form-item label="创建方式" required>
          <el-radio-group v-model="wizard.created_method" @change="onMethodChange">
            <el-radio value="custom" data-testid="plan-method-custom">自定义</el-radio>
            <el-radio value="template" data-testid="plan-method-template">选用模板</el-radio>
            <el-radio value="copy" data-testid="plan-method-copy">复制历史</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="wizard.created_method === 'template'" label="选择模板" required>
          <div class="w-full" data-testid="plan-template-select">
            <el-select
              v-model="wizard.template_id"
              placeholder="请选择模板"
              class="w-full"
              :loading="templatesLoading"
            >
              <el-option
                v-for="template in templates"
                :key="template.id"
                :label="`${template.name}（${template.day_num} 天 · ${template.item_count} 项）`"
                :value="template.id"
              />
            </el-select>
          </div>
          <div v-if="templateError" class="text-xs text-semantic-danger">{{ templateError }}</div>
          <div
            v-else-if="!templatesLoading && templates.length === 0"
            class="text-xs text-semantic-warning"
          >
            当前园所暂无模板，请改用自定义或复制历史。
          </div>
        </el-form-item>

        <el-form-item v-else-if="wizard.created_method === 'copy'" label="源食谱" required>
          <div class="w-full" data-testid="plan-copy-select">
            <el-select
              v-model="wizard.source_plan_id"
              filterable
              remote
              :remote-method="searchCopyPlans"
              :loading="copyLoading"
              placeholder="搜索同园历史周食谱"
              class="w-full"
              @focus="() => searchCopyPlans('')"
            >
              <el-option
                v-for="plan in copyPlans"
                :key="plan.id"
                :label="`${plan.name}（${plan.start_date} 至 ${plan.end_date}）`"
                :value="plan.id"
              />
            </el-select>
          </div>
          <div v-if="copyError" class="text-xs text-semantic-danger">{{ copyError }}</div>
          <div
            v-else-if="!copyLoading && copyPlans.length === 0"
            class="text-xs text-text-muted"
          >
            该学年暂无其他周食谱可复制。
          </div>
        </el-form-item>
      </el-form>

      <div class="text-xs leading-5 text-text-muted">
        选用模板 / 复制历史会把来源条目复制到新周期；缺失的食物会跳过并在创建完成后提示。
      </div>

      <template #footer>
        <el-button @click="wizardVisible = false">取消</el-button>
        <el-button
          type="primary"
          data-testid="plan-submit"
          :loading="creating"
          @click="submitWizard"
        >
          创建
        </el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type {
  DietApiError,
  DietCreatedMethod,
  DietPlanCreateBody,
  DietPlanListItem,
  DietTemplateListItem,
} from '~/types/api'
import { createDietPlan, deleteDietPlan, getDietPlans, getDietTemplates } from '~/api/diet'
import { getAcademicYears } from '~/api/settings'
import { addDaysISO, isMondayISO, nextMondayISO } from './components/plan-utils'

const { can } = usePermission()
const canManage = computed(() => can('diet:manage'))
const router = useRouter()

const { page, pageSize, total, resetPage } = usePagination()
const keyword = ref('')
const academicYearId = ref<number | undefined>(undefined)
const years = ref<{ id: number; year: string; is_current: boolean }[]>([])
const yearError = ref('')
const list = ref<DietPlanListItem[]>([])
const loading = ref(false)
const listError = ref('')

onMounted(async () => {
  await loadYears()
  await load()
})

async function loadYears() {
  yearError.value = ''
  try {
    const res = await getAcademicYears()
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
    const res = await getDietPlans({
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

// ===== 新建向导 =====

const wizardVisible = ref(false)
const creating = ref(false)
const createError = ref('')
const templates = ref<DietTemplateListItem[]>([])
const templatesLoading = ref(false)
const templateError = ref('')
const copyPlans = ref<DietPlanListItem[]>([])
const copyLoading = ref(false)
const copyError = ref('')

const wizard = reactive<{
  academic_year_id: number | undefined
  name: string
  start_date: string
  week_of_year: number | undefined
  created_method: DietCreatedMethod
  template_id: number | undefined
  source_plan_id: number | undefined
}>({
  academic_year_id: undefined,
  name: '',
  start_date: nextMondayISO(),
  week_of_year: undefined,
  created_method: 'custom',
  template_id: undefined,
  source_plan_id: undefined,
})

const wizardEndDate = computed(() =>
  wizard.start_date ? addDaysISO(wizard.start_date, 6) : '—',
)

function notMonday(date: Date) {
  return date.getDay() !== 1
}

function openWizard() {
  resetWizard()
  wizard.academic_year_id =
    academicYearId.value ??
    years.value.find((year) => year.is_current)?.id ??
    years.value[0]?.id
  wizardVisible.value = true
  void loadTemplates()
  void searchCopyPlans('')
}

function resetWizard() {
  creating.value = false
  createError.value = ''
  templateError.value = ''
  copyError.value = ''
  wizard.academic_year_id = academicYearId.value
  wizard.name = ''
  wizard.start_date = nextMondayISO()
  wizard.week_of_year = undefined
  wizard.created_method = 'custom'
  wizard.template_id = undefined
  wizard.source_plan_id = undefined
  templates.value = []
  copyPlans.value = []
  templatesLoading.value = false
  copyLoading.value = false
}

function onWizardYearChange() {
  if (wizard.created_method === 'copy') void searchCopyPlans('')
}

function onMethodChange() {
  createError.value = ''
  if (wizard.created_method === 'template' && templates.value.length === 0) {
    void loadTemplates()
  }
  if (wizard.created_method === 'copy' && copyPlans.value.length === 0) {
    void searchCopyPlans('')
  }
}

async function loadTemplates() {
  templatesLoading.value = true
  templateError.value = ''
  try {
    const res = await getDietTemplates()
    templates.value = res.list ?? []
  } catch (e: unknown) {
    templates.value = []
    templateError.value = e instanceof Error ? e.message : '模板加载失败'
  } finally {
    templatesLoading.value = false
  }
}

async function searchCopyPlans(query: string) {
  copyLoading.value = true
  copyError.value = ''
  try {
    const res = await getDietPlans({
      academic_year_id: wizard.academic_year_id,
      keyword: query.trim() || undefined,
      page: 1,
      page_size: 30,
    })
    copyPlans.value = res.list ?? []
  } catch (e: unknown) {
    copyPlans.value = []
    copyError.value = e instanceof Error ? e.message : '历史食谱加载失败'
  } finally {
    copyLoading.value = false
  }
}

async function submitWizard() {
  createError.value = ''
  const yearId = wizard.academic_year_id
  if (typeof yearId !== 'number') {
    createError.value = '请选择学年'
    return
  }
  const name = wizard.name.trim()
  if (!name) {
    createError.value = '请输入食谱名称'
    return
  }
  if (!wizard.start_date) {
    createError.value = '请选择开始日期'
    return
  }
  if (!isMondayISO(wizard.start_date)) {
    createError.value = '开始日期必须是周一'
    return
  }
  const endDate = addDaysISO(wizard.start_date, 6)
  const base = {
    name,
    academic_year_id: yearId,
    start_date: wizard.start_date,
    end_date: endDate,
    week_of_year: wizard.week_of_year ?? 0,
  }

  let body: DietPlanCreateBody
  if (wizard.created_method === 'template') {
    if (typeof wizard.template_id !== 'number') {
      createError.value = '请选择模板'
      return
    }
    body = { ...base, created_method: 'template', template_id: wizard.template_id }
  } else if (wizard.created_method === 'copy') {
    if (typeof wizard.source_plan_id !== 'number') {
      createError.value = '请选择源食谱'
      return
    }
    body = { ...base, created_method: 'copy', source_plan_id: wizard.source_plan_id }
  } else {
    body = { ...base, created_method: 'custom' }
  }

  creating.value = true
  try {
    const res = await createDietPlan(body)
    wizardVisible.value = false
    academicYearId.value = yearId
    if (res.warnings?.length) {
      await ElMessageBox.alert(res.warnings.join('；'), '创建完成，但有部分条目被跳过', {
        confirmButtonText: '前往编辑器',
      }).catch(() => undefined)
    } else {
      ElMessage.success('周食谱已创建')
    }
    await router.push(`/diet/plans/${res.id}`)
  } catch (e: unknown) {
    const err = e as DietApiError
    if (err.status === 409) {
      createError.value = '该学年下已存在同名周食谱，请更换名称'
    } else {
      createError.value = err.message || '创建失败'
    }
    if (err.status !== 409 && err.status !== 422) {
      ElMessage.error(createError.value)
    }
  } finally {
    creating.value = false
  }
}

// ===== 列表操作 =====

function openEditor(id: number) {
  router.push(`/diet/plans/${id}`)
}

async function remove(row: DietPlanListItem) {
  try {
    await ElMessageBox.confirm(
      `确定删除周食谱「${row.name}」？删除后列表与营养分析都不再返回该食谱。`,
      '删除确认',
      { type: 'warning', confirmButtonText: '确认删除' },
    )
  } catch {
    return
  }
  try {
    await deleteDietPlan(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}
</script>
