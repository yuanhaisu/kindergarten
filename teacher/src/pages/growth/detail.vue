<script setup>
import { ref, computed } from "vue"
import { onLoad, onUnload } from "@dcloudio/uni-app"
import { fetchTeacherGrowthLog, fetchTeacherGrowthComments, createTeacherGrowthComment, fetchGrowthTaggingStatus, reviewMediaSuggestion, confirmHighSuggestions, fetchPresentStudents, applyClassRoster } from "@/api/teacher-growth"
import { fetchTeacherClassStudents } from "@/api/teacher-class"
import { fetchTeacherAchievements, grantAchievement } from "@/api/teacher-achievements"
import { ensureLogin } from "@/utils/login"

const record = ref({})
const comments = ref([])
const commentText = ref("")
const loading = ref(true)
const loadError = ref("")
const submitting = ref(false)

// ---- 补授徽章（已发布记录可后挂，本记录作为见证）----
const achievements = ref([])
const grant = ref({ achievement_id: null, student_ids: [], as_witness: true, granting: false })
const grantOpen = ref(false)
// 授予选择器用「班级学生」：全班/个体记录都适用（个体记录的关联学生是班级学生的子集）。
const students = ref([])

async function loadStudents() {
  try {
    // 必须先登录：onLoad 里 loadStudents 与 loadDetail 并行，
    // 不等 ensureLogin 会带着空 token 发请求 → 401 → 静默吞成空列表
    await ensureLogin()
    const res = await fetchTeacherClassStudents()
    students.value = res.data?.list || []
  } catch (e) {
    // 学生列表失败不影响详情浏览，授予时会提示无可授予学生
  }
}

async function loadAchievements() {
  try {
    const res = await fetchTeacherAchievements()
    achievements.value = res.data?.list || []
  } catch (e) {
    // 徽章列表失败不影响详情浏览
  }
}

function toggleGrantStudent(sid) {
  const i = grant.value.student_ids.indexOf(sid)
  if (i >= 0) grant.value.student_ids.splice(i, 1)
  else grant.value.student_ids.push(sid)
}

function selectAchievement(id) {
  grant.value.achievement_id = grant.value.achievement_id === id ? null : id
}

async function doGrant() {
  const g = grant.value
  if (!g.achievement_id) {
    uni.showToast({ title: "请先选一枚徽章", icon: "none" })
    return
  }
  // 未勾选时授予给全班学生（详情页补授的合理默认）
  const target = g.student_ids.length ? g.student_ids : students.value.map((s) => s.student_id)
  if (!target.length) {
    uni.showToast({ title: "无可授予的学生", icon: "none" })
    return
  }
  g.granting = true
  try {
    await grantAchievement(g.achievement_id, {
      student_ids: target,
      log_id: g.as_witness ? record.value.id : undefined,
    })
    uni.showToast({ title: "已授予徽章", icon: "success" })
    grantOpen.value = false
    loadDetail(record.value.id)
  } catch (e) {
    uni.showToast({ title: e?.message || "授予失败", icon: "none" })
  } finally {
    g.granting = false
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
    comments.value = commentRes.data?.list || commentRes.data || []
    // 识别是发布后的异步过程：有未完成任务时轮询刷新建议（无任务则一次都不轮询）
    refreshTaggingStatus()
  } catch (err) {
    loadError.value = err?.message || "加载失败"
  } finally {
    loading.value = false
  }
}

// ---- 照片点名与识别建议（design D6/D8）----
// 点名呈现：student_ids → 姓名（班级学生表映射）；建议态：高置信带「高置信」标记（可一键确认），低置信仅提示。
const studentNameById = computed(() => {
  const map = {}
  students.value.forEach((s) => { map[s.student_id] = s.name })
  return map
})

function namedNames(m) {
  return (m.student_ids || []).map((id) => studentNameById.value[id] || `#${id}`)
}

function suggestionsOf(m) {
  return Array.isArray(m.student_suggestions) ? m.student_suggestions : []
}

function previewPhoto(idx) {
  const urls = (record.value.media || []).map((m) => m.thumb_url || m.url || m.file_url)
  uni.previewImage({ current: idx, urls })
}

