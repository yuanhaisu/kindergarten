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
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row as { id: number })">编辑</el-button>
          <el-button link type="primary" @click="openStarJar(row as { id: number; name: string })">
            星星罐
          </el-button>
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
        <el-form-item label="班级老师">
          <div class="w-full">
            <el-select
              v-model="selectedTeacherIds"
              multiple
              filterable
              placeholder="选择老师"
              class="w-full"
              @change="syncTeacherRows"
            >
              <el-option v-for="t in teacherOptions" :key="t.id" :label="t.nickname" :value="t.id" />
            </el-select>
            <div
              v-for="row in form.teacher_ids"
              :key="row.user_id"
              class="flex items-center gap-2 mt-2"
            >
              <span class="text-sm text-slate-600 flex-1 truncate">
                {{ teacherName(row.user_id) }}
              </span>
              <el-select v-model="row.title" size="small" class="!w-28">
                <el-option label="班主任" value="head" />
                <el-option label="配班" value="assistant" />
                <el-option label="保育员" value="nursery" />
              </el-select>
            </div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="starJarVisible" title="班级星星罐" width="480px" destroy-on-close>
      <div v-if="starJar" class="space-y-3">
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="班级">{{ starJarClassName }}</el-descriptions-item>
          <el-descriptions-item label="学期">{{ starJar.term }}</el-descriptions-item>
          <el-descriptions-item label="学期起止">
            {{ starJar.term_start }} ~ {{ starJar.term_end }}
          </el-descriptions-item>
          <el-descriptions-item label="当前状态">
            <el-tag :type="starJar.configured ? 'success' : 'info'" size="small">
              {{ starJar.configured ? `已配置 ${starJar.goal} 颗` : '未配置' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="starJar.updated_at" label="更新时间">
            {{ formatDateTime(starJar.updated_at) }}
          </el-descriptions-item>
        </el-descriptions>
        <el-form label-width="96px">
          <el-form-item label="星星目标" required>
            <el-input-number v-model="starJarGoal" :min="1" :step="5" />
            <span class="ml-2 text-xs text-text-hint">必须大于 0；按班级罐统计的任务会使用该目标</span>
          </el-form-item>
        </el-form>
      </div>
      <div v-else-if="starJarLoading" class="text-sm text-text-muted">加载中…</div>
      <template #footer>
        <el-button @click="starJarVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="starJarSaving"
          :disabled="!starJar || starJarGoal <= 0"
          @click="saveStarJar"
        >
          保存
        </el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type { ClassItem, ClassStarJar, TeacherItem } from '~/types/api'
import { createClass, deleteClass, getClasses, getClassDetail, updateClass } from '~/api/classes'
import { getTeachers } from '~/api/teachers'
import { getClassStarJar, setClassStarJar } from '~/api/tasks'
import { formatDateTime } from '~/utils/format'

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

/**
 * 后端 teacher_ids 的元素是 {user_id, title} 而非裸 id —— title 必填
 * （见 server/internal/service/class_service.go 的 classTeacherAssign）。
 * 传裸 id 数组会反序列化失败；传空数组则会被 ReplaceClassTeachers 清空该班老师。
 */
type TeacherAssign = { user_id: number; title: string }

const form = reactive({
  name: '',
  grade: '小班',
  academic_year: '',
  capacity: 30,
  teacher_ids: [] as TeacherAssign[],
})

const teacherOptions = ref<TeacherItem[]>([])
const selectedTeacherIds = ref<number[]>([])

async function loadTeachers() {
  try {
    const res = await getTeachers({ page: 1, page_size: 100, status: 1 })
    teacherOptions.value = res.list
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载老师列表失败')
  }
}

/** 把选中的 id 同步成行对象，保留已选职务，新加入的默认班主任。 */
function syncTeacherRows(ids: number[]) {
  const prevTitle = new Map(form.teacher_ids.map((r) => [r.user_id, r.title]))
  form.teacher_ids = ids.map((id) => ({ user_id: id, title: prevTitle.get(id) ?? 'head' }))
}

function teacherName(id: number) {
  return teacherOptions.value.find((t) => t.id === id)?.nickname ?? `#${id}`
}

function openCreate() {
  editId.value = null
  Object.assign(form, {
    name: '',
    grade: '小班',
    academic_year: '',
    capacity: 30,
    teacher_ids: [],
  })
  selectedTeacherIds.value = []
  dialogVisible.value = true
}

async function openEdit(row: { id: number }) {
  editId.value = row.id
  try {
    const d = await getClassDetail(row.id)
    // 详情已带回 teachers，必须原样回填：UpdateClass 会无条件 ReplaceClassTeachers，
    // 不回填就等于把该班老师全部清空。
    const assigns: TeacherAssign[] = d.teachers.map((t) => ({
      user_id: t.id,
      title: t.title,
    }))
    Object.assign(form, {
      name: d.name,
      grade: d.grade,
      academic_year: d.academic_year,
      capacity: d.capacity,
      teacher_ids: assigns,
    })
    selectedTeacherIds.value = assigns.map((a) => a.user_id)
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
        teacher_ids: form.teacher_ids,
      })
    } else {
      await createClass({
        name: form.name,
        grade: form.grade,
        academic_year: form.academic_year,
        capacity: form.capacity,
        teacher_ids: form.teacher_ids,
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

// ── 星星罐（当前学期目标，tasks 7.5）──
const starJarVisible = ref(false)
const starJarLoading = ref(false)
const starJarSaving = ref(false)
const starJar = ref<ClassStarJar | null>(null)
const starJarGoal = ref(1)
const starJarClassId = ref(0)
const starJarClassName = ref('')

/** 连点不同班级的「星星罐」时，慢的那个后返回会盖掉新选中的班级 */
let starJarSeq = 0

async function openStarJar(row: { id: number; name: string }) {
  const seq = ++starJarSeq
  starJarVisible.value = true
  starJarLoading.value = true
  starJar.value = null
  starJarClassId.value = row.id
  starJarClassName.value = row.name
  try {
    const jar = await getClassStarJar(row.id)
    if (seq !== starJarSeq) return
    starJar.value = jar
    // 未配置时给一个可用的起点：0 会被 PUT 拒绝（目标必须 > 0）。
    starJarGoal.value = jar.configured && jar.goal > 0 ? jar.goal : 10
  } catch (e: unknown) {
    if (seq !== starJarSeq) return
    ElMessage.error(e instanceof Error ? e.message : '加载班级星星罐失败')
  } finally {
    if (seq === starJarSeq) starJarLoading.value = false
  }
}

async function saveStarJar() {
  if (starJarGoal.value <= 0) {
    ElMessage.warning('星星目标必须大于 0')
    return
  }
  starJarSaving.value = true
  try {
    starJar.value = await setClassStarJar(starJarClassId.value, starJarGoal.value)
    ElMessage.success('已保存班级星星罐目标')
    starJarVisible.value = false
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    starJarSaving.value = false
  }
}

onMounted(() => {
  load()
  loadTeachers()
})
</script>
