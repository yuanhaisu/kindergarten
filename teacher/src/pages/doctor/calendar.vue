<script setup>
import { computed, reactive, ref } from "vue"
import { onLoad, onShow } from "@dcloudio/uni-app"
import {
  fetchDoctorCalendarEvents,
  createDoctorCalendarEvent,
  updateDoctorCalendarEvent,
  deleteDoctorCalendarEvent
} from "@/api/doctor-calendar"
import { fetchDict } from "@/api/dict"
import {
  formatDateLabel,
  formatMonthLabel,
  formatWeekdayShort,
  monthKeyOf,
  todayKey
} from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

const TYPE_ORDER = ["checkup", "notice", "activity", "holiday", "meeting"]
const DEFAULT_TYPES = [
  { key: "checkup", label: "体检" },
  { key: "notice", label: "通知" },
  { key: "activity", label: "活动" },
  { key: "holiday", label: "假期" },
  { key: "meeting", label: "家长会" }
]

const events = ref([])
const loading = ref(true)
const loadError = ref("")
const typeOptions = ref(DEFAULT_TYPES)
const today = ref(todayKey())
const selectedDate = ref("")
const collapsedDates = reactive({})

const sheetVisible = ref(false)
const sheetMode = ref("create")
const editId = ref(0)
const original = ref(null)
const submitting = ref(false)
const submittingKind = ref("")
const confirmVisible = ref(false)
const confirmLines = ref([])
const confirmNotify = ref(true)
const textDialogVisible = ref(false)
const textDialogKind = ref("date")
const textDialogValue = ref("")
const deleteVisible = ref(false)
const deleteTarget = ref(null)
const deleting = ref(false)
let pendingEdit = null
let skipNextShow = true

const emptyForm = () => ({
  title: "",
  event_type: "checkup",
  description: "",
  event_date: "",
  start_time: "09:00",
  end_time: "",
  location: "",
  is_all_day: false,
  notify: true
})
const form = reactive(emptyForm())

const typeMap = computed(() => {
  const map = {}
  typeOptions.value.forEach((item) => {
    map[item.key] = item.label
  })
  return map
})

const viewState = computed(() => {
  if (loading.value) return "loading"
  if (loadError.value) return "error"
  return events.value.length ? "normal" : "empty"
})
const total = computed(() => events.value.length)

const timeInvalid = computed(() => {
  if (!form.start_time || !form.end_time) return false
  return form.end_time <= form.start_time
})
const isPublishedEdit = computed(
  () => sheetMode.value === "edit" && original.value && original.value.status === "published"
)
const notifyVisible = computed(() => !isPublishedEdit.value)
const editNoteText = computed(() => {
  if (!original.value) return ""
  return original.value.status === "draft"
    ? "该事件仍为草稿——保存发布后，全园家长即可在日历中看到。"
    : "该事件已发布——保存时将让你确认是否通知家长（按变更内容预置建议）。"
})
const textDialogTitle = computed(() => {
  if (textDialogKind.value === "date") return "事件日期"
  return textDialogKind.value === "start" ? "开始时间" : "结束时间"
})
const textDialogHint = computed(() => {
  if (textDialogKind.value === "date") return "按 YYYY-MM-DD 填写"
  return textDialogKind.value === "start" ? "24 小时制，如 09:00" : "24 小时制，留空保持原值"
})
const textDialogMaxLength = computed(() => (textDialogKind.value === "date" ? 10 : 5))

function dateInfoOf(date) {
  const isToday = date === today.value
  return {
    day: Number(String(date).slice(8, 10)) || 0,
    monthLabel: formatMonthLabel(monthKeyOf(date)),
    label: isToday
      ? `今天 · ${formatWeekdayShort(date)}`
      : `${formatDateLabel(date)} ${formatWeekdayShort(date)}`,
    isToday
  }
}

function sortRows(rows) {
  return rows.slice().sort((a, b) => {
    if (!!a.is_all_day !== !!b.is_all_day) return a.is_all_day ? -1 : 1
    return String(a.start_time || "").localeCompare(String(b.start_time || ""))
  })
}
function rowsOf(date) {
  return sortRows(events.value.filter((event) => event.event_date === date))
}

const groups = computed(() => {
  const dates = Array.from(
    new Set(events.value.map((event) => event.event_date).filter(Boolean))
  )
  const upcoming = dates.filter((date) => date >= today.value).sort()
  const past = dates
    .filter((date) => date < today.value)
    .sort((a, b) => b.localeCompare(a))
  const make = (list, isPast) =>
    list.map((date) => ({ date, isPast, ...dateInfoOf(date), rows: rowsOf(date) }))
  return { upcoming: make(upcoming, false), past: make(past, true) }
})

const agendaItems = computed(() => {
  const items = groups.value.upcoming.map((group) => ({ kind: "group", ...group }))
  if (groups.value.past.length) items.push({ kind: "divider", key: "past-divider" })
  groups.value.past.forEach((group) => items.push({ kind: "group", ...group }))
  return items
})

function typeLabelOf(event) {
  return event.event_type_label || typeMap.value[event.event_type] || event.event_type || "通知"
}
function timeOf(event) {
  if (event.is_all_day) return "全天"
  if (!event.start_time) return "未设"
  return event.end_time ? `${event.start_time}–${event.end_time}` : event.start_time
}
function isCollapsed(date) {
  return Boolean(collapsedDates[date])
}
function toggleCollapse(date) {
  if (collapsedDates[date]) delete collapsedDates[date]
  else collapsedDates[date] = true
}
// 选中与折叠拆开：选中只负责「今天 vs 选中」的视觉语义并顺带展开，
// 折叠交给右侧 chevron，否则刚选中的组会立刻把事件藏起来。
function toggleSelect(date) {
  if (selectedDate.value === date) {
    selectedDate.value = ""
    return
  }
  selectedDate.value = date
  if (collapsedDates[date]) delete collapsedDates[date]
}

