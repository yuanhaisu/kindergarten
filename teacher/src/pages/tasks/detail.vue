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
import { fetchTeacherClassStudents } from "@/api/teacher-class"
import config from "@/config/env"
import { getToken } from "@/utils/auth"

const FREQ_OPTIONS = ["一次性", "每日", "每周"]
const FREQ_VALUES = ["once", "daily", "weekly"]
const RITUAL_OPTIONS = ["贴纸", "语音", "班级星星"]
const RITUAL_VALUES = ["sticker", "voice", "class_star"]
const FREQ_LABELS = { once: "一次性", daily: "每日", weekly: "每周" }
const RITUAL_LABELS = { sticker: "贴纸", voice: "语音", class_star: "班级星星" }
// sticker_key 是后端透传的英文 key，客户端只做展示映射（与家长端一致）。
const STICKER_GLYPHS = { star: "⭐", heart: "❤️", thumb: "👍", flower: "🌸", smile: "😊", clap: "👏" }
const STICKERS = ["star", "heart", "thumb"]
const SUPPLEMENT_STICKERS = ["flower", "heart", "star", "thumb"]

const taskId = ref(0)
const rows = ref([])
const task = ref({})
const taskError = ref("")
const stats = ref({})
const statsError = ref("")
const statsLoaded = ref(false)
// 与后端 assertTaskEditable 对齐：有提交后只允许 title/description/learning_link/allow_public。
// 提交数没拿到时同样按锁定处理——不能让老师在「不知道有没有提交」的状态下改结构再吃 422。
const structureLocked = computed(() => Number(stats.value.submission_count || 0) > 0)
const structureReadOnly = computed(() => !statsLoaded.value || structureLocked.value || !!statsError.value)

// 与后端 batchRespondAllowed 对齐：只有每日/每周的班级任务提供汇总入口。
// 任务详情没加载成功时（task.id 为空）不下结论，避免把加载失败误报成「需逐条回应」。
const canBatch = computed(() =>
  !!task.value.id &&
  task.value.target_student_id === 0 &&
  ["daily", "weekly"].includes(task.value.frequency)
)
// 「加星」的可用性看星星罐是否真的存在，而不是看任务默认仪式：
// D5 允许老师逐条调整仪式，task 罐的目标 >0 就说明星星有意义；
// 只有发⭐贴纸不进罐（D6：只有 ritual=class_star 的主回应计数）。
// 班级罐按学期独立配置（老师端与班级管理页都能设），任务 StarGoal 被忽略，所以按 scope 放行。
const starTracking = computed(() =>
  task.value.star_scope === "class" ? true : Number(task.value.star_goal) > 0
)
// 批准公开的前置条件（契约）：任务开启公开且不是个体任务。孩子是否仍在班由后端判定。
// 不满足时不给批准按钮，避免老师点了才吃 422；撤回永远保留。
const canApprovePublic = computed(() =>
  !!task.value.id && !!task.value.allow_public && !task.value.target_student_id
)
// 星星任务上不再把 ⭐ 当普通贴纸：两者同屏时老师会以为点⭐就是加星，
// 但 D6 只统计 ritual=class_star，⭐贴纸不进罐。
const responseStickers = computed(() =>
  starTracking.value ? STICKERS.filter((s) => s !== "star") : STICKERS
)
const batchIsStar = computed(() => task.value.recognition_ritual === "class_star")
// 汇总回应的载体跟着任务默认仪式：星星任务发 class_star，其余发 ⭐ 贴纸。
// 标签必须如实反映载体，避免老师以为「一键回应」发的是语音。
const batchLabel = computed(() => (batchIsStar.value ? "一键送星（本周期）" : "一键送贴纸（本周期）"))

// 旧周期（不在 period_start..period_end 内）的待回应只能逐条处理。
function isCurrentPeriod(row) {
  if (!task.value.period_start || !task.value.period_end || !row.date) return true
  return row.date >= task.value.period_start && row.date <= task.value.period_end
}

const edit = ref({
  title: "", description: "", learning_link: "", allow_public: false,
  frequency: "once", start_date: "", end_date: "", target_student_id: 0,
  recognition_ritual: "sticker", star_scope: "task", star_goal: 0
})
// 关闭公开是破坏性操作（会批量撤下已公开内容），用页面内两步确认；
// 原生 uni.showModal 不进视图层 DOM，CDP/自动化点不到，也无法验证。
const publicOffPending = ref(false)
// 改成个体任务会强制关闭公开并撤下已公开内容（后端 revokePublicForTask），
// 也是破坏性操作，同样用页面内两步确认，不能只在选择器里静默改掉。
const pendingIndividualId = ref(0)
const pendingIndividualName = ref("")
const students = ref([])
const studentsError = ref("")
const studentChoices = computed(() => {
  const opts = [{ id: 0, name: "全班（班级任务）" }]
  students.value.forEach((s) => opts.push({ id: s.student_id, name: s.name }))
  const tid = edit.value.target_student_id
  // 目标孩子转班/已不在在读名单时仍然要能显示「这是谁」，否则 picker 会错显成「全班」。
  if (tid && !opts.some((o) => o.id === tid)) {
    opts.push({ id: tid, name: `学生 #${tid}（不在本班在读名单）` })
  }
  return opts
})
const studentOptions = computed(() => studentChoices.value.map((o) => o.name))
const studentIndex = computed(() => {
  const i = studentChoices.value.findIndex((o) => o.id === (edit.value.target_student_id || 0))
  return i >= 0 ? i : 0
})
// -1 = 任务里是空/未知值。不能夹成第一项：那会在保存时把 "" 静默改成 once/sticker，
// 有提交的任务会因此吃结构锁 422（管理端/历史任务存在空仪式）。
const freqIndex = computed(() => FREQ_VALUES.indexOf(edit.value.frequency))
const ritualIndex = computed(() => RITUAL_VALUES.indexOf(edit.value.recognition_ritual))
// picker 的 value 不能是 -1；显示用 freqText/ritualText 单独处理。
const freqPickerValue = computed(() => Math.max(0, freqIndex.value))
const ritualPickerValue = computed(() => Math.max(0, ritualIndex.value))
const freqText = computed(() => (freqIndex.value >= 0 ? FREQ_OPTIONS[freqIndex.value] : "未设置"))
const ritualText = computed(() => (ritualIndex.value >= 0 ? RITUAL_OPTIONS[ritualIndex.value] : "未设置"))
const targetLabel = computed(() => studentChoices.value[studentIndex.value]?.name || "全班（班级任务）")
const starScopeLabel = computed(() => {
  // 按班级的目标来自 class_star_jars（老师/管理端都能设），不是管理端专属。
  if (edit.value.star_scope === "class") return "按班级（使用班级罐目标）"
  if (edit.value.star_scope === "task") return "按任务"
  return "未设置"
})

