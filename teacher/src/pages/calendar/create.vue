<script setup>
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import {
  createTeacherCalendarEvent,
  fetchTeacherCalendarEvent,
  updateTeacherCalendarEvent
} from "@/api/teacher-calendar"
import { fetchDict } from "@/api/dict"
import {
  daysInMonth,
  formatMonthDay,
  formatYmd,
  pad2,
  parseYmd,
  weekdayShort
} from "@/utils/calendar"
import PageShell from "@/components/PageShell.vue"

const today = new Date()

const makeForm = () => ({
  title: "",
  event_type: "activity",
  description: "",
  event_date: "",
  start_time: "09:00",
  end_time: "10:00",
  location: "",
  is_all_day: false,
  notify: true
})

const form = ref(makeForm())
const editId = ref(0)
const original = ref(null)
const loading = ref(false)
const loadError = ref("")
const typeOptions = ref([])
const typesLoading = ref(true)
const typesError = ref("")
const busy = ref(null) // "draft" | "publish"
const done = ref(false)
const touched = ref(false)
const failureText = ref("")
let lastAttempt = null

const confirmVisible = ref(false)
const confirmLines = ref([])
const confirmNotify = ref(true)

const pickerVisible = ref(false)
const pickerKind = ref("date") // date | start | end
const pickerValue = ref([0, 0, 0])
// picker-view 的 indicator-style 只接受 CSS 字符串，无法引用 SCSS 变量；
// rgba(247,243,234) 即 $color-bg-page，仅此一处用于滚轮选中带。
const pickerIndicatorStyle = "height: 88rpx; background-color: rgba(247, 243, 234, 0.9);"

const locked = computed(() => loading.value || Boolean(busy.value) || done.value)
const typeMap = computed(() => {
  const map = {}
  typeOptions.value.forEach((item) => {
    map[item.key] = item
  })
  return map
})
const selectedTypeLabel = computed(
  () => typeMap.value[form.value.event_type]?.label || form.value.event_type || ""
)

const missingTitle = computed(() => !form.value.title.trim())
const missingDate = computed(() => !form.value.event_date)
const invalidTime = computed(() => {
  if (form.value.is_all_day) return false
  const start = form.value.start_time
  const end = form.value.end_time
  if (!start || !end) return false
  return end <= start
})
const showTitleError = computed(() => touched.value && missingTitle.value)
const showDateError = computed(() => touched.value && missingDate.value)
const showTimeError = computed(() => touched.value && invalidTime.value)

// 阻断说明常驻：不看「是否点过发布」，字段一变摘要和按钮状态就跟着变。
const blockedReason = computed(() => {
  if (missingTitle.value && missingDate.value) return "还差标题和日期"
  if (missingTitle.value) return "还差标题"
  if (missingDate.value) return "还差活动日期"
  if (invalidTime.value) return "结束时间需晚于开始时间"
  return ""
})
const publishDisabled = computed(() => locked.value || Boolean(blockedReason.value))
const summaryBlocked = computed(() => !locked.value && Boolean(blockedReason.value))
const summaryText = computed(() => {
  if (loadError.value) return "活动加载失败"
  if (loading.value) return "正在加载…"
  if (!busy.value && blockedReason.value) return blockedReason.value
  if (busy.value === "draft") return "正在保存草稿…"
  if (busy.value === "publish") return editId.value ? "正在保存…" : "正在发布…"
  const when = form.value.event_date
    ? `${formatMonthDay(form.value.event_date)} ${
        form.value.is_all_day
          ? "全天"
          : [form.value.start_time, form.value.end_time].filter(Boolean).join("–") || "未设时间"
      }`
    : "待选择日期"
  const tail = editId.value
    ? changeLines.value.length
      ? "保存时确认通知"
      : "无变更，不推送通知"
    : form.value.notify
    ? "通知家长"
    : "不通知家长"
  return `${selectedTypeLabel.value || "未选类型"} · ${when} · ${tail}`
})
const publishText = computed(() => {
  if (busy.value === "publish") return editId.value ? "保存中…" : "发布中…"
  return editId.value ? "保存发布" : "发布活动"
})

const timeRangeOf = (event) => {
  if (!event) return ""
  if (event.is_all_day) return "全天"
  return [event.start_time, event.end_time].filter(Boolean).join("–") || "未设"
}

