<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { createTeacherNotice, fetchTeacherNotice, fetchTeacherNoticeTemplates, updateTeacherNotice, uploadNoticeAttachment } from "@/api/teacher-notices"
import { requestWithAuth } from "@/api/http"
import { NOTICE_CATEGORIES, noticeCategoryLabel } from "@/utils/notice"

const MAX_ATTACHMENTS = 9
// 与契约「单张 ≤20MB」同口径；上传通道也会拦，这里先拦一次避免白传。
const MAX_ATTACHMENT_BYTES = 20 * 1024 * 1024
// 定时时间统一按东八区提交（后端只认带时区的 RFC3339）。
const SCHEDULE_TIMEZONE = "+08:00"

type NoticeAttachment = {
  file_url: string
  file_type: string
  file_name: string
  file_size: number
  local_path?: string
  status?: "uploading" | "done" | "failed"
  progress?: number
}

const categories = NOTICE_CATEGORIES

const submitting = ref(false)
const submitMode = ref<"draft" | "publish">("publish")
const editingId = ref(0)
const loadingNotice = ref(false)
const isEditing = computed(() => editingId.value > 0)
const attempted = ref(false)
const showClassSheet = ref(false)
const showTemplateSheet = ref(false)
const showConfirm = ref(false)
const classError = ref("")
const classLoading = ref(false)
const templatesLoading = ref(false)
const templatesError = ref("")
const form = ref({
  category: "homework",
  title: "",
  content: "",
  need_confirm: true,
  scheduled_enabled: false,
  scheduled_at: "",
  attachments: [] as NoticeAttachment[],
  class_ids: [] as number[], // 接收范围：任教班级多选（notice-multitarget）
})

const classList = ref<any[]>([])
// 家长人数来自 /teacher/class/list 的 parent_count；旧接口或计数失败时字段缺失/null，
// 此处统一归一成 null，页面回退为只显示班级名，不把 null 渲染成「0 位家长」。
function parentCountOf(c: any): number | null {
  const raw = c?.parent_count
  if (raw === null || raw === undefined || raw === "") return null
  const n = Number(raw)
  return Number.isFinite(n) && n >= 0 ? n : null
}
function classMeta(c: any): string {
  const title = c.title_label || c.title || ""
  const count = parentCountOf(c)
  if (count === null) return title
  return title ? `${title} · ${count} 位家长` : `${count} 位家长`
}
const currentClassLabel = computed(() => {
  const names = classList.value
    .filter((c) => form.value.class_ids.includes(c.class_id))
    .map((c) => c.class_name)
  return names.length ? `${names.join("、")}全体家长` : "请选择班级"
})
const categoryLabel = computed(() => noticeCategoryLabel(form.value.category) || "其他")
const scheduledDate = computed(() => (form.value.scheduled_at || "").split(" ")[0] || "")
const scheduledTime = computed(() => (form.value.scheduled_at || "").split(" ")[1] || "")
const scheduledDateLabel = computed(() => {
  const [y, m, d] = scheduledDate.value.split("-")
  if (!y || !m || !d) return ""
  return `${y}年${Number(m)}月${Number(d)}日`
})
const scheduleLabel = computed(() => (form.value.scheduled_enabled && form.value.scheduled_at ? form.value.scheduled_at : "立即发送"))
const showTitleError = computed(() => attempted.value && !form.value.title.trim())
const showContentError = computed(() => attempted.value && !form.value.content.trim())
const selectedParentCount = computed(() => {
  if (form.value.class_ids.length === 0) return null
  let sum = 0
  for (const id of form.value.class_ids) {
    const count = parentCountOf(classList.value.find((c) => c.class_id === id))
    // 任一班缺人数就不给合计：少算比不显示更误导（跨班同一家长也会重复计数，见文案取舍）。
    if (count === null) return null
    sum += count
  }
  return sum
})
const classFoot = computed(() => {
  if (classError.value) return classError.value
  if (form.value.class_ids.length === 0) return "请选择接收班级"
  const count = selectedParentCount.value
  if (count !== null) return `${form.value.class_ids.length} 个班级 · ${count} 位家长`
  return `${form.value.class_ids.length} 个班级 · 发布后家长会收到微信提醒`
})
const uploadingCount = computed(() => form.value.attachments.filter((a) => a.status === "uploading").length)
const failedCount = computed(() => form.value.attachments.filter((a) => a.status === "failed").length)
const attachmentBlocked = computed(() => uploadingCount.value > 0 || failedCount.value > 0)
const canSubmit = computed(() => {
  if (loadingNotice.value || classLoading.value || classError.value || attachmentBlocked.value) return false
  if (!form.value.title.trim() || !form.value.content.trim()) return false
  if (form.value.class_ids.length === 0) return false
  if (form.value.scheduled_enabled && !form.value.scheduled_at) return false
  return true
})
// 草稿允许缺标题、缺正文、缺班级：只要标题或正文其一非空即可保存。
const canSaveDraft = computed(() => {
  if (loadingNotice.value || attachmentBlocked.value) return false
  return Boolean(form.value.title.trim() || form.value.content.trim())
})
const dockSummary = computed(() => {
  if (loadingNotice.value) return "通知加载中，请稍候"
  if (classLoading.value) return "正在加载班级信息…"
  if (classError.value) return "班级信息未加载，暂不能发布"
  if (uploadingCount.value > 0) return "图片上传中，完成后可发布"
  if (failedCount.value > 0) return "有图片上传失败，请重试或删除"
  if (!form.value.title.trim() && !form.value.content.trim()) return "填写标题或正文后可存草稿"
  if (!form.value.title.trim()) return "还差标题"
  if (!form.value.content.trim()) return "还差正文"
  if (form.value.class_ids.length === 0) return "还差接收班级"
  if (form.value.scheduled_enabled && !form.value.scheduled_at) return "还差发送时间"
  return `${currentClassLabel.value} · ${scheduleLabel.value} · ${form.value.need_confirm ? "需家长确认" : "无需确认"}`
})
const publishLabel = computed(() => {
  if (loadingNotice.value) return "加载中..."
  if (submitting.value && submitMode.value === "publish") return form.value.scheduled_enabled ? "定时中..." : "发布中..."
  return form.value.scheduled_enabled ? "定时发送" : "发布"
})
const confirmDesc = computed(() => {
  if (form.value.scheduled_enabled) return `将于 ${scheduleLabel.value} 自动发送给${currentClassLabel.value}。`
  return `发布后${currentClassLabel.value}会收到微信提醒。`
})

