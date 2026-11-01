<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span>班级相册</span>
        <span class="text-sm text-text-muted">园长只读视图：可下架已发布相册，不提供创建或编辑</span>
      </div>
    </template>

    <div class="mb-4 flex flex-wrap items-center gap-2">
      <el-select
        v-model="filters.classId"
        placeholder="全部班级"
        clearable
        class="!w-44"
        @change="reload"
      >
        <el-option v-for="c in classOptions" :key="c.id" :label="c.name" :value="c.id" />
      </el-select>
      <el-select
        v-model="filters.status"
        placeholder="全部状态"
        clearable
        class="!w-32"
        @change="reload"
      >
        <el-option v-for="o in statusOptions" :key="o.value" :label="o.label" :value="o.value" />
      </el-select>
      <el-select
        v-model="filters.category"
        placeholder="全部分类"
        clearable
        class="!w-32"
        @change="reload"
      >
        <el-option v-for="o in categoryOptions" :key="o.value" :label="o.label" :value="o.value" />
      </el-select>
      <el-button @click="resetFilters">重置</el-button>
    </div>

    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="list as unknown as Record<string, unknown>[]"
      :loading="loading"
      :total="total"
      empty-text="暂无相册"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
      <el-table-column prop="class_name" label="班级" width="120" />
      <el-table-column prop="category_label" label="分类" width="90" />
      <el-table-column prop="album_date" label="日期" width="120" />
      <el-table-column prop="photo_count" label="照片数" width="90" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTagType((row as AlbumListItem).status)" size="small">
            {{ statusLabel((row as AlbumListItem).status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="发布时间" width="150">
        <template #default="{ row }">
          <span class="text-sm text-text-muted">
            {{ formatDateTime((row as AlbumListItem).published_at) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="90" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail((row as AlbumListItem).id)">详情</el-button>
        </template>
      </el-table-column>
    </DataTable>

    <el-drawer v-model="detailVisible" title="相册详情" size="560px" destroy-on-close>
      <div v-loading="detailLoading" class="min-h-[200px] space-y-4">
        <template v-if="detail">
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="标题">{{ detail.title }}</el-descriptions-item>
            <el-descriptions-item label="班级">{{ detail.class_name || '—' }}</el-descriptions-item>
            <el-descriptions-item label="分类">{{ detail.category_label }}</el-descriptions-item>
            <el-descriptions-item label="相册日期">{{ detail.album_date || '—' }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="statusTagType(detail.status)" size="small">
                {{ statusLabel(detail.status) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="照片数">{{ detail.photo_count }}</el-descriptions-item>
            <el-descriptions-item label="发布时间">
              {{ formatDateTime(detail.published_at) }}
            </el-descriptions-item>
            <el-descriptions-item label="创建老师">
              {{ detail.creator?.nickname || '—' }}
            </el-descriptions-item>
          </el-descriptions>

          <div>
            <div class="mb-2 text-sm font-semibold text-text-primary">参与幼儿</div>
            <div v-if="detail.participants.length" class="flex flex-wrap gap-2">
              <el-tag
                v-for="s in detail.participants"
                :key="s.id"
                size="small"
                effect="plain"
              >
                {{ s.name }}
              </el-tag>
            </div>
            <div v-else class="text-sm text-text-hint">未标记参与幼儿</div>
          </div>

          <div>
            <div class="mb-2 text-sm font-semibold text-text-primary">
              照片（{{ detail.photos.length }}）
            </div>
            <div v-if="detail.photos.length" class="grid grid-cols-3 gap-2">
              <div v-for="(p, index) in detail.photos" :key="p.id">
                <el-image
                  :src="p.thumb_url || p.file_url"
                  :preview-src-list="previewUrls"
                  :initial-index="previewIndex(index)"
                  preview-teleported
                  fit="cover"
                  class="h-24 w-full rounded-lg"
                >
                  <template #error>
                    <div class="flex h-full w-full items-center justify-center text-text-hint">
                      <el-icon :size="20"><Picture /></el-icon>
                    </div>
                  </template>
                </el-image>
                <div
                  v-if="p.students.length"
                  class="mt-1 truncate text-xs text-text-muted"
                  :title="studentNames(p)"
                >
                  {{ studentNames(p) }}
                </div>
              </div>
            </div>
            <el-empty v-else description="暂无照片" :image-size="72" />
          </div>
        </template>
        <el-empty
          v-else-if="!detailLoading"
          description="未加载到相册详情"
          :image-size="72"
        />
      </div>

      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button
          v-if="detail?.status === 'published'"
          type="danger"
          :loading="unpublishing"
          @click="unpublishCurrent"
        >
          下架相册
        </el-button>
      </template>
    </el-drawer>
  </el-card>
</template>

<script setup lang="ts">
import { Picture } from '@element-plus/icons-vue'
import type { ClassItem } from '~/types/api'
import { getClasses } from '~/api/classes'
import {
  getAlbumDetail,
  getAlbums,
  unpublishAlbum,
  type AlbumCategory,
  type AlbumDetail,
  type AlbumListItem,
  type AlbumPhoto,
  type AlbumStatus,
} from '~/api/albums'
import { formatDateTime } from '~/utils/format'

type TagType = 'primary' | 'success' | 'info' | 'warning' | 'danger'

const { page, pageSize, total, resetPage } = usePagination()
const loading = ref(false)
const list = ref<AlbumListItem[]>([])
const classOptions = ref<ClassItem[]>([])
const filters = reactive<{
  classId?: number
  status?: AlbumStatus
  category?: AlbumCategory
}>({})

// 筛选四项与契约 22-albums.md 一致；「全部」用清空选择表达（不传参）
const statusOptions: { value: AlbumStatus; label: string }[] = [
  { value: 'draft', label: '草稿' },
  { value: 'published', label: '已发布' },
  { value: 'unpublished', label: '已下架' },
]
const categoryOptions: { value: AlbumCategory; label: string }[] = [
  { value: 'activity', label: '活动' },
  { value: 'daily', label: '日常' },
  { value: 'work', label: '作品' },
  { value: 'festival', label: '节日' },
]

function statusLabel(status: AlbumStatus) {
  return status === 'published' ? '已发布' : status === 'draft' ? '草稿' : '已下架'
}

function statusTagType(status: AlbumStatus): TagType {
  return status === 'published' ? 'success' : status === 'draft' ? 'info' : 'warning'
}

async function loadClassOptions() {
  try {
    const res = await getClasses({ page: 1, page_size: 200 })
    classOptions.value = res.list
  } catch (e: unknown) {
    // 班级筛选是辅助能力：失败时保持空列表，不阻断相册主列表
    console.error('[albums] 班级列表加载失败', e)
    classOptions.value = []
  }
}

async function load() {
  loading.value = true
  try {
    const res = await getAlbums({
      page: page.value,
      page_size: pageSize.value,
      class_id: filters.classId,
      status: filters.status,
      category: filters.category,
    })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载相册列表失败')
  } finally {
    loading.value = false
  }
}

function reload() {
  resetPage()
  load()
}

function resetFilters() {
  filters.classId = undefined
  filters.status = undefined
  filters.category = undefined
  reload()
}

const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<AlbumDetail | null>(null)
const unpublishing = ref(false)

const previewUrls = computed(() =>
  (detail.value?.photos ?? [])
    .map((p) => p.file_url || p.thumb_url)
    .filter((url) => !!url),
)

// preview-src-list 会过滤掉无 URL 的照片；initial-index 必须按过滤后的位置算，
// 否则某张图缺 URL 时预览会从错误的一张开始。
function previewIndex(index: number) {
  const photos = detail.value?.photos ?? []
  let visible = 0
  for (let i = 0; i < index && i < photos.length; i += 1) {
    if (photos[i].file_url || photos[i].thumb_url) visible += 1
  }
  return visible
}

function studentNames(photo: AlbumPhoto) {
  return photo.students.map((s) => s.name).join('、')
}

async function openDetail(id: number) {
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    detail.value = await getAlbumDetail(id)
  } catch (e: unknown) {
    detailVisible.value = false
    ElMessage.error(e instanceof Error ? e.message : '加载相册详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function unpublishCurrent() {
  const current = detail.value
  if (!current || current.status !== 'published') return
  try {
    await ElMessageBox.confirm(
      '下架后该班家长将立即看不到这本相册，确定下架？',
      '下架相册',
      { type: 'warning', confirmButtonText: '下架', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  unpublishing.value = true
  try {
    const res = await unpublishAlbum(current.id)
    if (detail.value?.id === current.id) {
      detail.value = { ...detail.value, status: res.status, published_at: null }
    }
    ElMessage.success(res.changed ? '已下架' : '相册已处于下架状态')
    await load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '下架失败')
  } finally {
    unpublishing.value = false
  }
}

onMounted(async () => {
  await loadClassOptions()
  await load()
})
</script>