// 变更清单；标题/内容/类型没有「新值」可展示，只标已更新（与设计一致）
const changeLines = computed(() => {
  const origin = original.value
  if (!origin) return []
  const lines = []
  if ((origin.event_date || "") !== form.value.event_date) {
    lines.push({ label: "日期", value: form.value.event_date ? formatMonthDay(form.value.event_date) : "未设" })
  }
  if (timeRangeOf(origin) !== timeRangeOf(form.value)) {
    lines.push({ label: "时间", value: timeRangeOf(form.value) })
  }
  if ((origin.location || "") !== (form.value.location || "")) {
    lines.push({ label: "地点", value: form.value.location || "未填写" })
  }
  if ((origin.title || "") !== form.value.title.trim()) lines.push({ label: "标题已更新" })
  if ((origin.description || "") !== (form.value.description || "")) lines.push({ label: "内容已更新" })
  if ((origin.event_type || "") !== form.value.event_type) lines.push({ label: "类型已更新" })
  return lines
})
const substantiveChange = computed(() =>
  changeLines.value.some((line) => ["日期", "时间", "地点"].includes(line.label))
)

/* ---------- 滚轮弹层 ---------- */
const pickerYears = computed(() => {
  const base = today.getFullYear()
  const years = []
  for (let y = base - 3; y <= base + 6; y += 1) years.push(y)
  return years
})
const pickerDayCount = computed(() => {
  const year = pickerYears.value[pickerValue.value[0]] || today.getFullYear()
  const month = (pickerValue.value[1] || 0) + 1
  return daysInMonth(year, month)
})
const pickerColumns = computed(() => {
  if (pickerKind.value !== "date") {
    return [
      Array.from({ length: 24 }, (_, i) => pad2(i)),
      Array.from({ length: 60 }, (_, i) => pad2(i))
    ]
  }
  return [
    pickerYears.value.map(String),
    Array.from({ length: 12 }, (_, i) => String(i + 1)),
    Array.from({ length: pickerDayCount.value }, (_, i) => String(i + 1))
  ]
})
const pickerUnits = computed(() => (pickerKind.value === "date" ? ["年", "月", "日"] : ["时", "分"]))
const pickerTitle = computed(() =>
  pickerKind.value === "date" ? "活动日期" : pickerKind.value === "start" ? "开始时间" : "结束时间"
)

function indexOfYear(year) {
  const idx = pickerYears.value.indexOf(year)
  return idx >= 0 ? idx : pickerYears.value.length - 1
}
function openDatePicker() {
  if (locked.value) return
  uni.hideKeyboard()
  const parts = parseYmd(form.value.event_date)
  const base = parts || { y: today.getFullYear(), m: today.getMonth() + 1, d: today.getDate() }
  pickerKind.value = "date"
  pickerValue.value = [indexOfYear(base.y), base.m - 1, Math.min(base.d - 1, daysInMonth(base.y, base.m) - 1)]
  pickerVisible.value = true
}
function openTimePicker(kind) {
  if (locked.value || form.value.is_all_day) return
  uni.hideKeyboard()
  const raw = kind === "start" ? form.value.start_time : form.value.end_time
  const match = String(raw || "").match(/^(\d{1,2}):(\d{2})$/)
  const h = match ? Math.min(23, Number(match[1])) : 9
  const m = match ? Math.min(59, Number(match[2])) : 0
  pickerKind.value = kind
  pickerValue.value = [h, m]
  pickerVisible.value = true
}
function onPickerChange(e) {
  const next = [...(e.detail?.value || [])]
  if (pickerKind.value === "date") {
    const year = pickerYears.value[next[0]] || today.getFullYear()
    const total = daysInMonth(year, (next[1] || 0) + 1)
    if (next[2] > total - 1) next[2] = total - 1
  }
  pickerValue.value = next
}
function closePicker() {
  pickerVisible.value = false
}
function confirmPicker() {
  if (pickerKind.value === "date") {
    const year = pickerYears.value[pickerValue.value[0]] || today.getFullYear()
    const month = (pickerValue.value[1] || 0) + 1
    const day = Math.min((pickerValue.value[2] || 0) + 1, daysInMonth(year, month))
    form.value.event_date = formatYmd(year, month, day)
  } else {
    const value = `${pad2(pickerValue.value[0] || 0)}:${pad2(pickerValue.value[1] || 0)}`
    if (pickerKind.value === "start") form.value.start_time = value
    else form.value.end_time = value
  }
  pickerVisible.value = false
}

/* ---------- 表单 ---------- */
function onTitleInput(e) {
  form.value.title = e.detail?.value || ""
}
function onDescriptionInput(e) {
  form.value.description = e.detail?.value || ""
}
function onLocationInput(e) {
  form.value.location = e.detail?.value || ""
}
function noop() {}
function selectType(key) {
  if (locked.value) return
  form.value.event_type = key
}
function toggleAllDay() {
  if (locked.value) return
  form.value.is_all_day = !form.value.is_all_day
}
function toggleNotify() {
  if (locked.value) return
  form.value.notify = !form.value.notify
}

