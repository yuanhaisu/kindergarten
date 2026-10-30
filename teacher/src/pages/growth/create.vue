<script setup>
import { computed, reactive, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import {
  createTeacherGrowthLog,
  fetchTeacherGrowthLog,
  updateTeacherGrowthLog,
  uploadGrowthMedia,
  fetchPresentStudents
} from "@/api/teacher-growth"
import { fetchTeacherClassStudents, fetchTeacherClassInfo } from "@/api/teacher-class"
import { fetchDict } from "@/api/dict"
import { fetchTeacherAchievements, grantAchievement } from "@/api/teacher-achievements"
import { addDays, formatDateAnchor, todayKey } from "@/utils/time"
import { pickImages } from "@/utils/media"
import PageShell from "@/components/PageShell.vue"

const MAX_PHOTOS = 20
const MAX_PHOTO_BYTES = 20 * 1024 * 1024
const UPLOAD_TIMEOUT_MS = 30000

const submitting = ref(false)
const submitError = ref("")
const students = ref([])
const className = ref("")
const classStudentCount = ref(0)

const form = ref({
  date: todayKey(),
  breakfast_rating: 0,
  lunch_rating: 0,
  dinner_rating: 0,
  nap_duration: 0,
  sleep_quality: 0,
  morning_mood: 0,
  afternoon_mood: 0,
  performance_tags: [],
  content: "",
  media: [],
  sync_to_feed: true,
  target_type: 1,
  student_ids: []
})

const editId = ref(null)
const isEdit = computed(() => editId.value != null)
const dateAnchor = computed(() => formatDateAnchor(form.value.date))
const nextDisabled = computed(() => isEdit.value || form.value.date >= todayKey())
const nextDayHint = computed(() => (nextDisabled.value ? "" : "后一天"))
const childCount = computed(() => classStudentCount.value || students.value.length)

const pickedStudents = computed(() =>
  form.value.student_ids
    .map(id => students.value.find(s => s.student_id === id))
    .filter(Boolean)
)
const pickedPreview = computed(() => pickedStudents.value.slice(0, 3))
const pickedText = computed(() => {
  const names = pickedStudents.value.map(s => s.name)
  if (!names.length) {
    return form.value.student_ids.length ? `已选 ${form.value.student_ids.length} 人` : "还没有选择幼儿"
  }
  if (names.length <= 3) return names.join("、")
  return `${names.slice(0, 3).join("、")} 等 ${names.length} 人`
})

const meals = [
  { field: "breakfast_rating", label: "早餐" },
  { field: "lunch_rating", label: "午餐" },
  { field: "dinner_rating", label: "晚餐" }
]
const moods = [
  { field: "morning_mood", label: "上午" },
  { field: "afternoon_mood", label: "下午" }
]
const ratingOptions = [
  { value: 3, emoji: "😋", label: "好" },
  { value: 2, emoji: "😐", label: "一般" },
  { value: 1, emoji: "😞", label: "不好" }
]
const sleepOptions = [1, 2, 3]
const qualityOptions = [
  { value: 3, label: "很好" },
  { value: 2, label: "一般" },
  { value: 1, label: "不好" }
]

const performanceOptions = ref([])

function prevDay() {
  if (isEdit.value) return
  form.value.date = addDays(form.value.date, -1)
}

function nextDay() {
  if (isEdit.value) return
  const next = addDays(form.value.date, 1)
  if (next <= todayKey()) form.value.date = next
}

function setMode(mode) {
  if (isEdit.value || form.value.target_type === mode) return
  form.value.target_type = mode
  if (mode === 1) {
    form.value.student_ids = []
    form.value.sync_to_feed = true
  } else {
    form.value.sync_to_feed = false
  }
}

function setRating(field, value) {
  form.value[field] = form.value[field] === value ? 0 : value
}

function toggleTag(tag) {
  const list = form.value.performance_tags
  const index = list.indexOf(tag)
  if (index >= 0) list.splice(index, 1)
  else list.push(tag)
}

// ---- 选择幼儿 ----
const sheetOpen = ref(false)
const sheetMode = ref("pick") // pick | single | batch
const sheetQuery = ref("")
const sheetPicked = ref(new Set())
const sheetTargets = ref([])

const sheetCandidates = computed(() => {
  let pool = students.value
  if (form.value.target_type === 2 && sheetMode.value !== "pick") {
    pool = pool.filter(s => form.value.student_ids.includes(s.student_id))
  }
  const query = sheetQuery.value.trim().toLowerCase()
  if (!query) return pool
  return pool.filter(s => String(s.name || "").toLowerCase().includes(query))
})

const sheetBatchExisting = computed(() => {
  if (sheetMode.value !== "batch") return new Set()
  const set = new Set()
  sheetTargets.value.forEach(index => {
    const media = form.value.media[index]
    ;(media?.student_ids || []).forEach(id => set.add(id))
  })
  return set
})

const sheetSelectedCount = computed(() => sheetPicked.value.size)

const allSheetSelected = computed(() => {
  if (!sheetCandidates.value.length) return false
  return sheetCandidates.value.every(s =>
    sheetPicked.value.has(s.student_id) || sheetBatchExisting.value.has(s.student_id)
  )
})

const sheetTitle = computed(() => {
  if (sheetMode.value === "single") return "照片点名"
  if (sheetMode.value === "batch") return "批量点名"
  return "选择幼儿"
})

const sheetDesc = computed(() => {
  if (sheetMode.value === "single") return "勾选这张照片里的孩子（AI 建议可一并确认）"
  if (sheetMode.value === "batch") {
    return `勾选的孩子会追加到所选 ${sheetTargets.value.length} 张照片，已有名单保留`
  }
  return `${className.value || "本班"} · 共 ${students.value.length} 位 · 已选 ${sheetPicked.value.size} 人`
})

function reloadSheetPicked(next) {
  sheetPicked.value = next
}

function openPickSheet() {
  sheetMode.value = "pick"
  sheetQuery.value = ""
  reloadSheetPicked(new Set(form.value.student_ids))
  sheetOpen.value = true
}

function openSingleNaming(index) {
  if (batchMode.value) {
    toggleBatchSelect(index)
    return
  }
  const media = form.value.media[index]
  if (!media || media.status !== "done") return
  sheetMode.value = "single"
  sheetTargets.value = [index]
  sheetQuery.value = ""
  // 预勾：已确认名单 + 高置信建议（阈值由服务端判定，前端不硬编码）
  const pre = new Set(media.student_ids || [])
  ;(media.suggestions || []).forEach(s => {
    if (s.tier === "high") pre.add(s.student_id)
  })
  reloadSheetPicked(pre)
  sheetOpen.value = true
}

function openBatchNaming() {
  const targets = [...batchSelected.value].sort((a, b) => a - b)
  if (!targets.length) {
    uni.showToast({ title: "请先选择照片", icon: "none" })
    return
  }
  sheetMode.value = "batch"
  sheetTargets.value = targets
  sheetQuery.value = ""
  reloadSheetPicked(new Set()) // 批量 = 追加语义，从空开始
  sheetOpen.value = true
}

function toggleSheetStudent(id) {
  if (sheetBatchExisting.value.has(id)) {
    uni.showToast({ title: "该孩子已在照片上（批量只做追加）", icon: "none" })
    return
  }
  const next = new Set(sheetPicked.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  reloadSheetPicked(next)
}

function toggleSelectAllSheet() {
  if (allSheetSelected.value) {
    const keep = new Set()
    sheetCandidates.value.forEach(s => {
      if (sheetBatchExisting.value.has(s.student_id)) keep.add(s.student_id)
    })
    reloadSheetPicked(keep)
    return
  }
  const next = new Set(sheetPicked.value)
  sheetCandidates.value.forEach(s => {
    if (!sheetBatchExisting.value.has(s.student_id)) next.add(s.student_id)
  })
  reloadSheetPicked(next)
}

function closeSheet() {
  sheetOpen.value = false
}

function confirmSheet() {
  const picked = [...sheetPicked.value]
  if (sheetMode.value === "pick") {
    form.value.student_ids = picked
  } else if (sheetMode.value === "single") {
    const media = form.value.media[sheetTargets.value[0]]
    if (media) {
      media.student_ids = picked
      media._named = true
    }
    uni.showToast({ title: picked.length ? `已点名 ${picked.length} 位幼儿` : "已清空这张照片的点名", icon: "none" })
  } else {
    if (!picked.length) {
      closeSheet()
      cancelBatch()
      uni.showToast({ title: "未选择新增孩子，未做改动", icon: "none" })
      return
    }
    sheetTargets.value.forEach(index => {
      const media = form.value.media[index]
      if (!media) return
      const merged = new Set(media.student_ids || [])
      picked.forEach(id => merged.add(id))
      media.student_ids = [...merged]
      media._named = true
    })
    cancelBatch()
  }
  closeSheet()
}

// ---- 照片 ----
const batchMode = ref(false)
const batchSelected = ref(new Set())

const doneMediaIndexes = computed(() =>
  form.value.media.map((media, index) => (media.status === "done" ? index : -1)).filter(i => i >= 0)
)
const allPhotosSelected = computed(
  () => doneMediaIndexes.value.length > 0 && batchSelected.value.size === doneMediaIndexes.value.length
)
const namedPhotoCount = computed(
  () => form.value.media.filter(media => media.status === "done" && (media.student_ids || []).length).length
)
const uploadingCount = computed(() => form.value.media.filter(media => media.status === "uploading").length)
const failedCount = computed(() => form.value.media.filter(media => media.status === "failed").length)

function startBatch() {
  if (!form.value.media.length) {
    uni.showToast({ title: "先添加照片，再点名", icon: "none" })
    return
  }
  if (!doneMediaIndexes.value.length) {
    uni.showToast({ title: "等照片上传完成，再点名", icon: "none" })
    return
  }
  batchMode.value = true
  batchSelected.value = new Set()
}

function cancelBatch() {
  batchMode.value = false
  batchSelected.value = new Set()
}

function toggleBatchSelect(index) {
  const media = form.value.media[index]
  if (!media || media.status !== "done") return
  const next = new Set(batchSelected.value)
  if (next.has(index)) next.delete(index)
  else next.add(index)
  batchSelected.value = next
}

function toggleSelectAllPhotos() {
  batchSelected.value = allPhotosSelected.value ? new Set() : new Set(doneMediaIndexes.value)
}

function mediaDisplayUrl(media) {
  return media.thumb_url || media.file_url || media._temp || ""
}

function mediaBadgeText(media) {
  const named = (media.student_ids || []).length
  if (named) return `点名 ${named}`
  const suggestions = (media.suggestions || []).length
  if (suggestions) return `AI 建议 ${suggestions}`
  return "待点名"
}

function withTimeout(promise, ms) {
  return Promise.race([
    promise,
    new Promise((_, reject) => setTimeout(() => reject(new Error("上传超时")), ms))
  ])
}

function fillUploaded(media, data) {
  media.file_id = data.id
  media.file_url = data.file_url
  media.thumb_url = data.thumb_url || data.file_url
  media.file_type = data.file_type || "image"
  media.progress = 100
  media.status = "done"
}

async function uploadOne(tempPath) {
  // 必须用 reactive：直接 push 裸对象后改 raw 属性不会触发渲染，
  // 进度百分比与「上传中 → 完成」的状态迁移就永远停在首帧。
  const media = reactive({
    file_id: 0,
    file_url: "",
    thumb_url: "",
    file_type: "image",
    ref_desc: "",
    status: "uploading",
    progress: 0,
    _temp: tempPath,
    student_ids: [],
    suggestions: [],
    _named: true
  })
  form.value.media.push(media)
  try {
    const res = await withTimeout(
      uploadGrowthMedia(tempPath, progress => {
        if (media.status === "uploading") media.progress = progress
      }),
      UPLOAD_TIMEOUT_MS
    )
    fillUploaded(media, res.data || {})
  } catch (err) {
    media.status = "failed"
  }
}

async function pickPhotos() {
  const remain = MAX_PHOTOS - form.value.media.length
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
    uni.showToast({ title: `有 ${rejected} 张照片超过 20MB，未添加`, icon: "none" })
  }
}

function retryMedia(index) {
  const media = form.value.media[index]
  if (!media || media.status !== "failed" || !media._temp) return
  media.status = "uploading"
  media.progress = 0
  withTimeout(
    uploadGrowthMedia(media._temp, progress => {
      if (media.status === "uploading") media.progress = progress
    }),
    UPLOAD_TIMEOUT_MS
  )
    .then(res => fillUploaded(media, res.data || {}))
    .catch(() => {
      media.status = "failed"
    })
}

function removeMedia(index) {
  form.value.media.splice(index, 1)
}

// ---- 徽章（可选） ----
const achievements = ref([])
const grant = ref({ achievement_id: null, student_ids: [], as_witness: true })

function selectAchievement(id) {
  grant.value.achievement_id = grant.value.achievement_id === id ? null : id
}

function toggleGrantStudent(id) {
  const list = grant.value.student_ids
  const index = list.indexOf(id)
  if (index >= 0) list.splice(index, 1)
  else list.push(id)
}

async function loadAchievements() {
  try {
    const res = await fetchTeacherAchievements()
    achievements.value = res.data?.list || []
  } catch (err) {
    // 徽章列表失败不阻塞发布——授予是可选动作
    console.warn("[growth-create] load achievements failed:", err?.message || err)
  }
}

async function grantAfterPublish(logId) {
  const current = grant.value
  if (!current.achievement_id) return
  const target = current.student_ids.length ? current.student_ids : form.value.student_ids
  if (!target.length) {
    uni.showToast({ title: "请选择要授予的学生", icon: "none" })
    return
  }
  try {
    await grantAchievement(current.achievement_id, {
      student_ids: target,
      log_id: current.as_witness ? logId : undefined
    })
    uni.showToast({ title: "已授予徽章", icon: "success" })
  } catch (err) {
    // 授予失败不影响已发布的记录——单独提示
    uni.showToast({ title: err?.message || "授予徽章失败", icon: "none" })
  }
}

// ---- 加载 ----
async function loadStudents() {
  try {
    const res = await fetchTeacherClassStudents()
    students.value = res.data?.list || []
  } catch (err) {
    // 学生列表失败不阻塞页面；选择幼儿时会提示暂无名单
    console.warn("[growth-create] load students failed:", err?.message || err)
  }
}

async function loadClassInfo() {
  try {
    const res = await fetchTeacherClassInfo()
    className.value = res.data?.name || ""
    classStudentCount.value = Number(res.data?.student_count || 0)
  } catch (err) {
    console.warn("[growth-create] load class failed:", err?.message || err)
  }
}

async function loadPerformanceTags() {
  try {
    const res = await fetchDict("performance_tag")
    performanceOptions.value = (res.data?.items || []).map(item => item.item_label)
  } catch (err) {
    performanceOptions.value = [
      "乐于助人", "积极发言", "遵守纪律", "动手能力强",
      "专注力强", "有礼貌", "爱干净", "团结友善", "勇于尝试"
    ]
  }
}

async function loadExistingLog(id) {
  try {
    const res = await fetchTeacherGrowthLog(id)
    const record = res.data || {}
    const f = form.value
    f.content = record.content || ""
    f.breakfast_rating = record.breakfast_rating || 0
    f.lunch_rating = record.lunch_rating || 0
    f.dinner_rating = record.dinner_rating || 0
    f.nap_duration = record.nap_duration || 0
    f.sleep_quality = record.sleep_quality || 0
    f.morning_mood = record.morning_mood || 0
    f.afternoon_mood = record.afternoon_mood || 0
    f.performance_tags = Array.isArray(record.performance_tags) ? [...record.performance_tags] : []
    f.target_type = record.target_type === 2 ? 2 : 1
    f.sync_to_feed = record.sync_to_feed !== false
    if (record.date) f.date = String(record.date).slice(0, 10)
    f.student_ids = (record.related_students || []).map(s => s.id)
    // 既有照片回显（编辑态）：保留 media.id 供服务端识别「保留项」，ref_desc 可继续编辑
    f.media = (record.media || []).map(media => ({
      id: media.id || 0,
      file_id: media.file_id || 0,
      file_url: media.file_url || "",
      thumb_url: media.thumb_url || media.file_url || "",
      file_type: media.file_type || "image",
      ref_desc: media.ref_desc || "",
      status: "done",
      progress: 100,
      student_ids: Array.isArray(media.student_ids) ? [...media.student_ids] : [],
      suggestions: Array.isArray(media.student_suggestions) ? media.student_suggestions : [],
      _named: false
    }))
  } catch (err) {
    uni.showToast({ title: err?.message || "加载记录失败", icon: "none" })
  }
}

onLoad(options => {
  const id = options?.id
  if (id) {
    editId.value = id
    loadExistingLog(id)
  }
  loadStudents()
  loadClassInfo()
  loadPerformanceTags()
  loadAchievements()
})

// ---- 提交 ----
const blockReasons = computed(() => {
  const reasons = []
  if (form.value.target_type === 2 && form.value.student_ids.length === 0) {
    reasons.push("请先选择至少一位幼儿")
  }
  if (!form.value.content.trim()) reasons.push("请输入记录内容")
  if (uploadingCount.value) reasons.push(`${uploadingCount.value} 张照片上传中，完成后可发布`)
  if (failedCount.value) reasons.push(`有 ${failedCount.value} 张照片上传失败，请重试或移除`)
  return reasons
})
const canSubmit = computed(() => blockReasons.value.length === 0 && !submitting.value)
const bottomSpace = computed(() => (submitError.value ? "440rpx" : "300rpx"))

const scopeText = computed(() => {
  if (isEdit.value) {
    return form.value.target_type === 2 ? "该记录仅被选中的幼儿家长可见" : "该记录对全班家长可见"
  }
  if (form.value.target_type === 2) {
    return form.value.student_ids.length
      ? `仅 ${form.value.student_ids.length} 位幼儿家长可见`
      : "仅选中的幼儿家长可见"
  }
  return `${className.value || "本班"}全体家长可见`
})

const publishLabel = computed(() => {
  if (submitting.value) return isEdit.value ? "保存中…" : "发布中…"
  if (submitError.value) return isEdit.value ? "重新保存" : "重新发布"
  if (isEdit.value) return "保存修改"
  return form.value.target_type === 2 ? "发布个体记录" : "发布班级记录"
})

// 提交载荷：只送引用（id/file_id/ref_desc），URL 等物理属性由服务端反查。
// student_ids 仅在老师动过这张照片的点名时携带（_named），未动过的照片不触碰既有点名。
function mediaPayload() {
  return form.value.media.map(media => {
    const item = { id: media.id || 0, file_id: media.file_id, ref_desc: media.ref_desc || "" }
    if (media._named) item.student_ids = media.student_ids || []
    return item
  })
}

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  submitError.value = ""
  try {
    const payload = { ...form.value, media: mediaPayload() }
    if (isEdit.value) {
      await updateTeacherGrowthLog(editId.value, payload, { silent: true })
      uni.showToast({ title: "已保存", icon: "success" })
    } else {
      const created = await createTeacherGrowthLog(payload, { silent: true })
      await grantAfterPublish(created?.data?.id)
      uni.showToast({ title: "已发布", icon: "success" })
    }
    setTimeout(() => uni.navigateBack(), 600)
  } catch (err) {
    submitError.value = err?.message || (isEdit.value ? "保存失败" : "发布失败")
  } finally {
    submitting.value = false
  }
}

