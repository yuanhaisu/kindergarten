<script setup>
import { ref } from "vue"
import { onLoad, onShow } from "@dcloudio/uni-app"
import { fetchDoctorCalendarEvents, createDoctorCalendarEvent, updateDoctorCalendarEvent, deleteDoctorCalendarEvent } from "@/api/doctor-calendar"
import { fetchDict } from "@/api/dict"

// 医生端园级事件发布页（ui-design-notes §5）：
// 复用老师端表单骨架；受众固定为「全园家长可见」；编辑已发布事件的保存确认逻辑与老师端一致。

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

const form = ref(emptyForm())
const submitting = ref(false)
const selectedDate = ref("请选择日期")

const editId = ref(0)
const original = ref(null)

const confirmVisible = ref(false)
const confirmNotify = ref(true)
const changeLines = ref([])
const SUBSTANTIVE_PREFIXES = ["日期", "时间", "地点"]

const events = ref([])
const loading = ref(false)

const eventTypes = ref([{ item_key: "checkup", item_label: "体检" }])
async function loadTypes() {
  try {
    const res = await fetchDict("calendar_event_type")
    const items = res.data?.items || []
    if (items.length) eventTypes.value = items
  } catch { /* 降级体检 */ }
}

function selectType(et) { form.value.event_type = et }

function pickDate() {
  const today = new Date()
  uni.showModal({
    title: "输入日期",
    editable: true,
    placeholderText: `如 ${today.getFullYear()}-${String(today.getMonth()+1).padStart(2,"0")}-${String(today.getDate()).padStart(2,"0")}`,
    success(res) {
      if (res.confirm && res.content) {
        form.value.event_date = res.content.trim()
        selectedDate.value = res.content.trim()
      }
    }
  })
}
function pickStartTime() {
  uni.showModal({ title: "输入开始时间", editable: true, placeholderText: "如 09:00",
    success(res) { if (res.confirm && res.content) form.value.start_time = res.content.trim() } })
}
function pickEndTime() {
  uni.showModal({ title: "输入结束时间", editable: true, placeholderText: "如 11:00",
    success(res) { if (res.confirm && res.content) form.value.end_time = res.content.trim() } })
}

function timeRange(o) {
  if (!o.start_time) return "未设"
  return o.end_time ? `${o.start_time}-${o.end_time}` : o.start_time
}
function substantiveLines() {
  const o = original.value
  if (!o) return []
  const lines = []
  if ((o.event_date || "") !== form.value.event_date) lines.push(`日期 ${o.event_date || "未设"} → ${form.value.event_date}`)
  if (timeRange(o) !== timeRange(form.value)) lines.push(`时间 ${timeRange(o)} → ${timeRange(form.value)}`)
  if ((o.location || "") !== (form.value.location || "")) lines.push(`地点 ${o.location || "无"} → ${form.value.location || "无"}`)
  if (o.title !== form.value.title.trim()) lines.push("标题已更新")
  if ((o.description || "") !== (form.value.description || "")) lines.push("内容已更新")
  if (o.event_type !== form.value.event_type) lines.push("类型已更新")
  return lines
}

function validate() {
  if (!form.value.title.trim()) { uni.showToast({ title: "请输入事件标题", icon: "none" }); return false }
  if (!form.value.event_date) { uni.showToast({ title: "请选择事件日期", icon: "none" }); return false }
  return true
}

function buildPayload(status, notify) {
  return {
    title: form.value.title.trim(),
    event_date: form.value.event_date,
    event_type: form.value.event_type,
    description: form.value.description,
    start_time: form.value.start_time,
    end_time: form.value.end_time,
    location: form.value.location,
    is_all_day: form.value.is_all_day,
    status,
    notify
  }
}

async function save(status, notify) {
  submitting.value = true
  try {
    if (editId.value) await updateDoctorCalendarEvent(editId.value, buildPayload(status, notify))
    else await createDoctorCalendarEvent(buildPayload(status, notify))
    uni.showToast({ title: status === "draft" ? "已存草稿" : (editId.value ? "已保存" : "发布成功"), icon: "success" })
    resetForm()
    await loadList()
  } catch (err) {
    uni.showToast({ title: err?.message || "保存失败", icon: "none" })
  } finally { submitting.value = false }
}

