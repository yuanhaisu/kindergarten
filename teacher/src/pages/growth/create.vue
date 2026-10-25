<script setup>
import { ref, computed } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { createTeacherGrowthLog, fetchTeacherGrowthLog, updateTeacherGrowthLog, uploadGrowthMedia, fetchPresentStudents } from "@/api/teacher-growth"
import { fetchTeacherClassStudents } from "@/api/teacher-class"
import { fetchDict } from "@/api/dict"
import { fetchTeacherAchievements, grantAchievement } from "@/api/teacher-achievements"

function todayStr() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`
}

function formatDate(ymd) {
  const d = new Date(ymd.replace(/-/g, "/"))
  const w = ["日", "一", "二", "三", "四", "五", "六"]
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日 星期${w[d.getDay()]}`
}

function addDays(ymd, n) {
  const d = new Date(ymd.replace(/-/g, "/"))
  d.setDate(d.getDate() + n)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const dd = String(d.getDate()).padStart(2, "0")
  return `${y}-${m}-${dd}`
}

const submitting = ref(false)
const students = ref([])
// 学生选择器：30 人班的检索入口（本地过滤，无需后端接口）
const studentQuery = ref("")
const filteredStudents = computed(() => {
  const q = studentQuery.value.trim().toLowerCase()
  if (!q) return students.value
  return students.value.filter(s => (s.name || "").toLowerCase().includes(q))
})
const form = ref({
  date: todayStr(),
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

const dateDisplay = computed(() => formatDate(form.value.date))

// 编辑态：从详情页带 id 进入时更新原记录，而非新建。
// 日期在编辑态只读——记录的日期是「这一天发生了什么」的锚点，改动会让家长端时间线错位。
const editId = ref(null)
const isEdit = computed(() => editId.value != null)

function prevDay() { if (isEdit.value) return; form.value.date = addDays(form.value.date, -1) }
function nextDay() {
  if (isEdit.value) return
  const n = addDays(form.value.date, 1)
  if (n <= todayStr()) form.value.date = n
}

function toggleTag(tag) {
  const i = form.value.performance_tags.indexOf(tag)
  if (i >= 0) form.value.performance_tags.splice(i, 1)
  else form.value.performance_tags.push(tag)
}

function toggleStudent(sid) {
  const i = form.value.student_ids.indexOf(sid)
  if (i >= 0) form.value.student_ids.splice(i, 1)
  else form.value.student_ids.push(sid)
}

// ---- 授予徽章（可选）----
// 记录先发布，再授予：见证需要 log_id，所以授予必须发生在发布成功之后。
const achievements = ref([])
const grant = ref({ achievement_id: null, student_ids: [], as_witness: true })

function toggleGrantStudent(sid) {
  const i = grant.value.student_ids.indexOf(sid)
  if (i >= 0) grant.value.student_ids.splice(i, 1)
  else grant.value.student_ids.push(sid)
}

function selectAchievement(id) {
  grant.value.achievement_id = grant.value.achievement_id === id ? null : id
}

async function loadAchievements() {
  try {
    const res = await fetchTeacherAchievements()
    achievements.value = res.data?.list || []
  } catch (e) {
    // 徽章列表失败不阻塞发布——授予是可选动作
  }
}

async function grantAfterPublish(logId) {
  const g = grant.value
  if (!g.achievement_id) return
  const target = g.student_ids.length ? g.student_ids : form.value.student_ids
  if (!target.length) {
    uni.showToast({ title: "请选择要授予的学生", icon: "none" })
    return
  }
  try {
    await grantAchievement(g.achievement_id, {
      student_ids: target,
      log_id: g.as_witness ? logId : undefined,
    })
    uni.showToast({ title: "已授予徽章", icon: "success" })
  } catch (e) {
    // 授予失败不影响已发布的记录——单独提示
    uni.showToast({ title: e?.message || "授予徽章失败", icon: "none" })
  }
}

function switchTargetType() {
  form.value.target_type = form.value.target_type === 2 ? 1 : 2
  if (form.value.target_type === 1) {
    form.value.student_ids = []
    form.value.sync_to_feed = true
  } else {
    form.value.sync_to_feed = false
    loadStudents()
  }
}

async function loadStudents() {
  try {
    const res = await fetchTeacherClassStudents()
    students.value = res.data?.list || []
  } catch (e) {
    // 学生列表加载失败不阻塞页面
  }
}

onLoad((options) => {
  const id = options?.id
  if (id) {
    editId.value = id
    loadExistingLog(id)
  }
  loadStudents()
  loadPerformanceTags()
  loadAchievements()
})

async function loadExistingLog(id) {
  try {
    const res = await fetchTeacherGrowthLog(id)
    const rec = res.data || {}
    const f = form.value
    f.content = rec.content || ""
    f.breakfast_rating = rec.breakfast_rating || 0
    f.lunch_rating = rec.lunch_rating || 0
    f.dinner_rating = rec.dinner_rating || 0
    f.nap_duration = rec.nap_duration || 0
    f.sleep_quality = rec.sleep_quality || 0
    f.morning_mood = rec.morning_mood || 0
    f.afternoon_mood = rec.afternoon_mood || 0
    f.performance_tags = Array.isArray(rec.performance_tags) ? rec.performance_tags : []
    f.target_type = rec.target_type === 2 ? 2 : 1
    f.sync_to_feed = rec.sync_to_feed !== false
    // 详情响应日期为 "2006-01-02" 格式，可直接回填
    if (rec.date) f.date = String(rec.date).slice(0, 10)
    f.student_ids = (rec.related_students || []).map(s => s.id)
    // 既有照片回显（编辑态）：保留 media.id 供服务端识别「保留项」，ref_desc 可继续编辑
    f.media = (rec.media || []).map(m => ({
      id: m.id || 0,
      file_id: m.file_id || 0,
      file_url: m.file_url || "",
      thumb_url: m.thumb_url || m.file_url || "",
      file_type: m.file_type || "image",
      ref_desc: m.ref_desc || "",
      status: "done",
      // 点名（D8）：已确认名单；AI 建议仅供展示与预勾——未在编辑态提交的建议保持原状
      student_ids: Array.isArray(m.student_ids) ? [...m.student_ids] : [],
      suggestions: Array.isArray(m.student_suggestions) ? m.student_suggestions : [],
      _named: false
    }))
  } catch (e) {
    uni.showToast({ title: e?.message || "加载记录失败", icon: "none" })
  }
}

function addMedia() {
  uni.chooseImage({
    count: 20 - form.value.media.length,
    success(res) { res.tempFilePaths.forEach(p => uploadOne(p)) }
  })
}

// 上传超时护栏：网络挂起时不能永远停在「上传中」——超时即判失败、可重试（specs/growth-media-pipeline「上传失败可重试」）。
const UPLOAD_TIMEOUT_MS = 30000

function withTimeout(promise, ms) {
  return Promise.race([
    promise,
    new Promise((_, reject) => setTimeout(() => reject(new Error("上传超时")), ms))
  ])
}

function fillUploaded(item, d) {
  item.file_id = d.id
  item.file_url = d.file_url
  item.thumb_url = d.thumb_url || d.file_url
  item.file_type = d.file_type || "image"
  item.status = "done"
}

// 每张照片独立上传：成功项保存 file_id（提交时的引用标识）与展示 URL；
// 失败项保留本地路径并标记 failed，供逐张重试。
async function uploadOne(tempPath) {
  const item = { file_id: 0, file_url: "", thumb_url: "", file_type: "image", ref_desc: "", status: "uploading", _temp: tempPath, student_ids: [], suggestions: [], _named: true }
  form.value.media.push(item)
  try {
    const res = await withTimeout(uploadGrowthMedia(tempPath), UPLOAD_TIMEOUT_MS)
    fillUploaded(item, res.data || {})
  } catch (e) {
    item.status = "failed"
  }
}

function retryMedia(i) {
  const item = form.value.media[i]
  if (!item || item.status !== "failed" || !item._temp) return
  item.status = "uploading"
  withTimeout(uploadGrowthMedia(item._temp), UPLOAD_TIMEOUT_MS)
    .then(res => { fillUploaded(item, res.data || {}) })
    .catch(() => { item.status = "failed" })
}

function mediaDisplayUrl(item) {
  return item.thumb_url || item.file_url || item._temp || ""
}

function removeMedia(i) { form.value.media.splice(i, 1) }

// ── 照片点名（D8：批量追加 + 单张微调）─────────────────────────────
// 批量 = 把勾选的学生「追加」到所选照片（不叠加，TC-NEW-PTG-0003/0004）；
// 单张 = 编辑这张照片的完整名单（加/减皆可，TC-NEW-PTG-0005/0006）。

const batchMode = ref(false)
const batchSelected = ref(new Set())
const namingOpen = ref(false)
const namingTargets = ref([]) // 待点名的照片下标
const namingStudents = ref(new Set())
// 显式区分「批量」与「单张」入口：之前用 targets.length===1 推断，
// 导致批量只选 1 张时悄悄变成「替换」语义（已有名单被清掉）——与追加承诺不符
const namingIsBatch = ref(false)

// 批量模式下这批照片**已有**的点名（并集）：面板要把它显性展示出来，
// 否则「已点 2 人 + 新增 1 人 = 点名 3」对老师是惊吓（2026-10-15 手动测试反馈）
const batchExisting = computed(() => {
  if (!namingIsBatch.value) return new Set()
  const s = new Set()
  namingTargets.value.forEach(i => {
    const m = form.value.media[i]
    if (m) (m.student_ids || []).forEach(id => s.add(id))
  })
  return s
})

// 确定后的每张人数 = 新增 ∪ 已有。两者可能重叠（「全班」会一次选中含已有的整份名单），
// 简单相加会把重叠的孩子算两次（曾实测：2 新增 + 2 已有显示「共 4 人」，实际 2 人）。
function namingTotalCount() {
  if (!namingIsBatch.value) return namingStudents.value.size
  const all = new Set(namingStudents.value)
  batchExisting.value.forEach(id => all.add(id))
  return all.size
}

// 可点名学生 = 记录范围内的学生（白名单与后端一致：个体记录仅目标学生）
const namingCandidates = computed(() => {
  if (form.value.target_type === 2) {
    return students.value.filter(s => form.value.student_ids.includes(s.student_id))
  }
  return students.value
})

function namedCount(m) {
  return Array.isArray(m.student_ids) ? m.student_ids.length : 0
}

function mediaSuggestionCount(m) {
  return Array.isArray(m.suggestions) ? m.suggestions.length : 0
}

// 单张：建议的置信分级（high 预勾、low 仅提示）——阈值由服务端判定，前端不硬编码
function suggestionOf(sid) {
  if (namingTargets.value.length !== 1) return null
  const m = form.value.media[namingTargets.value[0]]
  if (!m || !Array.isArray(m.suggestions)) return null
  return m.suggestions.find(s => s.student_id === sid) || null
}

function startBatch() {
  batchMode.value = true
  batchSelected.value = new Set()
}

function cancelBatch() {
  batchMode.value = false
  batchSelected.value = new Set()
}

function toggleBatchSelect(i) {
  const m = form.value.media[i]
  if (!m || m.status !== "done") return
  const next = new Set(batchSelected.value)
  if (next.has(i)) next.delete(i)
  else next.add(i)
  batchSelected.value = next
}

// 全选可点名的照片（同活动连续命名时省掉逐张点击，D10）
const doneMediaIndexes = computed(() =>
  form.value.media.map((m, i) => (m.status === "done" ? i : -1)).filter((i) => i >= 0),
)
const allPhotosSelected = computed(
  () => doneMediaIndexes.value.length > 0 && batchSelected.value.size === doneMediaIndexes.value.length,
)

function toggleSelectAll() {
  batchSelected.value = allPhotosSelected.value ? new Set() : new Set(doneMediaIndexes.value)
}

function openSingleNaming(i) {
  if (batchMode.value) { toggleBatchSelect(i); return }
  const m = form.value.media[i]
  if (m.status !== "done") return
  namingTargets.value = [i]
  namingIsBatch.value = false
  // 预勾：已确认名单 + 高置信建议（D6：预勾只是默认选中，可取消）
  const pre = new Set(m.student_ids || [])
  ;(m.suggestions || []).forEach(s => { if (s.tier === "high") pre.add(s.student_id) })
  namingStudents.value = pre
  namingOpen.value = true
}

function openBatchNaming() {
  const targets = [...batchSelected.value].sort((a, b) => a - b)
  if (!targets.length) {
    uni.showToast({ title: "请先选择照片", icon: "none" })
    return
  }
  namingTargets.value = targets
  namingIsBatch.value = true
  namingStudents.value = new Set() // 批量追加语义：从空开始，不覆盖各自已有点名
  namingOpen.value = true
}

// 已有点名不可在批量面板里取消（取消也不生效——合并语义是并集），点击给个说明
function toggleNamingStudent(sid) {
  if (batchExisting.value.has(sid)) {
    uni.showToast({ title: "该孩子已在照片上（批量只做追加）", icon: "none" })
    return
  }
  const next = new Set(namingStudents.value)
  if (next.has(sid)) next.delete(sid)
  else next.add(sid)
  namingStudents.value = next
}

function closeNaming() {
  namingOpen.value = false
}

// 全班 = 记录当日的到场学生作为默认名单（D11）：填入后仍可逐人增减——
// 「全班」是默认值而非锁定（拍合影时总有个别孩子不在画面）。
const rosterLoading = ref(false)

async function pickWholeClass() {
  if (rosterLoading.value) return
  rosterLoading.value = true
  try {
    const res = await fetchPresentStudents(String(form.value.date || "").slice(0, 10))
    const list = res.data?.list || []
    const allowed = new Set(namingCandidates.value.map(s => s.student_id))
    const present = list.filter(id => allowed.has(id))
    let picked = present
    // 批量模式：已在该批照片上的孩子不进入「新增」勾选集（与 toggleNamingStudent 的追加语义一致，
    // 也避免点它们时弹出「已在照片上」却不能取消的困惑）
    if (namingIsBatch.value) picked = present.filter(id => !batchExisting.value.has(id))
    if (!picked.length) {
      // 批量模式下还有另一种空：到场名单非空，但已全部在所选照片上（无需追加）
      const title = namingIsBatch.value && present.length ? "到场孩子已全部在这批照片上" : "当日到场名单为空"
      uni.showToast({ title, icon: "none" })
      return
    }
    namingStudents.value = new Set(picked)
    // 批量是追加语义：报「新增 N 人」并带上到场总数，避免把筛选后的新增数误读成到场数
    const title = namingIsBatch.value
      ? `新增 ${picked.length} 人（当日到场共 ${present.length} 人）`
      : `已选当日到场 ${picked.length} 人，可增减`
    uni.showToast({ title, icon: "none" })
  } catch (e) {
    uni.showToast({ title: e?.message || "获取到场名单失败", icon: "none" })
  } finally {
    rosterLoading.value = false
  }
}

function confirmNaming() {
  const picked = [...namingStudents.value]
  if (!namingIsBatch.value) {
    // 单张微调 = 编辑这张照片的完整名单（加/减皆可）
    const m = form.value.media[namingTargets.value[0]]
    if (m) { m.student_ids = picked; m._named = true }
  } else {
    // 批量 = 追加：与各照片已有点名取并集，重复应用不叠加（TC-NEW-PTG-0003/0004）
    if (!picked.length) {
      // 无新增 = 无操作。不能把照片标记为「已动过」——提交时会带上空名单，连带清掉它的 AI 建议
      // （「全班」把在场孩子一次选完时最易走到这里：全是「已有」，新增集为空）
      cancelBatch()
      namingOpen.value = false
      uni.showToast({ title: "未选择新增孩子，未做改动", icon: "none" })
      return
    }
    namingTargets.value.forEach(i => {
      const m = form.value.media[i]
      if (!m) return
      const merged = new Set(m.student_ids || [])
      picked.forEach(sid => merged.add(sid))
      m.student_ids = [...merged]
      m._named = true
    })
    cancelBatch()
  }
  namingOpen.value = false
}

const mediaPending = computed(() => form.value.media.filter(m => m.status !== "done").length)

// 提交载荷：只送引用（id/file_id/ref_desc），URL 等物理属性由服务端反查（基线 L2）。
// student_ids 仅在老师动过这张照片的点名时携带（_named）——未动过的照片不提交该字段，
// 服务端保持其既有点名与待处理建议不变（D8：提交即最终名单，缺省即不触碰）。
function mediaPayload() {
  return form.value.media.map(m => {
    const item = { id: m.id || 0, file_id: m.file_id, ref_desc: m.ref_desc || "" }
    if (m._named) item.student_ids = m.student_ids || []
    return item
  })
}

async function publish() {
  if (!form.value.content.trim()) {
    uni.showToast({ title: "请输入记录内容", icon: "none" })
    return
  }
  if (form.value.target_type === 2 && form.value.student_ids.length === 0) {
    uni.showToast({ title: "请选择至少一位学生", icon: "none" })
    return
  }
  // 存在未上传完成/失败的素材时阻断提交（宁可不出，不静默丢图）
  if (mediaPending.value > 0) {
    const failed = form.value.media.some(m => m.status === "failed")
    uni.showToast({ title: failed ? "有照片上传失败，请重试或移除" : "照片上传中，请稍候", icon: "none" })
    return
  }
  submitting.value = true
  try {
    const payload = { ...form.value, media: mediaPayload() }
    if (isEdit.value) {
      await updateTeacherGrowthLog(editId.value, payload)
      uni.showToast({ title: "已保存", icon: "success" })
    } else {
      const created = await createTeacherGrowthLog(payload)
      // 授予是可选的追加动作，且需要刚发布的记录作见证——放在发布成功之后
      await grantAfterPublish(created?.data?.id)
      uni.showToast({ title: "已发布", icon: "success" })
    }
    setTimeout(() => uni.navigateBack(), 600)
  } catch (e) {
    uni.showToast({ title: e?.message || (isEdit.value ? "保存失败" : "发布失败"), icon: "none" })
  } finally {
    submitting.value = false
  }
}

const sleepOptions = [1, 2, 3]
const qualityOptions = [
  { value: 3, label: "很好" },
  { value: 2, label: "一般" },
  { value: 1, label: "不好" }
]

const performanceOptions = ref([])

async function loadPerformanceTags() {
  try {
    const res = await fetchDict("performance_tag")
    performanceOptions.value = (res.data?.items || []).map(i => i.item_label)
  } catch (e) {
    // fallback to defaults
    performanceOptions.value = [
      "乐于助人", "积极发言", "遵守纪律", "动手能力强",
      "专注力强", "有礼貌", "爱干净", "团结友善", "勇于尝试"
    ]
  }
}
</script>

<template>
  <view class="page">
    <scroll-view class="body" scroll-y>
      <!-- 日期选择栏（编辑态只读：记录日期不可改） -->
      <view class="date-bar">
        <view class="date-arrow" :class="{ disabled: isEdit }" @click="prevDay"><text>◀</text></view>
        <text class="date-text">{{ dateDisplay }}</text>
        <view class="date-arrow" :class="{ disabled: isEdit || form.date >= todayStr() }" @click="nextDay">
          <text>▶</text>
        </view>
      </view>

      <!-- 快速记录模板 — 仅班级记录 -->
      <view v-if="form.target_type === 1" class="section">
        <text class="section-title">快速记录</text>

        <view class="template-row">
          <!-- 饮食 -->
          <view class="template-card">
            <view class="card-header">
              <view class="card-icon" style="background:rgba(220,252,231,1)">🍽️</view>
              <text class="card-title">饮食</text>
            </view>
            <view class="card-body">
              <!-- 早餐 -->
              <view class="meal-row">
                <text class="meal-label">早餐</text>
                <view class="emoji-row">
                  <view class="emoji-btn" :style="{ background: form.breakfast_rating === 3 ? '#dcfce7' : '#f1f5f9' }" @click="form.breakfast_rating = form.breakfast_rating === 3 ? 0 : 3">😋</view>
                  <view class="emoji-btn" :style="{ background: form.breakfast_rating === 2 ? '#dcfce7' : '#f1f5f9' }" @click="form.breakfast_rating = form.breakfast_rating === 2 ? 0 : 2">😐</view>
                  <view class="emoji-btn" :style="{ background: form.breakfast_rating === 1 ? '#dcfce7' : '#f1f5f9' }" @click="form.breakfast_rating = form.breakfast_rating === 1 ? 0 : 1">😞</view>
                </view>
              </view>
              <!-- 午餐 -->
              <view class="meal-row">
                <text class="meal-label">午餐</text>
                <view class="emoji-row">
                  <view class="emoji-btn" :style="{ background: form.lunch_rating === 3 ? '#dcfce7' : '#f1f5f9' }" @click="form.lunch_rating = form.lunch_rating === 3 ? 0 : 3">😋</view>
                  <view class="emoji-btn" :style="{ background: form.lunch_rating === 2 ? '#dcfce7' : '#f1f5f9' }" @click="form.lunch_rating = form.lunch_rating === 2 ? 0 : 2">😐</view>
                  <view class="emoji-btn" :style="{ background: form.lunch_rating === 1 ? '#dcfce7' : '#f1f5f9' }" @click="form.lunch_rating = form.lunch_rating === 1 ? 0 : 1">😞</view>
                </view>
              </view>
              <!-- 晚餐 -->
              <view class="meal-row">
                <text class="meal-label">晚餐</text>
                <view class="emoji-row">
                  <view class="emoji-btn" :style="{ background: form.dinner_rating === 3 ? '#dcfce7' : '#f1f5f9' }" @click="form.dinner_rating = form.dinner_rating === 3 ? 0 : 3">😋</view>
                  <view class="emoji-btn" :style="{ background: form.dinner_rating === 2 ? '#dcfce7' : '#f1f5f9' }" @click="form.dinner_rating = form.dinner_rating === 2 ? 0 : 2">😐</view>
                  <view class="emoji-btn" :style="{ background: form.dinner_rating === 1 ? '#dcfce7' : '#f1f5f9' }" @click="form.dinner_rating = form.dinner_rating === 1 ? 0 : 1">😞</view>
                </view>
              </view>
            </view>
          </view>

          <!-- 睡眠 -->
          <view class="template-card">
            <view class="card-header">
              <view class="card-icon" style="background:rgba(243,232,255,1)">💤</view>
              <text class="card-title">睡眠</text>
            </view>
            <view class="card-body">
              <text class="field-label">午睡时长</text>
              <view class="chip-row">
                <view
                  v-for="h in sleepOptions" :key="h"
                  class="chip"
                  :class="{ active: form.nap_duration === h }"
                  @click="form.nap_duration = form.nap_duration === h ? 0 : h"
                >{{ h }}小时</view>
              </view>
              <text class="field-label" style="margin-top:12rpx">睡眠质量</text>
              <view class="chip-row">
                <view
                  v-for="q in qualityOptions" :key="q.value"
                  class="chip"
                  :class="{ active: form.sleep_quality === q.value }"
                  @click="form.sleep_quality = form.sleep_quality === q.value ? 0 : q.value"
                >{{ q.label }}</view>
              </view>
            </view>
          </view>
        </view>

        <view class="template-row">
          <!-- 情绪 -->
          <view class="template-card">
            <view class="card-header">
              <view class="card-icon" style="background:rgba(255,237,213,1)">😊</view>
              <text class="card-title">情绪</text>
            </view>
            <view class="card-body">
<!-- 上午 -->
              <view class="meal-row">
                <text class="meal-label">上午</text>
                <view class="emoji-row">
                  <view class="emoji-btn" :style="{ background: form.morning_mood === 3 ? '#dcfce7' : '#f1f5f9' }" @click="form.morning_mood = form.morning_mood === 3 ? 0 : 3">😋</view>
                  <view class="emoji-btn" :style="{ background: form.morning_mood === 2 ? '#dcfce7' : '#f1f5f9' }" @click="form.morning_mood = form.morning_mood === 2 ? 0 : 2">😐</view>
                  <view class="emoji-btn" :style="{ background: form.morning_mood === 1 ? '#dcfce7' : '#f1f5f9' }" @click="form.morning_mood = form.morning_mood === 1 ? 0 : 1">😞</view>
                </view>
              </view>
              <!-- 下午 -->
              <view class="meal-row">
                <text class="meal-label">下午</text>
                <view class="emoji-row">
                  <view class="emoji-btn" :style="{ background: form.afternoon_mood === 3 ? '#dcfce7' : '#f1f5f9' }" @click="form.afternoon_mood = form.afternoon_mood === 3 ? 0 : 3">😋</view>
                  <view class="emoji-btn" :style="{ background: form.afternoon_mood === 2 ? '#dcfce7' : '#f1f5f9' }" @click="form.afternoon_mood = form.afternoon_mood === 2 ? 0 : 2">😐</view>
                  <view class="emoji-btn" :style="{ background: form.afternoon_mood === 1 ? '#dcfce7' : '#f1f5f9' }" @click="form.afternoon_mood = form.afternoon_mood === 1 ? 0 : 1">😞</view>
                </view>
              </view>
            </view>
          </view>

          <!-- 表现 -->
          <view class="template-card">
            <view class="card-header">
              <view class="card-icon" style="background:rgba(254,226,226,1)">🌟</view>
              <text class="card-title">表现</text>
            </view>
            <view class="card-body">
              <view class="tag-cloud">
                <view
                  v-for="t in performanceOptions" :key="t"
                  class="tag"
                  :class="{ active: form.performance_tags.includes(t) }"
                  @click="toggleTag(t)"
                >{{ t }}</view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <!-- 指定学生模式 — 学生选择器 + 表现标签 -->
      <view v-if="form.target_type === 2" class="section">
        <view class="student-head">
          <text class="section-title">选择学生</text>
          <text class="student-count">已选 {{ form.student_ids.length }} 人</text>
        </view>
        <view v-if="students.length" class="student-search">
          <input
            v-model="studentQuery"
            class="search-input"
            placeholder="搜索学生姓名"
            placeholder-style="color: #94A3B8"
          />
        </view>
        <view class="student-picker">
          <view
            v-for="s in filteredStudents"
            :key="s.student_id"
            class="student-chip"
            :class="{ selected: form.student_ids.includes(s.student_id) }"
            @click="toggleStudent(s.student_id)"
          >
            <text>{{ s.name }}</text>
          </view>
          <view v-if="!students.length" class="student-empty">
            <text>班级暂无学生</text>
          </view>
          <view v-else-if="!filteredStudents.length" class="student-empty">
            <text>未找到匹配的学生</text>
          </view>
        </view>

        <text class="section-title" style="margin-top: 32rpx">表现标签</text>
        <view class="tag-cloud">
          <view
            v-for="t in performanceOptions" :key="t"
            class="tag"
            :class="{ active: form.performance_tags.includes(t) }"
            @click="toggleTag(t)"
          >{{ t }}</view>
        </view>
      </view>

      <!-- 照片/视频上传 -->
      <view class="section">
        <text class="section-title">照片/视频上传</text>
        <view class="upload-card">
          <!-- 点名工具条：批量多选 / 单张在照片角标进入 -->
          <view v-if="form.media.length" class="naming-bar">
            <template v-if="!batchMode">
              <text class="naming-bar-hint">已点名 {{ form.media.filter(m => namedCount(m)).length }} 张</text>
              <view class="naming-bar-btn" @click="startBatch">批量点名</view>
            </template>
            <template v-else>
              <text class="naming-bar-hint">已选 {{ batchSelected.size }} 张照片</text>
              <view class="naming-bar-btn" @click="toggleSelectAll">
                {{ allPhotosSelected ? '取消全选' : '全选' }}
              </view>
              <view class="naming-bar-btn primary" @click="openBatchNaming">选择学生</view>
              <view class="naming-bar-btn" @click="cancelBatch">取消</view>
            </template>
          </view>
          <!-- 拍照与点名规范（D11 SOP）：单人照是 AI 点名的好输入；合影走「全班」一键 -->
          <text v-if="form.media.length" class="naming-sop-hint">
            拍照建议：尽量拍单人照（AI 自动点名更准）；集体照用「全班」一键点名
          </text>
          <view class="media-grid">
            <view class="media-item add-btn" @click="addMedia">
              <text class="add-icon">＋</text>
              <text class="add-text">添加</text>
            </view>
            <view
              v-for="(m, i) in form.media" :key="i"
              class="media-item"
              :class="{ picking: batchMode, picked: batchSelected.has(i) }"
              @click="batchMode && toggleBatchSelect(i)"
            >
              <image :src="mediaDisplayUrl(m)" mode="aspectFill" class="media-img" />
              <view v-if="m.status === 'uploading'" class="media-mask">
                <text class="media-mask-text">上传中</text>
              </view>
              <view v-else-if="m.status === 'failed'" class="media-mask media-mask-failed" @click.stop="retryMedia(i)">
                <text class="media-mask-text">上传失败 · 重试</text>
              </view>
              <view v-if="batchMode" class="media-check">
                <text>{{ batchSelected.has(i) ? '☑' : '☐' }}</text>
              </view>
              <template v-else>
                <view v-if="m.status === 'done'" class="media-naming-badge" @click.stop="openSingleNaming(i)">
                  <text>{{ namedCount(m) ? '点名 ' + namedCount(m) : '点名' }}</text>
                </view>
                <view v-if="m.status === 'done' && mediaSuggestionCount(m)" class="media-ai-tag" @click.stop="openSingleNaming(i)">
                  <text>AI建议 {{ mediaSuggestionCount(m) }}</text>
                </view>
                <view class="media-remove" @click.stop="removeMedia(i)">✕</view>
              </template>
            </view>
          </view>
          <!-- 每张照片的引用层描述（这次为什么用这张） -->
          <view v-if="form.media.length" class="media-desc-list">
            <view v-for="(m, i) in form.media" :key="'d' + i" class="media-desc-row">
              <image :src="mediaDisplayUrl(m)" mode="aspectFill" class="media-desc-thumb" />
              <input v-model="m.ref_desc" class="media-desc-input" placeholder="为这张照片写句说明（可选）" placeholder-style="color: #94A3B8" />
            </view>
          </view>
          <view class="upload-hint">
            <text>ℹ️</text>
            <text>最多可上传20张照片（每张不超过20MB）</text>
          </view>
        </view>
      </view>

      <!-- 记录内容 -->
      <view class="section">
        <text class="section-title">记录内容</text>
        <view class="content-card">
          <textarea
            v-model="form.content"
            class="content-area"
            placeholder="请输入记录内容，描述今天孩子们的活动和表现..."
          />
        </view>
      </view>

      <!-- 底部留白 -->
      <view style="height:140rpx"></view>
      <!-- 授予徽章（可选）：仅新建时提供，发布成功后授予，本条记录可作为获奖见证 -->
      <view class="section" v-if="!isEdit && achievements.length">
        <text class="section-title">授予徽章（可选）</text>
        <view class="badge-pick-row">
          <view
            v-for="a in achievements" :key="a.id"
            class="badge-pick"
            :class="{ active: grant.achievement_id === a.id }"
            @click="selectAchievement(a.id)"
          >
            <image v-if="a.icon" :src="a.icon" class="badge-pick-icon" mode="aspectFit" />
            <text class="badge-pick-name">{{ a.name }}</text>
          </view>
        </view>

        <template v-if="grant.achievement_id">
          <text class="section-title" style="margin-top: 24rpx">授予给</text>
          <view class="student-row" v-if="students.length">
            <view
              v-for="s in students" :key="s.student_id"
              class="student-chip"
              :class="{ selected: grant.student_ids.includes(s.student_id) }"
              @click="toggleGrantStudent(s.student_id)"
            >
              {{ s.name }}
            </view>
          </view>
          <view class="option-hint" v-else>
            <text>不选则默认授予给本记录的目标学生</text>
          </view>

          <view class="option-row" @click="grant.as_witness = !grant.as_witness" style="margin-top: 16rpx">
            <text class="option-check">{{ grant.as_witness ? '☑' : '☐' }}</text>
            <text class="option-label" :class="{ active: grant.as_witness }">本记录作为获奖见证（家长将在该记录看到徽章）</text>
          </view>
        </template>
      </view>
    </scroll-view>

    <!-- 照片点名面板（底部弹层）：单张 = 编辑完整名单；批量 = 追加勾选的学生 -->
    <view v-if="namingOpen" class="naming-mask" @click="closeNaming">
      <view class="naming-sheet" @click.stop>
        <view class="naming-head">
          <text class="naming-title">
            {{ namingTargets.length === 1 ? '照片点名' : `批量点名 · ${namingTargets.length} 张照片` }}
          </text>
          <text class="naming-close" @click="closeNaming">✕</text>
        </view>
        <text class="naming-tip">
          {{ namingIsBatch
            ? `勾选的孩子将追加到这批照片；已有名单已标出（确定后每张共 ${namingTotalCount()} 人）`
            : '勾选这张照片里的孩子（AI 建议可一并确认）' }}
        </text>
        <!-- D11：全班一键 = 当日到场学生（服务端口径），填入后仍可逐人增减。
             个体记录的候选本来就只有目标孩子，「全班」语义不成立，不呈现。 -->
        <view v-if="form.target_type !== 2" class="naming-roster-row">
          <view class="naming-roster-btn" :class="{ busy: rosterLoading }" @click="pickWholeClass">
            {{ rosterLoading ? '获取到场名单…' : '全班（当日到场）' }}
          </view>
          <text class="naming-roster-hint">合影/集体照用这个：自动填入当天到场的孩子，再按画面增减</text>
        </view>
        <scroll-view scroll-y class="naming-scroll">
          <view class="student-picker">
            <view
              v-for="s in namingCandidates" :key="s.student_id"
              class="student-chip"
              :class="{
                selected: namingStudents.has(s.student_id) || batchExisting.has(s.student_id),
                existing: batchExisting.has(s.student_id),
              }"
              @click="toggleNamingStudent(s.student_id)"
            >
              <text>{{ s.name }}</text>
              <text v-if="batchExisting.has(s.student_id)" class="naming-existing-mark">已有</text>
              <text v-else-if="suggestionOf(s.student_id)" class="naming-ai-mark">
                {{ suggestionOf(s.student_id).tier === 'high' ? 'AI·高' : 'AI' }}
              </text>
            </view>
            <view v-if="!namingCandidates.length" class="student-empty">
              <text>{{ form.target_type === 2 ? '请先选择本记录的目标学生' : '班级暂无学生' }}</text>
            </view>
          </view>
        </scroll-view>
        <view class="naming-footer">
          <view class="naming-btn" @click="closeNaming">取消</view>
          <view class="naming-btn primary" @click="confirmNaming">
            确定{{ namingTotalCount() ? `（共 ${namingTotalCount()} 人）` : '' }}
          </view>
        </view>
      </view>
    </view>

    <!-- 底部操作栏（编辑态锁定可见性设置，仅允许改内容） -->
    <view class="bottom-bar">
      <view class="bottom-options">
        <template v-if="isEdit">
          <view class="option-hint">
            <text>{{ form.target_type === 2 ? '该记录仅被选中的学生家长可见' : '该记录对全班家长可见' }}</text>
          </view>
        </template>
        <template v-else>
          <view v-if="form.target_type === 1" class="option-row" @click="form.sync_to_feed = !form.sync_to_feed">
            <text class="option-check">{{ form.sync_to_feed ? '☑' : '☐' }}</text>
            <text class="option-label active">同步发送到班级动态</text>
          </view>
          <view v-if="form.target_type === 2" class="option-hint">
            <text>该记录仅被选中的学生家长可见</text>
          </view>
          <view class="option-row" @click="switchTargetType">
            <text class="option-check">{{ form.target_type === 2 ? '☑' : '☐' }}</text>
            <text class="option-label" :class="{ active: form.target_type === 2 }">仅发送给指定家长</text>
          </view>
        </template>
      </view>
      <button class="publish-btn" :loading="submitting" @click="publish">
        {{ submitting ? (isEdit ? "保存中..." : "发布中...") : (isEdit ? "保存修改" : (form.target_type === 2 ? "发布个体记录" : "发布班级记录")) }}
      </button>
    </view>
  </view>
</template>

<style scoped lang="scss">

.page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: $color-bg-card;
}

