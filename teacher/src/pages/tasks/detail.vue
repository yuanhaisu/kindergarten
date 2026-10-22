<script setup>
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { fetchTeacherTask, fetchTeacherTaskCheckins, respondTeacherTask, batchRespondTeacherTask, updateTeacherTask } from "@/api/teacher-tasks"
import config from "@/config/env"
import { getToken } from "@/utils/auth"

const taskId = ref(0)
const rows = ref([])
const task = ref({})
const taskError = ref("")
// 与后端 batchRespondAllowed 对齐：只有每日/每周的班级任务提供汇总入口。
// 任务详情没加载成功时（task.id 为空）不下结论，避免把加载失败误报成「需逐条回应」。
const canBatch = computed(() =>
  !!task.value.id &&
  task.value.target_student_id === 0 &&
  ["daily", "weekly"].includes(task.value.frequency)
)
// 旧周期（不在 period_start..period_end 内）的待回应只能逐条处理。
function isCurrentPeriod(row) {
  if (!task.value.period_start || !task.value.period_end || !row.date) return true
  return row.date >= task.value.period_start && row.date <= task.value.period_end
}
const edit = ref({ title: "", description: "", learning_link: "", allow_public: false })
const loading = ref(false)
// 防连点：第二次请求必然拿到 0 条，紧跟在「已回应 N 条」后面会被读成失败。
const batchBusy = ref(false)
// 单条回应的门闩。贴纸是并排的小按钮，连点两下会真发两次请求，
// 第二下必然落到 409，用户看到的是「回应失败」——明明第一次已经成功了。
// 闩在「行」上而不是全页开关：A 的请求还在飞时去回应 B 是正常操作，不该一起被挡住。
const respondingIds = ref([])
const isResponding = (id) => respondingIds.value.includes(id)
const status = ref("pending")
const stickers = ["star", "heart", "thumb"]
const recorder = uni.getRecorderManager()
let recordingRow = null
const recording = ref(false)

recorder.onStop((res) => {
  recording.value = false
  const row = recordingRow
  recordingRow = null
  if (!row || !res?.tempFilePath) return
  uni.showLoading({ title: "上传语音" })
  uni.uploadFile({
    url: config.uploadURL,
    filePath: res.tempFilePath,
    name: "file",
    formData: { biz_type: "task" },
    header: { Authorization: `Bearer ${getToken()}` },
    success: (up) => {
      try {
        const body = JSON.parse(up.data || "{}")
        if (body.code !== 0 || !body.data?.id) {
          uni.showToast({ title: body.message || "上传失败", icon: "none" })
          return
        }
        respondVoice(row, body.data.id)
      } catch (e) {
        uni.showToast({ title: "上传失败", icon: "none" })
      }
    },
    fail: () => uni.showToast({ title: "上传失败", icon: "none" }),
    complete: () => uni.hideLoading()
  })
})

async function respondVoice(row, mediaFileID) {
  try {
    await respondTeacherTask(taskId.value, row.id, { ritual: "voice", media_file_id: mediaFileID })
    uni.showToast({ title: "语音已发送", icon: "success" })
    load()
  } catch (e) {
    uni.showToast({ title: e?.message || "语音回应失败", icon: "none" })
  }
}

function startRecord(row) {
  if (recording.value) return
  recordingRow = row
  recording.value = true
  recorder.start({ format: "mp3", duration: 15000 })
}

function stopRecord() {
  if (recording.value) recorder.stop()
}

async function loadTask() {
  try {
    const res = await fetchTeacherTask(taskId.value)
    task.value = res.data || {}
    taskError.value = ""
    edit.value = {
      title: task.value.title || "",
      description: task.value.description || "",
      learning_link: task.value.learning_link || "",
      allow_public: !!task.value.allow_public
    }
  } catch (e) {
    // 任务详情只影响编辑区与汇总入口，不阻塞提交列表；但要显式说明加载失败。
    taskError.value = e?.message || "任务详情加载失败"
  }
}

