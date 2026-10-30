<script setup>
/**
 * 接送管理（老师端）
 *
 * 数据分三路：overview（四统计）、students?status=（当前筛选）、
 * students?status=not_picked_up（滞留条）。滞留条不用 overview.overdue_not_picked：
 * 那是「放学时点仍在园」的累计口径，之后被接走也不清零；本页要的是此刻仍在园。
 */
import { computed, ref } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import {
  fetchTeacherPickupOverview,
  fetchTeacherPickupStudents,
  fetchStudentAuthPersons,
  manualTeacherPickup,
  notifyOverdueParent,
  scanTeacherPickup
} from "@/api/teacher-pickup"
import { fetchTeacherClassInfo } from "@/api/teacher-class"
import { formatClock, formatDateLabel, formatWeekdayShort, todayKey } from "@/utils/time"
import { personLabel, recordPersonLabel } from "@/utils/relation"
import PageShell from "@/components/PageShell.vue"
import StateBlock from "@/components/StateBlock.vue"
import AppButton from "@/components/AppButton.vue"

const FILTERS = [
  { key: "not_picked_up", label: "未接走", listLabel: "未接走" },
  { key: "all", label: "全部", listLabel: "全部" },
  { key: "arrived", label: "已入园", listLabel: "已入园" },
  { key: "picked_up", label: "已接走", listLabel: "已接走" }
]

const MANUAL_STATUS = { not_arrived: "未到园", not_picked_up: "已入园", picked_up: "已接走" }

const today = ref(todayKey())
const overview = ref({})
const students = ref([])
const pendingStudents = ref([])
const filterStatus = ref("not_picked_up")
const loading = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const className = ref("")
const notified = ref({})
const notifying = ref({})
const busy = ref({})

const showPicker = ref(false)
const pickerIntent = ref("authorized")
const pickerStudents = ref([])
const pickerLoading = ref(false)
const pickerError = ref("")

const showManual = ref(false)
const manualStudent = ref(null)
const manualMode = ref("authorized")
const manualRemark = ref("")
const manualOverrideName = ref("")
const authPersons = ref([])
const selectedAuthId = ref(null)
const authPersonsLoading = ref(false)
const authPersonsError = ref("")
const submitting = ref(false)

const showScan = ref(false)
const scanPhase = ref("loading")
const scanError = ref("")
const scanResult = ref(null)

let loadSeq = 0
let pickerSeq = 0
let authSeq = 0
let scanSeq = 0

const todayLabel = computed(() => `${formatDateLabel(today.value)} ${formatWeekdayShort(today.value)}`)

// 四统计口径以 overview 为准；旧页面读的 overview.not_picked 契约里不存在，
// 会静默退化成 total - picked_up（把未到园也算成未接走）。
const stats = computed(() => ({
  notArrived: Number(overview.value.not_arrived || 0),
  pending: Number(overview.value.not_picked_up || 0),
  picked: Number(overview.value.picked_up || 0),
  abnormal: Number(overview.value.abnormal || 0)
}))

const statsLoading = computed(() => loading.value && !hasLoaded.value)

const overdueStudents = computed(() =>
  pendingStudents.value.filter(item => item.is_overdue && item.pickup_status === "not_picked_up")
)

const overdueText = computed(() => {
  const list = overdueStudents.value
  if (!list.length) return ""
  const deadline = list.find(item => item.pickup_deadline)?.pickup_deadline || ""
  return deadline
    ? `${list.length} 名孩子已过放学时间 ${deadline} 仍在园`
    : `${list.length} 名孩子已过放学时间仍在园`
})

const activeFilter = computed(
  () => FILTERS.find(item => item.key === filterStatus.value) || FILTERS[0]
)

const listTitle = computed(() => `${activeFilter.value.listLabel}学生（${students.value.length}人）`)

const isPageEmpty = computed(
  () =>
    !loading.value &&
    !loadError.value &&
    hasLoaded.value &&
    filterStatus.value === "not_picked_up" &&
    students.value.length === 0
)

// 默认筛选（已入园未接走）为空有两种原因：今天还没人入园，或已入园的都接走了。
// 只按计数区分，避免把「无人入园」误述成「全部孩子已完成接送」。
const emptyKind = computed(() =>
  Number(overview.value.picked_up || 0) === 0 && Number(overview.value.not_picked_up || 0) === 0
    ? "no_arrival"
    : "all_picked"
)

const showOverview = computed(() => !loadError.value && !isPageEmpty.value)
const showListHead = computed(() => showOverview.value && !loading.value)

const manualSub = computed(() => {
  const student = manualStudent.value
  if (!student) return ""
  return `${className.value || "本班"} · ${MANUAL_STATUS[student.pickup_status] || "—"}`
})

const canSubmitManual = computed(() => {
  if (submitting.value) return false
  if (manualMode.value === "authorized") return selectedAuthId.value !== null
  return manualOverrideName.value.trim() !== ""
})

const scanStudentName = computed(() => scanResult.value?.student?.name || "")
const scanTypeLabel = computed(() => scanResult.value?.type_label || "接走")
const scanTime = computed(() => formatClock(scanResult.value?.recorded_at))
const scanPerson = computed(() => {
  const person = scanResult.value?.person
  if (!person) return "—"
  // 扫码响应的 person.name 可能已带「（姓名）」括号，不能再追加一次关系。
  return recordPersonLabel(person.name, person.relation) || "—"
})