async function loadTypes() {
  typesLoading.value = true
  typesError.value = ""
  try {
    const res = await fetchDict("calendar_event_type")
    const items = (res.data?.items || []).map((item) => ({
      key: item.item_key,
      label: item.item_label || item.item_key
    }))
    typeOptions.value = items
    // 新建才回落到字典第一项；编辑保留事件原类型，避免把已停用类型的旧事件静默改类型
    if (!editId.value && items.length && !items.some((item) => item.key === form.value.event_type)) {
      form.value.event_type = items[0].key
    }
  } catch {
    typesError.value = "类型加载失败"
  } finally {
    typesLoading.value = false
    ensureSelectedType()
  }
}

// 编辑的事件类型可能已被字典停用：仍把它作为一项展示，否则选中的胶囊会消失
function ensureSelectedType() {
  const key = form.value.event_type
  if (!key || typeOptions.value.some((item) => item.key === key)) return
  if (!editId.value) return
  typeOptions.value = [
    ...typeOptions.value,
    { key, label: original.value?.event_type_label || key }
  ]
}

async function loadEditEvent() {
  if (!editId.value) return
  loading.value = true
  loadError.value = ""
  try {
    const res = await fetchTeacherCalendarEvent(editId.value, { silent: true })
    const event = res.data?.event
    if (!event) {
      loadError.value = "活动加载失败"
      return
    }
    original.value = event
    form.value = {
      title: event.title || "",
      event_type: event.event_type || "activity",
      description: event.description || "",
      event_date: event.event_date || "",
      // 编辑保留原值：用 09:00/10:00 填充会把「原本没有时刻」误算成一次时间变更
      start_time: event.start_time || "",
      end_time: event.end_time || "",
      location: event.location || "",
      is_all_day: !!event.is_all_day,
      notify: false
    }
    ensureSelectedType()
  } catch {
    loadError.value = "活动加载失败"
  } finally {
    loading.value = false
  }
}

function buildPayload(status, notify) {
  return {
    title: form.value.title.trim(),
    event_date: form.value.event_date,
    event_type: form.value.event_type,
    description: form.value.description,
    start_time: form.value.is_all_day ? "" : form.value.start_time,
    end_time: form.value.is_all_day ? "" : form.value.end_time,
    location: form.value.location,
    is_all_day: form.value.is_all_day,
    status,
    notify
  }
}

async function persist(status, notify) {
  if (locked.value || done.value) return
  busy.value = status === "draft" ? "draft" : "publish"
  failureText.value = ""
  lastAttempt = { status, notify }
  try {
    const payload = buildPayload(status, notify)
    if (editId.value) await updateTeacherCalendarEvent(editId.value, payload, { silent: true })
    else await createTeacherCalendarEvent(payload, { silent: true })
    done.value = true
    uni.showToast({
      title: status === "draft" ? "已存草稿" : editId.value ? "已保存" : "发布成功",
      icon: "success"
    })
    setTimeout(() => uni.navigateBack(), 900)
  } catch (err) {
    const prefix = status === "draft" ? "保存失败" : "发布失败"
    failureText.value = `${prefix}：${err?.message || "网络异常"}，请重试`
  } finally {
    busy.value = null
  }
}

function submitDraft() {
  if (locked.value || done.value) return
  touched.value = true
  if (missingTitle.value || missingDate.value) return
  persist("draft", false)
}

function submitPublish() {
  if (locked.value || done.value) return
  touched.value = true
  if (blockedReason.value) return
  // 编辑已发布活动：无变更静默保存；有变更时先确认是否通知家长（日期/时间/地点预置开）
  if (editId.value && original.value?.status === "published") {
    if (!changeLines.value.length) {
      persist("published", false)
      return
    }
    confirmLines.value = changeLines.value
    confirmNotify.value = substantiveChange.value
    confirmVisible.value = true
    return
  }
  persist("published", form.value.notify)
}

function retrySubmit() {
  if (!lastAttempt || locked.value) return
  persist(lastAttempt.status, lastAttempt.notify)
}

function cancelConfirm() {
  confirmVisible.value = false
}
function saveConfirm() {
  if (busy.value) return
  confirmVisible.value = false
  persist("published", confirmNotify.value)
}

onLoad((options) => {
  const id = Number(options?.id || 0)
  editId.value = id
  if (id) {
    uni.setNavigationBarTitle({ title: "编辑活动" })
    loadEditEvent()
  }
  loadTypes()
})
</script>

