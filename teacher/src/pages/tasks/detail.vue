<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh, onUnload } from "@dcloudio/uni-app"
import {
  fetchTeacherTask,
  fetchTeacherTaskStats,
  fetchTeacherTaskCheckins,
  respondTeacherTask,
  batchRespondTeacherTask,
  supplementTeacherResponse,
  reviewTeacherTaskPublic,
  updateTeacherTask,
  fetchTaskShareCard
} from "@/api/teacher-tasks"
import { fetchTeacherClassInfo } from "@/api/teacher-class"
import config from "@/config/env"
import { getToken } from "@/utils/auth"
import {
  formatClock,
  formatCommentTime,
  formatDateLabel,
  formatRecordTime,
  todayKey
} from "@/utils/time"
import {
  REVIEW_FILTERS,
  freqLabel,
  isTaskActive,
  publicStateHint,
  publicStateText,
  reviewKey,
  ritualLabel,
  starTracking,
  stickerGlyph
} from "@/utils/tasks"
import PageShell from "@/components/PageShell.vue"

const SUPPLEMENT_STICKERS = ["flower", "heart", "star", "thumb"]

const taskId = ref(0)
const task = ref({})
const taskError = ref("")
const className = ref("")
const today = ref(todayKey())

const stats = ref({})
const statsError = ref("")
const statsLoaded = ref(false)
let statsSeq = 0

const pendingRows = ref([])
const respondedRows = ref([])
const hasLoaded = ref(false)
const loading = ref(false)
const loadError = ref("")
let checkinSeq = 0

// 服务端只有 pending / responded 两段；切段不重新请求，两段同时拉到本地，
// 这样审核 chip 的计数（全局）与「切到另一段看 N 条」才有真实依据。
const status = ref("pending")
const reviewFilter = ref("all")
const rows = computed(() =>
  status.value === "pending" ? pendingRows.value : respondedRows.value
)
const filteredRows = computed(() => {
  if (reviewFilter.value === "all") return rows.value
  return rows.value.filter((row) => reviewKey(row) === reviewFilter.value)
})

const hardError = computed(() => Boolean(loadError.value) && !hasLoaded.value)
// 首次渲染到 onLoad 发起请求之间也算加载中：否则会先闪一下空列表。
const isInitialLoading = computed(() => !hasLoaded.value && !hardError.value)
const isEmpty = computed(
  () =>
    hasLoaded.value &&
    !loadError.value &&
    pendingRows.value.length === 0 &&
    respondedRows.value.length === 0
)
const showContent = computed(() => !isInitialLoading.value && !hardError.value)

const taskActive = computed(() => isTaskActive(task.value, today.value))
const taskRange = computed(() => {
  const start = formatDateLabel(task.value.start_date)
  const end = formatDateLabel(task.value.end_date)
  if (!start && !end) return "—"
  return `${start || "—"} ~ ${end || "—"}`
})
const taskScopeLabel = computed(() =>
  Number(task.value.target_student_id) > 0 ? "个体任务" : "全班"
)
const starCount = computed(() =>
  task.value.star_count == null ? null : Number(task.value.star_count)
)
const showStarStrip = computed(
  () => Boolean(task.value.id) && starTracking(task.value) && starCount.value !== null
)
const starStripSub = computed(() =>
  task.value.star_scope === "class"
    ? "按班级星星罐"
    : `按任务星星罐 · 目标 ${Number(task.value.star_goal || 0)}`
)

// stats 计数失败时退回当前已加载的行数：分段的数字宁可少算，也不能显示成 0 当作事实。
function statCount(key, fallback) {
  if (!statsLoaded.value || statsError.value) return fallback
  const value = Number(stats.value?.[key])
  return Number.isFinite(value) ? value : fallback
}
const pendingCount = computed(() => statCount("pending_count", pendingRows.value.length))
const respondedCount = computed(() => statCount("responded_count", respondedRows.value.length))
const totalDisplay = computed(() =>
  statsLoaded.value && !statsError.value ? Number(stats.value.total_students || 0) : "—"
)
// 契约里 participant_count 是去重学生数；用「已提交」而不是「已参与」时，
// 这里仍取去重口径（brief B2），行数口径的 submission_count 只用于结构锁定。
const submittedDisplay = computed(() =>
  statsLoaded.value && !statsError.value ? Number(stats.value.participant_count || 0) : "—"
)
const structureLocked = computed(() => Number(stats.value?.submission_count || 0) > 0)
const structureHint = computed(() => {
  if (statsError.value) return "未能确认提交数，结构按只读展示；关闭弹层后可在统计卡重试。"
  if (structureLocked.value) {
    return `已有 ${Number(stats.value.submission_count)} 条提交，结构已锁定；要调整结构请新建任务。`
  }
  return "结构字段只在创建时设置；如需调整请新建任务。"
})

const reviewCounts = computed(() => {
  const counts = { all: 0, pending: 0, approved: 0, revoked: 0 }
  for (const row of [...pendingRows.value, ...respondedRows.value]) {
    counts.all += 1
    const key = reviewKey(row)
    if (key in counts) counts[key] += 1
  }
  return counts
})

const otherRows = computed(() =>
  status.value === "pending" ? respondedRows.value : pendingRows.value
)
const otherFilteredCount = computed(() => {
  if (reviewFilter.value === "all") return otherRows.value.length
  return otherRows.value.filter((row) => reviewKey(row) === reviewFilter.value).length
})
const filterJumpText = computed(() => {
  if (!otherFilteredCount.value) return ""
  const target = status.value === "pending" ? "已回应" : "待回应"
  return `切到「${target}」看 ${otherFilteredCount.value} 条`
})

// 与后端 batchRespondAllowed 对齐：只有每日/每周的班级任务提供汇总入口。
const canBatch = computed(
  () =>
    Boolean(task.value.id) &&
    Number(task.value.target_student_id || 0) === 0 &&
    ["daily", "weekly"].includes(task.value.frequency)
)
const batchIsStar = computed(() => task.value.recognition_ritual === "class_star")
const batchLabel = computed(() =>
  batchIsStar.value ? "一键送星（本周期）" : "一键送贴纸（本周期）"
)
const showBatch = computed(
  () =>
    showContent.value &&
    status.value === "pending" &&
    !isEmpty.value &&
    pendingRows.value.length > 0
)
const showSettings = computed(() => showContent.value && !isEmpty.value && Boolean(task.value.id))

// 批准公开的前置条件（契约）：任务开启公开且不是个体任务。孩子是否仍在班由后端判定。
const canApprovePublic = computed(
  () =>
    Boolean(task.value.id) &&
    Boolean(task.value.allow_public) &&
    Number(task.value.target_student_id || 0) === 0
)

// ---------- 请求 ----------
async function loadTask() {
  try {
    const res = await fetchTeacherTask(taskId.value, { silent: true })
    task.value = res.data || {}
    taskError.value = ""
    fillEdit()
  } catch (e) {
    // 任务详情只影响编辑区、汇总入口与批量入口，不阻塞提交列表；但要显式说明失败。
    taskError.value = e?.message || "任务详情加载失败"
  }
}

async function loadStats() {
  const seq = ++statsSeq
  try {
    const res = await fetchTeacherTaskStats(taskId.value, { silent: true })
    if (seq !== statsSeq) return
    stats.value = res.data || {}
    statsError.value = ""
  } catch (e) {
    if (seq !== statsSeq) return
    statsError.value = "提交数加载失败"
  } finally {
    if (seq === statsSeq) statsLoaded.value = true
  }
}

async function loadClassInfo() {
  try {
    const res = await fetchTeacherClassInfo()
    className.value = res.data?.name || ""
  } catch (e) {
    // 班级名只用于任务卡副标题；失败时退回「本班」。
    console.warn("[tasks-detail] load class failed:", e?.message || e)
  }
}

