<script setup>
/**
 * 任务详情 · 一次提交的完整回路
 *
 * 四个区块按源码的独立条件渲染：
 * 任务说明 → 未读提示 → 老师回应 → 我的提交 / 今日提交 → 打卡历史。
 * 回应看 responses、未读看 has_unread_supplement、提交看 submitted_current_period，
 * 三者互不绑定：老师可以在家长本周期还没提交时先回应。
 */
import { computed, ref } from "vue"
import { onLoad, onUnload } from "@dcloudio/uni-app"
import { getCurrentStudentId, setCurrentStudentId } from "@/utils/currentStudent"
import { parseSceneTaskId } from "@/utils/taskScene"
import { fetchParentHomeOverview, switchStudent } from "@/api/parent-home"
import { ensureLogin } from "@/utils/login"
import { getToken } from "@/utils/auth"
import config from "@/config/env"
import {
  formatCommentTime,
  formatDateLabel,
  formatMonthDayWeekday,
  formatRecordTime,
  parseDateOnly
} from "@/utils/time"
import {
  REACTIONS,
  checkinPeriodHint,
  frequencyTaskLabel,
  isRevoking,
  publicActionLabel,
  publicBadge,
  publicNote,
  publicStateOf,
  responseGlyph,
  ritualLabel,
  starJarValue,
  stickerGlyph
} from "@/utils/tasks"
import {
  fetchTaskDetail,
  checkinTask,
  fetchCheckinHistory,
  markTaskRead,
  reactResponse,
  requestPublicCheckin,
  cancelPublicCheckin,
  withStudentScope
} from "@/api/parent-task"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  alert: "", // error-warning-line
  add: "", // add-line
  close: "", // close-line
  play: "", // play-circle-line
  chat: "", // chat-3-line
  check: "" // check-line
}

const detail = ref({})
const history = ref([])
// 后端分页上限 50；页面只展示最近 5 条，total 用于脚注说明还有多少没显示。
const historyTotal = ref(0)
const checkinContent = ref("")
const mediaFileID = ref(0)
const mediaPreview = ref("")
const uploading = ref(false)
const submitting = ref(false)
const loading = ref(true)
const loadError = ref("")
const unbound = ref(false)
const retrying = ref(false)
const taskId = ref(0)
// 深链（消息 action_path / 小程序码 scene）里携带的目标孩子。
// 它只用来提示「你正在看的是另一个孩子」，不参与取数：
// 家长端所有任务接口都必须用当前孩子的 student_id（D10 不回退）。
const linkedStudentId = ref(0)
const currentSid = ref(0)
const currentChildName = ref("")
// 历史为空时拿不到 student_name，用 overview 兜底提示条里的孩子姓名。
const childNameFallback = ref("")
// 读标记清掉之后 this 仍为 true：用来解释「为什么刚才有红点」。
const hadUnread = ref(false)
const reactingIds = ref([])
const publicBusyIds = ref([])
// 撤下的二次确认走页面内两步点击，而不是 uni.showModal：原生弹窗不进入视图层 DOM，
// 两步点击同样是显式确认，且能在真实交互里验证。
const pendingCancelId = ref(0)
const kidTarget = ref(null)
// 播放状态按 URL 归属：主回应与补充回应可能同时有语音，布尔会让所有按钮一起变成播放中。
const playingUrl = ref("")
const inputFocus = ref(false)
let audio = null

// 草稿归属的孩子：换孩子时清空草稿，避免把 A 的文字/照片记到 B 名下。
const draftSid = ref(0)
let loadSeq = 0

const isEnded = computed(() => Number(detail.value.status) === 0)
const submitted = computed(() => Boolean(detail.value.submitted_current_period))
const currentCheckin = computed(() => detail.value.current_checkin || null)
const responses = computed(() => (Array.isArray(detail.value.responses) ? detail.value.responses : []))
const requirements = computed(() => (Array.isArray(detail.value.requirements) ? detail.value.requirements : []))
// 已结束任务不能提交；本周期已提交时打卡区隐藏，换成「我的提交」。
const showCheckinForm = computed(() => Boolean(detail.value.id) && !submitted.value && !isEnded.value)
// current_checkin 是最近一次提交；已提交或任务已结束时才作为「我的提交」展示，
// 避免把上一周期的旧提交误标成这一周期。
const showSubmitCard = computed(() => Boolean(currentCheckin.value) && (submitted.value || isEnded.value))
const visibleHistory = computed(() => history.value.slice(0, 5))
const showHistoryMore = computed(() => historyTotal.value > 5)
const childMismatch = computed(
  () => Boolean(linkedStudentId.value) && Boolean(currentSid.value) && linkedStudentId.value !== currentSid.value
)
const childLabel = computed(() => currentChildName.value || childNameFallback.value || "当前孩子")

// 老师账号已删除时 nickname 为空串，回退「老师的回应」而不是显示「的回应」。
const feedbackTitle = computed(() => {
  const name = String(responses.value[0]?.teacher?.nickname || "").trim()
  return name ? `${name}的回应` : "老师的回应"
})
// 计数与设计稿一致：主回应 + 每条补充回应都算一条。
const feedbackCount = computed(() =>
  responses.value.reduce(
    (sum, resp) => sum + 1 + (Array.isArray(resp.supplements) ? resp.supplements.length : 0),
    0
  )
)

const statusChip = computed(() => {
  if (isEnded.value) return { label: "已结束", tone: "ended" }
  if (!submitted.value) return { label: "待提交", tone: "todo" }
  // 「已通过」取本周期提交的公开审核状态；老师回应是独立区块，不占用这个 chip。
  if (currentCheckin.value?.public_status === "approved") return { label: "已通过", tone: "ok" }
  return { label: "已提交", tone: "wait" }
})