<template>
  <PageShell bottom-space="320rpx" max-width="430px">
    <view class="event-form" :class="{ 'is-busy': locked }">
      <view v-if="loadError" class="error-block">
        <view class="error-icon"><text class="icon">&#xf2c2;</text></view>
        <text class="error-title">活动加载失败</text>
        <text class="error-desc">网络异常，未获取到活动详情。</text>
        <view class="error-btn" @click="loadEditEvent"><text>重新加载</text></view>
      </view>

      <template v-else>
        <!-- 基本信息 -->
        <text class="group-title">基本信息</text>
        <view class="event-card">
          <view v-if="loading" class="card-skeleton">
            <view class="sk sk-label" />
            <view class="sk sk-input" />
            <view class="sk sk-label is-short" />
            <view class="sk-pills">
              <view v-for="i in 5" :key="i" class="sk sk-pill" />
            </view>
            <view class="sk sk-label" />
            <view class="sk sk-textarea" />
          </view>
          <view v-else class="card-live is-pad">
            <view class="field">
              <view class="field-label">
                <text class="label-text">活动标题</text>
                <text class="req">*</text>
              </view>
              <input
                class="event-input is-title"
                :class="{ 'is-invalid': showTitleError }"
                :value="form.title"
                :disabled="locked"
                maxlength="60"
                placeholder="请输入活动标题"
                placeholder-style="color: #94A3B8"
                @input="onTitleInput"
              />
              <view v-if="showTitleError" class="field-error">
                <text class="icon">&#xeca1;</text>
                <text>请输入活动标题</text>
              </view>
            </view>

            <view class="field">
              <view class="field-label">
                <text class="label-text">活动类型</text>
                <text class="req">*</text>
              </view>
              <view v-if="typesLoading" class="sk-pills">
                <view v-for="i in 5" :key="i" class="sk sk-pill" />
              </view>
              <view v-else-if="typesError" class="type-error">
                <text class="type-error-text">{{ typesError }}</text>
                <text class="type-error-retry" @click="loadTypes">重试</text>
              </view>
              <view v-else class="type-pills">
                <view
                  v-for="option in typeOptions"
                  :key="option.key"
                  class="type-pill"
                  :class="{ 'is-active': form.event_type === option.key, 'is-disabled': locked }"
                  @click="selectType(option.key)"
                ><text>{{ option.label }}</text></view>
              </view>
            </view>

            <view class="field">
              <view class="field-label"><text class="label-text">活动描述</text></view>
              <textarea
                class="event-input is-textarea"
                :value="form.description"
                :disabled="locked"
                maxlength="500"
                auto-height
                placeholder="请输入活动描述，如活动内容、注意事项等"
                placeholder-style="color: #94A3B8"
                @input="onDescriptionInput"
              />
            </view>
          </view>
        </view>

        <!-- 时间地点 -->
        <text class="group-title">时间地点</text>
        <view class="event-card">
          <view v-if="loading" class="card-skeleton">
            <view v-for="i in 3" :key="i" class="sk-row">
              <view class="sk sk-line is-label" />
              <view class="sk sk-line is-value" />
            </view>
            <view class="sk-row"><view class="sk sk-switch" /></view>
            <view class="sk sk-input is-last" />
          </view>
          <view v-else class="card-live">
            <view
              class="event-cell"
              :class="{ 'is-locked': locked, 'is-invalid': showDateError }"
              @click="openDatePicker"
            >
              <view class="cell-label">
                <text>活动日期</text>
                <text class="req">*</text>
              </view>
              <view class="cell-trailing">
                <text class="cell-value" :class="{ 'is-placeholder': !form.event_date }">
                  {{ form.event_date || "请选择日期" }}
                </text>
                <text v-if="form.event_date" class="cell-badge">{{ weekdayShort(form.event_date) }}</text>
                <text class="icon cell-chevron">&#xea6e;</text>
              </view>
            </view>
            <view v-if="showDateError" class="field-error is-cell">
              <text class="icon">&#xeca1;</text>
              <text>请选择活动日期</text>
            </view>

            <view
              class="event-cell is-divided"
              :class="{ 'is-locked': locked || form.is_all_day }"
              @click="openTimePicker('start')"
            >
              <text class="cell-label">开始时间</text>
              <view class="cell-trailing">
                <text class="cell-value">{{ form.is_all_day ? "全天" : form.start_time || "未设置" }}</text>
                <text class="icon cell-chevron">&#xea6e;</text>
              </view>
            </view>

            <view
              class="event-cell is-divided"
              :class="{ 'is-locked': locked || form.is_all_day, 'is-invalid': showTimeError }"
              @click="openTimePicker('end')"
            >
              <text class="cell-label">结束时间</text>
              <view class="cell-trailing">
                <text class="cell-value">{{ form.is_all_day ? "全天" : form.end_time || "未设置" }}</text>
                <text class="icon cell-chevron">&#xea6e;</text>
              </view>
            </view>
            <view v-if="showTimeError" class="field-error is-cell">
              <text class="icon">&#xeca1;</text>
              <text>结束时间需晚于开始时间</text>
            </view>

            <view class="switch-row is-divided" @click="toggleAllDay">
              <view class="switch-copy">
                <text class="switch-title">全天活动</text>
                <text class="switch-hint">不设具体时刻（如假期）</text>
              </view>
              <view class="switch-control" :class="{ 'is-on': form.is_all_day }" aria-label="全天活动">
                <view class="switch-track" />
              </view>
            </view>

            <view class="field is-inset">
              <view class="field-label"><text class="label-text">活动地点</text></view>
              <input
                class="event-input"
                :value="form.location"
                :disabled="locked"
                maxlength="60"
                placeholder="请输入活动地点"
                placeholder-style="color: #94A3B8"
                @input="onLocationInput"
              />
            </view>
          </view>
        </view>

        <!-- 发布设置 -->
        <text class="group-title">发布设置</text>
        <view class="event-card">
          <view v-if="loading" class="card-skeleton">
            <view class="sk-row"><view class="sk sk-switch" /></view>
            <view class="sk sk-note" />
          </view>
          <view v-else class="card-live">
            <view v-if="editId && original && original.status === 'published'" class="card-note is-solo">
              <text class="icon note-icon">&#xee59;</text>
              <text>该活动已发布——保存时将让你确认是否通知家长（按变更内容预置建议）。</text>
            </view>
            <view v-else class="switch-row" @click="toggleNotify">
              <view class="switch-copy">
                <text class="switch-title">通知家长</text>
                <text class="switch-hint">发布后向本班家长推送通知（可在通知模块查看已读）</text>
              </view>
              <view class="switch-control" :class="{ 'is-on': form.notify }" aria-label="通知家长">
                <view class="switch-track" />
              </view>
            </view>
            <view class="card-note">
              <text class="icon note-icon">&#xee59;</text>
              <text>存为草稿不会通知家长</text>
            </view>
          </view>
        </view>
      </template>
    </view>

    <template #bottom>
      <view class="submit-dock">
        <view v-if="failureText" class="failure-bar">
          <text class="icon failure-icon">&#xeca1;</text>
          <text class="failure-text">{{ failureText }}</text>
          <view class="failure-retry" @click="retrySubmit"><text>重试</text></view>
        </view>
        <text
          class="submit-summary"
          :class="{ 'is-blocked': summaryBlocked, 'is-busy': Boolean(loading || busy) }"
        >{{ summaryText }}</text>
        <view class="submit-actions">
          <view
            class="btn-draft"
            :class="{ 'is-loading': busy === 'draft', 'is-disabled': locked || Boolean(loadError) }"
            :aria-disabled="locked || Boolean(loadError)"
            @click="submitDraft"
          >
            <view v-if="busy === 'draft'" class="spinner" />
            <text class="btn-text">{{ busy === "draft" ? "保存中…" : "存草稿" }}</text>
          </view>
          <view
            class="btn-publish"
            :class="{ 'is-loading': busy === 'publish', 'is-disabled': publishDisabled }"
            :aria-disabled="publishDisabled"
            @click="submitPublish"
          >
            <view v-if="busy === 'publish'" class="spinner" />
            <text class="btn-text">{{ publishText }}</text>
          </view>
        </view>
      </view>
    </template>

    <!-- 日期 / 时间滚轮弹层 -->
    <view v-if="pickerVisible" class="sheet-layer">
      <view class="sheet-scrim" @click="closePicker" @touchmove.stop.prevent="noop" />
      <view class="sheet picker-sheet">
        <view class="sheet-grab" />
        <view class="picker-head">
          <view class="picker-btn picker-cancel" @click="closePicker"><text>取消</text></view>
          <text class="picker-title">{{ pickerTitle }}</text>
          <view class="picker-btn picker-done" @click="confirmPicker"><text>完成</text></view>
        </view>
        <picker-view
          class="wheel"
          :value="pickerValue"
          :indicator-style="pickerIndicatorStyle"
          @change="onPickerChange"
        >
          <picker-view-column v-for="(column, ci) in pickerColumns" :key="ci">
            <view v-for="(item, ii) in column" :key="ii" class="wheel-item">
              <text>{{ item }}</text>
              <text class="wheel-unit">{{ pickerUnits[ci] }}</text>
            </view>
          </picker-view-column>
        </picker-view>
      </view>
    </view>

    <!-- 编辑已发布活动：保存前的通知确认 -->
    <view v-if="confirmVisible" class="sheet-layer">
      <view class="sheet-scrim" @click="cancelConfirm" @touchmove.stop.prevent="noop" />
      <view class="sheet confirm-sheet">
        <view class="sheet-grab" />
        <view class="confirm-head">
          <text class="confirm-title">通知家长？</text>
          <text class="confirm-sub">将通知本班全体家长</text>
        </view>
        <text class="change-caption">本次变更</text>
        <view class="change-list">
          <view v-for="(line, i) in confirmLines" :key="i" class="change-row">
            <view class="change-bullet" />
            <text class="change-label">{{ line.label }}</text>
            <text v-if="line.value" class="change-value">{{ line.value }}</text>
          </view>
        </view>
        <view class="switch-row confirm-notify" @click="confirmNotify = !confirmNotify">
          <view class="switch-copy">
            <text class="switch-title">发布变更通知</text>
            <text class="switch-hint">家长端会在日历顶部看到此变更</text>
          </view>
          <view class="switch-control" :class="{ 'is-on': confirmNotify }" aria-label="发布变更通知">
            <view class="switch-track" />
          </view>
        </view>
        <view class="confirm-actions">
          <view class="btn-draft" @click="cancelConfirm"><text class="btn-text">取消</text></view>
          <view class="btn-publish" :class="{ 'is-loading': busy === 'publish' }" @click="saveConfirm">
            <view v-if="busy === 'publish'" class="spinner" />
            <text class="btn-text">保存</text>
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