async function loadTypes() {
  try {
    const res = await fetchDict("calendar_event_type")
    const items = (res.data?.items || []).map((item) => ({
      key: item.item_key,
      label: item.item_label || item.item_key
    }))
    if (!items.length) return
    const rank = (key) => {
      const index = TYPE_ORDER.indexOf(key)
      return index === -1 ? TYPE_ORDER.length : index
    }
    typeOptions.value = items.slice().sort((a, b) => rank(a.key) - rank(b.key))
  } catch {
    // 字典不可用时保留内置 5 类，表单仍可发布
  }
}

async function loadEvents({ retry = false, background = false } = {}) {
  if (!background) loading.value = true
  const previousError = loadError.value
  loadError.value = ""
  today.value = todayKey()
  try {
    const res = await fetchDoctorCalendarEvents({}, { silent: true })
    events.value = res.data?.list || []
    if (retry) uni.showToast({ title: "已重新加载", icon: "none" })
  } catch {
    if (background) {
      // 保存/删除已经成功，列表刷新失败只提示，不要把整页打回错误态。
      loadError.value = previousError
      uni.showToast({ title: "刷新列表失败", icon: "none" })
      return
    }
    events.value = []
    loadError.value = "网络异常，未获取到园级事件。"
    uni.showToast({ title: "加载事件失败", icon: "none" })
  } finally {
    if (!background) loading.value = false
  }
}

function openCreate() {
  if (submitting.value) return
  sheetMode.value = "create"
  editId.value = 0
  original.value = null
  Object.assign(form, emptyForm())
  if (typeOptions.value.length && !typeOptions.value.some((item) => item.key === form.event_type)) {
    form.event_type = typeOptions.value[0].key
  }
  sheetVisible.value = true
}

function openEdit(event) {
  if (submitting.value) return
  sheetMode.value = "edit"
  editId.value = event.id
  original.value = { ...event }
  Object.assign(form, {
    title: event.title || "",
    event_type: event.event_type || "checkup",
    description: event.description || "",
    event_date: event.event_date || "",
    start_time: event.start_time || "",
    end_time: event.end_time || "",
    location: event.location || "",
    is_all_day: !!event.is_all_day,
    notify: event.status === "draft" ? false : event.notify !== false
  })
  if (!typeOptions.value.some((item) => item.key === form.event_type)) {
    typeOptions.value = [
      ...typeOptions.value,
      { key: form.event_type, label: event.event_type_label || form.event_type }
    ]
  }
  sheetVisible.value = true
}

function closeSheet(force = false) {
  if (submitting.value && !force) return
  sheetVisible.value = false
}

function cancelConfirm() {
  if (submitting.value) return
  confirmVisible.value = false
  pendingEdit = null
}

function noop() {}
function setType(key) {
  if (submitting.value) return
  form.event_type = key
}
function onTitleInput(event) {
  form.title = event.detail?.value ?? ""
}
function onDescInput(event) {
  form.description = event.detail?.value ?? ""
}
function onLocationInput(event) {
  form.location = event.detail?.value ?? ""
}

function isValidDate(value) {
  const match = String(value).match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (!match) return false
  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  const date = new Date(year, month - 1, day)
  return date.getFullYear() === year && date.getMonth() === month - 1 && date.getDate() === day
}
function isValidTime(value) {
  return /^([01]\d|2[0-3]):[0-5]\d$/.test(value)
}

// 服务端不校验日期/时间格式，前端必须在这里拦下；留空表示保持原值。
// 用页内可编辑对话框而不是原生 showModal：样式与设计稿一致，也便于自动化验证。
function openTextDialog(kind) {
  if (submitting.value) return
  textDialogKind.value = kind
  textDialogValue.value =
    kind === "date" ? form.event_date : kind === "start" ? form.start_time : form.end_time
  textDialogVisible.value = true
}
function closeTextDialog() {
  if (submitting.value) return
  textDialogVisible.value = false
}
function onTextDialogInput(event) {
  textDialogValue.value = event.detail?.value ?? ""
}
function confirmTextDialog() {
  const next = String(textDialogValue.value || "").trim()
  if (!next) {
    closeTextDialog()
    return
  }
  if (textDialogKind.value === "date") {
    if (!isValidDate(next)) {
      uni.showToast({ title: "请按 YYYY-MM-DD 格式填写日期", icon: "none" })
      return
    }
    form.event_date = next
    closeTextDialog()
    return
  }
  if (!isValidTime(next)) {
    uni.showToast({ title: "请按 HH:MM 格式填写时间", icon: "none" })
    return
  }
  if (textDialogKind.value === "start") form.start_time = next
  if (textDialogKind.value === "end") form.end_time = next
  closeTextDialog()
}