function toggleClass(classId: number) {
  const i = form.value.class_ids.indexOf(classId)
  if (i >= 0) form.value.class_ids.splice(i, 1)
  else form.value.class_ids.push(classId)
}

async function loadClassList() {
  classLoading.value = true
  classError.value = ""
  try {
    const res: any = await requestWithAuth({ url: "/teacher/class/list", method: "GET" })
    const data = res.data || res
    const raw = Array.isArray(data) ? data : (data.list || [])
    classList.value = raw.map((c) => ({ ...c, class_id: Number(c.class_id) }))
    // 编辑模式等详情返回后再按 class_names 映射，避免默认选中第一个班。
    if (!isEditing.value && classList.value.length > 0 && form.value.class_ids.length === 0) {
      form.value.class_ids = [Number(classList.value[0].class_id)]
    }
  } catch (e) {
    console.error("loadClassList failed", e)
    classError.value = "班级列表加载失败，点击重试"
  } finally {
    classLoading.value = false
  }
}

async function loadNotice(id: number) {
  loadingNotice.value = true
  try {
    const res: any = await fetchTeacherNotice(id)
    const notice = res.data || res
    const noticeStatus = Number(notice?.status)
    // 只有草稿（0）与定时（1）可编辑；已发布/已撤回必须走撤回
    if (noticeStatus !== 0 && noticeStatus !== 1) {
      uni.showToast({ title: "仅草稿或定时通知可编辑", icon: "none" })
      setTimeout(() => uni.navigateBack(), 600)
      return
    }

    form.value.category = notice.category || "homework"
    form.value.title = notice.title || ""
    form.value.content = notice.content || ""
    form.value.need_confirm = notice.need_confirm !== false
    form.value.scheduled_enabled = noticeStatus === 1 && Boolean(notice.scheduled_at)
    form.value.scheduled_at = noticeStatus === 1 ? fromRFC3339ToDisplay(notice.scheduled_at) : ""
    // 详情返回的附件与提交入参同形；回填后编辑可增删（整体替换）。
    form.value.attachments = (notice.attachments || [])
      .map((item) => ({
        file_url: item?.file_url || "",
        file_type: item?.file_type || "image",
        file_name: item?.file_name || "",
        file_size: Number(item?.file_size || 0),
        status: "done" as const,
        progress: 100,
      }))
      .filter((item) => item.file_url)

    // 详情契约只回 class_names，没有 class_ids；按名称映射回任教班级。
    const names = Array.isArray(notice.class_names) ? notice.class_names : []
    const ids = []
    const missing = []
    names.forEach((name) => {
      const matched = classList.value.find((c) => c.class_name === name)
      if (matched) ids.push(Number(matched.class_id))
      else missing.push(name)
    })
    form.value.class_ids = ids
    if (missing.length) {
      uni.showToast({ title: "部分接收班级未匹配，请重新选择", icon: "none" })
    }
  } catch (e) {
    uni.showToast({ title: e?.message || "通知加载失败", icon: "none" })
  } finally {
    loadingNotice.value = false
  }
}

function selectCategory(key) {
  form.value.category = key
}

function fileNameOf(filePath: string) {
  const parts = String(filePath || "").split(/[\\/]/)
  return parts[parts.length - 1] || "图片"
}

function enqueueUpload(filePath: string, size: number) {
  if (form.value.attachments.length >= MAX_ATTACHMENTS) return
  const item: NoticeAttachment = {
    file_url: "",
    file_type: "image",
    file_name: fileNameOf(filePath),
    file_size: size || 0,
    local_path: filePath,
    status: "uploading",
    progress: 0,
  }
  form.value.attachments.push(item)
  // 取回数组中的响应式代理再改状态：直接改原始 item 不会触发视图更新。
  const target = form.value.attachments[form.value.attachments.length - 1]
  startUpload(target)
}