.event-form {
  width: 100%;
  padding-bottom: $spacing-xl;
}
.event-form.is-busy {
  opacity: 0.72;
}
.group-title {
  display: block;
  margin: 0 8rpx 16rpx;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  line-height: 1.3;
  letter-spacing: 0.02em;
  color: $color-text-muted;
}
.event-card {
  margin-bottom: 36rpx;
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.card-live.is-pad {
  display: flex;
  flex-direction: column;
  gap: 28rpx;
  padding: 28rpx;
}
.field-label {
  display: flex;
  align-items: center;
  margin-bottom: 12rpx;
}
.label-text {
  font-size: $typography-font-size-base;
  font-weight: 500;
  line-height: 1.3;
  color: $color-text-body;
}
.req {
  margin-left: 6rpx;
  font-size: $typography-font-size-base;
  color: $color-semantic-danger;
}
.event-input {
  box-sizing: border-box;
  display: block;
  width: 100%;
  height: 88rpx;
  padding: 0 24rpx;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  font-size: $typography-font-size-lg;
  line-height: 1.4;
  color: $color-text-primary;
}
.event-input.is-title {
  font-weight: 500;
}
.event-input.is-invalid {
  border-color: $color-semantic-danger;
}
.event-input.is-textarea {
  height: auto;
  min-height: 184rpx;
  padding: 20rpx 24rpx;
  line-height: 1.5;
}
.type-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 14rpx;
}
.type-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 112rpx;
  min-height: 88rpx;
  padding: 0 28rpx;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  font-size: $typography-font-size-base;
  line-height: 1;
  color: $color-text-body;
}
.type-pill.is-active {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
  font-weight: 600;
  color: $color-text-primary;
}
.type-pill.is-disabled {
  opacity: 0.7;
}
.type-error {
  display: flex;
  align-items: center;
  gap: 14rpx;
  padding: 20rpx 24rpx;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
}
.type-error-text {
  flex: 1;
  font-size: $typography-font-size-sm;
  color: $color-semantic-danger-strong;
}
.type-error-retry {
  flex: none;
  font-size: $typography-font-size-sm;
  font-weight: 600;
  color: $color-semantic-danger-strong;
}
.field-error {
  display: flex;
  align-items: flex-start;
  gap: 10rpx;
  margin-top: 12rpx;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-semantic-danger-strong;
}
.field-error .icon {
  margin-top: 2rpx;
  font-size: $typography-font-size-sm;
}
.field-error.is-cell {
  margin: 0;
  padding: 0 28rpx 20rpx;
  background: $color-bg-page;
}

