<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex justify-between">
        <span>园所公告</span>
        <el-button type="primary" @click="openCreate">发布公告</el-button>
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
      <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
      <el-table-column prop="status_label" label="状态" width="100" />
      <el-table-column prop="view_count" label="浏览" width="80" />
      <el-table-column label="操作" width="260">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail((row as { id: number }).id)">详情</el-button>
          <el-button link type="primary" @click="openEdit((row as { id: number }).id)">编辑</el-button>
          <el-button link @click="toggleTop((row as { id: number; is_top: boolean }).id, !(row as { is_top: boolean }).is_top)">
            {{ (row as { is_top: boolean }).is_top ? '取消置顶' : '置顶' }}
          </el-button>
          <el-button link type="danger" @click="remove((row as { id: number }).id)">删除</el-button>
        </template>
      </el-table-column>
    </DataTable>
    <el-dialog v-model="visible" :title="editId ? '编辑公告' : '发布公告'" width="560px" destroy-on-close>
      <el-form :model="form" label-width="100px">
        <el-form-item label="标题"><el-input v-model="form.title" /></el-form-item>
        <el-form-item label="内容"><el-input v-model="form.content" type="textarea" :rows="6" /></el-form-item>
        <el-form-item label="封面URL"><el-input v-model="form.cover_url" /></el-form-item>
        <el-form-item label="重要">
          <el-switch v-model="form.is_important" />
        </el-form-item>
        <el-form-item label="置顶">
          <el-switch v-model="form.is_top" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">
          {{ editId ? '保存' : '发布' }}
        </el-button>
      </template>
    </el-dialog>
    <el-dialog v-model="detailVisible" title="公告详情" width="640px" destroy-on-close>
      <div v-loading="detailLoading" class="space-y-3">
        <template v-if="detail">
          <div>
            <div class="text-base font-semibold text-slate-800">{{ detail.title }}</div>
            <div class="mt-1 text-xs text-slate-500">
              <span class="mr-3">状态：{{ getStatusLabel(detail.status) }}</span>
              <span class="mr-3">浏览：{{ detail.view_count }}</span>
              <span>发布：{{ detail.published_at || '-' }}</span>
            </div>
          </div>
          <div class="rounded bg-slate-50 p-3 text-sm leading-6 text-slate-700 whitespace-pre-wrap">
            {{ detail.content || '暂无内容' }}
          </div>
          <div class="text-xs text-slate-500">
            创建人：{{ detail.creator?.nickname || '-' }} · 创建时间：{{ detail.created_at || '-' }}
          </div>
        </template>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type { AnnouncementDetail, AnnouncementItem } from '~/types/api'
import {
  createAnnouncement,
  deleteAnnouncement,
  getAnnouncement,
  getAnnouncements,
  setAnnouncementTop,
  updateAnnouncement,
} from '~/api/announcements'

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<AnnouncementItem[]>([])
const editId = ref<number | null>(null)
const saving = ref(false)
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<AnnouncementDetail | null>(null)

async function load() {
  loading.value = true
  try {
    const res = await getAnnouncements({ page: page.value, page_size: pageSize.value })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const visible = ref(false)
const form = reactive({
  title: '',
  content: '',
  cover_url: '',
  is_important: false,
  is_top: false,
  target_roles: ['parent', 'teacher'] as string[],
  attachments: [] as unknown[],
})

function openCreate() {
  editId.value = null
  Object.assign(form, {
    title: '',
    content: '',
    cover_url: '',
    is_important: false,
    is_top: false,
    target_roles: ['parent', 'teacher'],
    attachments: [],
  })
  visible.value = true
}

async function openEdit(id: number) {
  editId.value = id
  try {
    const res = await getAnnouncement(id)
    Object.assign(form, {
      title: res.title ?? '',
      content: res.content ?? '',
      cover_url: res.cover_url ?? '',
      is_important: !!res.is_important,
      is_top: !!res.is_top,
      target_roles: (res.target_roles?.length ? res.target_roles : ['parent', 'teacher']) as string[],
      attachments: res.attachments ?? [],
    })
    visible.value = true
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载详情失败')
  }
}

function getStatusLabel(status: number) {
  if (status === 2) return '已发布'
  if (status === 1) return '待审核'
  if (status === 3) return '已驳回'
  return `状态${status}`
}

async function openDetail(id: number) {
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    detail.value = await getAnnouncement(id)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载详情失败')
    detailVisible.value = false
  } finally {
    detailLoading.value = false
  }
}

async function submit() {
  if (!form.title.trim()) {
    ElMessage.warning('请填写标题')
    return
  }
  saving.value = true
  try {
    if (editId.value) {
      await updateAnnouncement(editId.value, { ...form, title: form.title.trim(), scheduled_at: null })
      ElMessage.success('已更新')
    } else {
      await createAnnouncement({ ...form, title: form.title.trim(), scheduled_at: null })
      ElMessage.success('已发布')
    }
    visible.value = false
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  } finally {
    saving.value = false
  }
}

async function toggleTop(id: number, is_top: boolean) {
  try {
    await setAnnouncementTop(id, { is_top })
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  }
}

async function remove(id: number) {
  try {
    await ElMessageBox.confirm('确定删除？', '提示', { type: 'warning' })
    await deleteAnnouncement(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* */
  }
}

onMounted(load)
</script>
