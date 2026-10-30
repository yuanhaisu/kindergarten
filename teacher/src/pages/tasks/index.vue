<script setup>
import { computed, ref } from "vue"
import { onLoad, onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import {
  createTeacherTask,
  fetchTeacherTasks,
  fetchClassStarJar,
  setClassStarJar
} from "@/api/teacher-tasks"
import { fetchTeacherClassInfo, fetchTeacherClassStudents } from "@/api/teacher-class"
import { addDays, formatDateLabel, todayKey } from "@/utils/time"
import {
  FREQ_OPTIONS,
  RITUAL_OPTIONS,
  freqLabel,
  isTaskActive,
  ritualLabel,
  taskStarText
} from "@/utils/tasks"
import PageShell from "@/components/PageShell.vue"

const FILTERS = [
  { key: "all", label: "全部", status: "" },
  { key: "active", label: "进行中", status: "active" },
  { key: "ended", label: "已结束", status: "ended" }
]

const taskList = ref([])
const stats = ref({})
const loading = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const filter = ref("all")
const today = ref(todayKey())
const summaryPending = ref(0)
const classInfo = ref({})
let loadSeq = 0

const activeFilter = computed(
  () => FILTERS.find((item) => item.key === filter.value) || FILTERS[0]
)
const counts = computed(() => ({
  all: Number(stats.value?.all || 0),
  active: Number(stats.value?.active || 0),
  ended: Number(stats.value?.ended || 0)
}))
const hardError = computed(() => Boolean(loadError.value) && !hasLoaded.value)
// 首次渲染到 onShow 发起请求之间也算加载中：否则会先闪一下「空列表 + 底栏」。
const isInitialLoading = computed(() => !hasLoaded.value && !hardError.value)
// 列表为空且统计也为 0 才算全局空态。只看 stats 的话，统计字段缺失时
// 会把真实卡片当成空态藏起来。
const isAllEmpty = computed(
  () =>
    hasLoaded.value &&
    !loadError.value &&
    taskList.value.length === 0 &&
    counts.value.all === 0
)
const viewState = computed(() => {
  if (isInitialLoading.value) return "loading"
  if (hardError.value) return "error"
  if (isAllEmpty.value) return "empty"
  return "normal"
})
const showCounts = computed(
  () => hasLoaded.value && !isInitialLoading.value && !hardError.value && !isAllEmpty.value
)
const showSummary = computed(
  () => hasLoaded.value && !hardError.value && !isAllEmpty.value
)
const showActionBar = computed(() => viewState.value !== "empty")
const isFilteredEmpty = computed(
  () => viewState.value === "normal" && hasLoaded.value && taskList.value.length === 0
)
const className = computed(() => classInfo.value?.name || "")

// ---------- 班级星星罐 ----------
// 目标与进度由后端按「当前学期 + star_scope=class 的主回应 class_star 数」算好，
// 客户端只负责展示与设置目标（契约见 backend/api/api-teacher/09-class-and-tasks.md）。
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
const jarReady = computed(() => !jarLoading.value && !jarError.value && jarConfigured.value)
const jarStars = computed(() => Number(jar.value?.stars || 0))
const jarGoal = computed(() => Number(jar.value?.goal || 0))
const jarPercent = computed(() => {
  if (!jarConfigured.value) return 0
  return Math.min(100, Math.max(0, Number(jar.value?.progress) || 0))
})

function applyJar(data) {
  // 成功但空载荷说明响应形状不对，不能当成「未配置」——那会把卡片静默画成空白。
  if (!data || typeof data !== "object") {
    jar.value = null
    jarError.value = "班级星星罐数据异常"
    return
  }
  jar.value = data
  jarError.value = ""
  if (!goalDirty.value) {
    const goal = Number(data.goal || 0)
    goalInput.value = goal > 0 ? String(goal) : ""
  }
  // 目标还不存在时「按班级」不可选：把表单里残留的选择收回，
  // 否则弹层会显示一个提交必被 422 的范围。
  if (!jarConfigured.value && form.value.star_scope === "class") {
    form.value.star_scope = "task"
  }
}

async function loadJar() {
  const seq = ++jarSeq
  jarLoading.value = true
  try {
    const res = await fetchClassStarJar({ silent: true })
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
    await setClassStarJar(goal, { silent: true })
    goalDirty.value = false
    // PUT 只回目标，不含 stars / progress：保存后必须再 GET 一次拿完整视图，
    // 否则「已集 3 / 30」会被写成「已集 0 / 2」。
    await loadJar()
    uni.showToast({ title: "已保存", icon: "success" })
  } catch (e) {
    // 请求层是 silent 的，所有失败都必须在这里说话，否则点保存没有任何反馈。
    if (goal <= 0 && e?.code === 42200) {
      uni.showToast({ title: "目标必须大于 0；0 表示未配置", icon: "none" })
    } else {
      uni.showToast({ title: e?.message || "保存失败", icon: "none" })
    }
  } finally {
    savingGoal.value = false
  }
}

// ---------- 学生名单 ----------
const students = ref([])
const studentsTotal = ref(0)
const studentsError = ref("")
const studentsRetrying = ref(false)
const classSize = computed(() => Math.max(0, Number(studentsTotal.value || 0)))

async function loadStudents() {
  try {
    const res = await fetchTeacherClassStudents()
    const list = Array.isArray(res.data?.list) ? res.data.list : []
    students.value = list
    studentsTotal.value = Number(res.data?.total || list.length || 0)
    studentsError.value = ""
  } catch (e) {
    students.value = []
    studentsTotal.value = 0
    studentsError.value = "班级学生加载失败"
    // 名单失败时范围只剩「班级」：把已选的孩子收回，避免提交一个指向未知 id 的个体任务。
    form.value.target_student_id = 0
    scopeOpen.value = false
  }
}

async function retryStudents() {
  if (studentsRetrying.value) return
  studentsRetrying.value = true
  await loadStudents()
  studentsRetrying.value = false
  if (!studentsError.value) {
    uni.showToast({ title: "学生名单已加载", icon: "none" })
  }
}

async function loadClassInfo() {
  try {
    const res = await fetchTeacherClassInfo()
    classInfo.value = res.data || {}
  } catch (e) {
    // 班级名只用于汇总条；失败时退回「本班」，不阻断任务列表。
    console.warn("[tasks-index] load class failed:", e?.message || e)
  }
}

// ---------- 列表 ----------
function sumPending(list) {
  return list.reduce((sum, task) => sum + Number(task.pending_responses || 0), 0)
}

async function loadTasks() {
  const seq = ++loadSeq
  loading.value = true
  const status = activeFilter.value.status
  try {
    // 当前筛选走服务端 status（分页语义随之变化）；汇总条的「待回应」需要
    // 全部任务的合计，切到筛选时再拉一次全量列表，失败则沿用上一次的合计。
    const currentPromise = fetchTeacherTasks(
      { page: 1, page_size: 20, ...(status ? { status } : {}) },
      { silent: true }
    )
    const allPromise =
      filter.value === "all"
        ? null
        : fetchTeacherTasks({ page: 1, page_size: 20 }, { silent: true }).catch(() => null)
    const current = await currentPromise
    const all = allPromise ? await allPromise : current
    if (seq !== loadSeq) return
    taskList.value = Array.isArray(current.data?.list) ? current.data.list : []
    stats.value = current.data?.stats || all?.data?.stats || {}
    if (all?.data?.list) {
      summaryPending.value = sumPending(all.data.list)
    } else if (filter.value === "all") {
      summaryPending.value = sumPending(taskList.value)
    }
    today.value = todayKey()
    hasLoaded.value = true
    loadError.value = ""
  } catch (e) {
    if (seq !== loadSeq) return
    loadError.value = "任务加载失败"
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
  }
}

async function handleRetry() {
  await loadTasks()
  if (!loadError.value) {
    uni.showToast({ title: "已重新加载", icon: "none" })
  }
}

function selectFilter(key) {
  if (filter.value === key) return
  if (isInitialLoading.value) return
  filter.value = key
  loadTasks()
}

function resetToAll() {
  if (filter.value === "all") return
  filter.value = "all"
  loadTasks()
}

// ---------- 卡片文案 ----------
function formatRange(task) {
  const start = formatDateLabel(task.start_date)
  const end = formatDateLabel(task.end_date)
  if (!start && !end) return "—"
  return `${start || "—"}–${end || "—"}`
}

function statusText(task) {
  return isTaskActive(task, today.value) ? "进行中" : "已结束"
}

function participantCount(task) {
  return Number(task.participant_count || 0)
}

function participantPercent(task) {
  if (Number(task.target_student_id) > 0) return null
  if (classSize.value <= 0) return null
  return Math.min(100, Math.round((participantCount(task) / classSize.value) * 100))
}

function pendingLabel(task) {
  const total = Number(task.pending_responses || 0)
  const period = task.pending_period
  if (typeof period !== "number" || period === total) return `待回应 ${total}`
  return `待回应 ${total}（本周期 ${period}）`
}

function goDetail(id) {
  uni.navigateTo({ url: `/pages/tasks/detail?id=${id}` })
}

// ---------- 新建任务 ----------
// star_goal 用字符串保存输入框原值，提交时才转数字；空字符串不等于 0。
function defaultForm() {
  const start = todayKey()
  return {
    title: "",
    description: "",
    learning_link: "",
    frequency: "once",
    start_date: start,
    end_date: addDays(start, 7),
    target_student_id: 0,
    allow_public: false,
    recognition_ritual: "sticker",
    star_scope: "task",
    star_goal: ""
  }
}

const composeOpen = ref(false)
const creating = ref(false)
const scopeOpen = ref(false)
const starGoalError = ref("")
const form = ref(defaultForm())

const scopeOptions = computed(() => {
  const options = [
    {
      id: 0,
      name: "班级",
      sub: studentsError.value ? "学生名单未加载" : `全班 ${studentsTotal.value} 人`
    }
  ]
  if (!studentsError.value) {
    students.value.forEach((student) => {
      options.push({ id: student.student_id, name: student.name, sub: "" })
    })
  }
  return options
})
const selectedScope = computed(
  () => scopeOptions.value.find((item) => item.id === Number(form.value.target_student_id || 0)) || scopeOptions.value[0]
)
const scopeValue = computed(() =>
  Number(form.value.target_student_id) > 0 ? selectedScope.value.name : "班级"
)
const scopeSub = computed(() => {
  if (Number(form.value.target_student_id) > 0) return "个体任务 · 仅该家长可见"
  return studentsError.value ? "学生名单未加载" : `全班 ${studentsTotal.value} 人`
})
const goalRequired = computed(
  () => form.value.recognition_ritual === "class_star" && form.value.star_scope === "task"
)
const canSubmit = computed(
  () =>
    !creating.value &&
    form.value.title.trim() !== "" &&
    form.value.learning_link.trim() !== ""
)

function openCompose() {
  if (creating.value) return
  form.value = defaultForm()
  starGoalError.value = ""
  scopeOpen.value = false
  composeOpen.value = true
}

function closeCompose() {
  if (creating.value) return
  composeOpen.value = false
  scopeOpen.value = false
}

function toggleScopeMenu() {
  scopeOpen.value = !scopeOpen.value
}

function selectScope(option) {
  form.value.target_student_id = Number(option.id || 0)
  // 个体任务强制私密（后端也会强制），UI 先收口，避免「保存后还是关的」。
  if (form.value.target_student_id > 0) form.value.allow_public = false
  scopeOpen.value = false
}

function selectFrequency(value) {
  form.value.frequency = value
}

function selectRitual(value) {
  form.value.recognition_ritual = value
  if (value !== "class_star") starGoalError.value = ""
}

function selectStarScope(value) {
  if (value === "class") {
    if (jarLoading.value) {
      uni.showToast({ title: "正在读取班级星星罐，请稍候", icon: "none" })
      return
    }
    if (jarError.value) {
      uni.showToast({ title: "班级星星罐读取失败，请先重试", icon: "none" })
      return
    }
    if (!jarConfigured.value) {
      uni.showToast({ title: "请先设置班级星星罐目标", icon: "none" })
      return
    }
    // 班级罐目标独立于任务：切过去时清空任务目标，避免残留数字被当成生效值。
    form.value.star_goal = ""
  }
  form.value.star_scope = value
  starGoalError.value = ""
}

function onStarGoalInput(e) {
  form.value.star_goal = e.detail.value
  starGoalError.value = ""
}

function onPublicChange(e) {
  form.value.allow_public = !!e.detail.value
}

function onStartDate(e) {
  const value = e.detail.value
  form.value.start_date = value
  // 起止顺序必须合法；只往后推结束日，不静默保留一个非法区间。
  if (form.value.end_date < value) form.value.end_date = value
}

function onEndDate(e) {
  const value = e.detail.value
  if (value < form.value.start_date) {
    uni.showToast({ title: "结束日期不能早于开始日期", icon: "none" })
    return
  }
  form.value.end_date = value
}

async function createTask() {
  if (!canSubmit.value) return
  if (form.value.end_date < form.value.start_date) {
    uni.showToast({ title: "结束日期不能早于开始日期", icon: "none" })
    return
  }
  const goal = Number(String(form.value.star_goal || "").trim() || 0)
  if (goalRequired.value && !(goal > 0)) {
    starGoalError.value = "选择班级星星仪式时，请填写大于 0 的目标"
    uni.showToast({ title: "请填写大于 0 的星星目标", icon: "none" })
    return
  }
  if (form.value.star_scope === "class" && !jarConfigured.value) {
    uni.showToast({ title: "请先设置班级星星罐目标", icon: "none" })
    return
  }
  creating.value = true
  try {
    await createTeacherTask(
      {
        title: form.value.title.trim(),
        description: form.value.description.trim(),
        learning_link: form.value.learning_link.trim(),
        target_student_id: Number(form.value.target_student_id || 0),
        frequency: form.value.frequency,
        start_date: form.value.start_date,
        end_date: form.value.end_date,
        allow_public:
          Number(form.value.target_student_id || 0) === 0 && !!form.value.allow_public,
        recognition_ritual: form.value.recognition_ritual,
        star_scope: form.value.star_scope,
        star_goal: form.value.star_scope === "class" ? 0 : goal
      },
      { silent: true }
    )
    composeOpen.value = false
    form.value = defaultForm()
    uni.showToast({ title: "已创建", icon: "success" })
    // 回到「全部」再刷新：新建的任务一定是进行中，若停在「已结束」筛选里
    // 会看不到刚创建的任务，看起来像创建失败。
    filter.value = "all"
    // 重新拉列表而不是本地插卡：stats 计数、待回应合计与参与人数都要跟着变，
    // 本地拼出来的卡片会和服务端口径漂移。
    await loadTasks()
  } catch (e) {
    uni.showToast({ title: e?.message || "创建任务失败", icon: "none" })
  } finally {
    creating.value = false
  }
}

function noop() {}

onLoad(() => {
  loadStudents()
  loadClassInfo()
})
onShow(() => {
  loadTasks()
  loadJar()
})
onPullDownRefresh(() => {
  loadTasks()
  loadJar()
})
</script>

<template>
  <PageShell bottom-space="128rpx">
    <view class="page" :class="'state-' + viewState">
      <!-- 三段状态筛选：计数取服务端 stats，筛选本身走 status（分页语义不变） -->
      <view class="filters">
        <view class="filter-track">
          <view
            v-for="item in FILTERS"
            :key="item.key"
            class="filter"
            :class="{ 'is-active': filter === item.key, 'is-disabled': isInitialLoading }"
            @click="selectFilter(item.key)"
          >
            <text class="filter-label">{{ item.label }}</text>
            <text v-if="showCounts" class="filter-count">{{ counts[item.key] }}</text>
          </view>
        </view>
      </view>

      <!-- 班级汇总：共 N 个任务 · M 条待回应（M 为全部任务 pending_responses 之和） -->
      <view v-if="showSummary" class="list-head">
        <text class="list-class">{{ className || "本班" }}</text>
        <text class="list-sep">·</text>
        <text>共 {{ counts.all }} 个任务</text>
        <view class="list-pending">
          <view class="pending-dot" />
          <text><text class="pending-num">{{ summaryPending }}</text> 条待回应</text>
        </view>
      </view>

      <!-- 加载：3 张任务骨架（筛选计数隐藏） -->
      <view v-if="viewState === 'loading'" class="skeleton-list">
        <view v-for="i in 3" :key="i" class="skeleton-card">
          <view class="sk sk-title" />
          <view class="sk sk-meta" />
          <view class="sk sk-desc" />
          <view class="sk sk-desc sk-short" />
          <view class="sk sk-bar" />
        </view>
      </view>

      <!-- 错误：首次加载失败才占满内容区；有旧数据时用列表上方的错误条 -->
      <view v-else-if="viewState === 'error'" class="state-panel">
        <view class="state-icon is-error"><text class="icon">&#xECA1;</text></view>
        <text class="state-title">任务加载失败</text>
        <text class="state-desc">网络异常，未获取到任务列表</text>
        <view class="state-actions">
          <view class="btn-ghost" @click="handleRetry">重试</view>
        </view>
      </view>

      <!-- 空：保留星星罐，隐藏任务列表与筛选计数 -->
      <view v-else-if="viewState === 'empty'" class="state-panel">
        <view class="state-icon is-empty"><text class="icon">&#xF186;</text></view>
        <text class="state-title">还没有亲子任务</text>
        <text class="state-desc">创建后家长可在小程序里查看并回应</text>
        <view class="state-actions">
          <view class="btn-primary" @click="openCompose">新建任务</view>
        </view>
      </view>

      <!-- 正常列表 -->
      <view v-else class="task-list">
        <view v-if="loadError" class="error-strip" @click="handleRetry">
          {{ loadError }} · 显示的是上次拉取的结果，点击重试
        </view>
        <view
          v-for="task in taskList"
          :key="task.id"
          class="task-card"
          @click="goDetail(task.id)"
        >
          <view class="task-head">
            <text class="task-title">{{ task.title }}</text>
            <text
              class="status-pill"
              :class="isTaskActive(task, today) ? 'is-live' : 'is-ended'"
            >
              <view v-if="isTaskActive(task, today)" class="status-dot" />
              <text>{{ statusText(task) }}</text>
            </text>
          </view>
          <view class="task-meta">
            <text>{{ freqLabel(task.frequency) }}</text>
            <text class="meta-dot">·</text>
            <text>{{ ritualLabel(task.recognition_ritual) }}</text>
            <text class="meta-dot">·</text>
            <text>{{ formatRange(task) }}</text>
            <text v-if="Number(task.target_student_id) > 0" class="scope-tag">个体任务</text>
          </view>
          <text v-if="task.description" class="task-desc">{{ task.description }}</text>
          <view class="task-stats" :class="{ 'is-single': Number(task.target_student_id) > 0 }">
            <view class="progress-head">
              <text v-if="Number(task.target_student_id) > 0" class="progress-label">已提交 <text class="progress-strong">{{ participantCount(task) }}</text> 条</text>
              <text v-else class="progress-label">已参与 <text class="progress-strong">{{ participantCount(task) }}</text><text v-if="classSize > 0">/{{ classSize }}</text><text v-else> 人</text></text>
              <text v-if="participantPercent(task) !== null" class="progress-pct">
                {{ participantPercent(task) }}%
              </text>
            </view>
            <view v-if="participantPercent(task) !== null" class="progress-track">
              <view
                class="progress-fill"
                :class="{ 'is-ended': !isTaskActive(task, today) }"
                :style="{ width: participantPercent(task) + '%' }"
              />
            </view>
          </view>
          <text v-if="taskStarText(task)" class="task-star">
            <text class="star-icon icon">&#xF186;</text>
            <text>{{ taskStarText(task) }}</text>
          </text>
          <view class="task-foot">
            <text
              class="pending-badge"
              :class="{ 'is-zero': !Number(task.pending_responses) }"
            >{{ pendingLabel(task) }}</text>
            <view class="task-link">
              <text>查看回应</text>
              <text class="chev icon">&#xEA6E;</text>
            </view>
          </view>
        </view>

        <!-- 筛选后为空：不是「没有任务」，不能复用全局空态文案 -->
        <view v-if="isFilteredEmpty" class="filter-empty">
          <text class="filter-empty-text">当前筛选下没有任务</text>
          <view class="text-link" @click="resetToAll">切到「全部」</view>
        </view>
      </view>

      <!-- 班级星星罐：列表之后的次级设置，只展示集体进度，不展示个人排名。
           任务列表失败时罐仍独立加载：两者是不同接口，不能互相掩盖。 -->
      <view class="jar-card">
        <view class="jar-head">
          <view class="jar-icon"><text class="icon">&#xF186;</text></view>
          <view class="jar-heading">
            <text class="jar-title">班级星星罐</text>
            <text v-if="jar && jarConfigured" class="jar-term">{{ jar.term }}</text>
          </view>
        </view>

        <view v-if="jarLoading && !jar" class="jar-skeleton">
          <view class="sk sk-jar-count" />
          <view class="sk sk-bar" />
          <view class="sk sk-jar-input" />
        </view>

        <text v-else-if="jarError" class="jar-error" @click="loadJar">
          {{ jarError }}，点击重试
        </text>

        <view v-else class="jar-live">
          <template v-if="jarConfigured">
            <view class="jar-stats">
              <view class="jar-count-row">
                <text class="jar-count">
                  <text class="jar-count-prefix">已集</text>
                  <text class="jar-count-num">{{ jarStars }}</text>
                  <text class="jar-count-total">/ {{ jarGoal }}</text>
                </text>
                <text class="jar-pct">{{ jarPercent }}%</text>
                <text v-if="jar.over_goal" class="jar-over">已超额</text>
              </view>
              <view class="progress-track jar-track">
                <view class="progress-fill" :style="{ width: jarPercent + '%' }" />
              </view>
              <text class="jar-note">只记录班级集体进度，不展示个人排名</text>
            </view>
          </template>
          <text v-else class="jar-empty-note">本学期还没设目标；设好之后才能创建『按班级』任务</text>

          <view class="jar-form">
            <text class="jar-form-label">学期目标</text>
            <input
              class="jar-input"
              type="number"
              :value="goalInput"
              placeholder="目标颗数"
              @input="onGoalInput"
            />
            <view class="jar-save" :class="{ 'is-busy': savingGoal }" @click="saveGoal">
              {{ savingGoal ? "保存中…" : jarConfigured ? "更新目标" : "设置目标" }}
            </view>
          </view>
        </view>
      </view>
    </view>

    <template #bottom>
      <view v-if="showActionBar" class="action-bar">
        <view class="action-create" @click="openCompose">
          <text class="cta-plus icon">&#xEA13;</text>
          <text>新建任务</text>
        </view>
      </view>
    </template>

    <!-- 新建任务弹层：长表单收进底部弹层，列表成为首屏主体 -->
    <view v-if="composeOpen" class="sheet-layer">
      <view class="sheet-mask" @click="closeCompose" @touchmove.stop.prevent="noop" />
      <view class="sheet">
        <view class="sheet-grab" />
        <view class="sheet-head">
          <view class="sheet-head-main">
            <text class="sheet-title">新建任务</text>
            <text class="sheet-sub">{{ className || "本班" }} · 家长会在小程序里看到</text>
          </view>
          <view class="sheet-close" @click="closeCompose"><text class="icon">&#xEB99;</text></view>
        </view>

        <scroll-view scroll-y class="sheet-body">
          <view class="field">
            <text class="field-label">任务标题 <text class="required">必填</text></text>
            <input
              v-model="form.title"
              class="text-input"
              type="text"
              maxlength="40"
              placeholder="如：每日阅读 10 分钟"
            />
          </view>

          <view class="field">
            <text class="field-label">任务说明</text>
            <textarea
              v-model="form.description"
              class="text-area"
              maxlength="200"
              placeholder="如：每天陪孩子读 10 分钟绘本"
            />
          </view>

          <view class="field">
            <text class="field-label">和幼儿园学习的关系 <text class="required">必填</text></text>
            <input
              v-model="form.learning_link"
              class="text-input"
              type="text"
              maxlength="80"
              placeholder="如：延续本周的春天主题"
            />
          </view>

          <view class="field">
            <text class="field-label">范围</text>
            <view class="scope-trigger" @click="toggleScopeMenu">
              <view class="scope-trigger-main">
                <text class="scope-value">{{ scopeValue }}</text>
                <text class="scope-sub">{{ scopeSub }}</text>
              </view>
              <text class="scope-chevron icon">&#xEA4E;</text>
            </view>
            <view v-if="scopeOpen" class="scope-menu">
              <scroll-view scroll-y class="scope-scroll">
                <view
                  v-for="option in scopeOptions"
                  :key="option.id"
                  class="scope-option"
                  :class="{ 'is-selected': Number(form.target_student_id) === option.id }"
                  @click="selectScope(option)"
                >
                  <view class="scope-option-main">
                    <text class="scope-option-name">{{ option.name }}</text>
                    <text v-if="option.sub" class="scope-option-sub">{{ option.sub }}</text>
                  </view>
                  <text
                    v-if="Number(form.target_student_id) === option.id"
                    class="scope-option-check icon"
                  >&#xEB7B;</text>
                </view>
              </scroll-view>
            </view>
            <text v-if="studentsError" class="hint is-warning" @click="retryStudents">
              班级学生加载失败，<text class="inline-link">点击重试</text>；恢复前只能创建班级任务
            </text>
            <text v-if="Number(form.target_student_id) > 0" class="hint is-info">
              个体任务只有该孩子家长可见，不能公开
            </text>
          </view>

          <view class="field">
            <text class="field-label">频率</text>
            <view class="segment">
              <view
                v-for="option in FREQ_OPTIONS"
                :key="option.value"
                class="seg"
                :class="{ 'is-on': form.frequency === option.value }"
                @click="selectFrequency(option.value)"
              >{{ option.label }}</view>
            </view>
          </view>

          <view class="field">
            <text class="field-label">起止日期</text>
            <view class="date-row">
              <picker mode="date" :value="form.start_date" @change="onStartDate">
                <view class="date-cell">
                  <text class="date-cap">开始</text>
                  <text class="date-value">{{ formatDateLabel(form.start_date) || "选择日期" }}</text>
                </view>
              </picker>
              <picker mode="date" :value="form.end_date" @change="onEndDate">
                <view class="date-cell">
                  <text class="date-cap">结束</text>
                  <text class="date-value">{{ formatDateLabel(form.end_date) || "选择日期" }}</text>
                </view>
              </picker>
            </view>
          </view>

          <view class="field">
            <text class="field-label">回应仪式</text>
            <view class="segment">
              <view
                v-for="option in RITUAL_OPTIONS"
                :key="option.value"
                class="seg"
                :class="{ 'is-on': form.recognition_ritual === option.value }"
                @click="selectRitual(option.value)"
              >{{ option.label }}</view>
            </view>
          </view>

          <view class="field">
            <text class="field-label">星星罐范围</text>
            <view class="segment segment-2">
              <view
                class="seg"
                :class="{ 'is-on': form.star_scope === 'task' }"
                @click="selectStarScope('task')"
              >按任务</view>
              <view
                class="seg"
                :class="{ 'is-on': form.star_scope === 'class', 'is-disabled': !jarReady }"
                @click="selectStarScope('class')"
              >按班级</view>
            </view>
            <text v-if="form.star_scope === 'class'" class="hint is-info">
              使用班级罐目标 {{ jarGoal }} 颗，本任务不再单独设目标
            </text>
            <text v-else-if="!jarReady" class="hint is-warning">
              班级罐还没设目标，暂时不能选择「按班级」
            </text>
          </view>

          <view v-if="form.star_scope === 'task'" class="field">
            <text class="field-label">星星目标</text>
            <input
              class="text-input"
              type="number"
              :value="form.star_goal"
              maxlength="4"
              placeholder="0 = 不启用；选班级星星仪式时必填"
              @input="onStarGoalInput"
            />
            <text v-if="starGoalError" class="hint is-warning">{{ starGoalError }}</text>
          </view>

          <view v-if="Number(form.target_student_id) === 0" class="switch-row">
            <text class="switch-label">允许家长申请公开</text>
            <switch :checked="form.allow_public" @change="onPublicChange" />
          </view>
          <text
            v-if="Number(form.target_student_id) === 0 && form.allow_public"
            class="hint is-warning"
          >家长可申请把内容放上班级墙，需你逐条审核并确认内容中只有提交者孩子</text>
        </scroll-view>

        <view class="sheet-footer">
          <view class="footer-btn">
            <view class="btn-cancel" @click="closeCompose">取消</view>
          </view>
          <view class="footer-btn">
            <button
              class="btn-create"
              :class="{ 'is-disabled': !canSubmit }"
              :disabled="!canSubmit"
              @click="createTask"
            >{{ creating ? "创建中…" : "创建任务" }}</button>
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
  padding-bottom: $spacing-xl;
}