function startUpload(item: NoticeAttachment) {
  if (!item.local_path) return
  item.status = "uploading"
  item.progress = 0
  uploadNoticeAttachment(item.local_path, (progress) => {
    // http.js 的进度回调透传的是 0–100 的数值，不是 UploadTask 的事件对象
    if (typeof progress === "number") {
      item.progress = Math.max(0, Math.min(100, progress))
    }
  })
    .then((res) => {
      const data = res?.data || res || {}
      if (!data.file_url) throw new Error("上传返回缺少地址")
      item.file_url = data.file_url
      item.file_type = data.file_type || "image"
      item.file_name = data.file_name || item.file_name
      item.file_size = Number(data.file_size || item.file_size || 0)
      item.progress = 100
      item.status = "done"
    })
    .catch(() => {
      // 失败保留占位图，用户可重试或删除；不让它静默消失。
      item.status = "failed"
    })
}

function addAttachment() {
  const remain = MAX_ATTACHMENTS - form.value.attachments.length
  if (remain <= 0) {
    uni.showToast({ title: `最多上传 ${MAX_ATTACHMENTS} 张图片`, icon: "none" })
    return
  }
  uni.chooseImage({
    count: remain,
    success(res) {
      const picked = (res.tempFiles || []).map((f) => ({ path: f.path, size: Number(f.size || 0) }))
      const files = picked.length
        ? picked
        : (res.tempFilePaths || []).map((path) => ({ path, size: 0 }))
      files.forEach((file) => {
        if (file.size > MAX_ATTACHMENT_BYTES) {
          uni.showToast({ title: "单张图片不能超过 20MB", icon: "none" })
          return
        }
        enqueueUpload(file.path, file.size)
      })
    },
  })
}
function retryUpload(i: number) {
  const item = form.value.attachments[i]
  if (!item || item.status !== "failed") return
  startUpload(item)
}
function removeAttachment(i: number) {
  form.value.attachments.splice(i, 1)
}

function todayString() {
  const now = new Date()
  const m = String(now.getMonth() + 1).padStart(2, "0")
  const d = String(now.getDate()).padStart(2, "0")
  return `${now.getFullYear()}-${m}-${d}`
}

function toggleSchedule() {
  form.value.scheduled_enabled = !form.value.scheduled_enabled
  if (form.value.scheduled_enabled && !form.value.scheduled_at) {
    form.value.scheduled_at = `${todayString()} 09:00`
  }
}

function onScheduleDateChange(e) {
  form.value.scheduled_at = `${e.detail.value} ${scheduledTime.value || "09:00"}`
}
function onScheduleTimeChange(e) {
  form.value.scheduled_at = `${scheduledDate.value || todayString()} ${e.detail.value}`
}

// 后端只认带时区的 RFC3339；把本地展示用的 `YYYY-MM-DD HH:mm` 补成 `+08:00`。
function toRFC3339(dateTimeText: string) {
  const [date, time = "09:00"] = String(dateTimeText || "").split(" ")
  if (!date || !time) return ""
  return `${date}T${time.length === 5 ? `${time}:00` : time}${SCHEDULE_TIMEZONE}`
}
function scheduleInFuture(dateTimeText: string) {
  const iso = toRFC3339(dateTimeText)
  if (!iso) return false
  const ts = new Date(iso).getTime()
  return Number.isFinite(ts) && ts > Date.now()
}

// 后端回传带时区的 RFC3339，编辑定时时按东八区还原成 picker 用的 `YYYY-MM-DD HH:mm`，
// 不依赖设备本地时区（与 toRFC3339 的 +08:00 口径对称）。
function fromRFC3339ToDisplay(iso: string) {
  if (!iso) return ""
  const date = new Date(String(iso))
  if (Number.isNaN(date.getTime())) return ""
  const cst = new Date(date.getTime() + (8 * 60 + date.getTimezoneOffset()) * 60000)
  const pad = (v: number) => String(v).padStart(2, "0")
  return `${cst.getFullYear()}-${pad(cst.getMonth() + 1)}-${pad(cst.getDate())} ${pad(cst.getHours())}:${pad(cst.getMinutes())}`
}

function openClassSheet() {
  if (classLoading.value) return
  if (classError.value) {
    loadClassList()
    return
  }
  showClassSheet.value = true
}
function closeClassSheet() {
  showClassSheet.value = false
}
function openTemplateSheet() {
  showTemplateSheet.value = true
}
function closeTemplateSheet() {
  showTemplateSheet.value = false
}

// 模板
const templates = ref<any[]>([])
async function loadTemplates() {
  templatesLoading.value = true
  templatesError.value = ""
  try {
    const res = await fetchTeacherNoticeTemplates()
    templates.value = res.data?.list || []
  } catch {
    templates.value = []
    templatesError.value = "模板加载失败"
    uni.showToast({ title: "模板加载失败", icon: "none" })
  } finally {
    templatesLoading.value = false
  }
}
function useTemplate(t: Record<string, any>) {
  form.value.title = t.title || ""
  form.value.content = t.content || ""
  if (t.category) form.value.category = t.category
  showTemplateSheet.value = false
}

onLoad((options) => {
  const id = Number(options?.id || 0)
  if (id) {
    editingId.value = id
    uni.setNavigationBarTitle({ title: "编辑通知" })
  }
})

onMounted(async () => {
  await Promise.all([loadClassList(), loadTemplates()])
  if (editingId.value) await loadNotice(editingId.value)
})