async function loadCheckins() {
  if (!taskId.value) return
  const seq = ++checkinSeq
  loading.value = true
  try {
    const [pendingRes, respondedRes] = await Promise.all([
      fetchTeacherTaskCheckins(taskId.value, { status: "pending" }, { silent: true }),
      fetchTeacherTaskCheckins(taskId.value, { status: "responded" }, { silent: true })
    ])
    if (seq !== checkinSeq) return
    pendingRows.value = Array.isArray(pendingRes.data?.list) ? pendingRes.data.list : []
    respondedRows.value = Array.isArray(respondedRes.data?.list) ? respondedRes.data.list : []
    loadError.value = ""
    hasLoaded.value = true
    // 撤回的确认不跨刷新沿用：否则上一轮点过「撤回公开」没确认，刷新后一次点击就撤掉。
    reviewConfirm.value = { id: 0, mode: "" }
    // 每轮加载都作废勾选。服务端没有「本轮申请时间」，家长「撤回→重新申请」
    // 若发生在两次加载之间，pending 状态看起来没变，保留勾选就等于绕过重确认。
    scopeConfirmed.value = {}
  } catch (e) {
    if (seq !== checkinSeq) return
    loadError.value = "回应加载失败"
  } finally {
    if (seq === checkinSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
  }
}

// 只刷新周期与星数，不覆盖任务对象的其它字段（编辑弹层可能正在用它们）。
async function refreshPeriod(seq) {
  try {
    const res = await fetchTeacherTask(taskId.value, { silent: true })
    if (seq !== undefined && seq !== checkinSeq) return
    if (res.data?.period_start) task.value.period_start = res.data.period_start
    if (res.data?.period_end) task.value.period_end = res.data.period_end
    // 服务端明确给 null（星数读失败）时也要落下来，让 UI 隐藏「已集」，
    // 不能让一个陈旧的 0 继续冒充当前值。
    if (res.data?.star_count !== undefined) task.value.star_count = res.data.star_count
  } catch (e) {
    // 周期/星数刷新失败不影响提交列表；旧区间最多导致「本周期」判断短暂陈旧
  }
}

async function retryAll() {
  today.value = todayKey()
  await Promise.all([loadTask(), loadStats(), loadCheckins(), loadClassInfo()])
  if (!loadError.value) {
    uni.showToast({ title: "已重新加载", icon: "none" })
  }
}

async function refreshCheckins() {
  await Promise.all([loadCheckins(), loadStats(), loadTask()])
  if (!loadError.value) {
    uni.showToast({ title: "已重新加载", icon: "none" })
  }
}

// 等列表刷新落地再解锁：刷新还在飞时解锁，按钮立刻又活了，等于没加闩。
// 但 uni.request 没设 timeout（走微信默认 60s），封顶 3 秒，卡住时不能一直锁着。
async function refreshWithCap() {
  const seq = checkinSeq + 1
  await Promise.race([
    // 不整单 loadTask：编辑弹层里可能有未保存的输入，刷新任务对象会把它覆盖掉。
    Promise.all([loadCheckins(), loadStats(), refreshPeriod(seq)]),
    new Promise((resolve) => setTimeout(resolve, 3000))
  ])
  if (loading.value) loading.value = false
}

// ---------- 公开审核 ----------
const scopeConfirmed = ref({})
const reviewingIds = ref([])
const reviewConfirm = ref({ id: 0, mode: "" })
const isReviewing = (id) => reviewingIds.value.includes(id)

function toggleScopeConfirm(id) {
  if (isReviewing(id)) return
  scopeConfirmed.value = { ...scopeConfirmed.value, [id]: !scopeConfirmed.value[id] }
}

function askReview(row, mode) {
  if (isReviewing(row.id)) return
  reviewConfirm.value = { id: row.id, mode }
}

function cancelReviewConfirm() {
  reviewConfirm.value = { id: 0, mode: "" }
}

async function reviewPublic(row, approve) {
  if (isReviewing(row.id)) return
  if (approve && !scopeConfirmed.value[row.id]) {
    uni.showToast({ title: "请先勾选确认内容中只有提交者孩子", icon: "none" })
    return
  }
  if (!approve && reviewConfirm.value.id !== row.id) return
  const mode = approve ? "approve" : reviewConfirm.value.mode
  reviewingIds.value = [...reviewingIds.value, row.id]
  try {
    await reviewTeacherTaskPublic(
      taskId.value,
      row.id,
      { approve, scope_confirmed: approve },
      { silent: true }
    )
    if (approve) {
      uni.showToast({ title: "已批准公开", icon: "success" })
    } else if (mode === "revoke") {
      uni.showToast({ title: "已撤回公开", icon: "success" })
    } else {
      uni.showToast({ title: "已拒绝公开申请", icon: "success" })
    }
    reviewConfirm.value = { id: 0, mode: "" }
    // 这一轮审核已结束，确认勾选作废；家长再次申请时要重新确认。
    scopeConfirmed.value = { ...scopeConfirmed.value, [row.id]: false }
    await refreshWithCap()
  } catch (e) {
    uni.showToast({ title: e?.message || "审核失败", icon: "none" })
  } finally {
    reviewingIds.value = reviewingIds.value.filter((id) => id !== row.id)
  }
}

// ---------- 单条/批量回应 ----------
const batchBusy = ref(false)
const respondingIds = ref([])
const isResponding = (id) => respondingIds.value.includes(id)
function addResponding(id) {
  if (!respondingIds.value.includes(id)) respondingIds.value = [...respondingIds.value, id]
}
function removeResponding(id) {
  respondingIds.value = respondingIds.value.filter((x) => x !== id)
}

const recording = ref(false)
const recordingRowId = ref(0)
const recordingMode = ref("")
const recordingSeconds = ref(0)
const recorder = uni.getRecorderManager()
let recordingRow = null
let recordingTimer = null
let discardRecording = false
let pageGone = false
let audio = null

function startRecordTimer() {
  stopRecordTimer()
  recordingSeconds.value = 0
  recordingTimer = setInterval(() => {
    recordingSeconds.value += 1
  }, 1000)
}

function stopRecordTimer() {
  if (recordingTimer) {
    clearInterval(recordingTimer)
    recordingTimer = null
  }
}

function recordLabel() {
  const seconds = recordingSeconds.value
  const mm = Math.floor(seconds / 60)
  const ss = String(seconds % 60).padStart(2, "0")
  return `停止 ${mm}:${ss}`
}

recorder.onStop((res) => {
  recording.value = false
  recordingRowId.value = 0
  stopRecordTimer()
  const row = recordingRow
  const mode = recordingMode.value
  recordingRow = null
  recordingMode.value = ""
  const discard = discardRecording
  discardRecording = false
  // 页面已卸载/已取消：不再上传，否则会为一次离开页面的录音多打一个接口。
  if (pageGone || discard || !row || !res?.tempFilePath) return
  uploadVoice(res.tempFilePath, row, mode)
})

// 录音失败（无麦克风权限/被占用）必须收闩并说话：否则 recording 永远为 true，
// 页面停在「停止」按钮上，再点也没有任何反应。
recorder.onError(() => {
  recording.value = false
  recordingRowId.value = 0
  recordingRow = null
  recordingMode.value = ""
  stopRecordTimer()
  uni.showToast({ title: "录音失败，请检查麦克风权限", icon: "none" })
})

// 上传期也要占住这一行：否则老师能在「上传语音」转圈时点贴纸/一键汇总，
// 先关掉提交，随后语音回应必然 409，刚录的音频白丢。
function uploadVoice(filePath, row, mode) {
  addResponding(row.id)
  uni.showLoading({ title: "上传语音" })
  uni.uploadFile({
    url: config.uploadURL,
    filePath,
    name: "file",
    formData: { biz_type: "task" },
    header: { Authorization: `Bearer ${getToken()}` },
    success: (up) => {
      if (pageGone) {
        removeResponding(row.id)
        return
      }
      try {
        const body = JSON.parse(up.data || "{}")
        if (body.code !== 0 || !body.data?.id) {
          removeResponding(row.id)
          uni.showToast({ title: body.message || "上传失败", icon: "none" })
          return
        }
        if (mode === "supplement") supplementVoice(row, body.data.id)
        else respondVoice(row, body.data.id)
      } catch (e) {
        removeResponding(row.id)
        uni.showToast({ title: "上传失败", icon: "none" })
      }
    },
    fail: () => {
      removeResponding(row.id)
      if (!pageGone) uni.showToast({ title: "上传失败", icon: "none" })
    },
    complete: () => uni.hideLoading()
  })
}

// D9：通知失败不回滚回应，但计数必须告诉老师，否则家长没收到也没人知道。
function notifySuffix(data) {
  const failed = Number(data?.notify_failed_count || 0)
  return failed > 0 ? `，${failed} 个家长未收到通知` : ""
}

async function respondVoice(row, mediaFileID) {
  try {
    const res = await respondTeacherTask(
      taskId.value,
      row.id,
      { ritual: "voice", media_file_id: mediaFileID },
      { silent: true }
    )
    uni.showToast({ title: "语音已发送" + notifySuffix(res.data), icon: "none" })
    await refreshWithCap()
  } catch (e) {
    uni.showToast({ title: e?.message || "语音回应失败", icon: "none" })
  } finally {
    removeResponding(row.id)
  }
}

async function supplementVoice(row, mediaFileID) {
  if (supplementBusy.value) {
    removeResponding(row.id)
    return
  }
  supplementBusy.value = true
  try {
    await supplementTeacherResponse(
      taskId.value,
      row.id,
      { ritual: "voice", media_file_id: mediaFileID },
      { silent: true }
    )
    uni.showToast({ title: "已追加语音补充", icon: "success" })
    supplementFor.value = 0
    await refreshWithCap()
  } catch (e) {
    uni.showToast({ title: e?.message || "补充回应失败", icon: "none" })
  } finally {
    supplementBusy.value = false
    removeResponding(row.id)
  }
}

// 录音期间不能发起别的回应：批量/贴纸先把提交关掉，录音结束后的语音回应
// 必然撞上 409「已被其他回应关闭」，用户看到的是莫名其妙的失败。
function recordingBlocked() {
  if (!recording.value) return false
  uni.showToast({ title: "请先停止当前录音", icon: "none" })
  return true
}

function startRecord(row, mode) {
  if (recording.value) {
    if (recordingRowId.value !== row.id) {
      uni.showToast({ title: "请先停止当前录音", icon: "none" })
    }
    return
  }
  if (batchBusy.value || isResponding(row.id) || isReviewing(row.id)) {
    uni.showToast({ title: "这一行正在处理，请稍后再录", icon: "none" })
    return
  }
  recordingRow = row
  recordingRowId.value = row.id
  recordingMode.value = mode
  recording.value = true
  startRecordTimer()
  recorder.start({ format: "mp3", duration: 15000 })
}

function stopRecord() {
  if (recording.value) recorder.stop()
}

// 离开当前列表/刷新/收起到期面板前调用：该行的「停止」即将消失，继续录只会在
// 15s 后自动停止并上传一段老师以为已取消的语音。
function cancelRecordingIfAny(message) {
  if (!recording.value) return
  discardRecording = true
  try {
    recorder.stop()
  } catch (e) {
    // stop 不回调时下面兜底复位
  }
  uni.showToast({ title: message, icon: "none" })
  setTimeout(() => {
    if (!recording.value) return
    recording.value = false
    recordingRowId.value = 0
    recordingRow = null
    recordingMode.value = ""
    discardRecording = false
    stopRecordTimer()
  }, 1200)
}

function playAudio(url) {
  if (!url) {
    uni.showToast({ title: "这条语音没有媒体地址", icon: "none" })
    return
  }
  if (audio) {
    audio.stop()
    audio.destroy()
    audio = null
  }
  audio = uni.createInnerAudioContext()
  audio.src = url
  audio.onError(() => uni.showToast({ title: "语音播放失败", icon: "none" }))
  audio.play()
}

async function respond(row, sticker) {
  // 批量在飞时单条必然撞车：那一行可能刚被批量关掉，再发就 409。两个闩互相感知。
  if (recordingBlocked()) return
  if (isResponding(row.id) || batchBusy.value) return
  addResponding(row.id)
  try {
    const res = await respondTeacherTask(
      taskId.value,
      row.id,
      { ritual: "sticker", sticker_key: sticker },
      { silent: true }
    )
    const suffix = notifySuffix(res.data)
    uni.showToast({ title: "已回应" + suffix, icon: suffix ? "none" : "success" })
    await refreshWithCap()
  } catch (e) {
    uni.showToast({ title: e?.message || "回应失败", icon: "none" })
  } finally {
    removeResponding(row.id)
  }
}

async function respondStar(row) {
  if (recordingBlocked()) return
  if (isResponding(row.id) || batchBusy.value) return
  addResponding(row.id)
  try {
    const res = await respondTeacherTask(
      taskId.value,
      row.id,
      { ritual: "class_star" },
      { silent: true }
    )
    const suffix = notifySuffix(res.data)
    uni.showToast({ title: "已加星" + suffix, icon: suffix ? "none" : "success" })
    await refreshWithCap()
  } catch (e) {
    uni.showToast({ title: e?.message || "加星失败", icon: "none" })
  } finally {
    removeResponding(row.id)
  }
}

async function batchRespond() {
  if (recordingBlocked()) return
  // 反向也必须拦：单条在飞时点「一键送星」，会撞上正在关闭的那一行，
  // 返回里就会带 failed_ids，用户看到「已回应 1 条，1 条失败」——又是一种假失败。
  if (batchBusy.value || respondingIds.value.length) return
  batchBusy.value = true
  try {
    // 周期汇总跟随任务默认仪式：班级星星任务必须发 class_star 才进星星罐，
    // 发一枚星形贴纸只是普通贴纸，不计星（D6）。
    const payload = batchIsStar.value
      ? { ritual: "class_star" }
      : { ritual: "sticker", sticker_key: "star" }
    const res = await batchRespondTeacherTask(taskId.value, payload, { silent: true })
    const count = res.data?.responded_count || 0
    const failed = res.data?.failed_ids?.length || 0
    const suffix = notifySuffix(res.data)
    if (count === 0) {
      uni.showToast({ title: "本周期已无待回应提交", icon: "none" })
    } else if (failed > 0) {
      uni.showToast({ title: `已回应 ${count} 条，${failed} 条失败${suffix}`, icon: "none" })
    } else {
      uni.showToast({ title: `已回应 ${count} 条${suffix}`, icon: suffix ? "none" : "success" })
    }
    await refreshWithCap()
  } catch (e) {
    uni.showToast({ title: e?.message || "批量回应失败", icon: "none" })
  } finally {
    batchBusy.value = false
  }
}

// ---------- 补充回应 ----------
const supplementFor = ref(0)
const supplementBusy = ref(false)

function toggleSupplement(row) {
  if (supplementFor.value === row.id) {
    // 收起面板会让这一行的「停止」消失，先取消录音，避免 15s 后自动发出。
    if (recording.value && recordingRowId.value === row.id) cancelRecordingIfAny("已取消录音")
    supplementFor.value = 0
    return
  }
  supplementFor.value = row.id
}

async function supplement(row, sticker) {
  if (recordingBlocked()) return
  if (supplementBusy.value || isResponding(row.id)) return
  supplementBusy.value = true
  try {
    await supplementTeacherResponse(
      taskId.value,
      row.id,
      { ritual: "sticker", sticker_key: sticker },
      { silent: true }
    )
    uni.showToast({ title: "已追加补充回应", icon: "success" })
    supplementFor.value = 0
    await refreshWithCap()
  } catch (e) {
    uni.showToast({ title: e?.message || "补充回应失败", icon: "none" })
  } finally {
    supplementBusy.value = false
  }
}

function firstSupplement(row) {
  const list = row.supplements
  return Array.isArray(list) && list.length ? list[0] : null
}

function supplementGlyph(row) {
  const first = firstSupplement(row)
  if (!first) return ""
  return first.ritual === "voice" ? "" : stickerGlyph(first.sticker_key)
}

function supplementLabel(row) {
  const first = firstSupplement(row)
  if (!first) return ""
  if (first.ritual === "voice") return first.content ? `语音补充 · ${first.content}` : "语音补充"
  return first.content || "贴纸补充"
}

// ---------- 页面展示辅助 ----------
function initialOf(name) {
  return String(name || "").slice(0, 1) || "生"
}

// 家长提交时间：今天只显示「今天 HH:mm」，其他日期用本地化的「10月18日 18:40」。
// 纯日期与时刻分开取，避免把 2026-10-18 当 UTC 解析成前一天。
function submitTime(row) {
  const clock = formatClock(row.created_at)
  if (row.date && row.date === today.value) return clock ? `今天 ${clock}` : "今天"
  return formatRecordTime(row.date, row.created_at)
}

// 旧周期（不在 period_start..period_end 内）的待回应只能逐条处理。
function isCurrentPeriod(row) {
  if (!task.value.period_start || !task.value.period_end || !row.date) return true
  return row.date >= task.value.period_start && row.date <= task.value.period_end
}

function switchStatus(next) {
  if (status.value === next) return
  // 当前行的「停止」按钮会随列表切换消失，先取消录音。
  cancelRecordingIfAny("已取消录音")
  status.value = next
  // 提交数可能在这一页停留期间变化（家长刚提交），切段时一并重取。
  loadStats()
}

function previewMedia(row) {
  if (!row.media_url) return
  uni.previewImage({ urls: [row.media_url] })
}

// ---------- 编辑任务（结构只读） ----------
const editOpen = ref(false)
const savingEdit = ref(false)
const publicOffPending = ref(false)
const edit = ref({ title: "", description: "", learning_link: "", allow_public: false })

function fillEdit() {
  edit.value = {
    title: task.value.title || "",
    description: task.value.description || "",
    learning_link: task.value.learning_link || "",
    allow_public: !!task.value.allow_public
  }
  publicOffPending.value = false
}

function openEdit() {
  if (!task.value.id) return
  fillEdit()
  editOpen.value = true
}

function closeEdit() {
  if (savingEdit.value) return
  editOpen.value = false
  publicOffPending.value = false
}

// 关闭公开是破坏性操作（会批量撤下已公开内容），必须显式确认；
// 原生 uni.showModal 不进视图层 DOM，CDP/自动化点不到，所以用页面内两步确认。
function onPublicToggle(e) {
  const next = !!e.detail.value
  if (!next && edit.value.allow_public) {
    // 先让开关视觉跟随操作，但不允许直接保存：必须点确认。
    edit.value.allow_public = false
    publicOffPending.value = true
    return
  }
  edit.value.allow_public = next
  publicOffPending.value = false
  if (next) uni.showToast({ title: "已开启公开，保存后生效", icon: "none" })
}

function cancelPublicOff() {
  edit.value.allow_public = true
  publicOffPending.value = false
}

function confirmPublicOff() {
  edit.value.allow_public = false
  publicOffPending.value = false
  uni.showToast({ title: "已关闭公开，保存后生效", icon: "none" })
}

const structureStarText = computed(() => {
  if (task.value.star_scope === "class") {
    if (task.value.star_count == null) return "按班级"
    return `按班级 · 已集 ${Number(task.value.star_count)} 星`
  }
  return `按任务 · 目标 ${Number(task.value.star_goal || 0)}`
})

async function saveEdit() {
  if (savingEdit.value) return
  if (publicOffPending.value) {
    uni.showToast({ title: "请先确认是否关闭公开", icon: "none" })
    return
  }
  if (!edit.value.title.trim() || !edit.value.learning_link.trim()) {
    uni.showToast({ title: "标题和学习关联必填", icon: "none" })
    return
  }
  savingEdit.value = true
  try {
    // 结构字段不发送：后端对已有提交的任务会 422，本页也只允许改
    // 标题/说明/学习关联/公开开关（与 saveEdit 的契约一致）。
    await updateTeacherTask(
      taskId.value,
      {
        title: edit.value.title.trim(),
        description: edit.value.description,
        learning_link: edit.value.learning_link.trim(),
        // target_student_id 不是「缺省保留」字段：后端把它当零值，
        // 省略会把个体任务静默改回班级任务。结构虽只读，也必须原样回传。
        target_student_id: Number(task.value.target_student_id || 0),
        allow_public:
          !!edit.value.allow_public && Number(task.value.target_student_id || 0) === 0
      },
      { silent: true }
    )
    editOpen.value = false
    uni.showToast({ title: "已保存", icon: "success" })
    // 关公开/改个体会在服务端批量撤销公开状态；不重取提交列表，
    // 当前页仍会显示「已公开 · 本班家长可见」，与实际相反。
    await Promise.all([loadTask(), loadStats(), loadCheckins()])
  } catch (e) {
    uni.showToast({ title: e?.message || "保存失败", icon: "none" })
  } finally {
    savingEdit.value = false
  }
}

// ---------- 生成小程序码 ----------
const shareBusy = ref(false)

// 生成家长端任务小程序码并保存到相册，老师随后转发班级群。
// 码由后端用**家长端**凭据生成（老师端码家长扫了会进错端），客户端只做落盘。
async function shareTaskCard() {
  if (!taskId.value || shareBusy.value) return
  shareBusy.value = true
  let stage = "generate"
  uni.showLoading({ title: "生成小程序码", mask: true })
  try {
    const res = await fetchTaskShareCard(taskId.value, { silent: true })
    const base64 = res.data?.image_base64 || ""
    if (!base64) {
      // 接口 200 但没给图：必须自己说话，否则点了没反应。
      uni.showToast({ title: "小程序码为空，请检查家长端微信配置", icon: "none" })
      return
    }
    // USER_DATA_PATH 是小程序可写的持久目录；写临时文件是为了喂给 saveImageToPhotosAlbum，
    // 它只认文件路径，不认 base64。
    const filePath = `${wx.env.USER_DATA_PATH}/task-share-${taskId.value}.png`
    stage = "write"
    await withTimeout(writeBase64File(filePath, base64), 8000, "写入图片失败")
    uni.hideLoading()
    stage = "save"
    // 相册授权是系统弹窗：用户不理它会永远不回调，按钮就一直「生成中…」。
    await withTimeout(saveImageToAlbum(filePath), 15000, "保存到相册超时")
    uni.showToast({ title: "已保存，可转发班级群", icon: "none" })
  } catch (e) {
    if (stage === "generate") {
      uni.showToast({ title: e?.message || "生成小程序码失败", icon: "none" })
      return
    }
    const errMsg = e?.errMsg || e?.message || ""
    if (/timeout|超时/i.test(errMsg)) {
      uni.showToast({ title: "保存超时，请重试", icon: "none" })
    } else if (/auth|deny/i.test(errMsg)) {
      uni.showToast({ title: "请允许保存到相册后重试", icon: "none" })
    } else {
      uni.showToast({ title: "保存到相册失败", icon: "none" })
    }
  } finally {
    uni.hideLoading()
    shareBusy.value = false
  }
}

function withTimeout(promise, ms, message) {
  return Promise.race([
    promise,
    new Promise((_, reject) => setTimeout(() => reject(new Error(message)), ms))
  ])
}

function writeBase64File(filePath, base64) {
  return new Promise((resolve, reject) => {
    wx.getFileSystemManager().writeFile({
      filePath,
      data: base64,
      encoding: "base64",
      success: resolve,
      fail: reject
    })
  })
}

function saveImageToAlbum(filePath) {
  return new Promise((resolve, reject) => {
    uni.saveImageToPhotosAlbum({
      filePath,
      success: resolve,
      fail: reject
    })
  })
}

function noop() {}

onLoad((opts) => {
  taskId.value = Number(opts?.id || 0)
  today.value = todayKey()
  if (!taskId.value) {
    // 缺 id 的深链不能一直停在骨架屏；直接落到错误态并提供重试（重试会重新读取即失败）。
    loadError.value = "回应加载失败"
    return
  }
  loadTask()
  loadStats()
  loadCheckins()
  loadClassInfo()
})

onUnload(() => {
  // 先置 pageGone：停止录音会触发 onStop，那里看到 pageGone 就不再上传。
  pageGone = true
  stopRecordTimer()
  if (recording.value) recorder.stop()
  if (audio) {
    audio.stop()
    audio.destroy()
    audio = null
  }
})

onPullDownRefresh(() => {
  // 下拉会重排列表，录音行可能不在原位；先取消，避免 15s 后自动发出。
  cancelRecordingIfAny("已取消录音")
  today.value = todayKey()
  Promise.all([loadTask(), loadStats(), loadCheckins()]).finally(() =>
    uni.stopPullDownRefresh()
  )
})
</script>

<template>
  <PageShell>
    <view class="page">
      <!-- 加载：任务 3 行 + 四格统计 + 3 张回应卡骨架 -->
      <view v-if="isInitialLoading" class="skeleton-view">
        <view class="sk-hero">
          <view class="sk sk-chip" />
          <view class="sk sk-title" />
          <view class="sk sk-meta" />
          <view class="sk sk-line" />
          <view class="sk sk-line sk-short" />
        </view>
        <view class="sk-stats">
          <view v-for="i in 4" :key="i" class="sk sk-stat" />
        </view>
        <view v-for="i in 3" :key="i" class="sk-card">
          <view class="sk-head">
            <view class="sk sk-avatar" />
            <view class="sk sk-who" />
          </view>
          <view class="sk sk-line" />
          <view class="sk sk-line sk-short" />
          <view class="sk sk-action" />
        </view>
      </view>

      <!-- 失败：提交列表是这一页的主体，首次失败时整块替换，避免把「没有数据」当成结论 -->
      <view v-else-if="hardError" class="state-panel">
        <view class="state-icon is-danger"><text class="icon">&#xECA1;</text></view>
        <text class="state-title">回应加载失败</text>
        <text class="state-desc">网络异常，未获取到提交列表</text>
        <view class="state-actions">
          <view class="btn-primary" @click="retryAll">重试</view>
        </view>
      </view>

      <template v-else>
        <!-- 任务信息：只负责定场景 -->
        <view v-if="task.id" class="hero">
          <view class="hero-chips">
            <text class="chip" :class="taskActive ? 'chip-live' : 'chip-ended'">
              {{ taskActive ? "进行中" : "已结束" }}
            </text>
            <text class="chip chip-freq">{{ freqLabel(task.frequency) }}</text>
          </view>
          <text class="hero-title">{{ task.title }}</text>
          <view class="hero-meta">
            <text>{{ className || "本班" }}</text>
            <text class="meta-dot" />
            <text>{{ taskRange }}</text>
            <text class="meta-dot" />
            <text>{{ taskScopeLabel }}</text>
          </view>
          <view v-if="task.learning_link" class="hero-learning">
            <text class="hero-learning-label">学习关联</text>
            <text class="hero-learning-text">{{ task.learning_link }}</text>
          </view>
          <text v-if="task.description" class="hero-desc">{{ task.description }}</text>
          <view v-if="showStarStrip" class="star-strip">
            <text class="star-glyph icon">&#xF186;</text>
            <view class="star-copy">
              <text class="star-strong">本任务已集 {{ starCount }} 星</text>
              <text class="star-sub">{{ starStripSub }}</text>
            </view>
          </view>
        </view>
        <view v-else-if="taskError" class="error-strip" @click="loadTask">
          任务详情加载失败，点击重试；提交列表仍可查看
        </view>

        <!-- 统计 + 分段 + 审核筛选合并成一张卡：计数只保留一处 -->
        <view class="board">
          <view class="board-sum">
            <text><text class="board-num">{{ totalDisplay }}</text> 应交</text>
            <text class="meta-dot" />
            <text><text class="board-num">{{ submittedDisplay }}</text> 已提交</text>
            <text v-if="statsError" class="board-retry" @click="loadStats">
              提交数加载失败，点击重试
            </text>
          </view>
          <view class="seg">
            <view
              class="seg-btn"
              :class="{ 'is-active': status === 'pending' }"
              @click="switchStatus('pending')"
            >
              <text>待回应</text>
              <text class="seg-count">{{ pendingCount }}</text>
            </view>
            <view
              class="seg-btn"
              :class="{ 'is-active': status === 'responded' }"
              @click="switchStatus('responded')"
            >
              <text>已回应</text>
              <text class="seg-count">{{ respondedCount }}</text>
            </view>
          </view>
          <view v-if="!isEmpty" class="review-filter">
            <view
              v-for="item in REVIEW_FILTERS"
              :key="item.key"
              class="filter-chip"
              :class="{ 'is-active': reviewFilter === item.key }"
              @click="reviewFilter = item.key"
            >
              <text>{{ item.label }}</text>
              <text class="chip-count">{{ reviewCounts[item.key] }}</text>
            </view>
          </view>
        </view>

        <!-- 批量条：只对当前周期的待回应生效 -->
        <view v-if="showBatch" class="batch-strip">
          <view
            v-if="canBatch"
            class="batch-btn"
            :class="{ 'is-busy': batchBusy }"
            @click="batchRespond"
          >
            <text class="icon">&#xEB89;</text>
            <text>{{ batchBusy ? "汇总中…" : batchLabel }}</text>
          </view>
          <view v-else-if="task.id" class="batch-hint-box">一次性任务与个体任务需逐条回应</view>
          <text class="batch-hint">「加星」才计入星星罐；贴纸回应不计星</text>
        </view>

        <!-- 空：没有家长提交 -->
        <view v-if="isEmpty" class="state-panel">
          <view class="state-icon is-warm"><text class="icon">&#xEB51;</text></view>
          <text class="state-title">还没有家长提交回应</text>
          <text class="state-desc">把任务发给家长，提交后会出现在这里。</text>
          <view class="state-actions">
            <view class="btn-primary" @click="shareTaskCard">生成小程序码</view>
            <view class="btn-secondary" @click="refreshCheckins">刷新</view>
          </view>
        </view>

        <template v-else>
          <view v-if="loadError" class="error-strip" @click="retryAll">
            {{ loadError }} · 显示的是上次拉取的结果，点击重试
          </view>

          <view class="list">
            <view v-for="row in filteredRows" :key="row.id" class="checkin">
              <view class="checkin-head">
                <view class="avatar">
                  <text>{{ initialOf(row.student?.name) }}</text>
                </view>
                <view class="who">
                  <view class="who-line">
                    <text class="who-name">{{ row.student?.name || `学生 ${row.student_id}` }}</text>
                    <text class="tag-submit">家长提交</text>
                  </view>
                  <text class="when">{{ submitTime(row) }}</text>
                </view>
              </view>

              <text
                v-if="status === 'pending' && !isCurrentPeriod(row)"
                class="period-warning"
              >旧周期 · 请逐条回应</text>

              <text class="checkin-text" :class="{ 'is-empty': !row.content }">
                {{ row.content || "（一键完成）" }}
              </text>

              <view v-if="row.media_url" class="media-grid">
                <image
                  class="media-tile"
                  :src="row.media_url"
                  mode="aspectFill"
                  @click="previewMedia(row)"
                />
              </view>

              <!-- 公开审核四态：pending / approved / revoked；none 不渲染 -->
              <view v-if="reviewKey(row) !== 'none'" class="review-block">
                <view v-if="reviewKey(row) === 'pending'" class="rv rv-pending">
                  <view class="rv-head">
                    <text class="badge badge-warning">待审核</text>
                    <text class="rv-state">家长申请公开，待你审核</text>
                  </view>
                  <view
                    v-if="canApprovePublic"
                    class="scope-check"
                    @click="toggleScopeConfirm(row.id)"
                  >
                    <view class="scope-box" :class="{ 'is-on': scopeConfirmed[row.id] }">
                      <text v-if="scopeConfirmed[row.id]" class="icon scope-tick">&#xEB7B;</text>
                    </view>
                    <text class="scope-text">我确认内容中只有提交者孩子</text>
                  </view>
                  <text v-else class="rv-blocked">
                    {{ task.id ? "任务已关闭公开，不能批准；可撤回申请" : "任务详情未加载，暂时无法批准公开；可先撤回" }}
                  </text>
                  <view class="rv-actions" :class="{ 'is-busy': isReviewing(row.id) }">
                    <view
                      v-if="canApprovePublic"
                      class="rv-btn approve"
                      :class="{ 'is-disabled': !scopeConfirmed[row.id] }"
                      @click="reviewPublic(row, true)"
                    >
                      <text v-if="isReviewing(row.id)" class="spinner" />
                      <text>{{ isReviewing(row.id) ? "批准中…" : "批准公开" }}</text>
                    </view>
                    <template v-if="reviewConfirm.id !== row.id">
                      <view class="rv-btn reject" @click="askReview(row, 'reject')">拒绝</view>
                    </template>
                    <template v-else>
                      <view class="rv-btn reject" @click="reviewPublic(row, false)">
                        <text v-if="isReviewing(row.id)" class="spinner" />
                        <text>{{ isReviewing(row.id) ? "处理中…" : "确认拒绝？" }}</text>
                      </view>
                      <view class="rv-btn plain" @click="cancelReviewConfirm">取消</view>
                    </template>
                  </view>
                </view>

                <view v-else-if="reviewKey(row) === 'approved'" class="rv rv-approved">
                  <view class="rv-head">
                    <text class="badge badge-success">
                      {{ row.public_visible ? "已公开" : "已通过" }}
                    </text>
                    <text class="rv-state">{{ publicStateText(row) }}</text>
                  </view>
                  <text v-if="publicStateHint(row)" class="rv-hint">{{ publicStateHint(row) }}</text>
                  <view class="rv-actions is-end" :class="{ 'is-busy': isReviewing(row.id) }">
                    <template v-if="reviewConfirm.id !== row.id">
                      <view class="rv-btn ghost-danger" @click="askReview(row, 'revoke')">撤回公开</view>
                    </template>
                    <template v-else>
                      <view class="rv-btn reject" @click="reviewPublic(row, false)">
                        <text v-if="isReviewing(row.id)" class="spinner" />
                        <text>{{ isReviewing(row.id) ? "处理中…" : "确认撤回？" }}</text>
                      </view>
                      <view class="rv-btn plain" @click="cancelReviewConfirm">取消</view>
                    </template>
                  </view>
                </view>

                <view v-else-if="reviewKey(row) === 'revoked'" class="rv rv-revoked">
                  <view class="rv-head">
                    <text class="badge">未公开</text>
                    <text class="rv-state">未公开</text>
                  </view>
                </view>
              </view>

              <!-- 待回应：单条贴纸 / 加星 / 录音 -->
              <view
                v-if="status === 'pending'"
                class="respond-row"
                :class="{ 'is-busy': batchBusy || isResponding(row.id) }"
              >
                <view class="react" @click="respond(row, 'heart')">
                  <text class="icon react-icon">&#xEE0E;</text>
                  <text>爱心</text>
                </view>
                <view class="react" @click="respond(row, 'thumb')">
                  <text class="icon react-icon">&#xF206;</text>
                  <text>点赞</text>
                </view>
                <view v-if="starTracking(task)" class="react react-text" @click="respondStar(row)">
                  加星
                </view>
                <view
                  v-if="!(recording && recordingRowId === row.id)"
                  class="react react-text"
                  @click="startRecord(row, 'respond')"
                >
                  <text class="icon react-icon">&#xEF4F;</text>
                  <text>录音</text>
                </view>
                <view v-else class="react is-recording" @click="stopRecord">
                  <view class="rec-dot" />
                  <text>{{ recordLabel() }}</text>
                </view>
              </view>

              <!-- 已回应：老师回应 + 补充回应 -->
              <view v-if="status === 'responded' && row.response" class="response-block">
                <view class="response-head">
                  <text class="response-label">你的回应</text>
                  <text class="response-time">{{ formatCommentTime(row.response.created_at) }}</text>
                </view>
                <view v-if="row.response.ritual === 'sticker'" class="response-main">
                  <text class="response-sticker icon">{{ stickerGlyph(row.response.sticker_key) }}</text>
                  <text class="response-text">贴纸回应</text>
                </view>
                <view
                  v-else-if="row.response.ritual === 'voice'"
                  class="voice-play"
                  @click="playAudio(row.response.media_url)"
                >
                  <text class="icon voice-icon">&#xF00A;</text>
                  <text class="voice-label">点击播放语音</text>
                </view>
                <view v-else class="response-main">
                  <text class="response-sticker icon">&#xF186;</text>
                  <text class="response-text">班级星星</text>
                </view>
                <text v-if="row.response.ritual === 'class_star'" class="response-note">
                  已计入班级星星罐
                </text>
                <text v-if="row.response.content" class="response-content">
                  {{ row.response.content }}
                </text>

                <view v-if="row.supplements && row.supplements.length" class="supplement-line">
                  <text class="supplement-count">补充回应 {{ row.supplements.length }} 条</text>
                  <text class="supplement-preview">
                    <text v-if="supplementGlyph(row)" class="icon">{{ supplementGlyph(row) }}</text>
                    <text>{{ supplementLabel(row) }}</text>
                  </text>
                </view>

                <view class="supplement-toggle" @click="toggleSupplement(row)">
                  <text>补充回应（个别夸一句）</text>
                  <text class="icon chev" :class="{ 'is-open': supplementFor === row.id }">&#xEA4E;</text>
                </view>

                <view v-if="supplementFor === row.id" class="supplement-panel" :class="{ 'is-busy': supplementBusy }">
                  <!-- 历史补充展开后可回放语音；收起态只显示预览，避免长列表被撑开。 -->
                  <view v-if="row.supplements && row.supplements.length" class="supplement-history">
                    <view v-for="sp in row.supplements" :key="sp.id" class="supplement-item">
                      <text v-if="sp.ritual === 'sticker'" class="icon supplement-item-glyph">{{ stickerGlyph(sp.sticker_key) }}</text>
                      <view v-else class="supplement-play" @click="playAudio(sp.media_url)">
                        <text class="icon">&#xF00A;</text>
                        <text>播放语音补充</text>
                      </view>
                      <text v-if="sp.content" class="supplement-item-text">{{ sp.content }}</text>
                    </view>
                  </view>
                  <view class="supplement-stickers">
                    <view
                      v-for="sticker in SUPPLEMENT_STICKERS"
                      :key="sticker"
                      class="react"
                      @click="supplement(row, sticker)"
                    >
                      <text class="icon react-icon">{{ stickerGlyph(sticker) }}</text>
                    </view>
                  </view>
                  <view
                    class="btn-soft"
                    @click="recording && recordingRowId === row.id ? stopRecord() : startRecord(row, 'supplement')"
                  >
                    <text v-if="recording && recordingRowId === row.id" class="rec-dot" />
                    <text>{{ recording && recordingRowId === row.id ? recordLabel() : "录语音补充" }}</text>
                  </view>
                  <text class="supplement-hint">补充回应不发通知，家长在任务详情里看到；不发星星</text>
                  <view class="text-link" @click="toggleSupplement(row)">收起</view>
                </view>
              </view>
            </view>
          </view>

          <!-- 当前筛选无结果：给出另一段的条数与跳转 -->
          <view v-if="!isEmpty && filteredRows.length === 0" class="filter-empty">
            <text>当前筛选下没有记录</text>
            <view v-if="filterJumpText" class="text-link" @click="switchStatus(status === 'pending' ? 'responded' : 'pending')">
              {{ filterJumpText }}
            </view>
          </view>
        </template>

        <!-- 任务设置：编辑与分享下沉到列表末尾，避免抢审核主线的注意力 -->
        <view v-if="showSettings" class="settings">
          <text class="settings-title">任务设置</text>
          <view class="settings-cell" @click="openEdit">
            <text class="settings-label">编辑任务</text>
            <text class="settings-value">标题、说明、公开</text>
            <text class="settings-chev icon">&#xEA6E;</text>
          </view>
          <view class="settings-cell" @click="shareTaskCard">
            <text class="settings-label">生成小程序码</text>
            <text class="settings-value">{{ shareBusy ? "生成中…" : "转发班级群" }}</text>
            <text class="settings-chev icon">&#xEA6E;</text>
          </view>
        </view>
      </template>
    </view>

    <!-- 编辑任务：结构只读，公开开关只在保存时生效 -->
    <view v-if="editOpen" class="sheet-layer">
      <view class="sheet-mask" @click="closeEdit" @touchmove.stop.prevent="noop" />
      <view class="sheet is-tall">
        <view class="sheet-grab" />
        <view class="sheet-head">
          <view class="sheet-head-main">
            <text class="sheet-title">编辑任务</text>
            <text class="sheet-sub">
              {{ structureLocked ? "结构已锁定，可改标题、说明与公开设置" : "结构只读，可改标题、说明与公开设置" }}
            </text>
          </view>
          <view class="sheet-close" @click="closeEdit"><text class="icon">&#xEB99;</text></view>
        </view>

        <scroll-view scroll-y class="sheet-body">
          <view class="field">
            <text class="field-label">任务标题</text>
            <input
              v-model="edit.title"
              class="text-input"
              type="text"
              maxlength="30"
              placeholder="任务标题"
            />
          </view>
          <view class="field">
            <text class="field-label">任务说明</text>
            <textarea
              v-model="edit.description"
              class="text-area"
              maxlength="200"
              placeholder="任务说明"
            />
          </view>
          <view class="field">
            <text class="field-label">学习关联</text>
            <input
              v-model="edit.learning_link"
              class="text-input"
              type="text"
              maxlength="80"
              placeholder="和幼儿园学习的关系"
            />
          </view>

          <view v-if="Number(task.target_student_id) > 0" class="switch-row">
            <text class="switch-label">指派给个别学生的任务不可公开</text>
          </view>
          <template v-else>
            <view class="switch-row">
              <view class="switch-copy">
                <text class="switch-label">允许家长申请公开</text>
                <text class="field-hint">关闭后，已公开内容暂不对本班展示</text>
              </view>
              <switch :checked="edit.allow_public" @change="onPublicToggle" />
            </view>
            <view v-if="publicOffPending" class="danger-confirm">
              <text class="danger-title">关闭公开后，已公开内容将全部撤下，家长需要重新申请</text>
              <view class="danger-actions">
                <view class="rv-btn reject" @click="confirmPublicOff">确认关闭</view>
                <view class="rv-btn plain" @click="cancelPublicOff">取消</view>
              </view>
            </view>
          </template>

          <view class="struct">
            <view class="struct-head">
              <text class="struct-title">任务结构</text>
              <text class="lock-chip">
                <text class="icon">&#xEECE;</text>
                <text>{{ structureLocked ? "已锁定" : "只读" }}</text>
              </text>
            </view>
            <view class="struct-row">
              <text class="struct-label">频率</text>
              <text class="struct-value">{{ freqLabel(task.frequency) }}</text>
            </view>
            <view class="struct-row">
              <text class="struct-label">日期</text>
              <text class="struct-value">{{ taskRange }}</text>
            </view>
            <view class="struct-row">
              <text class="struct-label">范围</text>
              <text class="struct-value">{{ taskScopeLabel }}</text>
            </view>
            <view class="struct-row">
              <text class="struct-label">仪式</text>
              <text class="struct-value">{{ ritualLabel(task.recognition_ritual) }}</text>
            </view>
            <view class="struct-row">
              <text class="struct-label">星星</text>
              <text class="struct-value">{{ structureStarText }}</text>
            </view>
            <text class="struct-hint">{{ structureHint }}</text>
          </view>
        </scroll-view>

        <view class="sheet-footer">
          <view class="footer-btn">
            <view class="btn-cancel" @click="closeEdit">取消</view>
          </view>
          <view class="footer-btn">
            <view class="btn-create" :class="{ 'is-busy': savingEdit }" @click="saveEdit">
              {{ savingEdit ? "保存中…" : "保存" }}
            </view>
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

.page {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  padding-bottom: $spacing-xl;
}

/* ---------- 任务卡 ---------- */
.hero {
  padding: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.hero-chips {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.chip {
  display: inline-flex;
  align-items: center;
  min-height: 44rpx;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.chip-live {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.chip-ended {
  background: $color-bg-page;
  color: $color-text-muted;
}

.chip-freq {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.hero-title {
  display: block;
  margin-top: 20rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-bold;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.hero-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-sm;
  margin-top: 12rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
}

.meta-dot {
  width: 6rpx;
  height: 6rpx;
  flex: none;
  border-radius: $border-radius-full;
  background: $color-text-hint;
}

.hero-learning {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs $spacing-md;
  margin-top: 20rpx;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.55;
}

.hero-learning-label {
  flex: none;
  align-self: flex-start;
  padding: 4rpx 12rpx;
  border-radius: $border-radius-sm;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
}

.hero-learning-text {
  min-width: 0;
  overflow-wrap: anywhere;
}

.hero-desc {
  display: block;
  margin-top: 16rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.star-strip {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: 20rpx;
  padding: $spacing-md 20rpx;
  border-radius: $border-radius-lg;
  background: $color-brand-primary-bg;
}

.star-glyph {
  flex: none;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-lg;
}

.star-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2rpx;
}

.star-strong {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

.star-sub {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

/* ---------- 统计卡 ---------- */
.board {
  padding: 20rpx $spacing-lg $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.board-sum {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.board-num {
  margin-right: 2rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-bold;
}

.board-retry {
  margin-left: auto;
  color: $color-semantic-danger;
  line-height: 1.4;
}

.seg {
  display: flex;
  gap: $spacing-xs;
  margin-top: 16rpx;
  padding: 6rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.seg-btn {
  flex: 1;
  min-width: 0;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  border-radius: $border-radius-full;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.seg-btn.is-active {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.seg-count {
  min-width: 40rpx;
  height: 36rpx;
  padding: 0 12rpx;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.seg-btn.is-active .seg-count {
  color: $color-brand-primary-dark;
}

.review-filter {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
  margin-top: 20rpx;
}

.filter-chip {
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  display: inline-flex;
  align-items: center;
  gap: $spacing-sm;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1;
}

.filter-chip.is-active {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

.chip-count {
  font-size: $typography-font-size-xs;
  color: inherit;
}

/* ---------- 批量条 ---------- */
.batch-strip {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
}

.batch-btn {
  min-height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  padding: 0 $spacing-lg;
  border-radius: $component-card-radius;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.batch-btn.is-busy {
  opacity: 0.7;
}

.batch-hint-box {
  padding: $spacing-md;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  text-align: center;
}

.batch-hint {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  text-align: center;
}

/* ---------- 提交卡 ---------- */
.list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.checkin {
  padding: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.checkin-head {
  display: flex;
  align-items: center;
  gap: 20rpx;
}

.avatar {
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $gradient-brand-soft;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.who {
  min-width: 0;
}

.who-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.who-name {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.tag-submit {
  padding: 2rpx 12rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

.when {
  display: block;
  margin-top: 4rpx;
  color: $color-text-hint;
  font-size: $typography-font-size-xs;
}

.checkin-text {
  display: block;
  margin-top: 20rpx;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.65;
  overflow-wrap: anywhere;
}

.checkin-text.is-empty {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
}

.period-warning {
  display: inline-flex;
  margin-top: 16rpx;
  padding: $spacing-xs $spacing-md;
  border-radius: $border-radius-lg;
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.media-grid {
  display: flex;
  margin-top: 20rpx;
}

.media-tile {
  width: 200rpx;
  height: 200rpx;
  border-radius: $component-card-radius;
  background: $color-bg-page;
}

/* ---------- 审核块 ---------- */
.review-block {
  margin-top: $spacing-lg;
}

.rv {
  padding: 20rpx $spacing-lg;
  border-radius: $component-card-radius;
}

.rv-pending {
  background: $color-semantic-warning-bg;
}

.rv-approved {
  background: $color-semantic-success-bg;
}

.rv-revoked {
  background: $color-bg-page;
}

.rv-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.badge {
  display: inline-flex;
  align-items: center;
  min-height: 40rpx;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.badge-warning {
  color: $color-semantic-warning-strong;
}

.badge-success {
  color: $color-semantic-success-strong;
}

.rv-state {
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.rv-hint,
.rv-blocked {
  display: block;
  margin-top: 12rpx;
  color: $color-semantic-warning-strong;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.scope-check {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-height: 88rpx;
  margin-top: 8rpx;
}

.scope-box {
  width: 36rpx;
  height: 36rpx;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: 10rpx;
  background: $color-bg-card;
}

.scope-box.is-on {
  border-color: $color-brand-primary;
  background: $color-brand-primary;
}

.scope-tick {
  color: $color-text-on-brand;
  font-size: $typography-font-size-xs;
}

.scope-text {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.45;
}

.rv-actions {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;
  margin-top: $spacing-md;
}

.rv-actions.is-end {
  justify-content: flex-end;
}

.rv-actions.is-busy {
  opacity: 0.6;
}

.rv-btn {
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  border-radius: $border-radius-lg;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
}

.rv-btn.approve {
  flex: 1;
  background: $color-semantic-success;
  color: $color-text-on-brand;
}

.rv-btn.approve.is-disabled {
  background: $color-bg-page;
  color: $color-text-hint;
}

.rv-btn.reject {
  flex: 1;
  border: 1rpx solid $color-semantic-danger;
  background: $color-bg-card;
  color: $color-semantic-danger;
}

.rv-btn.ghost-danger {
  border: 1rpx solid $color-semantic-danger;
  background: $color-bg-card;
  color: $color-semantic-danger;
}

.rv-btn.plain {
  flex: 1;
  background: $color-bg-card;
  color: $color-text-muted;
}

.spinner {
  width: 24rpx;
  height: 24rpx;
  flex: none;
  border: 4rpx solid currentColor;
  border-right-color: transparent;
  border-radius: $border-radius-full;
  animation: spin 700ms linear infinite;
}

/* ---------- 回应操作 ---------- */
.respond-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-lg;
}

.respond-row.is-busy {
  opacity: 0.6;
}

.react {
  min-width: 88rpx;
  min-height: 88rpx;
  padding: 0 $spacing-md;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1;
}

.react-icon {
  font-size: $typography-font-size-lg;
}

.react-text {
  padding: 0 $spacing-lg;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.react.is-recording {
  padding: 0 $spacing-lg;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

.rec-dot {
  width: 12rpx;
  height: 12rpx;
  flex: none;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
  animation: rec-pulse 1s $animation-easing-default infinite;
}

/* ---------- 已回应 ---------- */
.response-block {
  margin-top: $spacing-lg;
  padding: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-page;
}

.response-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}

.response-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.response-time {
  color: $color-text-hint;
  font-size: $typography-font-size-xs;
}

.response-main {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: 12rpx;
}

.response-sticker {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-2xl;
}

.response-text {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.response-note {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.response-content {
  display: block;
  margin-top: 8rpx;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.voice-play {
  min-height: 88rpx;
  margin-top: 16rpx;
  padding: 0 $spacing-lg;
  display: flex;
  align-items: center;
  gap: $spacing-md;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-base;
}

.voice-icon {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
}

.voice-label {
  color: $color-text-body;
}

.supplement-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-sm;
  margin-top: 20rpx;
}

.supplement-count {
  padding: 4rpx 14rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
}

.supplement-preview {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.supplement-toggle {
  min-height: 88rpx;
  margin-top: 16rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  border-top: 1rpx solid $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

.chev {
  flex: none;
  font-size: $typography-font-size-base;
  transition: transform $animation-duration-fast $animation-easing-default;
}

.chev.is-open {
  transform: rotate(180deg);
}

.supplement-panel {
  margin-top: 16rpx;
  padding: 20rpx;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
}

.supplement-panel.is-busy {
  opacity: 0.6;
}

.supplement-history {
  padding-bottom: $spacing-md;
  margin-bottom: $spacing-md;
  border-bottom: 1rpx solid $color-bg-page;
}

.supplement-item {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-sm;
  margin-top: $spacing-xs;
}

.supplement-item-glyph {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-lg;
}

.supplement-play {
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  min-height: 64rpx;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

.supplement-item-text {
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.supplement-stickers {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.btn-soft {
  min-height: 88rpx;
  margin-top: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  border-radius: $border-radius-lg;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.supplement-hint {
  display: block;
  margin-top: 12rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.55;
}

.text-link {
  min-height: 88rpx;
  display: inline-flex;
  align-items: center;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

/* ---------- 筛选空 / 任务设置 ---------- */
.filter-empty {
  padding: $spacing-xl $spacing-lg;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-xs;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.settings {
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  overflow: hidden;
}

.settings-title {
  display: block;
  padding: $spacing-md $spacing-lg 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.settings-cell {
  min-height: 104rpx;
  padding: $spacing-md $spacing-lg;
  display: flex;
  align-items: center;
  gap: 20rpx;
  border-top: 1rpx solid $color-bg-page;
}

.settings-label {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
}

.settings-value {
  margin-left: auto;
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  text-align: right;
}

.settings-chev {
  flex: none;
  color: $color-text-hint;
  font-size: $typography-font-size-md;
}

/* ---------- 状态面板 ---------- */
.state-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 68rpx $spacing-lg 60rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.state-icon {
  width: 112rpx;
  height: 112rpx;
  margin-bottom: $spacing-lg;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-3xl;
}

.state-icon.is-warm {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.state-icon.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.state-desc {
  max-width: 520rpx;
  margin-top: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.state-actions {
  width: 100%;
  margin-top: $spacing-xl;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.btn-primary,
.btn-secondary {
  min-height: 88rpx;
  padding: 0 $spacing-xl;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $component-card-radius;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.btn-primary {
  background: $gradient-brand;
  color: $color-text-primary;
  box-shadow: $shadow-sm;
}

.btn-secondary {
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-card;
  color: $color-text-body;
  font-weight: $typography-font-weight-medium;
}

.error-strip {
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger-strong;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

/* ---------- 骨架 ---------- */
.skeleton-view {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.sk-hero,
.sk-card {
  padding: $spacing-lg;
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sk-head {
  display: flex;
  align-items: center;
  gap: 20rpx;
}

.sk-stats {
  display: flex;
  gap: $spacing-sm;
}

.sk-stat {
  flex: 1;
  height: 116rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

.sk {
  display: block;
  height: 24rpx;
  border-radius: $border-radius-md;
  background: $color-bg-page;
  animation: sk-pulse 1.4s $animation-easing-default infinite;
}

.sk-chip {
  width: 112rpx;
  height: 40rpx;
  border-radius: $border-radius-full;
}

.sk-title {
  width: 78%;
  height: 44rpx;
}

.sk-meta {
  width: 55%;
  height: 22rpx;
}

.sk-line {
  width: 100%;
}

.sk-short {
  width: 68%;
}

.sk-avatar {
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
}

.sk-who {
  width: 45%;
  height: 48rpx;
}

.sk-action {
  width: 100%;
  height: 80rpx;
  border-radius: $border-radius-lg;
}

/* ---------- 底部弹层（编辑） ---------- */
.sheet-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 100;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.sheet-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: $color-text-primary;
  opacity: 0.44;
}

.sheet {
  position: relative;
  z-index: 1;
  width: 100%;
  max-height: 84vh;
  display: flex;
  flex-direction: column;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.sheet.is-tall {
  max-height: 88vh;
}

.sheet-grab {
  flex: none;
  width: 72rpx;
  height: 8rpx;
  margin: $spacing-md auto 0;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
  opacity: 0.7;
}

.sheet-head {
  flex: none;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: $spacing-md;
  padding: $spacing-md $spacing-xl $spacing-sm;
}

.sheet-head-main {
  min-width: 0;
}

.sheet-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.sheet-sub {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
}

.sheet-close {
  flex: none;
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: $color-text-body;
  font-size: $typography-font-size-lg;
}

.sheet-body {
  flex: 1 1 auto;
  min-height: 0;
  max-height: 62vh;
  padding: 0 $spacing-xl $spacing-md;
  box-sizing: border-box;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
  margin-bottom: $spacing-lg;
}

.field-label {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.field-hint {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
}

.text-input {
  box-sizing: border-box;
  width: 100%;
  height: 88rpx;
  padding: 0 $spacing-md;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
}

.text-area {
  box-sizing: border-box;
  width: 100%;
  min-height: 168rpx;
  padding: $spacing-md;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.5;
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  min-height: 88rpx;
  margin-top: $spacing-md;
  padding: 0 2rpx;
}

.switch-copy {
  flex: 1;
  min-width: 0;
}

.switch-label {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

.danger-confirm {
  margin-top: $spacing-md;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
}

.danger-title {
  display: block;
  color: $color-semantic-danger-strong;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.danger-actions {
  display: flex;
  gap: $spacing-md;
  margin-top: $spacing-md;
}

.struct {
  margin-top: $spacing-lg;
  padding: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-page;
}

.struct-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}

.struct-title {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.lock-chip {
  display: inline-flex;
  align-items: center;
  gap: 6rpx;
  padding: 4rpx 14rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.struct-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: $spacing-lg;
  padding: 14rpx 0;
  border-top: 1rpx solid $color-bg-card;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.struct-label {
  flex: none;
  color: $color-text-muted;
}

.struct-value {
  min-width: 0;
  color: $color-text-body;
  text-align: right;
  overflow-wrap: anywhere;
}

.struct-hint {
  display: block;
  margin-top: 12rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.55;
}

.sheet-footer {
  flex: none;
  display: flex;
  gap: $spacing-md;
  padding: $spacing-md $spacing-xl calc(#{$spacing-xl} + env(safe-area-inset-bottom));
  border-top: 1rpx solid $color-border-subtle;
}

.footer-btn {
  flex: 1;
  min-width: 0;
}

.btn-cancel {
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.btn-create {
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.btn-create.is-busy {
  opacity: 0.7;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes rec-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.35;
  }
}

@keyframes sk-pulse {
  0%,
  100% {
    opacity: 0.55;
  }
  50% {
    opacity: 1;
  }
}
</style>
