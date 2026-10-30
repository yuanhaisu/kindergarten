<script setup>
import { computed, ref } from "vue"
import { onLoad, onShow, onUnload } from "@dcloudio/uni-app"
import {
  fetchTeacherGrowthLog,
  fetchTeacherGrowthComments,
  createTeacherGrowthComment,
  fetchGrowthTaggingStatus,
  reviewMediaSuggestion,
  confirmHighSuggestions,
  fetchPresentStudents,
  applyClassRoster,
  likeTeacherGrowthComment,
  unlikeTeacherGrowthComment
} from "@/api/teacher-growth"
import { fetchTeacherClassStudents } from "@/api/teacher-class"
import { fetchTeacherAchievements, grantAchievement } from "@/api/teacher-achievements"
import { ensureLogin } from "@/utils/login"
import { formatCommentTime, formatRecordTime } from "@/utils/time"
import { relationLabel } from "@/utils/relation"
import PageShell from "@/components/PageShell.vue"

const record = ref({})
const comments = ref([])
const commentTotal = ref(0)
const commentText = ref("")
const replyTo = ref(null)
const inputFocused = ref(false)
const loading = ref(true)
const hasLoaded = ref(false)
const loadError = ref("")
const submitting = ref(false)

const isInitialLoading = computed(() => loading.value && !hasLoaded.value)
const showError = computed(() => Boolean(loadError.value) && !hasLoaded.value)
const locked = computed(() => isInitialLoading.value || showError.value)

const creatorName = computed(() => record.value.creator?.nickname || "老师")
const creatorInitial = computed(() => String(creatorName.value).slice(0, 1) || "师")
const recordTime = computed(() => formatRecordTime(record.value.date, record.value.created_at))

const tags = computed(() =>
  Array.isArray(record.value.performance_tags)
    ? record.value.performance_tags.filter(Boolean)
    : []
)

const paragraphs = computed(() =>
  String(record.value.content || "")
    .split(/\n+/)
    .map(text => text.trim())
    .filter(Boolean)
)

const media = computed(() => (Array.isArray(record.value.media) ? record.value.media : []))
const isClassRecord = computed(() => record.value.target_type !== 2)

const relatedStudents = computed(() =>
  Array.isArray(record.value.related_students) ? record.value.related_students : []
)
const relatedCount = computed(() => record.value.related_students_count || relatedStudents.value.length)
const relatedShown = computed(() => relatedStudents.value.slice(0, 6))
const relatedMore = computed(() => Math.max(0, relatedCount.value - relatedShown.value.length))

function initialOf(user) {
  const name = String(user?.nickname || "")
  return name ? name.slice(0, 1) : "家"
}

function relationText(user) {
  // 9 码由共享枚举映射；未知值原样展示（与接送页同一口径，不猜测、不吞成「其他」）。
  return relationLabel(user?.relation)
}

// ---- 照片点名 / AI 建议 ----
const students = ref([])
const studentNameById = computed(() => {
  const map = {}
  students.value.forEach(student => {
    map[student.student_id] = student.name
  })
  return map
})

function namedNames(mediaItem) {
  return (mediaItem.student_ids || []).map(id => studentNameById.value[id] || `#${id}`)
}

function suggestionsOf(mediaItem) {
  return Array.isArray(mediaItem.student_suggestions) ? mediaItem.student_suggestions : []
}

const highSuggestionCount = computed(() =>
  media.value.reduce(
    (total, mediaItem) => total + suggestionsOf(mediaItem).filter(s => s.tier === "high").length,
    0
  )
)

const aiTouched = ref(false)
const aiGroups = computed(() =>
  media.value
    .map((mediaItem, index) => ({ media: mediaItem, index, items: suggestionsOf(mediaItem) }))
    .filter(group => group.items.length > 0)
)
const aiDoneVisible = computed(() => aiTouched.value && aiGroups.value.length === 0)

const groupPhotos = computed(() =>
  isClassRecord.value
    ? media.value
        .map((mediaItem, index) => ({ media: mediaItem, index }))
        .filter(group => group.media.is_group_photo)
    : []
)

const failedMedia = ref(new Set())

function markMediaFailed(index) {
  const next = new Set(failedMedia.value)
  next.add(index)
  failedMedia.value = next
}

function retryMedia(index) {
  const next = new Set(failedMedia.value)
  next.delete(index)
  failedMedia.value = next
}

// ---- 照片预览 ----
const previewOpen = ref(false)
const previewIndex = ref(0)
const previewMeta = computed(() => {
  const total = media.value.length
  if (!total) return ""
  return `第 ${previewIndex.value + 1} 张 / 共 ${total} 张`
})

function openPreview(index) {
  if (locked.value || !media.value.length) return
  previewIndex.value = index
  previewOpen.value = true
}

function closePreview() {
  previewOpen.value = false
}

function onPreviewChange(event) {
  const current = Number(event?.detail?.current)
  if (Number.isInteger(current)) previewIndex.value = current
}

function stepPreview(delta) {
  const total = media.value.length
  if (total < 2) return
  previewIndex.value = (previewIndex.value + delta + total) % total
}

// ---- AI 建议处理 ----
const confirmingHigh = ref(false)