function submitDraft() {
  if (!validate()) return
  save("draft", false)
}

function submitPublish() {
  if (!validate()) return
  if (editId.value && original.value?.status === "published") {
    const lines = substantiveLines()
    if (lines.length === 0) { save("published", false); return }
    changeLines.value = lines
    confirmNotify.value = lines.some(l => SUBSTANTIVE_PREFIXES.some(p => l.startsWith(p)))
    confirmVisible.value = true
    return
  }
  save("published", form.value.notify)
}

function confirmSave() { confirmVisible.value = false; save("published", confirmNotify.value) }
function cancelConfirm() { confirmVisible.value = false }

function resetForm() {
  editId.value = 0
  original.value = null
  form.value = emptyForm()
  selectedDate.value = "请选择日期"
}

function startEdit(e) {
  editId.value = e.id
  original.value = e
  form.value = {
    title: e.title || "",
    event_type: e.event_type || "checkup",
    description: e.description || "",
    event_date: e.event_date || "",
    start_time: e.start_time || "",
    end_time: e.end_time || "",
    location: e.location || "",
    is_all_day: !!e.is_all_day,
    notify: false
  }
  selectedDate.value = e.event_date || "请选择日期"
  uni.pageScrollTo({ scrollTop: 0, duration: 200 })
}

function removeEvent(e) {
  uni.showModal({
    title: "删除事件",
    content: `确定删除「${e.title}」吗？删除后全园家长不再可见。`,
    success: async (r) => {
      if (!r.confirm) return
      try {
        await deleteDoctorCalendarEvent(e.id)
        if (editId.value === e.id) resetForm()
        uni.showToast({ title: "已删除", icon: "success" })
        await loadList()
      } catch (err) {
        uni.showToast({ title: err?.message || "删除失败", icon: "none" })
      }
    }
  })
}

async function loadList() {
  loading.value = true
  try {
    const res = await fetchDoctorCalendarEvents({})
    events.value = (res.data?.list || []).slice().sort((a, b) => (b.event_date || "").localeCompare(a.event_date || ""))
  } catch {
    events.value = []
    uni.showToast({ title: "加载事件失败", icon: "none" })
  } finally { loading.value = false }
}

onLoad(() => { loadTypes() })
onShow(() => { loadList() })
</script>

