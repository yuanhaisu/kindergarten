<script setup>
import { ref, computed } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { createTeacherGrowthLog } from "@/api/teacher-growth"
import { fetchTeacherClassStudents } from "@/api/teacher-class"
import { fetchDict } from "@/api/dict"

function todayStr() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`
}

function formatDate(ymd) {
  const d = new Date(ymd.replace(/-/g, "/"))
  const w = ["日", "一", "二", "三", "四", "五", "六"]
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日 星期${w[d.getDay()]}`
}

function addDays(ymd, n) {
  const d = new Date(ymd.replace(/-/g, "/"))
  d.setDate(d.getDate() + n)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const dd = String(d.getDate()).padStart(2, "0")
  return `${y}-${m}-${dd}`
}

const submitting = ref(false)
const students = ref([])
const form = ref({
  date: todayStr(),
  breakfast_rating: 0,
  lunch_rating: 0,
  dinner_rating: 0,
  nap_duration: 0,
  sleep_quality: 0,
  morning_mood: 0,
  afternoon_mood: 0,
  performance_tags: [],
  content: "",
  media: [],
  sync_to_feed: true,
  target_type: 1,
  student_ids: []
})

const dateDisplay = computed(() => formatDate(form.value.date))

function prevDay() { form.value.date = addDays(form.value.date, -1) }
function nextDay() {
  const n = addDays(form.value.date, 1)
  if (n <= todayStr()) form.value.date = n
}

function toggleTag(tag) {
  const i = form.value.performance_tags.indexOf(tag)
  if (i >= 0) form.value.performance_tags.splice(i, 1)
  else form.value.performance_tags.push(tag)
}

function toggleStudent(sid) {
  const i = form.value.student_ids.indexOf(sid)
  if (i >= 0) form.value.student_ids.splice(i, 1)
  else form.value.student_ids.push(sid)
}

function switchTargetType() {
  form.value.target_type = form.value.target_type === 2 ? 1 : 2
  if (form.value.target_type === 1) {
    form.value.student_ids = []
    form.value.sync_to_feed = true
  } else {
    form.value.sync_to_feed = false
    loadStudents()
  }
}

async function loadStudents() {
  try {
    const res = await fetchTeacherClassStudents()
    students.value = res.data?.list || []
  } catch (e) {
    // 学生列表加载失败不阻塞页面
  }
}

onLoad(() => {
  loadStudents()
  loadPerformanceTags()
})

function addMedia() {
  uni.chooseImage({
    count: 20 - form.value.media.length,
    success(res) { form.value.media.push(...res.tempFilePaths) }
  })
}
function removeMedia(i) { form.value.media.splice(i, 1) }

async function publish() {
  if (!form.value.content.trim()) {
    uni.showToast({ title: "请输入记录内容", icon: "none" })
    return
  }
  if (form.value.target_type === 2 && form.value.student_ids.length === 0) {
    uni.showToast({ title: "请选择至少一位学生", icon: "none" })
    return
  }
  submitting.value = true
  try {
    await createTeacherGrowthLog(form.value)
    uni.showToast({ title: "已发布", icon: "success" })
    setTimeout(() => uni.navigateBack(), 600)
  } catch (e) {
    uni.showToast({ title: e?.message || "发布失败", icon: "none" })
  } finally {
    submitting.value = false
  }
}

const sleepOptions = [1, 2, 3]
const qualityOptions = [
  { value: 3, label: "很好" },
  { value: 2, label: "一般" },
  { value: 1, label: "不好" }
]

const performanceOptions = ref([])

async function loadPerformanceTags() {
  try {
    const res = await fetchDict("performance_tag")
    performanceOptions.value = (res.data?.items || []).map(i => i.item_label)
  } catch (e) {
    // fallback to defaults
    performanceOptions.value = [
      "乐于助人", "积极发言", "遵守纪律", "动手能力强",
      "专注力强", "有礼貌", "爱干净", "团结友善", "勇于尝试"
    ]
  }
}
</script>

