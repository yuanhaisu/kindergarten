<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-text-primary">套餐</h1>
        <p class="mt-1 text-xs text-text-muted">
          套餐目录可以改；已成交订单只认成交时的快照，不会被这里的改动影响。
        </p>
      </div>
      <el-button type="primary" @click="openCreate">新建套餐</el-button>
    </div>

    <el-table v-loading="loading" :data="rows" border stripe>
      <el-table-column prop="code" label="编码" width="130" />
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="tier" label="档位" width="70" align="center" />
      <el-table-column label="价格" width="110" align="right">
        <template #default="{ row }">{{ formatMoney(row.price_cents) }}</template>
      </el-table-column>
      <el-table-column label="周期" width="80" align="center">
        <template #default="{ row }">{{ row.term_months }} 个月</template>
      </el-table-column>
      <el-table-column label="包含模块" min-width="220">
        <template #default="{ row }">
          <el-tag v-for="code in row.module_codes" :key="code" size="small" class="mr-1">
            {{ moduleName(code) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
            {{ row.status === 'active' ? '可售' : '已下架' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="170" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="primary" @click="openTiers(row)">档位</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="editVisible" :title="editingId ? '编辑套餐' : '新建套餐'" width="520px">
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="96px">
        <el-form-item label="套餐编码" prop="code">
          <el-input v-model="form.code" :disabled="!!editingId" placeholder="如 basic / flagship" />
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="档位">
          <el-input-number v-model="form.tier" :min="1" :max="9" />
        </el-form-item>
        <el-form-item label="价格（元）">
          <el-input-number v-model="priceYuan" :min="0" :precision="2" :step="100" />
          <span class="ml-2 text-xs text-text-muted">每个计费周期、每所园</span>
        </el-form-item>
        <el-form-item label="周期（月）">
          <el-input-number v-model="form.term_months" :min="1" :max="36" />
        </el-form-item>
        <el-form-item label="包含模块">
          <el-checkbox-group v-model="form.module_codes">
            <el-checkbox v-for="m in modules" :key="m.code" :value="m.code" :disabled="m.code === 'base'">
              {{ m.name }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio value="active">可售</el-radio>
            <el-radio value="archived">下架</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitEdit">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="tiersVisible" title="幼儿数档位" width="620px">
      <p class="mb-3 text-xs text-text-muted">
        按在园幼儿数落入的区间决定价格系数（100 = 不加价）。档位变更只影响之后创建的订单。
      </p>
      <el-table :data="tiers" size="small" border>
        <el-table-column label="档位" width="80">
          <template #default="{ row }">
            <el-input-number v-model="row.tier" :min="1" :max="9" size="small" controls-position="right" />
          </template>
        </el-table-column>
        <el-table-column label="幼儿数下限" width="140">
          <template #default="{ row }">
            <el-input-number v-model="row.min_students" :min="0" size="small" controls-position="right" />
          </template>
        </el-table-column>
        <el-table-column label="上限（0=无）" width="140">
          <template #default="{ row }">
            <el-input-number v-model="row.max_students" :min="0" size="small" controls-position="right" />
          </template>
        </el-table-column>
        <el-table-column label="价格系数 %" width="130">
          <template #default="{ row }">
            <el-input-number v-model="row.price_multiplier" :min="1" size="small" controls-position="right" />
          </template>
        </el-table-column>
        <el-table-column label="" width="70">
          <template #default="{ $index }">
            <el-button link type="danger" @click="tiers.splice($index, 1)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-button class="mt-3" size="small" @click="tiers.push({ tier: tiers.length + 1, min_students: 0, max_students: 0, price_multiplier: 100 })">
        添加档位
      </el-button>
      <template #footer>
        <el-button @click="tiersVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitTiers">保存档位</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import {
  createPlan,
  listModules,
  listPlans,
  listPlanTiers,
  replacePlanTiers,
  updatePlan,
  type ModuleDef,
  type PlanRow,
} from '~/api/platform'

definePageMeta({ layout: 'platform' })

const rows = ref<PlanRow[]>([])
const modules = ref<ModuleDef[]>([])
const loading = ref(false)
const submitting = ref(false)

const editVisible = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()
const form = reactive({
  code: '',
  name: '',
  tier: 1,
  price_cents: 0,
  term_months: 6,
  module_codes: ['base'] as string[],
  status: 'active',
})
const priceYuan = computed({
  get: () => form.price_cents / 100,
  set: (v: number) => {
    form.price_cents = Math.round(v * 100)
  },
})
const formRules: FormRules = {
  code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

const tiersVisible = ref(false)
const tiersPlanId = ref<number | null>(null)
const tiers = ref<{ tier: number; min_students: number; max_students: number; price_multiplier: number }[]>([])

async function load() {
  loading.value = true
  try {
    const [plans, mods] = await Promise.all([listPlans(), listModules()])
    rows.value = plans.list || []
    modules.value = mods.list || []
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  Object.assign(form, {
    code: '', name: '', tier: 1, price_cents: 0, term_months: 6,
    module_codes: ['base'], status: 'active',
  })
  editVisible.value = true
}

function openEdit(row: PlanRow) {
  editingId.value = row.id
  Object.assign(form, {
    code: row.code, name: row.name, tier: row.tier, price_cents: row.price_cents,
    term_months: row.term_months, module_codes: [...row.module_codes], status: row.status,
  })
  editVisible.value = true
}

async function submitEdit() {
  await formRef.value?.validate().catch(() => Promise.reject())
  submitting.value = true
  try {
    const body = { ...form, module_codes: uniqueCodes(form.module_codes) }
    if (editingId.value) {
      await updatePlan(editingId.value, body)
    } else {
      await createPlan(body)
    }
    ElMessage.success('已保存')
    editVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    submitting.value = false
  }
}

async function openTiers(row: PlanRow) {
  tiersPlanId.value = row.id
  tiersVisible.value = true
  try {
    const data = await listPlanTiers(row.id)
    tiers.value = data.list?.length
      ? data.list
      : [{ tier: 1, min_students: 0, max_students: 0, price_multiplier: 100 }]
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载档位失败')
  }
}

async function submitTiers() {
  if (!tiersPlanId.value) return
  submitting.value = true
  try {
    await replacePlanTiers(tiersPlanId.value, tiers.value)
    ElMessage.success('档位已保存')
    tiersVisible.value = false
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存档位失败')
  } finally {
    submitting.value = false
  }
}

function uniqueCodes(codes: string[]) {
  return Array.from(new Set(['base', ...codes]))
}
function moduleName(code: string) {
  return modules.value.find((m) => m.code === code)?.name || code
}
function formatMoney(cents: number) {
  return `¥${(cents / 100).toFixed(2)}`
}

onMounted(load)
</script>