async function saveEdit() {
  if (!edit.value.title.trim() || !edit.value.learning_link.trim()) {
    uni.showToast({ title: "标题和学习关联必填", icon: "none" })
    return
  }
  try {
    await updateTeacherTask(taskId.value, {
      title: edit.value.title,
      description: edit.value.description,
      learning_link: edit.value.learning_link,
      allow_public: edit.value.allow_public,
      target_student_id: task.value.target_student_id || 0,
      frequency: task.value.frequency,
      start_date: task.value.start_date,
      end_date: task.value.end_date,
      recognition_ritual: task.value.recognition_ritual || "sticker",
      star_scope: task.value.star_scope || "task",
      star_goal: task.value.star_goal || 0,
      status: task.value.status ?? 1
    })
    uni.showToast({ title: "已保存", icon: "success" })
    loadTask()
  } catch (e) {
    uni.showToast({ title: e?.message || "保存失败", icon: "none" })
  }
}

// 只刷新周期字段，不覆盖编辑区里未保存的输入。
async function refreshPeriod() {
  try {
    const res = await fetchTeacherTask(taskId.value)
    if (res.data?.period_start) task.value.period_start = res.data.period_start
    if (res.data?.period_end) task.value.period_end = res.data.period_end
  } catch (e) {
    // 周期刷新失败不影响提交列表；旧区间最多导致「本周期」判断短暂陈旧
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
      refreshPeriod()
    ])
    if (seq !== loadSeq) return
    rows.value = res.data?.list || []
  } catch (e) {
    if (seq !== loadSeq) return
    uni.showToast({ title: "加载提交失败", icon: "none" })
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

async function respond(row, sticker) {
  // 批量在飞时单条必然撞车：那一行可能刚被批量关掉，再发就 409。两个闩互相感知。
  if (isResponding(row.id) || batchBusy.value) return
  respondingIds.value = [...respondingIds.value, row.id]
  try {
    await respondTeacherTask(taskId.value, row.id, { ritual: "sticker", sticker_key: sticker })
    uni.showToast({ title: "已回应", icon: "success" })
    // 与 batchRespond 同形状：等刷新落地再解锁，否则列表还在飞行期间按钮又活了。
    // 这里不会 reject——load() 自己吞掉异常并显示「加载提交失败」。
    await Promise.race([load(), new Promise((r) => setTimeout(r, 3000))])
  } catch (e) {
    uni.showToast({ title: e?.message || "回应失败", icon: "none" })
  } finally {
    respondingIds.value = respondingIds.value.filter((id) => id !== row.id)
  }
}

async function batchRespond() {
  // 反向也必须拦：单条在飞时点「一键送星」，会撞上正在关闭的那一行，
  // 返回里就会带 failed_ids，用户看到「已回应 1 条，1 条失败」——又是一种假失败。
  if (batchBusy.value || respondingIds.value.length) return
  batchBusy.value = true
  try {
    const res = await batchRespondTeacherTask(taskId.value, { ritual: "sticker", sticker_key: "star" })
    // 只关闭「本周期」的待回应提交；并发/重复点击落败时 count 为 0，不能报成「已回应 0 条」。
    const count = res.data?.responded_count || 0
    const failed = res.data?.failed_ids?.length || 0
    if (count === 0) {
      uni.showToast({ title: "本周期已无待回应提交", icon: "none" })
    } else if (failed > 0) {
      uni.showToast({ title: `已回应 ${count} 条，${failed} 条失败`, icon: "none" })
    } else {
      uni.showToast({ title: `已回应 ${count} 条`, icon: "success" })
    }
    // 必须等刷新落地再解锁：load() 不 await 的话，finally 会立刻放开门闩，
    // 列表还在飞行的这段时间里按钮仍可点，第二次点击会真的发请求并弹「已无待回应」。
    // 但要封顶：uni.request 没设 timeout，走微信默认 60s，卡住时门闩不能一直锁着。
    await Promise.race([load(), new Promise((r) => setTimeout(r, 3000))])
  } catch (e) {
    uni.showToast({ title: e?.message || "批量回应失败", icon: "none" })
  } finally {
    batchBusy.value = false
  }
}

function switchStatus(s) {
  status.value = s
  load()
}

onLoad((opts) => {
  taskId.value = Number(opts?.id || 0)
  loadTask()
  load()
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
      <view class="switch-row">
        <text>允许申请公开</text>
        <switch :checked="edit.allow_public" @change="edit.allow_public = $event.detail.value" />
      </view>
      <view class="batch" @click="saveEdit">保存</view>
    </view>
    <view class="tabs">
      <view class="tab" :class="{ active: status === 'pending' }" @click="switchStatus('pending')">待回应</view>
      <view class="tab" :class="{ active: status === 'responded' }" @click="switchStatus('responded')">已回应</view>
    </view>
    <!-- 与后端一致：汇总回应只关闭当前周期，旧周期的待回应走单条回应 -->
    <view v-if="status === 'pending' && rows.length && canBatch" class="batch" :class="{ busy: batchBusy || respondingIds.length }" @click="batchRespond">一键送星（本周期）</view>
    <!-- 任务详情没加载成功时不作结论，避免把加载失败误报成「需逐条回应」 -->
    <view v-if="status === 'pending' && rows.length && task.id && !canBatch" class="batch-hint">一次性任务与个体任务需逐条回应</view>
    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="!rows.length" class="state">暂无提交</view>
    <view v-for="row in rows" :key="row.id" class="card">
      <text class="name">{{ row.student?.name || `学生 ${row.student_id}` }}</text>
      <text class="content">{{ row.content || "（一键完成）" }}</text>
      <text v-if="status === 'pending' && !isCurrentPeriod(row)" class="old-period">旧周期 · 请逐条回应</text>
      <text v-if="row.media_url" class="media">已附照片/语音</text>
      <view v-if="status === 'pending'" class="actions" :class="{ busy: batchBusy || isResponding(row.id) }">
        <view v-for="s in stickers" :key="s" class="sticker" @click="respond(row, s)">{{ s }}</view>
        <view v-if="!recording" class="sticker" @click="startRecord(row)">录音</view>
        <view v-else class="sticker recording" @click="stopRecord">停止</view>
      </view>
      <text v-else class="done">{{ row.response?.content || "已回应" }}</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg 28rpx 48rpx; background: $color-bg-page; }
.edit-card { margin-bottom: $spacing-lg; }
.input, .textarea { box-sizing: border-box; width: 100%; margin-top: $spacing-sm; padding: $spacing-sm $spacing-md; border-radius: $border-radius-lg; background: $color-bg-page; font-size: $typography-font-size-base; }
.textarea { min-height: 140rpx; }
.switch-row { display: flex; align-items: center; justify-content: space-between; margin-top: $spacing-md; color: $color-text-body; font-size: $typography-font-size-sm; }
.tabs { display: flex; gap: $spacing-md; margin-bottom: $spacing-lg; }
.tab { flex: 1; text-align: center; padding: $spacing-md 0; border-radius: $border-radius-lg; background: $color-bg-card; color: $color-text-muted; }
.tab.active { background: $color-brand-primary; color: #fff; }
.batch { margin-bottom: $spacing-lg; padding: $spacing-md; text-align: center; border-radius: $border-radius-lg; background: $color-semantic-success-bg; color: $color-semantic-success; }
.batch.busy { opacity: 0.6; }
.state { padding: 120rpx 0; text-align: center; color: $color-text-hint; }
.batch-hint { margin-bottom: $spacing-lg; padding: $spacing-md; text-align: center; border-radius: $border-radius-lg; background: $color-bg-page; color: $color-text-muted; font-size: $typography-font-size-sm; }
.old-period { display: block; margin-top: $spacing-xs; color: $color-semantic-warning; font-size: $typography-font-size-sm; }
.error-card { background: $color-semantic-danger-bg; }
.error-title { display: block; color: $color-semantic-danger; font-size: $typography-font-size-md; font-weight: 700; }
.error-hint { display: block; margin-top: 8rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
.card { margin-bottom: $spacing-md; padding: $spacing-lg; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: $shadow-sm; }
.name { display: block; font-size: $typography-font-size-md; font-weight: 700; color: $color-text-primary; }
.content { display: block; margin-top: $spacing-xs; font-size: $typography-font-size-base; color: $color-text-body; }
.media { display: block; margin-top: $spacing-xs; font-size: $typography-font-size-sm; color: $color-text-muted; }
.actions { display: flex; gap: $spacing-md; margin-top: $spacing-md; }
.actions.busy { opacity: 0.6; }
.sticker { padding: $spacing-xs $spacing-lg; border-radius: $border-radius-full; background: $color-bg-page; color: $color-brand-primary; font-size: $typography-font-size-sm; }
.recording { background: $color-semantic-danger-bg; color: $color-semantic-danger; }
.done { display: block; margin-top: $spacing-sm; color: $color-semantic-success; font-size: $typography-font-size-sm; }
</style>
