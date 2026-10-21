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
      <el-table-column label="人脸底库" width="100">
        <template #default="{ row }">
          <el-tag :type="faceTagType((row as StudentItem).face_status)" size="small" effect="light">
            {{ (row as StudentItem).face_status_label }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="enroll_date" label="入园日" width="120" />
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail((row as { id: number }).id)">详情</el-button>
          <el-button link @click="openEdit(row as { id: number })">编辑</el-button>
          <el-button link @click="openFaces(row as StudentItem)">底库</el-button>
        </template>
      </el-table-column>
    </DataTable>

    <el-dialog v-model="facesVisible" title="人脸底库照片" width="600px" destroy-on-close>
      <div v-if="faces" class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2 text-sm">
            <span>注册状态：</span>
            <el-tag :type="faceTagType(faces.status)" size="small">{{ faces.status_label }}</el-tag>
            <span class="text-text-muted">{{ faces.total }}/{{ faces.max_count }} 张</span>
          </div>
          <div class="flex items-center gap-2">
            <el-upload
              :auto-upload="false"
              :show-file-list="false"
              accept="image/*"
              multiple
              :disabled="!canAddFaces"
              :on-change="onFacePicked"
            >
              <el-button size="small" :disabled="!canAddFaces" :loading="facesBusy">上传照片</el-button>
            </el-upload>
            <el-button v-if="faces.status === 2" size="small" :loading="facesBusy" @click="doRetryFaces">
              重试注册
            </el-button>
          </div>
        </div>
        <el-alert
          v-if="!faces.enabled"
          type="info"
          :closable="false"
          title="本园所未开启人脸识别，暂不能录入底库照片"
        />
        <div v-if="faces.total === 0" class="text-text-muted text-sm py-6 text-center">
          尚未录入人脸照片。上传 1-5 张正面清晰照片后自动注册到识别底库。
        </div>
        <div v-else class="grid grid-cols-3 gap-3">
          <div
            v-for="f in faces.list"
            :key="f.id"
            class="relative border border-border-hairline rounded-lg overflow-hidden bg-bg-card"
          >
            <img :src="f.face_url" class="w-full h-28 object-cover" alt="底库照片" />
            <div class="p-2 space-y-1">
              <el-tag :type="faceTagType(f.status)" size="small">{{ f.status_label }}</el-tag>
              <div v-if="f.last_error" class="text-xs text-semantic-danger truncate" :title="f.last_error">
                {{ f.last_error }}
              </div>
            </div>
            <el-button
              class="!absolute top-1 right-1"
              size="small"
              type="danger"
              circle
              :icon="Delete"
              :loading="facesBusy"
              @click="removeFace(f)"
            />
          </div>
        </div>
      </div>
    </el-dialog>
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
import { Delete } from '@element-plus/icons-vue'
import type { StudentDetail, StudentFace, StudentFacesResult, StudentItem } from '~/types/api'
import { downloadBlob } from '~/utils/export'
import { getClasses } from '~/api/classes'
import { uploadFile } from '~/api/common'
import {
  addStudentFaces,
  createStudent,
  deleteStudentFace,
  exportStudentsBlob,
  getStudent,
  getStudentFaces,
  getStudents,
  importStudents,
  retryStudentFaces,
} from '~/api/students'

const { page, pageSize, total, resetPage } = usePagination()
const loading = ref(false)
const list = ref<StudentItem[]>([])
const route = useRoute()
// 顶栏人员速查跳转过来时预填关键词（/students?keyword=xxx）
const keyword = ref(typeof route.query.keyword === 'string' ? route.query.keyword : '')
const status = ref<number | undefined>()

// 同页再次速查时组件被复用，须跟随 query 更新并重查
watch(
  () => route.query.keyword,
  (kw) => {
    const next = typeof kw === 'string' ? kw : ''
    if (next !== keyword.value) {
      keyword.value = next
      resetPage()
      load()
    }
  },
)

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

// ── 人脸底库 ────────────────────────────────────────────────────────
// face_status: 0未录入 1已注册 2注册失败 3注册中
function faceTagType(status: number) {
  switch (status) {
    case 1:
      return 'success'
    case 2:
      return 'danger'
    case 3:
      return 'warning'
    default:
      return 'info'
  }
}

const facesVisible = ref(false)
const faces = ref<StudentFacesResult | null>(null)
const facesBusy = ref(false)
const faceStudentId = ref(0)
const canAddFaces = computed(
  () => !!faces.value && faces.value.enabled && faces.value.total < faces.value.max_count,
)

async function openFaces(row: StudentItem) {
  faceStudentId.value = row.id
  facesVisible.value = true
  await refreshFaces()
}

async function refreshFaces() {
  try {
    faces.value = await getStudentFaces(faceStudentId.value)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载底库失败')
  }
}

// el-upload 多选时逐文件触发 on-change；用微任务聚合一次批次，避免每张各发一次注册调用
let pendingFaceFiles: File[] = []
let flushScheduled = false
function onFacePicked(uploadFile: { raw?: File }) {
  const f = uploadFile.raw
  if (!f) return
  pendingFaceFiles.push(f)
  if (!flushScheduled) {
    flushScheduled = true
    // 同一次选择触发的多个 on-change 在本轮同步执行完毕后才轮到微任务
    queueMicrotask(flushFaceUploads)
  }
}

async function flushFaceUploads() {
  flushScheduled = false
  const files = pendingFaceFiles
  pendingFaceFiles = []
  if (files.length === 0) return
  facesBusy.value = true
  try {
    const uploaded = await Promise.all(files.map((f) => uploadFile(f, 'face')))
    faces.value = await addStudentFaces(
      faceStudentId.value,
      uploaded.map((u) => u.id),
    )
    ElMessage.success('已提交注册')
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '上传失败')
    await refreshFaces()
  } finally {
    facesBusy.value = false
  }
}

async function doRetryFaces() {
  facesBusy.value = true
  try {
    faces.value = await retryStudentFaces(faceStudentId.value)
    ElMessage.success('已重试注册')
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '重试失败')
  } finally {
    facesBusy.value = false
  }
}

async function removeFace(f: StudentFace) {
  try {
    await deleteStudentFace(faceStudentId.value, f.id)
    ElMessage.success('已移除')
    await refreshFaces()
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '移除失败')
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