function validate() {
  if (!form.title.trim()) {
    uni.showToast({ title: "请填写事件标题", icon: "none" })
    return false
  }
  if (!form.event_date) {
    uni.showToast({ title: "请选择事件日期", icon: "none" })
    return false
  }
  // 输入对话框已经校验过一次；这里再兜一次底，防止旧数据/异常字段绕过格式检查。
  if (!isValidDate(form.event_date)) {
    uni.showToast({ title: "请按 YYYY-MM-DD 格式填写日期", icon: "none" })
    return false
  }
  if (form.start_time && !isValidTime(form.start_time)) {
    uni.showToast({ title: "请按 HH:MM 格式填写时间", icon: "none" })
    return false
  }
  if (form.end_time && !isValidTime(form.end_time)) {
    uni.showToast({ title: "请按 HH:MM 格式填写时间", icon: "none" })
    return false
  }
  if (timeInvalid.value) {
    uni.showToast({ title: "结束时间需晚于开始时间", icon: "none" })
    return false
  }
  return true
}

function buildPayload(status, notify) {
  return {
    title: form.title.trim(),
    event_date: form.event_date,
    event_type: form.event_type,
    description: form.description.trim(),
    start_time: form.start_time || "",
    end_time: form.end_time || "",
    location: form.location.trim(),
    is_all_day: original.value ? !!original.value.is_all_day : false,
    status,
    notify
  }
}

async function persist(status, notify) {
  if (submitting.value) return
  submitting.value = true
  submittingKind.value = status === "draft" ? "draft" : "publish"
  const startedAt = Date.now()
  try {
    const payload = buildPayload(status, notify)
    if (editId.value) await updateDoctorCalendarEvent(editId.value, payload, { silent: true })
    else await createDoctorCalendarEvent(payload, { silent: true })
    // 设计稿要求「保存中…/发布中…」至少可见 800ms；真实请求更快时补足，
    // 否则按钮文案只闪一帧，用户会把重复点击当成没反应。
    const remain = 800 - (Date.now() - startedAt)
    if (remain > 0) await new Promise((resolve) => setTimeout(resolve, remain))
    uni.showToast({
      title: status === "draft" ? "已存草稿" : editId.value ? "已保存" : "发布成功",
      icon: "success"
    })
    confirmVisible.value = false
    pendingEdit = null
    closeSheet(true)
    // 不 await：后台刷新不延长提交禁用态；loadEvents 内部已兜住错误。
    loadEvents({ background: true })
  } catch (err) {
    uni.showToast({ title: err?.message || "保存失败", icon: "none" })
  } finally {
    submitting.value = false
    submittingKind.value = ""
  }
}

function submitDraft() {
  if (submitting.value || !validate()) return
  persist("draft", false)
}

function rangeOf(event) {
  if (!event) return "未设"
  if (event.is_all_day) return "全天"
  if (!event.start_time) return "未设"
  return event.end_time ? `${event.start_time}–${event.end_time}` : event.start_time
}
function buildChanges() {
  const before = original.value
  if (!before) return []
  const lines = []
  if ((before.event_date || "") !== form.event_date) {
    lines.push({ label: "日期", value: `${before.event_date || "未设"} → ${form.event_date || "未设"}` })
  }
  const beforeRange = rangeOf(before)
  const afterRange = rangeOf(form)
  if (beforeRange !== afterRange) {
    lines.push({ label: "时间", value: `${beforeRange} → ${afterRange}` })
  }
  if ((before.location || "") !== form.location.trim()) {
    lines.push({ label: "地点", value: `${before.location || "无"} → ${form.location.trim() || "无"}` })
  }
  if ((before.title || "") !== form.title.trim()) lines.push({ label: "标题已更新", value: "" })
  if ((before.description || "") !== form.description.trim()) lines.push({ label: "内容已更新", value: "" })
  if ((before.event_type || "") !== form.event_type) lines.push({ label: "类型已更新", value: "" })
  return lines
}
function hasSubstantiveChange() {
  const before = original.value
  if (!before) return false
  return (
    (before.event_date || "") !== form.event_date ||
    rangeOf(before) !== rangeOf(form) ||
    (before.location || "") !== form.location.trim()
  )
}

function submitPublish() {
  if (submitting.value || !validate()) return
  if (isPublishedEdit.value) {
    const lines = buildChanges()
    if (!lines.length) {
      persist("published", false)
      return
    }
    pendingEdit = { id: editId.value }
    confirmLines.value = lines
    confirmNotify.value = hasSubstantiveChange()
    confirmVisible.value = true
    return
  }
  persist("published", form.notify)
}

function saveConfirm() {
  if (submitting.value || !pendingEdit) return
  persist("published", confirmNotify.value)
}

function openDelete(event) {
  if (submitting.value || deleting.value) return
  deleteTarget.value = event
  deleteVisible.value = true
}
function closeDelete() {
  if (deleting.value) return
  deleteVisible.value = false
  deleteTarget.value = null
}
async function confirmDelete() {
  if (!deleteTarget.value || deleting.value) return
  const event = deleteTarget.value
  deleting.value = true
  try {
    await deleteDoctorCalendarEvent(event.id, { silent: true })
    if (editId.value === event.id) closeSheet(true)
    deleteVisible.value = false
    deleteTarget.value = null
    uni.showToast({ title: "已删除", icon: "success" })
    loadEvents({ background: true })
  } catch (err) {
    uni.showToast({ title: err?.message || "删除失败", icon: "none" })
  } finally {
    deleting.value = false
  }
}

onLoad(() => {
  loadTypes()
  loadEvents()
})
// 发布/编辑/删除都走本页，返回时仍需刷新：别的入口可能改过数据。
onShow(() => {
  if (skipNextShow) {
    skipNextShow = false
    return
  }
  loadEvents({ background: true })
})
</script>

