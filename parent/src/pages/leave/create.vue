<script setup>
/**
 * 请假申请 · 一张安静的表单
 *
 * 提交按钮固定在底栏：未填齐时按钮呈禁用外观，但点击只做就地校验
 * （三处红字 + 底栏「还需填写」），不发请求也不弹 toast。
 * 请假类型不作为阻断项：后端 leave_type 缺省落 other。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { createLeave } from "@/api/parent-leave"
import { getToken } from "@/utils/auth"
import config from "@/config/env"
import {
  formatMonthTitle,
  formatWeekday,
  formatYmd,
  inclusiveDays,
  parseDateOnly,
  todayKey
} from "@/utils/time"
import { LEAVE_TYPES } from "@/utils/leave"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG。
const ICON = {
  check: "", // check-line
  chevron: "", // arrow-right-s-line
  chevronLeft: "", // arrow-left-s-line
  close: "", // close-line
  add: "", // add-line
  alert: "" // error-warning-line
}

const WEEKDAYS = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"]
const MAX_PHOTOS = 3
const MAX_PHOTO_BYTES = 20 * 1024 * 1024

const todayStr = todayKey()

const loading = ref(true)
const loadError = ref("")
const child = ref({ id: 0, name: "", class_name: "" })
const form = ref({
  student_id: 0,
  leave_type: "",
  reason: "",
  start_date: "",
  end_date: "",
  emergency_contact: { name: "", phone: "" }
})
const photos = ref([])
const attempted = ref(false)
const submitting = ref(false)
const submitted = ref(false)
const submitFailed = ref(false)

let photoSeq = 0

const missing = computed(() => {
  const list = []
  if (!form.value.start_date) list.push("开始日期")
  if (!form.value.end_date) list.push("结束日期")
  if (!form.value.reason.trim()) list.push("请假原因")
  return list
})
const fieldErrors = computed(() => ({
  start: !form.value.start_date ? "请选择开始日期" : "",
  end: !form.value.end_date ? "请选择结束日期" : "",
  reason: !form.value.reason.trim() ? "请填写请假原因" : ""
}))
const showFieldErrors = computed(() => attempted.value && !loading.value && !loadError.value)
const showBlockingHint = computed(
  () => showFieldErrors.value && missing.value.length > 0
)
const days = computed(() => inclusiveDays(form.value.start_date, form.value.end_date))
const reasonLength = computed(() => Array.from(form.value.reason || "").length)
const childInitial = computed(() => {
  const name = String(child.value.name || "")
  return name ? Array.from(name)[0] : "?"
})
const canAddPhoto = computed(
  () => !submitting.value && !loading.value && photos.value.length < MAX_PHOTOS
)
const canSubmit = computed(
  () =>
    !loading.value &&
    !loadError.value &&
    !submitting.value &&
    !submitted.value &&
    missing.value.length === 0
)
const dockSpace = computed(() => (submitFailed.value ? "340rpx" : "240rpx"))

const dockHint = computed(() => {
  if (loading.value) return "正在加载…"
  if (loadError.value) return "加载失败，请重试"
  if (submitting.value) return "正在提交，请稍候"
  if (showBlockingHint.value) return `还需填写：${missing.value.join("、")}`
  if (!form.value.start_date || !form.value.end_date) return "请选择请假时间 · 提交后进入待审批"
  return `共 ${days.value} 天 · 提交后进入待审批`
})

const daysTotalText = computed(() => (days.value > 0 ? `共 ${days.value} 天` : "共 — 天"))
const daysNote = computed(() => (days.value > 0 ? "含首尾两天" : "选择日期后自动计算"))
const startWeekday = computed(() => formatWeekday(form.value.start_date))
const endWeekday = computed(() => formatWeekday(form.value.end_date))

// ---- 日期弹层 -------------------------------------------------
const sheetOpen = ref(false)
const sheetField = ref("start")
const sheetYear = ref(0)
const sheetMonth = ref(0)
const sheetPending = ref("")

const sheetTitle = computed(() => (sheetField.value === "start" ? "开始日期" : "结束日期"))
const sheetMonthLabel = computed(() => formatMonthTitle(sheetYear.value, sheetMonth.value))
// 结束日期下限：开始日期（若已选且晚于今天）；开始日期下限：今天。
const sheetMinDate = computed(() =>
  sheetField.value === "end" ? form.value.start_date || todayStr : todayStr
)
const sheetConfirmDisabled = computed(
  () => !sheetPending.value || sheetPending.value < sheetMinDate.value
)

const prevMonthDisabled = computed(() => {
  const now = new Date()
  return (
    sheetYear.value < now.getFullYear() ||
    (sheetYear.value === now.getFullYear() && sheetMonth.value <= now.getMonth())
  )
})

const calendarCells = computed(() => {
  const year = sheetYear.value
  const month = sheetMonth.value
  const leading = (new Date(year, month, 1).getDay() + 6) % 7
  const total = new Date(year, month + 1, 0).getDate()
  const cells = []
  for (let i = 0; i < leading; i += 1) cells.push({ key: `blank-${i}`, blank: true })
  const minDate = sheetMinDate.value
  for (let day = 1; day <= total; day += 1) {
    const dateStr = formatYmd(new Date(year, month, day))
    cells.push({
      key: dateStr,
      blank: false,
      day,
      dateStr,
      disabled: dateStr < minDate,
      selected: dateStr === sheetPending.value,
      today: dateStr === todayStr
    })
  }
  return cells
})

function selectType(key) {
  if (submitting.value || loading.value) return
  form.value.leave_type = key
}

function openSheet(field) {
  if (submitting.value || loading.value || loadError.value) return
  const base =
    field === "start"
      ? form.value.start_date
      : form.value.end_date || form.value.start_date
  const anchor = (base && parseDateOnly(base)) || new Date()
  sheetField.value = field
  sheetYear.value = anchor.getFullYear()
  sheetMonth.value = anchor.getMonth()
  sheetPending.value = base || ""
  sheetOpen.value = true
}

function closeSheet() {
  sheetOpen.value = false
}

function moveMonth(delta) {
  if (delta < 0 && prevMonthDisabled.value) return
  const next = new Date(sheetYear.value, sheetMonth.value + delta, 1)
  sheetYear.value = next.getFullYear()
  sheetMonth.value = next.getMonth()
  // 翻月不清空待选日期：翻走再翻回时选中态与「确定」都要保留（审查 P2-1）。
}

function pickDate(cell) {
  if (!cell || cell.blank || cell.disabled) return
  sheetPending.value = cell.dateStr
}

function confirmSheet() {
  if (sheetConfirmDisabled.value) return
  const picked = sheetPending.value
  if (sheetField.value === "start") {
    form.value.start_date = picked
    // 结束日期不得早于开始日期：开始后移时结束自动跟随。
    if (form.value.end_date && form.value.end_date < picked) {
      form.value.end_date = picked
    }
  } else {
    form.value.end_date = picked
  }
  sheetOpen.value = false
}

// ---- 孩子信息 -------------------------------------------------
async function loadChild() {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const res = await fetchParentHomeOverview(getCurrentStudentId() || undefined)
    syncCurrentStudentFromOverview(res.data)
    const stu = res.data?.current_student
    if (!stu?.id) throw new Error("未获取到孩子信息")
    child.value = {
      id: stu.id,
      name: stu.name || "",
      class_name: stu.class_name || ""
    }
    form.value.student_id = stu.id
  } catch (err) {
    console.warn("[leave-create] load child failed:", err?.message || err)
    loadError.value = "孩子信息加载失败"
  } finally {
    loading.value = false
  }
}

// ---- 附件上传 -------------------------------------------------
function photoName(file, index) {
  const raw = String(file?.name || "").trim()
  if (raw) return raw
  return `照片 ${index + 1}`
}

async function runUpload(key) {
  const photo = photos.value.find(p => p.key === key)
  if (!photo || !photo.localPath) return
  photos.value = photos.value.map(p => (p.key === key ? { ...p, status: "uploading" } : p))
  uni.showLoading({ title: "上传中", mask: true })
  await new Promise(resolve => {
    uni.uploadFile({
      url: config.uploadURL,
      filePath: photo.localPath,
      name: "file",
      formData: { biz_type: "leave" },
      header: { Authorization: `Bearer ${getToken()}` },
      success: up => {
        let body = {}
        try {
          body = JSON.parse(up.data || "{}")
        } catch (e) {
          body = {}
        }
        if (up.statusCode >= 200 && up.statusCode < 300 && body.code === 0 && body.data?.file_url) {
          const data = body.data
          photos.value = photos.value.map(p =>
            p.key === key
              ? {
                  ...p,
                  status: "done",
                  file_url: data.file_url,
                  file_name: data.file_name || p.file_name,
                  file_size: Number(data.file_size) || p.file_size
                }
              : p
          )
          uni.showToast({ title: "照片已上传", icon: "success" })
        } else {
          photos.value = photos.value.map(p => (p.key === key ? { ...p, status: "failed" } : p))
          uni.showToast({ title: body.message || "照片上传失败", icon: "none" })
        }
      },
      fail: () => {
        photos.value = photos.value.map(p => (p.key === key ? { ...p, status: "failed" } : p))
        uni.showToast({ title: "照片上传失败", icon: "none" })
      },
      complete: () => {
        uni.hideLoading()
        resolve()
      }
    })
  })
}

function addPhotos() {
  if (submitting.value || loading.value || photos.value.length >= MAX_PHOTOS) return
  uni.chooseImage({
    count: MAX_PHOTOS - photos.value.length,
    sizeType: ["compressed"],
    sourceType: ["album", "camera"],
    success: async res => {
      const files = res.tempFiles || []
      for (let i = 0; i < files.length; i += 1) {
        if (photos.value.length >= MAX_PHOTOS) break
        const file = files[i]
        const path = file?.path || res.tempFilePaths?.[i] || ""
        if (!path) continue
        if (Number(file?.size) > MAX_PHOTO_BYTES) {
          uni.showToast({ title: "单张图片不能超过 20MB", icon: "none" })
          continue
        }
        photoSeq += 1
        const name = photoName(file, photos.value.length)
        const entry = {
          key: `photo-${photoSeq}`,
          localPath: path,
          status: "uploading",
          file_url: "",
          file_name: name,
          file_size: Number(file?.size) || 0
        }
        photos.value = [...photos.value, entry]
        await runUpload(entry.key)
      }
    },
    fail: err => {
      // 用户主动取消不提示。
      if (String(err?.errMsg || "").includes("cancel")) return
      uni.showToast({ title: "选择照片失败", icon: "none" })
    }
  })
}

function removePhoto(key) {
  if (submitting.value) return
  photos.value = photos.value.filter(p => p.key !== key)
}

function retryPhoto(key) {
  if (submitting.value) return
  const photo = photos.value.find(p => p.key === key)
  // 上传中的照片不重复发起；只有失败的才允许重试。
  if (!photo || photo.status !== "failed") return
  runUpload(key)
}

function photoPayload() {
  return photos.value
    .filter(p => p.status === "done" && p.file_url)
    .map(p => ({
      file_url: p.file_url,
      file_type: "image",
      file_name: p.file_name,
      file_size: p.file_size
    }))
}

// ---- 提交 -----------------------------------------------------
async function handleSubmit(isRetry = false) {
  attempted.value = true
  if (loading.value || loadError.value || submitting.value || submitted.value) return
  // 校验失败只留在页面上：不发请求、不弹 toast（设计稿「点击不提交」）。
  if (missing.value.length > 0) return
  if (!form.value.student_id) {
    uni.showToast({ title: "未获取到孩子信息", icon: "none" })
    return
  }
  if (photos.value.some(p => p.status === "uploading")) {
    uni.showToast({ title: "照片上传中，请稍候", icon: "none" })
    return
  }
  submitting.value = true
  submitFailed.value = false
  try {
    const payload = {
      student_id: form.value.student_id,
      reason: form.value.reason.trim(),
      start_date: form.value.start_date,
      end_date: form.value.end_date,
      days: days.value,
      attachments: photoPayload()
    }
    if (form.value.leave_type) payload.leave_type = form.value.leave_type
    const contact = {
      name: form.value.emergency_contact.name.trim(),
      phone: form.value.emergency_contact.phone.trim()
    }
    if (contact.name || contact.phone) payload.emergency_contact = contact
    await createLeave(payload)
    // 成功后到自动返回之间仍有 800ms，锁住按钮避免重复提交。
    submitted.value = true
    uni.showToast({ title: isRetry ? "已重新提交" : "提交成功", icon: "success" })
    setTimeout(() => {
      // 成功后回到请假记录；列表 onShow 会刷新。直接打开创建页时退不回去，就重定向。
      uni.navigateBack({
        fail: () => uni.redirectTo({ url: "/pages/leave/index" })
      })
    }, 800)
  } catch (err) {
    console.warn("[leave-create] submit failed:", err?.message || err)
    // 表单内容不丢，底栏上方挂常驻失败条 + 重试。
    submitFailed.value = true
  } finally {
    submitting.value = false
  }
}

function retrySubmit() {
  if (submitting.value) return
  submitFailed.value = false
  handleSubmit(true)
}

onLoad(() => {
  loadChild()
})
</script>

<template>
  <PageShell :bottom-space="dockSpace" max-width="430px">
    <!-- ===== 加载骨架：五段，与表单同构 ===== -->
    <view v-if="loading" class="loading-flow">
      <view class="sk-section">
        <view class="sk-heading" />
        <view class="sk-type-grid">
          <view v-for="i in 3" :key="i" class="sk-type">
            <view class="sk sk-title" />
            <view class="sk sk-caption" />
          </view>
        </view>
      </view>
      <view class="sk-section">
        <view class="sk-heading" />
        <view class="sk-card">
          <view class="sk-row"><view class="sk sk-label" /><view class="sk sk-date" /></view>
          <view class="sk-row"><view class="sk sk-label" /><view class="sk sk-date" /></view>
          <view class="sk-total"><view class="sk sk-short" /><view class="sk sk-short" /></view>
        </view>
      </view>
      <view class="sk-section">
        <view class="sk-heading" />
        <view class="sk-card sk-reason">
          <view class="sk sk-reason-line" />
          <view class="sk sk-reason-line" />
          <view class="sk sk-reason-line sk-reason-line--short" />
        </view>
      </view>
      <view class="sk-section">
        <view class="sk-heading" />
        <view class="sk-card">
          <view class="sk-field"><view class="sk sk-field-label" /><view class="sk sk-input" /></view>
          <view class="sk-field"><view class="sk sk-field-label" /><view class="sk sk-input" /></view>
        </view>
      </view>
      <view class="sk-section">
        <view class="sk-heading" />
        <view class="sk-card sk-attachment">
          <view class="sk sk-label sk-label--wide" />
          <view class="sk-photo-row"><view class="sk sk-photo" /><view class="sk sk-photo" /></view>
        </view>
      </view>
    </view>

    <!-- ===== 孩子信息加载失败 ===== -->
    <view v-else-if="loadError" class="state-card">
      <view class="state-orb state-orb--danger">
        <text class="state-icon">{{ ICON.alert }}</text>
      </view>
      <text class="state-title">{{ loadError }}</text>
      <text class="state-desc">请检查网络后重试，或返回重新进入</text>
      <view class="state-action" @click="loadChild">
        <text>重试</text>
      </view>
    </view>

    <template v-else>
      <!-- ===== 当前孩子：只读上下文，不是切换器 ===== -->
      <view class="child-context">
        <view class="child-avatar">
          <text>{{ childInitial }}</text>
        </view>
        <view class="child-line">
          <text class="child-prefix">为</text>
          <text class="child-name">{{ child.name || "当前孩子" }}</text>
          <text v-if="child.class_name" class="child-sep">·</text>
          <text v-if="child.class_name" class="child-class">{{ child.class_name }}</text>
        </view>
      </view>

      <view class="form-flow" :class="{ 'is-submitting': submitting }">
        <!-- ===== 请假类型 ===== -->
        <view class="form-section">
          <view class="section-head">
            <text class="section-title">请假类型</text>
          </view>
          <view class="type-grid">
            <view
              v-for="t in LEAVE_TYPES"
              :key="t.key"
              class="type-tile"
              :class="{ 'is-selected': form.leave_type === t.key }"
              @click="selectType(t.key)"
            >
              <view class="type-check">
                <text>{{ ICON.check }}</text>
              </view>
              <text class="type-name">{{ t.label }}</text>
              <text class="type-desc">{{ t.desc }}</text>
            </view>
          </view>
        </view>

        <!-- ===== 请假时间 ===== -->
        <view class="form-section">
          <view class="section-head">
            <text class="section-title">请假时间</text>
            <text class="section-meta">必填</text>
          </view>
          <view class="form-card">
            <view class="date-field">
              <view class="date-row" @click="openSheet('start')">
                <text class="date-label">开始日期</text>
                <view class="date-readout">
                  <text class="date-value" :class="{ 'is-set': Boolean(form.start_date) }">{{ form.start_date || "请选择" }}</text>
                  <text v-if="startWeekday" class="date-weekday">{{ startWeekday }}</text>
                </view>
                <text class="date-chevron">{{ ICON.chevron }}</text>
              </view>
              <text v-if="showFieldErrors && fieldErrors.start" class="field-error">{{ fieldErrors.start }}</text>
            </view>
            <view class="date-field">
              <view class="date-row" @click="openSheet('end')">
                <text class="date-label">结束日期</text>
                <view class="date-readout">
                  <text class="date-value" :class="{ 'is-set': Boolean(form.end_date) }">{{ form.end_date || "请选择" }}</text>
                  <text v-if="endWeekday" class="date-weekday">{{ endWeekday }}</text>
                </view>
                <text class="date-chevron">{{ ICON.chevron }}</text>
              </view>
              <text v-if="showFieldErrors && fieldErrors.end" class="field-error">{{ fieldErrors.end }}</text>
            </view>
            <view class="days-summary">
              <text class="days-total">{{ daysTotalText }}</text>
              <text class="days-note">{{ daysNote }}</text>
            </view>
          </view>
        </view>

        <!-- ===== 请假原因 ===== -->
        <view class="form-section">
          <view class="section-head">
            <text class="section-title">请假原因</text>
            <text class="section-meta">必填</text>
          </view>
          <view class="form-card reason-card">
            <textarea
              v-model="form.reason"
              class="reason-input"
              placeholder="请详细描述请假原因..."
              :maxlength="500"
              :disabled="submitting"
              :adjust-position="true"
            />
            <text v-if="showFieldErrors && fieldErrors.reason" class="field-error field-error--reason">{{ fieldErrors.reason }}</text>
            <view class="reason-foot" :class="{ 'is-near': reasonLength >= 450 }">
              <text>{{ reasonLength }}</text>
              <text>/500</text>
            </view>
          </view>
        </view>

        <!-- ===== 接送人（选填） ===== -->
        <view class="form-section">
          <view class="section-head">
            <text class="section-title">接送人信息</text>
            <text class="section-meta">选填</text>
          </view>
          <view class="form-card">
            <view class="stack-field">
              <text class="stack-label">姓名</text>
              <input
                v-model="form.emergency_contact.name"
                class="text-input"
                placeholder="接送人姓名"
                :disabled="submitting"
              />
            </view>
            <view class="stack-field">
              <text class="stack-label">手机号</text>
              <input
                v-model="form.emergency_contact.phone"
                class="text-input"
                type="number"
                placeholder="接送人手机号"
                :maxlength="11"
                :disabled="submitting"
              />
            </view>
          </view>
        </view>

        <!-- ===== 附件（选填） ===== -->
        <view class="form-section">
          <view class="section-head">
            <text class="section-title">附件</text>
            <text class="section-meta">选填</text>
          </view>
          <view class="form-card attachment-card">
            <text class="upload-intro">病假条、就诊记录等照片</text>
            <text class="upload-meta">选填，最多 3 张</text>
            <view class="upload-grid">
              <view v-for="photo in photos" :key="photo.key" class="photo-cell">
                <view class="photo-frame">
                  <image class="photo-img" :src="photo.localPath" mode="aspectFill" />
                  <view v-if="photo.status !== 'done'" class="photo-mask" @click="retryPhoto(photo.key)">
                    <text class="photo-mask-text">{{ photo.status === "uploading" ? "上传中…" : "上传失败 · 重试" }}</text>
                  </view>
                </view>
                <view class="remove-photo" @click.stop="removePhoto(photo.key)">
                  <view class="remove-dot">
                    <text>{{ ICON.close }}</text>
                  </view>
                </view>
              </view>
              <view v-if="canAddPhoto" class="add-cell" @click="addPhotos">
                <view class="add-inner">
                  <text class="add-icon">{{ ICON.add }}</text>
                  <text class="add-label">添加照片</text>
                </view>
              </view>
            </view>
          </view>
        </view>
      </view>
    </template>

    <!-- ===== 底部固定提交栏 ===== -->
    <template #bottom>
      <view class="submit-dock">
        <view v-if="submitFailed" class="failure-bar">
          <text class="failure-icon">{{ ICON.alert }}</text>
          <text class="failure-text">提交失败，请重试</text>
          <view class="failure-retry" @click="retrySubmit">
            <text>重试</text>
          </view>
        </view>
        <text class="submit-hint" :class="{ 'is-blocking': showBlockingHint }">{{ dockHint }}</text>
        <view
          class="submit-btn"
          :class="{ 'is-disabled': !canSubmit, 'is-loading': submitting }"
          @click="handleSubmit(false)"
        >
          <view v-if="submitting" class="spinner" />
          <text>{{ submitting ? "提交中…" : "提交申请" }}</text>
        </view>
      </view>
    </template>

    <!-- ===== 日期弹层 ===== -->
    <view v-if="sheetOpen" class="sheet-layer">
      <view class="sheet-scrim" @click="closeSheet" />
      <view class="date-sheet">
        <view class="sheet-grabber" />
        <view class="sheet-head">
          <text class="sheet-title">{{ sheetTitle }}</text>
        </view>
        <view class="calendar-head">
          <view
            class="month-btn"
            :class="{ 'is-disabled': prevMonthDisabled }"
            @click="moveMonth(-1)"
          >
            <text>{{ ICON.chevronLeft }}</text>
          </view>
          <text class="calendar-month">{{ sheetMonthLabel }}</text>
          <view class="month-btn" @click="moveMonth(1)">
            <text>{{ ICON.chevron }}</text>
          </view>
        </view>
        <view class="weekday-grid">
          <text v-for="w in WEEKDAYS" :key="w" class="weekday-cell">{{ w }}</text>
        </view>
        <view class="calendar-grid">
          <view v-for="cell in calendarCells" :key="cell.key" class="calendar-cell">
            <view v-if="cell.blank" class="calendar-blank" />
            <view
              v-else
              class="calendar-day"
              :class="{
                'is-selected': cell.selected,
                'is-disabled': cell.disabled,
                'is-today': cell.today
              }"
              @click="pickDate(cell)"
            >
              <text>{{ cell.day }}</text>
            </view>
          </view>
        </view>
        <view class="sheet-actions">
          <view class="sheet-cancel" @click="closeSheet">
            <text>取消</text>
          </view>
          <view
            class="sheet-confirm"
            :class="{ 'is-disabled': sheetConfirmDisabled }"
            @click="confirmSheet"
          >
            <text>确定</text>
          </view>
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 请假申请 · 冷雾白底 + 白色软卡；暖橙只落在选中类型、日期胶囊与提交按钮。
// ============================================================

// ---- 当前孩子 -------------------------------------------------
.child-context {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  margin: 4rpx 4rpx 36rpx;
  min-width: 0;
}

.child-avatar {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  box-shadow: $shadow-sm;
}

.child-line {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: $spacing-xs;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.child-prefix {
  color: $color-text-muted;
  font-weight: $typography-font-weight-normal;
}

.child-name {
  font-weight: $typography-font-weight-semibold;
}

.child-sep {
  color: $color-text-hint;
}

.child-class {
  color: $color-text-body;
}

// ---- 表单骨架 -------------------------------------------------
.form-flow {
  display: flex;
  flex-direction: column;
  gap: 36rpx;
  transition: opacity $animation-duration-fast $animation-easing-default;
}

.form-flow.is-submitting {
  opacity: 0.62;
  pointer-events: none;
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  min-width: 0;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  min-height: 36rpx;
  padding: 0 4rpx;
}

.section-title {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.section-meta {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.form-card {
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

// ---- 请假类型 -------------------------------------------------
.type-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-md;
}

.type-tile {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: $spacing-sm;
  min-width: 0;
  min-height: 192rpx;
  padding: $spacing-xl 20rpx 26rpx;
  border: 2rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  transition:
    border-color $animation-duration-fast $animation-easing-default,
    background-color $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.985);
  }
}

.type-tile.is-selected {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
}

.type-name {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.type-tile.is-selected .type-name {
  font-weight: $typography-font-weight-semibold;
}

.type-desc {
  max-width: 100%;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  overflow-wrap: break-word;
}

.type-tile.is-selected .type-desc {
  color: $color-text-body;
}

.type-check {
  position: absolute;
  top: 14rpx;
  right: 14rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36rpx;
  height: 36rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
  font-family: remixicon;
  font-size: 20rpx;
  line-height: 1;
  opacity: 0;
  transform: scale(0.6);
  transition:
    opacity $animation-duration-fast $animation-easing-default,
    transform $animation-duration-fast $animation-easing-default;
}

.type-tile.is-selected .type-check {
  opacity: 1;
  transform: scale(1);
}

// ---- 请假时间 -------------------------------------------------
.date-field + .date-field {
  border-top: 1rpx solid $color-border-subtle;
}

.date-row {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  width: 100%;
  min-height: 112rpx;
  padding: 20rpx 32rpx;

  &:active {
    background: $color-bg-page;
  }
}

.date-label {
  flex: none;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.date-readout {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: $spacing-xs;
  min-width: 0;
  margin-left: auto;
}

.date-value {
  color: $color-text-muted;
  font-size: $typography-font-size-md;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.date-value.is-set {
  padding: $spacing-xs 18rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-text-primary;
  font-weight: $typography-font-weight-medium;
}

.date-weekday {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

.date-chevron {
  flex: none;
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.days-summary {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-lg;
  padding: 22rpx 32rpx 24rpx;
  border-top: 1rpx solid $color-border-subtle;
}

.days-total {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.days-note {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  text-align: right;
}

// ---- 就地校验 -------------------------------------------------
.field-error {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  margin: 0 32rpx 20rpx;
  // 深一档的红：白底上 $color-semantic-danger 只有 3.76:1（审查 P2-4）。
  color: $color-semantic-danger-strong;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;

  &::before {
    content: "";
    flex: none;
    width: 8rpx;
    height: 8rpx;
    margin-top: 14rpx;
    border-radius: $border-radius-full;
    background: currentColor;
  }
}

.field-error--reason {
  margin: $spacing-sm 0 0;
}

// ---- 请假原因 -------------------------------------------------
.reason-card {
  padding: 28rpx 32rpx 20rpx;
}

.reason-input {
  display: block;
  width: 100%;
  min-height: 192rpx;
  padding: 0;
  border: 0;
  background: transparent;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
  box-sizing: border-box;
}

.reason-foot {
  display: flex;
  justify-content: flex-end;
  margin-top: 14rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.reason-foot.is-near {
  color: $color-brand-primary-dark;
  font-weight: $typography-font-weight-semibold;
}

// ---- 接送人 ---------------------------------------------------
.stack-field {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  min-width: 0;
  padding: $spacing-lg $spacing-xl;

  & + & {
    border-top: 1rpx solid $color-border-subtle;
  }
}

.stack-label {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.text-input {
  display: block;
  width: 100%;
  height: $component-input-height;
  padding: 0 24rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-input-radius;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.4;
  box-sizing: border-box;
}

// ---- 附件 -----------------------------------------------------
.attachment-card {
  padding: 28rpx 32rpx 32rpx;
}

.upload-intro {
  display: block;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.6;
}

.upload-meta {
  display: block;
  margin-top: 2rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.4;
}

// 上限 3 张：三列网格刚好铺满，不留空槽（审查 P2-3）。
.upload-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-md;
  margin-top: 24rpx;
}

// aspect-ratio 小程序不支持：用 padding-top 撑正方形。
.photo-cell,
.add-cell {
  position: relative;
  width: 100%;
  padding-top: 100%;
}

.photo-frame {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  overflow: hidden;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.photo-img {
  width: 100%;
  height: 100%;
}

.photo-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: $spacing-sm;
  // 单处半透明遮罩，无对应 token，按规范局部硬编码。
  background: rgba(30, 41, 59, 0.62);
}

.photo-mask-text {
  color: $color-text-on-brand;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  text-align: center;
}

// 删除热区 88rpx（44px），满足触控下限；可见圆点只有 40rpx。
// 热区收在缩略图右上角内侧，避免与相邻格子的点击区重叠。
.remove-photo {
  position: absolute;
  top: 0;
  right: 0;
  z-index: 2;
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  width: 88rpx;
  height: 88rpx;
  padding: $spacing-xs;
}

.remove-dot {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40rpx;
  height: 40rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-family: remixicon;
  font-size: 20rpx;
  line-height: 1;
  box-shadow: $shadow-sm;
}

.add-inner {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  padding: $spacing-md;
  border: 2rpx dashed $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  color: $color-text-muted;

  &:active {
    border-color: $color-brand-primary-light;
    background: $color-brand-primary-bg;
  }
}

.add-icon {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: 44rpx;
  line-height: 1;
}

.add-label {
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

// ---- 加载骨架 -------------------------------------------------
.loading-flow {
  display: flex;
  flex-direction: column;
  gap: 36rpx;
}

.sk-section {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.sk-heading {
  display: block;
  width: 116rpx;
  height: 22rpx;
  margin: 6rpx 4rpx;
  border-radius: $border-radius-sm;
}

.sk {
  background: $color-border-subtle;
  opacity: 0.48;
  animation: sk-pulse 1.6s $animation-easing-default infinite;
}

.sk-type-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-md;
}

.sk-type {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 20rpx;
  min-height: 192rpx;
  padding: $spacing-xl 24rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sk-title {
  width: 92rpx;
  height: 28rpx;
  border-radius: $border-radius-sm;
}

.sk-caption {
  width: 74%;
  height: 18rpx;
  border-radius: $border-radius-sm;
}

.sk-card {
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
}

.sk-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-xl;
  min-height: 114rpx;
  padding: 0 32rpx;

  & + & {
    border-top: 1rpx solid $color-border-subtle;
  }
}

.sk-label {
  width: 108rpx;
  height: 20rpx;
  border-radius: $border-radius-sm;
}

.sk-label--wide {
  width: 46%;
  height: 20rpx;
}

.sk-date {
  width: 244rpx;
  height: 48rpx;
  border-radius: $border-radius-full;
}

.sk-total {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  padding: $spacing-lg $spacing-xl;
  border-top: 1rpx solid $color-border-subtle;
}

.sk-short {
  width: 108rpx;
  height: 18rpx;
  border-radius: $border-radius-sm;
}

.sk-reason {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  min-height: 268rpx;
  padding: $spacing-xl;
}

.sk-reason-line {
  width: 100%;
  height: 20rpx;
  border-radius: $border-radius-sm;
}

.sk-reason-line--short {
  width: 58%;
}

.sk-field {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  padding: $spacing-lg $spacing-xl;

  & + & {
    border-top: 1rpx solid $color-border-subtle;
  }
}

.sk-field-label {
  width: 76rpx;
  height: 18rpx;
  border-radius: $border-radius-sm;
}

.sk-input {
  width: 100%;
  height: 88rpx;
  border-radius: $component-input-radius;
}

.sk-attachment {
  padding: 28rpx 32rpx 32rpx;
}

.sk-photo-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-md;
  margin-top: 24rpx;
}

.sk-photo {
  width: 100%;
  padding-top: 100%;
  border-radius: $border-radius-lg;
}

// ---- 底部提交栏 -----------------------------------------------
.submit-dock {
  padding: 20rpx 28rpx calc(24rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-xl $border-radius-xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.submit-hint {
  display: block;
  min-height: 40rpx;
  margin: 0 0 $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: break-word;
}

.submit-hint.is-blocking {
  color: $color-semantic-danger-strong;
  font-weight: $typography-font-weight-medium;
}

.submit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  width: 100%;
  height: 96rpx;
  padding: 0 36rpx;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  box-shadow: $shadow-md;
  transition:
    transform $animation-duration-fast $animation-easing-default,
    box-shadow $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.99);
    box-shadow: $shadow-sm;
  }
}

.submit-btn.is-disabled {
  border-color: $color-border-hairline;
  background: $color-bg-page;
  color: $color-text-muted;
  box-shadow: none;
}

.submit-btn.is-loading,
.submit-btn.is-loading.is-disabled {
  border-color: transparent;
  background: $gradient-brand;
  color: $color-text-on-accent;
  box-shadow: $shadow-md;
}

.spinner {
  width: 30rpx;
  height: 30rpx;
  border: 4rpx solid currentColor;
  border-right-color: transparent;
  border-radius: $border-radius-full;
  animation: spin 0.7s linear infinite;
}

.failure-bar {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
  padding: $spacing-sm $spacing-sm $spacing-sm $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-semantic-danger-bg;
}

.failure-icon {
  flex: none;
  color: $color-semantic-danger-strong;
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

.failure-text {
  flex: 1;
  min-width: 0;
  color: $color-semantic-danger-strong;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

.failure-retry {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 128rpx;
  height: $component-input-height;
  padding: 0 24rpx;
  border: 1rpx solid $color-semantic-danger-strong;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-semantic-danger-strong;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;

  &:active {
    transform: scale(0.97);
  }
}

// ---- 孩子信息加载失败 -----------------------------------------
.state-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 40rpx;
  padding: 64rpx 40rpx;
  background: $color-bg-card;
  border-radius: $component-card-radius;
  box-shadow: $shadow-sm;
  text-align: center;
}

.state-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.state-orb--danger {
  background: $color-semantic-danger-bg;
}

.state-icon {
  color: $color-text-hint;
  font-family: remixicon;
  font-size: 44rpx;
  line-height: 1;
}

.state-orb--danger .state-icon {
  color: $color-semantic-danger-strong;
}

.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.state-desc {
  max-width: 480rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.state-action {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 208rpx;
  height: $component-input-height;
  margin-top: $spacing-xl;
  padding: 0 40rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  box-shadow: $shadow-md;
}

// ---- 日期弹层 -------------------------------------------------
.sheet-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 60;
  display: flex;
  align-items: flex-end;
}

.sheet-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: $color-text-primary;
  opacity: 0.4;
}

.date-sheet {
  position: relative;
  z-index: 1;
  width: 100%;
  max-height: 92vh;
  padding: $spacing-md $spacing-xl calc(24rpx + env(safe-area-inset-bottom));
  overflow-y: auto;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.sheet-grabber {
  width: 72rpx;
  height: 8rpx;
  margin: 0 auto 16rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
}

.sheet-head {
  padding: 4rpx 0 20rpx;
  text-align: center;
}

.sheet-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.calendar-head {
  display: grid;
  grid-template-columns: 88rpx minmax(0, 1fr) 88rpx;
  align-items: center;
  gap: $spacing-xs;
}

.month-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  color: $color-text-body;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;

  &:active {
    background: $color-bg-page;
    border-radius: $border-radius-full;
  }
}

.month-btn.is-disabled {
  opacity: 0.3;
  pointer-events: none;
}

.calendar-month {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  text-align: center;
}

.weekday-grid,
.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 4rpx;
}

.weekday-grid {
  padding: $spacing-xs 0 4rpx;
}

.weekday-cell {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.6;
  text-align: center;
}

.calendar-cell {
  min-width: 0;
}

.calendar-blank {
  display: block;
  height: 88rpx;
}

.calendar-day {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 88rpx;
  border: 2rpx solid transparent;
  border-radius: $border-radius-xl;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  font-variant-numeric: tabular-nums;

  &:active {
    background: $color-bg-page;
  }
}

.calendar-day.is-today::after {
  content: "";
  position: absolute;
  bottom: 10rpx;
  left: 50%;
  width: 6rpx;
  height: 6rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
  transform: translateX(-50%);
}

.calendar-day.is-selected {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
  font-weight: $typography-font-weight-semibold;
}

.calendar-day.is-selected::after {
  display: none;
}

.calendar-day.is-disabled {
  color: $color-text-hint;
  opacity: 0.5;
  pointer-events: none;
}

.sheet-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20rpx;
  margin-top: 24rpx;
}

.sheet-cancel,
.sheet-confirm {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 96rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.sheet-cancel {
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-page;
  color: $color-text-body;

  &:active {
    transform: scale(0.98);
  }
}

.sheet-confirm {
  background: $gradient-brand;
  color: $color-text-on-accent;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.98);
  }
}

.sheet-confirm.is-disabled {
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-page;
  color: $color-text-muted;
  box-shadow: none;
  pointer-events: none;
}

// ---- 动效 -----------------------------------------------------
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes sk-pulse {
  0%,
  100% {
    opacity: 0.38;
  }
  50% {
    opacity: 0.7;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
    opacity: 0.55;
  }

  .spinner {
    animation-duration: 1.4s;
  }

  .type-tile,
  .submit-btn {
    transition: none;
  }
}
</style>
