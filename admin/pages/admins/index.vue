<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex justify-between">
        <span>管理员</span>
        <el-button type="primary" @click="openCreate">添加管理员</el-button>
      </div>
    </template>
    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="list as unknown as Record<string, unknown>[]"
      :loading="loading"
      :total="total"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column prop="nickname" label="昵称" />
      <el-table-column prop="phone" label="手机" width="130" />
      <el-table-column label="角色" min-width="140">
        <template #default="{ row }">
          {{ (row as { admin_role?: { name: string } }).admin_role?.name }}
        </template>
      </el-table-column>
      <el-table-column prop="status_label" label="状态" width="90" />
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row as AdminItem)">编辑</el-button>
          <el-button link @click="resetPwd((row as { id: number }).id)">重置密码</el-button>
          <el-button link type="danger" @click="remove((row as { id: number }).id)">删除</el-button>
        </template>
      </el-table-column>
    </DataTable>
    <el-dialog v-model="visible" :title="editingId ? '编辑管理员' : '添加管理员'" width="440px" destroy-on-close>
      <el-form :model="form" label-width="90px">
        <el-form-item label="手机"><el-input v-model="form.phone" :disabled="!!editingId" /></el-form-item>
        <el-form-item label="昵称"><el-input v-model="form.nickname" /></el-form-item>
        <el-form-item label="角色ID"><el-input-number v-model="form.role_id" :min="1" class="!w-full" /></el-form-item>
        <el-form-item v-if="!editingId" label="初始密码"><el-input v-model="form.password" type="password" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="submit">{{ editingId ? '保存' : '新增' }}</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type { AdminItem } from '~/types/api'
import { createAdmin, deleteAdmin, getAdmins, resetAdminPassword, updateAdmin } from '~/api/admins'

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<AdminItem[]>([])

async function load() {
  loading.value = true
  try {
    const res = await getAdmins({ page: page.value, page_size: pageSize.value })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const visible = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({
  phone: '',
  nickname: '',
  role_id: 1,
  password: 'Abc123456',
})

function resetForm() {
  form.phone = ''
  form.nickname = ''
  form.role_id = 1
  form.password = 'Abc123456'
}

function openCreate() {
  editingId.value = null
  resetForm()
  visible.value = true
}

function openEdit(row: AdminItem) {
  editingId.value = row.id
  form.phone = row.phone
  form.nickname = row.nickname
  form.role_id = row.admin_role?.id || 1
  form.password = 'Abc123456'
  visible.value = true
}

async function submit() {
  try {
    if (editingId.value) {
      await updateAdmin(editingId.value, {
        nickname: form.nickname,
        role_id: form.role_id,
      })
      ElMessage.success('已更新')
    } else {
      await createAdmin(form)
      ElMessage.success('已添加')
    }
    visible.value = false
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  }
}

async function resetPwd(id: number) {
  try {
    const { value } = await ElMessageBox.prompt('新密码', '重置密码', {
      inputType: 'password',
      inputValue: 'NewPass123',
    })
    await resetAdminPassword(id, { new_password: value })
    ElMessage.success('已重置')
  } catch {
    /* */
  }
}

async function remove(id: number) {
  try {
    await ElMessageBox.confirm('确定删除该管理员？', '提示', { type: 'warning' })
    await deleteAdmin(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* */
  }
}

onMounted(load)
</script>
