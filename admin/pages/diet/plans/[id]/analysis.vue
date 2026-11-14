<template>
  <div v-loading="loading" class="min-h-32 space-y-4">
    <el-alert
      v-if="loadError"
      type="error"
      :closable="false"
      show-icon
      :title="loadError"
      data-testid="analysis-load-error"
    >
      <div class="mt-2 flex flex-wrap gap-2">
        <el-button size="small" @click="load">重试</el-button>
        <el-button size="small" @click="goEditor">前往编辑器</el-button>
        <el-button size="small" @click="goList">返回列表</el-button>
      </div>
    </el-alert>

    <el-card v-if="emptyMessage" shadow="never" data-testid="analysis-empty">
      <el-empty :description="emptyMessage">
        <el-button type="primary" data-testid="analysis-empty-editor" @click="goEditor">
          前往编辑器录入明细
        </el-button>
        <el-button @click="goList">返回列表</el-button>
      </el-empty>
    </el-card>

    <el-card v-if="analysis" shadow="never">
      <template #header>
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div class="analysis-no-print flex flex-wrap items-center gap-2">
              <el-button link type="primary" @click="goList">← 返回列表</el-button>
              <el-button link type="primary" @click="goEditor">返回编辑器</el-button>
            </div>
            <div class="mt-1 text-lg font-semibold text-text-primary">
              {{ analysis.plan.name }} · 营养分析
            </div>
            <div class="text-sm text-text-body">
              {{ analysis.plan.start_date }} 至 {{ analysis.plan.end_date }} · 有明细
              {{ analysis.plan.day_num }} 天 · 折算系数 {{ analysis.plan.coefficient }}
            </div>
          </div>
          <div class="analysis-no-print flex flex-wrap gap-2">
            <el-button
              v-if="canExport"
              data-testid="analysis-export"
              :loading="exporting"
              @click="exportExcel"
            >
              导出 Excel
            </el-button>
            <el-button type="primary" data-testid="analysis-print" @click="printReport">
              打印
            </el-button>
          </div>
        </div>
      </template>

      <div class="analysis-no-print mb-2 text-xs leading-5 text-text-muted">
        数值按后端报表口径原样展示，不做二次取整或改写；DRIs 行为全日原值。
        同一份报表存在三个能量总量、三餐供能比合计不等于 100%、钙来源不互斥等非标准口径，均按参考产品保留。
      </div>

      <el-tabs v-model="activeTab" class="analysis-tabs" data-testid="analysis-tabs">
        <el-tab-pane label="营养分析表" name="nutrition" class="analysis-print-pane">
          <div data-testid="analysis-tab-nutrition" class="space-y-5">
            <AnalysisNutritionPanel
              :nutrition="analysis.nutrition"
              :macro-energy="analysis.macro_energy"
              :protein-source="analysis.protein_source"
              :fat-source="analysis.fat_source"
            />
            <div class="border-t border-border-subtle pt-4">
              <AnalysisConclusionPanel :conclusion="analysis.conclusion" />
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="三餐供能比" name="meal" class="analysis-print-pane">
          <div data-testid="analysis-tab-meal-energy">
            <AnalysisMealEnergyPanel :meal-energy="analysis.meal_energy" />
          </div>
        </el-tab-pane>

        <el-tab-pane label="膳食构成" name="composition" class="analysis-print-pane">
          <div data-testid="analysis-tab-composition">
            <AnalysisCompositionPanel :composition="analysis.composition" />
          </div>
        </el-tab-pane>

        <el-tab-pane label="钙铁来源" name="calcium_iron" class="analysis-print-pane">
          <div data-testid="analysis-tab-calcium-iron">
            <AnalysisCalciumIronPanel :calcium-iron="analysis.calcium_iron" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import type { DietAnalysis, DietApiError } from '~/types/api'