function initialOf(name) {
  return String(name || "").slice(0, 1) || "生"
}

function isPendingOverdue(student) {
  return student.pickup_status === "not_picked_up" && Boolean(student.is_overdue)
}

function statusPills(student) {
  const pills = []
  if (student.pickup_status === "not_arrived") pills.push({ key: "wait", label: "未到园", tone: "wait" })
  if (student.pickup_status === "not_picked_up") pills.push({ key: "pending", label: "待接走", tone: "pending" })
  if (student.pickup_status === "picked_up") pills.push({ key: "done", label: "已接走", tone: "done" })
  if (isPendingOverdue(student)) pills.push({ key: "overdue", label: "滞留", tone: "overdue" })
  if (student.is_authorized_override) pills.push({ key: "override", label: "例外", tone: "override" })
  return pills
}

function cardClass(student) {
  return {
    "is-overdue": isPendingOverdue(student),
    "is-done": student.pickup_status === "picked_up"
  }
}

// 结构化授权人优先；字段缺失（老后端）或为空时回退兼容串，回退时不显示关系。
function authText(student) {
  const list = Array.isArray(student.authorized_person_list) ? student.authorized_person_list : []
  if (list.length) {
    return list
      .map(item => personLabel(item.name, item.relation))
      .filter(Boolean)
      .join("、")
  }
  const legacy = String(student.authorized_persons || "").trim()
  return legacy || "暂无"
}

function alertText(student) {
  return student.pickup_deadline
    ? `放学 ${student.pickup_deadline}，请尽快联系家长`
    : "已过放学时间，请尽快联系家长"
}

function pickerStatusText(student) {
  if (student.pickup_status === "not_arrived") return "需先入园"
  const label = MANUAL_STATUS[student.pickup_status] || "—"
  return isPendingOverdue(student) ? `${label} · 滞留` : label
}

function isNotified(student) {
  return Boolean(notified.value[student.student_id])
}

function mapPickupError(err) {
  const msg = String(err?.message || "登记失败")
  if (msg.includes("先登记入园")) return "该生今日尚未入园，请先登记入园后再登记接走"
  if (msg.includes("需填写理由")) return "重复接走或提前接走需填写理由"
  if (msg.includes("不在有效授权")) return "所选接送人授权已失效，请重新选择或标例外"
  return msg
}

async function loadClassName() {
  if (className.value) return
  try {
    const res = await fetchTeacherClassInfo()
    className.value = res.data?.name || ""
  } catch (err) {
    // 班级名只用于标题和弹层副标题；失败退回「本班」，不阻断接送数据
    console.warn("[pickup-index] load class failed:", err?.message || err)
  }
}

async function loadPickup() {
  const seq = ++loadSeq
  const filter = filterStatus.value
  loading.value = true
  loadError.value = ""
  today.value = todayKey()

  try {
    const studentsPromise = fetchTeacherPickupStudents({ status: filter }, { silent: true })
    // 默认筛选下滞留名单就是同一份；其他筛选要单独拉，否则滞留条会随筛选消失。
    const pendingPromise =
      filter === "not_picked_up"
        ? studentsPromise
        : fetchTeacherPickupStudents({ status: "not_picked_up" }, { silent: true })

    const [overviewRes, studentsRes, pendingRes] = await Promise.all([
      fetchTeacherPickupOverview({ silent: true }),
      studentsPromise,
      pendingPromise
    ])
    if (seq !== loadSeq) return

    overview.value = overviewRes.data || {}
    students.value = Array.isArray(studentsRes.data?.list) ? studentsRes.data.list : []
    pendingStudents.value = Array.isArray(pendingRes.data?.list) ? pendingRes.data.list : []
    hasLoaded.value = true
    if (!className.value) loadClassName()
  } catch (err) {
    if (seq !== loadSeq) return
    // 失败时绝不能留空列表：这一页的空列表语义是「孩子都被接走了」。
    console.warn("[pickup-index] load failed:", err?.message || err)
    loadError.value = "接送数据加载失败"
    overview.value = {}
    students.value = []
    pendingStudents.value = []
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
  }
}

function selectFilter(key) {
  if (key === filterStatus.value) return
  filterStatus.value = key
  loadPickup()
}

function goRecords() {
  uni.navigateTo({ url: "/pages/pickup/records" })
}

async function handleDropOff(student) {
  if (busy.value[student.student_id]) return
  busy.value = { ...busy.value, [student.student_id]: true }
  try {
    await manualTeacherPickup({ student_id: student.student_id, type: 1 }, { silent: true })
    uni.showToast({ title: `${student.name} 已入园`, icon: "success" })
    await loadPickup()
  } catch (err) {
    uni.showToast({ title: mapPickupError(err), icon: "none" })
  } finally {
    const next = { ...busy.value }
    delete next[student.student_id]
    busy.value = next
  }
}