<template>
  <PageShell max-width="430px">
    <view class="agenda-page">
      <view class="scope-panel">
        <view class="scope-pill">
          <text class="icon scope-icon">&#xedd1;</text>
          <text>全园家长可见</text>
        </view>
        <text class="scope-desc">发布后，全园老师与家长会在日历中同步看到。</text>
      </view>

      <view class="agenda-toolbar">
        <view class="agenda-heading">
          <text class="agenda-title">园级日程</text>
          <view class="count-slot">
            <view v-if="viewState === 'loading'" class="count-skeleton" />
            <view v-else-if="viewState === 'error'" class="count-line"><text>园级事件未载入</text></view>
            <view v-else-if="viewState === 'empty'" class="count-line"><text>共 0 场</text></view>
            <view v-else class="count-line">
              <text>共 </text>
              <text class="count-strong">{{ total }}</text>
              <text> 场</text>
              <text class="count-unit">园级事件</text>
            </view>
          </view>
        </view>
        <view v-if="viewState !== 'empty'" class="btn-publish" @click="openCreate">
          <text class="icon btn-icon">&#xea13;</text>
          <text>发布</text>
        </view>
      </view>

      <view v-if="viewState === 'loading'" class="skeleton-list">
        <view class="sk-month" />
        <view class="sk-head">
          <view class="sk-date" />
          <view class="sk-copy">
            <view class="sk-line is-title" />
            <view class="sk-line is-meta" />
          </view>
        </view>
        <view v-for="i in 2" :key="i" class="sk-card">
          <view class="sk-line is-tag" />
          <view class="sk-line is-title" />
          <view class="sk-line" />
          <view class="sk-line is-short" />
        </view>
      </view>

      <view v-else-if="viewState === 'error'" class="state-panel">
        <view class="error-mark"><text>!</text></view>
        <text class="state-title">事件加载失败</text>
        <text class="state-desc">网络异常，未获取到园级事件。</text>
        <view class="btn-ghost state-cta" @click="loadEvents({ retry: true })"><text>重新加载</text></view>
      </view>

      <view v-else-if="viewState === 'empty'" class="state-panel">
        <view class="empty-art">
          <view class="empty-orbit" />
          <view class="empty-page">
            <view class="empty-binding" />
            <text class="empty-day">26</text>
          </view>
          <view class="empty-dot" />
        </view>
        <text class="state-title">还没有园级事件</text>
        <text class="state-desc">发布后全园家长都能在日历中看到。</text>
        <view class="btn-primary state-cta" @click="openCreate">
          <text class="icon btn-icon">&#xea13;</text>
          <text>发布第一条事件</text>
        </view>
      </view>

      <view v-else class="agenda-groups">
        <template v-for="item in agendaItems" :key="item.kind === 'divider' ? item.key : item.date">
          <view v-if="item.kind === 'divider'" class="past-divider"><text>已结束</text></view>
          <view
            v-else
            class="agenda-group"
            :class="{
              'is-selected': selectedDate === item.date,
              'is-collapsed': isCollapsed(item.date),
              'is-past': item.isPast
            }"
          >
            <view class="month-row"><text class="month-tag">{{ item.monthLabel }}</text></view>
            <view class="day-head">
              <view class="day-select" @click="toggleSelect(item.date)">
                <text class="date-badge" :class="{ 'is-today': item.isToday }">{{ item.day }}</text>
                <view class="day-copy">
                  <text class="day-title">{{ item.label }}</text>
                  <text class="day-sub">{{ item.rows.length }} 场</text>
                </view>
              </view>
              <view
                class="day-toggle"
                :aria-expanded="!isCollapsed(item.date)"
                @click="toggleCollapse(item.date)"
              >
                <text class="icon day-chevron" :class="{ 'is-open': !isCollapsed(item.date) }">&#xea4e;</text>
              </view>
            </view>

            <view v-if="!isCollapsed(item.date)" class="day-events">
              <view v-for="event in item.rows" :key="event.id" class="event-card">
                <view class="event-open">
                  <view class="event-top">
                    <text class="type-tag">{{ typeLabelOf(event) }}</text>
                    <text v-if="event.status === 'draft'" class="draft-tag">草稿</text>
                    <text class="event-time">{{ timeOf(event) }}</text>
                  </view>
                  <text class="event-title">{{ event.title }}</text>
                  <text v-if="event.description" class="event-desc">{{ event.description }}</text>
                  <view class="event-meta">
                    <view v-if="event.location" class="event-place">
                      <text class="icon place-icon">&#xef14;</text>
                      <text class="place-text">{{ event.location }}</text>
                    </view>
                    <text class="scope-tag">全园</text>
                  </view>
                </view>
                <view class="event-actions">
                  <view class="event-action" @click="openEdit(event)"><text>编辑</text></view>
                  <view class="event-action is-danger" @click="openDelete(event)"><text>删除</text></view>
                </view>
              </view>
            </view>
          </view>
        </template>
      </view>
    </view>

    <!-- 发布 / 编辑表单 -->
    <view v-if="sheetVisible" class="sheet-layer">
      <view class="sheet-scrim" @click="closeSheet" @touchmove.stop.prevent="noop" />
      <view class="sheet">
        <view class="sheet-grab" />
        <view class="sheet-head">
          <text class="sheet-title">{{ sheetMode === "edit" ? "编辑园级事件" : "发布园级事件" }}</text>
          <view class="sheet-close" @click="closeSheet">
            <text class="icon">&#xeb99;</text>
          </view>
        </view>

        <scroll-view scroll-y class="sheet-scroll">
          <view v-if="sheetMode === 'edit'" class="edit-note">
            <text class="edit-note-mark">i</text>
            <text class="edit-note-text">{{ editNoteText }}</text>
          </view>

          <view class="field">
            <text class="field-label">事件标题<text class="required-note">必填</text></text>
            <input
              class="input"
              :value="form.title"
              maxlength="60"
              placeholder="如：全园体检"
              placeholder-style="color: #94A3B8"
              @input="onTitleInput"
            />
          </view>

          <view class="field">
            <text class="field-label">事件类型</text>
            <view class="type-switch">
              <view
                v-for="option in typeOptions"
                :key="option.key"
                class="type-option"
                :class="{ 'is-selected': form.event_type === option.key }"
                @click="setType(option.key)"
              ><text>{{ option.label }}</text></view>
            </view>
          </view>

          <view class="field">
            <text class="field-label">事件说明</text>
            <textarea
              class="input textarea"
              :value="form.description"
              maxlength="500"
              placeholder="如体检项目、注意事项、需家长配合的事项"
              placeholder-style="color: #94A3B8"
              @input="onDescInput"
            />
          </view>

          <view class="field">
            <text class="field-label">事件日期<text class="required-note">必填</text></text>
            <view class="input input-button" @click="openTextDialog('date')">
              <text class="control-value" :class="{ 'is-placeholder': !form.event_date }">
                {{ form.event_date || "请选择日期" }}
              </text>
              <text class="icon control-icon">&#xea4e;</text>
            </view>
          </view>

          <view class="field-pair">
            <view class="field">
              <text class="field-label">开始时间</text>
              <view class="input input-button" @click="openTextDialog('start')">
                <text class="control-value" :class="{ 'is-placeholder': !form.start_time }">
                  {{ form.start_time || "未设" }}
                </text>
                <text class="icon control-icon">&#xea4e;</text>
              </view>
            </view>
            <view class="field">
              <text class="field-label">结束时间</text>
              <view class="input input-button" :class="{ 'is-invalid': timeInvalid }" @click="openTextDialog('end')">
                <text class="control-value" :class="{ 'is-placeholder': !form.end_time }">
                  {{ form.end_time || "未设" }}
                </text>
                <text class="icon control-icon">&#xea4e;</text>
              </view>
            </view>
          </view>
          <text v-if="timeInvalid" class="field-error">结束时间需晚于开始时间</text>

          <view class="field">
            <text class="field-label">事件地点</text>
            <input
              class="input"
              :value="form.location"
              maxlength="60"
              placeholder="如：医务室"
              placeholder-style="color: #94A3B8"
              @input="onLocationInput"
            />
          </view>

          <view v-if="notifyVisible" class="switch-row" @click="form.notify = !form.notify">
            <view class="switch-copy">
              <text class="switch-title">通知家长</text>
              <text class="switch-help">发布后向全园家长推送通知</text>
            </view>
            <view class="switch" :class="{ 'is-on': form.notify }"><view class="switch-track"><view class="switch-thumb" /></view></view>
          </view>
        </scroll-view>

        <view class="sheet-footer" :class="{ 'is-edit': sheetMode === 'edit' }">
          <template v-if="sheetMode === 'edit'">
            <view class="btn btn-quiet" @click="closeSheet"><text>取消编辑</text></view>
            <view class="btn btn-ghost" :aria-disabled="submitting" @click="submitDraft">
              <text>{{ submittingKind === "draft" ? "保存中…" : "存为草稿" }}</text>
            </view>
            <view class="btn btn-primary" :aria-disabled="submitting" @click="submitPublish">
              <text>{{ submittingKind === "publish" ? "保存中…" : "保存发布" }}</text>
            </view>
          </template>
          <template v-else>
            <view class="btn btn-ghost" :aria-disabled="submitting" @click="submitDraft">
              <text>{{ submittingKind === "draft" ? "保存中…" : "存为草稿" }}</text>
            </view>
            <view class="btn btn-primary" :aria-disabled="submitting" @click="submitPublish">
              <text>{{ submittingKind === "publish" ? "发布中…" : "立即发布" }}</text>
            </view>
          </template>
        </view>
      </view>
    </view>

    <!-- 已发布变更：通知确认 -->
    <view v-if="confirmVisible" class="sheet-layer is-confirm">
      <view class="sheet-scrim" @click="cancelConfirm" @touchmove.stop.prevent="noop" />
      <view class="sheet confirm-sheet">
        <view class="sheet-grab" />
        <view class="confirm-head">
          <text class="sheet-title">通知家长？</text>
          <text class="confirm-sub">将通知全园家长</text>
        </view>
        <view class="confirm-body">
          <text class="change-caption">本次变更</text>
          <view class="change-list">
            <view v-for="(line, index) in confirmLines" :key="index" class="change-row">
              <view class="change-bullet" />
              <text class="change-label">{{ line.label }}</text>
              <text v-if="line.value" class="change-value">{{ line.value }}</text>
            </view>
          </view>
          <view class="switch-row confirm-notify" @click="confirmNotify = !confirmNotify">
            <view class="switch-copy">
              <text class="switch-title">发布变更通知</text>
              <text class="switch-help">家长端会在日历顶部看到此变更</text>
            </view>
            <view class="switch" :class="{ 'is-on': confirmNotify }"><view class="switch-track"><view class="switch-thumb" /></view></view>
          </view>
        </view>
        <view class="sheet-footer">
          <view class="btn btn-ghost" @click="cancelConfirm"><text>取消</text></view>
          <view class="btn btn-primary" :aria-disabled="submitting" @click="saveConfirm"><text>保存</text></view>
        </view>
      </view>
    </view>

    <!-- 微信风格可编辑对话框：日期 / 时间 -->
    <view v-if="textDialogVisible" class="dialog-layer">
      <view class="dialog-scrim" @click="closeTextDialog" @touchmove.stop.prevent="noop" />
      <view class="wx-dialog">
        <text class="wx-dialog-title">{{ textDialogTitle }}</text>
        <text class="wx-dialog-hint">{{ textDialogHint }}</text>
        <input
          class="wx-dialog-input"
          :value="textDialogValue"
          :maxlength="textDialogMaxLength"
          :focus="textDialogVisible"
          @input="onTextDialogInput"
        />
        <view class="wx-dialog-actions">
          <view class="wx-dialog-button" @click="closeTextDialog"><text>取消</text></view>
          <view class="wx-dialog-button is-strong" @click="confirmTextDialog"><text>确定</text></view>
        </view>
      </view>
    </view>

    <!-- 微信风格 alertdialog：删除事件 -->
    <view v-if="deleteVisible" class="dialog-layer">
      <view class="dialog-scrim" @click="closeDelete" @touchmove.stop.prevent="noop" />
      <view class="wx-alert">
        <text class="wx-dialog-title">删除事件</text>
        <text class="wx-alert-body">确定删除「{{ deleteTarget?.title }}」吗？删除后全园家长不再可见。</text>
        <view class="wx-dialog-actions">
          <view class="wx-dialog-button" @click="closeDelete"><text>取消</text></view>
          <view class="wx-dialog-button is-danger" :aria-disabled="deleting" @click="confirmDelete">
            <text>{{ deleting ? "删除中…" : "删除" }}</text>
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

