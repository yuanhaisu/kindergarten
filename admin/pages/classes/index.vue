<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span>班级管理</span>
        <el-button type="primary" @click="openCreate">新建班级</el-button>
      </div>
    </template>
    <SearchBar v-model:keyword="keyword" @search="load" @reset="load" />
    <el-select v-model="grade" placeholder="年级" clearable class="!w-32 mb-4 mr-2" @change="load">
      <el-option label="小班" value="小班" />
      <el-option label="中班" value="中班" />
      <el-option label="大班" value="大班" />
    </el-select>
    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="list as unknown as Record<string, unknown>[]"
      :loading="loading"
      :total="total"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column prop="name" label="班级" min-width="120" />
      <el-table-column prop="grade" label="年级" width="90" />
      <el-table-column prop="academic_year" label="学年" width="120" />
      <el-table-column prop="student_count" label="人数" width="80" />
      <el-table-column prop="capacity" label="容量" width="80" />
      <el-table-column label="老师" min-width="200">
        <template #default="{ row }">
          <span class="text-sm text-slate-600">
            {{
              (row as { teachers?: { nickname: string; title_label: string }[] }).teachers
                ?.map((t) => `${t.nickname}(${t.title_label})`)
                .join('、') || '-'
            }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row as { id: number })">编辑</el-button>
          <el-button link type="danger" @click="onDelete(row as { id: number; name: string })">
            删除
          </el-button>
        </template>
      </el-table-column>
    </DataTable>
    <el-dialog v-model="dialogVisible" :title="editId ? '编辑班级' : '新建班级'" width="520px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="年级">
          <el-select v-model="form.grade" class="w-full">
            <el-option label="小班" value="小班" />
            <el-option label="中班" value="中班" />
            <el-option label="大班" value="大班" />
          </el-select>
        </el-form-item>
        <el-form-item label="学年"><el-input v-model="form.academic_year" /></el-form-item>
        <el-form-item label="容量"><el-input-number v-model="form.capacity" :min="1" class="!w-full" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type { ClassItem } from '~/types/api'
import { createClass, deleteClass, getClasses, getClassDetail, updateClass } from '~/api/classes'

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<ClassItem[]>([])
const keyword = ref('')
const grade = ref<string | undefined>()

async function load() {
  loading.value = true
  try {
    const res = await getClasses({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value || undefined,
      grade: grade.value,
    })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const dialogVisible = ref(false)
const editId = ref<number | null>(null)
const saving = ref(false)
const form = reactive({
  name: '',
  grade: '小班',
  academic_year: '',
  capacity: 30,
})

function openCreate() {
  editId.value = null
  Object.assign(form, { name: '', grade: '小班', academic_year: '', capacity: 30 })
  dialogVisible.value = true
}

async function openEdit(row: { id: number }) {
  editId.value = row.id
  try {
    const d = await getClassDetail(row.id)
    Object.assign(form, {
      name: d.name,
      grade: d.grade,
      academic_year: d.academic_year,
      capacity: d.capacity,
    })
    dialogVisible.value = true
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载详情失败')
  }
}

async function save() {
  saving.value = true
  try {
    if (editId.value) {
      await updateClass(editId.value, {
        name: form.name,
        grade: form.grade,
        capacity: form.capacity,
        teacher_ids: [],
      })
    } else {
      await createClass({
        name: form.name,
        grade: form.grade,
        academic_year: form.academic_year,
        capacity: form.capacity,
        teacher_ids: [],
      })
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    await load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function onDelete(row: { id: number; name: string }) {
  try {
    await ElMessageBox.confirm(`确定删除班级「${row.name}」？`, '提示', { type: 'warning' })
    await deleteClass(row.id)
    ElMessage.success('已删除')
    await load()
  } catch {
    /* cancel */
  }
}

onMounted(() => {
  load()
})
</script>