async function handleNotify(student) {
  const studentId = student.student_id
  if (isNotified(student) || notifying.value[studentId]) return
  notifying.value = { ...notifying.value, [studentId]: true }
  try {
    await notifyOverdueParent(studentId, { silent: true })
    notified.value = { ...notified.value, [studentId]: true }
    uni.showToast({ title: `已通知 ${student.name} 家长`, icon: "none" })
  } catch (err) {
    uni.showToast({ title: err?.message || "通知失败", icon: "none" })
  } finally {
    const next = { ...notifying.value }
    delete next[studentId]
    notifying.value = next
  }
}

function hintNeedArrive(student) {
  uni.showToast({ title: `${student.name} 今日尚未入园，请先登记入园`, icon: "none" })
}

async function openPicker(intent = "authorized") {
  pickerIntent.value = intent
  showPicker.value = true
  pickerError.value = ""
  pickerLoading.value = true
  const seq = ++pickerSeq
  try {
    const res = await fetchTeacherPickupStudents({}, { silent: true })
    if (seq !== pickerSeq) return
    pickerStudents.value = Array.isArray(res.data?.list) ? res.data.list : []
  } catch (err) {
    if (seq !== pickerSeq) return
    pickerError.value = err?.message || "学生列表加载失败"
    pickerStudents.value = []
  } finally {
    if (seq === pickerSeq) pickerLoading.value = false
  }
}

function reloadPicker() {
  openPicker(pickerIntent.value)
}

function closePicker() {
  showPicker.value = false
}

function selectPickerStudent(student) {
  if (student.pickup_status === "not_arrived") {
    uni.showToast({ title: `${student.name} 今日尚未入园，请先登记入园`, icon: "none" })
    return
  }
  showPicker.value = false
  openManual(student, pickerIntent.value)
}

function openManual(student, mode) {
  manualStudent.value = student
  manualMode.value = mode || "authorized"
  manualRemark.value = ""
  manualOverrideName.value = ""
  selectedAuthId.value = null
  authPersons.value = []
  authPersonsError.value = ""
  showManual.value = true
  loadAuthPersons(student.student_id)
}

async function loadAuthPersons(studentId) {
  const seq = ++authSeq
  authPersonsLoading.value = true
  authPersonsError.value = ""
  try {
    const res = await fetchStudentAuthPersons(studentId, { silent: true })
    if (seq !== authSeq) return
    authPersons.value = Array.isArray(res.data?.list) ? res.data.list : []
  } catch (err) {
    if (seq !== authSeq) return
    authPersonsError.value = err?.message || "授权接送人加载失败"
    authPersons.value = []
  } finally {
    if (seq === authSeq) authPersonsLoading.value = false
  }
}

function reloadAuthPersons() {
  if (manualStudent.value) loadAuthPersons(manualStudent.value.student_id)
}

function closeManual() {
  if (submitting.value) return
  showManual.value = false
}

function switchMode(mode) {
  if (manualMode.value === mode) return
  manualMode.value = mode
  selectedAuthId.value = null
  manualOverrideName.value = ""
}

function selectAuthPerson(id) {
  selectedAuthId.value = id
}

async function submitManual() {
  if (!manualStudent.value || submitting.value) return
  if (manualMode.value === "authorized" && selectedAuthId.value === null) {
    uni.showToast({ title: "请先选择一位授权接送人", icon: "none" })
    return
  }
  if (manualMode.value === "override" && !manualOverrideName.value.trim()) {
    uni.showToast({ title: "请填写例外接送人姓名", icon: "none" })
    return
  }
  const reason = manualRemark.value.trim()
  // 契约：重复接走（当日第 2+ 次）没理由必被拒。前端先挡一次，避免白跑请求。
  if (manualStudent.value.pickup_status === "picked_up" && !reason) {
    uni.showToast({ title: "重复接走需填写理由", icon: "none" })
    return
  }

  const isOverride = manualMode.value === "override"
  submitting.value = true
  try {
    await manualTeacherPickup(
      {
        student_id: manualStudent.value.student_id,
        type: 2,
        authorization_id: isOverride ? null : selectedAuthId.value,
        is_authorized_override: isOverride,
        override_name: isOverride ? manualOverrideName.value.trim() : "",
        remark: reason
      },
      { silent: true }
    )
    const name = manualStudent.value.name
    showManual.value = false
    uni.showToast({
      title: isOverride ? `已按例外登记 · ${name}` : `${name} 登记成功`,
      icon: "none"
    })
    await loadPickup()
  } catch (err) {
    // 失败保留弹层：重复接走 / 提前接走缺理由时，用户可直接补填再提交。
    uni.showToast({ title: mapPickupError(err), icon: "none" })
  } finally {
    submitting.value = false
  }
}

function isCancelMsg(msg) {
  return /cancel/i.test(msg)
}

function isPermissionMsg(msg) {
  return /auth|permission|deny|denied|forbidden/i.test(msg)
}

function startScan() {
  uni.scanCode({
    success: res => {
      const code = res?.result || res?.scanCode || ""
      if (!code) {
        uni.showToast({ title: "二维码识别失败", icon: "none" })
        return
      }
      verifyQr(code)
    },
    fail: err => {
      const msg = String(err?.errMsg || err?.message || "")
      if (isCancelMsg(msg)) {
        uni.showToast({ title: "已取消扫码", icon: "none" })
        return
      }
      if (isPermissionMsg(msg)) {
        uni.showToast({ title: "未获得相机权限，请在微信设置中开启后重试", icon: "none" })
        return
      }
      uni.showToast({ title: "扫码失败，请重试", icon: "none" })
    }
  })
}