function freqLabel(v) {
  return FREQ_LABELS[v] || v || "未设置"
}
function ritualLabel(v) {
  return RITUAL_LABELS[v] || v || "未设置"
}
function stickerGlyph(key) {
  return STICKER_GLYPHS[key] || "🌟"
}

// D9：通知失败不回滚回应，但计数必须告诉老师，否则家长没收到也没人知道。
function notifySuffix(data) {
  const failed = Number(data?.notify_failed_count || 0)
  return failed > 0 ? `，${failed} 个家长未收到通知` : ""
}

const loading = ref(false)
// 防连点：第二次请求必然拿到 0 条，紧跟在「已回应 N 条」后面会被读成失败。
const batchBusy = ref(false)
// 单条回应的门闩。贴纸是并排的小按钮，连点两下会真发两次请求，
// 第二下必然落到 409，用户看到的是「回应失败」——明明第一次已经成功了。
// 闩在「行」上而不是全页开关：A 的请求还在飞时去回应 B 是正常操作，不该一起被挡住。
const respondingIds = ref([])
const isResponding = (id) => respondingIds.value.includes(id)
function addResponding(id) {
  if (!respondingIds.value.includes(id)) respondingIds.value = [...respondingIds.value, id]
}
function removeResponding(id) {
  respondingIds.value = respondingIds.value.filter((x) => x !== id)
}
const status = ref("pending")
const recorder = uni.getRecorderManager()
let recordingRow = null
const recording = ref(false)
const recordingRowId = ref(0)
const recordingMode = ref("")
// 小程序码生成/落盘期间锁按钮：微信生成接口不是幂等的瞬时操作，连点会排出一串请求。
const shareBusy = ref(false)
// 补充回应由 checkins 的 supplements 字段返回，成功刷新后直接渲染，不再靠会话内标记。
const supplementFor = ref(0)
const supplementBusy = ref(false)
// 公开审核：勾选确认、请求门闩、撤回的两步确认都按行保存。
const scopeConfirmed = ref({})
const reviewingIds = ref([])
const revokeConfirmId = ref(0)
const isReviewing = (id) => reviewingIds.value.includes(id)
let audio = null
let pageGone = false
// 切 tab/下拉刷新时取消录音：那一行的「停止」不再渲染，15s 后自动停止还会
// 把一段老师以为已取消的语音发出去。
let discardRecording = false

recorder.onStop((res) => {
  recording.value = false
  recordingRowId.value = 0
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
  uni.showToast({ title: "录音失败，请检查麦克风权限", icon: "none" })
})

// 上传期也要占住这一行：否则老师能在「上传语音」转圈时点⭐/一键汇总，
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
      // 上传在飞时离开页面：不再发回应，也不要在别的页面弹 toast。
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
        // 成功路径的闩由 respondVoice/supplementVoice 的 finally 释放。
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