async function fetchRosterList() {
  const res = await fetchPresentStudents(String(form.value.date || "").slice(0, 10))
  return res.data?.list || []
}

// 课堂点名面板里的「全班（当日到场）」快捷入口：填入后仍可逐人增减。
const rosterLoading = ref(false)
async function pickWholeClass() {
  if (rosterLoading.value) return
  rosterLoading.value = true
  try {
    const present = await fetchRosterList()
    // 名单范围与「可点名学生」一致：个体记录只允许本记录的目标幼儿
    const scope = form.value.target_type === 2
      ? students.value.filter(s => form.value.student_ids.includes(s.student_id))
      : students.value
    const allowed = new Set(scope.map(s => s.student_id))
    const candidates = present.filter(id => allowed.has(id))
    const picked = sheetMode.value === "batch"
      ? candidates.filter(id => !sheetBatchExisting.value.has(id))
      : candidates
    if (!picked.length) {
      uni.showToast({ title: "当日到场名单为空", icon: "none" })
      return
    }
    reloadSheetPicked(new Set(picked))
    uni.showToast({ title: `已选当日到场 ${picked.length} 人，可增减`, icon: "none" })
  } catch (err) {
    uni.showToast({ title: err?.message || "获取到场名单失败", icon: "none" })
  } finally {
    rosterLoading.value = false
  }
}
</script>