// 发布/定时：标题、正文、接收班级齐全；定时还需未来时间。
function validatePublish() {
  if (loadingNotice.value) return "通知加载中，请稍候"
  if (classError.value) return "班级信息未加载，请先重试"
  if (uploadingCount.value > 0) return "图片上传中，请稍候"
  if (failedCount.value > 0) return "有图片上传失败，请重试或删除"
  if (!form.value.title.trim()) return "请输入通知标题"
  if (!form.value.content.trim()) return "请输入通知内容"
  if (form.value.class_ids.length === 0) return "请选择接收班级"
  if (form.value.scheduled_enabled) {
    if (!form.value.scheduled_at) return "请选择发送时间"
    if (!scheduleInFuture(form.value.scheduled_at)) return "发送时间需晚于当前时间"
  }
  return ""
}

// 存草稿：标题或正文其一即可，班级可空。
function validateDraft() {
  if (loadingNotice.value) return "通知加载中，请稍候"
  if (uploadingCount.value > 0) return "图片上传中，请稍候"
  if (failedCount.value > 0) return "有图片上传失败，请重试或删除"
  if (!form.value.title.trim() && !form.value.content.trim()) return "请至少填写标题或正文"
  return ""
}

function attachmentPayload() {
  return form.value.attachments
    .filter((a) => a.status !== "failed" && a.file_url)
    .map((a) => ({
      file_url: a.file_url,
      file_type: a.file_type || "image",
      file_name: a.file_name || "",
      file_size: Number(a.file_size || 0),
    }))
}

function onPublishClick() {
  attempted.value = true
  if (loadingNotice.value) {
    uni.showToast({ title: "通知加载中，请稍候", icon: "none" })
    return
  }
  if (classError.value) {
    uni.showToast({ title: "班级信息未加载，请先重试", icon: "none" })
    return
  }
  const msg = validatePublish()
  if (msg) {
    uni.showToast({ title: msg, icon: "none" })
    return
  }
  submitMode.value = "publish"
  showConfirm.value = true
}

function onSaveDraft() {
  if (submitting.value) return
  const msg = validateDraft()
  if (msg) {
    uni.showToast({ title: msg, icon: "none" })
    return
  }
  submit("draft")
}

async function doSubmit() {
  showConfirm.value = false
  await submit("publish")
}

// mode=draft → status 0；mode=publish → 开了定时 status 1，否则 status 2。
async function submit(mode: "draft" | "publish") {
  if (loadingNotice.value || submitting.value) {
    return
  }
  const msg = mode === "draft" ? validateDraft() : validatePublish()
  if (msg) { uni.showToast({ title: msg, icon: "none" }); return }
  const status = mode === "draft" ? 0 : (form.value.scheduled_enabled ? 1 : 2)
  const payload = {
    ...form.value,
    status,
    // 只有定时才带时间；其余状态由后端清空，避免草稿残留旧定时
    scheduled_at: status === 1 ? toRFC3339(form.value.scheduled_at) : null,
    attachments: attachmentPayload(),
  }
  submitMode.value = mode
  submitting.value = true
  let succeeded = false
  try {
    if (isEditing.value) {
      await updateTeacherNotice(editingId.value, payload)
    } else {
      await createTeacherNotice(payload)
    }
    const doneText = status === 0 ? "草稿已保存" : (status === 1 ? "已设置定时发送" : "已发布")
    uni.showToast({ title: doneText, icon: "success" })
    succeeded = true
    setTimeout(() => {
        const pages = getCurrentPages()
        pages.length > 1 ? uni.navigateBack() : uni.reLaunch({ url: "/pages/index/index" })
      }, 600)
  } catch (e) {
    uni.showToast({ title: e?.message || (status === 0 ? "草稿保存失败" : "发布失败"), icon: "none" })
  } finally {
    // 成功后保持“发布中/保存中”直到 600ms 后的返回，避免这段时间内重复提交。
    if (!succeeded) submitting.value = false
  }
}
</script>