async function respondVoice(row, mediaFileID) {
  try {
    const res = await respondTeacherTask(taskId.value, row.id, { ritual: "voice", media_file_id: mediaFileID })
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
    await supplementTeacherResponse(taskId.value, row.id, { ritual: "voice", media_file_id: mediaFileID })
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

function startRecord(row, mode) {
  if (recording.value) {
    // 只在「另一行」正在录时打扰：同一行的停止按钮就在原处。
    if (recordingRowId.value !== row.id) {
      uni.showToast({ title: "请先停止当前录音", icon: "none" })
    }
    return
  }
  // 批量/该行其它回应在飞时录完也会撞 409，干脆不让开始。
  if (batchBusy.value || isResponding(row.id)) {
    uni.showToast({ title: "这一行正在处理，请稍后再录", icon: "none" })
    return
  }
  recordingRow = row
  recordingRowId.value = row.id
  recordingMode.value = mode
  recording.value = true
  recorder.start({ format: "mp3", duration: 15000 })
}

function stopRecord() {
  if (recording.value) recorder.stop()
}

// 离开当前列表/刷新前调用：该行的「停止」即将消失，继续录只会在 15s 后
// 自动停止并上传一段老师以为已取消的语音。
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

function fillEdit(t) {
  edit.value = {
    title: t.title || "",
    description: t.description || "",
    learning_link: t.learning_link || "",
    allow_public: !!t.allow_public,
    // 不把空/未知枚举归一化成默认值：保存时会把 "" 变成 once/sticker，
    // 有提交的任务因此被结构锁拒绝（管理端/历史任务存在这种行）。
    frequency: t.frequency || "",
    start_date: t.start_date || "",
    end_date: t.end_date || "",
    target_student_id: t.target_student_id || 0,
    recognition_ritual: t.recognition_ritual || "",
    star_scope: t.star_scope || "",
    star_goal: Number(t.star_goal || 0)
  }
  publicOffPending.value = false
  pendingIndividualId.value = 0
  pendingIndividualName.value = ""
}

// 结构被锁定时，只读摘要与保存都必须用**任务当前值**，不能用户改了一半的 edit：
// 家长可能在老师编辑期间提交，切 tab/保存前发现锁后，未保存的结构值不能当作现状。
function syncStructureFromTask() {
  if (!task.value.id) return
  edit.value.frequency = task.value.frequency || ""
  edit.value.start_date = task.value.start_date || ""
  edit.value.end_date = task.value.end_date || ""
  edit.value.target_student_id = task.value.target_student_id || 0
  edit.value.recognition_ritual = task.value.recognition_ritual || ""
  edit.value.star_scope = task.value.star_scope || ""
  edit.value.star_goal = Number(task.value.star_goal || 0)
}

// 结构字段是否被用户改过（只比较结构，不看标题/说明/学习关联/公开）。
function hasStructuralChanges() {
  return (
    edit.value.frequency !== (task.value.frequency || "") ||
    edit.value.start_date !== (task.value.start_date || "") ||
    edit.value.end_date !== (task.value.end_date || "") ||
    Number(edit.value.target_student_id || 0) !== Number(task.value.target_student_id || 0) ||
    edit.value.recognition_ritual !== (task.value.recognition_ritual || "") ||
    edit.value.star_scope !== (task.value.star_scope || "") ||
    Number(edit.value.star_goal || 0) !== Number(task.value.star_goal || 0)
  )
}

async function loadTask() {
  try {
    const res = await fetchTeacherTask(taskId.value)
    task.value = res.data || {}
    taskError.value = ""
    fillEdit(task.value)
  } catch (e) {
    // 任务详情只影响编辑区与汇总入口，不阻塞提交列表；但要显式说明加载失败。
    taskError.value = e?.message || "任务详情加载失败"
  }
}

// 切 tab/保存都会重取；后返回的旧响应不能覆盖新结果（否则可能把「有提交」洗回 0，
// 结构编辑器重新打开，保存时才 422）。
let statsSeq = 0
async function loadStats() {
  const seq = ++statsSeq
  try {
    const res = await fetchTeacherTaskStats(taskId.value)
    if (seq !== statsSeq) return
    stats.value = res.data || {}
    statsError.value = ""
  } catch (e) {
    if (seq !== statsSeq) return
    // 提交数拿不到时不能猜「没有提交」：那会让结构编辑器开着，保存时才被后端 422。
    statsError.value = e?.message || "提交数加载失败"
  } finally {
    if (seq === statsSeq) {
      statsLoaded.value = true
      // 锁定了（或无法确认）就把结构编辑回填成任务当前值：
      // 只读摘要不能把未保存的改动显示成现状，保存也不能把它们发出去。
      if (structureReadOnly.value) syncStructureFromTask()
    }
  }
}

// 保存前的锁状态复核：家长提交不改变结构值，所以回填 task 是安全的。
// 读失败时也按锁定处理（回填结构），宁可这次只存安全字段，不发出可能被 422 的整单。
async function refreshStatsBeforeSave() {
  try {
    const res = await fetchTeacherTaskStats(taskId.value)
    stats.value = res.data || {}
    statsError.value = ""
    statsLoaded.value = true
    if (structureLocked.value) syncStructureFromTask()
  } catch (e) {
    statsError.value = e?.message || "提交数加载失败"
    statsLoaded.value = true
    syncStructureFromTask()
  }
}

async function loadStudents() {
  try {
    const res = await fetchTeacherClassStudents()
    students.value = res.data?.list || []
    studentsError.value = ""
  } catch (e) {
    students.value = []
    studentsError.value = e?.message || "班级学生加载失败"
  }
}

function onFreqChange(e) {
  edit.value.frequency = FREQ_VALUES[Number(e.detail.value)] || "once"
}
function onRitualChange(e) {
  edit.value.recognition_ritual = RITUAL_VALUES[Number(e.detail.value)] || "sticker"
}
function applyTargetStudent(id) {
  edit.value.target_student_id = id || 0
  // 个体任务强制私密（后端也会强制），UI 先收口，避免出现「保存后还是关的」的假象。
  if (edit.value.target_student_id > 0) {
    edit.value.allow_public = false
    publicOffPending.value = false
  }
  pendingIndividualId.value = 0
  pendingIndividualName.value = ""
}

function onStudentChange(e) {
  const opt = studentChoices.value[Number(e.detail.value)]
  const id = opt?.id || 0
  // 从「允许公开」的班级任务改成个体任务 = 关闭公开 + 撤下已公开内容，先确认。
  if (id > 0 && edit.value.allow_public) {
    pendingIndividualId.value = id
    pendingIndividualName.value = opt.name
    return
  }
  applyTargetStudent(id)
}

function confirmIndividual() {
  applyTargetStudent(pendingIndividualId.value)
}
function cancelIndividual() {
  pendingIndividualId.value = 0
  pendingIndividualName.value = ""
}

function onPublicToggle(e) {
  const next = !!e.detail.value
  if (!next && edit.value.allow_public) {
    // 先落到 false 让开关视觉跟随操作，但不允许直接保存：必须点确认。
    edit.value.allow_public = false
    publicOffPending.value = true
    // 两个破坏性确认不要同时挂着，否则会出现两块确认框、保存被两次拦。
    pendingIndividualId.value = 0
    pendingIndividualName.value = ""
    return
  }
  edit.value.allow_public = next
  if (next) publicOffPending.value = false
}
function cancelPublicOff() {
  edit.value.allow_public = true
  publicOffPending.value = false
}
function confirmPublicOff() {
  edit.value.allow_public = false
  publicOffPending.value = false
  pendingIndividualId.value = 0
  pendingIndividualName.value = ""
}

async function saveEdit() {
  if (pendingIndividualId.value) {
    uni.showToast({ title: "请先确认是否改为个体任务", icon: "none" })
    return
  }
  if (publicOffPending.value) {
    uni.showToast({ title: "请先确认是否关闭公开", icon: "none" })
    return
  }
  if (!edit.value.title.trim() || !edit.value.learning_link.trim()) {
    uni.showToast({ title: "标题和学习关联必填", icon: "none" })
    return
  }
  if (edit.value.end_date < edit.value.start_date) {
    uni.showToast({ title: "结束日期不能早于开始日期", icon: "none" })
    return
  }
  if (edit.value.recognition_ritual === "class_star" && edit.value.star_scope === "task" && Number(edit.value.star_goal) <= 0) {
    uni.showToast({ title: "班级星星仪式需要星星目标大于 0", icon: "none" })
    return
  }
  // 家长可能在老师编辑期间提交：保存前再确认一次锁状态。
  // 先记下用户改过结构没有，回填后才能判断这次保存该不该放行。
  const structuralChanged = hasStructuralChanges()
  await refreshStatsBeforeSave()
  if (structureReadOnly.value) {
    if (structuralChanged) {
      syncStructureFromTask()
      uni.showToast({
        title: statsError.value
          ? "无法确认结构是否可改，已恢复原结构；请重试保存标题/说明/公开"
          : "任务已有家长提交，结构已锁定；已恢复原结构，请再保存",
        icon: "none"
      })
      return
    }
    // 没有结构改动：只发安全字段，结构字段用任务当前值（不会触发 422）。
    syncStructureFromTask()
  }
  try {
    await updateTeacherTask(taskId.value, {
      title: edit.value.title,
      description: edit.value.description,
      learning_link: edit.value.learning_link,
      allow_public: edit.value.allow_public && edit.value.target_student_id === 0,
      target_student_id: edit.value.target_student_id || 0,
      frequency: edit.value.frequency,
      start_date: edit.value.start_date,
      end_date: edit.value.end_date,
      recognition_ritual: edit.value.recognition_ritual,
      // 空值原样发：后端把 "" 当「不修改」，不能替它填 task。
      star_scope: edit.value.star_scope,
      star_goal: Number(edit.value.star_goal) || 0,
      status: task.value.status ?? 1
    })
    uni.showToast({ title: "已保存", icon: "success" })
    // 关公开/改个体会在服务端批量撤销公开状态；不重取提交列表的话，
    // 当前页仍会显示「已公开 · 本班家长可见」，与实际相反。
    await Promise.all([loadTask(), loadStats(), load()])
  } catch (e) {
    uni.showToast({ title: e?.message || "保存失败", icon: "none" })
  }
}

// 只刷新周期与星数，不覆盖编辑区里未保存的输入。
// 星数必须在每次回应/汇总后重取：否则刚加的星要退出重进才看得到。
async function refreshPeriod(seq) {
  try {
    const res = await fetchTeacherTask(taskId.value)
    // 后返回的旧请求不能覆盖新一次 load 的星数/区间。
    if (seq !== undefined && seq !== loadSeq) return
    if (res.data?.period_start) task.value.period_start = res.data.period_start
    if (res.data?.period_end) task.value.period_end = res.data.period_end
    // 服务端明确给 null（星数读失败）时也要落下来，让 UI 隐藏「已集」，
    // 不能让一个陈旧的 0 继续冒充当前值。
    if (res.data?.star_count !== undefined) task.value.star_count = res.data.star_count
  } catch (e) {
    // 周期/星数刷新失败不影响提交列表；旧区间最多导致「本周期」判断短暂陈旧
  }
}

// 请求序号：切 tab 会并发发起两次 load，后返回的旧请求不能覆盖新页签的数据。
let loadSeq = 0

async function load() {
  if (!taskId.value) return
  const seq = ++loadSeq
  loading.value = true
  try {
    const [res] = await Promise.all([
      fetchTeacherTaskCheckins(taskId.value, { status: status.value }),
      refreshPeriod(seq)
    ])
    if (seq !== loadSeq) return
    rows.value = res.data?.list || []
    // 撤回的两步确认不跨刷新沿用：否则上一轮点过「撤回公开」没确认，
    // 刷新后同一行会直接停在「确认撤下？」上，一次点击就撤掉。
    revokeConfirmId.value = 0
    // 每轮加载都作废勾选。服务端没有「本轮申请时间」，家长「撤回→重新申请」
    // 若发生在两次加载之间，pending 状态看起来没变，保留勾选就等于绕过 D7 的重确认。
    scopeConfirmed.value = {}
  } catch (e) {
    if (seq !== loadSeq) return
    uni.showToast({ title: "加载提交失败", icon: "none" })
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

// 等列表刷新落地再解锁：刷新还在飞时解锁，按钮立刻又活了，等于没加闩。
// 但 `uni.request` 没设 timeout（走微信默认 60s），封顶 3 秒，卡住时不能一直锁着。
// 封顶命中时把 loading 一并收回——否则闩开了、按钮可点，屏幕上还写着「加载中...」。
async function refreshWithCap() {
  await Promise.race([load(), new Promise((r) => setTimeout(r, 3000))])
  if (loading.value) loading.value = false
}

// 录音期间不能发起别的回应：批量/贴纸先把提交关掉，录音结束后的语音回应
// 必然撞上 409「已被其他回应关闭」，用户看到的是莫名其妙的失败。
function recordingBlocked() {
  if (!recording.value) return false
  uni.showToast({ title: "请先停止当前录音", icon: "none" })
  return true
}

async function respond(row, sticker) {
  // 批量在飞时单条必然撞车：那一行可能刚被批量关掉，再发就 409。两个闩互相感知。
  if (recordingBlocked()) return
  if (isResponding(row.id) || batchBusy.value) return
  addResponding(row.id)
  try {
    const res = await respondTeacherTask(taskId.value, row.id, { ritual: "sticker", sticker_key: sticker })
    const suffix = notifySuffix(res.data)
    uni.showToast({ title: "已回应" + suffix, icon: suffix ? "none" : "success" })
    // 这里不会 reject——load() 自己吞掉异常并显示「加载提交失败」。
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
    const res = await respondTeacherTask(taskId.value, row.id, { ritual: "class_star" })
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
    // 发一枚 ⭐ 贴纸只是普通贴纸，不计星（D6）。
    const payload = batchIsStar.value
      ? { ritual: "class_star" }
      : { ritual: "sticker", sticker_key: "star" }
    const res = await batchRespondTeacherTask(taskId.value, payload)
    // 只关闭「本周期」的待回应提交；并发/重复点击落败时 count 为 0，不能报成「已回应 0 条」。
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
    // 必须等刷新落地再解锁：load() 不 await 的话，finally 会立刻放开门闩，
    // 列表还在飞行的这段时间里按钮仍可点，第二次点击会真的发请求并弹「已无待回应」。
    await refreshWithCap()
  } catch (e) {
    uni.showToast({ title: e?.message || "批量回应失败", icon: "none" })
  } finally {
    batchBusy.value = false
  }
}

function toggleSupplement(row) {
  supplementFor.value = supplementFor.value === row.id ? 0 : row.id
}

async function supplement(row, sticker) {
  if (recordingBlocked()) return
  if (supplementBusy.value || isResponding(row.id)) return
  supplementBusy.value = true
  try {
    await supplementTeacherResponse(taskId.value, row.id, { ritual: "sticker", sticker_key: sticker })
    uni.showToast({ title: "已追加补充回应", icon: "success" })
    supplementFor.value = 0
    await refreshWithCap()
  } catch (e) {
    uni.showToast({ title: e?.message || "补充回应失败", icon: "none" })
  } finally {
    supplementBusy.value = false
  }
}

function toggleScopeConfirm(id) {
  scopeConfirmed.value = { ...scopeConfirmed.value, [id]: !scopeConfirmed.value[id] }
}

function askRevoke(id) {
  revokeConfirmId.value = id
}
function cancelRevoke() {
  revokeConfirmId.value = 0
}

// 批准必须显式确认「内容中只有提交者孩子」；界面先拦一次，后端仍会再校验（422）。
async function reviewPublic(row, approve) {
  if (isReviewing(row.id)) return
  if (approve && !scopeConfirmed.value[row.id]) {
    uni.showToast({ title: "请先勾选确认内容中只有提交者孩子", icon: "none" })
    return
  }
  reviewingIds.value = [...reviewingIds.value, row.id]
  try {
    await reviewTeacherTaskPublic(taskId.value, row.id, {
      approve,
      scope_confirmed: approve
    })
    uni.showToast({ title: approve ? "已批准公开" : "已撤下", icon: "success" })
    revokeConfirmId.value = 0
    // 这一轮审核已结束，确认勾选作废；家长再次申请时要重新确认。
    scopeConfirmed.value = { ...scopeConfirmed.value, [row.id]: false }
    await refreshWithCap()
  } catch (e) {
    uni.showToast({ title: e?.message || "审核失败", icon: "none" })
  } finally {
    reviewingIds.value = reviewingIds.value.filter((id) => id !== row.id)
  }
}

// 只有真的处于公开状态机里的行才渲染审核块；字段缺失（旧后端）时不画空心卡片。
function hasPublicState(row) {
  return ["pending", "approved", "revoked"].includes(row.public_status)
}

function publicStateText(row) {
  if (row.public_status === "pending") return "家长申请公开，待你审核"
  if (row.public_status === "approved") {
    return row.public_visible ? "已公开 · 本班家长可见" : "已通过，暂不对本班展示"
  }
  // 「老师拒绝」与「家长取消」写的是同一个 revoked_at，不替任何一方下结论。
  if (row.public_status === "revoked") return "未公开"
  return ""
}

function publicStateHint(row) {
  if (row.public_status !== "approved" || row.public_visible) return ""
  // 状态分开表达（B4-a 的教训）：approved 不等于此刻在墙上。
  // 后端口径还包含「任务已结束/已停用」——漏掉会让老师以为只是公开开关的问题。
  return "任务已结束或停用、未开启公开、已改成个体任务，或孩子已不在本班，本班家长暂时看不到"
}

// 生成家长端任务小程序码并保存到相册，老师随后转发班级群。
// 码由后端用**家长端**凭据生成（老师端码家长扫了会进错端），客户端只做落盘。
async function shareTaskCard() {
  if (!taskId.value || shareBusy.value) return
  shareBusy.value = true
  let stage = "generate"
  uni.showLoading({ title: "生成小程序码", mask: true })
  try {
    const res = await fetchTaskShareCard(taskId.value)
    const base64 = res.data?.image_base64 || ""
    if (!base64) {
      // 接口 200 但没给图：http 层不会弹任何提示，这里必须自己说话，否则点了没反应。
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
    // 这里必须封顶，超时后把按钮放回可点状态。
    // （loading 已在写完后收起，免得遮罩盖住系统授权弹窗。）
    await withTimeout(saveImageToAlbum(filePath), 15000, "保存到相册超时")
    uni.showToast({ title: "已保存，可转发班级群", icon: "none" })
  } catch (e) {
    if (stage === "generate") {
      // 请求失败时 http 层已经弹过后端文案（含「请检查家长端微信配置」），这里不再重复。
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
    // 统一在这里收 loading：早退分支（空图）不会走 catch，
    // 只靠 catch 收会让遮罩一直挂着。
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

function switchStatus(s) {
  // 当前行的「停止」按钮会随列表切换消失，先取消录音。
  cancelRecordingIfAny("已取消录音")
  status.value = s
  // 提交数可能在这一页停留期间变化（家长刚提交）。切 tab 时一并重取，
  // 否则结构编辑器还开着，保存时才被后端 422「有提交后结构不可改」。
  loadStats()
  load()
}

onLoad((opts) => {
  taskId.value = Number(opts?.id || 0)
  loadTask()
  loadStats()
  loadStudents()
  load()
})

onUnload(() => {
  // 先置 pageGone：停止录音会触发 onStop，那里看到 pageGone 就不再上传。
  pageGone = true
  if (recording.value) recorder.stop()
  if (audio) {
    audio.stop()
    audio.destroy()
    audio = null
  }
})

// 409 的文案是「请刷新后重试」，而这一页原先没有任何刷新手段：没有 onShow、
// 也没开下拉刷新，撞上「该提交已被其他回应关闭」只能退出重进，那条陈旧行还
// 一直在屏幕上，再点还是 409。
onPullDownRefresh(() => {
  // 下拉会重排列表，录音行可能不在原位；先取消，避免 15s 后自动发出。
  cancelRecordingIfAny("已取消录音")
  Promise.all([loadTask(), loadStats(), load()]).finally(() => uni.stopPullDownRefresh())
})
</script>

<template>
  <view class="page">
    <view v-if="taskError" class="card error-card" @click="loadTask">
      <text class="error-title">{{ taskError }}</text>
      <text class="error-hint">点击重试；任务详情未加载时无法判断批量回应范围</text>
    </view>
    <view v-if="task.id" class="card edit-card">
      <text class="name">编辑任务</text>
      <input v-model="edit.title" class="input" placeholder="任务标题" />
      <textarea v-model="edit.description" class="textarea" placeholder="任务说明" />
      <input v-model="edit.learning_link" class="input" placeholder="和幼儿园学习的关系" />
      <!-- 公开只对班级任务开放；个体任务即使点了开关，后端也会静默强制关掉，
           老师会看到「保存成功 → 再进来还是关的」。这里直接不给开关并说明原因。 -->
      <view v-if="edit.target_student_id === 0" class="switch-row">
        <text>允许家长申请公开</text>
        <switch :checked="edit.allow_public" @change="onPublicToggle" />
      </view>
      <view v-else class="switch-row">
        <text>指派给个别学生的任务不可公开</text>
      </view>
      <!-- 关闭公开会批量撤下已公开内容，必须显式确认；pending 期间禁止保存。 -->
      <view v-if="publicOffPending" class="danger-confirm">
        <text class="danger-title">关闭公开后，已公开内容将全部撤下，家长需要重新申请</text>
        <view class="danger-actions">
          <view class="danger-btn" @click="confirmPublicOff">确认关闭</view>
          <view class="danger-btn plain" @click="cancelPublicOff">取消</view>
        </view>
      </view>
      <view v-if="pendingIndividualId" class="danger-confirm">
        <text class="danger-title">改为「{{ pendingIndividualName }}」的个体任务会关闭公开并撤下已公开内容，家长需要重新申请</text>
        <view class="danger-actions">
          <view class="danger-btn" @click="confirmIndividual">确认改为个体</view>
          <view class="danger-btn plain" @click="cancelIndividual">取消</view>
        </view>
      </view>

      <view class="structure">
        <view class="structure-head">
          <text class="structure-title">任务结构</text>
          <text v-if="!statsLoaded" class="structure-lock">正在确认是否已有提交…</text>
          <text v-else-if="structureLocked" class="structure-lock">已有 {{ stats.submission_count }} 条提交，结构已锁定</text>
          <text v-else-if="statsError" class="structure-lock">未能确认提交数，已按锁定处理</text>
        </view>
        <template v-if="structureReadOnly">
          <text class="structure-line">频率：{{ freqLabel(edit.frequency) }}</text>
          <text class="structure-line">日期：{{ edit.start_date }} ~ {{ edit.end_date }}</text>
          <text class="structure-line">范围：{{ targetLabel }}</text>
          <text class="structure-line">仪式：{{ ritualLabel(edit.recognition_ritual) }}</text>
          <text class="structure-line">星星：{{ starScopeLabel }}<text v-if="edit.star_scope === 'task'"> · 目标 {{ edit.star_goal || 0 }}</text><text v-if="task.star_count != null && Number(task.star_count) > 0 && (edit.star_scope === 'class' || Number(edit.star_goal) > 0)"> · 已集 {{ task.star_count }} 星<text v-if="edit.star_scope === 'class'">（本任务）</text></text></text>
          <text v-if="statsError" class="hint">未能确认是否已有提交，已按锁定处理；请点下方重试后再改结构</text>
          <text v-else-if="!statsLoaded" class="hint">正在确认提交数…</text>
          <text v-else class="hint">有提交后只能改标题、说明、学习关联与公开开关；要调整结构请新建任务</text>
          <text v-if="statsError" class="hint retry" @click="loadStats">提交数加载失败，点击重试</text>
        </template>
        <template v-else>
          <picker :range="FREQ_OPTIONS" :value="freqPickerValue" @change="onFreqChange">
            <view class="input picker">频率：{{ freqText }}</view>
          </picker>
          <picker mode="date" :value="edit.start_date" @change="edit.start_date = $event.detail.value">
            <view class="input picker">开始日期：{{ edit.start_date }}</view>
          </picker>
          <picker mode="date" :value="edit.end_date" @change="edit.end_date = $event.detail.value">
            <view class="input picker">结束日期：{{ edit.end_date }}</view>
          </picker>
          <picker :range="studentOptions" :value="studentIndex" @change="onStudentChange">
            <view class="input picker">范围：{{ targetLabel }}</view>
          </picker>
          <text v-if="studentsError" class="hint">班级学生加载失败，个体任务请谨慎修改范围</text>
          <picker :range="RITUAL_OPTIONS" :value="ritualPickerValue" @change="onRitualChange">
            <view class="input picker">回应仪式：{{ ritualText }}</view>
          </picker>
          <view class="input picker">星星罐：{{ starScopeLabel }}</view>
          <input
            v-if="edit.star_scope === 'task'"
            v-model.number="edit.star_goal"
            class="input"
            type="number"
            placeholder="星星目标（0=不启用；选班级星星仪式时必填）"
          />
          <text class="hint">按班级时使用班级罐目标（可在「亲子任务」页设置或更新）；任务一旦有提交，频率/日期/范围/仪式/星星目标都不能再改</text>
        </template>
      </view>
      <view class="batch" @click="saveEdit">保存</view>
    </view>
    <view v-if="task.id" class="card share-card">
      <view class="share-copy">
        <text class="share-title">让家长扫码打开任务</text>
        <text class="share-hint">生成家长端小程序码，保存到相册后转发班级群</text>
      </view>
      <view class="share-btn" :class="{ busy: shareBusy }" @click="shareTaskCard">
        <text>{{ shareBusy ? "生成中…" : "生成小程序码" }}</text>
      </view>
    </view>
    <view class="tabs">
      <view class="tab" :class="{ active: status === 'pending' }" @click="switchStatus('pending')">待回应</view>
      <view class="tab" :class="{ active: status === 'responded' }" @click="switchStatus('responded')">已回应</view>
    </view>
    <!-- 与后端一致：汇总回应只关闭当前周期，旧周期的待回应走单条回应 -->
    <view v-if="status === 'pending' && rows.length && canBatch" class="batch" :class="{ busy: batchBusy || respondingIds.length }" @click="batchRespond">{{ batchLabel }}</view>
    <!-- 任务详情没加载成功时不作结论，避免把加载失败误报成「需逐条回应」 -->
    <view v-if="status === 'pending' && rows.length && task.id && !canBatch" class="batch-hint">一次性任务与个体任务需逐条回应</view>
    <view v-if="status === 'pending' && rows.length && starTracking" class="batch-hint">「加星」才计入星星罐；贴纸回应不计星</view>
    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="!rows.length" class="state">暂无提交</view>
    <view v-for="row in rows" :key="row.id" class="card">
      <text class="name">{{ row.student?.name || `学生 ${row.student_id}` }}</text>
      <text class="content">{{ row.content || "（一键完成）" }}</text>
      <text v-if="status === 'pending' && !isCurrentPeriod(row)" class="old-period">旧周期 · 请逐条回应</text>
      <text v-if="row.media_url" class="media">已附照片/语音</text>

      <!-- 公开审核：批准前必须勾确认；撤回永远可用（任务关掉公开后也要能清干净）。 -->
      <view v-if="hasPublicState(row)" class="review" :class="'review-' + row.public_status">
        <text class="review-state">{{ publicStateText(row) }}</text>
        <text v-if="publicStateHint(row)" class="review-hint">{{ publicStateHint(row) }}</text>
        <view
          v-if="row.public_status === 'pending' && canApprovePublic"
          class="scope-confirm"
          @click="toggleScopeConfirm(row.id)"
        >
          <text class="scope-box">{{ scopeConfirmed[row.id] ? "☑" : "☐" }}</text>
          <text class="scope-text">我确认内容中只有提交者孩子</text>
        </view>
        <text v-else-if="row.public_status === 'pending' && task.id" class="review-hint">
          任务已关闭公开或已改成个体任务，不能批准；可撤回这条申请
        </text>
        <text v-else-if="row.public_status === 'pending'" class="review-hint">
          任务详情未加载，暂时无法批准公开；可先撤回
        </text>
        <view
          v-if="row.public_status === 'pending' || row.public_status === 'approved'"
          class="review-actions"
          :class="{ busy: isReviewing(row.id) }"
        >
          <view
            v-if="row.public_status === 'pending' && canApprovePublic"
            class="review-btn approve"
            :class="{ disabled: !scopeConfirmed[row.id] }"
            @click="reviewPublic(row, true)"
          >批准公开</view>
          <view v-if="revokeConfirmId !== row.id" class="review-btn reject" @click="askRevoke(row.id)">
            {{ row.public_status === "pending" ? "拒绝" : "撤回公开" }}
          </view>
          <template v-else>
            <view class="review-btn reject" @click="reviewPublic(row, false)">确认撤下？</view>
            <view class="review-btn plain" @click="cancelRevoke">取消</view>
          </template>
        </view>
      </view>

      <view v-if="status === 'pending'" class="actions" :class="{ busy: batchBusy || isResponding(row.id) }">
        <view v-for="s in responseStickers" :key="s" class="sticker" @click="respond(row, s)">{{ stickerGlyph(s) }}</view>
        <view v-if="starTracking" class="sticker star" @click="respondStar(row)">加星</view>
        <view v-if="!(recording && recordingRowId === row.id)" class="sticker" @click="startRecord(row, 'respond')">录音</view>
        <view v-else class="sticker recording" @click="stopRecord">停止</view>
      </view>

      <view v-if="status === 'responded' && row.response" class="response">
        <text class="response-label">你的回应</text>
        <text v-if="row.response.ritual === 'sticker'" class="response-glyph">{{ stickerGlyph(row.response.sticker_key) }}</text>
        <text v-else-if="row.response.ritual === 'voice'" class="response-play" @click="playAudio(row.response.media_url)">🔊 点击播放语音</text>
        <text v-else-if="row.response.ritual === 'class_star'" class="response-star">⭐ 班级星星</text>
        <text v-if="row.response.content" class="response-text">{{ row.response.content }}</text>
      </view>

      <!-- 历史补充回应由 checkins 返回；家长端按同样顺序展示。 -->
      <view v-if="row.supplements && row.supplements.length" class="supplements">
        <text class="supplements-label">补充回应 {{ row.supplements.length }} 条</text>
        <view v-for="sp in row.supplements" :key="sp.id" class="supplement-item">
          <text v-if="sp.ritual === 'sticker'" class="supplement-glyph">{{ stickerGlyph(sp.sticker_key) }}</text>
          <text v-else-if="sp.ritual === 'voice'" class="supplement-play" @click="playAudio(sp.media_url)">🔊 播放语音补充</text>
          <text v-if="sp.content" class="supplement-text">{{ sp.content }}</text>
        </view>
      </view>

      <!-- 补充回应只能追加到已有主回应的提交；仪式仅贴纸/语音，班级星星不可选。 -->
      <view v-if="status === 'responded' && row.response" class="supplement">
        <view v-if="supplementFor !== row.id" class="supplement-open" @click="toggleSupplement(row)">补充回应（个别夸一句）</view>
        <view v-else class="supplement-box" :class="{ busy: supplementBusy }">
          <text class="supplement-title">补充贴纸（只发给这个孩子）</text>
          <view class="supplement-stickers">
            <view v-for="s in SUPPLEMENT_STICKERS" :key="s" class="supplement-sticker" @click="supplement(row, s)">{{ stickerGlyph(s) }}</view>
          </view>
          <view class="supplement-row">
            <view v-if="!(recording && recordingRowId === row.id)" class="supplement-rec" @click="startRecord(row, 'supplement')">录语音补充</view>
            <view v-else class="supplement-rec recording" @click="stopRecord">停止录音</view>
            <view class="supplement-cancel" @click="toggleSupplement(row)">收起</view>
          </view>
          <text class="hint">补充回应不发通知，家长在任务详情里看到；不发星星</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $component-page-padding-y $component-page-padding-x 48rpx; background: $color-bg-page; }
.edit-card { margin-bottom: $spacing-lg; }
.input, .textarea { box-sizing: border-box; width: 100%; margin-top: $spacing-sm; padding: $spacing-sm $spacing-md; border-radius: $border-radius-lg; background: $color-bg-page; font-size: $typography-font-size-base; }
.input { min-height: $component-input-height; }
.textarea { min-height: 140rpx; }
.picker { display: flex; align-items: center; }
.switch-row { display: flex; align-items: center; justify-content: space-between; margin-top: $spacing-md; color: $color-text-body; font-size: $typography-font-size-sm; }
.hint { display: block; margin-top: $spacing-xs; color: $color-text-hint; font-size: $typography-font-size-xs; line-height: 1.5; }
.hint.retry { color: $color-brand-primary; }
.tabs { display: flex; gap: $spacing-md; margin-bottom: $spacing-lg; }
.tab { flex: 1; text-align: center; padding: $spacing-md 0; border-radius: $border-radius-lg; background: $color-bg-card; color: $color-text-muted; }
.tab.active { background: $color-brand-primary; color: $color-text-on-brand; }
.batch { margin-bottom: $spacing-lg; padding: $spacing-md; text-align: center; border-radius: $border-radius-lg; background: $color-semantic-success-bg; color: $color-semantic-success; }
.batch.busy { opacity: 0.6; }
.state { padding: 120rpx 0; text-align: center; color: $color-text-hint; }
.batch-hint { margin-bottom: $spacing-lg; padding: $spacing-md; text-align: center; border-radius: $border-radius-lg; background: $color-bg-page; color: $color-text-muted; font-size: $typography-font-size-sm; }
.old-period { display: block; margin-top: $spacing-xs; color: $color-semantic-warning; font-size: $typography-font-size-sm; }
.error-card { background: $color-semantic-danger-bg; }
.error-title { display: block; color: $color-semantic-danger; font-size: $typography-font-size-md; font-weight: 700; }
.error-hint { display: block; margin-top: 8rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
.card { margin-bottom: $spacing-md; padding: $component-card-padding; border-radius: $component-card-radius; background: $color-bg-card; box-shadow: $shadow-sm; }
.name { display: block; font-size: $typography-font-size-md; font-weight: 700; color: $color-text-primary; }
.content { display: block; margin-top: $spacing-xs; font-size: $typography-font-size-base; color: $color-text-body; }
.media { display: block; margin-top: $spacing-xs; font-size: $typography-font-size-sm; color: $color-text-muted; }
.actions { display: flex; flex-wrap: wrap; gap: $spacing-md; margin-top: $spacing-md; }
.actions.busy { opacity: 0.6; }
.sticker { padding: $spacing-xs $spacing-lg; border-radius: $border-radius-full; background: $color-bg-page; color: $color-brand-primary; font-size: $typography-font-size-lg; }
.sticker.star { font-size: $typography-font-size-sm; line-height: 2; }
.recording { background: $color-semantic-danger-bg; color: $color-semantic-danger; }
.response { margin-top: $spacing-sm; padding: $spacing-sm $spacing-md; border-radius: $border-radius-md; background: $color-bg-page; }
.response-label { display: block; color: $color-text-muted; font-size: $typography-font-size-xs; }
.response-glyph { display: block; margin-top: 4rpx; font-size: 56rpx; line-height: 1.2; }
.response-play { display: block; margin-top: 4rpx; color: $color-brand-primary; font-size: $typography-font-size-sm; }
.response-star { display: block; margin-top: 4rpx; color: $color-semantic-warning; font-size: $typography-font-size-sm; }
.response-text { display: block; margin-top: 4rpx; color: $color-text-body; font-size: $typography-font-size-sm; }
.supplements { margin-top: $spacing-sm; padding: $spacing-sm $spacing-md; border-radius: $border-radius-md; background: $color-brand-primary-bg; }
.supplements-label { display: block; color: $color-text-muted; font-size: $typography-font-size-xs; }
.supplement-item { margin-top: $spacing-xs; }
.supplement-glyph { font-size: 44rpx; line-height: 1.2; }
.supplement-play { color: $color-brand-primary; font-size: $typography-font-size-sm; }
.supplement-text { display: block; margin-top: 4rpx; color: $color-text-body; font-size: $typography-font-size-sm; }
.supplement { margin-top: $spacing-sm; }
.supplement-open { padding: $spacing-xs 0; color: $color-brand-primary; font-size: $typography-font-size-sm; }
.supplement-box { padding: $spacing-sm; border-radius: $border-radius-md; background: $color-brand-primary-bg; }
.supplement-box.busy { opacity: 0.6; }
.supplement-title { display: block; color: $color-text-body; font-size: $typography-font-size-xs; }
.supplement-stickers { display: flex; gap: $spacing-md; margin-top: $spacing-xs; }
.supplement-sticker { padding: 4rpx $spacing-sm; border-radius: $border-radius-full; background: $color-bg-card; font-size: 44rpx; line-height: 1.3; }
.supplement-row { display: flex; align-items: center; gap: $spacing-md; margin-top: $spacing-sm; }
.supplement-rec { padding: $spacing-xs $spacing-md; border-radius: $border-radius-full; background: $color-bg-card; color: $color-brand-primary; font-size: $typography-font-size-sm; }
.supplement-rec.recording { background: $color-semantic-danger-bg; color: $color-semantic-danger; }
.supplement-cancel { color: $color-text-muted; font-size: $typography-font-size-sm; }
.review { margin-top: $spacing-sm; padding: $spacing-sm $spacing-md; border-radius: $border-radius-md; background: $color-bg-page; }
.review-pending { background: $color-semantic-warning-bg; }
.review-approved { background: $color-semantic-success-bg; }
.review-state { display: block; color: $color-text-body; font-size: $typography-font-size-sm; font-weight: $typography-font-weight-bold; }
.review-pending .review-state { color: $color-semantic-warning; }
.review-approved .review-state { color: $color-semantic-success; }
.review-hint { display: block; margin-top: 4rpx; color: $color-text-muted; font-size: $typography-font-size-xs; line-height: 1.5; }
.scope-confirm { display: flex; align-items: center; gap: $spacing-sm; margin-top: $spacing-sm; }
.scope-box { color: $color-brand-primary; font-size: $typography-font-size-md; }
.scope-text { color: $color-text-body; font-size: $typography-font-size-sm; }
.review-actions { display: flex; flex-wrap: wrap; gap: $spacing-md; margin-top: $spacing-sm; }
.review-actions.busy { opacity: 0.6; }
.review-btn { padding: $spacing-xs $spacing-lg; border-radius: $border-radius-full; font-size: $typography-font-size-sm; }
.review-btn.approve { background: $color-semantic-success; color: $color-text-on-brand; }
.review-btn.approve.disabled { opacity: 0.4; }
.review-btn.reject { background: $color-semantic-danger-bg; color: $color-semantic-danger; }
.review-btn.plain { background: $color-bg-card; color: $color-text-muted; }
/* 1rpx 分隔线没有对应 token（token 覆盖圆角/间距/字号，不含描边宽度），局部特例 */
.structure { margin-top: $spacing-lg; padding-top: $spacing-md; border-top: 1rpx solid $color-bg-page; }
.structure-head { display: flex; align-items: baseline; justify-content: space-between; gap: $spacing-sm; }
.structure-title { color: $color-text-primary; font-size: $typography-font-size-md; font-weight: $typography-font-weight-bold; }
.structure-lock { color: $color-text-hint; font-size: $typography-font-size-xs; }
.structure-line { display: block; margin-top: $spacing-xs; color: $color-text-body; font-size: $typography-font-size-sm; }
.danger-confirm { margin-top: $spacing-md; padding: $spacing-sm $spacing-md; border-radius: $border-radius-md; background: $color-semantic-danger-bg; }
.danger-title { display: block; color: $color-semantic-danger; font-size: $typography-font-size-sm; line-height: 1.5; }
.danger-actions { display: flex; gap: $spacing-md; margin-top: $spacing-sm; }
.danger-btn { padding: $spacing-xs $spacing-lg; border-radius: $border-radius-full; background: $color-semantic-danger; color: $color-text-on-brand; font-size: $typography-font-size-sm; }
.danger-btn.plain { background: $color-bg-card; color: $color-text-muted; }
.share-card { display: flex; align-items: center; justify-content: space-between; gap: $spacing-md; }
.share-copy { flex: 1; }
.share-title { display: block; font-size: $typography-font-size-md; font-weight: $typography-font-weight-bold; color: $color-text-primary; }
.share-hint { display: block; margin-top: $spacing-xs; font-size: $typography-font-size-xs; color: $color-text-muted; }
.share-btn { flex-shrink: 0; padding: $spacing-sm $spacing-lg; border-radius: $border-radius-full; background: $color-brand-primary; }
.share-btn text { color: $color-text-on-brand; font-size: $typography-font-size-sm; }
.share-btn.busy { opacity: 0.6; }
</style>
