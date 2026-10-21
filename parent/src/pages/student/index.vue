<script setup>
import { ref } from "vue"
import { onLoad, onShow } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId, setCurrentStudentId } from "@/utils/currentStudent"
import { fetchStudentProfile, fetchStudents, unbindStudent } from "@/api/parent-student"
import { fetchHealthRecords } from "@/api/parent-health"

const profile = ref({})
const students = ref([])
const loading = ref(false)

const healthRecords = ref([])
const healthLoading = ref(false)
const healthError = ref("")

async function loadHealth() {
  const sid = getCurrentStudentId()
  if (!sid) {
    healthRecords.value = []
    return
  }
  healthLoading.value = true
  healthError.value = ""
  try {
    const res = await fetchHealthRecords(sid)
    healthRecords.value = res.data?.list || []
  } catch (err) {
    // 晨检加载失败不拖垮整页档案，但必须显式提示，不能静默留空
    healthError.value = err?.message || "晨检记录加载失败"
    healthRecords.value = []
  } finally {
    healthLoading.value = false
  }
}

function healthTone(status) {
  if (status === 1) return "warning"
  if (status === 2) return "danger"
  return "success"
}

function abnormalText(items) {
  if (!items || !items.length) return ""
  return items.map(i => `${i.part_label}${i.status_label}`).join("、")
}

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const [p, s] = await Promise.all([fetchStudentProfile(getCurrentStudentId()), fetchStudents()])
    profile.value = p.data || {}
    students.value = s.data?.list || []
    await loadHealth()
  } finally { loading.value = false }
}

// 从孩子选择/其他页返回时刷新；首次 onShow 紧随 onLoad，跳过
let skipNextShow = true
onShow(() => {
  if (skipNextShow) { skipNextShow = false; return }
  loadHealth()
})

async function handleUnbind(student) {
  const r = await new Promise(r => uni.showModal({ title: "确认解绑", content: `确定解除与 ${student.name} 的绑定吗?`, success: s => r(s.confirm) }))
  if (!r) return
  try {
    await unbindStudent(student.id)
    // 解绑的正好是当前孩子时，存储里会留下一个孤儿 id。
    // ResolveContext 对不属于该家长的 id 会静默回退到第一个孩子，
    // 界面看起来正常，用户却以为自己还在原来的孩子上。
    if (student.id === getCurrentStudentId()) {
      const next = students.value.find((s) => s.id !== student.id)
      setCurrentStudentId(next ? next.id : null)
    }
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

      <view class="card">
        <text class="section-title">最近晨检</text>
        <view v-if="healthLoading" class="state-inline">加载中...</view>
        <view v-else-if="healthError" class="state-inline is-error" @click="loadHealth">
          {{ healthError }}（点击重试）
        </view>
        <view v-else-if="healthRecords.length === 0" class="state-inline">近 7 天暂无晨检记录</view>
        <view v-else class="health-list">
          <view v-for="(r, idx) in healthRecords" :key="idx" class="health-item">
            <view class="health-head">
              <text class="health-date">{{ r.check_date }}</text>
              <text class="health-pill" :class="'is-' + healthTone(r.overall_status)">{{ r.overall_status_label }}</text>
            </view>
            <text v-if="r.abnormal_items && r.abnormal_items.length" class="health-abnormal">
              异常：{{ abnormalText(r.abnormal_items) }}
            </text>
            <text v-if="r.temperature" class="health-meta">体温 {{ r.temperature }}°C</text>
            <text v-if="r.is_isolated" class="health-isolated">已在隔离室等待家长</text>
            <text v-if="r.remark" class="health-remark">{{ r.remark }}</text>
          </view>
        </view>
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
.page { min-height: 100vh; padding: $spacing-lg 28rpx; background: $color-bg-page; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }
.card { padding: 28rpx; margin-bottom: 18rpx; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.title { display: block; margin-bottom: 18rpx; color: $color-text-primary; font-size: $typography-font-size-xl; font-weight: 800; }
.section-title { display: block; margin-bottom: 18rpx; color: $color-text-primary; font-size: $typography-font-size-md; font-weight: 700; }
.row { display: flex; justify-content: space-between; padding: $spacing-md 0; border-bottom: 1rpx solid rgba(106,77,42,0.06); }
.key { color: $color-text-muted; font-size: $typography-font-size-base; }
.val { color: $color-text-primary; font-size: $typography-font-size-base; }
.student-item { display: flex; justify-content: space-between; align-items: center; padding: 18rpx 0; border-bottom: 1rpx solid rgba(106,77,42,0.06); }
.s-name { color: $color-text-primary; font-size: $typography-font-size-base; }
.unbind-btn { padding: $spacing-xs 20rpx; border-radius: 999rpx; background: #fff0ef; color: #e6534f; font-size: $typography-font-size-xs; }

.state-inline { padding: $spacing-md 0; color: $color-text-hint; font-size: $typography-font-size-sm; }
.state-inline.is-error { color: $color-semantic-danger; }
.health-list { display: flex; flex-direction: column; gap: $spacing-md; }
.health-item { display: flex; flex-direction: column; gap: 6rpx; padding: $spacing-md; border-radius: $border-radius-lg; background: $color-bg-page; }
.health-head { display: flex; align-items: center; justify-content: space-between; }
.health-date { color: $color-text-primary; font-size: $typography-font-size-base; font-weight: 600; }
.health-pill { padding: 4rpx 16rpx; border-radius: $border-radius-full; font-size: $typography-font-size-xs; font-weight: 600; }
.health-pill.is-success { background: $color-semantic-success-bg; color: $color-semantic-success; }
.health-pill.is-warning { background: $color-semantic-warning-bg; color: $color-semantic-warning; }
.health-pill.is-danger { background: $color-semantic-danger-bg; color: $color-semantic-danger; }
.health-abnormal { color: $color-text-body; font-size: $typography-font-size-sm; }
.health-meta { color: $color-text-muted; font-size: $typography-font-size-sm; }
.health-isolated { color: $color-semantic-warning; font-size: $typography-font-size-sm; font-weight: 600; }
.health-remark { color: $color-text-muted; font-size: $typography-font-size-sm; line-height: 1.5; }
</style>