function resetScan() {
  scanPhase.value = "loading"
  scanError.value = ""
  scanResult.value = null
}

async function verifyQr(code) {
  const seq = ++scanSeq
  resetScan()
  showScan.value = true
  try {
    const res = await scanTeacherPickup({ qr_code: code }, { silent: true })
    if (seq !== scanSeq) {
      // 弹层已被关闭或被下一次扫码取代，但登记可能已经落库：
      // 不抢弹层，只刷新列表并给一条提示，避免「后台已接走、页面还显示在园」。
      loadPickup()
      const name = res.data?.student?.name || ""
      if (!showScan.value) {
        uni.showToast({ title: name ? `${name} 已登记` : "接送已登记", icon: "none" })
      }
      return
    }
    scanResult.value = res.data || null
    scanPhase.value = "success"
    await loadPickup()
  } catch (err) {
    if (seq !== scanSeq) return
    scanError.value = String(err?.message || "核验未通过")
    scanPhase.value = "failure"
  }
}

function closeScan() {
  // 关闭即放弃本次弹层展示；请求仍可能成功，由 verifyQr 的 seq 分支兜底刷新。
  scanSeq += 1
  showScan.value = false
}

function rescan() {
  startScan()
}

function scanOverride() {
  showScan.value = false
  openPicker("override")
}

function noop() {}

onShow(loadPickup)
onPullDownRefresh(loadPickup)
</script>