<template>
  <view class="page">
    <scroll-view class="body" scroll-y>
      <view class="content">
        <!-- 通知类型 -->
        <view class="group">
          <view class="group-head"><text class="group-title">通知类型</text></view>
          <view class="card type-card">
            <view class="chips">
              <view
                v-for="c in categories"
                :key="c.key"
                class="chip"
                :class="{ on: form.category === c.key }"
                @click="selectCategory(c.key)"
              >{{ c.label }}</view>
            </view>
          </view>
        </view>

        <!-- 快捷填写 -->
        <view class="group">
          <view class="group-head"><text class="group-title">快捷填写</text></view>
          <view class="card">
            <view class="cell cell-tap" @click="openTemplateSheet">
              <text class="cell-label">常用模板</text>
              <view class="cell-trail">
                <text v-if="templatesLoading" class="cell-value">加载中…</text>
                <text v-else-if="templatesError" class="class-error">{{ templatesError }}</text>
                <text v-else class="cell-value">{{ templates.length ? templates.length + " 条可用" : "暂无模板" }}</text>
                <text class="chev">›</text>
              </view>
            </view>
            <text class="cell-foot">套用模板后可继续修改内容</text>
          </view>
        </view>

        <!-- 通知内容 -->
        <view class="group">
          <view class="group-head"><text class="group-title">通知内容</text></view>
          <view class="card">
            <view class="cell field-cell">
              <text class="cell-label">标题</text>
              <input
                v-model="form.title"
                class="field-input"
                maxlength="80"
                placeholder="请输入通知标题"
                placeholder-class="field-placeholder"
              />
              <text class="field-count">{{ form.title.length }} / 80</text>
            </view>
            <view v-if="showTitleError" class="field-error"><text>请输入通知标题</text></view>

            <view class="cell field-cell field-cell--stack">
              <view class="field-head">
                <text class="cell-label">正文</text>
                <text class="field-count">{{ form.content.length }} 字</text>
              </view>
              <textarea
                v-model="form.content"
                class="field-textarea"
                maxlength="-1"
                placeholder="请输入通知内容..."
                placeholder-class="field-placeholder"
              />
            </view>
            <view v-if="showContentError" class="field-error"><text>请输入通知内容</text></view>

            <view class="cell attach-cell">
              <view class="attach-head">
                <text class="cell-label">图片</text>
                <text class="attach-count">{{ form.attachments.length }}/9</text>
              </view>
              <view class="thumbs">
                <view
                  v-for="(img, i) in form.attachments"
                  :key="img.file_url || img.local_path || i"
                  class="thumb"
                >
                  <image :src="img.local_path || img.file_url" mode="aspectFill" class="thumb-img" />
                  <view v-if="img.status === 'uploading'" class="thumb-mask">
                    <text>{{ img.progress || 0 }}%</text>
                  </view>
                  <view
                    v-else-if="img.status === 'failed'"
                    class="thumb-mask thumb-mask--error"
                    @click="retryUpload(i)"
                  >
                    <text>上传失败</text>
                    <text class="thumb-retry">点击重试</text>
                  </view>
                  <view class="thumb-del" @click="removeAttachment(i)">×</view>
                </view>
                <view v-if="form.attachments.length < 9" class="add-thumb" @click="addAttachment">
                  <text class="plus">＋</text>
                  <text>添加</text>
                </view>
              </view>
              <text class="edit-hint">仅支持图片，最多 9 张，单张不超过 20MB</text>
            </view>
          </view>
        </view>

        <!-- 接收范围 -->
        <view class="group">
          <view class="group-head">
            <text class="group-title">接收范围</text>
            <text class="group-note">必选</text>
          </view>
          <view class="card">
            <view class="cell cell-tap" :class="{ 'cell-disabled': classLoading }" @click="openClassSheet">
              <text class="cell-label">接收班级</text>
              <view class="cell-trail">
                <text v-if="classLoading" class="cell-value">加载中…</text>
                <text v-else-if="classError" class="class-error">{{ classError }}</text>
                <text v-else class="cell-value">{{ currentClassLabel }}</text>
                <text class="chev">›</text>
              </view>
            </view>
            <text class="cell-foot">{{ classFoot }}</text>
          </view>
        </view>

        <!-- 发送设置 -->
        <view class="group">
          <view class="group-head"><text class="group-title">发送设置</text></view>
          <view class="card">
            <view class="cell switch-cell">
              <view class="cell-text">
                <text class="cell-label">需要家长确认收到</text>
                <text class="cell-hint">开启后可查看已读未读状态</text>
              </view>
              <view class="switch" :class="{ on: form.need_confirm }" @click="form.need_confirm = !form.need_confirm">
                <view class="switch-knob" />
              </view>
            </view>

            <view class="cell switch-cell">
              <view class="cell-text">
                <text class="cell-label">定时发送</text>
                <text class="cell-hint">{{ scheduleLabel }}</text>
              </view>
              <view class="switch" :class="{ on: form.scheduled_enabled }" @click="toggleSchedule">
                <view class="switch-knob" />
              </view>
            </view>

            <view v-if="form.scheduled_enabled" class="schedule">
              <picker mode="date" :value="scheduledDate" @change="onScheduleDateChange">
                <view class="cell cell-tap">
                  <text class="cell-label">发送日期</text>
                  <view class="cell-trail">
                    <text class="cell-value">{{ scheduledDateLabel || "请选择日期" }}</text>
                    <text class="chev">›</text>
                  </view>
                </view>
              </picker>
              <picker mode="time" :value="scheduledTime" @change="onScheduleTimeChange">
                <view class="cell cell-tap">
                  <text class="cell-label">发送时间</text>
                  <view class="cell-trail">
                    <text class="cell-value">{{ scheduledTime || "请选择时间" }}</text>
                    <text class="chev">›</text>
                  </view>
                </view>
              </picker>
              <text class="cell-foot">到点后自动发送，家长会收到微信提醒</text>
            </view>
          </view>
        </view>
      </view>
    </scroll-view>

    <!-- 底部固定提交栏：主按钮发布/定时，次按钮存草稿 -->
    <view class="dock">
      <text class="dock-summary">{{ dockSummary }}</text>
      <view class="dock-actions">
        <button
          class="draft-btn"
          :class="{ 'is-disabled': !canSaveDraft || submitting || loadingNotice }"
          :disabled="submitting || loadingNotice"
          @click="onSaveDraft"
        >
          {{ submitting && submitMode === "draft" ? "保存中..." : "存草稿" }}
        </button>
        <button
          class="publish-btn"
          :class="{ 'is-disabled': !canSubmit || submitting || loadingNotice }"
          :disabled="submitting || loadingNotice"
          @click="onPublishClick"
        >
          {{ publishLabel }}
        </button>
      </view>
    </view>

    <!-- 接收班级弹层 -->
    <view v-if="showClassSheet" class="layer">
      <view class="scrim" @click="closeClassSheet" />
      <view class="sheet">
        <view class="sheet-head">
          <text class="sheet-btn" @click="closeClassSheet">取消</text>
          <text class="sheet-title">接收班级</text>
          <text class="sheet-btn sheet-btn--strong" @click="closeClassSheet">完成</text>
        </view>
        <view class="sheet-body">
          <view
            v-for="c in classList"
            :key="c.class_id"
            class="pick-row"
            @click="toggleClass(c.class_id)"
          >
            <view class="pick-copy">
              <text class="pick-name">{{ c.class_name }}</text>
              <text class="pick-meta">{{ classMeta(c) }}</text>
            </view>
            <view class="check-dot" :class="{ on: form.class_ids.includes(c.class_id) }">
              <text v-if="form.class_ids.includes(c.class_id)">✓</text>
            </view>
          </view>
          <text v-if="classList.length === 0" class="sheet-note">暂无可选班级</text>
          <text class="sheet-note">仅显示你任教的班级，可多选</text>
        </view>
      </view>
    </view>

    <!-- 常用模板弹层 -->
    <view v-if="showTemplateSheet" class="layer">
      <view class="scrim" @click="closeTemplateSheet" />
      <view class="sheet">
        <view class="sheet-head">
          <text class="sheet-btn" @click="closeTemplateSheet">取消</text>
          <text class="sheet-title">常用模板</text>
          <text class="sheet-btn" />
        </view>
        <view class="sheet-body">
          <view v-for="t in templates" :key="t.id" class="tpl-row" @click="useTemplate(t)">
            <view class="tpl-text">
              <text class="tpl-title">{{ t.title }}</text>
              <text class="tpl-preview">{{ t.content }}</text>
            </view>
            <text class="use-btn">使用</text>
          </view>
          <text v-if="templates.length === 0" class="sheet-note">暂无可用模板</text>
          <text class="sheet-note">选择后会自动填入标题、正文和类型</text>
        </view>
      </view>
    </view>

    <!-- 发布确认 -->
    <view v-if="showConfirm" class="layer layer--center">
      <view class="scrim" @click="showConfirm = false" />
      <view class="dialog">
        <view class="dialog-body">
          <text class="dialog-title">{{ form.scheduled_enabled ? "确认定时发送？" : "确认发布这条通知？" }}</text>
          <text class="dialog-desc">{{ confirmDesc }}</text>
          <view class="dialog-meta">
            <view class="dialog-row">
              <text class="dialog-key">类型</text>
              <text class="dialog-val">{{ categoryLabel }}</text>
            </view>
            <view class="dialog-row">
              <text class="dialog-key">发送方式</text>
              <text class="dialog-val">{{ scheduleLabel }}</text>
            </view>
          </view>
        </view>
        <view class="dialog-actions">
          <button class="dialog-btn" @click="showConfirm = false">取消</button>
          <button class="dialog-btn dialog-btn--primary" :disabled="submitting" @click="doSubmit">
            {{ submitting ? "处理中..." : (form.scheduled_enabled ? "确认定时" : "确认发布") }}
          </button>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">