.agenda-page {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

/* ---------- 范围条 ---------- */
.scope-panel {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  padding: $spacing-lg 28rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.scope-pill {
  display: inline-flex;
  align-items: center;
  gap: $spacing-sm;
  min-height: 56rpx;
  padding: 6rpx $component-badge-padding-x;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-text-primary;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
}
.scope-icon {
  flex: none;
  font-size: $typography-font-size-base;
  color: $color-brand-primary-dark;
}
.scope-desc {
  margin-top: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

/* ---------- 工具栏 ---------- */
.agenda-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  margin-top: $spacing-xl;
  margin-bottom: 20rpx;
}
.agenda-heading {
  display: flex;
  flex-direction: column;
  gap: 2rpx;
  min-width: 0;
}
.agenda-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
}
.count-slot {
  min-height: 40rpx;
}
.count-line {
  display: flex;
  align-items: baseline;
  min-height: 40rpx;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}
.count-strong {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
  font-variant-numeric: tabular-nums;
}
.count-unit {
  color: $color-text-body;
}
.count-skeleton {
  width: 184rpx;
  height: 20rpx;
  margin: 10rpx 0;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  animation: sk-pulse 1.4s $animation-easing-default infinite;
}
.btn-publish {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  min-width: 176rpx;
  min-height: 88rpx;
  padding: 0 36rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-md;
}
.btn-publish:active {
  opacity: 0.86;
}
.btn-icon {
  font-size: $typography-font-size-base;
}

/* ---------- 日期分组 ---------- */
.agenda-groups {
  display: flex;
  flex-direction: column;
  gap: 36rpx;
}
.agenda-group {
  min-width: 0;
}
.month-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-height: 32rpx;
  margin: 0 4rpx $spacing-xs;
}
.month-tag {
  flex: none;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
}
.month-row::after {
  content: "";
  flex: 1;
  height: 2rpx;
  background: $color-border-subtle;
}
.day-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 88rpx;
  align-items: center;
  gap: 4rpx;
  width: 100%;
}
.day-select {
  display: grid;
  grid-template-columns: 88rpx minmax(0, 1fr);
  align-items: center;
  gap: $spacing-lg;
  width: 100%;
  min-width: 0;
  min-height: 104rpx;
  padding: $spacing-xs 4rpx;
  border-radius: $border-radius-xl;
}
.day-select:active {
  background: $color-brand-primary-bg;
}
.date-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  box-sizing: border-box;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}
.date-badge.is-today {
  border-color: $color-brand-primary-light;
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-bold;
}
.agenda-group.is-selected .date-badge {
  border-color: $color-brand-primary;
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
}
.day-copy {
  display: flex;
  flex-direction: column;
  gap: 2rpx;
  min-width: 0;
}
.day-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
}
.day-sub {
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}
.day-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
}
.day-toggle:active {
  background: $color-brand-primary-bg;
}
.day-chevron {
  font-size: $typography-font-size-xl;
  color: $color-text-muted;
  transition: transform $animation-duration-base $animation-easing-default;
}
.day-chevron.is-open {
  transform: rotate(180deg);
}
.agenda-group.is-collapsed .day-events {
  display: none;
}
.day-events {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  margin-top: $spacing-md;
}
.past-divider {
  display: flex;
  align-items: center;
  gap: 20rpx;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
}
.past-divider::before,
.past-divider::after {
  content: "";
  flex: 1;
  height: 2rpx;
  background: $color-border-subtle;
}

