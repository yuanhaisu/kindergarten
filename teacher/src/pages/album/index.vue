<script setup>
import { computed, reactive, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import {
  createTeacherAlbum,
  fetchTeacherAlbum,
  fetchTeacherAlbumStats,
  fetchTeacherAlbums,
  uploadAlbumPhoto
} from "@/api/teacher-albums"
import { ALBUM_CATEGORIES, ALBUM_FILTERS, albumCategoryLabel, albumTagClass } from "@/utils/albums"
import { formatDateAnchor, formatDateLabel, formatMonthLabel, monthKeyOf, todayKey } from "@/utils/time"
import { pickImages } from "@/utils/media"
import PageShell from "@/components/PageShell.vue"

const PAGE_SIZE = 20
const MAX_PHOTOS = 50
const MAX_PHOTO_BYTES = 20 * 1024 * 1024
const UPLOAD_TIMEOUT_MS = 30000
// 设计稿：当前月与最近一个月默认展开，更早月份折叠，点月份头可展开
const DEFAULT_OPEN_MONTHS = 2

const STAT_ITEMS = [
  { key: "total_photos", label: "照片总数", icon: "", tone: "brand" },
  { key: "monthly_new", label: "本月新增", icon: "", tone: "success" },
  { key: "album_count", label: "相册数", icon: "", tone: "info" }
]

const stats = ref({ total_photos: 0, monthly_new: 0, album_count: 0 })
const albums = ref([])
const total = ref(0)
const activeCategory = ref("")
const loading = ref(true)
const loadingMore = ref(false)
// 首次成功返回前不把空列表当「还没有照片」——错误态必须先于空态判定
const hasLoaded = ref(false)
const loadError = ref("")
const expandedMonths = ref(new Set())
const collapsedMonths = ref(new Set())
let loadSeq = 0

const viewState = computed(() => {
  if (loading.value && !hasLoaded.value) return "loading"
  if (loadError.value && !hasLoaded.value) return "error"
  if (hasLoaded.value && total.value === 0) {
    return activeCategory.value ? "filterEmpty" : "empty"
  }
  return "normal"
})

const monthGroups = computed(() => {
  const groups = new Map()
  for (const album of albums.value) {
    const key = monthKeyOf(album?.album_date) || "unknown"
    if (!groups.has(key)) {
      groups.set(key, { key, label: formatMonthLabel(key) || "未知月份", items: [] })
    }
    groups.get(key).items.push(album)
  }
  return Array.from(groups.values())
})

const hasMore = computed(() => hasLoaded.value && albums.value.length < total.value)

function normalizeCount(value) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? n : 0
}

function applyStats(data) {
  stats.value = {
    total_photos: normalizeCount(data?.total_photos),
    monthly_new: normalizeCount(data?.monthly_new),
    album_count: normalizeCount(data?.album_count)
  }
}

function listParams(page) {
  const params = { page, page_size: PAGE_SIZE }
  if (activeCategory.value) params.category = activeCategory.value
  return params
}

// B21：列表优先 cover_urls（最多 3 张，按 sort_num 升序）；缺失或为空时回退单张 cover_url。
// 没有封面时不造图，交给模板显示占位格。
function coverTilesOf(album) {
  const listed = Array.isArray(album?.cover_urls)
    ? album.cover_urls.map(url => String(url || "").trim()).filter(Boolean)
    : []
  if (listed.length) return listed.slice(0, 3)
  const legacy = String(album?.cover_url || "").trim()
  return legacy ? [legacy] : []
}

function normalizeAlbum(album) {
  return { ...album, coverTiles: coverTilesOf(album) }
}

function resetMonthState() {
  expandedMonths.value = new Set()
  collapsedMonths.value = new Set()
}

async function loadAlbums({ skeleton = true } = {}) {
  const seq = ++loadSeq
  if (skeleton) {
    loading.value = true
    hasLoaded.value = false
    loadError.value = ""
  }
  try {
    const [statsRes, listRes] = await Promise.all([
      fetchTeacherAlbumStats({ silent: true }),
      fetchTeacherAlbums(listParams(1), { silent: true })
    ])
    if (seq !== loadSeq) return
    applyStats(statsRes.data)
    albums.value = (Array.isArray(listRes.data?.list) ? listRes.data.list : []).map(normalizeAlbum)
    total.value = normalizeCount(listRes.data?.total)
    resetMonthState()
    loadError.value = ""
    hasLoaded.value = true
  } catch (err) {
    if (seq !== loadSeq) return
    if (!hasLoaded.value) {
      // 首次/切分类失败：整页错误态，空列表不能冒充「还没有照片」
      albums.value = []
      total.value = 0
      loadError.value = "相册加载失败"
    } else {
      uni.showToast({ title: "刷新失败，请重试", icon: "none" })
    }
    console.warn("[album-index] load albums failed:", err?.message || err)
  } finally {
    if (seq === loadSeq) {
      loading.value = false
    }
  }
}

function retryLoad() {
  loadAlbums({ skeleton: true })
}

function selectCategory(value) {
  if (value === activeCategory.value || loading.value) return
  activeCategory.value = value
  loadAlbums({ skeleton: true })
}