<template>
  <PageShell bottom-space="128rpx">
    <view class="page">
      <!-- 今日概览 -->
      <view v-if="showOverview" class="overview-card">
        <view class="overview-head">
          <text class="overview-title">今日接送</text>
          <text class="overview-date">{{ todayLabel }}</text>
        </view>
        <view class="stat-grid">
          <view class="stat">
            <view v-if="statsLoading" class="stat-skel" />
            <text v-else class="stat-num">{{ stats.notArrived }}</text>
            <text class="stat-label">未到园</text>
          </view>
          <view class="stat">
            <view v-if="statsLoading" class="stat-skel" />
            <text v-else class="stat-num is-hot">{{ stats.pending }}</text>
            <text class="stat-label">待接走</text>
          </view>
          <view class="stat">
            <view v-if="statsLoading" class="stat-skel" />
            <text v-else class="stat-num">{{ stats.picked }}</text>
            <text class="stat-label">已接走</text>
          </view>
          <view class="stat">
            <view v-if="statsLoading" class="stat-skel" />
            <text v-else class="stat-num is-danger">{{ stats.abnormal }}</text>
            <text class="stat-label">异常</text>
          </view>
        </view>
        <view v-if="!statsLoading && overdueStudents.length" class="overdue-strip">
          <text class="overdue-icon icon">&#xea1d;</text>
          <text class="overdue-text">{{ overdueText }}</text>
        </view>
      </view>

      <text v-if="showOverview" class="scan-hint">家长出示接送码，扫码即可登记</text>

      <!-- 四段筛选：每次切换都按真实 status 重新请求 -->
      <view v-if="showOverview" class="filter-bar">
        <view
          v-for="item in FILTERS"
          :key="item.key"
          class="filter-btn"
          :class="{ 'is-active': filterStatus === item.key }"
          @click="selectFilter(item.key)"
        >
          <text>{{ item.label }}</text>
        </view>
      </view>

      <view v-if="showListHead" class="list-head">
        <text class="list-title">{{ listTitle }}</text>
        <text v-if="className" class="list-meta">{{ className }}</text>
      </view>

      <!-- 错误态必须排在空态之前：失败时不能显示成「都接走了」 -->
      <view v-if="loadError" class="state-view">
        <view class="state-icon is-danger"><text class="icon">&#xeca1;</text></view>
        <text class="state-title">{{ loadError }}</text>
        <text class="state-desc">网络异常，未获取到今日接送数据，请勿默认孩子都已接走</text>
        <view class="state-btn" @click="loadPickup">重试</view>
      </view>

      <!-- 默认筛选为空有两种原因，不能一律说成「都接走了」 -->
      <view v-else-if="isPageEmpty" class="state-view">
        <view class="state-icon"><text class="icon">&#xeb81;</text></view>
        <text v-if="emptyKind === 'no_arrival'" class="state-title">今天还没有孩子入园</text>
        <text v-else class="state-title">今天没有待接的孩子</text>
        <text v-if="emptyKind === 'no_arrival'" class="state-desc">登记入园后，待接走的孩子会显示在这里</text>
        <text v-else class="state-desc">全部孩子已完成接送，下拉可刷新</text>
        <view class="state-btn" @click="goRecords">查看今日记录</view>
      </view>

      <!-- 加载：统计 + 列表骨架 -->
      <view v-else-if="loading" class="loading-list">
        <view v-for="i in 3" :key="i" class="skeleton-card">
          <view class="sk sk-avatar" />
          <view class="sk-lines">
            <view class="sk sk-line is-wide" />
            <view class="sk sk-line" />
            <view class="sk sk-line is-short" />
          </view>
          <view class="sk sk-action" />
        </view>
      </view>

      <text v-else-if="students.length === 0" class="filter-empty">当前筛选下没有学生</text>

      <!-- 学生卡片 -->
      <view v-else class="student-list">
        <view
          v-for="student in students"
          :key="student.student_id"
          class="student-card"
          :class="cardClass(student)"
        >
          <view class="s-avatar"><text>{{ initialOf(student.name) }}</text></view>
          <view class="s-main">
            <view class="s-name-row">
              <text class="s-name">{{ student.name }}</text>
              <text
                v-for="pill in statusPills(student)"
                :key="pill.key"
                class="s-pill"
                :class="'is-' + pill.tone"
              >{{ pill.label }}</text>
            </view>
            <text class="s-auth">授权接送人：{{ authText(student) }}</text>
            <text v-if="student.arrive_time" class="s-meta">入园 {{ student.arrive_time }}</text>
            <text v-else-if="student.pickup_status === 'not_arrived'" class="s-meta">今日未登记入园</text>
            <text v-if="isPendingOverdue(student)" class="s-alert">{{ alertText(student) }}</text>
            <text v-if="student.last_remark" class="s-remark">上次登记：{{ student.last_remark }}</text>
          </view>
          <view class="s-actions">
            <template v-if="student.pickup_status === 'not_arrived'">
              <view class="a-btn is-success" @click="handleDropOff(student)">入园</view>
              <view class="a-btn is-off" @click="hintNeedArrive(student)">接走</view>
            </template>
            <template v-else-if="student.pickup_status === 'picked_up'">
              <view class="a-btn is-quiet" @click="openManual(student, 'authorized')">再接走</view>
            </template>
            <template v-else>
              <view class="a-btn is-primary" @click="openManual(student, 'authorized')">接走</view>
              <view
                v-if="student.is_overdue"
                class="a-btn"
                :class="isNotified(student) ? 'is-notified' : 'is-danger'"
                @click="handleNotify(student)"
              >{{ isNotified(student) ? "已通知" : "通知" }}</view>
            </template>
          </view>
        </view>
      </view>
    </view>

    <!-- 底部三动作 -->
    <template #bottom>
      <view class="actionbar">
        <view class="actionbar-left">
          <view class="bar-action" @click="openPicker('authorized')">
            <text class="bar-icon icon">&#xec86;</text>
            <text>手动登记</text>
          </view>
          <view class="bar-action" @click="goRecords">
            <text class="bar-icon icon">&#xecef;</text>
            <text>接送记录</text>
          </view>
        </view>
        <view class="scan-action" @click="startScan">
          <text class="icon">&#xf041;</text>
          <text>扫码接送</text>
        </view>
      </view>
    </template>

    <!-- 选择学生 -->
    <view v-if="showPicker" class="sheet-layer">
      <view class="sheet-mask" @click="closePicker" @touchmove.stop.prevent="noop" />
      <view class="sheet">
        <view class="sheet-grab" />
        <view class="sheet-head">
          <view class="sheet-head-main">
            <text class="sheet-title">选择学生</text>
            <text class="sheet-sub">为哪位孩子登记接送</text>
          </view>
          <view class="sheet-close" @click="closePicker"><text class="icon">&#xeb99;</text></view>
        </view>
        <scroll-view scroll-y class="sheet-body">
          <StateBlock v-if="pickerLoading" loading loading-text="加载学生列表..." />
          <StateBlock v-else-if="pickerError" :error="pickerError" retryable @retry="reloadPicker" />
          <view v-else-if="pickerStudents.length === 0" class="picker-empty">
            <text>当前班级暂无学生</text>
          </view>
          <view v-else class="picker-list">
            <view
              v-for="student in pickerStudents"
              :key="student.student_id"
              class="picker-row"
              :class="{ 'is-off': student.pickup_status === 'not_arrived' }"
              @click="selectPickerStudent(student)"
            >
              <text class="picker-name">{{ student.name }}</text>
              <text class="picker-status">{{ pickerStatusText(student) }}</text>
            </view>
          </view>
        </scroll-view>
      </view>
    </view>

    <!-- 手动登记 -->
    <view v-if="showManual" class="sheet-layer">
      <view class="sheet-mask" @click="closeManual" @touchmove.stop.prevent="noop" />
      <view class="sheet is-tall">
        <view class="sheet-grab" />
        <view class="sheet-head">
          <view class="sheet-head-main">
            <text class="sheet-title">手动登记 · {{ manualStudent?.name }}</text>
            <text class="sheet-sub">{{ manualSub }}</text>
          </view>
          <view class="sheet-close" @click="closeManual"><text class="icon">&#xeb99;</text></view>
        </view>
        <scroll-view scroll-y class="sheet-body">
          <view class="mode-switch">
            <view
              class="mode-btn"
              :class="{ 'is-active': manualMode === 'authorized' }"
              @click="switchMode('authorized')"
            >选授权人</view>
            <view
              class="mode-btn"
              :class="{ 'is-active': manualMode === 'override' }"
              @click="switchMode('override')"
            >标例外</view>
          </view>

          <view v-if="manualMode === 'authorized'" class="mode-panel">
            <text class="field-label">有效授权接送人</text>
            <view v-if="authPersonsLoading" class="inline-state">
              <text>加载授权人...</text>
            </view>
            <view v-else-if="authPersonsError" class="inline-state">
              <text>{{ authPersonsError }}</text>
              <text class="inline-retry" @click="reloadAuthPersons">重试</text>
            </view>
            <view v-else-if="authPersons.length === 0" class="inline-note">
              <text>暂无有效授权接送人，可切换到「标例外」</text>
            </view>
            <view v-else class="person-list">
              <view
                v-for="person in authPersons"
                :key="person.id"
                class="person-card"
                :class="{ 'is-selected': selectedAuthId === person.id }"
                @click="selectAuthPerson(person.id)"
              >
                <view class="person-radio">
                  <view v-if="selectedAuthId === person.id" class="person-dot" />
                </view>
                <view class="person-avatar"><text>{{ initialOf(person.name) }}</text></view>
                <view class="person-info">
                  <text class="person-name">{{ personLabel(person.name, person.relation) }}</text>
                  <text class="person-relation">{{ person.phone || "—" }}</text>
                </view>
              </view>
            </view>
          </view>

          <view v-else class="mode-panel">
            <text class="field-label">来接者姓名</text>
            <input
              v-model="manualOverrideName"
              class="text-input"
              type="text"
              maxlength="20"
              placeholder="如：邻居王五"
            />
            <text class="field-hint">非授权接送人将按例外放行登记</text>
          </view>

          <view class="mode-panel">
            <text class="field-label">理由</text>
            <textarea
              v-model="manualRemark"
              class="text-area"
              maxlength="200"
              placeholder="重复接走 / 提前接走时必填"
            />
          </view>
        </scroll-view>
        <view class="sheet-footer">
          <view class="footer-btn">
            <view class="btn-secondary" @click="closeManual">取消</view>
          </view>
          <view class="footer-btn">
            <AppButton type="ghost" block :disabled="!canSubmitManual" @click="submitManual">
              {{ submitting ? "提交中..." : "确认登记" }}
            </AppButton>
          </view>
        </view>
      </view>
    </view>

    <!-- 扫码核验 -->
    <view v-if="showScan" class="sheet-layer">
      <view class="sheet-mask" @click="closeScan" @touchmove.stop.prevent="noop" />
      <view class="sheet">
        <view class="sheet-grab" />
        <view class="sheet-head">
          <view class="sheet-head-main">
            <text class="sheet-title">扫码核验</text>
            <text class="sheet-sub">扫家长出示的接送二维码</text>
          </view>
          <view class="sheet-close" @click="closeScan"><text class="icon">&#xeb99;</text></view>
        </view>
        <view class="scan-body">
          <view v-if="scanPhase === 'loading'" class="scan-state">
            <view class="spinner" />
            <text class="scan-title">正在核验接送码…</text>
            <text class="scan-desc">请让家长保持二维码清晰可见</text>
          </view>

          <view v-else-if="scanPhase === 'failure'" class="scan-state">
            <view class="scan-icon is-danger"><text class="icon">&#xeb99;</text></view>
            <text class="scan-title">核验未通过</text>
            <text class="scan-reason">{{ scanError }}</text>
            <text class="scan-desc">可改为「标例外」手动登记，并补充理由</text>
            <view class="scan-actions">
              <view class="scan-action-btn is-primary" @click="scanOverride">标例外登记</view>
              <view class="scan-action-btn" @click="rescan">重新扫码</view>
            </view>
          </view>

          <view v-else class="scan-state">
            <view class="scan-icon is-success"><text class="icon">&#xeb81;</text></view>
            <text class="scan-title">核验通过</text>
            <text class="scan-person">{{ scanStudentName }} · {{ scanTypeLabel }} · {{ scanTime }}</text>
            <view class="scan-meta">
              <view class="scan-meta-row">
                <text class="scan-meta-label">接送人</text>
                <text class="scan-meta-value">{{ scanPerson }}</text>
              </view>
              <view class="scan-meta-row">
                <text class="scan-meta-label">核验方式</text>
                <text class="scan-meta-value">扫码</text>
              </view>
            </view>
            <view class="scan-done">
              <AppButton type="ghost" block @click="closeScan">完成</AppButton>
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
  padding-bottom: $spacing-xl;
}