// 一键采纳：高置信建议数量 >0 时出现「全部确认」——一次活动的合影可达数十条建议，
// 逐条点击会把 AI 加速变成负担（design D10）
const highSuggestionCount = computed(() =>
  (record.value.media || []).reduce(
    (acc, m) => acc + suggestionsOf(m).filter((s) => s.tier === "high").length,
    0,
  ),
)
const confirmingHigh = ref(false)

async function confirmAllHigh() {
  if (confirmingHigh.value) return
  confirmingHigh.value = true
  try {
    const res = await confirmHighSuggestions(record.value.id)
    const n = res.data?.confirmed ?? 0
    ;(record.value.media || []).forEach((m) => {
      const highs = suggestionsOf(m).filter((s) => s.tier === "high")
      if (!highs.length) return
      m.student_suggestions = suggestionsOf(m).filter((s) => s.tier !== "high")
      m.student_ids = [...(m.student_ids || []), ...highs.map((s) => s.student_id)]
    })
    uni.showToast({ title: `已确认 ${n} 条高置信建议`, icon: "none" })
  } catch (e) {
    uni.showToast({ title: e?.message || "批量确认失败", icon: "none" })
    refreshTaggingStatus()
  } finally {
    confirmingHigh.value = false
  }
}

// 本地即时反馈 + 后台落库：确认 → 移入点名；拒绝 → 移除建议。失败则回滚重取。
async function reviewSuggestion(m, s, action) {
  try {
    await reviewMediaSuggestion(m.id, s.student_id, action)
    const media = record.value.media || []
    const target = media.find((x) => x.id === m.id)
    if (target) {
      target.student_suggestions = suggestionsOf(target).filter((x) => x.student_id !== s.student_id)
      if (action === "confirm") {
        target.student_ids = [...(target.student_ids || []), s.student_id]
      }
    }
    uni.showToast({ title: action === "confirm" ? "已确认点名" : "已忽略", icon: "none" })
  } catch (e) {
    uni.showToast({ title: e?.message || "操作失败", icon: "none" })
    refreshTaggingStatus()
  }
}

// 大合影「一键全班」（D11）：名单 = 记录当日的到场学生（服务端口径），点击即落为确认点名。
// 到场名单惰性加载：仅在对全班记录的大合影上需要（「当日到场 N 人」的预告数）。
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
  } catch (e) {
    // 名单失败不阻断浏览；点击「全班」由服务端兜底（回读最终名单下发）
    presentLoaded.value = false
  }
}

async function applyRoster(m) {
  if (rosterBusyId.value) return
  rosterBusyId.value = m.id
  try {
    const res = await applyClassRoster(m.id)
    const n = res.data?.present ?? 0
    // 以服务端回读的最终名单为准（含「已拒绝不复活」与点名/到场名单的时序差——
    // 本地并集算不出这两者）
    if (Array.isArray(res.data?.named)) m.student_ids = res.data.named
    uni.showToast({ title: `已点名 ${n} 人（当日到场）`, icon: "none" })
  } catch (e) {
    uni.showToast({ title: e?.message || "全班点名失败", icon: "none" })
  } finally {
    rosterBusyId.value = 0
  }
}

// 轮询：仅当识别仍有未完成任务时进行（间隔 4s），完成后自动停止。
// 失败**不终止**轮询：退避重试（4s→8s→…→30s 封顶）——一次网络抖动不应让
// 建议永远到不了（旧行为遇错即停，要重进页面才恢复）。自愈由服务端兜底：
// 只要识别最终完成，下一次成功的轮询就会把建议/分型带上。
const TAGGING_POLL_MS = 4000
const TAGGING_RETRY_MAX_MS = 30000
let taggingTimer = null
let taggingInFlight = false
let taggingFailures = 0