async function confirmAllHigh() {
  if (confirmingHigh.value) return
  confirmingHigh.value = true
  try {
    const res = await confirmHighSuggestions(record.value.id)
    const count = res.data?.confirmed ?? 0
    media.value.forEach(mediaItem => {
      const highs = suggestionsOf(mediaItem).filter(s => s.tier === "high")
      if (!highs.length) return
      mediaItem.student_suggestions = suggestionsOf(mediaItem).filter(s => s.tier !== "high")
      mediaItem.student_ids = [...(mediaItem.student_ids || []), ...highs.map(s => s.student_id)]
    })
    aiTouched.value = true
    uni.showToast({ title: `已确认 ${count} 条高置信建议`, icon: "none" })
  } catch (err) {
    uni.showToast({ title: err?.message || "批量确认失败", icon: "none" })
    refreshTaggingStatus()
  } finally {
    confirmingHigh.value = false
  }
}

// 本地即时反馈 + 后台落库：确认 → 移入点名；拒绝 → 移除建议。失败则重取。
async function reviewSuggestion(mediaItem, suggestion, action) {
  try {
    await reviewMediaSuggestion(mediaItem.id, suggestion.student_id, action)
    const target = media.value.find(item => item.id === mediaItem.id)
    if (target) {
      target.student_suggestions = suggestionsOf(target).filter(s => s.student_id !== suggestion.student_id)
      if (action === "confirm") {
        target.student_ids = [...(target.student_ids || []), suggestion.student_id]
      }
    }
    aiTouched.value = true
    uni.showToast({ title: action === "confirm" ? "已确认点名" : "已忽略", icon: "none" })
  } catch (err) {
    uni.showToast({ title: err?.message || "操作失败", icon: "none" })
    refreshTaggingStatus()
  }
}

// 大合影「一键全班」：名单口径由服务端定（记录当日到场），客户端只消费回读结果。
const presentIds = ref([])
const presentLoaded = ref(false)
const rosterBusyId = ref(0)

async function ensurePresent() {
  if (presentLoaded.value) return
  presentLoaded.value = true
  try {
    const date = String(record.value.date || "").slice(0, 10)
    const res = await fetchPresentStudents(date)
    presentIds.value = res.data?.list || []
  } catch (err) {
    // 名单失败不阻断浏览；点击「全班」仍由服务端兜底
    presentLoaded.value = false
  }
}

async function applyRoster(mediaItem) {
  if (rosterBusyId.value) return
  rosterBusyId.value = mediaItem.id
  try {
    const res = await applyClassRoster(mediaItem.id)
    const count = res.data?.present ?? 0
    // 以服务端回读的最终名单为准（含「已拒绝不复活」）
    if (Array.isArray(res.data?.named)) mediaItem.student_ids = res.data.named
    uni.showToast({ title: `已点名 ${count} 人（当日到场）`, icon: "none" })
  } catch (err) {
    uni.showToast({ title: err?.message || "全班点名失败", icon: "none" })
  } finally {
    rosterBusyId.value = 0
  }
}

// ---- 识别状态轮询：4s 间隔；失败退避（4s→8s…→30s 封顶）；onUnload 停止 ----
const TAGGING_POLL_MS = 4000
const TAGGING_RETRY_MAX_MS = 30000
let taggingTimer = null
let taggingInFlight = false
let taggingFailures = 0

function applyTaggingStatus(data) {
  const byId = {}
  ;(data.media || []).forEach(item => {
    byId[item.id] = item
  })
  let hasGroupPhoto = false
  media.value.forEach(mediaItem => {
    const status = byId[mediaItem.id]
    if (!status) return
    mediaItem.student_suggestions = status.suggestions || []
    mediaItem.is_group_photo = !!status.is_group_photo
    if (mediaItem.is_group_photo) hasGroupPhoto = true
    if ((status.suggestions || []).length) aiTouched.value = true
  })
  if (hasGroupPhoto && isClassRecord.value) ensurePresent()
}

function scheduleTaggingPoll(delayMs) {
  stopTaggingPoll()
  taggingTimer = setTimeout(refreshTaggingStatus, delayMs)
}

async function refreshTaggingStatus() {
  if (!record.value.id || taggingInFlight) return
  stopTaggingPoll()
  taggingInFlight = true
  try {
    const res = await fetchGrowthTaggingStatus(record.value.id)
    const data = res.data || {}
    taggingFailures = 0
    applyTaggingStatus(data)
    if (data.pending) scheduleTaggingPoll(TAGGING_POLL_MS)
  } catch (err) {
    taggingFailures += 1
    const backoff = Math.min(TAGGING_POLL_MS * 2 ** (taggingFailures - 1), TAGGING_RETRY_MAX_MS)
    scheduleTaggingPoll(backoff)
  } finally {
    taggingInFlight = false
  }
}

function stopTaggingPoll() {
  if (taggingTimer) {
    clearTimeout(taggingTimer)
    taggingTimer = null
  }
}

onUnload(stopTaggingPoll)

// ---- 授徽章 ----
const achievements = ref([])
const grant = ref({ achievement_id: null, student_ids: [], as_witness: true, granting: false })
const grantOpen = ref(false)

