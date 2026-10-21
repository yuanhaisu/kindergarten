<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { createTeacherTask, fetchTeacherTasks } from "@/api/teacher-tasks"
import { fetchTeacherClassStudents } from "@/api/teacher-class"

function ymd(d) {
  const p = n => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}
function addDays(n) {
  const d = new Date()
  d.setDate(d.getDate() + n)
  return ymd(d)
}

function defaultForm() {
  return {
    title: "", description: "", learning_link: "", frequency: "once",
    start_date: ymd(new Date()), end_date: addDays(7),
    target_student_id: 0, allow_public: false,
    recognition_ritual: "sticker", star_scope: "task", star_goal: 0
  }
}

const tasks = ref([])
const students = ref([])
const form = ref(defaultForm())
const studentOptions = computed(() => ["班级", ...students.value.map(s => s.name)])
const studentIndex = ref(0)
const freqOptions = ["一次性", "每日", "每周"]
const freqValues = ["once", "daily", "weekly"]
const freqIndex = ref(0)
const ritualOptions = ["贴纸", "语音", "班级星星"]
const ritualValues = ["sticker", "voice", "class_star"]
const ritualIndex = ref(0)

async function loadStudents() {
  try {
    const res = await fetchTeacherClassStudents()
    students.value = res.data?.list || []
  } catch (e) {
    students.value = []
  }
}

async function loadTasks() {
  const res = await fetchTeacherTasks({ page: 1, page_size: 20 })
  tasks.value = res.data?.list || []
  uni.stopPullDownRefresh()
}

async function addTask() {
  if (!form.value.title.trim() || !form.value.learning_link.trim()) {
    uni.showToast({ title: "请填写任务标题和学习关联", icon: "none" })
    return
  }
  if (form.value.end_date < form.value.start_date) {
    uni.showToast({ title: "结束日期不能早于开始日期", icon: "none" })
    return
  }
  if (form.value.recognition_ritual === "class_star" && (!Number(form.value.star_goal) || Number(form.value.star_goal) <= 0)) {
    uni.showToast({ title: "选择班级星星时请设置星星目标", icon: "none" })
    return
  }
  try {
    await createTeacherTask({
      ...form.value,
      star_goal: Number(form.value.star_goal) || 0
    })
    form.value = defaultForm()
    studentIndex.value = 0
    freqIndex.value = 0
    ritualIndex.value = 0
    uni.showToast({ title: "已创建", icon: "success" })
    loadTasks()
  } catch (e) {
    uni.showToast({ title: e?.message || "创建失败", icon: "none" })
  }
}

function onStudentChange(e) {
  studentIndex.value = Number(e.detail.value)
  form.value.target_student_id = studentIndex.value === 0 ? 0 : students.value[studentIndex.value - 1].student_id
  if (form.value.target_student_id > 0) form.value.allow_public = false
}
function onFreqChange(e) {
  freqIndex.value = Number(e.detail.value)
  form.value.frequency = freqValues[freqIndex.value]
}
function onRitualChange(e) {
  ritualIndex.value = Number(e.detail.value)
  form.value.recognition_ritual = ritualValues[ritualIndex.value]
}

function goDetail(id) { uni.navigateTo({ url: `/pages/tasks/detail?id=${id}` }) }

onLoad(() => { loadStudents(); loadTasks() })
onPullDownRefresh(loadTasks)
</script>

<template>
  <view class="page">
    <view class="card form">
      <text class="title">亲子任务</text>
      <input v-model="form.title" class="input" placeholder="任务标题" />
      <textarea v-model="form.description" class="textarea" placeholder="任务说明" />
      <input v-model="form.learning_link" class="input" placeholder="和幼儿园学习的关系（必填）" />
      <picker :range="studentOptions" @change="onStudentChange">
        <view class="input picker">范围：{{ studentOptions[studentIndex] }}</view>
      </picker>
      <picker :range="freqOptions" @change="onFreqChange">
        <view class="input picker">频率：{{ freqOptions[freqIndex] }}</view>
      </picker>
      <picker mode="date" :value="form.start_date" @change="form.start_date = $event.detail.value">
        <view class="input picker">开始日期：{{ form.start_date }}</view>
      </picker>
      <picker mode="date" :value="form.end_date" @change="form.end_date = $event.detail.value">
        <view class="input picker">结束日期：{{ form.end_date }}</view>
      </picker>
      <picker :range="ritualOptions" @change="onRitualChange">
        <view class="input picker">回应仪式：{{ ritualOptions[ritualIndex] }}</view>
      </picker>
      <view class="input picker">星星罐：按任务（班级罐后续版本开放）</view>
      <input v-model.number="form.star_goal" class="input" type="number" placeholder="星星目标（0=不启用）" />
      <view v-if="form.target_student_id === 0" class="switch-row">
        <text>允许申请公开</text>
        <switch :checked="form.allow_public" @change="form.allow_public = $event.detail.value" />
      </view>
      <button class="submit" @click="addTask">创建任务</button>
    </view>
    <view v-for="task in tasks" :key="task.id" class="card" @click="goDetail(task.id)">
      <text class="task-title">{{ task.title }}</text>
      <text class="meta">{{ task.frequency }} · 待回应 {{ task.pending_responses || 0 }}</text>
      <text class="desc">{{ task.description }}</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 28rpx; background: $teacher-bg; }
.card { margin-bottom: 20rpx; padding: 28rpx; border-radius: 30rpx; background: $teacher-card; box-shadow: $teacher-shadow; }
.title, .task-title { display: block; color: $teacher-text; font-size: $typography-font-size-xl; font-weight: 800; }
.input, .textarea { box-sizing: border-box; width: 100%; margin-top: 18rpx; border-radius: 22rpx; background: #fff; font-size: $typography-font-size-md; }
.input { min-height: 84rpx; padding: 0 $spacing-lg; }
.textarea { min-height: 180rpx; padding: $spacing-lg; }
.submit { min-height: 84rpx; margin-top: 18rpx; border-radius: 22rpx; background: $teacher-green; color: #fff; line-height: 84rpx; }
.picker { display: flex; align-items: center; }
.switch-row { display: flex; align-items: center; justify-content: space-between; margin-top: 18rpx; color: $teacher-text; font-size: $typography-font-size-sm; }
.meta, .desc { display: block; margin-top: 10rpx; color: $teacher-muted; font-size: $typography-font-size-sm; }
</style>