const checkinHint = computed(() => checkinPeriodHint(detail.value.frequency))
const canSubmit = computed(
  () => !submitting.value && !uploading.value &&
    (Boolean(checkinContent.value.trim()) || Boolean(mediaFileID.value))
)

const errorTitle = computed(() => {
  if (unbound.value) return "未绑定孩子信息"
  return loadError.value || "任务加载失败"
})
const errorDesc = computed(() => {
  if (unbound.value) return "请先在首页绑定孩子，再查看亲子任务"
  if (loadError.value === "任务已结束或不存在") {
    return `可能老师已删除这条任务，或它不属于${childLabel.value}`
  }
  return "网络开了小差，请稍后再试"
})

// 申请公开受任务条件约束；撤回永远可用（契约：任务结束/转班也要能撤下）。
const publicApplyAllowed = computed(() => {
  const d = detail.value
  if (!d.allow_public || Number(d.target_student_id) || Number(d.status) !== 1 || isEnded.value) return false
  // 未开始的任务也不能申请：后端只对「已开始且未结束」的任务开放。
  const start = parseDateOnly(d.start_date)
  if (!start) return true
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  return start <= today
})

function canApply(row) {
  if (!publicApplyAllowed.value) return false
  // 一键完成（无文字无媒体）申请必然 422，直接不给入口。
  const content = String(row?.content || "").trim()
  return Boolean(content || row?.media_url)
}

const currentPublicState = computed(() => publicStateOf(currentCheckin.value))
const currentPublicBadge = computed(() => publicBadge(currentPublicState.value))
const currentPublicNote = computed(() => publicNote(currentPublicState.value, currentCheckin.value?.public_visible))
const showCurrentPublicAction = computed(() => {
  if (!currentCheckin.value) return false
  if (isRevoking(currentPublicState.value)) return true
  return canApply(currentCheckin.value)
})
const currentPublicActionLabel = computed(() =>
  publicActionLabel(currentPublicState.value, pendingCancelId.value === currentCheckin.value?.id)
)

function historyBadge(row) {
  return publicBadge(publicStateOf(row))
}

function historyNote(row) {
  return publicNote(publicStateOf(row), row?.public_visible)
}

function historyActionLabel(row) {
  return publicActionLabel(publicStateOf(row), pendingCancelId.value === row.id)
}

function showHistoryAction(row) {
  if (isRevoking(publicStateOf(row))) return true
  return canApply(row)
}

// 已结束 / 未开始的任务不能写读标记：后端对这类任务返回 404。
function shouldMarkRead(task) {
  if (!task?.id) return false
  if (Number(task.status) === 0) return false
  const start = parseDateOnly(task.start_date)
  if (!start) return true
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  return start <= today
}

function resetDraft() {
  checkinContent.value = ""
  mediaFileID.value = 0
  mediaPreview.value = ""
  inputFocus.value = false
}

async function resolveChildNameFallback(sid) {
  try {
    const res = await fetchParentHomeOverview(sid)
    const hit = (res.data?.students || []).find((s) => s.id === sid)
    if (hit?.name) childNameFallback.value = hit.name
  } catch (err) {
    // 只是提示条文案，失败不打扰用户；但别静默到查不出问题。
    console.warn("[tasks/detail] 取孩子姓名失败", err?.message || err)
  }
}