<template>
  <view class="page">
    <view class="scope-note">
      <text class="scope-chip">全园家长可见</text>
      <text class="scope-text">保健医发布的园级事件（体检等），全园老师与家长均可在日历中看到。</text>
    </view>

    <view class="form-card">
      <view class="section-head"><view class="indicator" /><text class="section-title">{{ editId ? "编辑园级事件" : "发布园级事件" }}</text><text v-if="editId" class="section-reset" @click="resetForm">取消编辑</text></view>
      <view class="field">
        <view class="field-label"><text class="label-text">事件标题</text><text class="label-required">*</text></view>
        <view class="input-box">
          <text class="input-icon">题</text>
          <input v-model="form.title" class="form-input" placeholder="如：全园体检" placeholder-style="color: #94A3B8; font-size: $typography-font-size-md" />
        </view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">事件类型</text></view>
        <view class="type-row">
          <view v-for="t in eventTypes" :key="t.item_key" class="type-btn" :class="{ active: form.event_type === t.item_key }" @click="selectType(t.item_key)"><text>{{ t.item_label }}</text></view>
        </view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">事件说明</text></view>
        <view class="textarea-box">
          <textarea v-model="form.description" class="form-textarea" placeholder="如体检项目、注意事项、需家长配合的事项" placeholder-style="color: #94A3B8; font-size: $typography-font-size-md" />
        </view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">事件日期</text><text class="label-required">*</text></view>
        <view class="input-box picker" @click="pickDate"><text class="input-icon">日</text><text class="picker-value">{{ selectedDate }}</text><text class="picker-chevron">▾</text></view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">事件时间</text></view>
        <view class="time-range">
          <view class="input-box picker half" @click="pickStartTime"><text class="input-icon">时</text><text class="picker-value">{{ form.start_time || "未设" }}</text></view>
          <text class="time-sep">至</text>
          <view class="input-box picker half" @click="pickEndTime"><text class="input-icon">时</text><text class="picker-value">{{ form.end_time || "未设" }}</text></view>
        </view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">事件地点</text></view>
        <view class="input-box"><text class="input-icon">点</text><input v-model="form.location" class="form-input" placeholder="如：医务室" placeholder-style="color: #94A3B8; font-size: $typography-font-size-md" /></view>
      </view>
      <view v-if="editId && original?.status === 'published'" class="notify-hint">
        <text class="notify-hint-text">该事件已发布——保存时将让你确认是否通知家长（按变更内容预置建议）。</text>
      </view>
      <view v-else class="notify-row">
        <view class="notify-left"><text class="notify-icon">铃</text><view class="notify-text"><text class="notify-title">通知家长</text><text class="notify-desc">发布后向全园家长推送通知</text></view></view>
        <view class="switch" :class="{ on: form.notify }" @click="form.notify = !form.notify"><view class="switch-dot" /></view>
      </view>
      <view class="bottom-actions">
        <button class="draft-btn" :loading="submitting" @click="submitDraft"><text class="draft-text">存为草稿</text></button>
        <button class="submit-btn" :loading="submitting" @click="submitPublish"><text class="submit-text">{{ editId ? "保存发布" : "立即发布" }}</text></button>
      </view>
    </view>

    <view class="list-card">
      <view class="section-head"><view class="indicator is-plain" /><text class="section-title">最近发布</text></view>
      <view v-if="loading" class="list-state"><text>加载中...</text></view>
      <view v-else-if="events.length === 0" class="list-state"><text>暂无园级事件</text></view>
      <view v-else class="event-list">
        <view v-for="e in events" :key="e.id" class="event-row">
          <view class="event-main">
            <view class="event-title-row">
              <text class="event-title">{{ e.title }}</text>
              <text v-if="e.status === 'draft'" class="mini-chip is-draft">草稿</text>
            </view>
            <text class="event-meta">{{ e.event_date }}{{ e.is_all_day ? " · 全天" : (e.start_time ? " · " + e.start_time : "") }}{{ e.location ? " · " + e.location : "" }}</text>
          </view>
          <view class="event-actions">
            <text class="action-edit" @click="startEdit(e)">编辑</text>
            <text class="action-delete" @click="removeEvent(e)">删除</text>
          </view>
        </view>
      </view>
    </view>
    <view class="bottom-safe" />

    <!-- 变更通知确认弹层（与老师端一致） -->
    <view v-if="confirmVisible" class="sheet-mask" @click="cancelConfirm">
      <view class="sheet" @click.stop>
        <text class="sheet-title">通知家长？</text>
        <text class="sheet-sub">将通知全园家长</text>
        <view class="sheet-changes">
          <view v-for="(line, i) in changeLines" :key="i" class="change-line">
            <view class="change-dot" />
            <text class="change-text">{{ line }}</text>
          </view>
        </view>
        <view class="notify-row sheet-notify">
          <view class="notify-left"><text class="notify-icon">铃</text><view class="notify-text"><text class="notify-title">发布变更通知</text><text class="notify-desc">家长端会在日历顶部看到此变更</text></view></view>
          <view class="switch" :class="{ on: confirmNotify }" @click="confirmNotify = !confirmNotify"><view class="switch-dot" /></view>
        </view>
        <view class="sheet-actions">
          <button class="sheet-cancel" @click="cancelConfirm"><text>取消</text></button>
          <button class="submit-btn sheet-save" :loading="submitting" @click="confirmSave"><text class="submit-text">保存</text></button>
        </view>
        <view class="bottom-safe" />
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: $spacing-lg $component-page-padding-x;
  background: $color-bg-page;
}