function isMonthOpen(group, index) {
  if (collapsedMonths.value.has(group.key)) return false
  if (expandedMonths.value.has(group.key)) return true
  return index < DEFAULT_OPEN_MONTHS
}

function toggleMonth(group, index) {
  const open = isMonthOpen(group, index)
  const expanded = new Set(expandedMonths.value)
  const collapsed = new Set(collapsedMonths.value)
  if (open) {
    expanded.delete(group.key)
    collapsed.add(group.key)
  } else {
    collapsed.delete(group.key)
    expanded.add(group.key)
  }
  expandedMonths.value = expanded
  collapsedMonths.value = collapsed
}

// 「查看全部」= 逐页把剩余相册取回来，而不是只展开 DOM
async function loadAll() {
  if (loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  const seq = loadSeq
  try {
    let nextPage = Math.floor(albums.value.length / PAGE_SIZE) + 1
    let guard = 0
    while (seq === loadSeq && albums.value.length < total.value && guard < 20) {
      const res = await fetchTeacherAlbums(listParams(nextPage), { silent: true })
      if (seq !== loadSeq) return
      const list = Array.isArray(res.data?.list) ? res.data.list : []
      if (!list.length) break
      const seen = new Set(albums.value.map(item => item.id))
      albums.value = albums.value.concat(
        list.filter(item => !seen.has(item.id)).map(normalizeAlbum)
      )
      total.value = normalizeCount(res.data?.total) || total.value
      nextPage += 1
      guard += 1
    }
  } catch (err) {
    uni.showToast({ title: err?.message || "加载更多失败", icon: "none" })
    console.warn("[album-index] load more failed:", err?.message || err)
  } finally {
    loadingMore.value = false
  }
}

// ---------------- 大图浏览 ----------------
const viewerOpen = ref(false)
const viewerAlbum = ref(null)
const viewerPhotos = ref([])
const viewerIndex = ref(0)
const viewerLoading = ref(false)
const viewerError = ref("")
let viewerSeq = 0
let touchStartX = 0
let touchStartY = 0

const viewerTitle = computed(() => viewerAlbum.value?.title || "相册")
const viewerDateText = computed(() => formatDateLabel(viewerAlbum.value?.album_date))
const viewerTotal = computed(
  () => viewerPhotos.value.length || normalizeCount(viewerAlbum.value?.photo_count)
)
const currentPhoto = computed(() => viewerPhotos.value[viewerIndex.value] || null)
const canPrev = computed(
  () => !viewerLoading.value && !viewerError.value && viewerIndex.value > 0
)
const canNext = computed(
  () =>
    !viewerLoading.value &&
    !viewerError.value &&
    viewerIndex.value < viewerTotal.value - 1
)

function fallbackPhotos(album) {
  const url = String(album?.cover_url || "").trim()
  return url ? [{ id: 0, file_url: url, thumb_url: url }] : []
}

async function openViewer(album, tileIndex = 0) {
  if (!album?.id) return
  closeUpload()
  viewerAlbum.value = album
  viewerOpen.value = true
  viewerPhotos.value = []
  viewerIndex.value = 0
  viewerLoading.value = true
  viewerError.value = ""
  const seq = ++viewerSeq
  try {
    const res = await fetchTeacherAlbum(album.id, { silent: true })
    if (seq !== viewerSeq) return
    const photos = Array.isArray(res.data?.photos)
      ? res.data.photos.filter(photo => photo?.file_url)
      : []
    viewerPhotos.value = photos.length ? photos : fallbackPhotos(album)
    const maxIndex = Math.max(0, viewerPhotos.value.length - 1)
    viewerIndex.value = Math.min(Math.max(0, tileIndex), maxIndex)
  } catch (err) {
    if (seq !== viewerSeq) return
    viewerPhotos.value = fallbackPhotos(album)
    viewerIndex.value = 0
    viewerError.value = "照片加载失败"
    console.warn("[album-index] load album detail failed:", err?.message || err)
  } finally {
    if (seq === viewerSeq) viewerLoading.value = false
  }
}

function retryViewer() {
  if (viewerAlbum.value) openViewer(viewerAlbum.value, viewerIndex.value)
}

function closeViewer() {
  if (!viewerOpen.value) return
  viewerSeq += 1
  viewerOpen.value = false
  viewerLoading.value = false
  viewerError.value = ""
  viewerPhotos.value = []
  viewerAlbum.value = null
  viewerIndex.value = 0
}

function stepViewer(delta) {
  if (!viewerOpen.value || viewerLoading.value || viewerError.value) return
  const next = viewerIndex.value + delta
  if (next < 0 || next > viewerTotal.value - 1) return
  viewerIndex.value = next
}

function onViewerTouchStart(event) {
  const touch = event?.touches?.[0] || event?.changedTouches?.[0]
  if (!touch) return
  touchStartX = touch.clientX
  touchStartY = touch.clientY
}

function onViewerTouchEnd(event) {
  if (viewerLoading.value || viewerError.value) return
  const touch = event?.changedTouches?.[0]
  if (!touch) return
  const dx = touch.clientX - touchStartX
  const dy = touch.clientY - touchStartY
  if (Math.abs(dx) < 40 || Math.abs(dx) < Math.abs(dy)) return
  stepViewer(dx < 0 ? 1 : -1)
}

// ---------------- 上传 ----------------
const uploadOpen = ref(false)
const form = ref({ title: "", category: "", date: todayKey() })
const mediaList = ref([])
const submitting = ref(false)
const uploadError = ref("")

const dateText = computed(() => formatDateAnchor(form.value.date).main || form.value.date)
const donePhotos = computed(() => mediaList.value.filter(item => item.status === "done"))
const uploadingCount = computed(
  () => mediaList.value.filter(item => item.status === "uploading").length
)
const failedCount = computed(
  () => mediaList.value.filter(item => item.status === "failed").length
)

const blockReasons = computed(() => {
  const reasons = []
  if (!form.value.title.trim()) reasons.push("请填写相册标题")
  if (!form.value.category) reasons.push("请选择分类")
  if (!donePhotos.value.length) reasons.push("请至少添加 1 张照片")
  if (uploadingCount.value) reasons.push(`${uploadingCount.value} 张照片上传中`)
  if (failedCount.value) reasons.push(`${failedCount.value} 张照片上传失败`)
  return reasons
})
const canSubmit = computed(() => blockReasons.value.length === 0 && !submitting.value)

const submitLabel = computed(() => {
  if (submitting.value) return "上传中…"
  if (uploadError.value) return "重新上传"
  return "上传相册"
})

function openUpload() {
  if (uploadOpen.value) return
  closeViewer()
  if (!form.value.date) form.value.date = todayKey()
  uploadError.value = ""
  uploadOpen.value = true
}

function closeUpload() {
  if (!uploadOpen.value) return
  uploadOpen.value = false
  uploadError.value = ""
}

function selectUploadCategory(value) {
  form.value.category = value
  uploadError.value = ""
}

function onDateChange(event) {
  const value = event?.detail?.value
  if (value) form.value.date = value
}

function withTimeout(promise, ms) {
  return Promise.race([
    promise,
    new Promise((_, reject) => setTimeout(() => reject(new Error("上传超时")), ms))
  ])
}

function fillUploaded(item, data) {
  item.file_url = data?.file_url || ""
  item.thumb_url = data?.thumb_url || data?.file_url || ""
  item.width = Number(data?.width || 0)
  item.height = Number(data?.height || 0)
  item.progress = 100
  item.status = item.file_url ? "done" : "failed"
}

function uploadOne(tempPath) {
  // reactive：直接改裸对象属性不触发渲染，进度与「上传中 → 完成」会停在首帧
  const item = reactive({
    _temp: tempPath,
    file_url: "",
    thumb_url: "",
    width: 0,
    height: 0,
    progress: 0,
    status: "uploading"
  })
  mediaList.value.push(item)
  withTimeout(
    uploadAlbumPhoto(tempPath, progress => {
      if (item.status === "uploading") item.progress = progress
    }),
    UPLOAD_TIMEOUT_MS
  )
    .then(res => fillUploaded(item, res.data || {}))
    .catch(err => {
      item.status = "failed"
      console.warn("[album-index] upload photo failed:", err?.message || err)
    })
}

async function pickPhotos() {
  const remain = MAX_PHOTOS - mediaList.value.length
  if (remain <= 0) {
    uni.showToast({ title: `最多 ${MAX_PHOTOS} 张照片`, icon: "none" })
    return
  }
  const files = await pickImages({ count: remain })
  let rejected = 0
  files.forEach(file => {
    if (file.size > MAX_PHOTO_BYTES) {
      rejected += 1
      return
    }
    uploadOne(file.path)
  })
  if (rejected) {
    uni.showToast({ title: `有 ${rejected} 张超过 20MB，未添加`, icon: "none" })
  }
}

function retryPhoto(index) {
  const item = mediaList.value[index]
  if (!item || item.status !== "failed" || !item._temp) return
  item.status = "uploading"
  item.progress = 0
  withTimeout(
    uploadAlbumPhoto(item._temp, progress => {
      if (item.status === "uploading") item.progress = progress
    }),
    UPLOAD_TIMEOUT_MS
  )
    .then(res => fillUploaded(item, res.data || {}))
    .catch(err => {
      item.status = "failed"
      console.warn("[album-index] retry upload failed:", err?.message || err)
    })
}

function removePhoto(index) {
  mediaList.value.splice(index, 1)
}

async function submitUpload() {
  if (!canSubmit.value) {
    if (blockReasons.value.length) {
      uni.showToast({ title: blockReasons.value[0], icon: "none" })
    }
    return
  }
  submitting.value = true
  uploadError.value = ""
  try {
    await createTeacherAlbum(
      {
        title: form.value.title.trim(),
        category: form.value.category,
        album_date: form.value.date,
        photos: donePhotos.value.map(item => ({
          file_url: item.file_url,
          thumb_url: item.thumb_url || "",
          width: item.width || 0,
          height: item.height || 0
        }))
      },
      { silent: true }
    )
    form.value = { title: "", category: "", date: todayKey() }
    mediaList.value = []
    uploadOpen.value = false
    uni.showToast({ title: "已上传", icon: "success" })
    // 统计与列表都刷新；保留已渲染内容，避免整页闪骨架
    await loadAlbums({ skeleton: false })
  } catch (err) {
    uploadError.value = err?.message || "上传失败，请重试"
    console.warn("[album-index] create album failed:", err?.message || err)
  } finally {
    submitting.value = false
  }
}

function noop() {}

onLoad(() => {
  loadAlbums({ skeleton: true })
})
</script>

<template>
  <PageShell bottom-space="150rpx" max-width="430px">
    <view class="album-page">
      <!-- 加载：统计 + 胶囊 + 两张卡骨架 -->
      <view v-if="viewState === 'loading'" class="state-loading">
        <view class="sk-summary">
          <view v-for="i in 3" :key="'sk-stat-' + i" class="sk-stat">
            <view class="sk sk-avatar" />
            <view class="sk-texts">
              <view class="sk sk-bar sk-bar--label" />
              <view class="sk sk-bar sk-bar--value" />
            </view>
          </view>
        </view>
        <view class="sk-filter">
          <view v-for="i in 5" :key="'sk-chip-' + i" class="sk sk-chip" />
        </view>
        <view class="sk-list">
          <view v-for="i in 2" :key="'sk-card-' + i" class="sk-card">
            <view class="sk-cover">
              <view v-for="j in 3" :key="'sk-square-' + j" class="sk sk-square" />
            </view>
            <view class="sk-lines">
              <view class="sk sk-bar sk-bar--title" />
              <view class="sk sk-bar sk-bar--meta" />
            </view>
          </view>
        </view>
      </view>

      <!-- 失败 -->
      <view v-else-if="viewState === 'error'" class="state-panel">
        <view class="state-icon is-danger"><text class="icon state-glyph">&#xECA1;</text></view>
        <text class="state-title">相册加载失败</text>
        <text class="state-copy">网络异常，未获取到照片</text>
        <view class="btn-soft state-action" @click="retryLoad">重试</view>
      </view>

      <!-- 空：本班还没有相册 -->
      <view v-else-if="viewState === 'empty'" class="state-panel">
        <view class="state-icon"><text class="icon state-glyph">&#xEE4B;</text></view>
        <text class="state-title">还没有照片</text>
        <text class="state-copy">上传后可在这里按分类查看</text>
        <view class="btn-soft state-action" @click="openUpload">上传照片</view>
      </view>

      <template v-else>
        <view class="stats-card">
          <view
            v-for="(item, index) in STAT_ITEMS"
            :key="item.key"
            class="stat-col"
            :class="{ 'stat-col--divided': index > 0 }"
          >
            <view class="stat-icon" :class="'stat-icon--' + item.tone">
              <text class="icon stat-glyph">{{ item.icon }}</text>
            </view>
            <view class="stat-text">
              <text class="stat-label">{{ item.label }}</text>
              <text class="stat-value">{{ stats[item.key] }}</text>
            </view>
          </view>
        </view>

        <view class="filter-row">
          <view
            v-for="item in ALBUM_FILTERS"
            :key="item.value || 'all'"
            class="filter-chip"
            :class="{ 'is-active': activeCategory === item.value }"
            @click="selectCategory(item.value)"
          >
            <text>{{ item.label }}</text>
          </view>
        </view>

        <!-- 分类筛选无结果：统计与筛选保留，列表区给提示 -->
        <view v-if="viewState === 'filterEmpty'" class="filter-empty">
          <text class="filter-empty-title">这个分类下还没有相册</text>
          <text class="filter-empty-copy">换个分类看看，或查看全部照片</text>
          <view class="btn-soft btn-soft--sm" @click="selectCategory('')">查看全部</view>
        </view>

        <view v-else class="album-list">
          <view v-for="(group, groupIndex) in monthGroups" :key="group.key" class="month-group">
            <view class="month-head" @click="toggleMonth(group, groupIndex)">
              <text class="month-title">{{ group.label }}</text>
              <view class="month-meta">
                <text class="month-count">{{ group.items.length }} 册</text>
                <text v-if="isMonthOpen(group, groupIndex)" class="icon month-chevron">&#xEA78;</text>
                <text v-else class="icon month-chevron">&#xEA4E;</text>
              </view>
            </view>

            <view v-if="isMonthOpen(group, groupIndex)" class="month-cards">
              <view v-for="album in group.items" :key="album.id" class="album-card">
                <view class="album-cover" :class="'album-cover--' + album.coverTiles.length">
                  <view
                    v-for="(cover, coverIndex) in album.coverTiles"
                    :key="coverIndex"
                    class="cover-cell"
                    :aria-label="'查看' + (album.title || '相册') + '第 ' + (coverIndex + 1) + ' 张照片'"
                    @click="openViewer(album, coverIndex)"
                  >
                    <image class="cover-img" :src="cover" mode="aspectFill" />
                  </view>
                  <view v-if="!album.coverTiles.length" class="cover-cell cover-cell--empty">
                    <text class="icon cover-glyph">&#xEE4B;</text>
                  </view>
                </view>

                <view class="album-info" @click="openViewer(album, 0)">
                  <view class="album-line">
                    <text class="album-name">{{ album.title || "未命名相册" }}</text>
                    <text class="album-count">{{ album.photo_count || 0 }}张</text>
                  </view>
                  <view class="album-line album-meta">
                    <text class="album-date">{{ formatDateLabel(album.album_date) }}</text>
                    <view class="tag" :class="albumTagClass(album.category)">
                      <text>{{ album.category_label || albumCategoryLabel(album.category) }}</text>
                    </view>
                  </view>
                </view>
              </view>
            </view>
          </view>

          <view v-if="hasMore" class="album-more">
            <text class="album-more-hint">已展示最近 {{ albums.length }} 册 · 共 {{ total }} 册</text>
            <view class="btn-soft btn-soft--more" :class="{ 'is-busy': loadingMore }" @click="loadAll">
              {{ loadingMore ? "加载中…" : `查看全部 ${total} 册` }}
            </view>
          </view>
        </view>
      </template>
    </view>

    <!-- 固定底栏：空态也保留 -->
    <template #bottom>
      <view class="actionbar">
        <view class="btn-upload" aria-label="上传照片" @click="openUpload">
          <text class="icon btn-glyph">&#xEA13;</text>
          <text class="btn-upload-text">上传照片</text>
        </view>
      </view>
    </template>

    <!-- 大图浏览：轻触遮罩关闭；首末张按钮禁用；左右滑动切图 -->
    <view
      v-if="viewerOpen"
      class="viewer"
      @touchstart="onViewerTouchStart"
      @touchend="onViewerTouchEnd"
      @touchmove.stop.prevent="noop"
    >
      <view class="viewer-scrim" @click="closeViewer" />
      <view class="viewer-sheet">
        <view class="viewer-handle" />
        <view class="viewer-head">
          <view class="viewer-heading">
            <text class="viewer-title">{{ viewerTitle }}</text>
            <text class="viewer-date">{{ viewerDateText }}</text>
          </view>
          <view class="viewer-close" aria-label="关闭大图" @click="closeViewer">
            <text class="icon">&#xEB99;</text>
          </view>
        </view>

        <view class="viewer-photo">
          <image
            v-if="currentPhoto && !viewerLoading && !viewerError"
            class="viewer-img"
            :src="currentPhoto.file_url"
            mode="aspectFit"
          />
          <view v-else-if="viewerLoading" class="viewer-placeholder">
            <view class="viewer-spinner" />
            <text class="viewer-placeholder-text">照片加载中…</text>
          </view>
          <view v-else class="viewer-placeholder">
            <text class="viewer-placeholder-text">{{ viewerError || "没有可显示的照片" }}</text>
            <view v-if="viewerError" class="viewer-retry" @click="retryViewer">重试</view>
          </view>
        </view>

        <view class="viewer-foot">
          <view
            class="viewer-nav"
            :class="{ 'is-disabled': !canPrev }"
            aria-label="上一张"
            @click="stepViewer(-1)"
          >
            <text class="icon">&#xEA64;</text>
          </view>
          <text class="viewer-count">第 {{ viewerIndex + 1 }} 张 / 共 {{ viewerTotal }} 张</text>
          <view
            class="viewer-nav viewer-nav--next"
            :class="{ 'is-disabled': !canNext }"
            aria-label="下一张"
            @click="stepViewer(1)"
          >
            <text class="icon">&#xEA6E;</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 上传弹层 -->
    <view v-if="uploadOpen" class="sheet-layer">
      <view class="sheet-scrim" @click="closeUpload" @touchmove.stop.prevent="noop" />
      <view class="upload-sheet">
        <view class="sheet-grab" />
        <view class="sheet-head">
          <text class="sheet-title">上传照片</text>
          <view class="sheet-close" aria-label="关闭上传" @click="closeUpload">
            <text class="icon">&#xEB99;</text>
          </view>
        </view>

        <scroll-view scroll-y class="upload-body">
          <view class="field">
            <text class="field-label">相册标题</text>
            <input
              v-model="form.title"
              class="field-input"
              type="text"
              maxlength="30"
              placeholder="例如：秋日郊游"
              placeholder-style="color: #94A3B8"
            />
          </view>

          <view class="field">
            <text class="field-label">分类</text>
            <view class="upload-cats">
              <view
                v-for="item in ALBUM_CATEGORIES"
                :key="item.value"
                class="upload-cat"
                :class="{ 'is-active': form.category === item.value }"
                @click="selectUploadCategory(item.value)"
              >
                <text>{{ item.label }}</text>
              </view>
            </view>
          </view>

          <view class="field">
            <text class="field-label">日期</text>
            <picker mode="date" :value="form.date" @change="onDateChange">
              <view class="field-picker">
                <text class="field-picker-value">{{ dateText }}</text>
                <text class="icon field-picker-icon">&#xEA4E;</text>
              </view>
            </picker>
          </view>

          <view class="field">
            <view class="field-head">
              <text class="field-label field-label--inline">照片</text>
              <text class="field-note">{{ mediaList.length }} / {{ MAX_PHOTOS }}</text>
            </view>
            <view class="upload-grid">
              <view v-for="(item, index) in mediaList" :key="index" class="upload-tile">
                <image class="upload-img" :src="item._temp" mode="aspectFill" />
                <view v-if="item.status === 'uploading'" class="upload-mask">
                  <view class="upload-spinner" />
                  <text>{{ item.progress || 0 }}%</text>
                </view>
                <view
                  v-else-if="item.status === 'failed'"
                  class="upload-mask is-failed"
                  @click="retryPhoto(index)"
                >
                  <text>上传失败</text>
                  <text class="upload-retry">点击重试</text>
                </view>
                <view v-else class="upload-done"><text class="icon">&#xEB7B;</text></view>
                <view class="upload-remove" @click.stop="removePhoto(index)">
                  <text class="icon">&#xEB99;</text>
                </view>
              </view>
              <view
                v-if="mediaList.length < MAX_PHOTOS"
                class="upload-tile upload-add"
                @click="pickPhotos"
              >
                <text class="icon upload-add-icon">&#xEE47;</text>
                <text class="upload-add-text">添加照片</text>
              </view>
            </view>
          </view>
        </scroll-view>

        <view v-if="uploadError" class="upload-error">
          <text class="upload-error-text">{{ uploadError }}</text>
        </view>
        <text v-else-if="blockReasons.length && !submitting" class="upload-hint">
          {{ blockReasons[0] }}
        </text>

        <view class="upload-submit" :class="{ 'is-disabled': !canSubmit }" @click="submitUpload">
          {{ submitLabel }}
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
.icon {
  font-family: remixicon;
  font-style: normal;
  line-height: 1;
}

.album-page {
  padding-bottom: $spacing-lg;
}

/* ---------- 统计卡 ---------- */
.stats-card {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  align-items: center;
  padding: $spacing-xl $spacing-md;
  background: $color-bg-card;
  border: 1rpx solid rgba($color-border-hairline, 0.4);
  border-radius: $component-card-radius;
  box-shadow: $shadow-md;
}

.stat-col {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  min-width: 0;
  padding: 0 2rpx;
}

.stat-col--divided::before {
  content: "";
  position: absolute;
  top: 14rpx;
  bottom: 14rpx;
  left: 0;
  width: 1rpx;
  background: rgba($color-border-hairline, 0.55);
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 56rpx;
  height: 56rpx;
  border-radius: $border-radius-lg;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.stat-icon--success {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.stat-icon--info {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.stat-glyph {
  font-size: $typography-font-size-xl;
}

.stat-text {
  display: flex;
  flex-direction: column;
  gap: 2rpx;
  min-width: 0;
}

.stat-label {
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
  color: $color-text-muted;
  white-space: nowrap;
}

.stat-value {
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-bold;
  line-height: 1.15;
  color: $color-text-primary;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

/* ---------- 分类胶囊 ---------- */
.filter-row {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 6rpx;
  margin-top: $spacing-lg;
  padding: $spacing-xs;
  background: rgba($color-bg-card, 0.72);
  border: 1rpx solid rgba($color-border-hairline, 0.35);
  border-radius: $border-radius-full;
  box-shadow: $shadow-sm;
}

.filter-chip {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  height: $component-input-height;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.filter-chip.is-active {
  background: $color-brand-primary-bg;
  border-color: $color-brand-primary;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.filter-chip:active {
  transform: scale(0.96);
}

/* ---------- 月份分组 ---------- */
.album-list {
  margin-top: $spacing-xl;
}

.month-group + .month-group {
  margin-top: 40rpx;
}

.month-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  padding-bottom: $spacing-md;
  border-bottom: 1rpx solid rgba($color-border-hairline, 0.5);
}

.month-title {
  min-width: 0;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  color: $color-text-primary;
}

.month-meta {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  flex: none;
}

.month-count {
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.3;
  color: $color-text-muted;
  font-variant-numeric: tabular-nums;
}

.month-chevron {
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.month-cards {
  display: flex;
  flex-direction: column;
  gap: $component-card-gap;
  margin-top: $spacing-lg;
}

/* ---------- 相册卡 ---------- */
.album-card {
  padding: 20rpx;
  background: $color-bg-card;
  border: 1rpx solid rgba($color-border-hairline, 0.32);
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

.album-cover {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
}

/* 有几张封面画几格，不重复裁同一张 */
.album-cover--0,
.album-cover--1 {
  grid-template-columns: minmax(0, 1fr);
}

.album-cover--2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.cover-cell {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 210rpx;
  overflow: hidden;
  border: 1rpx solid rgba($color-bg-card, 0.6);
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.cover-img {
  width: 100%;
  height: 100%;
}

.cover-glyph {
  font-size: $typography-font-size-2xl;
  color: $color-text-hint;
}

.cover-cell:active {
  transform: scale(0.97);
}

.album-info {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  padding: 20rpx 2rpx 2rpx;
}

.album-line {
  display: flex;
  align-items: flex-start;
  gap: $spacing-md;
  width: 100%;
  min-width: 0;
}

.album-name {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  color: $color-text-primary;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.album-count {
  flex: none;
  padding: 4rpx $spacing-md;
  border: 1rpx solid rgba($color-border-hairline, 0.32);
  border-radius: $border-radius-full;
  background: $color-bg-page;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  color: $color-text-muted;
}

.album-meta {
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
}

.album-date {
  min-width: 0;
  font-size: $typography-font-size-sm;
  line-height: 1.3;
  color: $color-text-muted;
  font-variant-numeric: tabular-nums;
}

.tag {
  flex: none;
  display: inline-flex;
  align-items: center;
  padding: 6rpx $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.tag--activity {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.tag--daily {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.tag--work {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.tag--festival {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
}

/* ---------- 列表尾部 ---------- */
.album-more {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
  margin-top: $spacing-xl;
  padding: 0 $spacing-md;
  text-align: center;
}

.album-more-hint {
  font-size: $typography-font-size-sm;
  line-height: 1.3;
  color: $color-text-muted;
  font-variant-numeric: tabular-nums;
}

.btn-soft {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  border: 1rpx solid rgba($color-brand-primary, 0.45);
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.btn-soft--more {
  min-width: 336rpx;
}

.btn-soft--sm {
  min-width: 240rpx;
  margin-top: $spacing-lg;
}

.btn-soft:active {
  transform: scale(0.96);
}

.btn-soft.is-busy {
  opacity: 0.6;
}

/* ---------- 筛选无结果 ---------- */
.filter-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: $spacing-xl;
  padding: 72rpx $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.filter-empty-title {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
}

.filter-empty-copy {
  margin-top: $spacing-xs;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-text-muted;
}

/* ---------- 空 / 错误 ---------- */
.state-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  padding: 80rpx $spacing-xl 120rpx;
  text-align: center;
}

.state-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 128rpx;
  height: 128rpx;
  margin-bottom: $spacing-xl;
  border: 1rpx solid rgba($color-brand-primary, 0.35);
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.state-icon.is-danger {
  border-color: rgba($color-semantic-danger, 0.35);
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.state-glyph {
  font-size: 56rpx;
}

.state-title {
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  color: $color-text-primary;
}

.state-copy {
  max-width: 520rpx;
  margin-top: $spacing-sm;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  color: $color-text-muted;
}

.state-action {
  min-width: 336rpx;
  margin-top: $spacing-lg;
}

/* ---------- 加载骨架 ---------- */
.state-loading {
  padding-bottom: $spacing-xs;
}

.sk-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  padding: $spacing-lg $spacing-sm;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

.sk-stat {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
}

.sk-stat + .sk-stat {
  border-left: 1rpx solid rgba($color-border-hairline, 0.45);
}

.sk {
  display: block;
  background: rgba($color-border-hairline, 0.35);
  animation: sk-breathe 1.6s ease-in-out infinite;
}

.sk-avatar {
  flex: none;
  width: 56rpx;
  height: 56rpx;
  border-radius: $border-radius-lg;
}

.sk-texts {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.sk-filter {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 6rpx;
  margin-top: $spacing-lg;
  padding: $spacing-xs;
  background: $color-bg-card;
  border-radius: $border-radius-full;
}

.sk-chip {
  height: $component-input-height;
  border-radius: $border-radius-full;
}

.sk-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  margin-top: $spacing-xl;
}

.sk-card {
  padding: 20rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

.sk-cover {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
}

.sk-square {
  height: 210rpx;
  border-radius: $border-radius-lg;
}

.sk-lines {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  padding: $spacing-md 2rpx 2rpx;
}

.sk-bar {
  height: 16rpx;
  border-radius: $border-radius-full;
}

.sk-bar--label {
  width: 76rpx;
}

.sk-bar--value {
  width: 56rpx;
  height: 28rpx;
  border-radius: $border-radius-sm;
}

.sk-bar--title {
  width: 52%;
  height: 24rpx;
}

.sk-bar--meta {
  width: 32%;
}

@keyframes sk-breathe {
  0%,
  100% {
    opacity: 0.55;
  }
  50% {
    opacity: 1;
  }
}

/* ---------- 底部上传栏 ---------- */
.actionbar {
  padding: $spacing-md $component-page-padding-x calc(#{$spacing-md} + env(safe-area-inset-bottom));
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  border-top: 1rpx solid rgba($color-border-hairline, 0.4);
  box-shadow: $shadow-md;
}

.btn-upload {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  width: 100%;
  height: 96rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  box-shadow: $shadow-md;
}

.btn-upload:active {
  transform: scale(0.985);
  box-shadow: $shadow-sm;
}

.btn-glyph {
  font-size: $typography-font-size-2xl;
}

.btn-upload-text {
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
}

/* ---------- 大图层 ---------- */
.viewer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 120;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.viewer-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba($color-text-primary, 0.55);
}

.viewer-sheet {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 430px;
  padding: $spacing-md $spacing-xl calc(#{$spacing-xl} + env(safe-area-inset-bottom));
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.viewer-handle {
  width: 72rpx;
  height: 8rpx;
  margin: 0 auto 20rpx;
  border-radius: $border-radius-full;
  background: rgba($color-border-hairline, 0.8);
}

.viewer-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: $spacing-lg;
  padding-bottom: $spacing-lg;
}

.viewer-heading {
  min-width: 0;
  padding-top: 2rpx;
}

.viewer-title {
  display: block;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  color: $color-text-primary;
}

.viewer-date {
  display: block;
  margin-top: 2rpx;
  font-size: $typography-font-size-sm;
  line-height: 1.3;
  color: $color-text-muted;
}

.viewer-close {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 88rpx;
  height: 88rpx;
  border: 1rpx solid rgba($color-border-hairline, 0.5);
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
}

.viewer-photo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 514rpx;
  overflow: hidden;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}

.viewer-img {
  width: 100%;
  height: 100%;
}

.viewer-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
  padding: 0 $spacing-lg;
  text-align: center;
}

.viewer-placeholder-text {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
}

.viewer-retry {
  margin-top: $spacing-xs;
  padding: $spacing-xs $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

.viewer-spinner {
  width: 56rpx;
  height: 56rpx;
  border: 5rpx solid $color-brand-primary-bg;
  border-top-color: $color-brand-primary;
  border-radius: $border-radius-full;
  animation: album-spin 0.9s linear infinite;
}

.viewer-foot {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding-top: 20rpx;
}

.viewer-nav {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 88rpx;
  height: 88rpx;
  border: 1rpx solid rgba($color-brand-primary, 0.4);
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xl;
}

.viewer-nav--next {
  border-color: transparent;
  background: $gradient-brand;
  color: $color-text-primary;
  box-shadow: $shadow-md;
}

.viewer-nav:active {
  transform: scale(0.94);
}

.viewer-nav.is-disabled {
  border-color: rgba($color-border-hairline, 0.5);
  background: $color-bg-page;
  color: $color-text-hint;
  box-shadow: none;
  opacity: 0.55;
}

.viewer-nav.is-disabled:active {
  transform: none;
}

.viewer-count {
  flex: 1;
  min-width: 0;
  text-align: center;
  font-size: $typography-font-size-sm;
  line-height: 1.3;
  color: $color-text-muted;
  font-variant-numeric: tabular-nums;
}

@keyframes album-spin {
  to {
    transform: rotate(360deg);
  }
}

/* ---------- 上传弹层 ---------- */
.sheet-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 90;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.sheet-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba($color-text-primary, 0.45);
}

.upload-sheet {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 430px;
  max-height: 88vh;
  padding: $spacing-xs $component-page-padding-x calc(#{$spacing-md} + env(safe-area-inset-bottom));
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.sheet-grab {
  width: 72rpx;
  height: 8rpx;
  margin: 0 auto 20rpx;
  border-radius: $border-radius-full;
  background: rgba($color-border-hairline, 0.8);
}

.sheet-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 88rpx;
}

.sheet-title {
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
}

.sheet-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-lg;
}

.upload-body {
  flex: 1;
  height: 56vh;
  min-height: 0;
}

.field {
  padding: $spacing-md 0;
  border-top: 1rpx solid $color-border-subtle;
}

.field:first-child {
  border-top: 0;
}

.field-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.field-label {
  display: block;
  margin-bottom: $spacing-xs;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  color: $color-text-body;
}

.field-label--inline {
  margin-bottom: 0;
}

.field-note {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.field-input {
  width: 100%;
  height: 88rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
}

.upload-cats {
  display: flex;
  gap: $spacing-xs;
}

.upload-cat {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  height: 72rpx;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.upload-cat.is-active {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

.field-picker {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 88rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}

.field-picker-value {
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

.field-picker-icon {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.upload-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: $spacing-sm;
  margin-top: $spacing-xs;
}

.upload-tile {
  position: relative;
  height: 164rpx;
  overflow: hidden;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.upload-img {
  width: 100%;
  height: 100%;
}

.upload-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2rpx;
  background: rgba($color-text-primary, 0.62);
  color: $color-text-on-brand;
  font-size: $typography-font-size-xs;
}

.upload-mask.is-failed {
  background: rgba($color-semantic-danger, 0.82);
}

.upload-spinner {
  width: 32rpx;
  height: 32rpx;
  border: 4rpx solid $color-text-on-brand;
  border-top-color: $color-brand-primary;
  border-radius: $border-radius-full;
  animation: album-spin 0.9s linear infinite;
}

.upload-retry {
  text-decoration: underline;
}

.upload-done {
  position: absolute;
  left: $spacing-xs;
  bottom: $spacing-xs;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36rpx;
  height: 36rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-success;
  color: $color-text-on-brand;
  font-size: $typography-font-size-xs;
}

.upload-remove {
  position: absolute;
  top: 4rpx;
  right: 4rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48rpx;
  height: 48rpx;
  border-radius: $border-radius-full;
  background: rgba($color-text-primary, 0.72);
  color: $color-text-on-brand;
  font-size: $typography-font-size-xs;
}

.upload-add {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2rpx;
  border-style: dashed;
  border-color: $color-border-hairline;
  color: $color-text-muted;
}

.upload-add-icon {
  font-size: $typography-font-size-3xl;
}

.upload-add-text {
  font-size: $typography-font-size-xs;
}

.upload-error {
  margin-top: $spacing-xs;
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.upload-hint {
  margin-top: $spacing-xs;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.upload-submit {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 96rpx;
  margin-top: $spacing-sm;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-md;
}

.upload-submit.is-disabled {
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-page;
  color: $color-text-hint;
  box-shadow: none;
}
</style>
