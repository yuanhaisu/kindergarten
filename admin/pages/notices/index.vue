<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span>通知管理</span>
        <div class="flex flex-wrap items-center gap-2">
          <el-input
            v-model="keyword"
            clearable
            placeholder="标题关键词"
            style="width: 180px"
            @clear="load"
            @keyup.enter="load"
          />
          <el-button type="primary" @click="openCreate">发布通知</el-button>
        </div>
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
      <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip />
      <el-table-column prop="category_label" label="分类" width="90" />
      <el-table-column label="发布范围" min-width="140">
        <template #default="{ row }">
          {{ ((row as unknown as NoticeItem).class_names || []).join('、') || '—' }}
        </template>
      </el-table-column>
      <el-table-column prop="read_rate" label="已读率" width="90" />
      <el-table-column prop="status_label" label="状态" width="100" />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit((row as { id: number }).id)">编辑</el-button>
          <el-button link type="primary" @click="pass((row as { id: number }).id)">审核通过</el-button>
          <el-button link type="danger" @click="del((row as { id: number }).id)">删除</el-button>
        </template>
      </el-table-column>
    </DataTable>

    <el-dialog v-model="dialogVisible" :title="editId ? '编辑通知' : '发布通知'" width="520px" destroy-on-close>
      <el-form :model="form" label-width="96px">
        <el-form-item label="发布范围" required>
          <div class="w-full">
            <div class="mb-2 flex flex-wrap gap-2">
              <el-check-tag
                v-for="g in gradeOptions"
                :key="g"
                :checked="form.grades.includes(g)"
                @change="toggleGrade(g)"
              >
                {{ g }}
              </el-check-tag>
            </div>
            <el-select
              v-model="form.class_ids"
              multiple
              filterable
              collapse-tags
              collapse-tags-tooltip
              placeholder="点选班级（可与年级圈选混用）"
              style="width: 100%"
            >
              <el-option v-for="c in classOptions" :key="c.id" :label="c.name" :value="c.id" />
            </el-select>
          </div>
        </el-form-item>
        <el-form-item label="分类" required>
          <el-select v-model="form.category" placeholder="选择分类" style="width: 100%">
            <el-option v-for="o in categoryOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="标题" required>
          <el-input v-model="form.title" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="正文">
          <el-input v-model="form.content" type="textarea" :rows="5" />
        </el-form-item>
        <el-form-item label="需家长确认">
          <el-switch v-model="form.need_confirm" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type { ClassItem, NoticeItem } from '~/types/api'
import { getClasses } from '~/api/classes'
import {
  auditNotice,
  createNotice,
  deleteNotice,
  getNotice,
  getNotices,
  updateNotice,
} from '~/api/notices'

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<NoticeItem[]>([])
const keyword = ref('')

const classOptions = ref<ClassItem[]>([])
const categoryOptions = [
  { value: 'homework', label: '作业' },
  { value: 'bring', label: '带物' },
  { value: 'activity', label: '活动' },
  { value: 'health', label: '健康' },
  { value: 'other', label: '其他' },
]

async function loadClassOptions() {
  try {
    const res = await getClasses({ page: 1, page_size: 200 })
    classOptions.value = res.list
  } catch {
    classOptions.value = []
  }
}

async function load() {
  loading.value = true
  try {
    const res = await getNotices({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function pass(id: number) {
  try {
    await auditNotice(id, { action: 'pass' })
    ElMessage.success('已通过')
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  }
}

async function del(id: number) {
  try {
    await ElMessageBox.confirm('确定删除？', '提示', { type: 'warning' })
    await deleteNotice(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* */
  }
}

const dialogVisible = ref(false)
const editId = ref<number | null>(null)
const saving = ref(false)
// 年级取值溯源 class_service.validateGrade（小班/中班/大班），不新增枚举
const gradeOptions = ['小班', '中班', '大班']
const form = reactive({
  class_ids: [] as number[],
  grades: [] as string[],
  category: 'homework',
  title: '',
  content: '',
  need_confirm: false,
})

function toggleGrade(g: string) {
  const i = form.grades.indexOf(g)
  if (i >= 0) form.grades.splice(i, 1)
  else form.grades.push(g)
}

function resetForm() {
  form.class_ids = classOptions.value[0] ? [classOptions.value[0].id] : []
  form.grades = []
  form.category = 'homework'
  form.title = ''
  form.content = ''
  form.need_confirm = false
}

function openCreate() {
  editId.value = null
  resetForm()
  dialogVisible.value = true
}

async function openEdit(id: number) {
  editId.value = id
  try {
    const d = await getNotice(id)
    // 详情只回 class_names（目标集），按名回填勾选
    form.class_ids = classOptions.value
      .filter((c) => (d.class_names || []).includes(c.name))
      .map((c) => c.id)
    form.grades = []
    form.category = d.category
    form.title = d.title
    form.content = d.content ?? ''
    form.need_confirm = !!d.need_confirm
    dialogVisible.value = true
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载详情失败')
  }
}

async function submitForm() {
  if (form.class_ids.length === 0 && form.grades.length === 0) {
    ElMessage.warning('请选择班级或圈选年级')
    return
  }
  if (!form.title.trim()) {
    ElMessage.warning('请填写标题')
    return
  }
  saving.value = true
  try {
    const body = {
      class_ids: form.class_ids,
      grades: form.grades,
      category: form.category,
      title: form.title.trim(),
      content: form.content,
      need_confirm: form.need_confirm,
    }
    if (editId.value) {
      await updateNotice(editId.value, body)
      ElMessage.success('已更新')
    } else {
      await createNotice(body)
      ElMessage.success('已发布')
    }
    dialogVisible.value = false
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await loadClassOptions()
  await load()
})
</script>