/* ---------- 事件卡 ---------- */
.event-card {
  min-width: 0;
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.event-open {
  padding: 28rpx $spacing-xl $spacing-lg;
}
.event-top {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  min-width: 0;
}
.type-tag,
.draft-tag,
.scope-tag {
  display: inline-flex;
  align-items: center;
  min-height: 40rpx;
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
}
.type-tag {
  flex: none;
  background: $color-bg-page;
  color: $color-text-body;
}
.draft-tag {
  flex: none;
  background: $color-semantic-warning-bg;
  color: $color-text-primary;
}
.event-time {
  flex: none;
  margin-left: auto;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
  font-variant-numeric: tabular-nums;
}
.event-title {
  display: -webkit-box;
  margin-top: $spacing-md;
  overflow: hidden;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
  overflow-wrap: anywhere;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.event-desc {
  display: -webkit-box;
  margin-top: $spacing-xs;
  overflow: hidden;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: anywhere;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.event-meta {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-lg;
  min-width: 0;
}
.event-place {
  display: inline-flex;
  align-items: flex-start;
  gap: $spacing-xs;
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}
.place-icon {
  flex: none;
  margin-top: 4rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}
.place-text {
  min-width: 0;
  overflow-wrap: anywhere;
}
.scope-tag {
  flex: none;
  margin-left: auto;
  border: 2rpx solid $color-border-hairline;
  background: $color-bg-card;
  color: $color-text-body;
}
.event-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  border-top: 2rpx solid $color-border-subtle;
}
.event-action {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 128rpx;
  min-height: 88rpx;
  padding: 0 28rpx;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
}
.event-action:active {
  background: $color-bg-page;
}
.event-action.is-danger:active {
  background: $color-semantic-danger-bg;
}

/* ---------- 加载 / 空 / 错误 ---------- */
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}
.sk-month,
.sk-line,
.sk-date {
  background: $color-border-subtle;
  animation: sk-pulse 1.4s $animation-easing-default infinite;
}
.sk-month {
  width: 156rpx;
  height: 20rpx;
  margin-left: 4rpx;
  border-radius: $border-radius-full;
}
.sk-head {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: $spacing-xs 4rpx;
}
.sk-date {
  flex: none;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
}
.sk-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-sm;
}
.sk-line {
  height: 18rpx;
  border-radius: $border-radius-full;
}
.sk-line.is-title {
  width: 42%;
  height: 24rpx;
}
.sk-line.is-meta {
  width: 22%;
}
.sk-line.is-tag {
  width: 68rpx;
  height: 32rpx;
}
.sk-line.is-short {
  width: 58%;
}
.sk-card {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.state-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 780rpx;
  padding: 56rpx 48rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}