/* ---------- 今日概览 ---------- */
.overview-card {
  padding: $spacing-lg $spacing-xl;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.overview-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
}

.overview-title {
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
}

.overview-date {
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  color: $color-text-body;
}

.stat-grid {
  display: flex;
  gap: $spacing-sm;
  margin-top: $spacing-lg;
}

.stat {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
}

.stat-num {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 64rpx;
  height: 60rpx;
  padding: 0 $spacing-sm;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1;
}

.stat-num.is-hot {
  background: $color-brand-primary-bg;
}

.stat-num.is-danger {
  color: $color-semantic-danger;
}

.stat-label {
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
  line-height: 1.3;
  text-align: center;
}

.stat-skel {
  width: 64rpx;
  height: 44rpx;
  margin: $spacing-xs 0;
  border-radius: $border-radius-md;
  background: $color-border-subtle;
  animation: skeleton-pulse 1.5s $animation-easing-default infinite;
}

.overdue-strip {
  display: flex;
  align-items: flex-start;
  gap: $spacing-md;
  margin-top: $spacing-lg;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
}

.overdue-icon {
  flex: none;
  margin-top: 2rpx;
  color: $color-semantic-danger;
  font-size: $typography-font-size-base;
}

.overdue-text {
  flex: 1;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  color: $color-text-primary;
  line-height: 1.45;
}