.scope-note { display: flex; align-items: center; gap: $spacing-md; margin-bottom: $spacing-lg; }
.scope-chip { flex-shrink: 0; padding: 4rpx $component-badge-padding-x; border-radius: $component-badge-radius; background: $color-semantic-info-bg; color: $color-semantic-info; font-size: $component-badge-font-size; font-weight: 500; }
.scope-text { font-size: $typography-font-size-sm; color: $color-text-muted; }

.form-card, .list-card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  margin-bottom: $spacing-lg;
}

.section-head { display: flex; align-items: center; gap: $spacing-md; margin-bottom: $spacing-xl; }
.indicator { width: 10rpx; height: 40rpx; border-radius: $border-radius-full; background: $color-brand-primary; }
.indicator.is-plain { background: $color-border-hairline; }
.section-title { font-size: $typography-font-size-lg; font-weight: 700; color: $color-text-primary; flex: 1; }
.section-reset { font-size: $typography-font-size-sm; color: $color-semantic-info; }

.field { margin-bottom: $spacing-xl; &:last-of-type { margin-bottom: 0; } }
.field-label { display: flex; align-items: center; gap: $spacing-xs; margin-bottom: $spacing-md; }
.label-text { font-size: $typography-font-size-base; font-weight: 500; color: $color-text-body; }
.label-required { font-size: $typography-font-size-base; color: $color-semantic-danger; }

.input-box {
  display: flex; align-items: center;
  height: $component-input-height; padding: 0 $spacing-xl;
  border: 2rpx solid $color-border-hairline;
  border-radius: $component-input-radius; background: $color-bg-card;
}
.input-box.picker { justify-content: space-between; &.half { flex: 1; } }
.input-icon { flex-shrink: 0; font-size: $typography-font-size-lg; font-weight: 700; color: $color-text-hint; margin-right: $spacing-md; }
.form-input { flex: 1; font-size: $typography-font-size-md; color: $color-text-primary; }
.picker-value { font-size: $typography-font-size-md; color: $color-text-primary; }
.picker-chevron { font-size: $typography-font-size-base; color: $color-text-hint; margin-left: $spacing-md; }

.textarea-box { padding: $spacing-lg $spacing-xl; border: 2rpx solid $color-border-hairline; border-radius: $component-input-radius; background: $color-bg-card; min-height: 160rpx; }
.form-textarea { width: 100%; min-height: 110rpx; font-size: $typography-font-size-md; color: $color-text-primary; }

.type-row { display: flex; flex-wrap: wrap; gap: $spacing-md; }
.type-btn {
  flex: 1; min-width: 160rpx;
  display: flex; align-items: center; justify-content: center;
  height: $component-input-height;
  border: 2rpx solid $color-border-hairline;
  border-radius: $component-input-radius; background: $color-bg-card;
  text { font-size: $typography-font-size-base; color: $color-text-muted; }
}
.type-btn.active { border-color: $color-brand-primary; background: $color-brand-primary-bg; text { color: $color-brand-primary-dark; font-weight: 600; } }

.time-range { display: flex; align-items: center; gap: $spacing-lg; }
.time-sep { font-size: $typography-font-size-md; color: $color-text-hint; flex-shrink: 0; }

.notify-row { display: flex; align-items: center; justify-content: space-between; margin-top: $spacing-xl; padding-top: $spacing-lg; border-top: 2rpx solid $color-bg-page; }
.notify-left { display: flex; align-items: center; gap: $spacing-md; }
.notify-icon { font-size: $typography-font-size-xl; font-weight: 700; color: $color-text-muted; }
.notify-text { display: flex; flex-direction: column; gap: 4rpx; }
.notify-title { font-size: $typography-font-size-md; font-weight: 500; color: $color-text-body; }
.notify-desc { font-size: $typography-font-size-sm; color: $color-text-hint; }
.notify-hint { margin-top: $spacing-xl; padding-top: $spacing-lg; border-top: 2rpx solid $color-bg-page; }
.notify-hint-text { font-size: $typography-font-size-sm; color: $color-text-muted; line-height: 1.6; }

