<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-text-primary">分销商</h1>
        <p class="mt-1 text-xs text-text-muted">
          分销商不登录系统。佣金比例在订单确认时冻结，之后修改只影响新订单。
        </p>
      </div>
      <el-button type="primary" @click="openCreate">新增分销商</el-button>
    </div>

    <el-table v-loading="loading" :data="rows" border stripe>
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="contact" label="联系人" width="110" />
      <el-table-column prop="phone" label="电话" width="140" />
      <el-table-column label="佣金比例" width="110" align="right">
        <template #default="{ row }">{{ row.commission_rate }}%</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
            {{ row.status === 1 ? '启用' : '停用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="110" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="editVisible" :title="editingId ? '编辑分销商' : '新增分销商'" width="460px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="96px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="联系人">
          <el-input v-model="form.contact" />
        </el-form-item>
        <el-form-item label="电话">
          <el-input v-model="form.phone" maxlength="11" />
        </el-form-item>
        <el-form-item label="佣金比例 %">
          <el-input-number v-model="form.commission_rate" :min="0" :max="100" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.active" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { createDistributor, listDistributors, updateDistributor, type DistributorRow } from '~/api/platform/distributors'

definePageMeta({ layout: 'platform' })

const rows = ref<DistributorRow[]>([])
const loading = ref(false)
const submitting = ref(false)
const editVisible = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()
const form = reactive({ name: '', contact: '', phone: '', commission_rate: 10, active: true })
const rules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

async function load() {
  loading.value = true
  try {
    const data = await listDistributors({ status: 0 })
    rows.value = data.list || []
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  Object.assign(form, { name: '', contact: '', phone: '', commission_rate: 10, active: true })
  editVisible.value = true
}

function openEdit(row: DistributorRow) {
  editingId.value = row.id
  Object.assign(form, {
    name: row.name, contact: row.contact, phone: row.phone,
    commission_rate: row.commission_rate, active: row.status === 1,
  })
  editVisible.value = true
}

async function submit() {
  await formRef.value?.validate().catch(() => Promise.reject())
  submitting.value = true
  try {
    const body = {
      name: form.name, contact: form.contact, phone: form.phone,
      commission_rate: form.commission_rate, status: form.active ? 1 : 0,
    }
    if (editingId.value) {
      await updateDistributor(editingId.value, body)
    } else {
      await createDistributor(body)
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

onMounted(load)
</script>