function applyTaggingStatus(data) {
  const byId = {}
  ;(data.media || []).forEach((x) => { byId[x.id] = x })
  let hasGroupPhoto = false
  ;(record.value.media || []).forEach((m) => {
    const x = byId[m.id]
    if (!x) return
    m.student_suggestions = x.suggestions || []
    m.is_group_photo = !!x.is_group_photo
    if (m.is_group_photo) hasGroupPhoto = true
  })
  // 分型由服务端判定（阈值不下发到端）；个体记录不做全班，无需取名单
  if (hasGroupPhoto && record.value.target_type !== 2) ensurePresent()
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
  } catch (e) {
    // 静默退避重试：识别状态不是浏览详情的关键路径，但也不该因单次失败永久停摆
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

function goBack() {
  uni.navigateBack()
}

function goEdit() {
  uni.navigateTo({ url: `/pages/growth/create?id=${record.value.id}` })
}

function handleShare() {
  // uni.shareAppMessage 由微信小程序 onShareAppMessage 钩子接管
  uni.showToast({ title: "点击右上角分享", icon: "none" })
}

function replyTo(parentId) {
  uni.showToast({ title: "回复功能开发中", icon: "none" })
}

async function submitComment() {
  const text = commentText.value.trim()
  if (!text) return

  submitting.value = true
  try {
    const res = await createTeacherGrowthComment(record.value.id, text)
    commentText.value = ""
    uni.showToast({ title: "评论成功", icon: "none" })
    // Reload comments
    const commentRes = await fetchTeacherGrowthComments(record.value.id)
    comments.value = commentRes.data?.list || commentRes.data || []
  } catch (err) {
    uni.showToast({ title: err?.message || "评论失败", icon: "none" })
  } finally {
    submitting.value = false
  }
}

function formatDate(dateStr) {
  if (!dateStr) return ""
  return dateStr.slice(0, 16)
}

function formatFullDate(dateStr) {
  if (!dateStr) return ""
  const d = new Date(dateStr)
  const y = d.getFullYear()
  const m = d.getMonth() + 1
  const day = d.getDate()
  const time = dateStr.slice(11, 16)
  return `${y}年${m}月${day}日 ${time}`
}

function formatTime(dateStr) {
  if (!dateStr) return ""
  return dateStr.slice(11, 16)
}

onLoad((options) => {
  const id = options?.id
  if (id) loadDetail(id)
  loadAchievements()
  loadStudents()
})
</script>

<template>
  <view class="page">
    <!-- 加载/错误状态 -->
    <view v-if="loading" class="state-wrap">
      <text class="state-text">加载中...</text>
    </view>
    <view v-else-if="loadError" class="state-wrap">
      <text class="state-text">{{ loadError }}</text>
      <button class="retry-btn" @click="loadDetail(record.id || '')">重试</button>
    </view>

    <template v-else>
      <!-- 记录头部 -->
      <view class="header-section">
        <view class="author-row">
          <view class="author-avatar">
            <text class="avatar-icon">{{ (record.creator?.nickname || "老师")[0] }}</text>
          </view>
          <view class="author-info">
            <text class="author-name">{{ record.creator?.nickname || "老师" }}</text>
            <text class="author-date">发布于 {{ formatFullDate(record.date || record.created_at) }}</text>
          </view>
        </view>
        <text v-if="record.performance_tags?.length" class="record-title">
          {{ record.performance_tags.join(" · ") }}
        </text>
        <text class="record-content">{{ record.content }}</text>
      </view>

      <!-- 照片区域：点名呈现 + AI 建议确认（高置信预勾标记 / 低置信提示） -->
      <view v-if="record.media?.length" class="photo-section">
        <!-- 省力路径（D10）：高置信建议一次性采纳，低置信仍逐条处理 -->
        <view v-if="highSuggestionCount" class="bulk-confirm">
          <text class="bulk-confirm-text">{{ highSuggestionCount }} 条高置信建议</text>
          <view class="bulk-confirm-btn" :class="{ busy: confirmingHigh }" @click="confirmAllHigh">
            {{ confirmingHigh ? '确认中…' : '全部确认' }}
          </view>
        </view>
        <view class="photo-grid">
          <view v-for="(m, idx) in record.media" :key="idx" class="photo-card">
            <image
              :src="m.thumb_url || m.url || m.file_url"
              mode="aspectFill"
              class="photo-item"
              @click="previewPhoto(idx)"
            />
            <view v-if="namedNames(m).length" class="photo-tags">
              <text class="photo-tags-label">点名</text>
              <text v-for="n in namedNames(m)" :key="n" class="photo-tag">{{ n }}</text>
            </view>
            <!-- 大合影（服务端分型）：一键全班（当日到场）——逐人确认一张合影注定不完整（D11）。
                 个体记录不做全班（服务端会拒绝），故不呈现入口；名单未取到时先不显示人数。 -->
            <view
              v-if="m.is_group_photo && record.target_type !== 2"
              class="photo-roster"
              :class="{ busy: rosterBusyId === m.id }"
              @click.stop="applyRoster(m)"
            >
              <text class="photo-roster-text">
                {{ rosterBusyId === m.id
                  ? '点名中…'
                  : presentLoaded
                    ? `大合影 · 一键全班（当日到场 ${presentIds.length} 人）`
                    : '大合影 · 一键全班' }}
              </text>
            </view>
            <view v-if="suggestionsOf(m).length" class="photo-sugs">
              <text class="photo-sugs-title">AI 识别建议</text>
              <view v-for="s in suggestionsOf(m)" :key="s.student_id" class="photo-sug-row">
                <view class="photo-sug-info">
                  <text class="photo-sug-name">{{ s.name }}</text>
                  <text class="photo-sug-tier" :class="s.tier === 'high' ? 'high' : 'low'">
                    {{ s.tier === 'high' ? '高置信' : '待确认' }}
                  </text>
                </view>
                <view class="sug-btn confirm" @click.stop="reviewSuggestion(m, s, 'confirm')">确认</view>
                <view class="sug-btn reject" @click.stop="reviewSuggestion(m, s, 'reject')">拒绝</view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <!-- 相关孩子（详情响应字段 related_students，仅个体记录有值） -->
      <view v-if="record.related_students?.length" class="student-section">
        <text class="section-label">相关孩子（{{ record.related_students_count || record.related_students.length }}人）</text>
        <view class="student-chips">
          <view
            v-for="(student, idx) in record.related_students.slice(0, 6)"
            :key="student.id || idx"
            class="student-chip"
            :class="{ primary: idx === 0 }"
          >
            <image
              v-if="student.avatar"
              :src="student.avatar"
              mode="aspectFill"
              class="chip-avatar"
            />
            <view v-else class="chip-avatar-placeholder" />
            <text class="chip-name">{{ student.name }}</text>
          </view>
          <view v-if="record.related_students.length > 6" class="student-chip more">
            <text>+{{ record.related_students.length - 6 }}人</text>
          </view>
        </view>
      </view>

      <!-- 互动统计 & 操作 -->
      <view class="stats-bar">
        <view class="stats-left">
          <view class="stat-item">
            <text class="stat-icon">赞</text>
            <text class="stat-text">{{ record.likes_count || 0 }}个点赞</text>
          </view>
          <view class="stat-item">
            <text class="stat-icon">评</text>
            <text class="stat-text">{{ record.comments_count || comments.length }}条评论</text>
          </view>
        </view>
        <view class="stats-right">
          <view class="action-btn edit-btn" @click="goEdit">
            <text>编辑</text>
          </view>
          <view class="action-btn share-btn" @click="handleShare">
            <text>分享</text>
          </view>
          <view v-if="achievements.length" class="action-btn grant-btn" @click="grantOpen = !grantOpen">
            <text>{{ grantOpen ? '收起' : '授徽章' }}</text>
          </view>
        </view>
      </view>

      <!-- 补授徽章：本记录可作为获奖见证 -->
      <view v-if="grantOpen" class="grant-panel">
        <text class="grant-title">选择徽章</text>
        <view class="badge-pick-row">
          <view
            v-for="a in achievements" :key="a.id"
            class="badge-pick"
            :class="{ selected: grant.achievement_id === a.id }"
            @click="selectAchievement(a.id)"
          >
            <image v-if="a.icon" :src="a.icon" class="badge-pick-icon" mode="aspectFit" />
            <text class="badge-pick-name">{{ a.name }}</text>
          </view>
        </view>

        <template v-if="grant.achievement_id">
          <text class="grant-title">授予给</text>
          <view class="badge-pick-row" v-if="students.length">
            <view
              v-for="s in students" :key="s.student_id"
              class="badge-pick"
              :class="{ selected: grant.student_ids.includes(s.student_id) }"
              @click="toggleGrantStudent(s.student_id)"
            >
              <text class="badge-pick-name">{{ s.name }}</text>
            </view>
          </view>
          <view v-else class="grant-hint"><text>不选则默认授予本班全部学生</text></view>

          <view class="grant-witness" @click="grant.as_witness = !grant.as_witness">
            <text>{{ grant.as_witness ? '☑' : '☐' }}</text>
            <text>本记录作为获奖见证</text>
          </view>

          <button class="grant-submit" :loading="grant.granting" @click="doGrant">
            {{ grant.granting ? '授予中...' : '确认授予' }}
          </button>
        </template>
      </view>

      <!-- 评论区域 -->
      <view v-if="comments.length" class="comment-section">
        <view class="comment-head">
          <text class="section-label">家长评论（{{ comments.length }}条）</text>
        </view>

        <view class="comment-list">
          <view
            v-for="comment in comments"
            :key="comment.id"
            class="comment-card"
          >
            <!-- 主评论 -->
            <view class="comment-main">
              <image
                v-if="comment.user?.avatar"
                :src="comment.user.avatar"
                mode="aspectFill"
                class="comment-avatar"
              />
              <view v-else class="comment-avatar-placeholder">
                <text>{{ (comment.user?.nickname || "家")[0] }}</text>
              </view>
              <view class="comment-body">
                <view class="comment-meta">
                  <text class="comment-author">{{ comment.user?.nickname || "家长" }}</text>
                  <view v-if="comment.user?.role_tag" class="comment-role-tag">
                    <text>{{ comment.user.role_tag }}</text>
                  </view>
                  <text class="comment-time">{{ formatTime(comment.created_at) }}</text>
                </view>
                <text class="comment-text">{{ comment.content }}</text>
                <view class="comment-actions">
                  <view class="comment-act" @click="replyTo(comment.id)">
                    <text class="comment-act-icon">赞</text>
                    <text class="comment-act-num">{{ comment.likes_count || 0 }}</text>
                  </view>
                  <view class="comment-act" @click="replyTo(comment.id)">
                    <text class="comment-act-icon">回</text>
                    <text>回复</text>
                  </view>
                </view>
              </view>
            </view>

            <!-- 老师回复 -->
            <view v-if="comment.replies?.length" class="teacher-reply">
              <view v-for="reply in comment.replies" :key="reply.id" class="reply-item">
                <text class="reply-author">{{ reply.user?.nickname || "老师" }}</text>
                <text class="reply-time">{{ formatTime(reply.created_at) }}</text>
                <text class="reply-text">{{ reply.content }}</text>
              </view>
            </view>
          </view>

        </view>
      </view>

      <!-- 底部留白 -->
      <view class="bottom-safe" />
    </template>

    <!-- 底部评论输入栏 -->
    <view class="bottom-bar">
      <view class="input-box">
        <input
          v-model="commentText"
          class="comment-input"
          placeholder="请输入评论..."
          placeholder-style="color: #94A3B8; font-size: $typography-font-size-md"
          :disabled="submitting"
          confirm-type="send"
          @confirm="submitComment"
        />
      </view>
      <view class="bottom-btn" @click="submitComment">
        <text class="bottom-btn-icon">发</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
$slate-200: #E2E8F0;
$slate-300: #CBD5E1;
$slate-600: #475569;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  background: #fff;
}