<template>
  <view class="page">
    <scroll-view class="body" scroll-y>
      <!-- 日期选择栏 -->
      <view class="date-bar">
        <view class="date-arrow" @click="prevDay"><text>◀</text></view>
        <text class="date-text">{{ dateDisplay }}</text>
        <view class="date-arrow" :class="{ disabled: form.date >= todayStr() }" @click="nextDay">
          <text>▶</text>
        </view>
      </view>

      <!-- 快速记录模板 — 仅班级记录 -->
      <view v-if="form.target_type === 1" class="section">
        <text class="section-title">快速记录</text>

        <view class="template-row">
          <!-- 饮食 -->
          <view class="template-card">
            <view class="card-header">
              <view class="card-icon" style="background:rgba(220,252,231,1)">🍽️</view>
              <text class="card-title">饮食</text>
            </view>
            <view class="card-body">
              <!-- 早餐 -->
              <view class="meal-row">
                <text class="meal-label">早餐</text>
                <view class="emoji-row">
                  <view class="emoji-btn" :style="{ background: form.breakfast_rating === 3 ? '#dcfce7' : '#f1f5f9' }" @click="form.breakfast_rating = form.breakfast_rating === 3 ? 0 : 3">😋</view>
                  <view class="emoji-btn" :style="{ background: form.breakfast_rating === 2 ? '#dcfce7' : '#f1f5f9' }" @click="form.breakfast_rating = form.breakfast_rating === 2 ? 0 : 2">😐</view>
                  <view class="emoji-btn" :style="{ background: form.breakfast_rating === 1 ? '#dcfce7' : '#f1f5f9' }" @click="form.breakfast_rating = form.breakfast_rating === 1 ? 0 : 1">😞</view>
                </view>
              </view>
              <!-- 午餐 -->
              <view class="meal-row">
                <text class="meal-label">午餐</text>
                <view class="emoji-row">
                  <view class="emoji-btn" :style="{ background: form.lunch_rating === 3 ? '#dcfce7' : '#f1f5f9' }" @click="form.lunch_rating = form.lunch_rating === 3 ? 0 : 3">😋</view>
                  <view class="emoji-btn" :style="{ background: form.lunch_rating === 2 ? '#dcfce7' : '#f1f5f9' }" @click="form.lunch_rating = form.lunch_rating === 2 ? 0 : 2">😐</view>
                  <view class="emoji-btn" :style="{ background: form.lunch_rating === 1 ? '#dcfce7' : '#f1f5f9' }" @click="form.lunch_rating = form.lunch_rating === 1 ? 0 : 1">😞</view>
                </view>
              </view>
              <!-- 晚餐 -->
              <view class="meal-row">
                <text class="meal-label">晚餐</text>
                <view class="emoji-row">
                  <view class="emoji-btn" :style="{ background: form.dinner_rating === 3 ? '#dcfce7' : '#f1f5f9' }" @click="form.dinner_rating = form.dinner_rating === 3 ? 0 : 3">😋</view>
                  <view class="emoji-btn" :style="{ background: form.dinner_rating === 2 ? '#dcfce7' : '#f1f5f9' }" @click="form.dinner_rating = form.dinner_rating === 2 ? 0 : 2">😐</view>
                  <view class="emoji-btn" :style="{ background: form.dinner_rating === 1 ? '#dcfce7' : '#f1f5f9' }" @click="form.dinner_rating = form.dinner_rating === 1 ? 0 : 1">😞</view>
                </view>
              </view>
            </view>
          </view>

          <!-- 睡眠 -->
          <view class="template-card">
            <view class="card-header">
              <view class="card-icon" style="background:rgba(243,232,255,1)">💤</view>
              <text class="card-title">睡眠</text>
            </view>
            <view class="card-body">
              <text class="field-label">午睡时长</text>
              <view class="chip-row">
                <view
                  v-for="h in sleepOptions" :key="h"
                  class="chip"
                  :class="{ active: form.nap_duration === h }"
                  @click="form.nap_duration = form.nap_duration === h ? 0 : h"
                >{{ h }}小时</view>
              </view>
              <text class="field-label" style="margin-top:12rpx">睡眠质量</text>
              <view class="chip-row">
                <view
                  v-for="q in qualityOptions" :key="q.value"
                  class="chip"
                  :class="{ active: form.sleep_quality === q.value }"
                  @click="form.sleep_quality = form.sleep_quality === q.value ? 0 : q.value"
                >{{ q.label }}</view>
              </view>
            </view>
          </view>
        </view>

        <view class="template-row">
          <!-- 情绪 -->
          <view class="template-card">
            <view class="card-header">
              <view class="card-icon" style="background:rgba(255,237,213,1)">😊</view>
              <text class="card-title">情绪</text>
            </view>
            <view class="card-body">