const awardHint = computed(() => {
  if (grant.value.student_ids.length) {
    const names = grant.value.student_ids
      .map(id => students.value.find(s => s.student_id === id)?.name)
      .filter(Boolean)
    return `将授予 ${grant.value.student_ids.length} 位小朋友${names.length ? `：${names.join("、")}` : ""}`
  }
  return `未选择孩子，将默认授予本班全部 ${students.value.length} 人`
})

function selectAchievement(id) {
  grant.value.achievement_id = grant.value.achievement_id === id ? null : id
}

function toggleGrantStudent(id) {
  const list = grant.value.student_ids
  const index = list.indexOf(id)
  if (index >= 0) list.splice(index, 1)
  else list.push(id)
}

async function doGrant() {
  const current = grant.value
  if (!current.achievement_id) {
    uni.showToast({ title: "请先选一枚徽章", icon: "none" })
    return
  }
  const target = current.student_ids.length
    ? current.student_ids
    : students.value.map(student => student.student_id)
  if (!target.length) {
    uni.showToast({ title: "无可授予的学生", icon: "none" })
    return
  }
  current.granting = true
  try {
    await grantAchievement(current.achievement_id, {
      student_ids: target,
      log_id: current.as_witness ? record.value.id : undefined
    })
    uni.showToast({ title: "已授予徽章", icon: "success" })
    grantOpen.value = false
  } catch (err) {
    uni.showToast({ title: err?.message || "授予失败", icon: "none" })
  } finally {
    current.granting = false
  }
}

// ---- 评论 ----
const sendDisabled = computed(
  () => locked.value || submitting.value || !commentText.value.trim()
)

// 回复条出现时底栏会变高，页面预留高度同步增加，避免遮住最后一条评论
const bottomSpace = computed(() => (replyTo.value ? "240rpx" : "180rpx"))

// 空评论态必须与计数同口径：列表为空时标题与互动条都显示 0，
// 否则会出现「还没有家长评论」旁边写着「5 条评论」的矛盾。
const displayCommentTotal = computed(() => (comments.value.length ? commentTotal.value : 0))

const likePending = new Set()

function findComment(id) {
  for (const comment of comments.value) {
    if (comment.id === id) return comment
    const reply = (comment.replies || []).find(item => item.id === id)
    if (reply) return reply
  }
  return null
}

// 评论赞：不做乐观翻转，以服务端返回的 liked / likes_count 为准。
// 同一条评论在途时忽略后续点击（快速双击会并发两个请求，先返回的提示会误导）。
async function toggleCommentLike(comment) {
  if (locked.value || likePending.has(comment.id)) return
  likePending.add(comment.id)
  try {
    const res = comment.liked
      ? await unlikeTeacherGrowthComment(comment.id)
      : await likeTeacherGrowthComment(comment.id)
    const next = res?.data?.liked
    if (typeof next !== "boolean") throw new Error("点赞状态未更新")
    const target = findComment(comment.id)
    if (target) {
      target.liked = next
      if (typeof res.data.likes_count === "number") target.likes_count = res.data.likes_count
    }
  } catch (err) {
    uni.showToast({ title: err?.message || "操作失败", icon: "none" })
  } finally {
    likePending.delete(comment.id)
  }
}

function startReply(comment) {
  if (locked.value) return
  replyTo.value = { id: comment.id, name: comment.user?.nickname || "这条评论" }
}

function cancelReply() {
  replyTo.value = null
}

function focusComment() {
  if (locked.value) return
  inputFocused.value = true
}

function applyComments(res) {
  comments.value = res.data?.list || []
  const total = res.data?.total
  commentTotal.value = typeof total === "number" ? total : comments.value.length
  if (aiGroups.value.length) aiTouched.value = true
}

async function refreshComments() {
  try {
    const res = await fetchTeacherGrowthComments(record.value.id)
    applyComments(res)
  } catch (err) {
    console.warn("[growth-detail] refresh comments failed:", err?.message || err)
  }
}

async function submitComment() {
  if (sendDisabled.value) return
  const text = commentText.value.trim()
  if (!text) return
  submitting.value = true
  try {
    await createTeacherGrowthComment(record.value.id, text, replyTo.value ? replyTo.value.id : null)
    commentText.value = ""
    replyTo.value = null
    uni.showToast({ title: "评论成功", icon: "success" })
    await refreshComments()
  } catch (err) {
    // 失败保留输入内容；http.js 已提示错误
    console.warn("[growth-detail] post comment failed:", err?.message || err)
  } finally {
    submitting.value = false
  }
}

// ---- 加载 ----
async function loadStudents() {
  try {
    await ensureLogin()
    const res = await fetchTeacherClassStudents()
    students.value = res.data?.list || []
  } catch (err) {
    console.warn("[growth-detail] load students failed:", err?.message || err)
  }
}

async function loadAchievements() {
  try {
    const res = await fetchTeacherAchievements()
    achievements.value = res.data?.list || []
  } catch (err) {
    console.warn("[growth-detail] load achievements failed:", err?.message || err)
  }
}

async function loadDetail(id) {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const [logRes, commentRes] = await Promise.all([
      fetchTeacherGrowthLog(id),
      fetchTeacherGrowthComments(id)
    ])
    record.value = logRes.data || {}
    applyComments(commentRes)
    hasLoaded.value = true
    failedMedia.value = new Set()
    refreshTaggingStatus()
  } catch (err) {
    console.warn("[growth-detail] load failed:", err?.message || err)
    loadError.value = "加载失败，请重试"
  } finally {
    loading.value = false
  }
}

