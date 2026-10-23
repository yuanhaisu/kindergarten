<script setup>
import { computed, ref } from "vue"
import { onLoad, onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { createTeacherTask, fetchTeacherTasks } from "@/api/teacher-tasks"
import { fetchTeacherClassStudents } from "@/api/teacher-class"

function ymd(d) {
  const p = n => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}
function addDays(n) {
  const d = new Date()
  d.setDate(d.getDate() + n)
  return ymd(d)
}

function defaultForm() {
  return {
    title: "", description: "", learning_link: "", frequency: "once",
    start_date: ymd(new Date()), end_date: addDays(7),
    target_student_id: 0, allow_public: false,
    recognition_ritual: "sticker", star_scope: "task", star_goal: 0
  }
}

const tasks = ref([])
const loadError = ref("")
const students = ref([])
// 学生名单失败时不能静默变成「只有班级可选」——老师会以为本班没有可指派的孩子。
const studentsError = ref("")
const form = ref(defaultForm())
const studentOptions = computed(() => ["班级", ...students.value.map(s => s.name)])
const studentIndex = ref(0)
const freqOptions = ["一次性", "每日", "每周"]
const freqValues = ["once", "daily", "weekly"]
const freqIndex = ref(0)
const ritualOptions = ["贴纸", "语音", "班级星星"]
const ritualValues = ["sticker", "voice", "class_star"]
const ritualIndex = ref(0)
const FREQ_LABELS = { once: "一次性", daily: "每日", weekly: "每周" }
const RITUAL_LABELS = { sticker: "贴纸", voice: "语音", class_star: "班级星星" }

// 班级星星罐的配置端点在管理端（C 切片），老师端当前没有读取/设置入口，
// 因此无法判断「是否已配置」。这里不猜：选项可见但不可选，并说明去哪配置。
function explainClassStarJar() {
  uni.showToast({ title: "班级星星罐暂未开放（待管理端配置后可用）", icon: "none" })
}

function freqLabel(v) {
  return FREQ_LABELS[v] || v || "—"
}
function ritualLabel(v) {
  return RITUAL_LABELS[v] || v || "—"
}

// star_count 由后端按「主回应 ritual=class_star 的提交数」算好（period 覆盖 N 条计 N 星）。
// 任务罐：count/goal；班级罐：只显示本任务贡献，总进度要等班级罐端点（C 切片）。
function starProgress(task) {
  // star_count 为 null/undefined 表示「没拿到」而不是 0：不画 0% 的假进度。
  if (task.star_count == null) return null
  if (task.star_scope === "class") return null
  const goal = Number(task.star_goal || 0)
  if (goal <= 0) return null
  return Math.min(100, Math.round((Number(task.star_count) / goal) * 100))
}

function starSummary(task) {
  if (task.star_count == null) return ""
  const count = Number(task.star_count)
  if (task.star_scope === "class") return `班级星星罐 · 本任务已集 ${count} 星`
  const goal = Number(task.star_goal || 0)
  if (goal <= 0) return ""
  // 目标下调到低于当前星数时显示超额，不出现 >100% 的进度条（D6）。
  if (count > goal) return `星星已超额 · ${count}/${goal}`
  if (count === goal) return `星星已集满 · ${count}/${goal}`
  return `星星 ${count}/${goal}`
}

// 个体标记独立于星星信息：没有星星信息也要能看出是个体任务。
function extraMeta(task) {
  const parts = []
  const stars = starSummary(task)
  if (stars) parts.push(stars)
  if (task.target_student_id > 0) parts.push("个体任务")
  return parts.join(" · ")
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

async function loadTasks() {
  // onShow 每次可见都会调用：失败时保留旧列表，但必须显式说明「这是旧数据」，
  // 否则失权/故障会被伪装成「没有任务」。
  try {
    const res = await fetchTeacherTasks({ page: 1, page_size: 20 })
    tasks.value = res.data?.list || []
    loadError.value = ""
  } catch (e) {
    loadError.value = e?.message || "加载任务失败"
  } finally {
    uni.stopPullDownRefresh()
  }
}

// 汇总回应只关当前周期：两个数字不一致时把本周期数量一并说明，
// 避免列表的「待回应 N」与详情页「一键送星」实际关闭的条数对不上。
function pendingLabel(task) {
  const total = task.pending_responses || 0
  const period = task.pending_period
  if (typeof period !== "number" || period === total) return `待回应 ${total}`
  return `待回应 ${total}（本周期 ${period}）`
}

async function addTask() {
  if (!form.value.title.trim() || !form.value.learning_link.trim()) {
    uni.showToast({ title: "请填写任务标题和学习关联", icon: "none" })
    return
  }
  if (form.value.end_date < form.value.start_date) {
    uni.showToast({ title: "结束日期不能早于开始日期", icon: "none" })
    return
  }
  if (form.value.recognition_ritual === "class_star" && (!Number(form.value.star_goal) || Number(form.value.star_goal) <= 0)) {
    uni.showToast({ title: "选择班级星星时请设置星星目标", icon: "none" })
    return
  }
  try {
    await createTeacherTask({
      ...form.value,
      star_goal: Number(form.value.star_goal) || 0
    })
    form.value = defaultForm()
    studentIndex.value = 0
    freqIndex.value = 0
    ritualIndex.value = 0
    uni.showToast({ title: "已创建", icon: "success" })
    loadTasks()
  } catch (e) {
    uni.showToast({ title: e?.message || "创建失败", icon: "none" })
  }
}

function onStudentChange(e) {
  studentIndex.value = Number(e.detail.value)
  form.value.target_student_id = studentIndex.value === 0 ? 0 : students.value[studentIndex.value - 1].student_id
  if (form.value.target_student_id > 0) form.value.allow_public = false
}
function onFreqChange(e) {
  freqIndex.value = Number(e.detail.value)
  form.value.frequency = freqValues[freqIndex.value]
}
function onRitualChange(e) {
  ritualIndex.value = Number(e.detail.value)
  form.value.recognition_ritual = ritualValues[ritualIndex.value]
}

function goDetail(id) { uni.navigateTo({ url: `/pages/tasks/detail?id=${id}` }) }

// 学生列表只在进入页面时取一次；任务列表用 onShow，详情页回应后返回才会刷新待回应数。
onLoad(() => { loadStudents() })
onShow(loadTasks)
onPullDownRefresh(loadTasks)
</script>

<template>
  <view class="page">
    <view class="card form">
      <text class="title">亲子任务</text>
      <input v-model="form.title" class="input" placeholder="任务标题" />
      <textarea v-model="form.description" class="textarea" placeholder="任务说明" />
      <input v-model="form.learning_link" class="input" placeholder="和幼儿园学习的关系（必填）" />
      <picker :range="studentOptions" @change="onStudentChange">
        <view class="input picker">范围：{{ studentOptions[studentIndex] }}</view>
      </picker>
      <text v-if="studentsError" class="hint" @click="loadStudents">{{ studentsError }}，点击重试；恢复前只能创建班级任务</text>
      <picker :range="freqOptions" @change="onFreqChange">
        <view class="input picker">频率：{{ freqOptions[freqIndex] }}</view>
      </picker>
      <picker mode="date" :value="form.start_date" @change="form.start_date = $event.detail.value">
        <view class="input picker">开始日期：{{ form.start_date }}</view>
      </picker>
      <picker mode="date" :value="form.end_date" @change="form.end_date = $event.detail.value">
        <view class="input picker">结束日期：{{ form.end_date }}</view>
      </picker>
      <picker :range="ritualOptions" @change="onRitualChange">
        <view class="input picker">回应仪式：{{ ritualOptions[ritualIndex] }}</view>
      </picker>
      <view class="scope-row">
        <text class="scope-label">星星罐范围</text>
        <view class="scope-options">
          <view class="scope-opt active">按任务</view>
          <view class="scope-opt disabled" @click="explainClassStarJar">按班级</view>
        </view>
      </view>
      <text class="hint">按任务时目标填在下方；按班级需管理端先配置班级罐（当前暂未开放）</text>
      <input v-model.number="form.star_goal" class="input" type="number" placeholder="星星目标（0=不启用；选班级星星仪式时必填）" />
      <view v-if="form.target_student_id === 0" class="switch-row">
        <text>允许家长申请公开</text>
        <switch :checked="form.allow_public" @change="form.allow_public = $event.detail.value" />
      </view>
      <text v-else class="hint">个体任务不能公开</text>
      <text v-if="form.target_student_id === 0 && form.allow_public" class="hint">家长可申请把内容放上班级墙，需你逐条审核并确认内容中只有提交者孩子</text>
      <button class="submit" @click="addTask">创建任务</button>
    </view>
    <!-- 加载失败：有旧数据时说明是旧结果，无数据时作为唯一错误态 -->
    <view v-if="loadError" class="card error-card" @click="loadTasks">
      <text class="error-title">{{ loadError }}</text>
      <text class="error-hint">{{ tasks.length ? "显示的是上次拉取的结果，点击重试" : "点击重试" }}</text>
    </view>
    <view v-for="task in tasks" :key="task.id" class="card" @click="goDetail(task.id)">
      <text class="task-title">{{ task.title }}</text>
      <text class="meta">{{ freqLabel(task.frequency) }} · {{ ritualLabel(task.recognition_ritual) }} · {{ pendingLabel(task) }}</text>
      <text v-if="extraMeta(task)" class="meta">{{ extraMeta(task) }}</text>
      <view v-if="starProgress(task) !== null" class="star-bar">
        <view class="star-fill" :style="{ width: starProgress(task) + '%' }"></view>
      </view>
      <text v-if="task.description" class="desc">{{ task.description }}</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 28rpx; background: $teacher-bg; }
.card { margin-bottom: 20rpx; padding: 28rpx; border-radius: 30rpx; background: $teacher-card; box-shadow: $teacher-shadow; }
.title, .task-title { display: block; color: $teacher-text; font-size: $typography-font-size-xl; font-weight: 800; }
.input, .textarea { box-sizing: border-box; width: 100%; margin-top: 18rpx; border-radius: 22rpx; background: $color-bg-card; font-size: $typography-font-size-md; }
.input { min-height: 84rpx; padding: 0 $spacing-lg; }
.textarea { min-height: 180rpx; padding: $spacing-lg; }
.submit { min-height: 84rpx; margin-top: 18rpx; border-radius: 22rpx; background: $teacher-green; color: $color-text-on-brand; line-height: 84rpx; }
.picker { display: flex; align-items: center; }
.switch-row { display: flex; align-items: center; justify-content: space-between; margin-top: 18rpx; color: $teacher-text; font-size: $typography-font-size-sm; }
.scope-row { display: flex; align-items: center; justify-content: space-between; margin-top: 18rpx; }
.scope-label { color: $color-text-body; font-size: $typography-font-size-sm; }
.scope-options { display: flex; gap: $spacing-sm; }
.scope-opt { padding: $spacing-xs $spacing-md; border-radius: $border-radius-full; background: $color-bg-page; color: $color-text-muted; font-size: $typography-font-size-sm; }
.scope-opt.active { background: $color-brand-primary; color: $color-text-on-brand; }
.scope-opt.disabled { opacity: 0.5; }
.hint { display: block; margin-top: $spacing-xs; color: $color-text-hint; font-size: $typography-font-size-xs; line-height: 1.5; }
.meta, .desc { display: block; margin-top: 10rpx; color: $teacher-muted; font-size: $typography-font-size-sm; }
/* 8rpx 进度条高度没有对应 token（token 只覆盖间距/圆角/字号），局部特例 */
.star-bar { height: 8rpx; margin-top: $spacing-xs; border-radius: $border-radius-full; background: $color-bg-page; overflow: hidden; }
.star-fill { height: 100%; border-radius: $border-radius-full; background: $color-brand-primary; }
.error-card { background: $color-semantic-danger-bg; }
.error-title { display: block; color: $color-semantic-danger; font-size: $typography-font-size-md; font-weight: 700; }
.error-hint { display: block; margin-top: 8rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
</style>
