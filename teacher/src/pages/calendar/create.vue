<script setup>
import { ref } from "vue"
import { createTeacherCalendarEvent } from "@/api/teacher-calendar"

const form = ref({
  title: "",
  event_type: "activity",
  description: "",
  event_date: "",
  start_time: "09:00",
  end_time: "10:00",
  location: "",
  is_all_day: false,
  color: "",
  importance: "normal",
  cover: "",
  is_published: true,
  attachments: []
})

const submitting = ref(false)
const selectedDate = ref("请选择日期")

const eventTypes = [
  { value: "notice", label: "通知" },
  { value: "activity", label: "活动" },
  { value: "holiday", label: "假期" },
  { value: "meeting", label: "家长会" },
  { value: "checkup", label: "体检" }
]

const importanceLevels = [
  { value: "normal", label: "普通" },
  { value: "important", label: "重要" },
  { value: "urgent", label: "紧急" }
]

function selectType(et) { form.value.event_type = et }
function selectImportance(level) { form.value.importance = level }

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
  uni.showModal({
    title: "输入开始时间",
    editable: true,
    placeholderText: "如 09:00",
    success(res) { if (res.confirm && res.content) form.value.start_time = res.content.trim() }
  })
}
function pickEndTime() {
  uni.showModal({
    title: "输入结束时间",
    editable: true,
    placeholderText: "如 17:00",
    success(res) { if (res.confirm && res.content) form.value.end_time = res.content.trim() }
  })
}

function uploadCover() {
  uni.chooseImage({ count: 1, success(res) { form.value.cover = res.tempFilePaths[0]; uni.showToast({ title: "封面已选择", icon: "success" }) } })
}

function addAttachment() {
  uni.chooseMessageFile({ count: 5, success(res) { form.value.attachments.push(...res.tempFiles.map(f => f.name)); uni.showToast({ title: `已添加 ${res.tempFiles.length} 个附件`, icon: "success" }) } })
}

function toggleNotify() { form.value.is_published = !form.value.is_published }

async function submitForm() {
  if (!form.value.title.trim()) { uni.showToast({ title: "请输入活动标题", icon: "none" }); return }
  if (!form.value.event_date) { uni.showToast({ title: "请选择活动日期", icon: "none" }); return }
  submitting.value = true
  try {
    await createTeacherCalendarEvent({
      title: form.value.title.trim(),
      event_date: form.value.event_date,
      event_type: form.value.event_type,
      description: form.value.description,
      start_time: form.value.start_time,
      end_time: form.value.end_time,
      location: form.value.location,
      is_all_day: form.value.is_all_day,
      color: form.value.color,
      importance: form.value.importance,
      cover: form.value.cover,
      is_published: form.value.is_published
    })
    uni.showToast({ title: "发布成功", icon: "success" })
    setTimeout(() => uni.navigateBack(), 1200)
  } catch (err) {
    uni.showToast({ title: err?.message || "发布失败", icon: "none" })
  } finally { submitting.value = false }
}
</script>