<!-- 上午 -->
              <view class="meal-row">
                <text class="meal-label">上午</text>
                <view class="emoji-row">
                  <view class="emoji-btn" :style="{ background: form.morning_mood === 3 ? '#dcfce7' : '#f1f5f9' }" @click="form.morning_mood = form.morning_mood === 3 ? 0 : 3">😋</view>
                  <view class="emoji-btn" :style="{ background: form.morning_mood === 2 ? '#dcfce7' : '#f1f5f9' }" @click="form.morning_mood = form.morning_mood === 2 ? 0 : 2">😐</view>
                  <view class="emoji-btn" :style="{ background: form.morning_mood === 1 ? '#dcfce7' : '#f1f5f9' }" @click="form.morning_mood = form.morning_mood === 1 ? 0 : 1">😞</view>
                </view>
              </view>
              <!-- 下午 -->
              <view class="meal-row">
                <text class="meal-label">下午</text>
                <view class="emoji-row">
                  <view class="emoji-btn" :style="{ background: form.afternoon_mood === 3 ? '#dcfce7' : '#f1f5f9' }" @click="form.afternoon_mood = form.afternoon_mood === 3 ? 0 : 3">😋</view>
                  <view class="emoji-btn" :style="{ background: form.afternoon_mood === 2 ? '#dcfce7' : '#f1f5f9' }" @click="form.afternoon_mood = form.afternoon_mood === 2 ? 0 : 2">😐</view>
                  <view class="emoji-btn" :style="{ background: form.afternoon_mood === 1 ? '#dcfce7' : '#f1f5f9' }" @click="form.afternoon_mood = form.afternoon_mood === 1 ? 0 : 1">😞</view>
                </view>
              </view>
            </view>
          </view>

          <!-- 表现 -->
          <view class="template-card">
            <view class="card-header">
              <view class="card-icon" style="background:rgba(254,226,226,1)">🌟</view>
              <text class="card-title">表现</text>
            </view>
            <view class="card-body">
              <view class="tag-cloud">
                <view
                  v-for="t in performanceOptions" :key="t"
                  class="tag"
                  :class="{ active: form.performance_tags.includes(t) }"
                  @click="toggleTag(t)"
                >{{ t }}</view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <!-- 指定学生模式 — 学生选择器 + 表现标签 -->
      <view v-if="form.target_type === 2" class="section">
        <text class="section-title">选择学生</text>
        <view class="student-picker">
          <view
            v-for="s in students"
            :key="s.id"
            class="student-chip"
            :class="{ selected: form.student_ids.includes(s.id) }"
            @click="toggleStudent(s.id)"
          >
            <text>{{ s.name }}</text>
          </view>
          <view v-if="students.length === 0" class="student-empty">
            <text>暂未加载学生列表</text>
          </view>
        </view>

        <text class="section-title" style="margin-top: 32rpx">表现标签</text>
        <view class="tag-cloud">
          <view
            v-for="t in performanceOptions" :key="t"
            class="tag"
            :class="{ active: form.performance_tags.includes(t) }"
            @click="toggleTag(t)"
          >{{ t }}</view>
        </view>
      </view>

      <!-- 照片/视频上传 -->
      <view class="section">
        <text class="section-title">照片/视频上传</text>
        <view class="upload-card">
          <view class="media-grid">
            <view class="media-item add-btn" @click="addMedia">
              <text class="add-icon">＋</text>
              <text class="add-text">添加</text>
            </view>
            <view v-for="(img, i) in form.media" :key="i" class="media-item">
              <image :src="img" mode="aspectFill" class="media-img" />
              <view class="media-remove" @click="removeMedia(i)">✕</view>
            </view>
          </view>
          <view class="upload-hint">
            <text>ℹ️</text>
            <text>最多可上传20张照片/5个视频，系统将自动识别孩子并推送</text>
          </view>
        </view>
      </view>

      <!-- 记录内容 -->
      <view class="section">
        <text class="section-title">记录内容</text>
        <view class="content-card">
          <textarea
            v-model="form.content"
            class="content-area"
            placeholder="请输入记录内容，描述今天孩子们的活动和表现..."
          />
        </view>
      </view>

      <!-- 底部留白 -->
      <view style="height:140rpx"></view>
    </scroll-view>

    <!-- 底部操作栏 -->
    <view class="bottom-bar">
      <view class="bottom-options">
        <view v-if="form.target_type === 1" class="option-row" @click="form.sync_to_feed = !form.sync_to_feed">
          <text class="option-check">{{ form.sync_to_feed ? '☑' : '☐' }}</text>
          <text class="option-label active">同步发送到班级动态</text>
        </view>
        <view v-if="form.target_type === 2" class="option-hint">
          <text>该记录仅被选中的学生家长可见</text>
        </view>
        <view class="option-row" @click="switchTargetType">
          <text class="option-check">{{ form.target_type === 2 ? '☑' : '☐' }}</text>
          <text class="option-label" :class="{ active: form.target_type === 2 }">仅发送给指定家长</text>
        </view>
      </view>
      <button class="publish-btn" :loading="submitting" @click="publish">
        {{ submitting ? "发布中..." : (form.target_type === 2 ? "发布个体记录" : "发布班级记录") }}
      </button>
    </view>
  </view>