.event-cell {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  width: 100%;
  min-height: 104rpx;
  padding: 14rpx 28rpx;
  background: transparent;
  text-align: left;
}
.event-cell.is-divided {
  border-top: 2rpx solid $color-bg-page;
}
.event-cell:active {
  background: $color-bg-page;
}
.event-cell.is-locked {
  background: $color-bg-page;
}
.cell-label {
  flex: 0 1 auto;
  min-width: 0;
  font-size: $typography-font-size-md;
  font-weight: 500;
  line-height: 1.3;
  color: $color-text-body;
}
.cell-trailing {
  display: flex;
  flex: 0 1 auto;
  align-items: center;
  justify-content: flex-end;
  gap: 10rpx;
  min-width: 0;
}
.cell-value {
  min-width: 0;
  font-size: $typography-font-size-lg;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  line-height: 1.3;
  text-align: right;
  color: $color-text-primary;
  overflow-wrap: anywhere;
}
.cell-value.is-placeholder {
  color: $color-text-hint;
  font-weight: 400;
}
.cell-badge {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  min-height: 40rpx;
  padding: 0 12rpx;
  border-radius: $border-radius-sm;
  background: $color-brand-primary-bg;
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1;
  color: $color-brand-primary-dark;
}
.cell-chevron {
  font-size: $typography-font-size-base;
  color: $color-text-hint;
}
.event-cell.is-locked .cell-label,
.event-cell.is-locked .cell-value {
  color: $color-text-hint;
}
.event-cell.is-locked .cell-chevron {
  color: $color-border-subtle;
}
.event-cell.is-invalid .cell-value {
  color: $color-semantic-danger;
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  width: 100%;
  min-height: 112rpx;
  padding: 20rpx 28rpx;
}
.switch-row.is-divided {
  border-top: 2rpx solid $color-bg-page;
}
.switch-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}
.switch-title {
  font-size: $typography-font-size-lg;
  font-weight: 500;
  line-height: 1.3;
  color: $color-text-primary;
}
.switch-hint {
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-text-muted;
}
.switch-control {
  position: relative;
  display: flex;
  flex: none;
  align-items: center;
  width: 96rpx;
  height: 56rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
  transition: background 0.2s;
}
.switch-control.is-on {
  background: $color-brand-primary;
}
.switch-track {
  position: absolute;
  top: 8rpx;
  left: 8rpx;
  width: 40rpx;
  height: 40rpx;
  border-radius: $border-radius-full;
  background: $color-text-on-brand;
  transition: transform 0.2s;
}
.switch-control.is-on .switch-track {
  transform: translateX(40rpx);
}
.field.is-inset {
  padding: 20rpx 28rpx 32rpx;
  border-top: 2rpx solid $color-bg-page;
}
.card-note {
  display: flex;
  align-items: flex-start;
  gap: 10rpx;
  padding: 22rpx 28rpx;
  border-top: 2rpx solid $color-bg-page;
  background: $color-bg-page;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-text-muted;
}
.card-note.is-solo {
  border-top: 0;
}
.note-icon {
  flex: none;
  margin-top: 4rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

/* ---------- 固定提交栏 ---------- */
.submit-dock {
  width: 100%;
  padding: 20rpx 28rpx calc(24rpx + env(safe-area-inset-bottom));
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}
.failure-bar {
  display: flex;
  align-items: center;
  gap: 14rpx;
  width: 100%;
  min-height: 104rpx;
  margin-bottom: 16rpx;
  padding: 6rpx 12rpx 6rpx 24rpx;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
  font-size: $typography-font-size-sm;
  line-height: 1.45;
  color: $color-semantic-danger-strong;
}
.failure-icon {
  flex: none;
  font-size: $typography-font-size-base;
}
.failure-text {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}
.failure-retry {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  min-width: 112rpx;
  min-height: 88rpx;
  padding: 0 24rpx;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  font-size: $typography-font-size-base;
  font-weight: 600;
  color: $color-text-primary;
  box-shadow: $shadow-sm;
}
.submit-summary {
  display: block;
  margin-bottom: 14rpx;
  padding: 0 4rpx;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  text-align: center;
  color: $color-text-muted;
  overflow-wrap: anywhere;
}
.submit-summary.is-blocked {
  font-weight: 600;
  color: $color-text-primary;
}
.submit-summary.is-busy {
  color: $color-text-hint;
}
.submit-actions {
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.6fr);
  gap: 20rpx;
  width: 100%;
}
.btn-draft,
.btn-publish {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  min-width: 0;
  min-height: 96rpx;
  padding: 0 24rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-lg;
  font-weight: 600;
  line-height: 1.2;
}
.btn-text {
  min-width: 0;
  overflow-wrap: anywhere;
}
.btn-draft {
  border: 2rpx solid $color-border-hairline;
  background: $color-bg-page;
  color: $color-text-primary;
  box-shadow: $shadow-sm;
}
.btn-publish {
  background: $gradient-brand;
  color: $color-text-on-brand;
  box-shadow: $shadow-md;
}
.btn-draft:active,
.btn-publish:active {
  transform: scale(0.97);
}
.btn-draft.is-disabled,
.btn-publish.is-disabled {
  transform: none;
  box-shadow: none;
}
.btn-draft.is-disabled {
  background: $color-bg-page;
  color: $color-text-hint;
}
.btn-publish.is-disabled {
  background: $color-bg-page;
  color: $color-text-muted;
}
.btn-draft.is-loading,
.btn-publish.is-loading {
  opacity: 1;
}
.spinner {
  width: 28rpx;
  height: 28rpx;
  border: 4rpx solid currentColor;
  border-top-color: transparent;
  border-radius: $border-radius-full;
  animation: spin 0.7s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* ---------- 弹层（滚轮 / 通知确认） ---------- */
/* 遮罩色是刻意的一次性值：design token 里没有 overlay 语义，全端仅此几处使用。 */
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
  width: 100%;
  max-width: 430px;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}
