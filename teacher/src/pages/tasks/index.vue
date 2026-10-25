<script setup>
import { computed, ref } from "vue"
import { onLoad, onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { createTeacherTask, fetchTeacherTasks, fetchClassStarJar, setClassStarJar } from "@/api/teacher-tasks"
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

// 班级星星罐：目标与进度由后端按「当前学期 + star_scope=class 的主回应 class_star 数」
// 算好（backend/api/api-teacher/09-class-and-tasks.md）。老师端只负责展示与设置目标。
const jar = ref(null)
const jarError = ref("")
// 首次 GET 返回前 configured 恒为 false：不区分「加载中」会把它误报成「未配置」。
const jarLoading = ref(true)
// 请求序号：onShow 的慢请求不能晚于 saveGoal 后的刷新落地，否则会把新进度刷回旧值。
let jarSeq = 0
const goalInput = ref("")
// 老师在输入框里改过就不要再被 onShow 的回填覆盖；保存成功后才复位。
const goalDirty = ref(false)
const savingGoal = ref(false)
const jarConfigured = computed(() => jar.value?.configured === true)

const jarLine = computed(() => {
  if (!jarConfigured.value) return ""
  const stars = Number(jar.value?.stars || 0)
  const goal = Number(jar.value?.goal || 0)
  // 超额只换文案：进度条宽度由后端封顶 100，任何位置都不出现 >100%。
  if (jar.value?.over_goal) return `已超额 ${stars}/${goal}`
  return `已集 ${stars} / ${goal}`
})

const scopeHint = computed(() => {
  if (form.value.star_scope === "class") {
    const goal = Number(jar.value?.goal || 0)
    return `按班级时使用下方班级罐目标${goal > 0 ? `（${goal} 颗）` : ""}，本任务不再单独设目标`
  }
  return "按任务时目标填在下方；选「班级星星」回应仪式时目标必须大于 0"
})

function applyJar(data) {
  // 成功但空载荷说明响应形状不对，不能当成「未配置」——那会把卡片静默画成空白。
  if (!data || typeof data !== "object") {
    jar.value = null
    jarError.value = "班级星星罐数据异常，点击重试"
    return
  }
  jar.value = data
  jarError.value = ""
  if (!goalDirty.value) {
    const goal = Number(data.goal || 0)
    goalInput.value = goal > 0 ? String(goal) : ""
  }
}

async function loadJar() {
  const seq = ++jarSeq
  jarLoading.value = true
  try {
    const res = await fetchClassStarJar()
    if (seq !== jarSeq) return
    applyJar(res.data)
  } catch (e) {
    if (seq !== jarSeq) return
    // 读失败不能退化成「未配置」：那会禁用「按班级」，老师会以为目标丢了。
    jar.value = null
    jarError.value = e?.message || "班级星星罐加载失败"
  } finally {
    if (seq === jarSeq) jarLoading.value = false
  }
}

function onGoalInput(e) {
  goalDirty.value = true
  goalInput.value = e.detail.value
}

async function saveGoal() {
  // 读取失败时输入框里的数字可能已经过期（别处改过目标）：宁可让老师先重试读取，
  // 也不要把一个过期的值盲写回库。
  if (jarError.value) {
    uni.showToast({ title: "班级星星罐读取失败，请先重试", icon: "none" })
    return
  }
  const raw = String(goalInput.value ?? "").trim()
  if (raw === "") {
    uni.showToast({ title: "请输入目标颗数", icon: "none" })
    return
  }
  const goal = Number(raw)
  if (!Number.isFinite(goal) || !Number.isInteger(goal)) {
    uni.showToast({ title: "目标须为整数颗", icon: "none" })
    return
  }
  if (savingGoal.value) return
  savingGoal.value = true
  try {
    await setClassStarJar(goal)
    goalDirty.value = false
    // PUT 只回目标（与管理端同一响应，不含 stars / progress）：直接拿它覆盖状态
    // 会把「已集 3 / 30」显示成「已集 0 / 2」。保存后重新 GET 拿完整视图。
    await loadJar()
    uni.showToast({ title: "已保存", icon: "success" })
  } catch (e) {
    // 只有「后端以 422 拒绝 + 目标本身 <=0」才换成能照着改的文案。
    // 其它失败（网络异常没有 code、500 带别的 code）不能被误归因成「目标值不对」；
    // 它们的文案由 http.js 统一弹出，这里不再叠一条。
    if (goal <= 0 && e?.code === 42200) {
      uni.showToast({ title: "目标必须大于 0；0 表示未配置", icon: "none" })
    }
  } finally {
    savingGoal.value = false
  }
}

// 按班级只能在后端确认「本学期已配置」后选：未配置就建任务会被统一校验拒绝
// （teacher/admin/activity 三个入口同一 helper），先在这里给可读原因。
function setScope(scope) {
  if (scope === "class") {
    // 三态必须分开：加载中不是「未配置」，读失败也不是「未配置」。
    if (jarLoading.value) {
      uni.showToast({ title: "正在读取班级星星罐，请稍候", icon: "none" })
      return
    }
    if (jarError.value) {
      uni.showToast({ title: "班级星星罐读取失败，请先重试", icon: "none" })
      return
    }
    if (!jarConfigured.value) {
      uni.showToast({ title: "请先设置下方班级星星罐目标", icon: "none" })
      return
    }
    // 班级罐目标独立于任务：切过去时清空任务目标，避免残留数字被当成生效值。
    form.value.star_goal = 0
  }
  form.value.star_scope = scope
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
  // 按班级用班级罐目标，任务目标不参与；只有按任务统计时才要求任务目标。
  if (form.value.star_scope === "task" &&
      form.value.recognition_ritual === "class_star" &&
      (!Number(form.value.star_goal) || Number(form.value.star_goal) <= 0)) {
    uni.showToast({ title: "按任务统计时需要星星目标大于 0", icon: "none" })
    return
  }
  try {
    await createTeacherTask({
      ...form.value,
      star_goal: form.value.star_scope === "class" ? 0 : (Number(form.value.star_goal) || 0)
    })
    form.value = defaultForm()
    studentIndex.value = 0
    freqIndex.value = 0
    ritualIndex.value = 0
    uni.showToast({ title: "已创建", icon: "success" })
    loadTasks()
  } catch (e) {
    // 失败文案由 http.js 统一弹出（含「请先配置班级星星罐」这类 422），
    // 这里再弹一条只会连闪两下；不是静默——请求层已经报过了。
    if (!e?.code && !e?.message) console.error("创建任务失败", e)
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

// 学生列表只在进入页面时取一次；任务列表与班级罐用 onShow——
// 详情页回应后会加星，返回本页必须看到新的进度。
onLoad(() => { loadStudents() })
onShow(() => { loadTasks(); loadJar() })
onPullDownRefresh(() => { loadTasks(); loadJar() })
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
          <view class="scope-opt" :class="{ active: form.star_scope === 'task' }" @click="setScope('task')">按任务</view>
          <view class="scope-opt" :class="{ active: form.star_scope === 'class', disabled: !jarConfigured }" @click="setScope('class')">按班级</view>
        </view>
      </view>
      <text class="hint">{{ scopeHint }}</text>
      <input v-if="form.star_scope === 'task'" v-model.number="form.star_goal" class="input" type="number" placeholder="星星目标（0=不启用；选班级星星仪式时必填）" />
      <view v-if="form.target_student_id === 0" class="switch-row">
        <text>允许家长申请公开</text>
        <switch :checked="form.allow_public" @change="form.allow_public = $event.detail.value" />
      </view>
      <text v-else class="hint">个体任务不能公开</text>
      <text v-if="form.target_student_id === 0 && form.allow_public" class="hint">家长可申请把内容放上班级墙，需你逐条审核并确认内容中只有提交者孩子</text>
      <button class="submit" @click="addTask">创建任务</button>
    </view>
    <!-- 班级星星罐：只展示集体进度，不展示个人排名（spec MUST NOT） -->
    <view class="card jar-card">
      <view class="jar-head">
        <text class="jar-title">班级星星罐</text>
        <text v-if="jar" class="jar-term">{{ jar.term }}</text>
      </view>
      <text v-if="jarLoading && !jar" class="hint">正在读取本学期的班级星星罐…</text>
      <text v-else-if="jarError" class="jar-error" @click="loadJar">{{ jarError }}，点击重试</text>
      <view v-else-if="jar" class="jar-body">
        <text v-if="!jarConfigured" class="hint">本学期还没设目标；设好之后才能创建「按班级」任务。</text>
        <text v-else class="jar-line">{{ jarLine }}</text>
        <view v-if="jarConfigured" class="star-bar">
          <view class="star-fill" :style="{ width: (jar.progress || 0) + '%' }"></view>
        </view>
      </view>
      <view class="jar-goal-row">
        <input class="input goal-input" type="number" :value="goalInput" placeholder="目标颗数（>0）" @input="onGoalInput" />
        <button class="goal-save" :disabled="savingGoal || !!jarError" @click="saveGoal">{{ jarConfigured ? "更新目标" : "设置目标" }}</button>
      </view>
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
.jar-head { display: flex; align-items: baseline; justify-content: space-between; }
.jar-title { color: $teacher-text; font-size: $typography-font-size-xl; font-weight: 800; }
.jar-term { color: $color-text-hint; font-size: $typography-font-size-xs; }
.jar-body { margin-top: $spacing-xs; }
.jar-line { display: block; color: $color-brand-primary; font-size: $typography-font-size-xl; font-weight: 700; }
.jar-error { display: block; margin-top: $spacing-xs; color: $color-semantic-danger; font-size: $typography-font-size-sm; }
.jar-goal-row { display: flex; align-items: center; margin-top: $spacing-md; gap: $spacing-sm; }
/* 输入框在行内：去掉 .input 的默认上外边距，否则与按钮不同高 */
.goal-input { flex: 1; margin-top: 0; }
/* 84/22 跟随本页 .input 的既有尺寸（组件级 token 是 88/24）：
   用 token 会让按钮比同一行的输入框高 4rpx，视觉上错位。 */
.goal-save { min-height: 84rpx; margin: 0; padding: 0 $spacing-lg; border-radius: 22rpx; background: $color-brand-primary; color: $color-text-on-brand; font-size: $typography-font-size-sm; line-height: 84rpx; }
.hint { display: block; margin-top: $spacing-xs; color: $color-text-hint; font-size: $typography-font-size-xs; line-height: 1.5; }
.meta, .desc { display: block; margin-top: 10rpx; color: $teacher-muted; font-size: $typography-font-size-sm; }
/* 8rpx 进度条高度没有对应 token（token 只覆盖间距/圆角/字号），局部特例 */
.star-bar { height: 8rpx; margin-top: $spacing-xs; border-radius: $border-radius-full; background: $color-bg-page; overflow: hidden; }
.star-fill { height: 100%; border-radius: $border-radius-full; background: $color-brand-primary; }
.error-card { background: $color-semantic-danger-bg; }
.error-title { display: block; color: $color-semantic-danger; font-size: $typography-font-size-md; font-weight: 700; }
.error-hint { display: block; margin-top: 8rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
</style>