.scan-hint {
  display: block;
  margin: $spacing-md 4rpx 0;
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

/* ---------- 四段筛选 ---------- */
.filter-bar {
  display: flex;
  gap: 6rpx;
  margin-top: $spacing-lg;
  padding: 6rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.filter-btn {
  flex: 1;
  min-width: 0;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
}

.filter-btn.is-active {
  background: $color-bg-card;
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.list-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin: $spacing-xl 4rpx $spacing-md;
}

.list-title {
  min-width: 0;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
}

.list-meta {
  flex: none;
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

/* ---------- 学生卡片 ---------- */
.student-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.student-card {
  display: flex;
  align-items: flex-start;
  gap: $spacing-lg;
  padding: $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.student-card.is-overdue {
  border-color: $color-semantic-danger;
}

.student-card.is-done {
  padding: $spacing-md $spacing-lg;
  border-color: transparent;
  background: transparent;
  box-shadow: none;
}

.s-avatar {
  flex: none;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-semantic-info-bg;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.student-card.is-done .s-avatar {
  background: $color-bg-page;
  color: $color-text-muted;
}

.s-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.s-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.s-name {
  min-width: 0;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
  overflow-wrap: anywhere;
}

.student-card.is-done .s-name {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  color: $color-text-body;
}

.s-pill {
  padding: 2rpx $component-badge-padding-x 4rpx;
  border-radius: $component-badge-radius;
  font-size: $component-badge-font-size;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.4;
  white-space: nowrap;
}

.s-pill.is-wait {
  border: 1rpx solid $color-border-hairline;
  background: transparent;
  color: $color-text-muted;
  font-weight: $typography-font-weight-medium;
}

.s-pill.is-pending {
  background: $color-bg-page;
  color: $color-text-body;
  font-weight: $typography-font-weight-medium;
}

.s-pill.is-done {
  background: $color-semantic-success-bg;
}

.s-pill.is-overdue {
  background: $color-semantic-danger-bg;
}

.s-pill.is-override {
  background: $color-semantic-warning-bg;
}

.s-auth {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.s-meta {
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
  line-height: 1.4;
}

.s-alert {
  align-self: flex-start;
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-md;
  background: $color-semantic-danger-bg;
  color: $color-text-primary;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

.s-remark {
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.s-actions {
  flex: none;
  width: 132rpx;
  align-self: center;
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
}

.a-btn {
  min-height: 88rpx;
  padding: 0 $spacing-md;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
  text-align: center;
}

.a-btn.is-primary {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.a-btn.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-text-primary;
}

.a-btn.is-success {
  background: $color-semantic-success-bg;
  color: $color-text-primary;
}

.a-btn.is-quiet {
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-card;
  color: $color-text-body;
  font-weight: $typography-font-weight-medium;
}

.a-btn.is-off {
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-page;
  color: $color-text-muted;
  font-weight: $typography-font-weight-medium;
}

.a-btn.is-notified {
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-page;
  color: $color-text-muted;
}

/* ---------- 加载骨架 ---------- */
.loading-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.skeleton-card {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
}

.sk {
  display: block;
  background: $color-border-subtle;
  border-radius: $border-radius-sm;
  animation: skeleton-pulse 1.5s $animation-easing-default infinite;
}

.sk-avatar {
  flex: none;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
}

.sk-lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.sk-line {
  width: 62%;
  height: 22rpx;
}

.sk-line.is-wide {
  width: 40%;
  height: 28rpx;
}

.sk-line.is-short {
  width: 30%;
  height: 20rpx;
}

.sk-action {
  flex: none;
  width: 132rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  opacity: 0.6;
}

/* ---------- 状态 ---------- */
.state-view {
  min-height: 60vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $spacing-lg;
  padding: 80rpx $spacing-xl;
  text-align: center;
}

.state-icon {
  width: 104rpx;
  height: 104rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-3xl;
}

.state-icon.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.state-title {
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
}

.state-desc {
  max-width: 480rpx;
  font-size: $typography-font-size-base;
  color: $color-text-muted;
  line-height: 1.6;
}

.state-btn {
  min-height: 88rpx;
  padding: 0 $spacing-xl;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.filter-empty {
  display: block;
  padding: 64rpx $spacing-xl;
  font-size: $typography-font-size-base;
  color: $color-text-muted;
  text-align: center;
}

/* ---------- 底部固定栏 ---------- */
.actionbar {
  height: calc(128rpx + env(safe-area-inset-bottom));
  padding: 0 $component-page-padding-x env(safe-area-inset-bottom);
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  border-top: 1rpx solid $color-border-hairline;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.actionbar-left {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.bar-action {
  min-width: 120rpx;
  height: 128rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.bar-icon {
  font-size: $typography-font-size-xl;
}

.scan-action {
  flex: none;
  height: 88rpx;
  padding: 0 $spacing-lg;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-md;
}

/* ---------- 底部弹层 ---------- */
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
  max-height: 82vh;
  padding-bottom: env(safe-area-inset-bottom);
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
  padding: $spacing-lg $spacing-xl $spacing-md;
}

.sheet-head-main {
  min-width: 0;
}

.sheet-title {
  display: block;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
}

.sheet-sub {
  display: block;
  margin-top: 4rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  line-height: 1.35;
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
  max-height: 60vh;
  padding: 0 $spacing-xl $spacing-lg;
  box-sizing: border-box;
}

.sheet-footer {
  flex: none;
  display: flex;
  gap: $spacing-md;
  padding: $spacing-md $spacing-xl $spacing-xl;
  border-top: 1rpx solid $color-border-hairline;
}

.footer-btn {
  flex: 1;
  min-width: 0;
}

.btn-secondary {
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
}

/* ---------- 选择学生 ---------- */
.picker-list {
  display: flex;
  flex-direction: column;
}

.picker-row {
  min-height: 104rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  border-bottom: 1rpx solid $color-border-subtle;
}

.picker-name {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
}

.picker-status {
  flex: none;
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

.picker-row.is-off .picker-name {
  color: $color-text-muted;
  font-weight: $typography-font-weight-medium;
}

.picker-row.is-off .picker-status {
  color: $color-text-hint;
}

.picker-empty {
  padding: 64rpx 0;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  text-align: center;
}

/* ---------- 手动登记 ---------- */
.mode-switch {
  display: flex;
  gap: 6rpx;
  padding: 6rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.mode-btn {
  flex: 1;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
}

.mode-btn.is-active {
  background: $color-bg-card;
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-sm;
}

.mode-panel {
  margin-top: $spacing-lg;
}

.field-label {
  display: block;
  margin-bottom: $spacing-sm;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  color: $color-text-body;
}

.field-hint {
  display: block;
  margin-top: $spacing-sm;
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.inline-state {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  padding: $spacing-lg 0;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.inline-retry {
  min-height: 88rpx;
  padding: 0 $spacing-md;
  display: flex;
  align-items: center;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

.inline-note {
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-semantic-warning-bg;
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.text-input {
  width: 100%;
  height: $component-input-height;
  padding: 0 $spacing-lg;
  box-sizing: border-box;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
}

.text-area {
  width: 100%;
  height: 168rpx;
  padding: $spacing-md $spacing-lg;
  box-sizing: border-box;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.6;
}

.person-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
}

.person-card {
  min-height: 112rpx;
  padding: $spacing-md $spacing-lg;
  display: flex;
  align-items: center;
  gap: $spacing-md;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
}

.person-card.is-selected {
  border-color: $color-brand-primary-light;
  background: $color-brand-primary-bg;
}

.person-radio {
  flex: none;
  width: 40rpx;
  height: 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2rpx solid $color-text-hint;
  border-radius: $border-radius-full;
}

.person-card.is-selected .person-radio {
  border-color: $color-brand-primary-dark;
}

.person-dot {
  width: 20rpx;
  height: 20rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-dark;
}

.person-avatar {
  flex: none;
  width: 72rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
}

.person-info {
  flex: 1;
  min-width: 0;
}

.person-name {
  display: block;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
}

.person-relation {
  display: block;
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

/* ---------- 扫码核验 ---------- */
.scan-body {
  padding: 0 $spacing-xl $spacing-xl;
}

.scan-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
  padding: $spacing-lg $spacing-md $spacing-xl;
  text-align: center;
}

.scan-title {
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
  line-height: 1.3;
}

.scan-desc {
  max-width: 480rpx;
  font-size: $typography-font-size-base;
  color: $color-text-muted;
  line-height: 1.55;
}

.scan-person {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
}

.scan-reason {
  width: 100%;
  padding: $spacing-md $spacing-lg;
  box-sizing: border-box;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  line-height: 1.55;
  text-align: left;
}

.scan-icon {
  width: 112rpx;
  height: 112rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-3xl;
}

.scan-icon.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.scan-icon.is-success {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.spinner {
  width: 68rpx;
  height: 68rpx;
  border: 6rpx solid $color-border-subtle;
  border-top-color: $color-brand-primary;
  border-radius: $border-radius-full;
  animation: spin 0.9s linear infinite;
}

.scan-meta {
  width: 100%;
  margin-top: $spacing-md;
  padding: $spacing-sm $spacing-lg;
  box-sizing: border-box;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.scan-meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  padding: $spacing-sm 0;
}

.scan-meta-row + .scan-meta-row {
  border-top: 1rpx solid $color-border-subtle;
}

.scan-meta-label {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.scan-meta-value {
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  color: $color-text-primary;
  text-align: right;
}

.scan-actions {
  width: 100%;
  display: flex;
  gap: $spacing-md;
  margin-top: $spacing-lg;
}

.scan-action-btn {
  flex: 1;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

.scan-action-btn.is-primary {
  border-color: transparent;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

.scan-done {
  width: 100%;
  margin-top: $spacing-lg;
}

@keyframes skeleton-pulse {
  0%,
  100% {
    opacity: 0.55;
  }
  50% {
    opacity: 1;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
