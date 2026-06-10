<template>
  <div class="flex gap-4" style="height: calc(100vh - 120px)">
    <!-- 左侧：字典类型列表 -->
    <el-card shadow="never" class="w-72 flex-shrink-0 flex flex-col">
      <template #header>
        <div class="flex items-center justify-between">
          <span>字典类型</span>
          <el-button size="small" type="primary" @click="openTypeDialog()">新建</el-button>
        </div>
      </template>
      <el-input
        v-model="typeSearch"
        placeholder="搜索类型..."
        size="small"
        clearable
        class="mb-3"
      />
      <div class="flex-1 overflow-y-auto -mx-2">
        <div
          v-for="t in filteredTypes"
          :key="t.id"
          class="flex items-center justify-between px-3 py-2 cursor-pointer rounded hover:bg-slate-50 transition-colors"
          :class="{ 'bg-blue-50': selectedTypeId === t.id }"
          @click="selectType(t.id)"
        >
          <div class="flex items-center gap-2 min-w-0">
            <span class="text-sm truncate">{{ t.type_name }}</span>
            <el-tag v-if="t.is_system" size="small" type="info" class="!text-xs">系统</el-tag>
          </div>
          <span class="text-xs text-slate-400 flex-shrink-0 ml-2">{{ t.item_count }}</span>
        </div>
        <el-empty v-if="filteredTypes.length === 0" description="暂无字典类型" :image-size="48" />
      </div>
    </el-card>

    <!-- 右侧：字典项列表 -->
    <el-card v-if="selectedTypeId" shadow="never" class="flex-1 flex flex-col">
      <template #header>
        <div class="flex items-center justify-between">
          <span>{{ currentType?.type_name }} — 字典项</span>
          <div class="flex gap-2">
            <el-button size="small" @click="openTypeDialog(currentType)">编辑类型</el-button>
            <el-button
              v-if="!currentType?.is_system"
              size="small"
              type="danger"
              @click="removeType(selectedTypeId)"
            >
              删除类型
            </el-button>
            <el-button size="small" type="primary" @click="openItemDialog()">新建项</el-button>
          </div>
        </div>
      </template>
      <el-table :data="items" stripe size="small" class="flex-1">
        <el-table-column label="排序" width="70" align="center">
          <template #default="{ $index }">{{ $index + 1 }}</template>
        </el-table-column>
        <el-table-column prop="item_label" label="标签" min-width="150">
          <template #default="{ row }">
            <template v-if="editingId !== row.id">
              <span class="cursor-pointer" @click="startInlineEdit(row)">{{ row.item_label }}</span>
            </template>
            <el-input
              v-else
              v-model="editLabel"
              size="small"
              @blur="saveInlineLabel(row)"
              @keyup.enter="saveInlineLabel(row)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="item_key" label="键值" width="120" />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-switch
              :model-value="row.is_enabled"
              size="small"
              @change="(val: boolean) => toggleItem(row, val)"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" align="center">
          <template #default="{ row }">
            <el-button size="small" text @click="openItemDialog(row)">编辑</el-button>
            <el-button size="small" text type="danger" @click="removeItem(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    <el-empty v-else description="请在左侧选择一个字典类型" class="flex-1" />

    <!-- 字典类型弹窗 -->
    <el-dialog
      v-model="typeDialogVisible"
      :title="editingTypeId ? '编辑字典类型' : '新建字典类型'"
      width="480px"
      @closed="resetTypeForm"
    >
      <el-form ref="typeFormRef" :model="typeForm" :rules="typeRules" label-width="100px">
        <el-form-item label="类型标识" prop="type_key">
          <el-input v-model="typeForm.type_key" :disabled="!!editingTypeId" placeholder="英文标识，如 performance_tag" />
        </el-form-item>
        <el-form-item label="类型名称" prop="type_name">
          <el-input v-model="typeForm.type_name" placeholder="中文名称，如 表现标签" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="typeForm.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="typeForm.sort_order" :min="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="typeDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="typeSaving" @click="saveType">保存</el-button>
      </template>
    </el-dialog>

    <!-- 字典项弹窗 -->
    <el-dialog
      v-model="itemDialogVisible"
      :title="editingItemId ? '编辑字典项' : '新建字典项'"
      width="480px"
      @closed="resetItemForm"
    >
      <el-form ref="itemFormRef" :model="itemForm" :rules="itemRules" label-width="100px">
        <el-form-item label="键值" prop="item_key">
          <el-input v-model="itemForm.item_key" :disabled="!!editingItemId" placeholder="英文标识" />
        </el-form-item>
        <el-form-item label="标签" prop="item_label">
          <el-input v-model="itemForm.item_label" placeholder="显示文本" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="itemForm.sort_order" :min="0" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="itemForm.is_enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="itemDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="itemSaving" @click="saveItem">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import {
  listDictTypes,
  createDictType,
  updateDictType,
  deleteDictType,
  listDictItems,
  createDictItem,
  updateDictItem,
  deleteDictItem,
} from '~/api/dict'
import type { DictTypeDetail, DictItemDetail } from '~/types/api'