// ===== 状态 =====
.state-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}

.state-text {
  font-size: $typography-font-size-md;
  color: $color-text-hint;
}

.retry-btn {
  width: 180rpx;
  height: 72rpx;
  margin-top: 28rpx;
  border-radius: 999rpx;
  background: $color-semantic-info;
  color: #fff;
  font-size: $typography-font-size-base;
}

// ===== 记录头部 =====
.header-section {
  padding: $spacing-xl 48rpx 32rpx;
  border-bottom: 2rpx solid $slate-200;
}

.author-row {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
}

.author-avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 96rpx;
  height: 96rpx;
  border-radius: 999rpx;
  background: #DBEAFE;
}

.avatar-icon {
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: $color-semantic-info;
}

.author-info {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.author-name {
  font-size: $typography-font-size-xl;
  font-weight: 600;
  color: $color-text-primary;
}

.author-date {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
}

.record-title {
  display: block;
  margin-top: 32rpx;
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

.record-content {
  display: block;
  margin-top: 16rpx;
  font-size: $typography-font-size-lg;
  color: $color-text-body;
  line-height: 1.7;
}

// ===== 照片区域 =====
.photo-section {
  padding: $spacing-xl 48rpx;
}

.photo-grid {
  display: grid;
  /* minmax(0, 1fr)：允许列被压缩——照片下的建议行内容较宽，
     纯 1fr 会按 min-content 把列撑到 240px+ 导致整块溢出屏幕 */
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  align-items: start;
  gap: $spacing-lg;
}

.photo-card {
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
}

.photo-item {
  width: 100%;
  aspect-ratio: 1;
  border-radius: $border-radius-xl;
}

/* 一键采纳高置信建议（D10） */
.bulk-confirm {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
  padding: $spacing-sm $spacing-md;
  background: $color-semantic-info-bg;
  border-radius: $border-radius-lg;
}
.bulk-confirm-text {
  font-size: $typography-font-size-sm;
  color: $color-semantic-info;
}
.bulk-confirm-btn {
  padding: $spacing-xs $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
  color: $color-text-on-brand;
  font-size: $typography-font-size-sm;
}
.bulk-confirm-btn.busy { opacity: 0.6; }

/* 大合影一键全班（D11）：一张合影的默认名单 = 记录当日的到场学生 */
.photo-roster {
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius-lg;
  background: $color-semantic-success-bg;
  border: 1rpx solid $color-semantic-success;
}
.photo-roster.busy { opacity: 0.6; }
.photo-roster-text {
  font-size: $typography-font-size-xs;
  color: $color-semantic-success;
}

/* 点名呈现：这张照片关联的孩子 */
.photo-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-xs;
}
.photo-tags-label {
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}
.photo-tag {
  padding: 2rpx $spacing-sm;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
}

/* AI 识别建议：确认后才对家长生效 */
.photo-sugs {
  background: $color-semantic-info-bg;
  border-radius: $border-radius-lg;
  padding: $spacing-sm;
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
}
.photo-sugs-title {
  font-size: $typography-font-size-xs;
  color: $color-semantic-info;
}
.photo-sug-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap; /* 窄列时按钮换行，避免挤出屏幕 */
  gap: $spacing-xs;
}
.photo-sug-info {
  flex: 1;
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  min-width: 0;
}
.photo-sug-name {
  font-size: $typography-font-size-sm;
  color: $color-text-primary;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.photo-sug-tier {
  font-size: $typography-font-size-xs;
}
.photo-sug-tier.high { color: $color-semantic-success; }
.photo-sug-tier.low { color: $color-text-hint; }
.sug-btn {
  padding: 2rpx $spacing-sm;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
}
.sug-btn.confirm {
  background: $color-brand-primary;
  color: $color-text-on-brand;
}
.sug-btn.reject {
  border: 1rpx solid $color-border-hairline;
  color: $color-text-muted;
}

// ===== 相关孩子 =====
.student-section {
  padding: 0 48rpx 32rpx;
}

.section-label {
  font-size: $typography-font-size-lg;
  font-weight: 700;
  color: $color-text-primary;
  margin-bottom: 24rpx;
}

.student-chips {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;
  margin-top: 24rpx;
}

.student-chip {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  padding: $spacing-sm 24rpx;
  border-radius: 999rpx;
  background: #F1F5F9;
}

.student-chip.primary {
  background: $color-semantic-info-bg;
}

.chip-avatar {
  width: 40rpx;
  height: 40rpx;
  border-radius: 999rpx;
}

.chip-avatar-placeholder {
  width: 40rpx;
  height: 40rpx;
  border-radius: 999rpx;
  background: $slate-300;
}

.chip-name {
  font-size: $typography-font-size-sm;
  font-weight: 500;

  .primary & {
    color: $color-semantic-info;
  }

  .student-chip:not(.primary):not(.more) & {
    color: $color-text-muted;
  }
}

.student-chip.more {
  background: #F1F5F9;

  text {
    font-size: $typography-font-size-sm;
    color: $color-text-muted;
  }
}

// ===== 互动统计 & 操作 =====
.stats-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-xl 48rpx;
  border-top: 2rpx solid $slate-200;
  border-bottom: 2rpx solid $slate-200;
}

