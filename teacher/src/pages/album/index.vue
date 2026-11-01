<script setup>
import { computed, reactive, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import {
  createTeacherAlbum,
  deleteTeacherAlbum,
  fetchTeacherAlbum,
  fetchTeacherAlbumStats,
  fetchTeacherAlbums,
  publishTeacherAlbum,
  unpublishTeacherAlbum,
  updateTeacherAlbum,
  uploadAlbumPhoto
} from "@/api/teacher-albums"
import { fetchTeacherClassStudents } from "@/api/teacher-class"
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

// 状态筛选与分类筛选正交：两者可叠加，状态默认「全部」不传 status。
const STATUS_FILTERS = [
  { value: "", label: "全部" },
  { value: "draft", label: "草稿" },
  { value: "published", label: "已发布" },
  { value: "unpublished", label: "已下架" }
]
const STATUS_LABELS = { draft: "草稿", published: "已发布", unpublished: "已下架" }

const stats = ref({ total_photos: 0, monthly_new: 0, album_count: 0 })
const albums = ref([])
const total = ref(0)
const activeCategory = ref("")
const activeStatus = ref("")
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
    return activeCategory.value || activeStatus.value ? "filterEmpty" : "empty"
  }
  return "normal"
})

const filterEmptyTitle = computed(() => {
  if (activeCategory.value && activeStatus.value) return "当前筛选下还没有相册"
  if (activeStatus.value) return "这个状态下还没有相册"
  return "这个分类下还没有相册"
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

function statusLabel(status) {
  return STATUS_LABELS[status] || ""
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
  if (activeStatus.value) params.status = activeStatus.value
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

function selectStatus(value) {
  if (value === activeStatus.value || loading.value) return
  activeStatus.value = value
  loadAlbums({ skeleton: true })
}

// 「筛选无结果」的出口要同时清掉分类与状态，否则点了「查看全部」仍停在空列表。
function resetFilters() {
  if (loading.value) return
  if (!activeCategory.value && !activeStatus.value) return
  activeCategory.value = ""
  activeStatus.value = ""
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
const viewerDetail = ref(null)
const actionBusy = ref(false)
let viewerSeq = 0
let touchStartX = 0
let touchStartY = 0

const viewerTitle = computed(() => viewerAlbum.value?.title || "相册")
const viewerDateText = computed(() => formatDateLabel(viewerAlbum.value?.album_date))
// 详情返回前先用列表行里的状态兜底，避免打开弹层时徽标空一拍。
const viewerStatus = computed(
  () => viewerDetail.value?.status || viewerAlbum.value?.status || ""
)
const viewerStatusLabel = computed(() => statusLabel(viewerStatus.value))
const viewerUnauthorizedCount = computed(() =>
  normalizeCount(viewerDetail.value?.unauthorized_photo_count)
)
const viewerUnauthText = computed(() =>
  viewerDetail.value ? `未授权照片 ${viewerUnauthorizedCount.value} 张` : ""
)
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
  viewerDetail.value = null
  viewerPhotos.value = []
  viewerIndex.value = 0
  viewerLoading.value = true
  viewerError.value = ""
  const seq = ++viewerSeq
  try {
    const res = await fetchTeacherAlbum(album.id, { silent: true })
    if (seq !== viewerSeq) return
    viewerDetail.value = res.data || null
    const photos = Array.isArray(res.data?.photos)
      ? res.data.photos.filter(photo => photo?.file_url)
      : []
    viewerPhotos.value = photos.length ? photos : fallbackPhotos(album)
    const maxIndex = Math.max(0, viewerPhotos.value.length - 1)
    viewerIndex.value = Math.min(Math.max(0, tileIndex), maxIndex)
  } catch (err) {
    if (seq !== viewerSeq) return
    viewerDetail.value = null
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
  viewerDetail.value = null
  viewerAlbum.value = null
  viewerIndex.value = 0
}

function stepViewer(delta) {
  if (!viewerOpen.value || viewerLoading.value || viewerError.value) return
  const next = viewerIndex.value + delta
  if (next < 0 || next > viewerTotal.value - 1) return
  viewerIndex.value = next
}

// ---------------- 发布 / 下架 / 删除 ----------------
function confirmDialog({ title, content, confirmText = "确定" }) {
  return new Promise(resolve => {
    uni.showModal({
      title,
      content,
      confirmText,
      cancelText: "取消",
      success: res => resolve(!!res.confirm),
      fail: () => resolve(false)
    })
  })
}

function applyAlbumStatus(id, status) {
  const index = albums.value.findIndex(item => item.id === id)
  if (index >= 0) albums.value[index] = { ...albums.value[index], status }
  if (viewerAlbum.value?.id === id) viewerAlbum.value = { ...viewerAlbum.value, status }
  if (viewerDetail.value) viewerDetail.value = { ...viewerDetail.value, status }
}

async function requestAlbumAction(id, action) {
  if (action === "publish") return publishTeacherAlbum(id, { silent: true })
  if (action === "unpublish") return unpublishTeacherAlbum(id, { silent: true })
  return deleteTeacherAlbum(id, { silent: true })
}

async function runViewerAction(action) {
  if (actionBusy.value) return
  const album = viewerAlbum.value
  if (!album?.id) return
  const id = album.id

  if (action === "publish") {
    const count = viewerUnauthorizedCount.value
    if (count > 0) {
      const ok = await confirmDialog({
        title: "发布确认",
        content: `${count} 张照片含未授权孩子，发布后其他家长不可见`,
        confirmText: "仍要发布"
      })
      if (!ok) return
    }
  } else if (action === "unpublish") {
    const ok = await confirmDialog({
      title: "下架相册",
      content: "下架后家长端立即不可见，之后可以重新发布。",
      confirmText: "下架"
    })
    if (!ok) return
  } else if (action === "delete") {
    const ok = await confirmDialog({
      title: "删除相册",
      content: "删除后相册从列表移除、家长端不可见；相册保留 180 天供审计。",
      confirmText: "删除"
    })
    if (!ok) return
  }

  actionBusy.value = true
  try {
    await requestAlbumAction(id, action)
    if (action === "delete") {
      // 先从列表移除，避免刷新失败时删掉的相册还留在眼前
      albums.value = albums.value.filter(item => item.id !== id)
      total.value = Math.max(0, total.value - 1)
      closeViewer()
      uni.showToast({ title: "已删除", icon: "success" })
    } else {
      const nextStatus = action === "publish" ? "published" : "unpublished"
      applyAlbumStatus(id, nextStatus)
      uni.showToast({ title: action === "publish" ? "已发布" : "已下架", icon: "success" })
    }
    // 统计只算已发布，列表可能因状态筛选变化；失败时 loadAlbums 保留当前视图并提示
    await loadAlbums({ skeleton: false })
  } catch (err) {
    // 失败不改本地状态，弹层与列表停在操作前
    uni.showToast({ title: err?.message || "操作失败，请重试", icon: "none" })
    console.warn(`[album-index] ${action} album failed:`, err?.message || err)
  } finally {
    actionBusy.value = false
  }
}

function onViewerPublish() {
  runViewerAction("publish")
}

function onViewerUnpublish() {
  runViewerAction("unpublish")
}

function onViewerDelete() {
  runViewerAction("delete")
}

// 编辑复用上传弹层：详情已加载，直接用它预填标题/分类/日期/照片/参与者
function onViewerEdit() {
  if (actionBusy.value) return
  const detail = viewerDetail.value
  if (!detail) return
  openEditForm(detail)
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

// ---------------- 上传 / 编辑 ----------------
const uploadOpen = ref(false)
// 非空 = 编辑既有相册（复用同一表单）；空 = 新建草稿
const editingAlbum = ref(null)
const isEditing = computed(() => Boolean(editingAlbum.value))
const form = ref({ title: "", category: "", date: todayKey(), participants: [] })
const mediaList = ref([])
const submitting = ref(false)
const uploadError = ref("")

// 班级在读名单：新建时默认全选；编辑时来自详情，只用来补全选择器。
// 加载失败不阻塞上传，但新建提交前必须重试，避免静默提交空参与者。
const students = ref([])
const studentsLoading = ref(false)
const studentsError = ref("")
const participantsReady = ref(false)
// 最近一次名单加载是否成功：失败时不能判断谁「已离班」，只能用专门的阻断理由
const rosterLoaded = ref(false)
// 编辑时名单加载失败，老师仍需显式确认一次参与者（含空选择）才能提交
const participantsConfirmed = ref(false)
// 相册详情里可能包含已不在读的幼儿，保留姓名/头像用于摘要与选择器
const knownStudents = ref([])
let studentsSeq = 0

const dateText = computed(() => formatDateAnchor(form.value.date).main || form.value.date)
const donePhotos = computed(() => mediaList.value.filter(item => item.status === "done"))
const uploadingCount = computed(
  () => mediaList.value.filter(item => item.status === "uploading").length
)
const failedCount = computed(
  () => mediaList.value.filter(item => item.status === "failed").length
)

const studentNameMap = computed(() => {
  const map = new Map()
  for (const s of knownStudents.value) map.set(Number(s.id), s.name || "")
  // 在读名单优先，姓名/班级变化以当前名单为准
  for (const s of students.value) map.set(Number(s.student_id), s.name || "")
  return map
})

function studentName(id) {
  return studentNameMap.value.get(Number(id)) || ""
}

function participantsTextOf(ids) {
  if (!ids.length) {
    if (studentsLoading.value && !participantsReady.value) return "正在加载班级名单…"
    return "未选择参与幼儿"
  }
  const names = ids.map(studentName).filter(Boolean)
  if (!names.length) return `已选 ${ids.length} 人`
  if (names.length <= 2) return `已选 ${ids.length} 人：${names.join("、")}`
  return `已选 ${ids.length} 人：${names.slice(0, 2).join("、")} 等`
}

const participantsText = computed(() => participantsTextOf(form.value.participants))

// 未逐张调整的照片交给后端按相册级 participants 兜底，因此这里只显示默认人数。
function photoParticipantsText(item) {
  if (Array.isArray(item.student_ids)) {
    return item.student_ids.length ? `${item.student_ids.length} 人` : "无人"
  }
  const total = form.value.participants.length
  return total ? `默认 ${total}` : "默认"
}

function photoParticipantsAria(item) {
  if (Array.isArray(item.student_ids)) {
    return item.student_ids.length ? `参与幼儿 ${item.student_ids.length} 人` : "参与幼儿无人"
  }
  const total = form.value.participants.length
  return total ? `参与幼儿默认 ${total} 人` : "参与幼儿默认未选择"
}

// ---------------- 参与者校验 ----------------
// 已离班/已删除的幼儿不能继续提交：后端会按「不属于本园本班」整单拒绝 422。
const hasAnyParticipantIds = computed(
  () =>
    form.value.participants.length > 0 ||
    mediaList.value.some(item => Array.isArray(item.student_ids) && item.student_ids.length > 0)
)

function staleIdsOf(ids) {
  // 名单没加载成功时无法判断谁已离班，交给 studentsError 分支处理；
  // 刷新中沿用上一次成功名单，避免请求间隙放过已离班 id
  if (!rosterLoaded.value || studentsError.value) return []
  const roster = new Set(students.value.map(s => Number(s.student_id)))
  return [...new Set(ids.map(Number))].filter(id => !roster.has(id))
}

const staleAlbumIds = computed(() => staleIdsOf(form.value.participants))
const stalePhotoIds = computed(() => {
  const ids = []
  for (const item of mediaList.value) {
    if (Array.isArray(item.student_ids)) ids.push(...item.student_ids)
  }
  return staleIdsOf(ids)
})
const staleParticipants = computed(() => [
  ...new Set([...staleAlbumIds.value, ...stalePhotoIds.value])
])
const staleParticipantsText = computed(() => {
  const names = staleParticipants.value.map(id => studentName(id) || "未知幼儿")
  if (!names.length) return ""
  return names.length > 3 ? `${names.slice(0, 3).join("、")} 等 ${names.length} 位` : names.join("、")
})

const participantsIssue = computed(() => {
  if (studentsLoading.value && !rosterLoaded.value) return "正在加载班级名单…"
  if (studentsError.value) {
    if (!isEditing.value) return "班级名单加载失败，请重试"
    if (hasAnyParticipantIds.value) return "班级名单加载失败，暂时无法确认参与幼儿，请重试"
    // 相册本来就没有参与者：名单没回来时需老师显式确认一次空选择
    return participantsConfirmed.value ? "" : "班级名单加载失败，请重试或打开参与幼儿确认空名单"
  }
  if (!isEditing.value && !participantsReady.value) {
    return studentsLoading.value ? "正在加载班级名单…" : "班级名单加载失败，请重试"
  }
  if (staleParticipants.value.length) {
    const where =
      staleAlbumIds.value.length && stalePhotoIds.value.length
        ? "请在「参与幼儿」或对应照片中取消选中"
        : stalePhotoIds.value.length
          ? "请点开对应照片的参与幼儿取消选中"
          : "请在「参与幼儿」中取消选中"
    return `有 ${staleParticipants.value.length} 位幼儿已不在班级名单，${where}`
  }
  return ""
})

const blockReasons = computed(() => {
  const reasons = []
  if (!form.value.title.trim()) reasons.push("请填写相册标题")
  if (!form.value.category) reasons.push("请选择分类")
  if (!donePhotos.value.length) reasons.push("请至少添加 1 张照片")
  if (uploadingCount.value) reasons.push(`${uploadingCount.value} 张照片上传中`)
  if (failedCount.value) reasons.push(`${failedCount.value} 张照片上传失败`)
  if (participantsIssue.value) reasons.push(participantsIssue.value)
  return reasons
})
const canSubmit = computed(() => blockReasons.value.length === 0 && !submitting.value)

const submitLabel = computed(() => {
  if (submitting.value) return isEditing.value ? "保存中…" : "上传中…"
  if (uploadError.value) return isEditing.value ? "重新保存" : "重新上传"
  return isEditing.value ? "保存修改" : "上传相册"
})

const sheetTitle = computed(() => (isEditing.value ? "编辑相册" : "上传照片"))

async function loadStudents({ defaultAll = false } = {}) {
  const seq = ++studentsSeq
  studentsLoading.value = true
  studentsError.value = ""
  try {
    const res = await fetchTeacherClassStudents({}, { silent: true })
    if (seq !== studentsSeq) return
    students.value = Array.isArray(res.data?.list) ? res.data.list : []
    rosterLoaded.value = true
    if (defaultAll) {
      form.value.participants = students.value.map(s => Number(s.student_id))
      participantsReady.value = true
    }
    // 选择器打开时名单才返回：未改动过就同步成默认全选
    if (pickerOpen.value && pickerMode.value === "album" && !pickerDirty.value) {
      pickerPicked.value = new Set(form.value.participants)
      pickerInitial.value = new Set(form.value.participants)
    }
  } catch (err) {
    if (seq !== studentsSeq) return
    students.value = []
    rosterLoaded.value = false
    studentsError.value = "班级名单加载失败"
    console.warn("[album-index] load class students failed:", err?.message || err)
  } finally {
    if (seq === studentsSeq) studentsLoading.value = false
  }
}

function resetForm() {
  editingAlbum.value = null
  form.value = { title: "", category: "", date: todayKey(), participants: [] }
  mediaList.value = []
  participantsReady.value = false
  participantsConfirmed.value = false
  knownStudents.value = []
  uploadError.value = ""
  closePicker()
}

function openUpload() {
  if (uploadOpen.value) return
  closeViewer()
  if (isEditing.value) resetForm()
  if (!form.value.date) form.value.date = todayKey()
  uploadError.value = ""
  uploadOpen.value = true
  loadStudents({ defaultAll: !participantsReady.value })
}

function closeUpload() {
  if (!uploadOpen.value) return
  uploadOpen.value = false
  uploadError.value = ""
  closePicker()
  // 编辑态取消即丢弃草稿；新建态保留已填内容，重新打开可继续
  if (isEditing.value) resetForm()
}

function collectKnownStudents(detail) {
  const map = new Map()
  for (const s of detail?.album_students || []) map.set(Number(s.id), s)
  for (const photo of detail?.photos || []) {
    for (const s of photo?.students || []) map.set(Number(s.id), s)
  }
  return [...map.values()]
}

function openEditForm(detail) {
  const photos = (Array.isArray(detail?.photos) ? detail.photos : []).filter(p => p?.file_url)
  if (!detail?.id || !photos.length) {
    uni.showToast({ title: "相册没有可编辑的照片", icon: "none" })
    return
  }
  closeViewer()
  editingAlbum.value = {
    id: detail.id,
    title: detail.title || "",
    status: detail.status || "",
    category: detail.category || "",
    album_date: detail.album_date || ""
  }
  form.value = {
    title: detail.title || "",
    category: detail.category || "",
    date: String(detail.album_date || "").slice(0, 10) || todayKey(),
    participants: (detail.album_students || []).map(s => Number(s.id))
  }
  knownStudents.value = collectKnownStudents(detail)
  // 既有照片带 id/file_id；students 为空表示未逐张关联（提交时不传 student_ids，沿用相册级兜底）
  mediaList.value = photos.map(photo =>
    reactive({
      id: Number(photo.id || 0),
      file_id: Number(photo.file_id || 0),
      file_url: photo.file_url || "",
      thumb_url: photo.thumb_url || photo.file_url || "",
      width: Number(photo.width || 0),
      height: Number(photo.height || 0),
      _temp: photo.thumb_url || photo.file_url || "",
      progress: 100,
      status: "done",
      student_ids:
        Array.isArray(photo.students) && photo.students.length
          ? photo.students.map(s => Number(s.id))
          : null
    })
  )
  participantsReady.value = true
  // 名单重新加载前，先要求老师显式确认一次参与者（空相册尤其需要）
  participantsConfirmed.value = false
  uploadError.value = ""
  uploadOpen.value = true
  loadStudents()
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
  // 上传响应的 id 就是契约里的 file_id；缺 id 视为失败，避免把新照片退化成 URL 路径
  item.file_id = Number(data?.id || 0)
  item.file_url = data?.file_url || ""
  item.thumb_url = data?.thumb_url || data?.file_url || ""
  item.width = Number(data?.width || 0)
  item.height = Number(data?.height || 0)
  item.progress = 100
  item.status = item.file_url && item.file_id ? "done" : "failed"
}

function uploadOne(tempPath) {
  // reactive：直接改裸对象属性不触发渲染，进度与「上传中 → 完成」会停在首帧
  const item = reactive({
    id: 0,
    file_id: 0,
    _temp: tempPath,
    file_url: "",
    thumb_url: "",
    width: 0,
    height: 0,
    progress: 0,
    status: "uploading",
    // null = 未逐张调整，提交时不带 student_ids，由后端按相册级参与者兜底
    student_ids: null
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

function onPhotoTileTap(item, index) {
  if (!item || item.status !== "done") return
  openPhotoParticipants(index)
}

// ---------------- 参与幼儿选择器 ----------------
const PICKUP_STATUS_LABELS = { not_arrived: "未入园", not_picked_up: "未接走", picked_up: "已接走" }
const pickerOpen = ref(false)
const pickerMode = ref("album") // album = 相册级默认；photo = 单张覆盖
const pickerIndex = ref(-1)
const pickerPicked = ref(new Set())
// 打开时的初始选择：dirty 靠「当前集合 vs 初始集合」比较，勾上又取消不算修改
const pickerInitial = ref(new Set())
const pickerDirty = ref(false)

function setsEqual(a, b) {
  if (a.size !== b.size) return false
  for (const value of a) {
    if (!b.has(value)) return false
  }
  return true
}

function syncPickerDirty() {
  pickerDirty.value = !setsEqual(pickerPicked.value, pickerInitial.value)
}

function pickupStatusLabel(status) {
  return PICKUP_STATUS_LABELS[status] || ""
}

const pickerTarget = computed(() =>
  pickerMode.value === "photo" ? mediaList.value[pickerIndex.value] || null : null
)
const pickerCanRestore = computed(() =>
  Boolean(pickerTarget.value && Array.isArray(pickerTarget.value.student_ids))
)
const pickerAllSelected = computed(
  () =>
    students.value.length > 0 &&
    students.value.every(s => pickerPicked.value.has(Number(s.student_id)))
)
// 打开时已选中、但不在当前名单里的幼儿：显式列成「已离班/已删除」行，供老师取消
const pickerStaleStudents = computed(() => {
  if (!rosterLoaded.value || studentsLoading.value) return []
  const roster = new Set(students.value.map(s => Number(s.student_id)))
  const known = new Map(knownStudents.value.map(s => [Number(s.id), s]))
  return [...pickerInitial.value]
    .map(Number)
    .filter(id => !roster.has(id))
    .map(id => {
      const student = known.get(id)
      return {
        id,
        name: student?.name || "已离班幼儿",
        avatar: student?.avatar || ""
      }
    })
})
const pickerTitle = computed(() => (pickerMode.value === "photo" ? "照片参与幼儿" : "参与幼儿"))
const pickerDesc = computed(() => {
  if (pickerMode.value === "photo") {
    const override = Array.isArray(pickerTarget.value?.student_ids)
    return `第 ${pickerIndex.value + 1} 张 · 已选 ${pickerPicked.value.size} 人 · ${
      override ? "已单独调整，可恢复为相册默认" : "不改动则沿用相册默认参与者"
    }`
  }
  if (studentsError.value) return "班级名单加载失败，重试后再确认参与者"
  return `本班在读 ${students.value.length} 位 · 已选 ${pickerPicked.value.size} 人 · 未逐张调整的照片都按这份名单`
})

function openAlbumParticipants() {
  pickerMode.value = "album"
  pickerIndex.value = -1
  pickerPicked.value = new Set(form.value.participants.map(Number))
  pickerInitial.value = new Set(pickerPicked.value)
  pickerDirty.value = false
  pickerOpen.value = true
  if (studentsError.value) {
    loadStudents({ defaultAll: !participantsReady.value && !isEditing.value })
  }
}

function openPhotoParticipants(index) {
  const item = mediaList.value[index]
  if (!item || item.status !== "done") return
  const current = Array.isArray(item.student_ids) ? item.student_ids : form.value.participants
  pickerMode.value = "photo"
  pickerIndex.value = index
  pickerPicked.value = new Set(current.map(Number))
  pickerInitial.value = new Set(pickerPicked.value)
  pickerDirty.value = false
  pickerOpen.value = true
  if (studentsError.value) loadStudents()
}

function closePicker() {
  pickerOpen.value = false
  pickerDirty.value = false
  pickerIndex.value = -1
  pickerMode.value = "album"
  pickerPicked.value = new Set()
  pickerInitial.value = new Set()
}

function retryStudents() {
  loadStudents({ defaultAll: !participantsReady.value && !isEditing.value })
}

function togglePickerStudent(id) {
  const key = Number(id)
  const next = new Set(pickerPicked.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  pickerPicked.value = next
  syncPickerDirty()
}

function togglePickerAll() {
  const visible = new Set(students.value.map(s => Number(s.student_id)))
  if (pickerAllSelected.value) {
    // 只取消名单内的孩子；已离班的历史参与者要单独取消，不能一并清掉
    pickerPicked.value = new Set([...pickerPicked.value].filter(id => !visible.has(id)))
  } else {
    const next = new Set(pickerPicked.value)
    visible.forEach(id => next.add(id))
    pickerPicked.value = next
  }
  syncPickerDirty()
}

function confirmPicker() {
  const ids = [...pickerPicked.value]
  if (pickerMode.value === "album") {
    form.value.participants = ids
    // 老师确认过选择（含空选择）：名单加载失败时，这是允许提交 [] 的显式意图
    participantsConfirmed.value = true
    // 名单未加载成功时不能把「空选择」当成有效参与者：新建提交仍会被 blockReasons 拦住
    if (!studentsError.value && !studentsLoading.value) participantsReady.value = true
    closePicker()
    return
  }
  const item = pickerTarget.value
  // 未改动 = 保持继承（不写 student_ids），只有真正改过才产生覆盖
  if (item && pickerDirty.value) {
    item.student_ids = ids
    uni.showToast({
      title: ids.length ? "已更新这张照片的参与幼儿" : "这张照片已清空参与幼儿",
      icon: "none"
    })
  }
  closePicker()
}

function restorePhotoParticipants() {
  const item = pickerTarget.value
  if (!item) return
  item.student_ids = null
  closePicker()
  uni.showToast({ title: "已恢复为相册默认", icon: "none" })
}

// ---------------- 提交 ----------------
// 既有照片带 id（服务端原地更新、保留照片 ID）；新照片不带 id，由 file_id 建行。
// student_ids 省略 = 沿用相册级参与者；数组（含 []）= 显式覆盖这一张。
function photoPayload(item) {
  const payload = {
    file_url: item.file_url,
    thumb_url: item.thumb_url || "",
    width: item.width || 0,
    height: item.height || 0
  }
  if (item.id) payload.id = item.id
  if (item.file_id) payload.file_id = item.file_id
  if (Array.isArray(item.student_ids)) payload.student_ids = [...item.student_ids]
  return payload
}

async function submitForm() {
  if (!canSubmit.value) {
    if (blockReasons.value.length) {
      uni.showToast({ title: blockReasons.value[0], icon: "none" })
    }
    return
  }
  submitting.value = true
  uploadError.value = ""
  const payload = {
    title: form.value.title.trim(),
    category: form.value.category,
    album_date: form.value.date,
    participants: [...form.value.participants],
    photos: donePhotos.value.map(photoPayload)
  }
  const target = isEditing.value ? { ...editingAlbum.value } : null
  try {
    if (target) {
      await updateTeacherAlbum(target.id, payload, { silent: true })
    } else {
      await createTeacherAlbum(payload, { silent: true })
    }
    resetForm()
    uploadOpen.value = false
    uni.showToast({ title: target ? "已保存" : "已上传", icon: "success" })
    // 统计与列表都刷新；保留已渲染内容，避免整页闪骨架
    await loadAlbums({ skeleton: false })
    if (target) {
      // 回到详情并重新拉取，编辑结果立即可见
      openViewer({
        id: target.id,
        title: payload.title,
        category: payload.category,
        album_date: payload.album_date,
        status: target.status,
        cover_url: payload.photos[0]?.file_url || "",
        photo_count: payload.photos.length
      })
    }
  } catch (err) {
    // 编辑失败保留表单，用户可直接改完重试
    uploadError.value = err?.message || (target ? "保存失败，请重试" : "上传失败，请重试")
    console.warn("[album-index] submit album failed:", err?.message || err)
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

        <view class="status-filter">
          <view
            v-for="item in STATUS_FILTERS"
            :key="'status-' + (item.value || 'all')"
            class="status-chip"
            :class="{ 'is-active': activeStatus === item.value }"
            @click="selectStatus(item.value)"
          >
            <text>{{ item.label }}</text>
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

        <!-- 筛选无结果：统计与筛选保留，列表区给提示 -->
        <view v-if="viewState === 'filterEmpty'" class="filter-empty">
          <text class="filter-empty-title">{{ filterEmptyTitle }}</text>
          <text class="filter-empty-copy">换个筛选条件看看，或查看全部照片</text>
          <view class="btn-soft btn-soft--sm" @click="resetFilters">查看全部</view>
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
                    <view class="album-tags">
                      <view v-if="statusLabel(album.status)" class="tag" :class="'tag--status-' + album.status">
                        <text>{{ statusLabel(album.status) }}</text>
                      </view>
                      <view class="tag" :class="albumTagClass(album.category)">
                        <text>{{ album.category_label || albumCategoryLabel(album.category) }}</text>
                      </view>
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

        <!-- 状态与未授权计数：详情返回后才有；计数 > 0 时发布需二次确认 -->
        <view v-if="viewerStatus" class="viewer-meta">
          <view class="tag" :class="'tag--status-' + viewerStatus">
            <text>{{ viewerStatusLabel }}</text>
          </view>
          <text
            v-if="viewerUnauthText"
            class="viewer-unauth"
            :class="{ 'is-warning': viewerUnauthorizedCount > 0 }"
          >
            {{ viewerUnauthText }}
          </text>
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

        <!-- 详情加载成功后才给操作入口：未拿到 unauthorized_photo_count 时不能发布 -->
        <view v-if="viewerDetail && !viewerLoading" class="viewer-actions">
          <view
            class="viewer-action viewer-action--neutral"
            :class="{ 'is-busy': actionBusy }"
            @click="onViewerEdit"
          >
            <text>编辑</text>
          </view>
          <view
            v-if="viewerStatus === 'published'"
            class="viewer-action viewer-action--primary"
            :class="{ 'is-busy': actionBusy }"
            @click="onViewerUnpublish"
          >
            <text>下架</text>
          </view>
          <view
            v-else
            class="viewer-action viewer-action--primary"
            :class="{ 'is-busy': actionBusy }"
            @click="onViewerPublish"
          >
            <text>{{ viewerStatus === "unpublished" ? "重新发布" : "发布" }}</text>
          </view>
          <view
            class="viewer-action viewer-action--danger"
            :class="{ 'is-busy': actionBusy }"
            @click="onViewerDelete"
          >
            <text>删除</text>
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
          <text class="sheet-title">{{ sheetTitle }}</text>
          <view class="sheet-close" :aria-label="isEditing ? '关闭编辑' : '关闭上传'" @click="closeUpload">
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
            <text class="field-label">参与幼儿</text>
            <view class="field-picker" @click="openAlbumParticipants">
              <text class="field-picker-value field-picker-value--ellipsis">{{ participantsText }}</text>
              <text class="icon field-picker-icon">&#xEA4E;</text>
            </view>
            <text v-if="studentsError" class="field-hint is-warning">
              {{ isEditing ? "班级名单加载失败，点击上方可重试；提交前需确认参与者" : "班级名单加载失败，点击上方可重试；提交前需补齐名单" }}
            </text>
            <text v-else-if="staleParticipants.length" class="field-hint is-warning">
              已不在班级名单：{{ staleParticipantsText }}，请打开参与幼儿取消选中
            </text>
            <text v-else class="field-hint">未逐张调整的照片都按这份名单</text>
          </view>

          <view class="field">
            <view class="field-head">
              <text class="field-label field-label--inline">照片</text>
              <text class="field-note">{{ mediaList.length }} / {{ MAX_PHOTOS }}</text>
            </view>
            <view class="upload-grid">
              <view
                v-for="(item, index) in mediaList"
                :key="index"
                class="upload-tile"
                :aria-label="'第 ' + (index + 1) + ' 张，' + photoParticipantsAria(item)"
                @click="onPhotoTileTap(item, index)"
              >
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
                <view v-if="item.status === 'done'" class="upload-people">
                  <text class="upload-people-text">{{ photoParticipantsText(item) }}</text>
                </view>
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

        <view class="upload-submit" :class="{ 'is-disabled': !canSubmit }" @click="submitForm">
          {{ submitLabel }}
        </view>
      </view>
    </view>

    <!-- 参与幼儿选择器：二级弹层，盖在上传/编辑弹层之上 -->
    <view v-if="pickerOpen" class="picker-layer">
      <view class="picker-scrim" @click="closePicker" @touchmove.stop.prevent="noop" />
      <view class="picker-sheet">
        <view class="picker-head">
          <view class="picker-cancel" @click="closePicker">取消</view>
          <text class="picker-title">{{ pickerTitle }}</text>
          <view v-if="students.length" class="picker-all" @click="togglePickerAll">
            {{ pickerAllSelected ? "取消全选" : "全选" }}
          </view>
          <view v-else class="picker-all" />
        </view>
        <text class="picker-desc">{{ pickerDesc }}</text>
        <view v-if="studentsLoading" class="picker-warn is-busy">正在加载班级名单…</view>
        <view v-else-if="studentsError" class="picker-warn" @click="retryStudents">
          班级名单加载失败，点击重试
        </view>
        <scroll-view scroll-y class="picker-list">
          <!-- 已离班/已删除的参与者：显式列出，老师可取消选中后提交 -->
          <view
            v-for="student in pickerStaleStudents"
            :key="'stale-' + student.id"
            class="picker-row is-stale"
            :class="{ 'is-on': pickerPicked.has(Number(student.id)) }"
            @click="togglePickerStudent(student.id)"
          >
            <view class="picker-avatar">
              <image v-if="student.avatar" class="picker-avatar-img" :src="student.avatar" mode="aspectFill" />
              <text v-else>{{ String(student.name || "?").slice(0, 1) }}</text>
            </view>
            <view class="picker-info">
              <text class="picker-name">{{ student.name || "已离班幼儿" }}</text>
              <text class="picker-status is-stale">已离班/已删除</text>
            </view>
            <view class="picker-check"><text class="icon">&#xEB7B;</text></view>
          </view>
          <view
            v-for="student in students"
            :key="student.student_id"
            class="picker-row"
            :class="{ 'is-on': pickerPicked.has(Number(student.student_id)) }"
            @click="togglePickerStudent(student.student_id)"
          >
            <view class="picker-avatar">
              <image v-if="student.avatar" class="picker-avatar-img" :src="student.avatar" mode="aspectFill" />
              <text v-else>{{ String(student.name || "?").slice(0, 1) }}</text>
            </view>
            <view class="picker-info">
              <text class="picker-name">{{ student.name || "未命名" }}</text>
              <text class="picker-status">{{ pickupStatusLabel(student.pickup_status) }}</text>
            </view>
            <view class="picker-check"><text class="icon">&#xEB7B;</text></view>
          </view>
          <view
            v-if="!students.length && !pickerStaleStudents.length && !studentsLoading"
            class="picker-empty"
          >
            {{ studentsError ? "名单未加载，请先重试" : "本班暂无在读幼儿" }}
          </view>
        </scroll-view>
        <view class="picker-foot">
          <view v-if="pickerCanRestore" class="picker-restore" @click="restorePhotoParticipants">
            恢复为相册默认
          </view>
          <view class="picker-confirm" @click="confirmPicker">
            确定（已选 {{ pickerPicked.size }} 人）
          </view>
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

/* ---------- 状态分段控件（紧凑，与分类胶囊区分） ---------- */
.status-filter {
  display: flex;
  margin-top: $spacing-lg;
  padding: $spacing-xs;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.status-chip + .status-chip {
  margin-left: $spacing-xs;
}

.status-chip {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.status-chip.is-active {
  border-color: $color-border-subtle;
  background: $color-bg-card;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.status-chip:active {
  transform: scale(0.96);
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

.album-tags {
  flex: none;
  display: flex;
  align-items: center;
}

.album-tags .tag + .tag {
  margin-left: $spacing-xs;
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

/* 发布状态：草稿中性、已发布成功色、已下架警告色 */
.tag--status-draft {
  background: $color-bg-page;
  color: $color-text-muted;
}

.tag--status-published {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.tag--status-unpublished {
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

.viewer-meta {
  display: flex;
  align-items: center;
  margin-bottom: $spacing-md;
}

.viewer-unauth {
  min-width: 0;
  margin-left: $spacing-sm;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  color: $color-text-muted;
}

.viewer-unauth.is-warning {
  color: $color-semantic-warning-strong;
  font-weight: $typography-font-weight-semibold;
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

.viewer-actions {
  display: flex;
  margin-top: $spacing-lg;
}

.viewer-action + .viewer-action {
  margin-left: $spacing-md;
}

.viewer-action {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.viewer-action--primary {
  border: 1rpx solid rgba($color-brand-primary, 0.45);
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.viewer-action--danger {
  border: 1rpx solid rgba($color-semantic-danger, 0.35);
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.viewer-action--neutral {
  border: 1rpx solid rgba($color-border-hairline, 0.5);
  background: $color-bg-page;
  color: $color-text-body;
}

.viewer-action:active {
  transform: scale(0.97);
}

.viewer-action.is-busy {
  opacity: 0.6;
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

.field-picker-value--ellipsis {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.field-hint {
  display: block;
  margin-top: $spacing-xs;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  color: $color-text-muted;
}

.field-hint.is-warning {
  color: $color-semantic-warning-strong;
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

/* 逐张参与幼儿摘要：与右上角删除按钮错开，长了就省略 */
.upload-people {
  position: absolute;
  top: 4rpx;
  left: 4rpx;
  max-width: calc(100% - 60rpx);
  padding: 2rpx 10rpx;
  border-radius: $border-radius-full;
  background: rgba($color-text-primary, 0.62);
}

.upload-people-text {
  display: block;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  color: $color-text-on-brand;
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

/* ---------- 参与幼儿选择器 ---------- */
.picker-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 110;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.picker-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba($color-text-primary, 0.5);
}

.picker-sheet {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 430px;
  max-height: 80vh;
  padding: $spacing-xs $component-page-padding-x calc(#{$spacing-md} + env(safe-area-inset-bottom));
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.picker-head {
  display: grid;
  grid-template-columns: 128rpx minmax(0, 1fr) 128rpx;
  align-items: center;
  min-height: 96rpx;
}

.picker-cancel,
.picker-all {
  display: flex;
  align-items: center;
  min-height: $component-input-height;
  font-size: $typography-font-size-md;
}

.picker-cancel {
  color: $color-text-muted;
}

.picker-all {
  justify-content: flex-end;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

.picker-title {
  text-align: center;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
}

.picker-desc {
  padding: 0 2rpx $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.picker-warn {
  margin-bottom: $spacing-xs;
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius-lg;
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.picker-warn.is-busy {
  background: $color-bg-page;
  color: $color-text-muted;
  opacity: 0.8;
}

.picker-list {
  flex: 1;
  min-height: 0;
  height: 560rpx;
}

.picker-row {
  display: grid;
  grid-template-columns: 72rpx minmax(0, 1fr) 44rpx;
  align-items: center;
  gap: $spacing-sm;
  min-height: 104rpx;
  padding: $spacing-xs 2rpx;
  border-radius: $border-radius-lg;
}

.picker-row + .picker-row {
  border-top: 1rpx solid $color-border-subtle;
}

.picker-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  overflow: hidden;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

.picker-avatar-img {
  width: 100%;
  height: 100%;
}

.picker-row.is-stale .picker-avatar {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
}

.picker-info {
  display: flex;
  flex-direction: column;
  gap: 2rpx;
  min-width: 0;
}

.picker-name {
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
}

.picker-status {
  color: $color-text-hint;
  font-size: $typography-font-size-xs;
  line-height: 1.3;
}

.picker-status.is-stale {
  color: $color-semantic-warning-strong;
  font-weight: $typography-font-weight-semibold;
}

.picker-check {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44rpx;
  height: 44rpx;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  color: transparent;
  font-size: $typography-font-size-xs;
}

.picker-row.is-on .picker-avatar {
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
}

.picker-row.is-on .picker-check {
  border-color: $color-brand-primary;
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
}

.picker-empty {
  padding: $spacing-xl $spacing-xs;
  text-align: center;
  color: $color-text-hint;
  font-size: $typography-font-size-base;
}

.picker-foot {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  padding-top: $spacing-sm;
}

.picker-restore {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  padding: 0 $spacing-md;
  border: 1rpx solid rgba($color-border-hairline, 0.5);
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.picker-confirm {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  min-height: $component-input-height;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-md;
}

.picker-restore:active,
.picker-confirm:active {
  transform: scale(0.97);
}
</style>