async function loadData(id, options = {}) {
  const keepView = Boolean(options.keepView)
  const seq = ++loadSeq
  taskId.value = Number(id) || 0
  if (!keepView) {
    loading.value = true
    loadError.value = ""
  }
  try {
    await ensureLogin()
    // 当前孩子解析与 403 重试统一走 withStudentScope（与列表页同一条路径）。
    const scoped = await withStudentScope((sid) => Promise.all([
      fetchTaskDetail(id, { student_id: sid }),
      fetchCheckinHistory(id, { student_id: sid, page: 1, page_size: 50 })
    ]))
    if (seq !== loadSeq) return
    if (!scoped) {
      detail.value = {}
      history.value = []
      unbound.value = true
      loadError.value = "未绑定孩子信息"
      return
    }
    unbound.value = false
    // 换孩子时清掉上一条没提交的草稿。
    if (draftSid.value && draftSid.value !== scoped.sid) resetDraft()
    draftSid.value = scoped.sid
    const [d, h] = scoped.result
    detail.value = d.data || {}
    history.value = h.data?.list || []
    historyTotal.value = Number(h.data?.total || history.value.length)
    currentSid.value = scoped.sid
    currentChildName.value = history.value[0]?.student_name || ""
    hadUnread.value = Boolean(detail.value.has_unread_supplement)
    pendingCancelId.value = 0
    loadError.value = ""
    if (childMismatch.value && !currentChildName.value) resolveChildNameFallback(scoped.sid)
    // 读标记失败只留日志：详情已经渲染，不能因为一次清理红点失败把整页挡成错误态。
    if (shouldMarkRead(detail.value)) {
      markTaskRead(id, scoped.sid).catch((err) => {
        console.warn("[tasks/detail] 清除未读标记失败", err?.message || err)
      })
    }
  } catch (err) {
    if (seq !== loadSeq) return
    console.warn("[tasks/detail] load failed:", err?.message || err)
    detail.value = {}
    history.value = []
    // 404 是任务删除/结束的常态入口（消息深链、扫码），给一句人话而不是白屏。
    loadError.value = err?.code === 40400 ? "任务已结束或不存在" : "任务加载失败"
    // 加载失败时提示条仍在：这条路径同样要能说清「你看的是谁」。
    if (childMismatch.value && currentSid.value) resolveChildNameFallback(currentSid.value)
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

async function handleRetry() {
  if (retrying.value || loading.value) return
  retrying.value = true
  await loadData(taskId.value, { keepView: true })
  retrying.value = false
  if (!loadError.value) uni.showToast({ title: "已重新加载", icon: "none" })
}

async function switchToMessageChild() {
  if (!linkedStudentId.value) return
  try {
    await switchStudent(linkedStudentId.value)
    setCurrentStudentId(linkedStudentId.value)
    await loadData(taskId.value)
  } catch (err) {
    uni.showToast({ title: err?.message || "切换孩子失败", icon: "none" })
  }
}

function chooseMedia() {
  if (uploading.value || submitting.value) return
  ensureLogin().then(() => {
    uni.chooseImage({
      count: 1,
      success: (res) => {
        const path = res.tempFilePaths?.[0]
        if (!path) return
        mediaPreview.value = path
        uploading.value = true
        uni.showLoading({ title: "上传中" })
        uni.uploadFile({
          url: config.uploadURL,
          filePath: path,
          name: "file",
          formData: { biz_type: "task" },
          header: { Authorization: `Bearer ${getToken()}` },
          success: (up) => {
            // 上传期间草稿被移除或换了孩子：丢弃这次结果，别把照片记到新草稿上。
            if (mediaPreview.value !== path) return
            try {
              const body = JSON.parse(up.data || "{}")
              if (body.code === 0 && body.data?.id) {
                mediaFileID.value = body.data.id
                uni.showToast({ title: "上传成功", icon: "success" })
              } else {
                mediaPreview.value = ""
                uni.showToast({ title: body.message || "上传失败", icon: "none" })
              }
            } catch (e) {
              mediaPreview.value = ""
              uni.showToast({ title: "上传失败", icon: "none" })
            }
          },
          fail: () => {
            if (mediaPreview.value === path) mediaPreview.value = ""
            uni.showToast({ title: "上传失败", icon: "none" })
          },
          complete: () => {
            uploading.value = false
            uni.hideLoading()
          }
        })
      }
    })
  }).catch((err) => {
    // 登录失败由全局登录流程处理；这里只留排查线索，不重复弹错。
    console.warn("[tasks/detail] ensureLogin failed:", err?.message || err)
  })
}

function removeMedia() {
  if (uploading.value) return
  mediaFileID.value = 0
  mediaPreview.value = ""
}

async function handleCheckin() {
  if (!canSubmit.value || !showCheckinForm.value || !detail.value.id) return
  submitting.value = true
  try {
    // student_id 走 query（第三参）：body 里的会被后端丢弃。
    const res = await checkinTask(detail.value.id, {
      content: checkinContent.value.trim(),
      media_file_id: mediaFileID.value || undefined
    }, currentSid.value || getCurrentStudentId())
    resetDraft()
    uni.showToast({ title: res.data?.message || "已提交，等待老师回应", icon: "none" })
    // 成功后重拉详情与历史：当前提交、公开入口、历史时间轴一起更新。
    await loadData(detail.value.id, { keepView: true })
  } catch (err) {
    uni.showToast({ title: err?.message || "打卡失败", icon: "none" })
  } finally {
    submitting.value = false
  }
}

async function quickCheckin() {
  if (submitting.value || !detail.value.id || !showCheckinForm.value) return
  submitting.value = true
  try {
    const res = await checkinTask(detail.value.id, { content: "" }, currentSid.value || getCurrentStudentId())
    resetDraft()
    uni.showToast({ title: res.data?.message || "已提交，等待老师回应", icon: "none" })
    await loadData(detail.value.id, { keepView: true })
  } catch (err) {
    uni.showToast({ title: err?.message || "提交失败", icon: "none" })
  } finally {
    submitting.value = false
  }
}

async function react(item, reaction) {
  if (!item?.id || reactingIds.value.includes(item.id)) return
  reactingIds.value = [...reactingIds.value, item.id]
  try {
    const res = await reactResponse(item.id, reaction, currentSid.value || getCurrentStudentId())
    // 同一家长再次提交是更新自己那一行，直接回显服务端结果。
    item.my_reaction = res.data?.reaction || reaction
    uni.showToast({ title: "已反馈给老师", icon: "none" })
  } catch (err) {
    uni.showToast({ title: err?.message || "反馈失败", icon: "none" })
  } finally {
    reactingIds.value = reactingIds.value.filter((id) => id !== item.id)
  }
}

function applyPublicResult(row, data) {
  if (!data) return
  const payload = {
    is_public_requested: Boolean(data.is_public_requested),
    public_scope_confirmed: Boolean(data.public_scope_confirmed),
    public_approved_at: data.public_approved_at || null,
    public_revoked_at: data.public_revoked_at || null,
    public_visible: Boolean(data.public_visible),
    public_status: data.public_status || undefined
  }
  Object.assign(row, payload)
  // 当前提交与历史里的同一条可能不是同一个对象引用：两处一起回显。
  if (currentCheckin.value && currentCheckin.value.id === row.id && currentCheckin.value !== row) {
    Object.assign(currentCheckin.value, payload)
  }
  const hit = history.value.find((h) => h.id === row.id)
  if (hit && hit !== row) Object.assign(hit, payload)
}

async function togglePublic(row) {
  if (!row?.id || publicBusyIds.value.includes(row.id)) return
  const state = publicStateOf(row)
  const cancelling = isRevoking(state)
  if (cancelling && pendingCancelId.value !== row.id) {
    // 第一次点只进入确认态，不碰接口；换一条操作时上一条退回普通态。
    pendingCancelId.value = row.id
    uni.showToast({ title: "再点一次确认撤下", icon: "none" })
    return
  }
  pendingCancelId.value = 0
  publicBusyIds.value = [...publicBusyIds.value, row.id]
  try {
    const sid = currentSid.value || getCurrentStudentId()
    const res = cancelling
      ? await cancelPublicCheckin(row.id, sid)
      : await requestPublicCheckin(row.id, sid)
    applyPublicResult(row, res.data)
    uni.showToast({
      title: cancelling ? "已撤下" : "已申请，等待老师审核",
      icon: "none"
    })
  } catch (err) {
    // 422 的原文已经说清「为什么不能公开」，直接透出比再包一层更准确。
    uni.showToast({ title: err?.message || "操作失败，请稍后重试", icon: "none" })
  } finally {
    publicBusyIds.value = publicBusyIds.value.filter((id) => id !== row.id)
  }
}

function previewImage(url) {
  if (!url) return
  uni.previewImage({ urls: [url], current: url })
}

function openKidMode(item) {
  if (!item) return
  kidTarget.value = item
}

function closeKidMode() {
  stopVoice()
  kidTarget.value = null
}

function playVoice(url) {
  if (!url) return
  if (playingUrl.value === url) {
    stopVoice()
    return
  }
  stopVoice()
  audio = uni.createInnerAudioContext()
  audio.src = url
  audio.onEnded(() => {
    playingUrl.value = ""
  })
  audio.onError(() => {
    playingUrl.value = ""
    uni.showToast({ title: "语音播放失败", icon: "none" })
  })
  audio.play()
  playingUrl.value = url
}

function stopVoice() {
  if (audio) {
    audio.stop()
    audio.destroy()
    audio = null
  }
  playingUrl.value = ""
}

function focusCheckin() {
  inputFocus.value = true
  uni.pageScrollTo({ selector: "#checkinCard", duration: 200 })
}

onLoad((opts) => {
  linkedStudentId.value = Number(opts?.student_id || 0)
  // 先落一次当前孩子：任务对当前孩子不可见时详情会 404，
  // 「消息是另一个孩子的」提示不能因为加载失败就消失。
  currentSid.value = getCurrentStudentId() || 0
  const id = Number(opts?.id || 0) || parseSceneTaskId(opts?.scene)
  if (!id) {
    loading.value = false
    loadError.value = "任务已结束或不存在"
    return
  }
  loadData(id)
})

onUnload(stopVoice)
</script>

<template>
  <PageShell bottom-space="48rpx" max-width="430px">
    <!-- 提示放在加载状态之外：任务对当前孩子不可见时会 404，这条出口不能跟着消失。 -->
    <view v-if="childMismatch" class="mismatch">
      <text class="mismatch-text">这条消息是另一个孩子的任务，当前看到的是「{{ childLabel }}」</text>
      <view class="mismatch-btn" @click="switchToMessageChild">
        <text class="mismatch-btn-text">切换查看</text>
      </view>
    </view>

    <!-- ===== 加载骨架：3 块 ===== -->
    <view v-if="loading" class="skeleton">
      <view class="sk-card">
        <view class="sk sk-pill" />
        <view class="sk sk-line sk-w-60" />
        <view class="sk sk-line sk-w-90" />
        <view class="sk sk-line sk-w-40" />
        <view class="sk-grid">
          <view class="sk sk-mini" />
          <view class="sk sk-mini" />
          <view class="sk sk-mini" />
        </view>
      </view>
      <view class="sk-card">
        <view class="sk sk-line sk-w-30" />
        <view class="sk sk-line sk-w-80" />
        <view class="sk sk-shot" />
      </view>
      <view class="sk-card">
        <view class="sk sk-line sk-w-25" />
        <view class="sk sk-line sk-w-70" />
      </view>
    </view>

    <!-- ===== 失败 / 未绑定 ===== -->
    <view v-else-if="loadError" class="error-card">
      <view class="error-orb">
        <text class="error-icon">{{ ICON.alert }}</text>
      </view>
      <text class="error-title">{{ errorTitle }}</text>
      <text class="error-desc">{{ errorDesc }}</text>
      <view class="retry-btn" :class="{ 'is-busy': retrying }" @click="handleRetry">
        <text>{{ retrying ? "重试中…" : "重试" }}</text>
      </view>
    </view>

    <view v-else class="content">
      <!-- ===== 任务说明 ===== -->
      <view class="card">
        <view class="task-topline">
          <text class="freq-pill">{{ frequencyTaskLabel(detail.frequency) }}</text>
          <text class="status-chip" :class="'chip--' + statusChip.tone">{{ statusChip.label }}</text>
        </view>
        <text class="task-title">{{ detail.title }}</text>
        <text v-if="detail.description" class="task-desc">{{ detail.description }}</text>
        <text v-if="detail.intro" class="task-intro">{{ detail.intro }}</text>

        <view v-if="requirements.length" class="reqs">
          <view v-for="(req, idx) in requirements" :key="idx" class="req">
            <text class="req-num">{{ idx + 1 }}</text>
            <text class="req-text">{{ req }}</text>
          </view>
        </view>

        <view class="meta-grid">
          <view class="meta-item">
            <text class="meta-label">截止时间</text>
            <text class="meta-value">{{ formatDateLabel(detail.end_date) || "—" }}</text>
          </view>
          <view class="meta-item">
            <text class="meta-label">星星罐</text>
            <text class="meta-value meta-value--accent">{{ starJarValue(detail) }}</text>
          </view>
        </view>

        <view v-if="detail.learning_link" class="link-box">
          <text class="link-label">学习关联</text>
          <text class="link-text">{{ detail.learning_link }}</text>
        </view>
      </view>

      <!-- ===== 未读悄悄话 ===== -->
      <view v-if="hadUnread" class="whisper">
        <text class="whisper-glyph">{{ ICON.chat }}</text>
        <text class="whisper-text">老师又给你留了一句悄悄话，往下面看 👇</text>
      </view>

      <!-- ===== 老师回应：与本周期是否提交、公开是否通过相互独立 ===== -->
      <view v-if="responses.length" class="card feedback">
        <view class="feedback-head">
          <text class="section-title">{{ feedbackTitle }}</text>
          <text class="feedback-hint">共 {{ feedbackCount }} 条</text>
        </view>

        <view v-for="resp in responses" :key="resp.id" class="resp">
          <view class="resp-head">
            <view class="resp-glyph">
              <text class="resp-glyph-text">{{ responseGlyph(resp) }}</text>
            </view>
            <view class="resp-meta">
              <text class="resp-tag">{{ ritualLabel(resp.ritual) }}</text>
              <text class="resp-time">{{ formatCommentTime(resp.created_at) }}</text>
            </view>
          </view>

          <text v-if="resp.content" class="resp-content">{{ resp.content }}</text>

          <view
            v-if="resp.media_url"
            class="voice"
            :class="{ 'is-playing': playingUrl === resp.media_url }"
            @click="playVoice(resp.media_url)"
          >
            <text class="voice-glyph">{{ ICON.play }}</text>
            <text class="voice-text">{{ playingUrl === resp.media_url ? "正在播放…" : "播放老师的语音" }}</text>
          </view>

          <view class="resp-foot">
            <view class="link-btn" @click="openKidMode(resp)">
              <text>给孩子看</text>
            </view>
          </view>

          <view class="reactions">
            <view
              v-for="reaction in REACTIONS"
              :key="reaction.key"
              class="reaction"
              :class="{
                'is-active': resp.my_reaction === reaction.key,
                'is-busy': reactingIds.includes(resp.id)
              }"
              @click="react(resp, reaction.key)"
            >
              <text class="reaction-text">{{ reaction.label }}</text>
            </view>
          </view>

          <!-- 补充回应：每条单独反馈，与主回应各自一个 my_reaction -->
          <view v-for="sup in (resp.supplements || [])" :key="sup.id" class="supplement">
            <view class="sup-head">
              <text class="sup-label">老师又补了一句</text>
              <text class="sup-time">{{ formatCommentTime(sup.created_at) }}</text>
            </view>

            <view v-if="sup.content" class="sup-content-row">
              <text v-if="sup.ritual === 'sticker'" class="sup-glyph">{{ stickerGlyph(sup.sticker_key) }}</text>
              <text class="sup-content">{{ sup.content }}</text>
            </view>

            <view
              v-if="sup.media_url"
              class="voice voice--nested"
              :class="{ 'is-playing': playingUrl === sup.media_url }"
              @click="playVoice(sup.media_url)"
            >
              <text class="voice-glyph">{{ ICON.play }}</text>
              <text class="voice-text">{{ playingUrl === sup.media_url ? "正在播放…" : "播放老师的语音" }}</text>
            </view>

            <view class="resp-foot">
              <view class="link-btn" @click="openKidMode(sup)">
                <text>给孩子看</text>
              </view>
            </view>

            <view class="reactions">
              <view
                v-for="reaction in REACTIONS"
                :key="reaction.key"
                class="reaction"
                :class="{
                  'is-active': sup.my_reaction === reaction.key,
                  'is-busy': reactingIds.includes(sup.id)
                }"
                @click="react(sup, reaction.key)"
              >
                <text class="reaction-text">{{ reaction.label }}</text>
              </view>
            </view>
          </view>
        </view>
      </view>

      <!-- ===== 我的提交：当前孩子最近一次提交 + 公开状态 ===== -->
      <view v-if="showSubmitCard" class="card submit-card">
        <view class="submit-head">
          <text class="section-title">我的提交</text>
          <text class="submit-date">{{ formatRecordTime(currentCheckin.date, currentCheckin.created_at) }}</text>
        </view>
        <text class="submit-text" :class="{ 'is-quiet': !currentCheckin.content }">
          {{ currentCheckin.content || "（一键完成）" }}
        </text>
        <image
          v-if="currentCheckin.media_url"
          class="submit-shot"
          :src="currentCheckin.media_url"
          mode="aspectFill"
          @click="previewImage(currentCheckin.media_url)"
        />
        <view class="public-row">
          <text v-if="currentPublicBadge" class="badge" :class="'badge--' + currentPublicBadge.tone">
            {{ currentPublicBadge.label }}
          </text>
          <text v-if="currentPublicNote" class="public-note">{{ currentPublicNote }}</text>
          <view
            v-if="showCurrentPublicAction"
            class="link-btn public-action"
            :class="{ 'is-busy': publicBusyIds.includes(currentCheckin.id) }"
            @click="togglePublic(currentCheckin)"
          >
            <text>{{ currentPublicActionLabel }}</text>
          </view>
        </view>
      </view>

      <!-- ===== 今日提交 ===== -->
      <!-- 给孩子看打开时移除原生 textarea，避免老基础库的原生组件盖住全屏遮罩 -->
      <view v-if="showCheckinForm && !kidTarget" id="checkinCard" class="card checkin-card">
        <view class="checkin-head">
          <text class="section-title">今日提交</text>
          <text class="checkin-hint">{{ checkinHint }}</text>
        </view>

        <view class="upload-row">
          <view v-if="!mediaPreview" class="upload-tile" @click="chooseMedia">
            <text class="upload-plus">{{ ICON.add }}</text>
            <text class="upload-label">添加照片</text>
          </view>
          <view v-else class="upload-preview">
            <image class="upload-image" :src="mediaPreview" mode="aspectFill" />
            <view v-if="!uploading" class="upload-remove" @click.stop="removeMedia">
              <text class="upload-remove-glyph">{{ ICON.close }}</text>
            </view>
            <view v-else class="upload-loading">
              <text class="upload-loading-text">上传中…</text>
            </view>
          </view>
          <view class="upload-copy">
            <text class="upload-title">照片（可选）</text>
            <text class="upload-note">每次提交支持 1 张照片</text>
            <view class="ghost-btn" :class="{ 'is-busy': submitting }" @click="quickCheckin">
              <text>今天完成了（一键提交）</text>
            </view>
          </view>
        </view>

        <textarea
          v-model="checkinContent"
          class="checkin-input"
          maxlength="500"
          placeholder="记录今天的内容..."
          :focus="inputFocus"
          :disabled="submitting"
          @blur="inputFocus = false"
        />

        <view class="checkin-foot">
          <text class="checkin-note">{{ canSubmit ? "可以提交了" : "填写内容或上传照片后可提交" }}</text>
          <view
            class="checkin-btn"
            :class="{ 'is-disabled': !canSubmit, 'is-busy': submitting }"
            @click="handleCheckin"
          >
            <text>{{ submitting ? "提交中…" : "打卡" }}</text>
          </view>
        </view>
      </view>

      <!-- ===== 打卡历史：时间轴 + 公开四态 ===== -->
      <view class="history">
        <view class="section-head">
          <text class="section-title">打卡历史</text>
        </view>

        <view v-if="visibleHistory.length" class="timeline">
          <view
            v-for="(item, idx) in visibleHistory"
            :key="item.id"
            class="tl-item"
            :class="{ 'is-latest': idx === 0 }"
          >
            <view class="tl-head">
              <text class="tl-date">{{ formatMonthDayWeekday(item.date) }}</text>
              <text v-if="historyBadge(item)" class="badge" :class="'badge--' + historyBadge(item).tone">
                {{ historyBadge(item).label }}
              </text>
            </view>
            <text class="tl-text" :class="{ 'is-quiet': !item.content }">{{ item.content || "（一键完成）" }}</text>
            <view v-if="item.media_url || historyNote(item) || showHistoryAction(item)" class="tl-body">
              <image
                v-if="item.media_url"
                class="tl-thumb"
                :src="item.media_url"
                mode="aspectFill"
                @click="previewImage(item.media_url)"
              />
              <view class="tl-foot">
                <text v-if="historyNote(item)" class="public-note">{{ historyNote(item) }}</text>
                <view
                  v-if="showHistoryAction(item)"
                  class="link-btn tl-action"
                  :class="{ 'is-busy': publicBusyIds.includes(item.id) }"
                  @click="togglePublic(item)"
                >
                  <text>{{ historyActionLabel(item) }}</text>
                </view>
              </view>
            </view>
          </view>
        </view>

        <text v-if="visibleHistory.length && showHistoryMore" class="history-more">
          仅显示最近 5 条提交（共 {{ historyTotal }} 条）
        </text>

        <view v-if="!visibleHistory.length" class="empty-history">
          <view class="empty-icon">
            <text class="empty-glyph">{{ ICON.check }}</text>
          </view>
          <text class="empty-title">还没有打卡记录</text>
          <text class="empty-desc">今天完成了就记一笔，老师会看到的</text>
          <view v-if="showCheckinForm" class="primary-btn empty-cta" @click="focusCheckin">
            <text>去打卡</text>
          </view>
        </view>
      </view>
    </view>

    <!-- ===== 给孩子看：全屏遮罩，轻触空白处关闭 ===== -->
    <view v-if="kidTarget" class="kid-mask" @click="closeKidMode">
      <view class="kid-body" @click.stop>
        <text class="kid-glyph">{{ responseGlyph(kidTarget) }}</text>
        <text v-if="kidTarget.content" class="kid-text">{{ kidTarget.content }}</text>
        <view
          v-if="kidTarget.media_url"
          class="kid-play"
          :class="{ 'is-playing': playingUrl === kidTarget.media_url }"
          @click="playVoice(kidTarget.media_url)"
        >
          <text class="kid-play-glyph">{{ ICON.play }}</text>
          <text class="kid-play-text">{{ playingUrl === kidTarget.media_url ? "正在播放…" : "听老师说" }}</text>
        </view>
        <text class="kid-hint">轻触空白处关闭</text>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 一次提交的完整回路：白卡 + 冷雾底；暖橙只落在主按钮、