.sheet-grab {
  display: block;
  width: 72rpx;
  height: 8rpx;
  margin: 14rpx auto 0;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
}
.picker-sheet {
  padding: 0 0 calc(16rpx + env(safe-area-inset-bottom));
}
.picker-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  align-items: center;
  min-height: 104rpx;
  padding: 0 20rpx;
  border-bottom: 2rpx solid $color-bg-page;
}
.picker-btn {
  display: flex;
  align-items: center;
  min-width: 104rpx;
  min-height: 88rpx;
  padding: 0 16rpx;
  border-radius: $border-radius-lg;
}
.picker-cancel {
  justify-self: start;
  font-size: $typography-font-size-md;
  color: $color-text-muted;
}
.picker-done {
  justify-self: end;
  justify-content: flex-end;
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
}
.picker-title {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  line-height: 1.3;
  text-align: center;
  color: $color-text-primary;
}
.wheel {
  width: 100%;
  height: 440rpx;
}
.wheel-item {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  height: 88rpx;
  font-size: $typography-font-size-2xl;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  line-height: 1;
  color: $color-text-primary;
}
.wheel-unit {
  font-size: $typography-font-size-sm;
  font-weight: 400;
  color: $color-text-muted;
}

.confirm-sheet {
  max-height: 80vh;
  padding: 0 0 calc(24rpx + env(safe-area-inset-bottom));
}
.confirm-head {
  padding: 6rpx 32rpx 20rpx;
}
.confirm-title {
  display: block;
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  line-height: 1.3;
  color: $color-text-primary;
}
.confirm-sub {
  display: block;
  margin-top: 4rpx;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  color: $color-text-muted;
}
.change-caption {
  display: block;
  margin: 0 32rpx;
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1.3;
  letter-spacing: 0.02em;
  color: $color-text-body;
}
.change-list {
  display: flex;
  flex-direction: column;
  gap: 10rpx;
  margin: 14rpx 32rpx 0;
  padding: 20rpx 24rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}