async function handleRetry() {
  if (loading.value) return
  await loadDetail(record.value.id)
  if (!loadError.value) uni.showToast({ title: "已重新加载", icon: "none" })
}

function goEdit() {
  uni.navigateTo({ url: `/pages/growth/create?id=${record.value.id}` })
}

function handleShare() {
  uni.showToast({ title: "点击右上角分享", icon: "none" })
}

onLoad(options => {
  const id = Number(options?.id)
  if (!id) {
    loading.value = false
    loadError.value = "加载失败，请重试"
    return
  }
  record.value = { id }
  loadDetail(id)
  loadStudents()
  loadAchievements()
})

// 从编辑页保存后返回时刷新正文/评论；首次 onShow 在 onLoad 之后触发，
// hasLoaded=false 时不重复请求。
onShow(() => {
  if (hasLoaded.value && record.value.id) loadDetail(record.value.id)
})
</script>

<template>
  <PageShell :bottom-space="bottomSpace">
    <view class="detail-page">
      <!-- 加载：整页骨架 -->
      <view v-if="isInitialLoading" class="skeleton">
        <view class="sk-card">
          <view class="sk-author">
            <view class="sk sk-avatar" />
            <view class="sk-lines">
              <view class="sk sk-line sk-line-short" />
              <view class="sk sk-line sk-line-mid" />
            </view>
          </view>
          <view class="sk sk-title" />
          <view class="sk sk-line" />
          <view class="sk sk-line" />
          <view class="sk-grid">
            <view v-for="i in 9" :key="i" class="sk sk-tile" />
          </view>
        </view>
        <view class="sk-card">
          <view class="sk sk-line sk-line-short" />
          <view v-for="i in 2" :key="i" class="sk-comment">
            <view class="sk sk-avatar" />
            <view class="sk-lines">
              <view class="sk sk-line sk-line-mid" />
              <view class="sk sk-line" />
            </view>
          </view>
        </view>
      </view>

      <!-- 失败：整页错误 + 重试 -->
      <view v-else-if="showError" class="error-view">
        <view class="error-icon"><text>!</text></view>
        <text class="error-text">加载失败，请重试</text>
        <view class="retry-load-btn" @click="handleRetry">重试</view>
      </view>

      <template v-else>
        <view class="record-card">
          <!-- 作者行 -->
          <view class="record-author">
            <view class="avatar">
              <image
                v-if="record.creator?.avatar"
                class="avatar-img"
                :src="record.creator.avatar"
                mode="aspectFill"
              />
              <text v-else>{{ creatorInitial }}</text>
            </view>
            <view class="author-info">
              <text class="author-name">{{ creatorName }}</text>
              <text class="author-time">发布于 {{ recordTime }}</text>
            </view>
          </view>

          <!-- 表现标签标题 -->
          <text v-if="tags.length" class="record-title">{{ tags.join(" · ") }}</text>

          <!-- 正文多段 -->
          <view v-if="paragraphs.length" class="record-body">
            <text v-for="(paragraph, index) in paragraphs" :key="index" class="record-paragraph">{{ paragraph }}</text>
          </view>

          <!-- 照片 -->
          <view v-if="media.length" class="media-section">
            <view class="media-head">
              <text class="section-title">照片</text>
              <text class="section-count">{{ media.length }} 张</text>
            </view>

            <view v-if="highSuggestionCount" class="ai-banner">
              <text class="ai-banner-text"><text class="ai-banner-strong">{{ highSuggestionCount }}</text> 条高置信建议</text>
              <view class="ai-banner-btn" @click="confirmAllHigh">{{ confirmingHigh ? "确认中…" : "全部确认" }}</view>
            </view>

            <view class="media-grid">
              <view v-for="(mediaItem, index) in media" :key="mediaItem.id || index" class="photo-cell">
                <view v-if="!failedMedia.has(index)" class="photo-open" @click="openPreview(index)">
                  <image
                    class="photo-img"
                    :src="mediaItem.thumb_url || mediaItem.file_url"
                    mode="aspectFill"
                    @error="markMediaFailed(index)"
                  />
                  <text v-if="mediaItem.is_group_photo" class="photo-flag">大合影</text>
                  <text v-else-if="suggestionsOf(mediaItem).length" class="photo-flag is-ai">AI 建议</text>
                  <view v-if="namedNames(mediaItem).length" class="photo-caption">
                    <text class="photo-caption-label">点名</text>
                    <text
                      v-for="name in namedNames(mediaItem)"
                      :key="name"
                      class="photo-kid"
                    >{{ name }}</text>
                  </view>
                </view>
                <view v-else class="photo-fail" @click="retryMedia(index)">
                  <text class="photo-fail-text">加载失败</text>
                  <text class="photo-fail-retry">重试</text>
                </view>
              </view>
            </view>

            <!-- 大合影 · 一键全班（仅全班记录；名单口径由服务端定） -->
            <view
              v-for="group in groupPhotos"
              :key="'group-' + group.media.id"
              class="group-photo-btn"
              :class="{ 'is-busy': rosterBusyId === group.media.id }"
              @click="applyRoster(group.media)"
            >
              <view class="group-photo-copy">
                <text class="group-photo-title">大合影 · 一键全班</text>
                <text class="group-photo-sub">
                  {{ presentLoaded ? `当日到场 ${presentIds.length} 人` : "当日到场名单待取" }}
                </text>
              </view>
            </view>

            <!-- AI 识别建议：按张分组 -->
            <view v-for="group in aiGroups" :key="'ai-' + group.media.id" class="ai-card">
              <view class="ai-card-head">
                <text class="ai-card-title">AI 识别建议</text>
                <text class="ai-card-ref">第 {{ group.index + 1 }} 张</text>
              </view>
              <view v-for="suggestion in group.items" :key="suggestion.student_id" class="ai-row">
                <view class="ai-avatar">{{ String(suggestion.name || "?").slice(0, 1) }}</view>
                <view class="ai-person">
                  <text class="ai-name">{{ suggestion.name }}</text>
                  <text
                    class="status"
                    :class="suggestion.tier === 'high' ? 'status-ok' : 'status-pending'"
                  >{{ suggestion.tier === "high" ? "高置信" : "待确认" }}</text>
                </view>
                <view class="ai-actions">
                  <view class="mini-btn yes" @click="reviewSuggestion(group.media, suggestion, 'confirm')">确认</view>
                  <view class="mini-btn" @click="reviewSuggestion(group.media, suggestion, 'reject')">拒绝</view>
                </view>
              </view>
            </view>
            <text v-if="aiDoneVisible" class="ai-done">识别建议已处理完</text>
          </view>

          <!-- 相关孩子（仅个体记录有值） -->
          <view v-if="relatedStudents.length" class="related-section">
            <text class="section-title">相关孩子<span class="section-count">（{{ relatedCount }}人）</span></text>
            <view class="kid-list">
              <view v-for="student in relatedShown" :key="student.id" class="kid-pill">
                <image
                  v-if="student.avatar"
                  class="kid-avatar-img"
                  :src="student.avatar"
                  mode="aspectFill"
                />
                <view v-else class="kid-avatar">{{ String(student.name || "?").slice(0, 1) }}</view>
                <text class="kid-name">{{ student.name }}</text>
              </view>
              <view v-if="relatedMore" class="kid-pill kid-more">+{{ relatedMore }}</view>
            </view>
          </view>

          <!-- 互动条 -->
          <view class="record-bar">
            <view class="record-stats">
              <text class="stat">{{ record.likes_count || 0 }} 个点赞</text>
              <text class="stat">{{ displayCommentTotal }} 条评论</text>
            </view>
            <view class="record-actions">
              <view class="action-btn" @click="goEdit">编辑</view>
              <view class="action-btn" @click="handleShare">分享</view>
              <view
                v-if="achievements.length"
                class="action-btn award-trigger"
                @click="grantOpen = !grantOpen"
              >{{ grantOpen ? "收起" : "授徽章" }}</view>
            </view>
          </view>
        </view>

        <!-- 授徽章面板 -->
        <view v-if="grantOpen" class="award-panel">
          <view class="award-head">
            <view class="award-head-copy">
              <text class="award-title">授徽章</text>
              <text class="award-sub">选一枚徽章，授予参与活动的小朋友</text>
            </view>
            <view class="panel-close" @click="grantOpen = false">
              <text class="icon">&#xeb99;</text>
            </view>
          </view>

          <view class="badge-row">
            <view
              v-for="achievement in achievements"
              :key="achievement.id"
              class="badge-choice"
              :class="{ 'is-selected': grant.achievement_id === achievement.id }"
              @click="selectAchievement(achievement.id)"
            >
              <image v-if="achievement.icon" class="badge-choice-icon" :src="achievement.icon" mode="aspectFit" />
              <text class="badge-choice-label">{{ achievement.name }}</text>
            </view>
          </view>

          <template v-if="grant.achievement_id">
            <text class="field-label">授予给</text>
            <view v-if="students.length" class="kid-choices">
              <view
                v-for="student in students"
                :key="student.student_id"
                class="kid-choice"
                :class="{ 'is-on': grant.student_ids.includes(student.student_id) }"
                @click="toggleGrantStudent(student.student_id)"
              >{{ student.name }}</view>
            </view>
            <text class="award-hint">{{ awardHint }}</text>

            <view class="witness-row" @click="grant.as_witness = !grant.as_witness">
              <view class="witness-box" :class="{ 'is-on': grant.as_witness }">
                <text class="icon">&#xeb7b;</text>
              </view>
              <text>本记录作为获奖见证</text>
            </view>

            <view class="award-submit" @click="doGrant">{{ grant.granting ? "授予中..." : "确认授予" }}</view>
          </template>
        </view>

        <!-- 家长评论 -->
        <view class="comments-card">
          <view class="comments-head">
            <text class="section-title">家长评论<span class="section-count">（{{ displayCommentTotal }}条）</span></text>
          </view>

          <view v-if="!comments.length" class="comment-empty">
            <view class="comment-empty-icon">
              <text class="icon">&#xeb51;</text>
            </view>
            <text class="comment-empty-text">还没有家长评论</text>
            <view class="empty-cta" @click="focusComment">写第一条评论</view>
          </view>

          <view v-else class="comment-list">
            <view v-for="comment in comments" :key="comment.id" class="comment">
              <image
                v-if="comment.user?.avatar"
                class="comment-avatar-img"
                :src="comment.user.avatar"
                mode="aspectFill"
              />
              <view v-else class="comment-avatar">{{ initialOf(comment.user) }}</view>
              <view class="comment-main">
                <view class="comment-top">
                  <text class="comment-name">{{ comment.user?.nickname || "家长" }}</text>
                  <text v-if="relationText(comment.user)" class="role-tag">{{ relationText(comment.user) }}</text>
                  <text class="comment-time">{{ formatCommentTime(comment.created_at) }}</text>
                </view>
                <text class="comment-text">{{ comment.content }}</text>
                <view class="comment-actions">
                  <view
                    class="link-btn"
                    :class="{ 'is-on': comment.liked }"
                    @click="toggleCommentLike(comment)"
                  >赞 {{ comment.likes_count || 0 }}</view>
                  <view class="link-btn" @click="startReply(comment)">回复</view>
                </view>

                <view v-if="comment.replies?.length" class="reply-thread">
                  <view v-for="reply in comment.replies" :key="reply.id" class="reply-card">
                    <view class="reply-head">
                      <text class="reply-name">{{ reply.user?.nickname || "老师" }}</text>
                      <text v-if="relationText(reply.user)" class="role-tag">{{ relationText(reply.user) }}</text>
                      <text class="comment-time">{{ formatCommentTime(reply.created_at) }}</text>
                    </view>
                    <text class="reply-text">{{ reply.content }}</text>
                    <view class="comment-actions">
                      <view
                        class="link-btn"
                        :class="{ 'is-on': reply.liked }"
                        @click="toggleCommentLike(reply)"
                      >赞 {{ reply.likes_count || 0 }}</view>
                    </view>
                  </view>
                </view>
              </view>
            </view>
          </view>
        </view>
      </template>
    </view>

    <!-- 固定评论栏 -->
    <template #bottom>
      <view class="comment-bar" :class="{ 'is-locked': locked }">
        <view v-if="replyTo" class="reply-strip">
          <text class="reply-context">回复 {{ replyTo.name }}</text>
          <view class="cancel-reply" @click="cancelReply">取消</view>
        </view>
        <view class="composer">
          <input
            v-model="commentText"
            class="comment-input"
            type="text"
            :disabled="locked"
            :focus="inputFocused"
            :maxlength="200"
            placeholder="请输入评论..."
            placeholder-style="color: #94A3B8"
            confirm-type="send"
            @blur="inputFocused = false"
            @confirm="submitComment"
          />
          <view
            class="send-btn"
            :class="{ 'is-disabled': sendDisabled }"
            @click="submitComment"
          >
            <text class="icon">&#xf0d9;</text>
          </view>
        </view>
      </view>
    </template>

    <!-- 照片预览（底部弹层） -->
    <view v-if="previewOpen" class="preview-layer">
      <view class="preview-backdrop" @click="closePreview" />
      <view class="preview-sheet">
        <view class="preview-head">
          <text class="preview-title">照片预览</text>
          <view class="preview-close" @click="closePreview">
            <text class="icon">&#xeb99;</text>
          </view>
        </view>
        <swiper class="preview-swiper" :current="previewIndex" @change="onPreviewChange">
          <swiper-item v-for="(mediaItem, index) in media" :key="mediaItem.id || index">
            <image class="preview-img" :src="mediaItem.file_url || mediaItem.thumb_url" mode="aspectFit" />
          </swiper-item>
        </swiper>
        <text class="preview-meta">{{ previewMeta }}</text>
        <view v-if="media.length > 1" class="preview-nav">
          <view class="preview-nav-btn" :class="{ 'is-disabled': previewIndex === 0 }" @click="stepPreview(-1)">上一张</view>
          <view class="preview-nav-btn" :class="{ 'is-disabled': previewIndex === media.length - 1 }" @click="stepPreview(1)">下一张</view>
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

