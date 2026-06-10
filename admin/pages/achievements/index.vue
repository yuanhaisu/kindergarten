<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex justify-between">
        <span>成就管理</span>
        <el-button type="primary" @click="openCreate">新建成就</el-button>
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
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="category_label" label="分类" width="100" />
      <el-table-column prop="earned_count" label="获得人数" width="100" />
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button link @click="edit(row as { id: number; name: string; description: string })">编辑</el-button>
          <el-button link type="danger" @click="remove((row as { id: number }).id)">删除</el-button>
        </template>
      </el-table-column>
    </DataTable>
    <el-dialog v-model="visible" :title="editId ? '编辑成就' : '新建成就'" width="480px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" /></el-form-item>
        <el-form-item label="图标URL"><el-input v-model="form.icon_url" /></el-form-item>
        <el-form-item label="分类"><el-input v-model="form.category" placeholder="attendance" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type { AchievementItem } from '~/types/api'
import {
  createAchievement,
  deleteAchievement,
  getAchievements,
  updateAchievement,
} from '~/api/achievements'

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<AchievementItem[]>([])

async function load() {
  loading.value = true
  try {
    const res = await getAchievements({ page: page.value, page_size: pageSize.value })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const visible = ref(false)
const editId = ref<number | null>(null)
const form = reactive({
  name: '',
  description: '',
  icon_url: '',
  category: 'growth',
  condition: 'custom',
  target_value: 1,
})

function openCreate() {
  editId.value = null
  Object.assign(form, {
    name: '',
    description: '',
    icon_url: '',
    category: 'growth',
    condition: 'custom',
    target_value: 1,
  })
  visible.value = true
}

function edit(row: { id: number; name: string; description: string }) {
  editId.value = row.id
  Object.assign(form, { name: row.name, description: row.description, icon_url: '', category: 'growth', condition: 'custom', target_value: 1 })
  visible.value = true
}

async function save() {
  try {
    if (editId.value) {
      await updateAchievement(editId.value, form)
    } else {
      await createAchievement(form)
    }
    ElMessage.success('已保存')
    visible.value = false
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  }
}

async function remove(id: number) {
  try {
    await ElMessageBox.confirm('确定删除？', '提示', { type: 'warning' })
    await deleteAchievement(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* */
  }
}

onMounted(load)
</script>
