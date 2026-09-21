<script setup>
import { ref } from "vue"
import { getCurrentStudentId } from "@/utils/currentStudent"
import { createLeave } from "@/api/parent-leave"

const form = ref({
  // 后端 CreateLeave 严格校验 student_id 归属（不属于该家长直接 403），
  // 所以这里必须是真实选中的孩子，不能是写死的 1
  student_id: getCurrentStudentId(),
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

// 只能选今天及以后；也用作 picker 的起始边界
const todayStr = (() => {
  const d = new Date()
  const p = (n) => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
})()

function onDateChange(field, e) {
  const val = e.detail.value
  form.value[field] = val
  // 结束日期不得早于开始日期 —— 后端会把它判为参数无效直接驳回
  if (field === "start_date" && form.value.end_date && form.value.end_date < val) {
    form.value.end_date = val
  }
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
        <picker class="date-picker" mode="date" :value="form.start_date" :start="todayStr"
          @change="onDateChange('start_date', $event)">
          <view class="date-item">
            <text class="date-label">开始日期</text>
            <text class="date-val">{{ form.start_date || "请选择" }}</text>
          </view>
        </picker>
        <text class="date-sep">~</text>
        <picker class="date-picker" mode="date" :value="form.end_date"
          :start="form.start_date || todayStr" @change="onDateChange('end_date', $event)">
          <view class="date-item">
            <text class="date-label">结束日期</text>
            <text class="date-val">{{ form.end_date || "请选择" }}</text>
          </view>
        </picker>
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
.page { min-height: 100vh; padding: $spacing-lg 28rpx 160rpx; background: $color-bg-page; }
.spacer-h12 { height: 24rpx; }
.spacer-h16 { height: 32rpx; }
.spacer-h32 { height: 64rpx; }

.card { padding: $spacing-xl; border-radius: $border-radius-lg; background: $color-bg-card; box-shadow: $shadow-sm; }
.section-title { font-size: $typography-font-size-md; font-weight: 600; color: $color-text-primary; }

.type-row { display: flex; gap: $spacing-md; }
.type-chip { flex: 1; padding: $spacing-lg 16rpx; border-radius: $border-radius-md; background: $color-bg-page; text-align: center; border: 2rpx solid transparent; }
.type-chip.active { background: #FFF7ED; border-color: $color-brand-primary; }
.type-label { display: block; font-size: $typography-font-size-base; font-weight: 600; color: $color-text-primary; }
.type-desc { display: block; margin-top: 8rpx; font-size: $typography-font-size-xs; color: $color-text-hint; }
.type-chip.active .type-label { color: $color-brand-primary; }

.date-row { display: flex; align-items: center; gap: $spacing-md; }
// picker 取代 .date-item 成为 flex 子项，不补这条两个日期框会塌成内容宽度
.date-picker { flex: 1; }
.date-item { flex: 1; width: 100%; padding: $spacing-lg; border-radius: $border-radius-md; background: $color-bg-page; }
.date-label { display: block; font-size: $typography-font-size-sm; color: $color-text-hint; }
.date-val { display: block; margin-top: 8rpx; font-size: $typography-font-size-md; color: $color-text-primary; font-weight: 500; }
.date-sep { font-size: $typography-font-size-md; color: $color-text-hint; }

.reason-input { width: 100%; height: 200rpx; padding: $spacing-lg; border-radius: $border-radius-md; background: $color-bg-page; font-size: $typography-font-size-base; color: $color-text-primary; box-sizing: border-box; }

.input { width: 100%; height: 88rpx; padding: 0 24rpx; border-radius: $border-radius-md; background: $color-bg-page; font-size: $typography-font-size-base; box-sizing: border-box; }

.submit-btn { width: calc(100% - 56rpx); height: 96rpx; margin: 0 28rpx; border-radius: 9999rpx; background: $color-brand-primary; color: #fff; font-size: $typography-font-size-lg; font-weight: 700; line-height: 96rpx; }
</style>