.detail-page {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  padding-bottom: $spacing-xl;
}

/* ---------- 记录主卡 ---------- */
.record-card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.record-author {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
}

.avatar {
  flex: none;
  width: $component-avatar-md;
  height: $component-avatar-md;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.avatar-img {
  width: 100%;
  height: 100%;
}

.author-info {
  min-width: 0;
}

.author-name {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.author-time {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
}

.record-title {
  display: block;
  margin-top: $spacing-lg;
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.record-body {
  margin-top: $spacing-md;
}

.record-paragraph {
  display: block;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.75;
  overflow-wrap: anywhere;

  & + .record-paragraph {
    margin-top: $spacing-xs;
  }
}

/* ---------- 区块标题 ---------- */
.section-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.section-count {
  color: $color-text-hint;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-normal;
}

/* ---------- 照片 ---------- */
.media-section {
  margin-top: $spacing-xl;
}

.media-head {
  display: flex;
  align-items: baseline;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
}

.ai-banner {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: $spacing-lg;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-semantic-info-bg;
}

.ai-banner-text {
  flex: 1;
  min-width: 0;
  color: $color-semantic-info;
  font-size: $typography-font-size-base;
  line-height: 1.4;
}

.ai-banner-strong {
  color: $color-semantic-info;
  font-weight: $typography-font-weight-semibold;
}

.ai-banner-btn {
  flex: none;
  min-height: 72rpx;
  padding: 0 $spacing-lg;
  display: flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-semantic-info;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

.media-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
}

.photo-cell {
  min-width: 0;
}

.photo-open {
  position: relative;
  width: 100%;
  height: 202rpx;
  overflow: hidden;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}

.photo-img {
  width: 100%;
  height: 100%;
  display: block;
}

.photo-flag {
  position: absolute;
  top: $spacing-xs;
  left: $spacing-xs;
  z-index: 2;
  padding: 2rpx $spacing-xs;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.6;
}

.photo-flag.is-ai {
  color: $color-semantic-info;
}

.photo-caption {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 2;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2rpx;
  padding: $spacing-xs;
  background: $color-bg-card;
}

.photo-caption-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.photo-kid {
  max-width: 100%;
  padding: 0 $spacing-xs;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  line-height: 1.6;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.photo-fail {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2rpx;
  width: 100%;
  height: 202rpx;
  padding: $spacing-xs;
  border: 2rpx dashed $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  color: $color-text-hint;
}

.photo-fail-text {
  font-size: $typography-font-size-xs;
}

.photo-fail-retry {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.group-photo-btn {
  display: flex;
  align-items: center;
  width: 100%;
  min-height: 112rpx;
  margin-top: $spacing-md;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.group-photo-btn.is-busy {
  opacity: 0.6;
}

.group-photo-copy {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2rpx;
}

.group-photo-title {
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

.group-photo-sub {
  color: $color-semantic-success;
  font-size: $typography-font-size-xs;
}

/* AI 识别建议 */
.ai-card {
  margin-top: $spacing-md;
  padding: $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
}

.ai-card-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: $spacing-xs;
}

.ai-card-title {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.ai-card-ref {
  color: $color-text-hint;
  font-size: $typography-font-size-xs;
}

.ai-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-width: 0;
  padding: $spacing-xs 0;

  & + .ai-row {
    border-top: 1rpx solid $color-border-subtle;
  }
}

.ai-avatar {
  flex: none;
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.ai-person {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-xs;
}

.ai-name {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
}

.status {
  padding: 0 $spacing-xs;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  line-height: 1.6;
}

.status-ok {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.status-pending {
  background: $color-bg-page;
  color: $color-text-muted;
}

.ai-actions {
  flex: none;
  display: flex;
  gap: $spacing-xs;
}

.mini-btn {
  min-height: 72rpx;
  min-width: 96rpx;
  padding: 0 $spacing-sm;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
}

.mini-btn.yes {
  border-color: transparent;
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
}

.ai-done {
  display: block;
  margin-top: $spacing-xs;
  color: $color-text-hint;
  font-size: $typography-font-size-sm;
}

/* ---------- 相关孩子 ---------- */
.related-section {
  margin-top: $spacing-xl;
}

.kid-list {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
  margin-top: $spacing-md;
}

.kid-pill {
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  max-width: 100%;
  min-height: 80rpx;
  padding: 0 $spacing-md 0 $spacing-xs;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.kid-avatar {
  flex: none;
  width: 56rpx;
  height: 56rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
}

.kid-avatar-img {
  flex: none;
  width: 56rpx;
  height: 56rpx;
  border-radius: $border-radius-full;
}

.kid-name {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kid-more {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
}

/* ---------- 互动条 ---------- */
.record-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  margin-top: $spacing-xl;
  padding-top: $spacing-lg;
  border-top: 1rpx solid $color-border-hairline;
}

.record-stats {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-lg;
  min-width: 0;
}

.stat {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.record-actions {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
}

.action-btn {
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  display: flex;
  align-items: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-base;
}

.award-trigger {
  border-color: transparent;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

/* ---------- 授徽章面板 ---------- */
.award-panel {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.award-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: $spacing-md;
}

.award-head-copy {
  min-width: 0;
}

.award-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
}

.award-sub {
  display: block;
  margin-top: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.panel-close {
  flex: none;
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-lg;
}

.badge-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
  margin-top: $spacing-lg;
}

.badge-choice {
  min-height: 88rpx;
  max-width: 100%;
  padding: 0 $spacing-lg;
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-base;
}

.badge-choice.is-selected {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.badge-choice-icon {
  width: 40rpx;
  height: 40rpx;
  border-radius: $border-radius-sm;
}

.badge-choice-label {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.field-label {
  display: block;
  margin: $spacing-xl 0 $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.kid-choices {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-xs;
}

.kid-choice {
  min-height: 88rpx;
  padding: 0 $spacing-xl;
  display: inline-flex;
  align-items: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-base;
}

.kid-choice.is-on {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-medium;
}

.award-hint {
  display: block;
  margin-top: $spacing-md;
  color: $color-text-hint;
  font-size: $typography-font-size-sm;
}

.witness-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-height: 88rpx;
  margin-top: $spacing-xs;
  color: $color-text-body;
  font-size: $typography-font-size-base;
}

.witness-box {
  flex: none;
  width: 44rpx;
  height: 44rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-sm;
  background: $color-bg-card;
  color: transparent;
  font-size: $typography-font-size-xs;
}

.witness-box.is-on {
  border-color: $color-brand-primary;
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
}

.award-submit {
  width: 100%;
  min-height: 96rpx;
  margin-top: $spacing-lg;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-medium;
}

/* ---------- 评论 ---------- */
.comments-card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.comments-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
}

.comment-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-xl;
}

.comment {
  display: flex;
  gap: $spacing-lg;
  min-width: 0;
}

.comment-avatar {
  flex: none;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.comment-avatar-img {
  flex: none;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
}

.comment-main {
  flex: 1;
  min-width: 0;
}

.comment-top {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-xs;
  min-width: 0;
}

.comment-name {
  max-width: 100%;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  overflow: hidden;
  text-overflow: ellipsis;
}

.role-tag {
  padding: 0 $spacing-xs;
  border-radius: $border-radius-full;
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
  font-size: $typography-font-size-xs;
  line-height: 1.6;
}

.comment-time {
  color: $color-text-hint;
  font-size: $typography-font-size-xs;
}

.comment-text {
  display: block;
  margin-top: $spacing-xs;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.comment-actions {
  display: flex;
  gap: $spacing-lg;
  margin-top: $spacing-xs;
}

.link-btn {
  min-height: 72rpx;
  display: inline-flex;
  align-items: center;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.link-btn.is-on {
  color: $color-brand-primary-dark;
}

.reply-thread {
  margin-top: $spacing-xs;
}

.reply-card {
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-bg-page;

  & + .reply-card {
    margin-top: $spacing-xs;
  }
}

.reply-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-xs;
}

.reply-name {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.reply-text {
  display: block;
  margin-top: $spacing-xs;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.comment-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-xs;
  padding: $spacing-xl $spacing-md;
  text-align: center;
}

.comment-empty-icon {
  color: $color-border-hairline;
  font-size: $typography-font-size-3xl;
}

.comment-empty-text {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
}

.empty-cta {
  min-height: 88rpx;
  margin-top: $spacing-xs;
  padding: 0 $spacing-xl;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

/* ---------- 骨架 ---------- */
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
  border-radius: $border-radius-md;
  animation: sk-pulse 1500ms $animation-easing-default infinite;
}

.sk-author {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  margin-bottom: $spacing-lg;
}

.sk-avatar {
  flex: none;
  width: $component-avatar-md;
  height: $component-avatar-md;
  border-radius: $border-radius-full;
}

.sk-lines {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
}

.sk-title {
  width: 62%;
  height: 36rpx;
  margin-bottom: $spacing-md;
}

.sk-line {
  width: 100%;
  height: 22rpx;
  margin-bottom: $spacing-xs;
}

.sk-line-short {
  width: 40%;
}

.sk-line-mid {
  width: 72%;
}

.sk-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
  margin-top: $spacing-lg;
}

.sk-tile {
  width: 100%;
  height: 202rpx;
  border-radius: $border-radius-xl;
}

.sk-comment {
  display: flex;
  gap: $spacing-lg;
  margin-top: $spacing-lg;
}

/* ---------- 失败态 ---------- */
.error-view {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-md;
  padding: 160rpx $spacing-xl;
  text-align: center;
}

.error-icon {
  width: 128rpx;
  height: 128rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-bold;
}

.error-text {
  color: $color-text-muted;
  font-size: $typography-font-size-md;
}

.retry-load-btn {
  min-height: 88rpx;
  padding: 0 $spacing-xl;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

/* ---------- 固定评论栏 ---------- */
.comment-bar {
  padding: $spacing-md $component-page-padding-x calc(#{$spacing-md} + env(safe-area-inset-bottom));
  background: $color-bg-card;
  border-top: 1rpx solid $color-border-hairline;
  box-shadow: $shadow-lg;
}

.comment-bar.is-locked {
  opacity: 0.5;
}

.reply-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: $spacing-xs;
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.reply-context {
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.cancel-reply {
  flex: none;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.composer {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.comment-input {
  flex: 1;
  min-width: 0;
  height: 88rpx;
  padding: 0 $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
}

.send-btn {
  flex: none;
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
}

.send-btn.is-disabled {
  background: $color-bg-page;
  color: $color-text-hint;
}

/* ---------- 预览弹层 ---------- */
.preview-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 120;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  align-items: center;
}

.preview-backdrop {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: $color-text-primary;
  opacity: 0.45;
}

.preview-sheet {
  position: relative;
  z-index: 1;
  width: 100%;
  padding: $spacing-sm $spacing-xl calc(#{$spacing-xl} + env(safe-area-inset-bottom));
  background: $color-bg-card;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  box-shadow: $shadow-lg;
}

.preview-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}

.preview-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
}

.preview-close {
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
}

.preview-swiper {
  width: 100%;
  height: 480rpx;
  margin-top: $spacing-sm;
}

.preview-img {
  width: 100%;
  height: 100%;
}

.preview-meta {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-hint;
  font-size: $typography-font-size-sm;
  text-align: center;
}

.preview-nav {
  display: flex;
  justify-content: space-between;
  gap: $spacing-md;
  margin-top: $spacing-sm;
}

.preview-nav-btn {
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  display: flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-base;
}

.preview-nav-btn.is-disabled {
  color: $color-text-hint;
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