</template>

<style scoped lang="scss">

.page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: $color-bg-card;
}

.body { flex: 1; padding-top: 16rpx; }

/* 日期栏 */
.date-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-xl 48rpx;
  background: $color-bg-card;
}
.date-arrow { padding: $spacing-xs; }
.date-arrow.disabled { opacity: 0.3; }
.date-text { font-size: $typography-font-size-xl; font-weight: 700; color: $color-text-primary; }

/* 通用 section */
.section { padding: $spacing-xl 48rpx; }
.section-title {
  display: block;
  margin-bottom: 24rpx;
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

/* 模板卡片 */
.template-row {
  display: flex;
  gap: $spacing-lg;
  margin-bottom: 24rpx;
}
.template-card {
  flex: 1;
  padding: $spacing-xl;
  background: $color-bg-card;
  border: 1rpx solid $color-bg-page;
  border-radius: $border-radius-xl;
  box-shadow: 0 2rpx 4rpx rgba(0,0,0,0.05);
}
.card-header {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: 16rpx;
}
.card-icon {
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-lg;
  font-size: $typography-font-size-2xl;
}
.card-title { font-size: $typography-font-size-md; font-weight: 700; color: $color-text-primary; }
.card-body { display: flex; flex-direction: column; gap: $spacing-xs; }
.meal-row { display: flex; align-items: center; justify-content: space-between; }
.meal-label { font-size: $typography-font-size-base; color: $color-text-muted; width: 54rpx; }
.emoji-row { display: flex; gap: $spacing-xs; }
.emoji-btn {
  width: 48rpx;
  height: 48rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-sm;
  font-size: $typography-font-size-md;
}
.field-label { font-size: $typography-font-size-base; color: $color-text-muted; }
.chip-row { display: flex; gap: $spacing-md; flex-wrap: wrap; }
.chip {
  padding: $spacing-sm 24rpx;
  background: $color-bg-page;
  border-radius: $border-radius-lg;
  font-size: $typography-font-size-base;
  color: $color-text-muted;
  &.active { background: #eff6ff; color: $color-semantic-info; font-weight: 600; }
}
.tag-cloud { display: flex; flex-wrap: wrap; gap: $spacing-sm; }
.tag {
  padding: $spacing-xs 16rpx;
  background: $color-bg-page;
  border-radius: $border-radius-sm;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  &.active { background: #eff6ff; color: $color-semantic-info; font-weight: 600; }
}

/* 上传 */
.upload-card {
  padding: $spacing-xl;
  background: $color-bg-card;
  border: 1rpx solid $color-bg-page;
  border-radius: $border-radius-xl;
  box-shadow: 0 2rpx 4rpx rgba(0,0,0,0.05);
}
.media-grid {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-lg;
}
.media-item {
  width: 160rpx;
  height: 160rpx;
  border-radius: $border-radius-xl;
  overflow: hidden;
  position: relative;
}
.add-btn {
  border: 1rpx dashed #cbd5e1;
  background: $color-bg-card;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
}
.add-icon { font-size: 48rpx; color: $color-text-hint; }
.add-text { font-size: $typography-font-size-sm; color: $color-text-hint; }
.media-img { width: 100%; height: 100%; }
.media-remove {
  position: absolute;
  top: 8rpx;
  right: 8rpx;
  width: 40rpx;
  height: 40rpx;
  background: rgba(0,0,0,0.5);
  color: #fff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: $typography-font-size-sm;
}
.upload-hint {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: 24rpx;
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

/* 记录内容 */
.content-card {
  padding: $spacing-xl;
  background: $color-bg-card;
  border: 1rpx solid $color-bg-page;
  border-radius: $border-radius-xl;
  box-shadow: 0 2rpx 4rpx rgba(0,0,0,0.05);
}
.content-area {
  width: 100%;
  height: 240rpx;
  font-size: $typography-font-size-lg;
  color: $color-text-primary;
  box-sizing: border-box;
}

/* 底部操作栏 */
.bottom-bar {
  padding: $spacing-xl 48rpx 48rpx 48rpx;
  background: $color-bg-card;
}
.bottom-options {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  margin-bottom: 24rpx;
}
.option-row { display: flex; align-items: center; gap: $spacing-md; }
.option-check { font-size: $typography-font-size-md; color: $color-semantic-info; width: 40rpx; }
.option-label { font-size: $typography-font-size-md; color: $color-text-hint; &.active { color: $color-semantic-info; } }
.publish-btn {
  width: 100%;
  height: 96rpx;
  background: linear-gradient(135deg, $color-semantic-info, $color-semantic-info);
  color: #fff;
  font-size: $typography-font-size-xl;
  font-weight: 700;
  border-radius: $border-radius-xl;
  border: 0;
  line-height: 96rpx;
}

/* 学生选择器 */
.student-picker {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;
}

.student-chip {
  padding: 14rpx 32rpx;
  border-radius: 999rpx;
  background: $color-bg-page;
  font-size: $typography-font-size-md;
  color: $color-text-muted;
  transition: all 0.2s;

  &.selected {
    background: #eff6ff;
    color: $color-semantic-info;
    font-weight: 600;
  }
}

.student-empty {
  width: 100%;
  text-align: center;
  padding: $spacing-xl;
  font-size: $typography-font-size-base;
  color: $color-text-hint;
}

.option-hint {
  padding: $spacing-sm 24rpx;
  margin-bottom: 8rpx;
  border-radius: $border-radius-md;
  background: #eff6ff;
  font-size: $typography-font-size-sm;
  color: $color-semantic-info;
}
</style>