.page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: $color-bg-page;
}

.body {
  flex: 1;
  min-height: 0;
}

.content {
  padding: $component-page-padding-y $component-page-padding-x calc(224rpx + env(safe-area-inset-bottom));
}

.group {
  margin-bottom: 36rpx;
}

.group-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  padding: 0 8rpx 16rpx;
}

.group-title {
  font-size: $typography-font-size-sm;
  font-weight: 600;
  line-height: 1.4;
  letter-spacing: 0.04em;
  color: $color-text-body;
}

.group-note {
  font-size: $typography-font-size-sm;
  font-weight: 500;
  color: $color-text-muted;
}

.card {
  background: $color-bg-card;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  box-shadow: $shadow-md;
  overflow: hidden;
}

/* 类型 */
.type-card {
  padding: 24rpx 28rpx;
}

.chips {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12rpx;
}

.chip {
  height: 88rpx;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 8rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: 500;
  line-height: 1.3;
}

.chip.on {
  background: $color-brand-primary-bg;
  border-color: $color-brand-primary;
  color: $color-brand-primary-dark;
  font-weight: 600;
}

/* cell */
.cell {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-height: 104rpx;
  padding: 20rpx 28rpx;
  border-top: 1rpx solid $color-border-hairline;
  max-width: 100%;
  box-sizing: border-box;
}

.card > .cell:first-child {
  border-top: 0;
}

.cell-tap:active {
  background: $color-bg-page;
}

.cell-disabled {
  opacity: 0.6;
}

.cell-label {
  flex: none;
  font-size: $typography-font-size-md;
  font-weight: 500;
  line-height: 1.4;
  color: $color-text-primary;
}

.cell-text {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
  flex: 1;
}

