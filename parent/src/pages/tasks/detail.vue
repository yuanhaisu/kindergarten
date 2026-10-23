<script setup>
import { computed, ref } from "vue"
import { onLoad, onUnload } from "@dcloudio/uni-app"
import { getCurrentStudentId, setCurrentStudentId } from "@/utils/currentStudent"
import { fetchParentHomeOverview, switchStudent } from "@/api/parent-home"
import { ensureLogin } from "@/utils/login"
import { getToken } from "@/utils/auth"
import config from "@/config/env"
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

const RITUAL_LABELS = { sticker: "贴纸", voice: "语音", class_star: "班级星星", text: "老师的话" }
// 小人卡上的图形：sticker_key 是后端透传的英文 key，客户端只做展示映射。
const STICKER_GLYPHS = { star: "⭐", heart: "❤️", thumb: "👍", flower: "🌸", smile: "😊", clap: "👏" }
const REACTIONS = [
  { key: "child_happy", label: "孩子很开心" },
  { key: "received", label: "收到了" },
  { key: "thanks", label: "谢谢老师" }
]

const detail = ref({})
const history = ref([])
// 后端分页上限 50。超出时页面要说明「只显示最近的」，
// 否则每日任务跑满一个月后，更早的提交会悄无声息地不见。
const historyTotal = ref(0)
const checkinContent = ref("")
const mediaFileID = ref(0)
const mediaName = ref("")
const loading = ref(false)
const loadError = ref("")
const taskId = ref(0)
// 深链（消息 action_path / 小程序码 scene）里携带的目标孩子。
// 它只用来提示「你正在看的是另一个孩子」，不参与取数：
// 家长端所有任务接口都必须用当前孩子的 student_id（D10 不回退）。
const linkedStudentId = ref(0)
const currentSid = ref(0)
const currentChildName = ref("")
// 提交历史里带 student_name，但新任务的通知最常见的情况是「还没提交」——历史为空。
// 提示条要说清「你正在看哪个孩子」，所以留一个从 overview 兜底的名字，
// 不与历史里的名字互相覆盖。
const childNameFallback = ref("")
const currentChildLabel = computed(() => currentChildName.value || childNameFallback.value || "当前孩子")
// 打开时是否还有未读补充：读标记清掉之后这个值仍为 true，用来解释「为什么刚才有红点」。
const hadUnread = ref(false)
const reactingIds = ref([])
const publicBusyIds = ref([])
// 撤下的二次确认走页面内的两步点击，而不是 uni.showModal：
// 原生弹窗在开发者工具里不进入视图层 DOM，CDP 既看不到也点不到，
// 这个「撤下已公开内容」的流程就无法用真实点击验证；两步点击同样是显式确认。
const pendingCancelId = ref(0)
const kidTarget = ref(null)
// 播放状态按 URL 归属：一条任务可能同时有主回应语音和补充回应语音，
// 用布尔会让所有语音按钮一起显示「正在播放…」。
const playingUrl = ref("")
let audio = null

const childMismatch = computed(() =>
  !!linkedStudentId.value && !!currentSid.value && linkedStudentId.value !== currentSid.value
)

// 小程序码 scene 形如 `t=123`，可能被 URL 编码成 `t%3D123`（或整串被编码一次）。
function parseSceneTaskId(scene) {
  if (!scene) return 0
  let raw = String(scene)
  try {
    raw = decodeURIComponent(raw)
  } catch (e) {
    // 非法编码就按原样解析，不要因此吞掉整个入口
  }
  const m = raw.match(/(?:^|&)t=(\d+)/)
  return m ? Number(m[1]) : 0
}

function ritualLabel(ritual) {
  return RITUAL_LABELS[ritual] || "回应"
}

function stickerGlyph(key) {
  return STICKER_GLYPHS[key] || "🌟"
}