.error-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  margin-bottom: $spacing-md;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}
.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
}
.state-desc {
  max-width: 500rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: anywhere;
}
.state-cta {
  margin-top: $spacing-xl;
}
.btn-primary,
.btn-ghost,
.btn-quiet,
.btn-publish {
  box-sizing: border-box;
}
.btn-primary,
.btn-ghost,
.btn-quiet {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  min-height: 88rpx;
  padding: 0 48rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
}
.btn-primary {
  background: $gradient-brand;
  color: $color-text-on-brand;
  box-shadow: $shadow-md;
}
.btn-ghost {
  border: 2rpx solid $color-brand-primary-light;
  background: $color-brand-primary-bg;
  color: $color-text-primary;
  box-shadow: $shadow-sm;
}
.btn-quiet {
  border: 2rpx solid $color-border-hairline;
  background: $color-bg-card;
  color: $color-text-body;
  box-shadow: $shadow-sm;
}
.btn-primary:active,
.btn-ghost:active,
.btn-quiet:active {
  opacity: 0.86;
}
.btn-primary[aria-disabled="true"],
.btn-ghost[aria-disabled="true"],
.btn-quiet[aria-disabled="true"] {
  opacity: 0.55;
  box-shadow: none;
}

/* 空态插画：日历页 + 轨道，纯 CSS，避免 WXML 内联 SVG */
.empty-art {
  position: relative;
  width: 216rpx;
  height: 216rpx;
  margin-bottom: $spacing-lg;
}
.empty-orbit {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  border: 2rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  opacity: 0.8;
}
.empty-page {
  position: absolute;
  top: 50rpx;
  left: 58rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
  width: 100rpx;
  height: 112rpx;
  padding-bottom: $spacing-md;
  box-sizing: border-box;
  border: 2rpx solid $color-brand-primary-light;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.empty-binding {
  position: absolute;
  top: $spacing-md;
  width: 40rpx;
  height: 6rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-light;
}
.empty-day {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}
.empty-dot {
  position: absolute;
  right: 36rpx;
  bottom: 42rpx;
  width: 16rpx;
  height: 16rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

/* ---------- 底部弹层 ---------- */
.sheet-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 300;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.sheet-layer.is-confirm {
  z-index: 400;
}
.sheet-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba(15, 23, 42, 0.45);
}
.sheet {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 430px;
  max-height: 90vh;
  overflow: hidden;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}
.sheet-grab {
  flex: none;
  width: 72rpx;
  height: 8rpx;
  margin: 14rpx auto 0;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
}
.sheet-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  flex: none;
  padding: $spacing-sm $spacing-lg $spacing-md 36rpx;
}
.sheet-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
}
.sheet-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-xl;
}
.sheet-close:active {
  background: $color-bg-page;
}
.sheet-scroll {
  flex: 1;
  min-height: 0;
  max-height: 60vh;
  padding: 4rpx $spacing-xl $spacing-xl;
  box-sizing: border-box;
}
.edit-note {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  margin-bottom: $spacing-lg;
  padding: 20rpx $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
}
.edit-note-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 36rpx;
  height: 36rpx;
  margin-top: 2rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}