.body { flex: 1; padding-top: 16rpx; }

/* 日期栏 */
.date-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-xl 48rpx;
  background: $color-bg-card;
}
.date-arrow { padding: $spacing-xs; }
.date-arrow.disabled { opacity: 0.3; }
.date-text { font-size: $typography-font-size-xl; font-weight: 700; color: $color-text-primary; }

/* 通用 section */
.section { padding: $spacing-xl 48rpx; }
.section-title {
  display: block;
  margin-bottom: 24rpx;
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

/* 模板卡片 */
.template-row {
  display: flex;
  gap: $spacing-lg;
  margin-bottom: 24rpx;
}
.template-card {
  flex: 1;
  padding: $spacing-xl;
  background: $color-bg-card;
  border: 1rpx solid $color-bg-page;
  border-radius: $border-radius-xl;
  box-shadow: 0 2rpx 4rpx rgba(120, 53, 15, 0.05);
}
.card-header {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: 16rpx;
}
.card-icon {
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-lg;
  font-size: $typography-font-size-2xl;
}
.card-title { font-size: $typography-font-size-md; font-weight: 700; color: $color-text-primary; }
.card-body { display: flex; flex-direction: column; gap: $spacing-xs; }
.meal-row { display: flex; align-items: center; justify-content: space-between; }
.meal-label { font-size: $typography-font-size-base; color: $color-text-muted; width: 54rpx; }
.emoji-row { display: flex; gap: $spacing-xs; }
.emoji-btn {
  width: 48rpx;
  height: 48rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-sm;
  font-size: $typography-font-size-md;
}
.field-label { font-size: $typography-font-size-base; color: $color-text-muted; }
.chip-row { display: flex; gap: $spacing-md; flex-wrap: wrap; }
.chip {
  padding: $spacing-sm 24rpx;
  background: $color-bg-page;
  border-radius: $border-radius-lg;
  font-size: $typography-font-size-base;
  color: $color-text-muted;
  &.active { background: #eff6ff; color: $color-semantic-info; font-weight: 600; }
}
.tag-cloud { display: flex; flex-wrap: wrap; gap: $spacing-sm; }
.tag {
  padding: $spacing-xs 16rpx;
  background: $color-bg-page;
  border-radius: $border-radius-sm;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  &.active { background: #eff6ff; color: $color-semantic-info; font-weight: 600; }
}

/* 上传 */
.upload-card {
  padding: $spacing-xl;
  background: $color-bg-card;
  border: 1rpx solid $color-bg-page;
  border-radius: $border-radius-xl;
  box-shadow: 0 2rpx 4rpx rgba(120, 53, 15, 0.05);
}
.media-grid {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-lg;
}
.media-item {
  width: 160rpx;
  height: 160rpx;
  border-radius: $border-radius-xl;
  overflow: hidden;
  position: relative;
}
.add-btn {
  border: 1rpx dashed #cbd5e1;
  background: $color-bg-card;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
}
.add-icon { font-size: 48rpx; color: $color-text-hint; }
.add-text { font-size: $typography-font-size-sm; color: $color-text-hint; }
.media-img { width: 100%; height: 100%; }
.media-remove {
  position: absolute;
  top: 8rpx;
  right: 8rpx;
  width: 40rpx;
  height: 40rpx;
  background: rgba(0,0,0,0.5);
  color: #fff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: $typography-font-size-sm;
}
/* 上传中/失败遮罩：对位在缩略图上，失败可点重试 */
.media-mask {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
}
.media-mask-failed { background: rgba(180, 30, 30, 0.55); }
.media-mask-text {
  color: #fff;
  font-size: $typography-font-size-sm;
  text-align: center;
  padding: 0 12rpx;
}
/* ── 照片点名（D8）────────────────────────────────────────────── */
.naming-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
}
.naming-bar-hint {
  margin-right: auto;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}
/* D11 拍照与点名规范提示（纯文案，不占操作位） */
.naming-sop-hint {
  display: block;
  margin-bottom: $spacing-md;
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}
.naming-bar-btn {
  padding: $spacing-xs $spacing-lg;
  border-radius: $border-radius-full;
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-card;
  font-size: $typography-font-size-sm;
  color: $color-text-body;
}
.naming-bar-btn.primary {
  background: $color-brand-primary;
  border-color: $color-brand-primary;
  color: \$color-text-on-brand;
}
/* 批量模式：照片可勾选 */
.media-item.picking { opacity: 0.92; }
.media-item.picked { border: 4rpx solid $color-brand-primary; border-radius: $border-radius-xl; }
.media-check {
  position: absolute;
  left: $spacing-xs;
  bottom: $spacing-xs;
  font-size: $typography-font-size-2xl;
  color: #fff;
  text-shadow: 0 0 6rpx rgba(0,0,0,0.6);
}
.media-naming-badge {
  position: absolute;
  left: 8rpx;
  bottom: 8rpx;
  padding: 4rpx 12rpx;
  border-radius: $border-radius-full;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: $typography-font-size-xs;
}
.media-ai-tag {
  position: absolute;
  left: 8rpx;
  top: 8rpx;
  padding: 4rpx 10rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
  font-size: $typography-font-size-xs;
}
/* 点名弹层 */
.naming-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  z-index: 100;
  display: flex;
  align-items: flex-end;
}
.naming-sheet {
  width: 100%;
  max-height: 75vh;
  background: $color-bg-card;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  padding: $spacing-xl;
  padding-bottom: calc(#{$spacing-xl} + env(safe-area-inset-bottom));
  display: flex;
  flex-direction: column;
}
.naming-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.naming-title {
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
}
.naming-close {
  font-size: $typography-font-size-lg;
  color: $color-text-hint;
  padding: 0 $spacing-sm;
}
.naming-tip {
  margin-top: $spacing-sm;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}
.naming-scroll {
  margin-top: $spacing-lg;
  max-height: 48vh;
}
/* D11 全班一键：默认名单 = 当日到场学生 */
.naming-roster-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-md;
}
.naming-roster-btn {
  flex-shrink: 0;
  padding: $spacing-xs $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
  font-size: $typography-font-size-sm;
}
.naming-roster-btn.busy { opacity: 0.6; }
.naming-roster-hint {
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}
.naming-ai-mark {
  margin-left: 8rpx;
  font-size: $typography-font-size-xs;
  color: $color-semantic-info;
}
/* 「已有」标记：批量面板里显性呈现照片上已点名的孩子（不可取消） */
.naming-existing-mark {
  margin-left: 8rpx;
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}
.student-chip.existing {
  opacity: 0.85;
}
.naming-footer {
  display: flex;
  gap: $spacing-lg;
  margin-top: $spacing-lg;
}
.naming-btn {
  flex: 1;
  height: 88rpx;
  border-radius: $border-radius-full;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: $typography-font-size-base;
  background: $color-bg-page;
  color: $color-text-body;
}
.naming-btn.primary {
  background: $color-brand-primary;
  color: \$color-text-on-brand;
}
/* 每张照片的引用层描述（L2） */
.media-desc-list {
  margin-top: 20rpx;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}
.media-desc-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}
.media-desc-thumb {
  width: 64rpx;
  height: 64rpx;
  border-radius: $border-radius-md;
  flex-shrink: 0;
}
.media-desc-input {
  flex: 1;
  min-width: 0;
  height: 64rpx;
  padding: 0 20rpx;
  background: $color-bg-page;
  border-radius: $border-radius-md;
  font-size: $typography-font-size-base;
  color: $color-text-primary;
}
.upload-hint {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: 24rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

/* 记录内容 */
.content-card {
  padding: $spacing-xl;
  background: $color-bg-card;
  border: 1rpx solid $color-bg-page;
  border-radius: $border-radius-xl;
  box-shadow: 0 2rpx 4rpx rgba(120, 53, 15, 0.05);
}
.content-area {
  width: 100%;
  height: 240rpx;
  font-size: $typography-font-size-lg;
  color: $color-text-primary;
  box-sizing: border-box;
}

/* 底部操作栏 */
.bottom-bar {
  padding: $spacing-xl 48rpx 48rpx 48rpx;
  background: $color-bg-card;
}
.bottom-options {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  margin-bottom: 24rpx;
}
.option-row { display: flex; align-items: center; gap: $spacing-md; }
.option-check { font-size: $typography-font-size-md; color: $color-semantic-info; width: 40rpx; }
.option-label { font-size: $typography-font-size-md; color: $color-text-hint; &.active { color: $color-semantic-info; } }
.publish-btn {
  width: 100%;
  height: 96rpx;
  background: linear-gradient(135deg, $color-semantic-info, $color-semantic-info);
  color: #fff;
  font-size: $typography-font-size-xl;
  font-weight: 700;
  border-radius: $border-radius-xl;
  border: 0;
  line-height: 96rpx;
}

/* 学生选择器 */
.student-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.student-count {
  font-size: $typography-font-size-sm;
  color: $color-semantic-info;
}

.student-search {
  margin-bottom: $spacing-md;
}

.search-input {
  height: 72rpx;
  padding: 0 28rpx;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

.student-picker {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;
}

.student-chip {
  padding: 14rpx 32rpx;
  border-radius: 999rpx;
  background: $color-bg-page;
  font-size: $typography-font-size-md;
  color: $color-text-muted;
  transition: all 0.2s;

  &.selected {
    background: #eff6ff;
    color: $color-semantic-info;
    font-weight: 600;
  }
}

.student-empty {
  width: 100%;
  text-align: center;
  padding: $spacing-xl;
  font-size: $typography-font-size-base;
  color: $color-text-hint;
}

.option-hint {
  padding: $spacing-sm 24rpx;
  margin-bottom: 8rpx;
  border-radius: $border-radius-md;
  background: #eff6ff;
  font-size: $typography-font-size-sm;
  color: $color-semantic-info;
}

// 授予徽章选择器
.badge-pick-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;
}

.badge-pick {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  padding: 14rpx 28rpx;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  transition: all 0.2s;

  &.selected,
  &.active {
    background: $color-semantic-info-bg;
    box-shadow: inset 0 0 0 2rpx $color-semantic-info;
  }
}

.badge-pick-icon {
  width: 40rpx;
  height: 40rpx;
  border-radius: $border-radius-sm;
}

.badge-pick-name {
  font-size: $typography-font-size-md;
  color: $color-text-primary;
  font-weight: 500;
}

.student-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;
}
</style>
