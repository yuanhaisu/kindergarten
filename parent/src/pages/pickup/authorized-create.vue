<script setup>
import { ref } from "vue"
import { createAuthorizedPerson } from "@/api/parent-pickup"
import { getCurrentStudentId } from "@/utils/currentStudent"

// 字段严格对齐后端的 CreateAuthorizedPersonInput。
//
// 刻意不放的两项：
// - auth_type（长期/临时）：新建接口不接受有效期（ExpireAt 只在 Update 里），
//   建出来的「临时授权」永远不会过期，不如不给这个选项。缺省即 permanent。
// - 短信验证码：后端 CreateAuthorizedPerson 并不校验 sms_code。
const relationOptions = ["爸爸", "妈妈", "爷爷", "奶奶", "外公", "外婆", "叔叔", "阿姨", "其他"]

const form = ref({
  name: "",
  phone: "",
  relation: "爸爸",
  id_card_number: "",
  allow_pickup: true,
  allow_view_record: false,
  allow_view_status: false,
  allow_growth_view: false,
  notify_on_pickup: true
})

const loading = ref(false)

function onRelationChange(e) {
  form.value.relation = relationOptions[Number(e.detail.value)] || relationOptions[0]
}

async function handleSubmit() {
  if (!form.value.name || !form.value.phone || !form.value.relation) {
    uni.showToast({ title: "请填写完整信息", icon: "none" })
    return
  }
  loading.value = true
  try {
    // student_id 走 query（第三个参数），body 里的会被后端忽略
    await createAuthorizedPerson({ ...form.value }, getCurrentStudentId())
    uni.showToast({ title: "已提交，待园所审核", icon: "none" })
    setTimeout(() => uni.navigateBack(), 900)
  } catch (err) {
    // http.js 的拦截器已经全局 toast 过了，这里不再重复提示
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <view class="page">
    <view class="card">
      <text class="section-title">基本信息</text>
      <view class="spacer-h16" />
      <input v-model="form.name" class="input" placeholder="姓名（必填）" />
      <view class="spacer-h12" />
      <input v-model="form.phone" class="input" placeholder="手机号（必填）" type="number" :maxlength="11" />
      <view class="spacer-h12" />
      <picker mode="selector" :range="relationOptions" :value="relationOptions.indexOf(form.relation)"
        @change="onRelationChange">
        <view class="input picker-field">
          <text class="picker-label">与孩子关系</text>
          <text class="picker-value">{{ form.relation }}</text>
        </view>
      </picker>
      <view class="spacer-h12" />
      <input v-model="form.id_card_number" class="input" placeholder="身份证号（选填，便于门卫核验）"
        :maxlength="18" />
    </view>

    <view class="spacer-h16" />

    <view class="card">
      <text class="section-title">权限</text>
      <view class="spacer-h16" />
      <view class="switch-row">
        <text class="switch-label">可接送孩子</text>
        <switch :checked="form.allow_pickup" @change="form.allow_pickup = $event.detail.value" />
      </view>
      <view class="switch-row">
        <text class="switch-label">可查看接送记录</text>
        <switch :checked="form.allow_view_record" @change="form.allow_view_record = $event.detail.value" />
      </view>
      <view class="switch-row">
        <text class="switch-label">可查看在园状态</text>
        <switch :checked="form.allow_view_status" @change="form.allow_view_status = $event.detail.value" />
      </view>
      <view class="switch-row">
        <text class="switch-label">可查看成长记录</text>
        <switch :checked="form.allow_growth_view" @change="form.allow_growth_view = $event.detail.value" />
      </view>
      <view class="switch-row">
        <text class="switch-label">接送时通知我</text>
        <switch :checked="form.notify_on_pickup" @change="form.notify_on_pickup = $event.detail.value" />
      </view>
    </view>

    <view class="hint">提交后需园所审核通过，授权才会生效。</view>

    <view class="spacer-h32" />
    <button class="submit-btn" :disabled="loading" @click="handleSubmit">
      {{ loading ? "提交中..." : "提交" }}
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

.input { width: 100%; height: 88rpx; padding: 0 24rpx; border-radius: $border-radius-md; background: $color-bg-page; font-size: $typography-font-size-base; color: $color-text-primary; box-sizing: border-box; }

.picker-field { display: flex; align-items: center; justify-content: space-between; }
.picker-label { font-size: $typography-font-size-base; color: $color-text-muted; }
.picker-value { font-size: $typography-font-size-base; color: $color-text-primary; }

.switch-row { display: flex; align-items: center; justify-content: space-between; padding: $spacing-sm 0; }
.switch-label { font-size: $typography-font-size-base; color: $color-text-primary; }

.hint { margin-top: $spacing-lg; padding: 0 8rpx; font-size: $typography-font-size-sm; color: $color-text-hint; }

.submit-btn { width: calc(100% - 56rpx); height: 96rpx; margin: 0 28rpx; border-radius: $border-radius-full; background: $color-brand-primary; color: #fff; font-size: $typography-font-size-lg; font-weight: 700; line-height: 96rpx; }
</style>