<template>
  <PageShell :bottom-space="bottomSpace">
    <view class="page">
      <!-- 日期锚点 -->
      <view class="day">
        <view class="day-arrow" :class="{ 'is-disabled': isEdit }" @click="prevDay">
          <text class="icon">&#xea64;</text>
        </view>
        <view class="day-main">
          <text class="day-date">{{ dateAnchor.main }}</text>
          <text class="day-sub">{{ dateAnchor.sub }}</text>
        </view>
        <view class="day-arrow" :class="{ 'is-disabled': nextDisabled }" @click="nextDay" :aria-label="nextDayHint">
          <text class="icon">&#xea6e;</text>
        </view>
      </view>

      <!-- 记录对象：填写前先明确可见范围 -->
      <view class="mode">
        <view
          class="mode-btn"
          :class="{ 'is-on': form.target_type === 1, 'is-disabled': isEdit }"
          @click="setMode(1)"
        >全班记录</view>
        <view
          class="mode-btn"
          :class="{ 'is-on': form.target_type === 2, 'is-disabled': isEdit }"
          @click="setMode(2)"
        >指定幼儿</view>
      </view>

      <!-- 快速记录（仅全班） -->
      <view v-if="form.target_type === 1" class="card">
        <view class="card-head">
          <text class="card-title">快速记录</text>
          <text class="card-note">可跳过</text>
        </view>
        <view class="obs-body">
          <view class="obs-row">
            <text class="obs-label">饮食</text>
            <view class="obs-main">
              <view v-for="meal in meals" :key="meal.field" class="meal">
                <text class="meal-name">{{ meal.label }}</text>
                <view class="rates">
                  <view
                    v-for="option in ratingOptions"
                    :key="option.value"
                    class="rate"
                    :class="{ 'is-on': form[meal.field] === option.value }"
                    @click="setRating(meal.field, option.value)"
                  >{{ option.emoji }}</view>
                </view>
              </view>
            </view>
          </view>

          <view class="obs-row">
            <text class="obs-label">睡眠</text>
            <view class="obs-main">
              <text class="sub-label">午睡时长</text>
              <view class="chips">
                <view
                  v-for="hours in sleepOptions"
                  :key="hours"
                  class="chip"
                  :class="{ 'is-on': form.nap_duration === hours }"
                  @click="form.nap_duration = form.nap_duration === hours ? 0 : hours"
                >{{ hours }} 小时</view>
              </view>
              <text class="sub-label">睡眠质量</text>
              <view class="chips">
                <view
                  v-for="option in qualityOptions"
                  :key="option.value"
                  class="chip"
                  :class="{ 'is-on': form.sleep_quality === option.value }"
                  @click="form.sleep_quality = form.sleep_quality === option.value ? 0 : option.value"
                >{{ option.label }}</view>
              </view>
            </view>
          </view>

          <view class="obs-row">
            <text class="obs-label">情绪</text>
            <view class="obs-main">
              <view v-for="mood in moods" :key="mood.field" class="meal">
                <text class="meal-name">{{ mood.label }}</text>
                <view class="rates">
                  <view
                    v-for="option in ratingOptions"
                    :key="option.value"
                    class="rate"
                    :class="{ 'is-on': form[mood.field] === option.value }"
                    @click="setRating(mood.field, option.value)"
                  >{{ option.emoji }}</view>
                </view>
              </view>
            </view>
          </view>

          <view class="obs-row">
            <text class="obs-label">表现</text>
            <view class="obs-main">
              <view class="tags">
                <view
                  v-for="tag in performanceOptions"
                  :key="tag"
                  class="tag"
                  :class="{ 'is-on': form.performance_tags.includes(tag) }"
                  @click="toggleTag(tag)"
                >{{ tag }}</view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <!-- 指定幼儿：已选摘要 / 未选空态按真实人数互斥 -->
      <view v-if="form.target_type === 2" class="card">
        <view class="card-head">
          <text class="card-title">选择幼儿</text>
          <text class="card-note">已选 {{ form.student_ids.length }} 人</text>
        </view>

        <view v-if="form.student_ids.length" class="picked">
          <view class="avatars">
            <view v-for="student in pickedPreview" :key="student.student_id" class="ava">
              {{ String(student.name || "?").slice(0, 1) }}
            </view>
            <view v-if="pickedStudents.length > 3" class="ava ava-more">+{{ pickedStudents.length - 3 }}</view>
          </view>
          <text class="picked-text">{{ pickedText }}</text>
          <view class="link-btn" @click="openPickSheet">编辑</view>
        </view>

        <view v-else class="pick-empty">
          <view class="pick-empty-art">
            <view class="pick-orb orb-a" />
            <view class="pick-orb orb-b" />
            <view class="pick-arc" />
          </view>
          <text class="pick-empty-title">还没有选择幼儿</text>
          <text class="pick-empty-hint">{{ className || "本班" }}共 {{ childCount || 0 }} 位孩子，从名单里挑一位或几位</text>
          <view class="btn-soft" @click="openPickSheet">选择幼儿</view>
        </view>

        <view class="tags-wrap">
          <text class="sub-label">表现标签</text>
          <view class="tags">
            <view
              v-for="tag in performanceOptions"
              :key="tag"
              class="tag"
              :class="{ 'is-on': form.performance_tags.includes(tag) }"
              @click="toggleTag(tag)"
            >{{ tag }}</view>
          </view>
        </view>

        <view class="fineprint">
          <text class="fineprint-icon icon">&#xee0f;</text>
          <text>这条记录只会推给所选幼儿的家长，不出现在班级动态</text>
        </view>
      </view>

      <!-- 照片（仅照片，不做视频） -->
      <view class="card">
        <view class="card-head">
          <text class="card-title">照片</text>
          <text class="card-note">{{ form.media.length }} / {{ MAX_PHOTOS }}</text>
        </view>

        <view class="media-bar">
          <template v-if="batchMode">
            <text class="media-tip">已选 {{ batchSelected.size }} 张照片</text>
            <view class="batch-tools">
              <view class="mini-btn" @click="toggleSelectAllPhotos">{{ allPhotosSelected ? "取消全选" : "全选" }}</view>
              <view class="mini-btn primary" @click="openBatchNaming">选择学生</view>
              <view class="mini-btn" @click="cancelBatch">取消</view>
            </view>
          </template>
          <template v-else>
            <text class="media-tip">已点名 {{ namedPhotoCount }} 张</text>
            <view class="mini-btn" @click="startBatch">批量点名</view>
          </template>
        </view>

        <view class="media-grid">
          <view
            v-for="(media, index) in form.media"
            :key="index"
            class="tile"
            :class="{ 'is-picked': batchMode && batchSelected.has(index) }"
            @click="batchMode ? toggleBatchSelect(index) : openSingleNaming(index)"
          >
            <image
              v-if="mediaDisplayUrl(media)"
              class="tile-img"
              :src="mediaDisplayUrl(media)"
              mode="aspectFill"
            />
            <view
              v-if="media.status === 'uploading'"
              class="tile-mask"
            >
              <view class="tile-spinner" />
              <text>{{ media.progress || 0 }}%</text>
            </view>
            <view
              v-else-if="media.status === 'failed'"
              class="tile-mask is-failed"
              @click.stop="retryMedia(index)"
            >
              <text>上传失败</text>
              <text class="tile-retry">点击重试</text>
            </view>

            <template v-if="!batchMode">
              <view v-if="media.status === 'done'" class="tile-badge" @click.stop="openSingleNaming(index)">
                <text>{{ mediaBadgeText(media) }}</text>
              </view>
              <view class="tile-remove" @click.stop="removeMedia(index)">
                <text class="icon">&#xeb99;</text>
              </view>
            </template>
            <view v-else class="tile-check" :class="{ 'is-on': batchSelected.has(index) }">
              <text class="icon">&#xeb7b;</text>
            </view>
          </view>

          <view class="tile tile-add" @click="pickPhotos">
            <text class="tile-add-icon icon">&#xea13;</text>
            <text class="tile-add-text">添加</text>
          </view>
        </view>

        <view v-if="form.media.length" class="desc-list">
          <view v-for="(media, index) in form.media" :key="'desc-' + index" class="desc-row">
            <view class="desc-thumb">
              <image
                v-if="mediaDisplayUrl(media)"
                class="desc-thumb-img"
                :src="mediaDisplayUrl(media)"
                mode="aspectFill"
              />
            </view>
            <input
              v-model="media.ref_desc"
              class="desc-input"
              type="text"
              placeholder="为这张照片写句说明（可选）"
              placeholder-style="color: #94A3B8"
            />
          </view>
        </view>

        <view class="fineprint">
          <text class="fineprint-dot" />
          <text>拍照建议：尽量拍单人照（AI 自动点名更准）；集体照用「全班」一键点名</text>
        </view>
        <view class="fineprint">
          <text class="fineprint-dot" />
          <text>最多 {{ MAX_PHOTOS }} 张，单张不超过 20MB</text>
        </view>
      </view>

      <!-- 记录内容 -->
      <view class="card">
        <view class="card-head">
          <text class="card-title">记录内容<text class="req-mark">*</text></text>
          <text class="card-note">{{ form.content.length }} 字</text>
        </view>
        <textarea
          v-model="form.content"
          class="textarea"
          :maxlength="-1"
          placeholder="写写今天孩子们做了什么、说了什么"
          placeholder-style="color: #94A3B8"
        />
      </view>

      <!-- 授予徽章（编辑态隐藏） -->
      <view v-if="!isEdit && achievements.length" class="card">
        <view class="card-head">
          <text class="card-title">授予徽章</text>
          <text class="card-note">可选</text>
        </view>
        <view class="badges">
          <view
            v-for="achievement in achievements"
            :key="achievement.id"
            class="badge-chip"
            :class="{ 'is-on': grant.achievement_id === achievement.id }"
            @click="selectAchievement(achievement.id)"
          >
            <image v-if="achievement.icon" class="badge-icon" :src="achievement.icon" mode="aspectFit" />
            <text>{{ achievement.name }}</text>
          </view>
        </view>

        <template v-if="grant.achievement_id">
          <view class="grant">
            <text class="sub-label">授予给</text>
            <view v-if="students.length" class="chips">
              <view
                v-for="student in students"
                :key="student.student_id"
                class="chip"
                :class="{ 'is-on': grant.student_ids.includes(student.student_id) }"
                @click="toggleGrantStudent(student.student_id)"
              >{{ student.name }}</view>
            </view>
            <text v-else class="grant-hint">不选则默认授予给本记录的目标学生</text>
            <view class="check-row" @click="grant.as_witness = !grant.as_witness">
              <view class="box" :class="{ 'is-on': grant.as_witness }">
                <text class="icon">&#xeb7b;</text>
              </view>
              <text>本记录作为获奖见证（家长将在该记录看到徽章）</text>
            </view>
          </view>
        </template>
      </view>
    </view>

    <!-- 幼儿多选弹层 -->
    <view v-if="sheetOpen" class="layer">
      <view class="scrim" @click="closeSheet" />
      <view class="sheet">
        <view class="sheet-head">
          <view class="sheet-cancel" @click="closeSheet">取消</view>
          <text class="sheet-title">{{ sheetTitle }}</text>
          <view class="sheet-all" @click="toggleSelectAllSheet">{{ allSheetSelected ? "取消全选" : "全选" }}</view>
        </view>
        <text class="sheet-desc">{{ sheetDesc }}</text>
        <view
          v-if="form.target_type === 1"
          class="roster-row"
          :class="{ 'is-busy': rosterLoading }"
          @click="pickWholeClass"
        >
          <text>{{ rosterLoading ? "获取到场名单…" : "全班（当日到场）" }}</text>
          <text class="roster-hint">按当日到场填入，可再增减</text>
        </view>
        <view class="search">
          <text class="search-icon icon">&#xf0d1;</text>
          <input v-model="sheetQuery" class="search-input" type="text" placeholder="搜索幼儿姓名" placeholder-style="color: #94A3B8" />
        </view>
        <scroll-view scroll-y class="sheet-list">
          <view
            v-for="student in sheetCandidates"
            :key="student.student_id"
            class="srow"
            :class="{ 'is-on': sheetPicked.has(student.student_id) || sheetBatchExisting.has(student.student_id) }"
            @click="toggleSheetStudent(student.student_id)"
          >
            <view class="savatar">{{ String(student.name || "?").slice(0, 1) }}</view>
            <text class="sname">{{ student.name }}</text>
            <view class="scheck">
              <text class="icon">&#xeb7b;</text>
            </view>
          </view>
          <view v-if="!sheetCandidates.length" class="sheet-empty">没有找到匹配的幼儿</view>
        </scroll-view>
        <view class="sheet-foot">
          <view class="publish" @click="confirmSheet">确定（已选 {{ sheetSelectedCount }} 人）</view>
        </view>
      </view>
    </view>

    <template #bottom>
      <view class="dock">
        <view v-if="submitError" class="err-strip">
          <text class="err-icon">!</text>
          <text class="err-text">{{ isEdit ? "保存失败" : "发布失败" }}：{{ submitError }}</text>
          <view class="err-retry" @click="submit">重试</view>
        </view>
        <view class="dock-meta">
          <text class="scope">{{ scopeText }}</text>
          <view v-if="!isEdit && form.target_type === 1" class="switch-row">
            <text>同步到班级动态</text>
            <view class="sw" :class="{ 'is-on': form.sync_to_feed }" @click="form.sync_to_feed = !form.sync_to_feed">
              <view class="sw-knob" />
            </view>
          </view>
        </view>
        <text v-if="blockReasons.length" class="dock-reason">{{ blockReasons.join(" · ") }}</text>
        <view class="publish" :class="{ 'is-disabled': !canSubmit }" @click="submit">{{ publishLabel }}</view>
      </view>
    </template>
  </PageShell>