.edit-note-text {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: anywhere;
}
.field {
  min-width: 0;
}
.field + .field,
.field + .field-pair,
.field-pair + .field,
.field-pair {
  margin-top: 28rpx;
}
.field-label {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  margin-bottom: $spacing-sm;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
}
.required-note {
  color: $color-text-body;
  font-weight: $typography-font-weight-medium;
}
.input {
  box-sizing: border-box;
  width: 100%;
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  line-height: 1.5;
}
.textarea {
  min-height: 168rpx;
  padding: 20rpx $spacing-lg;
}
.field-pair {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20rpx;
}
.field-pair .field {
  margin-top: 0;
}
.field-error {
  display: block;
  margin-top: $spacing-sm;
  color: $color-semantic-danger;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}
.input-button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}
.input-button:active {
  background: $color-bg-page;
}
.input-button.is-invalid {
  border-color: $color-semantic-danger;
}
.control-value {
  min-width: 0;
  overflow-wrap: anywhere;
}
.control-value.is-placeholder {
  color: $color-text-muted;
}
.control-icon {
  flex: none;
  font-size: $typography-font-size-md;
  color: $color-text-muted;
}
.type-switch {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}
.type-option {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
}
.type-option.is-selected {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}
.type-option:active {
  opacity: 0.86;
}
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  margin-top: $spacing-xl;
  padding: 20rpx $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}
.switch-copy {
  display: flex;
  flex-direction: column;
  gap: 2rpx;
  min-width: 0;
}
.switch-title {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}
.switch-help {
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
}
.switch {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 112rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}
.switch-track {
  display: flex;
  align-items: center;
  width: 92rpx;
  height: 56rpx;
  padding: 6rpx;
  box-sizing: border-box;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
  transition: background $animation-duration-base $animation-easing-default;
}
.switch-thumb {
  width: 44rpx;
  height: 44rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  transform: translateX(0);
  transition: transform $animation-duration-base $animation-easing-default;
}
.switch.is-on .switch-track {
  background: $color-brand-primary;
}
.switch.is-on .switch-thumb {
  transform: translateX(36rpx);
}
.sheet-footer {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: $spacing-md;
  flex: none;
  padding: 20rpx $spacing-xl calc(20rpx + env(safe-area-inset-bottom));
  border-top: 2rpx solid $color-border-subtle;
  background: $color-bg-card;
}
.sheet-footer.is-edit {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.sheet-footer .btn-primary,
.sheet-footer .btn-ghost,
.sheet-footer .btn-quiet {
  width: 100%;
  padding: 0 $spacing-md;
  box-shadow: none;
}

/* ---------- 变更确认 ---------- */
.confirm-sheet {
  max-height: 80vh;
}
.confirm-head {
  flex: none;
  padding: $spacing-sm $spacing-xl 20rpx;
}
.confirm-sub {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}
.confirm-body {
  min-height: 0;
  padding: 4rpx $spacing-xl $spacing-xl;
  overflow-y: auto;
}
.change-caption {
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
}
.change-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  margin-top: $spacing-sm;
  padding: 20rpx $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}
.change-row {
  display: flex;
  align-items: flex-start;
  gap: $spacing-md;
  min-width: 0;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}
.change-bullet {
  flex: none;
  width: 10rpx;
  height: 10rpx;
  margin-top: 14rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}
.change-label {
  color: $color-text-primary;
  font-weight: $typography-font-weight-medium;
}
.change-value {
  max-width: 54%;
  margin-left: auto;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  text-align: right;
  overflow-wrap: anywhere;
}
.confirm-notify {
  margin-top: $spacing-lg;
}

/* ---------- 页内对话框 ---------- */
.dialog-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 500;
  display: flex;
  align-items: center;
  justify-content: center;
}
.dialog-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba(15, 23, 42, 0.45);
}
.wx-dialog,
.wx-alert {
  position: relative;
  box-sizing: border-box;
  width: calc(100% - 96rpx);
  max-width: 640rpx;
  overflow: hidden;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}
.wx-dialog {
  padding: 40rpx 40rpx 0;
}
.wx-alert {
  padding-top: 44rpx;
}
.wx-dialog-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
  text-align: center;
}
.wx-dialog-hint {
  display: block;
  margin: $spacing-xs 0 $spacing-lg;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  text-align: center;
}
.wx-dialog-input {
  box-sizing: border-box;
  width: 100%;
  min-height: 88rpx;
  margin-bottom: 40rpx;
  padding: 0 $spacing-lg;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  text-align: center;
  font-variant-numeric: tabular-nums;
}
.wx-dialog-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0 -40rpx;
  border-top: 2rpx solid $color-border-subtle;
}
.wx-dialog-button {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 104rpx;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.25;
}
.wx-dialog-button + .wx-dialog-button {
  border-left: 2rpx solid $color-border-subtle;
}
.wx-dialog-button:active {
  background: $color-bg-page;
}
.wx-dialog-button.is-strong {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}
.wx-dialog-button.is-danger {
  color: $color-semantic-danger;
  font-weight: $typography-font-weight-semibold;
}
.wx-dialog-button.is-danger[aria-disabled="true"] {
  opacity: 0.55;
}
.wx-alert-body {
  display: block;
  margin: $spacing-md 40rpx 40rpx;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.5;
  text-align: center;
  overflow-wrap: anywhere;
}
</style>
