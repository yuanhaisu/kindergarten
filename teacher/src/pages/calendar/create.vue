<script setup>
import { ref } from "vue"
import { createTeacherCalendarEvent } from "@/api/teacher-calendar"

const form = ref({
  title: "",
  type: "notification",
  description: "",
  date: "",
  start_time: "09:00",
  end_time: "10:00",
  location: "",
  classes: [],
  notify_parents: true,
  importance: "important",
  cover: "",
  attachments: []
})

const submitting = ref(false)
const selectedDate = ref("2026年3月19日")

const eventTypes = [
  { value: "notification", label: "通知" },
  { value: "activity", label: "活动" },
  { value: "meeting", label: "家长会" }
]

const importanceLevels = [
  { value: "normal", label: "普通" },
  { value: "important", label: "重要" },
  { value: "urgent", label: "紧急" }
]

const availableClasses = ["中一班", "中二班", "大一班", "大二班", "小一班"]

function selectType(type) { form.value.type = type }
function selectImportance(level) { form.value.importance = level }

function toggleClass(className) {
  const idx = form.value.classes.indexOf(className)
  if (idx >= 0) form.value.classes.splice(idx, 1)
  else form.value.classes.push(className)
}

function addMoreClass() {
  uni.showActionSheet({
    itemList: availableClasses.filter(c => !form.value.classes.includes(c)),
    success(res) {
      const cls = availableClasses.filter(c => !form.value.classes.includes(c))[res.tapIndex]
      if (cls) form.value.classes.push(cls)
    }
  })
}

function pickDate() {
  uni.showToast({ title: "选择日期", icon: "none" })
}

function pickStartTime() {
  uni.showToast({ title: "选择开始时间", icon: "none" })
}

function pickEndTime() {
  uni.showToast({ title: "选择结束时间", icon: "none" })
}

function uploadCover() {
  uni.chooseImage({
    count: 1,
    success(res) {
      form.value.cover = res.tempFilePaths[0]
      uni.showToast({ title: "封面已选择", icon: "success" })
    }
  })
}

function addAttachment() {
  uni.chooseMessageFile({
    count: 5,
    success(res) {
      form.value.attachments.push(...res.tempFiles.map(f => f.name))
      uni.showToast({ title: `已添加 ${res.tempFiles.length} 个附件`, icon: "success" })
    }
  })
}

function toggleNotify() {
  form.value.notify_parents = !form.value.notify_parents
}

async function saveDraft() {
  uni.showToast({ title: "草稿已保存", icon: "success" })
}

