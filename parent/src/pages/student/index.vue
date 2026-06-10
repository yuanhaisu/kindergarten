<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchStudentProfile, fetchStudents, unbindStudent } from "@/api/parent-student"

const profile = ref({})
const students = ref([])
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const [p, s] = await Promise.all([fetchStudentProfile(1), fetchStudents()])
    profile.value = p.data || {}
    students.value = s.data?.list || []
  } finally { loading.value = false }
}

async function handleUnbind(student) {
  const r = await new Promise(r => uni.showModal({ title: "确认解绑", content: `确定解除与 ${student.name} 的绑定吗?`, success: s => r(s.confirm) }))
  if (!r) return
  try {
    await unbindStudent(student.id)
    uni.showToast({ title: "已解绑", icon: "success" })
    loadData()
  } catch (err) {
    uni.showToast({ title: err?.message || "解绑失败", icon: "none" })
  }
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>
    <template v-else>
      <view class="card">
        <text class="title">{{ profile.name || "孩子信息" }}</text>
        <view class="row"><text class="key">班级</text><text class="val">{{ profile.class_name }}</text></view>
        <view class="row"><text class="key">性别</text><text class="val">{{ profile.gender === 1 ? '男' : '女' }}</text></view>
        <view class="row"><text class="key">生日</text><text class="val">{{ profile.birthday }}</text></view>
        <view class="row"><text class="key">年龄</text><text class="val">{{ profile.age }}</text></view>
        <view v-if="profile.health_note" class="row"><text class="key">健康备注</text><text class="val">{{ profile.health_note }}</text></view>
      </view>

      <view v-if="students.length > 1" class="card">
        <text class="section-title">已绑定的孩子</text>
        <view v-for="s in students" :key="s.id" class="student-item">
          <text class="s-name">{{ s.name }} ({{ s.class_name }})</text>
          <button class="unbind-btn" @click="handleUnbind(s)">解绑</button>
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx; background: $parent-bg; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $parent-muted; font-size: 28rpx; }
.card { padding: 28rpx; margin-bottom: 18rpx; border-radius: 24rpx; background: $parent-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.title { display: block; margin-bottom: 18rpx; color: $parent-text; font-size: 32rpx; font-weight: 800; }
.section-title { display: block; margin-bottom: 18rpx; color: $parent-text; font-size: 28rpx; font-weight: 700; }
.row { display: flex; justify-content: space-between; padding: 16rpx 0; border-bottom: 1rpx solid rgba(106,77,42,0.06); }
.key { color: $parent-muted; font-size: 26rpx; }
.val { color: $parent-text; font-size: 26rpx; }
.student-item { display: flex; justify-content: space-between; align-items: center; padding: 18rpx 0; border-bottom: 1rpx solid rgba(106,77,42,0.06); }
.s-name { color: $parent-text; font-size: 26rpx; }
.unbind-btn { padding: 8rpx 20rpx; border-radius: 999rpx; background: #fff0ef; color: #e6534f; font-size: 22rpx; }
</style>