/* ---------- 筛选 ---------- */
.filters {
  padding: 0 2rpx $spacing-md;
}

.filter-track {
  display: flex;
  gap: $spacing-xs;
  padding: $spacing-xs;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
}

.filter {
  flex: 1;
  min-width: 0;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  border-radius: $border-radius-full;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.filter.is-active {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.filter.is-disabled {
  opacity: 0.6;
}

.filter-count {
  min-width: 36rpx;
  height: 36rpx;
  padding: 0 10rpx;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: inherit;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.filter.is-active .filter-count {
  background: $color-brand-primary;
  color: $color-text-on-brand;
}

/* ---------- 汇总条 ---------- */
.list-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-xs $spacing-sm;
  padding: $spacing-sm 2rpx $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}

.list-class {
  color: $color-text-body;
  font-weight: $typography-font-weight-semibold;
}

.list-sep {
  color: $color-text-hint;
}

.list-pending {
  display: inline-flex;
  align-items: center;
  gap: 10rpx;
  margin-left: auto;
  color: $color-semantic-warning-strong;
  font-weight: $typography-font-weight-medium;
}

.pending-dot {
  width: 12rpx;
  height: 12rpx;
  flex: none;
  border-radius: $border-radius-full;
  background: $color-semantic-warning;
}

.pending-num {
  font-weight: $typography-font-weight-semibold;
}

/* ---------- 任务卡 ---------- */
.task-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.task-card {
  padding: $spacing-lg $spacing-lg $spacing-xs;
  border: 1rpx solid $color-border-subtle;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.task-card:active {
  transform: scale(0.985);
}

.task-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20rpx;
}

.task-title {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.status-pill {
  flex: none;
  min-height: 44rpx;
  padding: 2rpx 18rpx;
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.status-pill.is-live {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.status-pill.is-ended {
  background: $color-bg-page;
  color: $color-text-muted;
}

.status-dot {
  width: 10rpx;
  height: 10rpx;
  flex: none;
  border-radius: $border-radius-full;
  background: $color-brand-primary-dark;
}

.task-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4rpx;
  margin-top: 10rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.55;
}

.meta-dot {
  margin: 0 8rpx;
  color: $color-border-hairline;
  font-weight: $typography-font-weight-bold;
}

.scope-tag {
  margin-left: 12rpx;
  padding: 2rpx 14rpx;
  border: 1rpx solid $color-brand-primary-light;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.5;
}

.task-desc {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  margin-top: 14rpx;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.task-stats {
  margin-top: 18rpx;
}

.progress-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: 12rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}

.progress-label {
  min-width: 0;
}

.progress-strong {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

.progress-pct {
  flex: none;
  color: $color-text-body;
  font-weight: $typography-font-weight-semibold;
}

.progress-track {
  height: 12rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  overflow: hidden;
}

.progress-fill {
  display: block;
  height: 100%;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

/* 已结束任务的进度不再用品牌橙，避免和「进行中」抢注意力 */
.progress-fill.is-ended {
  background: $color-text-muted;
}

.task-stats.is-single .progress-head {
  margin-bottom: 0;
}

.task-star {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  margin-top: 14rpx;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.55;
}

.star-icon {
  font-size: $typography-font-size-base;
}

.task-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: $spacing-sm 20rpx;
  margin-top: 18rpx;
  padding: 18rpx 0 20rpx;
  border-top: 1rpx dashed $color-border-subtle;
}

.pending-badge {
  min-height: 48rpx;
  padding: 2rpx 18rpx;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.45;
}

.pending-badge.is-zero {
  background: $color-bg-page;
  color: $color-text-muted;
}

.task-link {
  display: inline-flex;
  align-items: center;
  gap: 2rpx;
  flex: none;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.5;
}

.task-link .chev {
  font-size: $typography-font-size-sm;
}

/* ---------- 筛选空 ---------- */
.filter-empty {
  padding: $spacing-xl $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-xs;
}

.filter-empty-text {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.text-link {
  min-height: 88rpx;
  display: inline-flex;
  align-items: center;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

/* ---------- 状态面板 ---------- */
.state-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 84rpx $spacing-lg 72rpx;
  border: 1rpx solid $color-border-subtle;
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

.state-icon.is-empty {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.state-icon.is-error {
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
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
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
.btn-secondary,
.btn-ghost {
  min-height: 88rpx;
  padding: 0 $spacing-xl;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
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
}

.btn-ghost {
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-card;
  color: $color-text-body;
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
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.skeleton-card {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  padding: $spacing-xl $spacing-lg;
  border: 1rpx solid $color-border-subtle;
  border-radius: $component-card-radius;
  background: $color-bg-card;
}

.sk {
  display: block;
  height: 20rpx;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  animation: sk-pulse 1.3s $animation-easing-default infinite;
}

.sk-title {
  width: 52%;
  height: 30rpx;
}

.sk-meta {
  width: 78%;
  height: 18rpx;
}

.sk-desc {
  width: 100%;
}

.sk-short {
  width: 64%;
}

.sk-bar {
  width: 100%;
  height: 12rpx;
}

/* ---------- 班级星星罐 ---------- */
.jar-card {
  margin-top: $spacing-xl;
  padding: $spacing-xl;
  border: 1rpx solid $color-border-subtle;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.jar-head {
  display: flex;
  align-items: center;
  gap: 20rpx;
}

.jar-icon {
  width: 84rpx;
  height: 84rpx;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xl;
}

.jar-heading {
  min-width: 0;
}

.jar-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.jar-term {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.jar-live {
  margin-top: $spacing-lg;
}

.jar-count-row {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: $spacing-xs $spacing-md;
}

.jar-count {
  display: flex;
  align-items: baseline;
  gap: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
}

.jar-count-prefix {
  font-size: $typography-font-size-md;
}

.jar-count-num {
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-bold;
  line-height: 1.1;
}

.jar-count-total {
  color: $color-text-muted;
  font-size: $typography-font-size-md;
}

.jar-pct {
  margin-left: auto;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
}

.jar-over {
  padding: 2rpx 16rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
}

.jar-track {
  height: 16rpx;
  margin-top: 18rpx;
}

.jar-note {
  display: block;
  margin-top: 18rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.55;
}

.jar-empty-note {
  display: block;
  color: $color-semantic-warning-strong;
  font-size: $typography-font-size-xs;
  line-height: 1.55;
}

.jar-form {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-lg;
}

.jar-form-label {
  flex: none;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

.jar-input {
  flex: 1;
  min-width: 0;
  height: 88rpx;
  padding: 0 $spacing-md;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  text-align: center;
}

.jar-save {
  flex: none;
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.jar-save.is-busy {
  opacity: 0.6;
}

.jar-error {
  display: block;
  margin-top: $spacing-md;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.jar-skeleton {
  margin-top: $spacing-lg;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.sk-jar-count {
  width: 46%;
  height: 48rpx;
}

.sk-jar-input {
  width: 100%;
  height: 88rpx;
}

/* ---------- 固定底栏 ---------- */
.action-bar {
  width: 100%;
  box-sizing: border-box;
  padding: $spacing-md $component-page-padding-x calc(#{$spacing-md} + env(safe-area-inset-bottom));
  border-top: 1rpx solid $color-border-subtle;
  background: $color-bg-card;
}

.action-create {
  width: 100%;
  min-height: 96rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.action-create:active {
  transform: scale(0.98);
}

.cta-plus {
  font-size: $typography-font-size-3xl;
}

/* ---------- 新建弹层 ---------- */
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
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
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
  max-height: 58vh;
  padding: 0 $spacing-xl $spacing-md;
  box-sizing: border-box;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
  margin-bottom: $spacing-xl;
}

.field-label {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.required {
  display: inline-block;
  margin-left: 8rpx;
  padding: 2rpx 12rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.text-input {
  box-sizing: border-box;
  width: 100%;
  height: $component-input-height;
  padding: 0 $spacing-md;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
}

.text-area {
  box-sizing: border-box;
  width: 100%;
  min-height: 152rpx;
  padding: $spacing-md;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.5;
}

.scope-trigger {
  min-height: $component-input-height;
  padding: 10rpx $spacing-md;
  display: flex;
  align-items: center;
  gap: 20rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
}

.scope-trigger-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2rpx;
}

.scope-value {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.scope-sub {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.scope-chevron {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.scope-menu {
  margin-top: 4rpx;
  padding: 6rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
}

.scope-scroll {
  height: 420rpx;
}

.scope-option {
  min-height: 88rpx;
  padding: $spacing-xs $spacing-md;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  border-radius: $border-radius-lg;
}

.scope-option.is-selected {
  background: $color-brand-primary-bg;
}

.scope-option-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2rpx;
}

.scope-option-name {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.scope-option.is-selected .scope-option-name {
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

.scope-option-sub {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.scope-option-check {
  flex: none;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
}

.hint {
  display: block;
  font-size: $typography-font-size-xs;
  line-height: 1.55;
  color: $color-text-muted;
}

.hint.is-warning {
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius-lg;
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
}

.hint.is-info {
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius-lg;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.inline-link {
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
  text-decoration: underline;
}

.segment {
  display: flex;
  gap: 6rpx;
  padding: 6rpx;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.seg {
  flex: 1;
  min-width: 0;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.seg.is-on {
  background: $color-bg-card;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.seg.is-disabled {
  color: $color-text-hint;
}

.date-row {
  display: flex;
  gap: $spacing-md;
}

/* picker 是原生组件，不设 flex 时两个日期格会按内容宽度排布而不是等分 */
.date-row picker {
  flex: 1;
  min-width: 0;
}

.date-cell {
  min-height: 112rpx;
  padding: 18rpx $spacing-md;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
}

.date-cap {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.date-value {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  min-height: 88rpx;
  margin-bottom: $spacing-sm;
}

.switch-label {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
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
  width: 100%;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.btn-create::after {
  border: none;
}

.btn-create.is-disabled,
.btn-create[disabled] {
  background: $color-border-subtle;
  color: $color-text-hint;
  box-shadow: none;
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