.switch { position: relative; width: 96rpx; height: 56rpx; border-radius: $border-radius-full; background: $color-border-hairline; padding: $spacing-xs; transition: background 0.2s; flex-shrink: 0; margin-left: $spacing-lg; }
.switch.on { background: $color-brand-primary; }
.switch-dot { width: 40rpx; height: 40rpx; border-radius: $border-radius-full; background: $color-text-on-brand; transition: transform 0.2s; .switch.on & { transform: translateX(40rpx); } }

.bottom-actions { display: flex; gap: $spacing-lg; margin-top: $spacing-xl; }
.draft-btn { flex: 1; display: flex; align-items: center; justify-content: center; height: 88rpx; border: 2rpx solid $color-border-hairline; border-radius: $border-radius-full; background: $color-bg-card; }
.draft-text { font-size: $typography-font-size-base; font-weight: 500; color: $color-text-muted; }
.submit-btn { flex: 1.4; display: flex; align-items: center; justify-content: center; height: 88rpx; border-radius: $border-radius-full; background: $gradient-brand; box-shadow: $shadow-md; border: none; }
.submit-text { font-size: $typography-font-size-base; font-weight: 700; color: $color-text-on-brand; }

.list-state { padding: 60rpx 0; text-align: center; text { font-size: $typography-font-size-md; color: $color-text-hint; } }
.event-list { display: flex; flex-direction: column; gap: $spacing-md; }
.event-row { display: flex; align-items: center; justify-content: space-between; gap: $spacing-lg; padding: $spacing-lg; border-radius: $border-radius-xl; background: $color-bg-page; }
.event-main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 4rpx; }
.event-title-row { display: flex; align-items: center; gap: $spacing-sm; min-width: 0; }
.event-title { font-size: $typography-font-size-md; font-weight: 600; color: $color-text-primary; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mini-chip { flex-shrink: 0; padding: 2rpx 12rpx; border-radius: $border-radius-full; font-size: $typography-font-size-xs; font-weight: 500; }
.mini-chip.is-draft { background: $color-semantic-warning-bg; color: $color-semantic-warning; }
.event-meta { font-size: $typography-font-size-sm; color: $color-text-hint; }
.event-actions { flex-shrink: 0; display: flex; gap: $spacing-lg; }
.action-edit { font-size: $typography-font-size-sm; color: $color-semantic-info; }
.action-delete { font-size: $typography-font-size-sm; color: $color-semantic-danger; }

.bottom-safe { height: calc(60rpx + env(safe-area-inset-bottom)); }

/* 变更确认弹层 */
.sheet-mask { position: fixed; inset: 0; background: rgba(30, 41, 59, 0.45); z-index: 999; display: flex; align-items: flex-end; }
.sheet { width: 100%; box-sizing: border-box; background: $color-bg-card; border-radius: $border-radius-2xl $border-radius-2xl 0 0; box-shadow: $shadow-lg; padding: $spacing-xl $spacing-xl 0; }
.sheet-title { display: block; font-size: $typography-font-size-2xl; font-weight: 700; color: $color-text-primary; }
.sheet-sub { display: block; margin-top: $spacing-xs; font-size: $typography-font-size-sm; color: $color-text-hint; }
.sheet-changes { margin-top: $spacing-lg; display: flex; flex-direction: column; gap: $spacing-sm; }
.change-line { display: flex; align-items: center; gap: $spacing-md; }
.change-dot { flex-shrink: 0; width: 12rpx; height: 12rpx; border-radius: $border-radius-full; background: $color-semantic-warning; }
.change-text { font-size: $typography-font-size-base; color: $color-text-body; }
.sheet-notify { margin-top: $spacing-lg; }
.sheet-actions { display: flex; gap: $spacing-lg; margin-top: $spacing-xl; }
.sheet-cancel { flex: 1; display: flex; align-items: center; justify-content: center; height: 88rpx; border: 2rpx solid $color-border-hairline; border-radius: $border-radius-full; background: $color-bg-card; text { font-size: $typography-font-size-lg; color: $color-text-muted; } }
.sheet-save { flex: 1.4; height: 88rpx; }
</style>