// ===== Dict Types =====
const types = ref<DictTypeDetail[]>([])
const selectedTypeId = ref<number | null>(null)
const typeSearch = ref('')
const filteredTypes = computed(() => {
  if (!typeSearch.value) return types.value
  const kw = typeSearch.value.toLowerCase()
  return types.value.filter(
    t => t.type_name.includes(kw) || t.type_key.includes(kw),
  )
})
const currentType = computed(() => types.value.find(t => t.id === selectedTypeId.value))

async function loadTypes() {
  const res = await listDictTypes()
  types.value = res.list || []
}

function selectType(id: number) {
  selectedTypeId.value = id
  loadItems()
}

// ===== Type Dialog =====
const typeDialogVisible = ref(false)
const typeSaving = ref(false)
const editingTypeId = ref<number | null>(null)
const typeFormRef = ref<FormInstance>()
const typeForm = ref({ type_key: '', type_name: '', description: '', sort_order: 0 })
const typeRules: FormRules = {
  type_key: [{ required: true, message: '请输入类型标识', trigger: 'blur' }],
  type_name: [{ required: true, message: '请输入类型名称', trigger: 'blur' }],
}

function openTypeDialog(row?: DictTypeDetail | null) {
  editingTypeId.value = row?.id ?? null
  if (row) {
    typeForm.value = {
      type_key: row.type_key,
      type_name: row.type_name,
      description: row.description || '',
      sort_order: row.sort_order,
    }
  } else {
    typeForm.value = { type_key: '', type_name: '', description: '', sort_order: 0 }
  }
  typeDialogVisible.value = true
}

function resetTypeForm() {
  editingTypeId.value = null
}

async function saveType() {
  const valid = await typeFormRef.value?.validate().catch(() => false)
  if (!valid) return
  typeSaving.value = true
  try {
    if (editingTypeId.value) {
      await updateDictType(editingTypeId.value, typeForm.value)
      ElMessage.success('更新成功')
    } else {
      await createDictType(typeForm.value)
      ElMessage.success('创建成功')
    }
    typeDialogVisible.value = false
    await loadTypes()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '操作失败')
  } finally {
    typeSaving.value = false
  }
}

async function removeType(id: number) {
  try {
    await ElMessageBox.confirm('确认删除该字典类型？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await deleteDictType(id)
    ElMessage.success('删除成功')
    if (selectedTypeId.value === id) selectedTypeId.value = null
    await loadTypes()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '删除失败')
  }
}

// ===== Dict Items =====
const items = ref<DictItemDetail[]>([])
const editingId = ref<number | null>(null)
const editLabel = ref('')

async function loadItems() {
  if (!selectedTypeId.value) return
  const res = await listDictItems(selectedTypeId.value)
  items.value = res.list || []
}

function startInlineEdit(row: DictItemDetail) {
  editingId.value = row.id
  editLabel.value = row.item_label
  nextTick(() => {
    const input = document.querySelector('.el-input__inner') as HTMLInputElement
    input?.focus()
  })
}

async function saveInlineLabel(row: DictItemDetail) {
  if (editLabel.value === row.item_label) {
    editingId.value = null
    return
  }
  try {
    await updateDictItem(row.id, { item_label: editLabel.value })
    row.item_label = editLabel.value
    ElMessage.success('已更新')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '更新失败')
  }
  editingId.value = null
}

async function toggleItem(row: DictItemDetail, val: boolean) {
  try {
    await updateDictItem(row.id, { is_enabled: val })
    row.is_enabled = val
  } catch (e: any) {
    ElMessage.error('操作失败')
  }
}

// ===== Item Dialog =====
const itemDialogVisible = ref(false)
const itemSaving = ref(false)
const editingItemId = ref<number | null>(null)
const itemFormRef = ref<FormInstance>()
const itemForm = ref({ item_key: '', item_label: '', sort_order: 0, is_enabled: true })
const itemRules: FormRules = {
  item_key: [{ required: true, message: '请输入键值', trigger: 'blur' }],
  item_label: [{ required: true, message: '请输入标签', trigger: 'blur' }],
}

function openItemDialog(row?: DictItemDetail | null) {
  editingItemId.value = row?.id ?? null
  if (row) {
    itemForm.value = {
      item_key: row.item_key,
      item_label: row.item_label,
      sort_order: row.sort_order,
      is_enabled: row.is_enabled,
    }
  } else {
    itemForm.value = { item_key: '', item_label: '', sort_order: items.value.length + 1, is_enabled: true }
  }
  itemDialogVisible.value = true
}

function resetItemForm() {
  editingItemId.value = null
}

async function saveItem() {
  const valid = await itemFormRef.value?.validate().catch(() => false)
  if (!valid) return
  itemSaving.value = true
  try {
    if (editingItemId.value) {
      await updateDictItem(editingItemId.value, itemForm.value)
      ElMessage.success('更新成功')
    } else {
      if (!selectedTypeId.value) return
      await createDictItem(selectedTypeId.value, itemForm.value)
      ElMessage.success('创建成功')
    }
    itemDialogVisible.value = false
    await loadItems()
    await loadTypes() // refresh item count
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '操作失败')
  } finally {
    itemSaving.value = false
  }
}

async function removeItem(id: number) {
  try {
    await ElMessageBox.confirm('确认删除该字典项？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await deleteDictItem(id)
    ElMessage.success('删除成功')
    await loadItems()
    await loadTypes()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '删除失败')
  }
}

// ===== Init =====
onMounted(() => {
  loadTypes()
})
</script>