<template>
  <view class="page">
    <view class="form-card">
      <view class="section-head"><view class="indicator is-blue" /><text class="section-title">基本信息</text></view>
      <view class="field">
        <view class="field-label"><text class="label-text">活动标题</text><text class="label-required">*</text></view>
        <view class="input-box">
          <text class="input-icon">题</text>
          <input v-model="form.title" class="form-input" placeholder="请输入活动标题" placeholder-style="color: #94A3B8; font-size: $typography-font-size-md" />
        </view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">活动类型</text><text class="label-required">*</text></view>
        <view class="type-row">
          <view v-for="t in eventTypes" :key="t.value" class="type-btn" :class="{ active: form.event_type === t.value }" @click="selectType(t.value)"><text>{{ t.label }}</text></view>
        </view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">活动描述</text></view>
        <view class="textarea-box">
          <textarea v-model="form.description" class="form-textarea" placeholder="请输入活动描述，如活动内容、注意事项等" placeholder-style="color: #94A3B8; font-size: $typography-font-size-md" />
        </view>
      </view>
    </view>

    <view class="form-card">
      <view class="section-head"><view class="indicator is-green" /><text class="section-title">时间地点</text></view>
      <view class="field">
        <view class="field-label"><text class="label-text">活动日期</text><text class="label-required">*</text></view>
        <view class="input-box picker" @click="pickDate"><text class="input-icon">日</text><text class="picker-value">{{ selectedDate }}</text><text class="picker-chevron">▾</text></view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">活动时间</text></view>
        <view class="time-range">
          <view class="input-box picker half" @click="pickStartTime"><text class="input-icon">时</text><text class="picker-value">{{ form.start_time }}</text><text class="picker-chevron">▾</text></view>
          <text class="time-sep">至</text>
          <view class="input-box picker half" @click="pickEndTime"><text class="input-icon">时</text><text class="picker-value">{{ form.end_time }}</text><text class="picker-chevron">▾</text></view>
        </view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">活动地点</text></view>
        <view class="input-box"><text class="input-icon">点</text><input v-model="form.location" class="form-input" placeholder="请输入活动地点" placeholder-style="color: #94A3B8; font-size: $typography-font-size-md" /></view>
      </view>
    </view>

    <view class="form-card">
      <view class="section-head"><view class="indicator is-orange" /><text class="section-title">发布设置</text></view>
      <view class="notify-row">
        <view class="notify-left"><text class="notify-icon">铃</text><view class="notify-text"><text class="notify-title">通知家长</text><text class="notify-desc">发布后自动推送至家长端</text></view></view>
        <view class="switch" :class="{ on: form.is_published }" @click="toggleNotify"><view class="switch-dot" /></view>
      </view>
    </view>

    <view class="form-card">
      <view class="section-head"><view class="indicator is-purple" /><text class="section-title">更多设置</text></view>
      <view class="field">
        <view class="field-label"><text class="label-text">封面图片</text><text class="label-optional">（选填）</text></view>
        <view class="upload-area" @click="uploadCover">
          <image v-if="form.cover" :src="form.cover" mode="aspectFill" class="cover-preview" />
          <template v-else><text class="upload-icon">图</text><text class="upload-text">点击上传封面图片</text><text class="upload-hint">支持 JPG、PNG，大小不超过 5MB</text></template>
        </view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">附件</text><text class="label-optional">（选填）</text></view>
        <view class="attach-btn" @click="addAttachment"><text class="attach-icon">附</text><text class="attach-text">添加附件（文档、图片等）</text><text class="attach-arrow">▸</text></view>
        <view v-if="form.attachments.length" class="attach-list"><view v-for="(name, idx) in form.attachments" :key="idx" class="attach-item"><text>{{ name }}</text></view></view>
      </view>
      <view class="field">
        <view class="field-label"><text class="label-text">重要程度</text></view>
        <view class="type-row">
          <view v-for="level in importanceLevels" :key="level.value" class="type-btn" :class="{ active: form.importance === level.value, 'is-important': level.value === 'important' }" @click="selectImportance(level.value)"><text v-if="level.value === 'important'" class="importance-star">★</text><text>{{ level.label }}</text></view>
        </view>
      </view>
    </view>

    <view class="bottom-actions">
      <button class="submit-btn" :loading="submitting" @click="submitForm" style="flex: 1"><text class="submit-icon">发</text><text class="submit-text">立即发布</text></button>
    </view>
    <view class="bottom-safe" />
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: $spacing-lg 28rpx;
  background: $color-bg-page;
}

.form-card {
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: $spacing-lg;
}

.section-head {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: $spacing-xl;
}

.indicator {
  width: 10rpx;
  height: 40rpx;
  border-radius: $border-radius-full;
}