// 选中反馈、要求数字与时间轴最近圆点。
// ============================================================

// ---- 多孩提示（独立于加载状态） ------------------------------
.mismatch {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  padding: 20rpx $spacing-lg;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-semantic-warning-bg;
}

.mismatch-text {
  flex: 1;
  min-width: 0;
  color: $color-semantic-warning-strong;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  overflow-wrap: break-word;
}

// 白字压 --warning 只有 2.9:1，改用警告深底 + 白字（约 8:1）。
.mismatch-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  min-width: 176rpx;
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-semantic-warning-strong;

  &:active {
    transform: scale(0.98);
  }
}

.mismatch-btn-text {
  color: $color-text-on-brand;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
}

// ---- 通用块 --------------------------------------------------
.card {
  padding: $spacing-xl;
  margin-bottom: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.section-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

// ---- 任务说明 ------------------------------------------------
.task-topline {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-sm;
}

.freq-pill {
  display: inline-flex;
  align-items: center;
  min-height: 44rpx;
  padding: 0 $component-badge-padding-x;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.status-chip {
  display: inline-flex;
  align-items: center;
  min-height: 44rpx;
  padding: 0 $component-badge-padding-x;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.chip--todo {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.chip--wait {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
}

.chip--ok {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.chip--ended {
  background: $color-border-subtle;
  color: $color-text-body;
}

.task-title {
  display: block;
  margin-top: $spacing-md;
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  overflow-wrap: break-word;
}

.task-desc {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.task-intro {
  display: block;
  margin-top: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.reqs {
  margin-top: $spacing-lg;
}

.req {
  display: flex;
  align-items: flex-start;
  gap: $spacing-md;
}

.req + .req {
  margin-top: $spacing-sm;
}

.req-num {
  flex: none;
  width: 32rpx;
  height: 32rpx;
  margin-top: 2rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: 20rpx;
  font-weight: $typography-font-weight-semibold;
  line-height: 32rpx;
  text-align: center;
}

.req-text {
  flex: 1;
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: break-word;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: $spacing-sm;
  margin-top: $spacing-lg;
}

.meta-item {
  min-width: 0;
  padding: $spacing-md;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.meta-label {
  display: block;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.meta-value {
  display: block;
  margin-top: 4rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.meta-value--accent {
  color: $color-brand-primary-dark;
}

.link-box {
  margin-top: $spacing-md;
  padding: 20rpx $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-brand-primary-bg;
}

.link-label {
  display: block;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
}

.link-text {
  display: block;
  margin-top: 4rpx;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: break-word;
}

// ---- 未读悄悄话 ----------------------------------------------
.whisper {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  padding: $spacing-md $spacing-lg;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-semantic-warning-bg;
}

.whisper-glyph {
  flex: none;
  color: $color-semantic-warning-strong;
  font-family: remixicon;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}

.whisper-text {
  flex: 1;
  min-width: 0;
  color: $color-semantic-warning-strong;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

// ---- 老师回应 ------------------------------------------------
.feedback-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-sm;
}

.feedback-hint {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.resp {
  margin-top: 20rpx;
  padding: $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}

.resp + .resp {
  margin-top: $spacing-md;
}

.resp-head {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.resp-glyph {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.resp-glyph-text {
  font-size: $typography-font-size-3xl;
  line-height: 1;
}

.resp-meta {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}

.resp-tag {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
}

.resp-time {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.resp-content {
  display: block;
  margin-top: $spacing-md;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.voice {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  min-height: 88rpx;
  margin-top: $spacing-md;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;

  &:active {
    transform: scale(0.99);
  }
}

.voice-glyph {
  flex: none;
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

.voice-text {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
}

// 播放态用整条品牌色 + 深色字，避免白字压橙的低对比。
.voice.is-playing {
  background: $color-brand-primary;
}

.voice.is-playing .voice-glyph,
.voice.is-playing .voice-text {
  color: $color-text-primary;
}

.voice--nested {
  background: $color-bg-card;
}

.resp-foot {
  display: flex;
  justify-content: flex-end;
  margin-top: 4rpx;
}

.link-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  padding: 0 $spacing-xs;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;

  &:active {
    opacity: 0.72;
  }
}

.link-btn.is-busy {
  opacity: 0.5;
}

.reactions {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
  margin-top: $spacing-sm;
}

.reaction {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  min-height: 88rpx;
  padding: 0 4rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;

  &:active {
    transform: scale(0.98);
  }
}

.reaction-text {
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.3;
  text-align: center;
}

// 选中态用品牌橙底 + 深色字：白字压橙不足 4.5:1。
.reaction.is-active {
  border-color: $color-brand-primary;
  background: $color-brand-primary;
}

.reaction.is-active .reaction-text {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

.reaction.is-busy {
  opacity: 0.6;
}

.supplement {
  margin-top: 20rpx;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
}

.sup-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 6rpx;
}

.sup-label {
  color: $color-semantic-info;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
}

.sup-time {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.sup-content-row {
  display: flex;
  align-items: flex-start;
  gap: 6rpx;
  margin-top: 6rpx;
}

.sup-glyph {
  flex: none;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}

.sup-content {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: break-word;
}

// ---- 我的提交 ------------------------------------------------
.submit-head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-sm;
}

.submit-date {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.submit-text {
  display: block;
  margin-top: $spacing-md;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.submit-text.is-quiet {
  color: $color-text-muted;
}

.submit-shot {
  display: block;
  width: 100%;
  height: 336rpx;
  margin-top: $spacing-md;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
}

.public-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-sm;
  margin-top: 20rpx;
}

.badge {
  display: inline-flex;
  align-items: center;
  flex: none;
  min-height: 40rpx;
  padding: 0 $spacing-md;
  border-radius: $component-badge-radius;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.badge--ok {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.badge--pending {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
}

.badge--muted {
  background: $color-border-subtle;
  color: $color-text-body;
}

.public-note {
  flex: 1;
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  overflow-wrap: break-word;
}

.public-action {
  margin-left: auto;
}

// ---- 今日提交 ------------------------------------------------
.checkin-head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 6rpx;
}

.checkin-hint {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.upload-row {
  display: grid;
  grid-template-columns: 192rpx minmax(0, 1fr);
  gap: $spacing-lg;
  align-items: center;
  margin-top: $spacing-md;
}

.upload-tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  width: 192rpx;
  height: 192rpx;
  border: 3rpx dashed $color-brand-primary-light;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;

  &:active {
    transform: scale(0.98);
  }
}

.upload-plus {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.upload-label {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  line-height: 1.2;
}

.upload-preview {
  position: relative;
  width: 192rpx;
  height: 192rpx;
  overflow: hidden;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
}

.upload-image {
  display: block;
  width: 100%;
  height: 100%;
}

// 视觉圆点 56rpx，点击区补到 88rpx（≥44px）后再贴右上角。
.upload-remove {
  position: absolute;
  top: 0;
  right: 0;
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  width: 88rpx;
  height: 88rpx;
  padding: $spacing-xs;
}

.upload-remove-glyph {
  display: block;
  width: 56rpx;
  height: 56rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  line-height: 56rpx;
  text-align: center;
}

// 上传遮罩是单次使用的半透明层，无对应 token；按 token 规范局部硬编码并说明。
.upload-loading {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  padding: $spacing-xs 0;
  background: rgba(0, 0, 0, 0.45);
  text-align: center;
}

.upload-loading-text {
  color: $color-text-on-brand;
  font-size: $typography-font-size-xs;
}

.upload-copy {
  min-width: 0;
}

.upload-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.upload-note {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.ghost-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: $spacing-md;
  padding: 0 28rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;

  &:active {
    transform: scale(0.98);
  }
}

.ghost-btn.is-busy {
  opacity: 0.6;
}

.ghost-btn text {
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
}

.checkin-input {
  width: 100%;
  min-height: 192rpx;
  margin-top: $spacing-lg;
  padding: 20rpx $spacing-lg;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.checkin-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  margin-top: 20rpx;
}

.checkin-note {
  flex: 1;
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.checkin-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  min-width: 176rpx;
  min-height: 88rpx;
  padding: 0 40rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;

  &:active {
    transform: scale(0.98);
  }
}

.checkin-btn text {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
}

.checkin-btn.is-disabled {
  background: $color-bg-page;
  border: 1rpx solid $color-border-subtle;

  &:active {
    transform: none;
  }
}

.checkin-btn.is-disabled text {
  color: $color-text-hint;
  font-weight: $typography-font-weight-medium;
}

.checkin-btn.is-busy {
  opacity: 0.7;
}

// ---- 打卡历史 ------------------------------------------------
.history {
  margin-top: $spacing-xl;
}

.section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
}

.timeline {
  position: relative;
  margin-top: 20rpx;
  padding-left: 36rpx;
}

.timeline::before {
  content: "";
  position: absolute;
  top: 8rpx;
  bottom: 10rpx;
  left: 8rpx;
  width: 1rpx;
  background: $color-border-hairline;
}

.tl-item {
  position: relative;
  padding-bottom: 28rpx;
  min-width: 0;
}

.tl-item::before {
  content: "";
  position: absolute;
  top: 6rpx;
  left: -28rpx;
  width: 18rpx;
  height: 18rpx;
  border: 4rpx solid $color-bg-page;
  border-radius: $border-radius-full;
  background: $color-text-muted;
}

.tl-item.is-latest::before {
  background: $color-brand-primary;
}

.tl-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-sm;
}

.tl-date {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

.tl-text {
  display: block;
  margin-top: 6rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: break-word;
}

.tl-text.is-quiet {
  color: $color-text-muted;
}

.tl-body {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  margin-top: $spacing-sm;
}

.tl-thumb {
  flex: none;
  width: 112rpx;
  height: 112rpx;
  border-radius: $border-radius-md;
  background: $color-brand-primary-bg;
}

.tl-foot {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-sm;
  min-width: 0;
}

.tl-action {
  flex: none;
  margin-left: auto;
}

.history-more {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
}

.empty-history {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 20rpx;
  padding: 56rpx $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 104rpx;
  height: 104rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.empty-glyph {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: 44rpx;
  line-height: 1;
}

.empty-title {
  margin-top: 20rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.empty-desc {
  margin-top: 6rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.primary-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 176rpx;
  min-height: 88rpx;
  padding: 0 40rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;

  &:active {
    transform: scale(0.98);
  }
}

.primary-btn text {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
}

.empty-cta {
  margin-top: $spacing-lg;
}

// ---- 骨架 ----------------------------------------------------
.skeleton {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.sk-card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sk {
  display: block;
  background: $color-bg-page;
  border-radius: $border-radius-lg;
  animation: sk-breathe 1.3s $animation-easing-default infinite alternate;
}

.sk-pill {
  width: 128rpx;
  height: 40rpx;
  margin-bottom: $spacing-md;
  border-radius: $border-radius-full;
}

.sk-line {
  height: 24rpx;
  margin-bottom: $spacing-sm;
}

.sk-w-90 { width: 90%; }
.sk-w-80 { width: 80%; }
.sk-w-70 { width: 70%; }
.sk-w-60 { width: 60%; }
.sk-w-40 { width: 40%; }
.sk-w-30 { width: 30%; }
.sk-w-25 { width: 25%; }

.sk-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
  margin-top: $spacing-md;
}

.sk-mini {
  height: 88rpx;
}

.sk-shot {
  height: 280rpx;
  margin-top: $spacing-md;
  border-radius: $border-radius-xl;
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

// ---- 失败 / 未绑定 -------------------------------------------
.error-card {
  width: 100%;
  padding: 56rpx 40rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.error-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  margin: 0 auto;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
}

.error-icon {
  color: $color-semantic-danger-strong;
  font-family: remixicon;
  font-size: $typography-font-size-3xl;
  line-height: 1;
}

.error-title {
  display: block;
  margin-top: $spacing-md;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.error-desc {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  overflow-wrap: break-word;
}

.retry-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 176rpx;
  min-height: 88rpx;
  margin: $spacing-lg auto 0;
  padding: 0 56rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;

  &:active {
    transform: scale(0.98);
  }
}

.retry-btn text {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
}

.retry-btn.is-busy {
  opacity: 0.7;
}

// ---- 给孩子看 ------------------------------------------------
// 全屏遮罩是单次使用的半透明层，无对应 token；按 token 规范局部硬编码并说明。
.kid-mask {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48rpx;
  background: rgba(0, 0, 0, 0.72);
}

.kid-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: 600rpx;
  padding: 64rpx 48rpx;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
  text-align: center;
}

.kid-glyph {
  font-size: 128rpx;
  line-height: 1.1;
}

.kid-text {
  margin-top: $spacing-lg;
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.5;
  overflow-wrap: break-word;
}

.kid-play {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  min-height: 88rpx;
  margin-top: $spacing-lg;
  padding: 0 48rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;

  &:active {
    transform: scale(0.98);
  }
}

.kid-play-glyph,
.kid-play-text {
  color: $color-text-primary;
  line-height: 1.2;
}

.kid-play-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-md;
}

.kid-play-text {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.kid-play.is-playing {
  background: $color-brand-primary-light;
}

.kid-hint {
  margin-top: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
  }

  .mismatch-btn,
  .voice,
  .link-btn,
  .reaction,
  .upload-tile,
  .ghost-btn,
  .checkin-btn,
  .primary-btn,
  .retry-btn,
  .kid-play {
    transition: none;
  }
}
</style>