.cell-hint {
  font-size: $typography-font-size-sm;
  line-height: 1.45;
  color: $color-text-muted;
}

.cell-trail {
  flex: 1;
  min-width: 0;
  margin-left: auto;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8rpx;
}

.cell-value {
  font-size: $typography-font-size-md;
  line-height: 1.45;
  color: $color-text-muted;
  text-align: right;
  overflow-wrap: anywhere;
}

.chev {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-lg;
}

.cell-foot {
  display: block;
  padding: 0 28rpx 24rpx;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  line-height: 1.5;
  color: $color-text-muted;
}

/* 内容 */
.field-cell {
  padding: 0 28rpx;
  min-height: 108rpx;
}

.field-input {
  flex: 1;
  min-width: 0;
  width: 100%;
  padding: 30rpx 0;
  border: 0;
  background: transparent;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: 500;
  line-height: 1.5;
}

.field-count {
  flex: none;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  color: $color-text-muted;
}

.field-placeholder {
  color: $color-text-muted;
  font-weight: 400;
}

.field-cell--stack {
  display: block;
  padding: 24rpx 28rpx 28rpx;
  min-height: 0;
}

.field-cell--stack .cell-label {
  display: block;
  margin-bottom: 12rpx;
}

.field-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: 12rpx;
}

.field-head .cell-label {
  margin-bottom: 0;
}

.field-textarea {
  display: block;
  width: 100%;
  min-height: 168rpx;
  padding: 0;
  border: 0;
  background: transparent;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.75;
  box-sizing: border-box;
}

.field-error {
  display: flex;
  align-items: center;
  gap: 8rpx;
  padding: 18rpx 28rpx;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-base;
  font-weight: 500;
  line-height: 1.45;
  border-top: 1rpx solid $color-border-hairline;
}

.attach-cell {
  display: block;
  padding: 26rpx 28rpx 28rpx;
}

.attach-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8rpx;
  margin-bottom: 18rpx;
}

.attach-count {
  font-size: $typography-font-size-sm;
  font-weight: 500;
  color: $color-text-muted;
}

.thumbs {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16rpx;
}

.thumb {
  position: relative;
  // 3 列九宫格媒体位，设计稿 100px；token 无媒体专用尺寸。
  height: 200rpx;
  border-radius: $border-radius-lg;
  overflow: hidden;
  border: 1rpx solid $color-border-hairline;
  background: $color-bg-page;
}

.thumb-img {
  width: 100%;
  height: 100%;
  display: block;
}

.thumb-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  // 上传中/失败遮罩：token 无 overlay 色，单点特殊值（同 .thumb-del 的压暗层）
  background: rgba(30, 41, 59, 0.55);
  color: $color-text-on-brand;
  font-size: $typography-font-size-sm;
  font-weight: 600;
}

.thumb-mask--error {
  background: rgba(30, 41, 59, 0.72);
}

.thumb-retry {
  font-size: $typography-font-size-xs;
  font-weight: 500;
}

.thumb-del {
  position: absolute;
  top: 0;
  right: 0;
  width: 72rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: $color-text-on-brand;
  font-size: 36rpx;
  line-height: 1;
  // 图片右上角的半透明压暗层；token 中没有 overlay 色，按 DESIGN.md 允许单点特殊值。
  background: rgba(30, 41, 59, 0.55);
  border-bottom-left-radius: $border-radius-full;
}

.add-thumb {
  height: 200rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  border: 1rpx dashed $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: 500;
}

.add-thumb .plus {
  font-size: 44rpx;
  font-weight: 300;
  line-height: 1;
}

