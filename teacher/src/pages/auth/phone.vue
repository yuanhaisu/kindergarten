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
  // placeholder for agreement page navigation
  uni.showToast({ title: type + "页面开发中", icon: "none" })
}
</script>

<template>
  <view class="page">
    <!-- 顶部 logo 区域 -->
    <view class="brand-area">
      <view class="logo-box">
        <text class="logo-icon">星</text>
      </view>
      <text class="app-name">养成星球</text>
      <text class="app-sub">幼儿园数字化管理平台 · 老师端</text>
    </view>

    <!-- 登录表单卡片 -->
    <view class="form-card">
      <!-- 微信授权登录按钮 -->
      <button
        class="wx-login-btn"
        :class="{ disabled: !agreed }"
        :loading="submitting"
        :disabled="submitting"
        @click="handleWxLogin"
      >
        <text class="wx-btn-text">{{ submitting ? "登录中..." : "微信授权登录" }}</text>
      </button>

      <!-- 协议 -->
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

    <!-- 底部提示 -->
    <view class="bottom-hint">
      <text>仅对幼儿园老师开放，需园方统一开通账号</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
$blue-900: #1E3A8A;

.page {
  min-height: 100vh;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 96rpx 48rpx 96rpx;
  background: linear-gradient(180deg, #EFF6FF 0%, #FEFCE8 100%);
}

// ===== Logo 区域 =====
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
  margin-top: 32rpx;
  font-size: 56rpx;
  font-weight: 800;
  color: #1E40AF;
}

.app-sub {
  margin-top: 12rpx;
  font-size: 28rpx;
  color: #64748B;
}

// ===== 表单卡片 =====
.form-card {
  width: 100%;
  max-width: 640rpx;
  padding: 48rpx;
  border-radius: 32rpx;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
}

.wx-login-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 96rpx;
  border-radius: 24rpx;
  background: #07C160;
  border: none;

  &.disabled {
    opacity: 0.55;
  }
}

.wx-btn-text {
  font-size: 32rpx;
  font-weight: 600;
  color: #fff;
}

// ===== 协议 =====
.agreement-row {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  margin-top: 32rpx;
  gap: 8rpx;
}

.checkbox {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32rpx;
  height: 32rpx;
  border-radius: 8rpx;
  border: 2rpx solid #CBD5E1;
  background: #fff;
  flex-shrink: 0;
  margin-right: 8rpx;
}

.checkbox.checked {
  background: #3B82F6;
  border-color: #3B82F6;
}

.check-mark {
  font-size: 22rpx;
  font-weight: 700;
  color: #fff;
}

.agree-text {
  font-size: 24rpx;
  color: #64748B;
}

.agree-link {
  font-size: 24rpx;
  color: #3B82F6;
}

// ===== 底部提示 =====
.bottom-hint {
  margin-top: 64rpx;
  text-align: center;
}

.bottom-hint text {
  font-size: 24rpx;
  color: #94A3B8;
}
</style>
