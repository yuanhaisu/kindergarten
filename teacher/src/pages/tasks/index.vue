<script setup>
import { ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { createTeacherTask, fetchTeacherTasks } from "@/api/teacher-tasks"

const tasks = ref([])
const form = ref({ title: "", description: "", frequency: "once", points: 1 })

async function loadTasks() {
  const res = await fetchTeacherTasks({ page: 1, page_size: 20 })
  tasks.value = res.data?.list || []
  uni.stopPullDownRefresh()
}

async function addTask() {
  if (!form.value.title.trim()) {
    uni.showToast({ title: "请输入任务标题", icon: "none" })
    return
  }
  await createTeacherTask(form.value)
  form.value.title = ""
  form.value.description = ""
  uni.showToast({ title: "已创建", icon: "success" })
  loadTasks()
}

onLoad(loadTasks)
onPullDownRefresh(loadTasks)
</script>

<template>
  <view class="page">
    <view class="card form">
      <text class="title">亲子任务</text>
      <input v-model="form.title" class="input" placeholder="任务标题" />
      <textarea v-model="form.description" class="textarea" placeholder="任务说明" />
      <button class="submit" @click="addTask">创建任务</button>
    </view>
    <view v-for="task in tasks" :key="task.id" class="card">
      <text class="task-title">{{ task.title }}</text>
      <text class="meta">{{ task.frequency }} · {{ task.points }} 积分</text>
      <text class="desc">{{ task.description }}</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 28rpx; background: $teacher-bg; }
.card { margin-bottom: 20rpx; padding: 28rpx; border-radius: 30rpx; background: $teacher-card; box-shadow: $teacher-shadow; }
.title, .task-title { display: block; color: $teacher-text; font-size: 32rpx; font-weight: 800; }
.input, .textarea { box-sizing: border-box; width: 100%; margin-top: 18rpx; border-radius: 22rpx; background: #fff; font-size: 28rpx; }
.input { min-height: 84rpx; padding: 0 24rpx; }
.textarea { min-height: 180rpx; padding: 24rpx; }
.submit { min-height: 84rpx; margin-top: 18rpx; border-radius: 22rpx; background: $teacher-green; color: #fff; line-height: 84rpx; }
.meta, .desc { display: block; margin-top: 10rpx; color: $teacher-muted; font-size: 24rpx; }
</style>
