<script setup>
import { ref } from "vue"
import { bindPhoneLogin } from "@/utils/login"

const submitting = ref(false)
const agreed = ref(false)

function toggleAgreed() {
  agreed.value = !agreed.value
}

async function handleWxLogin() {
  if (!agreed.value) {
    uni.showToast({ title: "请先阅读并同意用户协议和隐私政策", icon: "none" })
    return
  }
  submitting.value = true

  try {
    await bindPhoneLogin()
    uni.showToast({ title: "登录成功", icon: "success" })
    uni.reLaunch({ url: "/pages/index/index" })
  } catch (err) {
    uni.showToast({ title: err?.message || "登录失败，请稍后重试", icon: "none" })
  } finally {
    submitting.value = false
  }
}

function goAgreement(type) {
  uni.showToast({ title: type + "页面开发中", icon: "none" })
}
</script>

<template>
  <view class="page">
    <view class="brand-area">
      <view class="logo-box">
        <text class="logo-icon">星</text>
      </view>
      <text class="app-name">养成星球</text>
      <text class="app-sub">幼儿园数字化管理平台 · 老师端</text>
    </view>

    <view class="form-card">
      <button
        class="wx-login-btn"
        :class="{ disabled: !agreed }"
        :loading="submitting"
        :disabled="submitting"
        @click="handleWxLogin"
      >
        <text class="wx-btn-text">{{ submitting ? "登录中..." : "微信授权登录" }}</text>
      </button>

      <view class="agreement-row">
        <view class="checkbox" :class="{ checked: agreed }" @click="toggleAgreed">
          <text v-if="agreed" class="check-mark">✓</text>
        </view>
        <text class="agree-text">我已阅读并同意</text>
        <text class="agree-link" @click="goAgreement('用户协议')">《用户协议》</text>
        <text class="agree-text">和</text>
        <text class="agree-link" @click="goAgreement('隐私政策')">《隐私政策》</text>
      </view>
    </view>

    <view class="bottom-hint">
      <text>仅对幼儿园老师开放，需园方统一开通账号</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 96rpx 48rpx 96rpx;
  background: linear-gradient(180deg, $color-semantic-info-bg 0%, #FEFCE8 100%);
}

.brand-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 96rpx;
}

.logo-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 192rpx;
  height: 192rpx;
  border-radius: 48rpx;
  background: linear-gradient(180deg, #60A5FA 0%, #67E8F9 100%);
  box-shadow: 0 16rpx 48rpx rgba(96, 165, 250, 0.32);
}

.logo-icon {
  font-size: 96rpx;
  font-weight: 800;
  color: #fff;
}

.app-name {
  margin-top: $spacing-xl;
  font-size: 56rpx;
  font-weight: 800;
  color: #1E40AF;
}

.app-sub {
  margin-top: 12rpx;
  font-size: $typography-font-size-md;
  color: $color-text-muted;
}

.form-card {
  width: 100%;
  max-width: 640rpx;
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
}

.wx-login-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 96rpx;
  border-radius: $border-radius-xl;
  background: #07C160;
  border: none;

  &.disabled {
    opacity: 0.55;
  }
}

.wx-btn-text {
  font-size: $typography-font-size-xl;
  font-weight: 600;
  color: #fff;
}

.agreement-row {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  margin-top: $spacing-xl;
  gap: $spacing-xs;
}

.checkbox {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32rpx;
  height: 32rpx;
  border-radius: $border-radius-sm;
  border: 2rpx solid #CBD5E1;
  background: #fff;
  flex-shrink: 0;
  margin-right: $spacing-xs;
}

.checkbox.checked {
  background: $color-semantic-info;
  border-color: $color-semantic-info;
}

.check-mark {
  font-size: $typography-font-size-xs;
  font-weight: 700;
  color: #fff;
}

.agree-text {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.agree-link {
  font-size: $typography-font-size-sm;
  color: $color-semantic-info;
}

.bottom-hint {
  margin-top: 64rpx;
  text-align: center;
}

.bottom-hint text {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}
</style>
