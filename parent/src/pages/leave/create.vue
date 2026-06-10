<script setup>
import { ref } from "vue"
import { createLeave } from "@/api/parent-leave"

const form = ref({
  student_id: 1,
  reason: "",
  start_date: "",
  end_date: "",
  attachments: [],
  emergency_contact: { name: "", phone: "" }
})
const leaveTypes = [
  { key: "病假", label: "病假", desc: "因病需要休息" },
  { key: "事假", label: "事假", desc: "个人或家庭事务" },
  { key: "其他", label: "其他", desc: "其他原因" }
]
const activeType = ref("")
const loading = ref(false)

function selectType(t) {
  activeType.value = t.key
}

function pickDate(field) {
  uni.showModal({ title: "选择日期", content: "请在真实环境中使用日期选择器" })
}

async function handleSubmit() {
  if (!form.value.reason || !form.value.start_date || !form.value.end_date) {
    uni.showToast({ title: "请填写完整信息", icon: "none" })
    return
  }
  loading.value = true
  try {
    await createLeave(form.value)
    uni.showToast({ title: "提交成功", icon: "success" })
    setTimeout(() => uni.navigateBack(), 1000)
  } catch (err) {
    uni.showToast({ title: err?.message || "提交失败", icon: "none" })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <view class="page">
    <!-- 请假类型 -->
    <view class="card">
      <text class="section-title">请假类型</text>
      <view class="spacer-h16" />
      <view class="type-row">
        <view v-for="t in leaveTypes" :key="t.key"
          class="type-chip" :class="{ active: activeType === t.key }"
          @click="selectType(t)">
          <text class="type-label">{{ t.label }}</text>
          <text class="type-desc">{{ t.desc }}</text>
        </view>
      </view>
    </view>

    <view class="spacer-h16" />

    <!-- 时间 -->
    <view class="card">
      <text class="section-title">请假时间</text>
      <view class="spacer-h16" />
      <view class="date-row">
        <view class="date-item" @click="pickDate('start_date')">
          <text class="date-label">开始日期</text>
          <text class="date-val">{{ form.start_date || "请选择" }}</text>
        </view>
        <text class="date-sep">~</text>
        <view class="date-item" @click="pickDate('end_date')">
          <text class="date-label">结束日期</text>
          <text class="date-val">{{ form.end_date || "请选择" }}</text>
        </view>
      </view>
    </view>

    <view class="spacer-h16" />

    <!-- 原因 -->
    <view class="card">
      <text class="section-title">请假原因</text>
      <view class="spacer-h16" />
      <textarea v-model="form.reason" class="reason-input"
        placeholder="请详细描述请假原因..." :maxlength="500" />
    </view>

    <view class="spacer-h16" />

    <!-- 接送人 -->
    <view class="card">
      <text class="section-title">接送人信息</text>
      <view class="spacer-h16" />
      <input v-model="form.emergency_contact.name" class="input"
        placeholder="接送人姓名" />
      <view class="spacer-h12" />
      <input v-model="form.emergency_contact.phone" class="input"
        placeholder="接送人手机号" type="number" :maxlength="11" />
    </view>

    <!-- 提交 -->
    <view class="spacer-h32" />
    <button class="submit-btn" :disabled="loading" @click="handleSubmit">
      {{ loading ? "提交中..." : "提交申请" }}
    </button>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx 160rpx; background: $parent-bg; }
.spacer-h12 { height: 24rpx; }
.spacer-h16 { height: 32rpx; }
.spacer-h32 { height: 64rpx; }

.card { padding: 32rpx; border-radius: $parent-radius-md; background: $parent-card; box-shadow: $parent-shadow; }
.section-title { font-size: 28rpx; font-weight: 600; color: $parent-text; }

.type-row { display: flex; gap: 16rpx; }
.type-chip { flex: 1; padding: 24rpx 16rpx; border-radius: $parent-radius; background: $parent-bg; text-align: center; border: 2rpx solid transparent; }
.type-chip.active { background: #FFF7ED; border-color: $parent-primary; }
.type-label { display: block; font-size: 26rpx; font-weight: 600; color: $parent-text; }
.type-desc { display: block; margin-top: 8rpx; font-size: 22rpx; color: $parent-hint; }
.type-chip.active .type-label { color: $parent-primary; }

.date-row { display: flex; align-items: center; gap: 16rpx; }
.date-item { flex: 1; padding: 24rpx; border-radius: $parent-radius; background: $parent-bg; }
.date-label { display: block; font-size: 24rpx; color: $parent-hint; }
.date-val { display: block; margin-top: 8rpx; font-size: 28rpx; color: $parent-text; font-weight: 500; }
.date-sep { font-size: 28rpx; color: $parent-hint; }

.reason-input { width: 100%; height: 200rpx; padding: 24rpx; border-radius: $parent-radius; background: $parent-bg; font-size: 26rpx; color: $parent-text; box-sizing: border-box; }

.input { width: 100%; height: 88rpx; padding: 0 24rpx; border-radius: $parent-radius; background: $parent-bg; font-size: 26rpx; box-sizing: border-box; }

.submit-btn { width: calc(100% - 56rpx); height: 96rpx; margin: 0 28rpx; border-radius: 9999rpx; background: $parent-primary; color: #fff; font-size: 30rpx; font-weight: 700; line-height: 96rpx; }
</style>