.stats-left {
  display: flex;
  gap: 48rpx;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.stat-icon {
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: $color-text-hint;
}

.stat-text {
  font-size: $typography-font-size-md;
  color: $color-text-muted;
}

.stats-right {
  display: flex;
  gap: $spacing-xl;
}

.action-btn {
  padding: $spacing-sm 24rpx;
  border-radius: $border-radius-lg;

  text {
    font-size: $typography-font-size-base;
    font-weight: 500;
  }
}

.edit-btn {
  background: #F1F5F9;

  text { color: $color-text-muted; }
}

.share-btn {
  background: $color-semantic-info-bg;

  text { color: $color-semantic-info; }
}

// ===== 评论区域 =====
.comment-section {
  padding: $spacing-xl 48rpx;
}

.comment-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 32rpx;
}

.view-all {
  font-size: $typography-font-size-md;
  color: $color-semantic-info;
}

.comment-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-xl;
}

.comment-card {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.comment-main {
  display: flex;
  gap: $spacing-lg;
}

.comment-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 999rpx;
  flex-shrink: 0;
}

.comment-avatar-placeholder {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: 999rpx;
  background: #DBEAFE;

  text {
    font-size: $typography-font-size-xl;
    font-weight: 700;
    color: $color-semantic-info;
  }
}