async function submitForm() {
  if (!form.value.title.trim()) {
    uni.showToast({ title: "请输入活动标题", icon: "none" })
    return
  }
  if (!form.value.classes.length) {
    uni.showToast({ title: "请选择参与班级", icon: "none" })
    return
  }

  submitting.value = true
  try {
    await createTeacherCalendarEvent({
      title: form.value.title.trim(),
      event_date: form.value.date || new Date().toISOString().slice(0, 10),
      event_type: form.value.type,
      description: form.value.description,
      location: form.value.location,
      classes: form.value.classes
    })
    uni.showToast({ title: "发布成功", icon: "success" })
    setTimeout(() => uni.navigateBack(), 1200)
  } catch (err) {
    uni.showToast({ title: err?.message || "发布失败", icon: "none" })
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <view class="page">
    <!-- 表单卡片 — 基本信息 -->
    <view class="form-card">
      <view class="section-head">
        <view class="indicator is-blue" />
        <text class="section-title">基本信息</text>
      </view>

      <!-- 活动标题 -->
      <view class="field">
        <view class="field-label">
          <text class="label-text">活动标题</text>
          <text class="label-required">*</text>
        </view>
        <view class="input-box">
          <text class="input-icon">题</text>
          <input
            v-model="form.title"
            class="form-input"
            placeholder="请输入活动标题"
            placeholder-style="color: #94A3B8; font-size: 28rpx"
          />
        </view>
      </view>

      <!-- 活动类型 -->
      <view class="field">
        <view class="field-label">
          <text class="label-text">活动类型</text>
          <text class="label-required">*</text>
        </view>
        <view class="type-row">
          <view
            v-for="t in eventTypes"
            :key="t.value"
            class="type-btn"
            :class="{ active: form.type === t.value }"
            @click="selectType(t.value)"
          >
            <text>{{ t.label }}</text>
          </view>
        </view>
      </view>

      <!-- 活动描述 -->
      <view class="field">
        <view class="field-label">
          <text class="label-text">活动描述</text>
        </view>
        <view class="textarea-box">
          <textarea
            v-model="form.description"
            class="form-textarea"
            placeholder="请输入活动描述，如活动内容、注意事项等"
            placeholder-style="color: #94A3B8; font-size: 28rpx"
          />
        </view>
      </view>
    </view>

    <!-- 表单卡片 — 时间地点 -->
    <view class="form-card">
      <view class="section-head">
        <view class="indicator is-green" />
        <text class="section-title">时间地点</text>
      </view>

      <!-- 活动日期 -->
      <view class="field">
        <view class="field-label">
          <text class="label-text">活动日期</text>
          <text class="label-required">*</text>
        </view>
        <view class="input-box picker" @click="pickDate">
          <text class="input-icon">日</text>
          <text class="picker-value">{{ selectedDate }}</text>
          <text class="picker-chevron">▾</text>
        </view>
      </view>

      <!-- 活动时间 -->
      <view class="field">
        <view class="field-label">
          <text class="label-text">活动时间</text>
        </view>
        <view class="time-range">
          <view class="input-box picker half" @click="pickStartTime">
            <text class="input-icon">时</text>
            <text class="picker-value">{{ form.start_time }}</text>
            <text class="picker-chevron">▾</text>
          </view>
          <text class="time-sep">至</text>
          <view class="input-box picker half" @click="pickEndTime">
            <text class="input-icon">时</text>
            <text class="picker-value">{{ form.end_time }}</text>
            <text class="picker-chevron">▾</text>
          </view>
        </view>
      </view>

      <!-- 活动地点 -->
      <view class="field">
        <view class="field-label">
          <text class="label-text">活动地点</text>
        </view>
        <view class="input-box">
          <text class="input-icon">点</text>
          <input
            v-model="form.location"
            class="form-input"
            placeholder="请输入活动地点"
            placeholder-style="color: #94A3B8; font-size: 28rpx"
          />
        </view>
      </view>
    </view>

    <!-- 表单卡片 — 参与班级 -->
    <view class="form-card">
      <view class="section-head">
        <view class="indicator is-orange" />
        <text class="section-title">参与班级</text>
      </view>

      <view class="field">
        <view class="field-label">
          <text class="label-text">选择班级</text>
          <text class="label-required">*</text>
        </view>
        <view class="class-chips">
          <view
            v-for="cls in form.classes"
            :key="cls"
            class="class-chip"
            @click="toggleClass(cls)"
          >
            <text class="chip-name">{{ cls }}</text>
            <text class="chip-remove">×</text>
          </view>
          <view class="class-chip add" @click="addMoreClass">
            <text class="chip-add-icon">+</text>
            <text class="chip-add-text">添加</text>
          </view>
        </view>
      </view>

      <!-- 通知家长 -->
      <view class="notify-row">
        <view class="notify-left">
          <text class="notify-icon">铃</text>
          <view class="notify-text">
            <text class="notify-title">通知家长</text>
            <text class="notify-desc">发布后自动推送至家长端</text>
          </view>
        </view>
        <view class="switch" :class="{ on: form.notify_parents }" @click="toggleNotify">
          <view class="switch-dot" />
        </view>
      </view>
    </view>

    <!-- 表单卡片 — 更多设置 -->
    <view class="form-card">
      <view class="section-head">
        <view class="indicator is-purple" />
        <text class="section-title">更多设置</text>
      </view>

      <!-- 封面图片 -->
      <view class="field">
        <view class="field-label">
          <text class="label-text">封面图片</text>
          <text class="label-optional">（选填）</text>
        </view>
        <view class="upload-area" @click="uploadCover">
          <image v-if="form.cover" :src="form.cover" mode="aspectFill" class="cover-preview" />
          <template v-else>
            <text class="upload-icon">图</text>
            <text class="upload-text">点击上传封面图片</text>
            <text class="upload-hint">支持 JPG、PNG，大小不超过 5MB</text>
          </template>
        </view>
      </view>

      <!-- 附件 -->
      <view class="field">
        <view class="field-label">
          <text class="label-text">附件</text>
          <text class="label-optional">（选填）</text>
        </view>
        <view class="attach-btn" @click="addAttachment">
          <text class="attach-icon">附</text>
          <text class="attach-text">添加附件（文档、图片等）</text>
          <text class="attach-arrow">▸</text>
        </view>
        <view v-if="form.attachments.length" class="attach-list">
          <view v-for="(name, idx) in form.attachments" :key="idx" class="attach-item">
            <text>{{ name }}</text>
          </view>
        </view>
      </view>

      <!-- 重要程度 -->
      <view class="field">
        <view class="field-label">
          <text class="label-text">重要程度</text>
        </view>
        <view class="type-row">
          <view
            v-for="level in importanceLevels"
            :key="level.value"
            class="type-btn"
            :class="{ active: form.importance === level.value, 'is-important': level.value === 'important' }"
            @click="selectImportance(level.value)"
          >
            <text v-if="level.value === 'important'" class="importance-star">★</text>
            <text>{{ level.label }}</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 底部操作 -->
    <view class="bottom-actions">
      <view class="draft-btn" @click="saveDraft">
        <text class="draft-icon">存</text>
        <text class="draft-text">保存草稿</text>
      </view>
      <button class="submit-btn" :loading="submitting" @click="submitForm">
        <text class="submit-icon">发</text>
        <text class="submit-text">立即发布</text>
      </button>
    </view>

    <view class="bottom-safe" />
  </view>
</template>

<style scoped lang="scss">
$blue-500: #3B82F6;
$blue-50: #EFF6FF;
$slate-50: #F8FAFC;
$slate-200: #E2E8F0;
$slate-300: #CBD5E1;
$slate-400: #94A3B8;
$slate-500: #64748B;
$slate-700: #334155;
$slate-800: #1E293B;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 24rpx 28rpx;
  background: $slate-50;
}

// ===== 表单卡片 =====
.form-card {
  padding: 40rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: 24rpx;
}

.section-head {
  display: flex;
  align-items: center;
  gap: 16rpx;
  margin-bottom: 32rpx;
}

.indicator {
  width: 10rpx;
  height: 40rpx;
  border-radius: 999rpx;
}

.indicator.is-blue   { background: $blue-500; }
.indicator.is-green  { background: #22C55E; }
.indicator.is-orange { background: #F97316; }
.indicator.is-purple { background: #8B5CF6; }

.section-title {
  font-size: 30rpx;
  font-weight: 700;
  color: $slate-800;
}

// ===== 字段 =====
.field {
  margin-bottom: 32rpx;

  &:last-child {
    margin-bottom: 0;
  }
}

.field-label {
  display: flex;
  align-items: center;
  gap: 8rpx;
  margin-bottom: 16rpx;
}

.label-text {
  font-size: 26rpx;
  font-weight: 500;
  color: $slate-700;
}

.label-required {
  font-size: 26rpx;
  color: #EF4444;
}

.label-optional {
  font-size: 24rpx;
  color: $slate-400;
}

.input-box {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 32rpx;
  border: 2rpx solid $slate-200;
  border-radius: 24rpx;
  background: $slate-50;
}

.input-box.picker {
  justify-content: space-between;

  &.half {
    flex: 1;
  }
}

.input-icon {
  flex-shrink: 0;
  font-size: 30rpx;
  font-weight: 700;
  color: $slate-400;
  margin-right: 16rpx;
}

.form-input {
  flex: 1;
  font-size: 28rpx;
  color: $slate-800;
}

.picker-value {
  font-size: 28rpx;
  color: $slate-800;
}

.picker-chevron {
  font-size: 26rpx;
  color: $slate-400;
  margin-left: 16rpx;
}

.textarea-box {
  padding: 24rpx 32rpx;
  border: 2rpx solid $slate-200;
  border-radius: 24rpx;
  background: $slate-50;
  min-height: 192rpx;
}

.form-textarea {
  width: 100%;
  min-height: 140rpx;
  font-size: 28rpx;
  color: $slate-800;
}

// ===== 类型选择 =====
.type-row {
  display: flex;
  gap: 16rpx;
}

.type-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8rpx;
  height: 88rpx;
  border: 2rpx solid $slate-200;
  border-radius: 24rpx;
  background: $slate-50;

  text {
    font-size: 26rpx;
    color: $slate-500;
  }
}

.type-btn.active {
  border-color: $blue-500;
  background: $blue-50;

  text { color: $blue-500; font-weight: 600; }
}

.type-btn.active.is-important {
  border-color: #F59E0B;
  background: #FFFBEB;

  text { color: #F59E0B; }
}

.importance-star {
  font-size: 28rpx;
  color: #F59E0B;
}

// ===== 时间范围 =====
.time-range {
  display: flex;
  align-items: center;
  gap: 24rpx;
}

.time-sep {
  font-size: 28rpx;
  color: $slate-400;
  flex-shrink: 0;
}

// ===== 班级选择 =====
.class-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 16rpx;
}

.class-chip {
  display: flex;
  align-items: center;
  gap: 8rpx;
  padding: 14rpx 24rpx;
  border-radius: 16rpx;
  background: $blue-50;
}

.chip-name {
  font-size: 24rpx;
  font-weight: 500;
  color: $blue-500;
}

.chip-remove {
  font-size: 28rpx;
  color: $blue-500;
}

.class-chip.add {
  border: 2rpx dashed $slate-300;
  background: transparent;
}

.chip-add-icon {
  font-size: 28rpx;
  color: $slate-400;
}

.chip-add-text {
  font-size: 24rpx;
  color: $slate-400;
}

// ===== 通知开关 =====
.notify-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 24rpx;
  padding-top: 24rpx;
  border-top: 2rpx solid #F1F5F9;
}

.notify-left {
  display: flex;
  align-items: center;
  gap: 16rpx;
}

.notify-icon {
  font-size: 32rpx;
  font-weight: 700;
  color: $slate-500;
}

.notify-text {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.notify-title {
  font-size: 28rpx;
  font-weight: 500;
  color: $slate-700;
}

.notify-desc {
  font-size: 24rpx;
  color: $slate-400;
}

.switch {
  position: relative;
  width: 96rpx;
  height: 56rpx;
  border-radius: 999rpx;
  background: $slate-300;
  padding: 8rpx;
  transition: background 0.2s;
}

.switch.on {
  background: $blue-500;
}

.switch-dot {
  width: 40rpx;
  height: 40rpx;
  border-radius: 999rpx;
  background: #fff;
  transition: transform 0.2s;

  .switch.on & {
    transform: translateX(40rpx);
  }
}

// ===== 上传 =====
.upload-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  height: 224rpx;
  border: 4rpx dashed $slate-300;
  border-radius: 24rpx;
  background: $slate-50;
  overflow: hidden;
}

.cover-preview {
  width: 100%;
  height: 100%;
}

.upload-icon {
  font-size: 56rpx;
  font-weight: 700;
  color: $slate-400;
}

.upload-text {
  font-size: 26rpx;
  color: $slate-400;
}

.upload-hint {
  font-size: 22rpx;
  color: $slate-300;
}

.attach-btn {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 32rpx;
  border: 2rpx solid $slate-200;
  border-radius: 24rpx;
  background: $slate-50;
}

.attach-icon {
  font-size: 30rpx;
  font-weight: 700;
  color: $slate-400;
  margin-right: 16rpx;
}

.attach-text {
  flex: 1;
  font-size: 28rpx;
  color: $slate-400;
}

.attach-arrow {
  font-size: 28rpx;
  color: $slate-400;
}

.attach-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
  margin-top: 16rpx;
}

.attach-item {
  padding: 8rpx 20rpx;
  border-radius: 12rpx;
  background: $blue-50;

  text {
    font-size: 24rpx;
    color: $blue-500;
  }
}

// ===== 底部操作 =====
.bottom-actions {
  display: flex;
  gap: 24rpx;
  margin-top: 8rpx;
}

.draft-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  height: 96rpx;
  border: 2rpx solid $slate-200;
  border-radius: 24rpx;
  background: #fff;
}

.draft-icon {
  font-size: 32rpx;
  color: $slate-500;
}

.draft-text {
  font-size: 30rpx;
  font-weight: 500;
  color: $slate-500;
}

.submit-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  height: 96rpx;
  border-radius: 24rpx;
  background: $blue-500;
  box-shadow: 0 8rpx 24rpx rgba(59, 130, 246, 0.3);
  border: none;
}

.submit-icon {
  font-size: 32rpx;
  color: #fff;
}

.submit-text {
  font-size: 30rpx;
  font-weight: 700;
  color: #fff;
}

.bottom-safe {
  height: calc(60rpx + env(safe-area-inset-bottom));
}
</style>