import { exportDietPlanAnalysis, getDietPlanAnalysis } from '~/api/diet'
import { downloadBlob } from '~/utils/export'
import AnalysisNutritionPanel from '../components/AnalysisNutritionPanel.vue'
import AnalysisConclusionPanel from '../components/AnalysisConclusionPanel.vue'
import AnalysisMealEnergyPanel from '../components/AnalysisMealEnergyPanel.vue'
import AnalysisCompositionPanel from '../components/AnalysisCompositionPanel.vue'
import AnalysisCalciumIronPanel from '../components/AnalysisCalciumIronPanel.vue'

const route = useRoute()
const router = useRouter()
const { can } = usePermission()
const canExport = computed(() => can('diet:export'))

const analysis = ref<DietAnalysis | null>(null)
const loading = ref(false)
const loadError = ref('')
const emptyMessage = ref('')
const activeTab = ref('nutrition')
const exporting = ref(false)

const planId = computed(() => Number(route.params.id))

// 切换食谱时旧请求可能后返回；用序号保证只有最新一次请求能落地
let loadSeq = 0

onMounted(load)
watch(
  () => route.params.id,
  () => {
    activeTab.value = 'nutrition'
    void load()
  },
)

async function load() {
  const seq = ++loadSeq
  const id = planId.value
  analysis.value = null
  loadError.value = ''
  emptyMessage.value = ''
  if (!Number.isFinite(id) || id <= 0) {
    loading.value = false
    loadError.value = '周食谱 ID 无效'
    return
  }
  loading.value = true
  try {
    const data = await getDietPlanAnalysis(id)
    if (seq !== loadSeq) return
    analysis.value = data
  } catch (e: unknown) {
    if (seq !== loadSeq) return
    const err = e as DietApiError
    if (err.status === 422) {
      // 契约：无明细走 422，页面转空态而不是错误态
      emptyMessage.value = err.message || '该食谱暂无明细，无法生成营养分析'
    } else if (err.status === 404) {
      loadError.value = '周食谱不存在或已被删除'
    } else {
      loadError.value = err.message || '营养分析加载失败'
    }
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

function goList() {
  return router.push('/diet/plans')
}

function goEditor() {
  const id = analysis.value?.plan.id ?? planId.value
  return router.push(Number.isFinite(id) && id > 0 ? `/diet/plans/${id}` : '/diet/plans')
}

function printReport() {
  if (!analysis.value) return
  window.print()
}

async function exportExcel() {
  const current = analysis.value
  if (!current) return
  exporting.value = true
  try {
    const blob = await exportDietPlanAnalysis(current.plan.id)
    downloadBlob(blob, '营养分析.xlsx')
  } catch (e: unknown) {
    const err = e as DietApiError
    ElMessage.error(
      err.status === 422
        ? '该食谱暂无明细，无法生成营养分析'
        : err.message || '导出失败，请稍后重试',
    )
  } finally {
    exporting.value = false
  }
}
</script>

<style>
/* 打印视图：隐藏导航与操作按钮，四个页签全部展开并逐页分页。 */
@media print {
  .analysis-no-print {
    display: none !important;
  }

  .analysis-tabs .el-tabs__header {
    display: none !important;
  }

  .analysis-tabs .el-tabs__content {
    overflow: visible !important;
  }

  .analysis-print-pane {
    display: block !important;
  }

  /* 打印不滚动：横向滚动容器必须放开，否则宽表会被裁掉 */
  .analysis-print-pane .overflow-x-auto,
  .analysis-print-pane .el-table__body-wrapper,
  .analysis-print-pane .el-table__header-wrapper {
    overflow: visible !important;
  }

  .analysis-print-pane + .analysis-print-pane {
    break-before: page;
  }

  .admin-shell > aside,
  .admin-shell > div > header,
  .admin-shell > div > [role='alert'] {
    display: none !important;
  }

  .admin-shell main {
    overflow: visible !important;
    padding: 0 !important;
  }

  .el-card {
    border: none !important;
    box-shadow: none !important;
  }

  .admin-shell main .el-card__body,
  .admin-shell main .el-card__header {
    padding-left: 0 !important;
    padding-right: 0 !important;
  }
}

@page {
  size: A4 landscape;
  margin: 8mm;
}
</style>