.comment-body {
  flex: 1;
  min-width: 0;
}

.comment-meta {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  flex-wrap: wrap;
}

.comment-author {
  font-size: $typography-font-size-md;
  font-weight: 600;
  color: $color-text-primary;
}

.comment-role-tag {
  padding: 4rpx 16rpx;
  border-radius: $border-radius-sm;
  background: $color-semantic-info-bg;

  text {
    font-size: $typography-font-size-xs;
    font-weight: 500;
    color: $color-semantic-info;
  }
}

.comment-time {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.comment-text {
  display: block;
  margin-top: 8rpx;
  font-size: $typography-font-size-md;
  color: $color-text-body;
  line-height: 1.5;
}

.comment-actions {
  display: flex;
  gap: $spacing-xl;
  margin-top: 4rpx;
}

.comment-act {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.comment-act-icon {
  font-size: $typography-font-size-md;
  font-weight: 700;
  color: $color-text-hint;
}

.comment-act-num,
.comment-act text:last-child {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

// 老师回复
.teacher-reply {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  margin-left: 104rpx;
}

.reply-item {
  padding: $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
}

.reply-author {
  font-size: $typography-font-size-base;
  font-weight: 600;
  color: $color-semantic-info;
}

.reply-time {
  display: inline;
  margin-left: 16rpx;
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.reply-text {
  display: block;
  margin-top: 8rpx;
  font-size: $typography-font-size-base;
  color: $color-text-body;
  line-height: 1.5;
}

.show-more {
  padding: $spacing-lg 0;
  text-align: center;

  text {
    font-size: $typography-font-size-md;
    color: $color-text-hint;
  }
}

// ===== 底部留白 =====
.bottom-safe {
  height: calc(160rpx + env(safe-area-inset-bottom));
}

// ===== 底部评论输入栏 =====
.bottom-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: $spacing-md 48rpx calc(16rpx + env(safe-area-inset-bottom));
  background: #fff;
  box-shadow: 0 -2rpx 20rpx rgba(120, 53, 15, 0.04);
}

.input-box {
  flex: 1;
  min-width: 0;
  height: 80rpx;
  border-radius: 999rpx;
  background: #F1F5F9;
  padding: 0 32rpx;
  display: flex;
  align-items: center;
}

.comment-input {
  width: 100%;
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

.bottom-btn {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  border-radius: 999rpx;
  background: $color-semantic-info;
}

.bottom-btn-icon {
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: #fff;
}

// 补授徽章面板
.grant-btn {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.grant-panel {
  margin-top: $spacing-lg;
  padding: $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
}

.grant-title {
  display: block;
  margin-bottom: $spacing-sm;
  font-size: $typography-font-size-md;
  font-weight: 600;
  color: $color-text-primary;
}

.badge-pick-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
}

.badge-pick {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  padding: $spacing-sm $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-bg-page;

  &.selected {
    background: $color-semantic-info-bg;
    box-shadow: inset 0 0 0 2rpx $color-semantic-info;
  }
}

.badge-pick-icon {
  width: 36rpx;
  height: 36rpx;
  border-radius: $border-radius-sm;
}

.badge-pick-name {
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

.grant-hint {
  margin-bottom: $spacing-md;
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.grant-witness {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  margin: $spacing-md 0;
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

.grant-submit {
  margin-top: $spacing-md;
  background: $color-semantic-info;
  color: #fff;
  font-size: $typography-font-size-lg;
  border-radius: $border-radius-lg;
  // 必须给足内边距：无 padding 时按钮只有 21px 高，
  // 低于 44px 最小可点击目标，tap 落不准、真机上也难点。
  padding: $spacing-lg $spacing-xl;
  line-height: 1.4;
  border: none;
}
</style>