// 后端给的是 UTC 的 RFC3339，直接截字符串会显示成 8 小时前的「凌晨」。
function shortTime(iso) {
  if (!iso) return ""
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return String(iso).slice(5, 16).replace("T", " ")
  const p = (n) => String(n).padStart(2, "0")
  return `${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

function canPublic() {
  // 个体任务不可公开；班级任务且任务开了公开才给入口。
  return !!detail.value.allow_public && !detail.value.target_student_id
}

// 审核状态由提交行字段推导：契约里 checkin-history 不带 public_status，
// 只有申请/取消接口返回它。已批准但仍可能是 public_visible=false（任务结束/转班），
// 所以「是否真的在墙上」一律看 public_visible，不用状态冒充。
function publicState(row) {
  if (row.public_revoked_at) return "revoked"
  if (row.public_approved_at) return "approved"
  if (row.is_public_requested) return "pending"
  return "none"
}

function publicLabel(row) {
  switch (publicState(row)) {
    case "pending":
      return "已申请，等待老师审核"
    case "approved":
      return row.public_visible ? "老师已通过，已在班级墙展示" : "老师已通过，暂未展示在班级墙"
    case "revoked":
      // 老师驳回与家长撤下在后端是同一个状态（都写 public_revoked_at），
      // 文案说成「已撤下」会让家长以为自己撤过——用中性的「未公开」。
      return "未公开"
    default:
      return ""
  }
}

// 撤回永远可用（任务关掉公开、已结束、孩子转班都能撤）；只有「申请」受内容与任务条件约束。
function publicActionLabel(row) {
  const state = publicState(row)
  if (state === "none") return "申请上班级墙"
  if (state === "revoked") return "重新申请"
  return pendingCancelId.value === row.id ? "确认撤下？" : "取消公开"
}

function showPublicAction(row) {
  const state = publicState(row)
  // 撤回永远可用：任务被关掉公开、已结束、甚至孩子转班，家长都必须能把内容撤下来。
  // 之前这里先判 canPublic()，老师一关公开开关，家长就再也点不到「撤下」了——直接违反契约。
  if (state === "pending" || state === "approved") return true
  if (!canPublic()) return false
  // 「申请」才受约束：任务没开公开 / 是个体任务 / 一键完成（无文字无媒体）都不给按钮，
  // 避免必然失败的点击。
  return !!(row.content || row.media_url)
}

async function loadData(id) {
  taskId.value = Number(id) || 0
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    // 当前孩子解析与 403 重试统一走 withStudentScope（与列表页同一条路径）。
    const scoped = await withStudentScope((sid) => Promise.all([
      fetchTaskDetail(id, { student_id: sid }),
      // 后端已分页（默认 20 条）；这里一次取到上限 50，并记下 total，
      // 超出一页时在页面上说明「只显示最近 N 条」，而不是静默少显示。
      fetchCheckinHistory(id, { student_id: sid, page: 1, page_size: 50 })
    ]))
    if (!scoped) {
      detail.value = {}
      history.value = []
      loadError.value = "未绑定孩子信息"
      return
    }
    const [d, h] = scoped.result
    detail.value = d.data || {}
    history.value = h.data?.list || []
    // 重新取数后确认态失效：否则重进页面时按钮还停在「确认撤下？」
    pendingCancelId.value = 0
    historyTotal.value = Number(h.data?.total || history.value.length)
    currentSid.value = scoped.sid
    currentChildName.value = history.value[0]?.student_name || ""
    hadUnread.value = !!detail.value.has_unread_supplement
    // 多孩提示条只在「消息里的孩子 ≠ 当前孩子」时出现；这时若当前孩子还没有任何提交，
    // 历史里拿不到姓名，提示条就会退化成「当前孩子」——那正是家长最需要看清是谁的时候。
    if (childMismatch.value && !currentChildName.value) resolveChildNameFallback(scoped.sid)
    // 读标记：详情已经渲染出来，红点必须清掉。失败只留日志，不能把整页挡成错误态。
    markTaskRead(id, scoped.sid).catch((err) => {
      console.warn("[tasks/detail] 清除未读标记失败", err?.message || err)
    })
  } catch (err) {
    detail.value = {}
    history.value = []
    // 404 在任务删除/结束后是常态入口（消息深链、扫码），要给一句人话而不是白屏。
    loadError.value = err?.code === 40400 ? "任务已结束或不存在" : (err?.message || "加载任务失败")
    // 加载失败时提示条仍在：这条路径（任务对当前孩子不可见）同样要能说清「你看的是谁」。
    if (childMismatch.value && currentSid.value) resolveChildNameFallback(currentSid.value)
  } finally {
    loading.value = false
  }
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

async function handleCheckin() {
  if (!detail.value.id) return
  const content = checkinContent.value.trim()
  if (!content && !mediaFileID.value) {
    uni.showToast({ title: "请填写内容或上传照片", icon: "none" })
    return
  }
  try {
    // student_id 走 query（第三参）：body 里的会被后端丢弃
    const res = await checkinTask(detail.value.id, {
      content,
      media_file_id: mediaFileID.value || undefined
    }, currentSid.value || getCurrentStudentId())
    checkinContent.value = ""
    mediaFileID.value = 0
    mediaName.value = ""
    detail.value.submitted_current_period = true
    uni.showToast({ title: res.data?.message || "已提交", icon: "success" })
    loadData(detail.value.id)
  } catch (err) {
    uni.showToast({ title: err?.message || "打卡失败", icon: "none" })
  }
}

async function quickCheckin() {
  if (!detail.value.id) return
  try {
    const res = await checkinTask(detail.value.id, { content: "" }, currentSid.value || getCurrentStudentId())
    mediaFileID.value = 0
    mediaName.value = ""
    detail.value.submitted_current_period = true
    uni.showToast({ title: res.data?.message || "已提交", icon: "success" })
    loadData(detail.value.id)
  } catch (e) {
    uni.showToast({ title: e?.message || "提交失败", icon: "none" })
  }
}

async function chooseMedia() {
  try {
    await ensureLogin()
  } catch (e) {
    return
  }
  uni.chooseImage({
    count: 1,
    success: (res) => {
      const path = res.tempFilePaths?.[0]
      if (!path) return
      uni.showLoading({ title: "上传中" })
      uni.uploadFile({
        url: config.uploadURL,
        filePath: path,
        name: "file",
        formData: { biz_type: "task" },
        header: { Authorization: `Bearer ${getToken()}` },
        success: (up) => {
          try {
            const body = JSON.parse(up.data || "{}")
            if (body.code === 0 && body.data?.id) {
              mediaFileID.value = body.data.id
              mediaName.value = body.data.file_name || "已上传"
              uni.showToast({ title: "上传成功", icon: "success" })
            } else {
              uni.showToast({ title: body.message || "上传失败", icon: "none" })
            }
          } catch (e) {
            uni.showToast({ title: "上传失败", icon: "none" })
          }
        },
        fail: () => uni.showToast({ title: "上传失败", icon: "none" }),
        complete: () => uni.hideLoading()
      })
    }
  })
}

async function react(item, reaction) {
  if (!item?.id || reactingIds.value.includes(item.id)) return
  reactingIds.value = [...reactingIds.value, item.id]
  try {
    const res = await reactResponse(item.id, reaction, currentSid.value || getCurrentStudentId())
    // 同一家长再次提交是更新自己那一行，直接回显服务端结果。
    item.my_reaction = res.data?.reaction || reaction
  } catch (err) {
    uni.showToast({ title: err?.message || "反馈失败", icon: "none" })
  } finally {
    reactingIds.value = reactingIds.value.filter((id) => id !== item.id)
  }
}

function mergePublicResult(row, data) {
  if (!data) return
  // 申请/取消接口返回的是完整公开状态，直接覆盖到这一行，避免整页重拉。
  row.is_public_requested = !!data.is_public_requested
  row.public_scope_confirmed = !!data.public_scope_confirmed
  row.public_approved_at = data.public_approved_at || null
  row.public_revoked_at = data.public_revoked_at || null
  row.public_visible = !!data.public_visible
}

async function togglePublic(row) {
  if (!row?.id || publicBusyIds.value.includes(row.id)) return
  const state = publicState(row)
  const cancelling = state === "pending" || state === "approved"
  if (cancelling && pendingCancelId.value !== row.id) {
    // 第一次点只进入确认态，不碰接口；换一行会把这行退回普通态。
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
    mergePublicResult(row, res.data)
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
  stopVoice()
  audio = uni.createInnerAudioContext()
  audio.src = url
  audio.onEnded(() => { playingUrl.value = "" })
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

onLoad((opts) => {
  linkedStudentId.value = Number(opts?.student_id || 0)
  // 先落一次当前孩子：任务对当前孩子不可见时（个体任务被人代开、跨班）详情会 404，
  // 那条「消息是另一个孩子的」提示不能因为加载失败就消失，否则家长看到的是「任务不存在」。
  currentSid.value = getCurrentStudentId() || 0
  const id = Number(opts?.id || 0) || parseSceneTaskId(opts?.scene)
  if (!id) {
    loadError.value = "任务不存在"
    return
  }
  loadData(id)
})

onUnload(stopVoice)
</script>

<template>
  <view class="page">
    <!-- 提示放在加载状态之外：任务对当前孩子不可见时详情会 404，
         这条「换孩子」的出口不能跟着消失。 -->
    <view v-if="childMismatch" class="notice">
      <text class="notice-text">这条消息是另一个孩子的任务，当前看到的是「{{ currentChildLabel }}」</text>
      <view class="notice-btn" @click="switchToMessageChild">切换查看</view>
    </view>
    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="loadError" class="state">
      <text class="state-text">{{ loadError }}</text>
      <view v-if="taskId" class="state-btn" @click="loadData(taskId)">重试</view>
    </view>
    <template v-else>

      <view class="card">
        <text class="title">{{ detail.title }}</text>
        <text class="desc">{{ detail.description }}</text>
        <text class="intro">{{ detail.intro }}</text>
        <view v-if="detail.requirements?.length" class="reqs">
          <text v-for="(r, i) in detail.requirements" :key="i" class="req">{{ i + 1 }}. {{ r }}</text>
        </view>
        <view class="stats">
          <text>{{ detail.frequency === "daily" ? "每日任务" : detail.frequency === "weekly" ? "每周任务" : "一次性任务" }}</text>
          <text v-if="detail.learning_link">和幼儿园学习相关</text>
        </view>
        <view v-if="detail.learning_link" class="link-box">
          <text class="link-label">学习关联</text>
          <text class="link-text">{{ detail.learning_link }}</text>
        </view>
      </view>

      <view v-if="hadUnread" class="unread-tip">
        <text>老师又给你留了一句悄悄话，往下面看 👇</text>
      </view>

      <view v-if="detail.responses?.length" class="section">
        <text class="section-title">老师的回应</text>
        <view v-for="r in detail.responses" :key="r.id" class="response-card">
          <view class="rc-head">
            <text class="rc-ritual">{{ ritualLabel(r.ritual) }}</text>
            <text class="rc-time">{{ shortTime(r.created_at) }}</text>
          </view>
          <view v-if="r.ritual === 'sticker' && r.sticker_key" class="rc-sticker">
            <text>{{ stickerGlyph(r.sticker_key) }}</text>
          </view>
          <view v-if="r.media_url" class="rc-voice" @click="playVoice(r.media_url)">
            <text>{{ playingUrl === r.media_url ? "正在播放…" : "▶ 播放老师的语音" }}</text>
          </view>
          <text v-if="r.content" class="rc-content">{{ r.content }}</text>

          <view class="rc-actions">
            <view class="kid-btn" @click="openKidMode(r)">给孩子看</view>
          </view>

          <view class="reactions">
            <view
              v-for="rc in REACTIONS"
              :key="rc.key"
              class="reaction"
              :class="{ active: r.my_reaction === rc.key, busy: reactingIds.includes(r.id) }"
              @click="react(r, rc.key)"
            >
              <text class="reaction-text">{{ rc.label }}</text>
            </view>
          </view>

          <view v-if="r.supplements?.length" class="supplements">
            <view v-for="sp in r.supplements" :key="sp.id" class="supplement">
              <view class="sp-head">
                <text class="sp-label">老师又补了一句</text>
                <text class="rc-time">{{ shortTime(sp.created_at) }}</text>
              </view>
              <view v-if="sp.ritual === 'sticker' && sp.sticker_key" class="sp-sticker">
                <text>{{ stickerGlyph(sp.sticker_key) }}</text>
              </view>
              <view v-if="sp.media_url" class="rc-voice" @click="playVoice(sp.media_url)">
                <text>{{ playingUrl === sp.media_url ? "正在播放…" : "▶ 播放老师的语音" }}</text>
              </view>
              <text v-if="sp.content" class="rc-content">{{ sp.content }}</text>
              <view class="rc-actions">
                <view class="kid-btn" @click="openKidMode(sp)">给孩子看</view>
              </view>
              <view class="reactions">
                <view
                  v-for="rc in REACTIONS"
                  :key="rc.key"
                  class="reaction"
                  :class="{ active: sp.my_reaction === rc.key, busy: reactingIds.includes(sp.id) }"
                  @click="react(sp, rc.key)"
                >
                  <text class="reaction-text">{{ rc.label }}</text>
                </view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <view v-if="detail.id && !detail.submitted_current_period" class="checkin-card">
        <text class="section-title">今日提交</text>
        <view class="media-btn" @click="chooseMedia">
          <text>{{ mediaName || "＋ 添加照片（可选）" }}</text>
        </view>
        <view class="media-btn" @click="quickCheckin">
          <text>今天完成了（一键提交）</text>
        </view>
        <textarea v-model="checkinContent" class="textarea" placeholder="记录今天的内容..." :maxlength="500" />
        <button class="checkin-btn" @click="handleCheckin">打卡</button>
      </view>

      <view class="section">
        <text class="section-title">打卡历史</text>
        <view v-for="h in history" :key="h.id" class="history-item">
          <view class="hi-head">
            <text class="h-date">{{ h.date }}</text>
            <text v-if="publicLabel(h)" class="hi-status">{{ publicLabel(h) }}</text>
          </view>
          <text class="h-content">{{ h.content || "（一键完成）" }}</text>
          <view v-if="showPublicAction(h)" class="hi-actions">
            <view
              class="hi-btn"
              :class="{ busy: publicBusyIds.includes(h.id) }"
              @click="togglePublic(h)"
            >
              <text>{{ publicActionLabel(h) }}</text>
            </view>
          </view>
        </view>
        <text v-if="historyTotal > history.length" class="history-hint">
          仅显示最近 {{ history.length }} 条提交（共 {{ historyTotal }} 条）
        </text>
      </view>
    </template>

    <!-- 「给孩子看」：放大贴纸 / 播放语音 / 老师的话，家长把手机递给孩子用 -->
    <view v-if="kidTarget" class="kid-mask" @click="closeKidMode">
      <view class="kid-body" @click.stop>
        <view v-if="kidTarget.ritual === 'sticker' && kidTarget.sticker_key" class="kid-sticker">
          <text>{{ stickerGlyph(kidTarget.sticker_key) }}</text>
        </view>
        <text v-if="kidTarget.content" class="kid-content">{{ kidTarget.content }}</text>
        <view v-if="kidTarget.media_url" class="kid-play" @click="playVoice(kidTarget.media_url)">
          <text>{{ playingUrl === kidTarget.media_url ? "正在播放…" : "▶ 听老师说" }}</text>
        </view>
        <text class="kid-hint">轻触空白处关闭</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg $component-page-padding-x $spacing-xl; background: $color-bg-page; }
.state { display: flex; flex-direction: column; gap: $spacing-lg; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }
.state-btn { padding: $spacing-sm 40rpx; border-radius: $border-radius-full; background: $color-brand-primary-bg; color: $color-brand-primary; font-size: $typography-font-size-sm; }
.notice { display: flex; align-items: center; gap: $spacing-md; padding: $spacing-md; margin-bottom: $spacing-lg; border-radius: $border-radius-lg; background: $color-semantic-warning-bg; }
.notice-text { flex: 1; color: $color-semantic-warning; font-size: $typography-font-size-sm; line-height: 1.5; }
.notice-btn { flex-shrink: 0; padding: 10rpx $spacing-lg; border-radius: $border-radius-full; background: $color-semantic-warning; color: $color-text-on-brand; font-size: $typography-font-size-xs; }
.card { padding: $component-card-padding; border-radius: $component-card-radius; background: $color-bg-card; box-shadow: $shadow-sm; }
.title { display: block; color: $color-text-primary; font-size: $typography-font-size-2xl; font-weight: $typography-font-weight-bold; }
.desc { display: block; margin-top: $spacing-sm; color: $color-text-body; font-size: $typography-font-size-base; }
.intro { display: block; margin-top: $spacing-md; color: $color-text-muted; font-size: $typography-font-size-sm; line-height: 1.6; }
.reqs { margin-top: $spacing-md; }
.req { display: block; margin-top: 6rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
.stats { display: flex; gap: $spacing-lg; margin-top: 20rpx; color: $color-semantic-info; font-size: $typography-font-size-sm; }
.link-box { margin-top: 20rpx; padding: $spacing-md; border-radius: $border-radius-lg; background: $color-brand-primary-bg; }
.link-label { display: block; color: $color-brand-primary; font-size: $typography-font-size-xs; font-weight: $typography-font-weight-semibold; }
.link-text { display: block; margin-top: 6rpx; color: $color-text-body; font-size: $typography-font-size-sm; line-height: 1.5; }
.unread-tip { margin-top: $spacing-lg; padding: $spacing-md; border-radius: $border-radius-lg; background: $color-semantic-warning-bg; color: $color-semantic-warning; font-size: $typography-font-size-sm; }
.section { margin-top: $spacing-lg; }
.section-title { display: block; margin-bottom: $spacing-md; color: $color-text-primary; font-size: $typography-font-size-md; font-weight: $typography-font-weight-bold; }
.response-card { padding: $spacing-lg; margin-bottom: $spacing-md; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: $shadow-sm; }
.rc-head { display: flex; align-items: center; justify-content: space-between; }
.rc-ritual { color: $color-brand-primary; font-size: $typography-font-size-sm; font-weight: $typography-font-weight-semibold; }
.rc-time { color: $color-text-hint; font-size: $typography-font-size-xs; }
.rc-sticker { margin-top: $spacing-sm; font-size: 72rpx; line-height: 1.2; }
.rc-voice { display: flex; align-items: center; justify-content: center; height: 76rpx; margin-top: $spacing-sm; border-radius: $border-radius-full; background: $color-brand-primary-bg; color: $color-brand-primary; font-size: $typography-font-size-sm; }
.rc-content { display: block; margin-top: $spacing-sm; color: $color-text-primary; font-size: $typography-font-size-base; line-height: 1.6; }
.rc-actions { display: flex; justify-content: flex-end; margin-top: $spacing-sm; }
.kid-btn { padding: $spacing-xs $spacing-lg; border-radius: $border-radius-full; background: $color-bg-page; color: $color-brand-primary; font-size: $typography-font-size-xs; }
.reactions { display: flex; gap: $spacing-sm; margin-top: $spacing-md; }
.reaction { flex: 1; padding: $spacing-sm 0; border-radius: $border-radius-full; background: $color-bg-page; text-align: center; }
.reaction.active { background: $color-brand-primary; }
.reaction.busy { opacity: 0.6; }
.reaction-text { color: $color-text-muted; font-size: $typography-font-size-xs; }
.reaction.active .reaction-text { color: $color-text-on-brand; }
.supplements { margin-top: $spacing-md; padding-top: $spacing-md; border-top: 2rpx solid $color-border-hairline; }
.supplement { padding: $spacing-md; margin-top: $spacing-sm; border-radius: $border-radius-lg; background: $color-bg-page; }
.sp-head { display: flex; align-items: center; justify-content: space-between; }
.sp-label { color: $color-semantic-info; font-size: $typography-font-size-xs; font-weight: $typography-font-weight-semibold; }
.sp-sticker { margin-top: 6rpx; font-size: 56rpx; line-height: 1.2; }
.checkin-card { margin-top: $spacing-lg; padding: $component-card-padding; border-radius: $component-card-radius; background: $color-bg-card; box-shadow: $shadow-sm; }
.media-btn { display: flex; align-items: center; justify-content: center; height: 96rpx; margin-bottom: $spacing-md; border: 2rpx dashed $color-border-hairline; border-radius: $border-radius-lg; color: $color-text-muted; font-size: $typography-font-size-sm; }
.textarea { width: 100%; height: 160rpx; padding: 20rpx; border: 2rpx solid $color-border-hairline; border-radius: $border-radius-lg; font-size: $typography-font-size-base; box-sizing: border-box; }
.checkin-btn { width: 100%; height: 80rpx; margin-top: 20rpx; border-radius: $border-radius-full; background: $color-brand-primary; color: $color-text-on-brand; font-size: $typography-font-size-md; font-weight: $typography-font-weight-bold; line-height: 80rpx; }
.history-item { padding: 20rpx; margin-bottom: $spacing-sm; border-radius: $border-radius-lg; background: $color-bg-card; }
.hi-head { display: flex; align-items: center; justify-content: space-between; }
.h-date { color: $color-text-muted; font-size: $typography-font-size-xs; }
.hi-status { color: $color-semantic-info; font-size: $typography-font-size-xs; }
.h-content { display: block; margin-top: 6rpx; color: $color-text-primary; font-size: $typography-font-size-sm; }
.hi-actions { display: flex; justify-content: flex-end; margin-top: $spacing-sm; }
.hi-btn { padding: $spacing-xs $spacing-lg; border-radius: $border-radius-full; background: $color-brand-primary-bg; color: $color-brand-primary; font-size: $typography-font-size-xs; }
.hi-btn.busy { opacity: 0.6; }
.history-hint { display: block; margin-top: $spacing-xs; color: $color-text-muted; font-size: $typography-font-size-xs; }
// 全屏遮罩只此一处（设计稿无对应 token），按 token 规范保留局部硬编码并在此说明。
.kid-mask { position: fixed; top: 0; left: 0; right: 0; bottom: 0; z-index: 999; display: flex; align-items: center; justify-content: center; background: rgba(0, 0, 0, 0.72); }
.kid-body { display: flex; flex-direction: column; align-items: center; padding: 64rpx 48rpx; width: 80%; border-radius: $border-radius-2xl; background: $color-bg-card; }
.kid-sticker { font-size: 220rpx; line-height: 1.1; }
.kid-content { margin-top: $spacing-lg; color: $color-text-primary; font-size: 44rpx; font-weight: $typography-font-weight-bold; text-align: center; line-height: 1.5; }
.kid-play { margin-top: $spacing-lg; padding: $spacing-md 48rpx; border-radius: $border-radius-full; background: $color-brand-primary; }
.kid-play text { color: $color-text-on-brand; font-size: $typography-font-size-md; }
.kid-hint { margin-top: $spacing-lg; color: $color-text-hint; font-size: $typography-font-size-xs; }
</style>