.change-row {
  display: flex;
  align-items: flex-start;
  gap: 14rpx;
  min-width: 0;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}
.change-bullet {
  flex: none;
  width: 10rpx;
  height: 10rpx;
  margin-top: 16rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}
.change-label {
  color: $color-text-primary;
  font-weight: 500;
}
.change-value {
  max-width: 54%;
  margin-left: auto;
  font-size: $typography-font-size-sm;
  text-align: right;
  color: $color-text-body;
  overflow-wrap: anywhere;
}
.confirm-notify {
  margin-top: 24rpx;
  border-top: 2rpx solid $color-bg-page;
}
.confirm-actions {
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.6fr);
  gap: 20rpx;
  margin: 28rpx 32rpx 0;
}

/* ---------- 骨架 ---------- */
.sk {
  display: block;
  border-radius: $border-radius-md;
  background: $color-bg-page;
  animation: sk-pulse 1.1s ease-in-out infinite alternate;
}
.sk-label {
  width: 144rpx;
  height: 20rpx;
  margin-bottom: 16rpx;
}
.sk-label.is-short {
  width: 104rpx;
}
.sk-input {
  width: 100%;
  height: 88rpx;
  margin-bottom: 28rpx;
  border-radius: $border-radius-lg;
}
.sk-input.is-last {
  margin: 20rpx 0 0;
}
.sk-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 14rpx;
  margin-bottom: 28rpx;
}
.sk-pill {
  width: 116rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
}
.sk-textarea {
  width: 100%;
  height: 168rpx;
  border-radius: $border-radius-lg;
}
.sk-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  height: 104rpx;
  border-bottom: 2rpx solid $color-bg-page;
}
.sk-row:last-child {
  border-bottom: 0;
}
.sk-line.is-label {
  width: 128rpx;
  height: 20rpx;
}
.sk-line.is-value {
  width: 176rpx;
  height: 24rpx;
}
.sk-switch {
  width: 96rpx;
  height: 56rpx;
  margin-left: auto;
  border-radius: $border-radius-full;
}
.sk-note {
  width: 320rpx;
  height: 20rpx;
  margin: 28rpx;
}
@keyframes sk-pulse {
  from {
    opacity: 0.55;
  }
  to {
    opacity: 1;
  }
}

/* ---------- 加载失败 ---------- */
.error-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14rpx;
  padding: 56rpx 40rpx 52rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}
.error-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 104rpx;
  height: 104rpx;
  margin-bottom: 4rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-3xl;
}
.error-title {
  font-size: $typography-font-size-md;
  font-weight: 500;
  color: $color-text-primary;
}
.error-desc {
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-text-muted;
}
.error-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: 8rpx;
  padding: 0 44rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-md;
  font-weight: 500;
}
</style>
