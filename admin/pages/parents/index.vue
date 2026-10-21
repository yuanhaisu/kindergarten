<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex justify-between">
        <span>家长管理</span>
        <el-button type="primary" @click="openDialog">添加家长</el-button>
      </div>
    </template>
    <SearchBar v-model:keyword="keyword" @search="() => { resetPage(); load() }" @reset="() => { resetPage(); load() }" />
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
      <el-table-column label="孩子" min-width="200">
        <template #default="{ row }">
          {{
            (row as { children?: { student_name: string; class_name: string }[] }).children
              ?.map((c) => `${c.student_name}(${c.class_name})`)
              .join('、') || '-'
          }}
        </template>
      </el-table-column>
      <el-table-column prop="status_label" label="状态" width="90" />
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-button link type="primary" @click="detail((row as { id: number }).id)">详情</el-button>
        </template>
      </el-table-column>
    </DataTable>
    <el-dialog v-model="visible" title="添加家长" width="480px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="真实姓名"><el-input v-model="form.real_name" /></el-form-item>
        <el-form-item label="昵称"><el-input v-model="form.nickname" /></el-form-item>
        <el-form-item label="手机"><el-input v-model="form.phone" /></el-form-item>
        <el-form-item label="班级">
          <el-select v-model="bindClassId" class="!w-full" placeholder="请选择班级" @change="onClassChange">
            <el-option v-for="c in classList" :key="c.id" :label="`${c.name}(${c.grade})`" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="学生">
          <el-select v-model="bindStudentId" class="!w-full" placeholder="请先选择班级" :disabled="!bindClassId">
            <el-option v-for="s in studentList" :key="s.id" :label="`${s.name}(${s.student_no})`" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="关系">
          <el-select v-model="bindRelation" class="!w-full">
            <el-option label="爸爸" value="father" />
            <el-option label="妈妈" value="mother" />
            <el-option label="爷爷" value="grandfather" />
            <el-option label="奶奶" value="grandmother" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="主要联系人">
          <el-switch v-model="bindIsPrimary" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-dialog>
    <el-drawer v-model="drawer" title="家长详情" size="400px">
      <pre v-if="detailJson" class="text-xs bg-slate-50 p-3 rounded overflow-auto">{{ detailJson }}</pre>
    </el-drawer>
  </el-card>
</template>

<script setup lang="ts">
import type { ParentItem } from '~/types/api'
import { createParent, getParent, getParents } from '~/api/parents'
import { getClasses } from '~/api/classes'
import { getStudents } from '~/api/students'

const { page, pageSize, total, resetPage } = usePagination()
const loading = ref(false)
const list = ref<ParentItem[]>([])
const route = useRoute()
// 顶栏人员速查跳转过来时预填关键词（/parents?keyword=xxx）
const keyword = ref(typeof route.query.keyword === 'string' ? route.query.keyword : '')

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
    const res = await getParents({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value || undefined,
    })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const visible = ref(false)
const saving = ref(false)
const classList = ref<{ id: number; name: string; grade: string }[]>([])
const studentList = ref<{ id: number; name: string; student_no: string }[]>([])
const bindClassId = ref<number | null>(null)
const bindStudentId = ref<number | null>(null)
const bindRelation = ref('mother')
const bindIsPrimary = ref(true)
const form = reactive({
  real_name: '',
  nickname: '',
  phone: '',
  gender: 2,
  student_bindings: [] as { student_id: number; relation: string; is_primary: boolean }[],
})

async function openDialog() {
  bindClassId.value = null
  bindStudentId.value = null
  studentList.value = []
  // 加载班级列表
  try {
    const res = await getClasses({ page: 1, page_size: 200 })
    classList.value = (res.list || []) as unknown as { id: number; name: string; grade: string }[]
  } catch { /* keep empty */ }
  visible.value = true
}

async function onClassChange(classId: number | null) {
  bindStudentId.value = null
  studentList.value = []
  if (!classId) return
  try {
    const res = await getStudents({ class_id: classId, page: 1, page_size: 200 })
    studentList.value = ((res.list || []) as unknown as { id: number; name: string; student_no: string }[])
  } catch { /* keep empty */ }
}

async function submit() {
  if (!bindStudentId.value) {
    ElMessage.warning('请选择学生')
    return
  }
  saving.value = true
  try {
    await createParent({
      ...form,
      student_bindings: [{ student_id: bindStudentId.value, relation: bindRelation.value, is_primary: bindIsPrimary.value }],
    })
    ElMessage.success('已添加')
    visible.value = false
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  } finally {
    saving.value = false
  }
}

const drawer = ref(false)
const detailJson = ref('')
async function detail(id: number) {
  try {
    const d = await getParent(id)
    detailJson.value = JSON.stringify(d, null, 2)
    drawer.value = true
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}

onMounted(load)
</script>
