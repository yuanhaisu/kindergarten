<template>
  <div class="space-y-6">
    <el-alert
      v-if="loadError"
      type="error"
      :closable="false"
      show-icon
      title="供餐配置加载失败"
      :description="loadError"
    >
      <el-button size="small" @click="load">重试</el-button>
    </el-alert>

    <el-alert
      v-if="saveError"
      type="error"
      :closable="true"
      show-icon
      title="保存失败"
      :description="saveError"
      @close="saveError = ''"
    />

    <el-card v-loading="loading" shadow="never">
      <template #header>
        <div class="flex items-center justify-between gap-4">
          <span>基本参数</span>
          <el-button v-if="canManage" type="primary" :loading="saving" :disabled="!form" @click="save">
            保存
          </el-button>
        </div>
      </template>

      <el-form v-if="form" :model="form" :disabled="!canManage" label-width="160px" class="max-w-4xl">
        <el-form-item label="托管形式">
          <el-radio-group v-model="form.care_type">
            <el-radio value="day_care">日托</el-radio>
            <el-radio value="full_care">全托</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="比例模式">
          <el-radio-group v-model="form.ratio_mode" @change="onRatioModeChange">
            <el-radio value="standard">标准比例</el-radio>
            <el-radio value="custom">自定比例</el-radio>
          </el-radio-group>
          <div class="text-text-muted text-xs leading-5">
            标准模式使用固定占比并跳过相邻上限；自定模式按餐次校验相邻上限。
          </div>
        </el-form-item>

        <el-form-item label="重量口径">
          <span class="text-text-primary">市品重量</span>
          <span class="text-text-muted text-xs ml-2">本版本固定按市品重量录入，不可切换</span>
        </el-form-item>

        <el-divider content-position="left">餐次</el-divider>

        <el-table :data="form.meals" row-key="meal_key" border class="w-full">
          <el-table-column prop="name" label="餐次" width="100" />
          <el-table-column label="就餐时间" width="180">
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
          <el-table-column label="供能占比(%)" width="160">
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
          <el-table-column label="启用" width="90">
            <template #default="{ row }">
              <el-switch v-model="(row as MealFormItem).enabled" />
            </template>
          </el-table-column>
        </el-table>

        <el-form-item label="折算系数" class="!mt-4">
          <span class="text-text-primary font-medium">{{ coefficient.toFixed(2) }}</span>
          <span class="text-text-muted text-xs ml-2">= 已启用餐次供能占比合计 ÷ 100（只读）</span>
        </el-form-item>

        <el-form-item v-if="!canManage" label=" ">
          <span class="text-text-muted text-sm">当前账号无 diet:manage 权限，仅可查看。</span>
        </el-form-item>
      </el-form>

      <el-empty v-else-if="!loading && !loadError" description="暂无供餐配置" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import type {
  DietApiError,
  DietCareType,
  DietConfigUpdateBody,
  DietFoodInputType,
  DietMealConfigInput,
  DietMealKey,
  DietRatioMode,
} from '~/types/api'
import { getDietConfig, updateDietConfig } from '~/api/diet'

// 契约默认值：首次读取由后端返回同一组数据；后端未下发时前端兜底展示
const MEAL_DEFAULTS: { key: DietMealKey; name: string; time: string; percent: number; enabled: boolean }[] = [
  { key: 'breakfast', name: '早餐', time: '07:00', percent: 25, enabled: true },
  { key: 'morning_snack', name: '早点', time: '09:00', percent: 5, enabled: true },
  { key: 'lunch', name: '午餐', time: '11:00', percent: 35, enabled: true },
  { key: 'afternoon_snack', name: '午点', time: '14:30', percent: 5, enabled: true },
  { key: 'dinner', name: '晚餐', time: '17:00', percent: 25, enabled: true },
  { key: 'evening_snack', name: '晚点', time: '20:00', percent: 5, enabled: false },
]

interface MealFormItem {
  meal_key: DietMealKey
  name: string
  time: string
  /** el-input-number 清空时给 undefined，本地保持三态，提交前统一校验 */
  energy_percent: number | null | undefined
  sort: number
  enabled: boolean
}

interface ConfigForm {
  care_type: DietCareType
  care_type_label: string
  food_input_type: DietFoodInputType
  food_input_type_label: string
  ratio_mode: DietRatioMode
  ratio_mode_label: string
  coefficient: number
  meals: MealFormItem[]
}

function defaultConfig(): ConfigForm {
  return {
    care_type: 'day_care',
    care_type_label: '日托',
    food_input_type: 'market_weight',
    food_input_type_label: '市品重量',
    ratio_mode: 'standard',
    ratio_mode_label: '标准比例',
    coefficient: 0.95,
    meals: MEAL_DEFAULTS.map((m, i) => ({
      meal_key: m.key,
      name: m.name,
      time: m.time,
      energy_percent: m.percent,
      sort: i + 1,
      enabled: m.enabled,
    })),
  }
}

const { can } = usePermission()
const canManage = computed(() => can('diet:manage'))

const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const saveError = ref('')
const form = ref<ConfigForm | null>(null)

const isStandard = computed(() => form.value?.ratio_mode === 'standard')

const coefficient = computed(() => {
  const meals = form.value?.meals ?? []
  const sum = meals.filter((m) => m.enabled).reduce((acc, m) => acc + (Number(m.energy_percent) || 0), 0)
  return sum / 100
})

onMounted(load)

async function load() {
  loading.value = true
  loadError.value = ''
  saveError.value = ''
  try {
    const res = await getDietConfig()
    form.value = res && Array.isArray(res.meals) && res.meals.length > 0 ? res : defaultConfig()
  } catch (e: unknown) {
    // 加载失败时不展示可编辑的默认值，避免把默认值误存回服务端
    form.value = null
    loadError.value = e instanceof Error ? e.message : '网络错误'
  } finally {
    loading.value = false
  }
}

/** 切回标准模式时占比必须回到固定值，否则提交会被 422 拒绝 */
function onRatioModeChange(mode: string | number | boolean | undefined) {
  if (mode !== 'standard' || !form.value) return
  for (const meal of form.value.meals) {
    const fixed = MEAL_DEFAULTS.find((d) => d.key === meal.meal_key)
    if (fixed) meal.energy_percent = fixed.percent
  }
}

async function save() {
  if (!form.value) return
  saveError.value = ''

  const meals: DietMealConfigInput[] = []
  for (const meal of form.value.meals) {
    if (meal.energy_percent == null || !Number.isFinite(meal.energy_percent)) {
      saveError.value = `「${meal.name}」供能占比不能为空`
      return
    }
    meals.push({
      meal_key: meal.meal_key,
      time: meal.time,
      energy_percent: meal.energy_percent,
      enabled: meal.enabled,
    })
  }

  const body: DietConfigUpdateBody = {
    care_type: form.value.care_type,
    food_input_type: form.value.food_input_type,
    ratio_mode: form.value.ratio_mode,
    meals,
  }

  saving.value = true
  try {
    form.value = await updateDietConfig(body)
    ElMessage.success('已保存')
  } catch (e: unknown) {
    const err = e as DietApiError
    saveError.value = err.message || '保存失败'
    if (err.status !== 422) {
      ElMessage.error(saveError.value)
    }
  } finally {
    saving.value = false
  }
}
</script>
