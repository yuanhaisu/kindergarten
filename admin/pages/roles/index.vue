<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex justify-between">
        <span>角色与权限</span>
        <el-button type="primary" @click="openCreate">新建角色</el-button>
      </div>
    </template>
    <el-table v-loading="loading" :data="roles" stripe border>
      <el-table-column prop="name" label="角色" />
      <el-table-column prop="description" label="说明" min-width="200" />
      <el-table-column prop="admin_count" label="管理员数" width="100" />
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button
            link
            type="primary"
            :disabled="(row as { is_system?: boolean }).is_system"
            @click="openEdit(row as { id: number; name: string; description: string; permissions: string[] })"
          >
            编辑
          </el-button>
          <el-button
            link
            type="danger"
            :disabled="(row as { is_system?: boolean }).is_system"
            @click="remove((row as { id: number }).id)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-dialog v-model="visible" :title="editId ? '编辑角色' : '新建角色'" width="640px" destroy-on-close>
      <el-form :model="form" label-width="88px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="说明"><el-input v-model="form.description" type="textarea" /></el-form-item>
        <el-form-item label="权限">
          <el-scrollbar max-height="360px">
            <div v-for="mod in permModules" :key="mod.module" class="mb-4">
              <div class="font-medium text-slate-700 mb-2">{{ mod.module_label }}</div>
              <el-checkbox-group v-model="form.permissions">
                <el-checkbox v-for="it in mod.items" :key="it.key" :label="it.key" class="!mr-4">
                  {{ it.label }}
                </el-checkbox>
              </el-checkbox-group>
            </div>
          </el-scrollbar>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type { PermissionModule, RoleItem } from '~/types/api'
import { createRole, deleteRole, getPermissionTree, getRoles, updateRole } from '~/api/roles'

const loading = ref(true)
const roles = ref<RoleItem[]>([])
const permModules = ref<PermissionModule[]>([])

async function load() {
  loading.value = true
  try {
    const [r, p] = await Promise.all([getRoles(), getPermissionTree()])
    roles.value = r.list ?? []
    permModules.value = p.permissions ?? []
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const visible = ref(false)
const saving = ref(false)
const editId = ref<number | null>(null)
const form = reactive({
  name: '',
  description: '',
  permissions: [] as string[],
})

function openCreate() {
  editId.value = null
  form.name = ''
  form.description = ''
  form.permissions = []
  visible.value = true
}

function openEdit(row: { id: number; name: string; description: string; permissions: string[] }) {
  editId.value = row.id
  form.name = row.name
  form.description = row.description
  form.permissions = [...(row.permissions || [])]
  visible.value = true
}

async function save() {
  saving.value = true
  try {
    if (editId.value) {
      await updateRole(editId.value, {
        name: form.name,
        description: form.description,
        permissions: form.permissions,
      })
    } else {
      await createRole({
        name: form.name,
        description: form.description,
        permissions: form.permissions,
      })
    }
    ElMessage.success('已保存')
    visible.value = false
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  } finally {
    saving.value = false
  }
}

async function remove(id: number) {
  try {
    await ElMessageBox.confirm('确定删除角色？', '提示', { type: 'warning' })
    await deleteRole(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* */
  }
}

onMounted(load)
</script>
