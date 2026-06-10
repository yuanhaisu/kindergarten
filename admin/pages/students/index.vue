<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap justify-between gap-2">
        <span>学生管理</span>
        <div class="flex gap-2">
          <el-upload
            :auto-upload="false"
            :show-file-list="false"
            accept=".xlsx,.xls"
            :on-change="onImportFile"
          >
            <el-button>导入 Excel</el-button>
          </el-upload>
          <el-button @click="doExport">导出名册</el-button>
          <el-button type="primary" @click="openCreate">添加学生</el-button>
        </div>
      </div>
    </template>
    <SearchBar v-model:keyword="keyword" @search="() => { resetPage(); load() }" @reset="() => { resetPage(); load() }" />
    <el-select v-model="status" placeholder="状态" clearable class="!w-28 mb-4 mr-2" @change="load">
      <el-option label="在读" :value="1" />
      <el-option label="毕业" :value="2" />
      <el-option label="退学" :value="3" />
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
      <el-table-column prop="name" label="姓名" width="100" />
      <el-table-column prop="student_no" label="学号" width="110" />
      <el-table-column prop="class_name" label="班级" />
      <el-table-column prop="status_label" label="状态" width="90" />
      <el-table-column prop="enroll_date" label="入园日" width="120" />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail((row as { id: number }).id)">详情</el-button>
          <el-button link @click="openEdit(row as { id: number })">编辑</el-button>
        </template>
      </el-table-column>
    </DataTable>
    <el-dialog v-model="detailVisible" title="学生详情" width="560px" destroy-on-close>
      <el-descriptions v-if="detail" :column="1" border size="small">
        <el-descriptions-item label="姓名">{{ detail.name }}</el-descriptions-item>
        <el-descriptions-item label="学号">{{ detail.student_no }}</el-descriptions-item>
        <el-descriptions-item label="班级">{{ detail.class_name }}</el-descriptions-item>
        <el-descriptions-item label="家长">
          <div v-for="p in detail.parents" :key="p.user_id" class="text-sm">
            {{ p.relation_label }} {{ p.name }} {{ p.phone }}
          </div>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
    <el-dialog v-model="formVisible" title="添加学生" width="520px">
      <el-form :model="studentForm" label-width="90px">
        <el-form-item label="姓名"><el-input v-model="studentForm.name" /></el-form-item>
        <el-form-item label="性别">
          <el-radio-group v-model="studentForm.gender">
            <el-radio :label="1">男</el-radio>
            <el-radio :label="2">女</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="生日">
          <el-date-picker v-model="studentForm.birthday" type="date" placeholder="选择日期" value-format="YYYY-MM-DD" class="!w-full" />
        </el-form-item>
        <el-form-item label="班级">
          <el-select v-model="studentForm.class_id" class="!w-full" placeholder="请选择班级">
            <el-option v-for="c in classList" :key="c.id" :label="`${c.name} (${c.grade})`" :value="c.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveStudent">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type { StudentDetail, StudentItem } from '~/types/api'
import { downloadBlob } from '~/utils/export'
import { getClasses } from '~/api/classes'
import {
  createStudent,
  exportStudentsBlob,
  getStudent,
  getStudents,
  importStudents,
} from '~/api/students'

const { page, pageSize, total, resetPage } = usePagination()
const loading = ref(false)
const list = ref<StudentItem[]>([])
const keyword = ref('')
const status = ref<number | undefined>()

async function load() {
  loading.value = true
  try {
    const res = await getStudents({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value || undefined,
      status: status.value,
    })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const detailVisible = ref(false)
const detail = ref<StudentDetail | null>(null)
async function openDetail(id: number) {
  try {
    detail.value = await getStudent(id)
    detailVisible.value = true
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}

const classList = ref<{ id: number; name: string; grade: string }[]>([])
const formVisible = ref(false)
const saving = ref(false)
const studentForm = reactive({
  name: '',
  gender: 1,
  birthday: '',
  class_id: 1,
  enroll_date: '',
  parents: [] as { name: string; phone: string; relation: string; is_primary: boolean; id_card_number: string }[],
})

async function openCreate() {
  Object.assign(studentForm, {
    name: '',
    gender: 1,
    birthday: '',
    class_id: null as unknown as number,
    enroll_date: new Date().toISOString().slice(0, 10),
    parents: [],
  })
  try {
    const res = await getClasses({ page: 1, page_size: 200 })
    classList.value = (res.list || []) as unknown as { id: number; name: string; grade: string }[]
    if (classList.value.length > 0 && !studentForm.class_id) {
      studentForm.class_id = classList.value[0].id
    }
  } catch { classList.value = [] }
  formVisible.value = true
}

function openEdit(row: { id: number }) {
  openDetail(row.id)
}

async function saveStudent() {
  saving.value = true
  try {
    await createStudent({
      ...studentForm,
      parents:
        studentForm.parents.length > 0
          ? studentForm.parents
          : [{ name: '家长', phone: '13800000000', relation: 'father', is_primary: true, id_card_number: '' }],
    })
    ElMessage.success('已创建')
    formVisible.value = false
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function onImportFile(uploadFile: { raw?: File }) {
  const f = uploadFile.raw
  if (!f) return
  try {
    const r = await importStudents(f)
    ElMessage.success(`导入完成：成功 ${r.success_count}，失败 ${r.fail_count}`)
    load()
  } catch (err: unknown) {
    ElMessage.error(err instanceof Error ? err.message : '导入失败')
  }
}

async function doExport() {
  try {
    const blob = await exportStudentsBlob({ status: status.value, format: 'xlsx' })
    downloadBlob(blob, '学生名册.xlsx')
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '导出失败')
  }
}

onMounted(load)
</script>