.indicator.is-blue   { background: $color-semantic-info; }
.indicator.is-green  { background: #22C55E; }
.indicator.is-orange { background: #F97316; }
.indicator.is-purple { background: #8B5CF6; }

.section-title {
  font-size: $typography-font-size-lg;
  font-weight: 700;
  color: $color-text-primary;
}

.field {
  margin-bottom: $spacing-xl;
  &:last-child { margin-bottom: 0; }
}

.field-label {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  margin-bottom: $spacing-md;
}

.label-text {
  font-size: $typography-font-size-base;
  font-weight: 500;
  color: #334155;
}

.label-required {
  font-size: $typography-font-size-base;
  color: $color-semantic-danger;
}

.label-optional {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.input-box {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 $spacing-xl;
  border: 2rpx solid #E2E8F0;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
}

.input-box.picker { justify-content: space-between; &.half { flex: 1; } }

.input-icon {
  flex-shrink: 0;
  font-size: $typography-font-size-lg;
  font-weight: 700;
  color: $color-text-hint;
  margin-right: $spacing-md;
}

.form-input {
  flex: 1;
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

.picker-value {
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

.picker-chevron {
  font-size: $typography-font-size-base;
  color: $color-text-hint;
  margin-left: $spacing-md;
}

.textarea-box {
  padding: $spacing-lg $spacing-xl;
  border: 2rpx solid #E2E8F0;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  min-height: 192rpx;
}

.form-textarea {
  width: 100%;
  min-height: 140rpx;
  font-size: $typography-font-size-md;
  color: $color-text-primary;
}

.type-row { display: flex; gap: $spacing-md; }

.type-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  height: 88rpx;
  border: 2rpx solid #E2E8F0;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  text { font-size: $typography-font-size-base; color: $color-text-muted; }
}

.type-btn.active {
  border-color: $color-semantic-info;
  background: $color-semantic-info-bg;
  text { color: $color-semantic-info; font-weight: 600; }
}

.type-btn.active.is-important {
  border-color: #F59E0B; background: #FFFBEB;
  text { color: #F59E0B; }
}

.importance-star { font-size: $typography-font-size-md; color: #F59E0B; }
.time-range { display: flex; align-items: center; gap: $spacing-lg; }
.time-sep { font-size: $typography-font-size-md; color: $color-text-hint; flex-shrink: 0; }

.class-chips { display: flex; flex-wrap: wrap; gap: $spacing-md; }

.class-chip {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  padding: 14rpx $spacing-lg;
  border-radius: $border-radius-md;
  background: $color-semantic-info-bg;
}

.chip-name { font-size: $typography-font-size-sm; font-weight: 500; color: $color-semantic-info; }
.chip-remove { font-size: $typography-font-size-md; color: $color-semantic-info; }
.class-chip.add { border: 2rpx dashed #CBD5E1; background: transparent; }
.chip-add-icon { font-size: $typography-font-size-md; color: $color-text-hint; }
.chip-add-text { font-size: $typography-font-size-sm; color: $color-text-hint; }

.notify-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: $spacing-lg;
  padding-top: $spacing-lg;
  border-top: 2rpx solid #F1F5F9;
}

.notify-left { display: flex; align-items: center; gap: $spacing-md; }
.notify-icon { font-size: $typography-font-size-xl; font-weight: 700; color: $color-text-muted; }
.notify-text { display: flex; flex-direction: column; gap: 4rpx; }
.notify-title { font-size: $typography-font-size-md; font-weight: 500; color: #334155; }
.notify-desc { font-size: $typography-font-size-sm; color: $color-text-hint; }

.switch {
  position: relative; width: 96rpx; height: 56rpx;
  border-radius: $border-radius-full; background: #CBD5E1;
  padding: $spacing-xs; transition: background 0.2s;
}
.switch.on { background: $color-semantic-info; }
.switch-dot { width: 40rpx; height: 40rpx; border-radius: $border-radius-full; background: #fff; transition: transform 0.2s; .switch.on & { transform: translateX(40rpx); } }

.upload-area {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: $spacing-sm; height: 224rpx; border: 4rpx dashed #CBD5E1;
  border-radius: $border-radius-xl; background: $color-bg-card; overflow: hidden;
}
.cover-preview { width: 100%; height: 100%; }
.upload-icon { font-size: 56rpx; font-weight: 700; color: $color-text-hint; }
.upload-text { font-size: $typography-font-size-base; color: $color-text-hint; }
.upload-hint { font-size: $typography-font-size-xs; color: #CBD5E1; }

.attach-btn {
  display: flex; align-items: center; height: 88rpx; padding: 0 $spacing-xl;
  border: 2rpx solid #E2E8F0; border-radius: $border-radius-xl; background: $color-bg-card;
}
.attach-icon { font-size: $typography-font-size-lg; font-weight: 700; color: $color-text-hint; margin-right: $spacing-md; }
.attach-text { flex: 1; font-size: $typography-font-size-md; color: $color-text-hint; }
.attach-arrow { font-size: $typography-font-size-md; color: $color-text-hint; }

.attach-list { display: flex; flex-wrap: wrap; gap: $spacing-sm; margin-top: $spacing-md; }
.attach-item { padding: $spacing-xs 20rpx; border-radius: $border-radius-md; background: $color-semantic-info-bg; text { font-size: $typography-font-size-sm; color: $color-semantic-info; } }

.bottom-actions { display: flex; gap: $spacing-lg; margin-top: $spacing-xs; }
.draft-btn {
  flex: 1; display: flex; align-items: center; justify-content: center;
  gap: $spacing-sm; height: 96rpx; border: 2rpx solid #E2E8F0;
  border-radius: $border-radius-xl; background: #fff;
}
.draft-icon { font-size: $typography-font-size-xl; color: $color-text-muted; }
.draft-text { font-size: $typography-font-size-lg; font-weight: 500; color: $color-text-muted; }

.submit-btn {
  flex: 1; display: flex; align-items: center; justify-content: center;
  gap: $spacing-sm; height: 96rpx; border-radius: $border-radius-xl;
  background: $color-semantic-info; box-shadow: 0 8rpx 24rpx rgba(59, 130, 246, 0.3); border: none;
}
.submit-icon { font-size: $typography-font-size-xl; color: #fff; }
.submit-text { font-size: $typography-font-size-lg; font-weight: 700; color: #fff; }

.bottom-safe { height: calc(60rpx + env(safe-area-inset-bottom)); }
</style>