</template>

<style scoped lang="scss">
.icon {
  font-family: remixicon;
  font-style: normal;
  line-height: 1;
}

.page {
  padding-bottom: $spacing-xl;
}

/* ---------- 日期 ---------- */
.day {
  display: grid;
  grid-template-columns: 88rpx minmax(0, 1fr) 88rpx;
  align-items: center;
  gap: $spacing-xs;
  padding: 2rpx 0 $spacing-md;
}

.day-arrow {
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-xl;

  &:active {
    background: $color-brand-primary-bg;
  }
}

.day-arrow.is-disabled {
  color: $color-text-hint;
  opacity: 0.45;

  &:active {
    background: transparent;
  }
}

.day-main {
  min-width: 0;
  text-align: center;
}

.day-date {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.day-sub {
  display: block;
  margin-top: 2rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

/* ---------- 记录对象分段 ---------- */
.mode {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: $spacing-xs;
  padding: $spacing-xs;
  margin-bottom: $spacing-md;
  background: $color-bg-card;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
}

.mode-btn {
  height: 80rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  color: $color-text-muted;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
}

.mode-btn.is-on {
  background: $color-bg-page;
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.mode-btn.is-disabled {
  opacity: 0.55;
}

/* ---------- 卡片 ---------- */
.card {
  margin-bottom: $spacing-md;
  padding: $spacing-lg;
  background: $color-bg-card;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  box-shadow: $shadow-md;
}

.card-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-xs;
  margin-bottom: $spacing-md;
}

.card-title {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.card-note {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.req-mark {
  color: $color-semantic-danger;
  font-weight: $typography-font-weight-semibold;
}

/* ---------- 快速记录 ---------- */
.obs-row {
  display: grid;
  grid-template-columns: 80rpx minmax(0, 1fr);
  gap: $spacing-sm;
  padding: $spacing-md 0;
  border-top: 1rpx solid $color-border-subtle;

  &:first-child {
    border-top: 0;
    padding-top: 2rpx;
  }

  &:last-child {
    padding-bottom: 2rpx;
  }
}

.obs-label {
  padding-top: $spacing-md;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.obs-main {
  min-width: 0;
}

.meal {
  display: grid;
  grid-template-columns: 72rpx minmax(0, 1fr);
  align-items: center;
  gap: $spacing-xs;

  & + .meal {
    margin-top: $spacing-xs;
  }
}

.meal-name {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
}

.rates {
  display: flex;
  gap: $spacing-xs;
}

.rate {
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 38rpx;
  line-height: 1;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  border: 1rpx solid transparent;

  &:active {
    transform: scale(0.92);
  }
}

.rate.is-on {
  background: $color-brand-primary-bg;
  border-color: $color-brand-primary;
  box-shadow: $shadow-sm;
}

.sub-label {
  display: block;
  margin: 2rpx 0 $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.obs-row .sub-label + .sub-label {
  margin-top: $spacing-sm;
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
}

.chip {
  min-height: 72rpx;
  padding: 0 $spacing-sm;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  border: 1rpx solid transparent;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

.chip.is-on {
  background: $color-brand-primary-bg;
  border-color: $color-brand-primary;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

.tags {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
}

.tag {
  min-height: 68rpx;
  padding: 0 $spacing-sm;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  border: 1rpx solid transparent;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.tag.is-on {
  background: $color-brand-primary-bg;
  border-color: $color-brand-primary;
  color: $color-brand-primary-dark;
}

/* ---------- 指定幼儿 ---------- */
.picked {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: $spacing-xs;
  padding: $spacing-xs 0 2rpx;
}

.avatars {
  display: flex;
  flex: none;
}

.ava {
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  border: 4rpx solid $color-bg-card;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;

  & + .ava {
    margin-left: -20rpx;
  }
}

.ava-more {
  background: $color-bg-page;
  color: $color-text-muted;
}

.picked-text {
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  overflow-wrap: anywhere;
}

.link-btn {
  min-height: 88rpx;
  padding: 0 2rpx;
  display: flex;
  align-items: center;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.pick-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-xs;
  padding: $spacing-lg $spacing-md $spacing-lg;
  border: 2rpx dashed $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  text-align: center;
}

.pick-empty-art {
  position: relative;
  width: 120rpx;
  height: 120rpx;
  margin-bottom: $spacing-xs;
}

.pick-orb {
  position: absolute;
  border-radius: $border-radius-full;
}

.orb-a {
  top: 18rpx;
  left: 14rpx;
  width: 64rpx;
  height: 64rpx;
  background: $color-semantic-warning-bg;
  border: 2rpx solid $color-brand-primary-light;
}

.orb-b {
  top: 40rpx;
  left: 48rpx;
  width: 56rpx;
  height: 56rpx;
  background: $color-semantic-success-bg;
  border: 2rpx solid $color-semantic-success;
}

.pick-arc {
  position: absolute;
  right: 8rpx;
  bottom: 4rpx;
  left: 8rpx;
  height: 28rpx;
  border-bottom: 2rpx solid $color-brand-primary-light;
  border-radius: $border-radius-full;
}

.pick-empty-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.pick-empty-hint {
  margin-bottom: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.btn-soft {
  min-height: 88rpx;
  padding: 0 $spacing-xl;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  border: 1rpx solid $color-brand-primary;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.tags-wrap {
  margin-top: $spacing-md;
  padding-top: $spacing-sm;
  border-top: 1rpx solid $color-border-subtle;
}

/* ---------- 照片 ---------- */
.media-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: $spacing-xs;
  min-height: 72rpx;
  margin-bottom: $spacing-xs;
}

.media-tip {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.mini-btn {
  flex: none;
  min-height: 64rpx;
  padding: 0 $spacing-sm;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.mini-btn.primary {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.batch-tools {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
}

.media-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-xs;
}

.tile {
  position: relative;
  height: 208rpx;
  overflow: hidden;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  border: 1rpx solid $color-border-hairline;
}

.tile.is-picked {
  border-color: $color-brand-primary;
}

.tile-img {
  width: 100%;
  height: 100%;
  display: block;
}

.tile-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  background: $color-text-primary;
  color: $color-text-on-brand;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.tile-mask.is-failed {
  background: $color-semantic-danger;
}

.tile-spinner {
  width: 40rpx;
  height: 40rpx;
  border: 4rpx solid $color-text-on-brand;
  border-top-color: $color-brand-primary;
  border-radius: $border-radius-full;
  animation: spin 900ms linear infinite;
}

.tile-retry {
  color: $color-text-on-brand;
  text-decoration: underline;
}

.tile-badge {
  position: absolute;
  left: $spacing-xs;
  bottom: $spacing-xs;
  max-width: calc(100% - #{$spacing-md});
  min-height: 40rpx;
  padding: 0 $spacing-xs;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.tile-remove {
  position: absolute;
  top: 4rpx;
  right: 4rpx;
  width: 56rpx;
  height: 56rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-text-primary;
  color: $color-text-on-brand;
  font-size: $typography-font-size-xs;
}

.tile-check {
  position: absolute;
  top: $spacing-xs;
  right: $spacing-xs;
  width: 40rpx;
  height: 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  border: 2rpx solid $color-bg-card;
  background: $color-text-body;
  color: transparent;
  font-size: $typography-font-size-xs;
}

.tile-check.is-on {
  background: $color-brand-primary-dark;
  border-color: $color-brand-primary;
  color: $color-text-on-brand;
}

.tile-add {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2rpx;
  border-style: dashed;
  background: $color-bg-page;
  color: $color-text-muted;

  &:active {
    background: $color-brand-primary-bg;
  }
}

.tile-add-icon {
  font-size: $typography-font-size-3xl;
}

.tile-add-text {
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.desc-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
  margin-top: $spacing-sm;
}

.desc-row {
  display: grid;
  grid-template-columns: 56rpx minmax(0, 1fr);
  align-items: center;
  gap: $spacing-xs;
}

.desc-thumb {
  width: 56rpx;
  height: 56rpx;
  overflow: hidden;
  border-radius: $border-radius-md;
  background: $color-bg-page;
}

.desc-thumb-img {
  width: 100%;
  height: 100%;
}

.desc-input {
  width: 100%;
  min-width: 0;
  min-height: 72rpx;
  padding: 0 $spacing-sm;
  border-radius: $border-radius-md;
  background: $color-bg-page;
  border: 1rpx solid transparent;
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
}

.fineprint {
  display: flex;
  align-items: flex-start;
  gap: $spacing-xs;
  margin-top: $spacing-sm;
  color: $color-text-hint;
  font-size: $typography-font-size-xs;
  line-height: 1.55;
}

.fineprint-icon {
  flex: none;
  margin-top: 2rpx;
  font-size: $typography-font-size-sm;
}

.fineprint-dot {
  flex: none;
  width: 8rpx;
  height: 8rpx;
  margin-top: 10rpx;
  border-radius: $border-radius-full;
  background: $color-text-hint;
}

/* ---------- 正文 ---------- */
.textarea {
  display: block;
  width: 100%;
  height: 216rpx;
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  border: 1rpx solid transparent;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.7;
}

/* ---------- 徽章 ---------- */
.badges {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
}

.badge-chip {
  min-height: 76rpx;
  padding: 0 $spacing-sm;
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  border: 1rpx solid transparent;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.badge-chip.is-on {
  background: $color-brand-primary-bg;
  border-color: $color-brand-primary;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

.badge-icon {
  width: 40rpx;
  height: 40rpx;
  border-radius: $border-radius-sm;
}

.grant {
  margin-top: $spacing-md;
  padding-top: $spacing-sm;
  border-top: 1rpx solid $color-border-subtle;
}

.grant-hint {
  display: block;
  color: $color-text-hint;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.check-row {
  display: flex;
  align-items: flex-start;
  gap: $spacing-xs;
  min-height: 88rpx;
  margin-top: $spacing-xs;
  padding-top: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.check-row .box {
  flex: none;
  width: 36rpx;
  height: 36rpx;
  margin-top: 2rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-sm;
  border: 2rpx solid $color-border-hairline;
  color: transparent;
  font-size: $typography-font-size-xs;
}

.check-row .box.is-on {
  background: $color-brand-primary-dark;
  border-color: $color-brand-primary;
  color: $color-text-on-brand;
}

/* ---------- 幼儿弹层 ---------- */
.layer {
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

.scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: $color-text-primary;
  opacity: 0.45;
}

.sheet {
  position: relative;
  width: 100%;
  max-height: 82%;
  display: flex;
  flex-direction: column;
  padding: $spacing-xs $component-page-padding-x calc(#{$spacing-md} + env(safe-area-inset-bottom));
  background: $color-bg-card;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  box-shadow: $shadow-lg;
}

.sheet-head {
  display: grid;
  grid-template-columns: 128rpx minmax(0, 1fr) 128rpx;
  align-items: center;
  min-height: 96rpx;
}

.sheet-cancel,
.sheet-all {
  min-height: 88rpx;
  display: flex;
  align-items: center;
  font-size: $typography-font-size-md;
}

.sheet-cancel {
  color: $color-text-muted;
}

.sheet-all {
  justify-content: flex-end;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

.sheet-title {
  text-align: center;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
}

.sheet-desc {
  padding: 0 2rpx $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.roster-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-xs;
  margin-bottom: $spacing-xs;
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius-lg;
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.roster-row.is-busy {
  opacity: 0.6;
}

.roster-hint {
  color: $color-semantic-success;
  font-size: $typography-font-size-xs;
}

.search {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  height: 80rpx;
  padding: 0 $spacing-sm;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  border: 1rpx solid $color-border-hairline;
  color: $color-text-hint;
}

.search-icon {
  flex: none;
  font-size: $typography-font-size-md;
}

.search-input {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
}

.sheet-list {
  flex: 1;
  min-height: 0;
  height: 520rpx;
  margin-top: $spacing-xs;
}

.srow {
  display: grid;
  grid-template-columns: 72rpx minmax(0, 1fr) 44rpx;
  align-items: center;
  gap: $spacing-sm;
  min-height: 104rpx;
  padding: $spacing-xs 2rpx;
  border-radius: $border-radius-lg;

  & + .srow {
    border-top: 1rpx solid $color-border-subtle;
  }
}

.savatar {
  width: 72rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

.sname {
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
}

.scheck {
  width: 44rpx;
  height: 44rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  border: 2rpx solid $color-border-hairline;
  color: transparent;
  font-size: $typography-font-size-xs;
}

.srow.is-on .savatar {
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
}

.srow.is-on .scheck {
  background: $color-brand-primary-dark;
  border-color: $color-brand-primary;
  color: $color-text-on-brand;
}

.sheet-empty {
  padding: $spacing-xl $spacing-xs;
  text-align: center;
  color: $color-text-hint;
  font-size: $typography-font-size-base;
}

.sheet-foot {
  padding-top: $spacing-sm;
}

/* ---------- 底栏 ---------- */
.dock {
  padding: $spacing-xs $component-page-padding-x calc(#{$spacing-sm} + env(safe-area-inset-bottom));
  background: $color-bg-card;
  border-top: 1rpx solid $color-border-hairline;
  box-shadow: $shadow-lg;
}

.err-strip {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  margin-bottom: $spacing-xs;
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.err-icon {
  flex: none;
  width: 28rpx;
  height: 28rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  border: 2rpx solid $color-semantic-danger;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-bold;
  line-height: 1;
}

.err-text {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}

.err-retry {
  flex: none;
  min-height: 60rpx;
  padding: 0 $spacing-xs;
  display: flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  border: 1rpx solid $color-semantic-danger;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

.dock-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-xs;
  min-height: 56rpx;
  margin-bottom: $spacing-xs;
}

.scope {
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.switch-row {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.sw {
  position: relative;
  width: 80rpx;
  height: 48rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
}

.sw.is-on {
  background: $color-brand-primary;
}

.sw-knob {
  position: absolute;
  top: 4rpx;
  left: 4rpx;
  width: 40rpx;
  height: 40rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sw.is-on .sw-knob {
  left: 36rpx;
}

.dock-reason {
  margin-bottom: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.45;
}

.publish {
  width: 100%;
  min-height: 96rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.985);
  }
}

.publish.is-disabled {
  background: $color-bg-page;
  border: 1rpx solid $color-border-hairline;
  color: $color-text-hint;
  box-shadow: none;

  &:active {
    transform: none;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