.edit-hint {
  display: block;
  margin-top: 12rpx;
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

/* 范围 */
.class-error {
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  line-height: 1.4;
  text-align: right;
  overflow-wrap: anywhere;
}

/* 发送设置 */
.switch-cell {
  min-height: 128rpx;
  gap: 28rpx;
}

.switch {
  position: relative;
  flex: none;
  width: 96rpx;
  height: 56rpx;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
  transition: background $animation-duration-fast;
}

.switch.on {
  background: $color-brand-primary;
  border-color: $color-brand-primary;
}

.switch-knob {
  position: absolute;
  top: 4rpx;
  left: 4rpx;
  width: 44rpx;
  height: 44rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast;
}

.switch.on .switch-knob {
  transform: translateX(40rpx);
}

.schedule {
  border-top: 1rpx solid $color-border-hairline;
  background: $color-brand-primary-bg;
}

.schedule .cell {
  border-top: 1rpx solid $color-border-hairline;
}

.schedule .cell:first-child {
  border-top: 0;
}

/* 底部提交栏 */
.dock {
  position: fixed;
  left: 50%;
  bottom: 0;
  transform: translateX(-50%);
  width: 100%;
  max-width: 430px;
  z-index: 30;
  padding: 20rpx 28rpx calc(24rpx + env(safe-area-inset-bottom));
  background: $color-bg-card;
  border-top: 1rpx solid $color-border-hairline;
  box-sizing: border-box;
}

.dock-summary {
  display: block;
  margin-bottom: 16rpx;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  line-height: 1.45;
  color: $color-text-muted;
  text-align: center;
  overflow-wrap: anywhere;
}

.dock-actions {
  display: flex;
  align-items: stretch;
  gap: $spacing-md;
}

.draft-btn {
  flex: 0 0 216rpx;
  min-height: 96rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-brand-primary-light;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-lg;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.publish-btn {
  flex: 1;
  min-width: 0;
  min-height: 96rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid transparent;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-xl;
  font-weight: 600;
  letter-spacing: 0.02em;
  box-shadow: $shadow-md;
}

.draft-btn.is-disabled,
.draft-btn[disabled] {
  background: $color-bg-page;
  border-color: $color-border-hairline;
  color: $color-text-muted;
}

.publish-btn.is-disabled,
.publish-btn[disabled] {
  background: $color-bg-card;
  border-color: $color-border-hairline;
  color: $color-text-muted;
  box-shadow: none;
}

/* 弹层 */
.layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 80;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.layer--center {
  align-items: center;
  padding: 48rpx;
}

.scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: $color-text-primary;
  opacity: 0.42;
}

.sheet {
  position: relative;
  width: 100%;
  max-width: 430px;
  max-height: 82vh;
  display: flex;
  flex-direction: column;
  background: $color-bg-card;
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  box-shadow: $shadow-lg;
  padding-bottom: env(safe-area-inset-bottom);
  overflow: hidden;
}

.sheet-head {
  flex: none;
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 8rpx;
  min-height: 108rpx;
  padding: 0 16rpx;
  border-bottom: 1rpx solid $color-border-hairline;
}

.sheet-title {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  line-height: 1.4;
  text-align: center;
  color: $color-text-primary;
}

.sheet-btn {
  justify-self: start;
  min-width: 120rpx;
  min-height: 88rpx;
  display: inline-flex;
  align-items: center;
  padding: 0 16rpx;
  background: transparent;
  color: $color-text-body;
  font-size: $typography-font-size-lg;
  font-weight: 400;
  line-height: 1;
}

.sheet-btn--strong {
  justify-self: end;
  color: $color-brand-primary-dark;
  font-weight: 600;
}

.sheet-body {
  overflow-y: auto;
  max-height: 62vh;
}

.sheet-note {
  display: block;
  padding: 20rpx 32rpx 28rpx;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-text-muted;
}

.pick-row {
  width: 100%;
  min-height: 112rpx;
  display: flex;
  align-items: center;
  gap: 20rpx;
  padding: 16rpx 32rpx;
  border-top: 1rpx solid $color-border-hairline;
  box-sizing: border-box;
}

.pick-row:first-child {
  border-top: 0;
}

.pick-copy {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.pick-name {
  font-size: $typography-font-size-lg;
  font-weight: 500;
  line-height: 1.4;
  color: $color-text-primary;
}

.pick-meta {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.check-dot {
  flex: none;
  width: 48rpx;
  height: 48rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-on-brand;
  font-size: $typography-font-size-sm;
}

.check-dot.on {
  background: $color-brand-primary;
  border-color: $color-brand-primary;
}

/* 模板 */
.tpl-row {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 24rpx;
  padding: 28rpx 32rpx;
  border-top: 1rpx solid $color-border-hairline;
  box-sizing: border-box;
}

.tpl-row:first-child {
  border-top: 0;
}

.tpl-text {
  flex: 1;
  min-width: 0;
}

.tpl-title {
  display: block;
  font-size: $typography-font-size-md;
  font-weight: 600;
  line-height: 1.4;
  color: $color-text-primary;
  margin-bottom: 6rpx;
}

.tpl-preview {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
  color: $color-text-muted;
}

.use-btn {
  flex: none;
  min-height: 88rpx;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 28rpx;
  border: 1rpx solid $color-brand-primary-light;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: 600;
  line-height: 1;
}

/* 确认框 */
.dialog {
  position: relative;
  width: 100%;
  max-width: 624rpx;
  background: $color-bg-card;
  border-radius: $border-radius-xl;
  box-shadow: $shadow-lg;
  overflow: hidden;
}

.dialog-body {
  padding: 44rpx 40rpx 36rpx;
  text-align: center;
}

.dialog-title {
  display: block;
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  line-height: 1.4;
  color: $color-text-primary;
  margin-bottom: 16rpx;
}

.dialog-desc {
  display: block;
  font-size: $typography-font-size-md;
  line-height: 1.6;
  color: $color-text-body;
  overflow-wrap: anywhere;
}

.dialog-meta {
  margin-top: 28rpx;
  padding: 20rpx 24rpx;
  display: grid;
  gap: 12rpx;
  background: $color-bg-page;
  border-radius: $border-radius-lg;
  text-align: left;
}

.dialog-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 24rpx;
}

.dialog-key {
  flex: none;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  color: $color-text-muted;
}

.dialog-val {
  font-size: $typography-font-size-base;
  font-weight: 500;
  color: $color-text-primary;
  text-align: right;
  overflow-wrap: anywhere;
}

.dialog-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  border-top: 1rpx solid $color-border-hairline;
}

.dialog-btn {
  min-height: 104rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  background: transparent;
  color: $color-text-body;
  font-size: $typography-font-size-xl;
  font-weight: 500;
}

.dialog-btn + .dialog-btn {
  border-left: 1rpx solid $color-border-hairline;
  color: $color-brand-primary-dark;
  font-weight: 600;
}

.dialog-btn::after {
  border: none;
}
</style>
