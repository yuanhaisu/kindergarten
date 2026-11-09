<script setup>
/**
 * 隐私政策阻断页：静默登录后缺同意时进入。
 * 页面不在 tabBar；同意后 switchTab 回首页（家长端有 4 个 tab）。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { getConsentStatus, recordConsent } from "@/api/consent"
import { clearAuth } from "@/utils/auth"

const ICON = {
  check: "\u{eb7b}" // remixicon check-line，与登录页同一套字形
}

const status = ref({ policy_version: "", consented: false })
const loading = ref(true)
const submitting = ref(false)
const agreed = ref(false)
const error = ref("")

const versionLabel = computed(() =>
  status.value.policy_version ? `版本 ${status.value.policy_version}` : ""
)

async function loadStatus() {
  loading.value = true
  error.value = ""
  try {
    const res = await getConsentStatus()
    status.value = res.data || {}
    if (status.value.consented) goHome()
  } catch (e) {
    error.value = e?.message || "读取政策状态失败，请稍后重试"
  } finally {
    loading.value = false
  }
}

onLoad(() => {
  loadStatus()
})

async function submit() {
  if (loading.value || submitting.value) return
  if (!agreed.value) {
    uni.showToast({ title: "请先勾选同意", icon: "none" })
    return
  }
  submitting.value = true
  error.value = ""
  try {
    await recordConsent(status.value.policy_version)
    uni.showToast({ title: "已同意", icon: "success" })
    setTimeout(goHome, 600)
  } catch (e) {
    if (e?.code === 40030) {
      await loadStatus()
      uni.showToast({ title: "政策已更新，请重新阅读", icon: "none" })
    } else {
      error.value = e?.message || "提交失败，请重试"
    }
  } finally {
    submitting.value = false
  }
}

function goHome() {
  uni.switchTab({ url: "/pages/index/index" })
}

function decline() {
  uni.showModal({
    title: "暂不同意",
    content: "不同意隐私政策将无法使用家长端。",
    confirmText: "退出登录",
    success(res) {
      if (!res.confirm) return
      clearAuth()
      uni.reLaunch({ url: "/pages/auth/login" })
    }
  })
}

function openPolicy(type) {
  uni.navigateTo({ url: `/pages/auth/policy?type=${type}` })
}
</script>

<template>
  <view class="page">
    <view class="card">
      <view class="head">
        <text class="title">隐私政策与用户协议</text>
        <text v-if="versionLabel" class="version">{{ versionLabel }}</text>
      </view>

      <text class="lead">在使用养成星球家长端前，请阅读并同意以下内容。</text>

      <view class="points">
        <view class="point">
          <text class="point-title">我们处理哪些信息</text>
          <text class="point-body">手机号、微信身份、孩子的姓名与照片、出勤与接送记录等。</text>
        </view>
        <view class="point">
          <text class="point-title">谁可以看到</text>
          <text class="point-body">孩子的照片默认只给本班家长、老师与园所管理者；班内共享可随时撤回。</text>
        </view>
        <view class="point">
          <text class="point-title">你的权利</text>
          <text class="point-body">可查询、更正、删除信息，撤回照片分享授权，或通过邮箱申请注销。</text>
        </view>
      </view>

      <view class="links">
        <text class="link" @click="openPolicy('privacy')">查看完整《隐私政策》</text>
        <text class="link" @click="openPolicy('terms')">查看《用户协议》</text>
      </view>

      <view v-if="error" class="error" role="alert">{{ error }}</view>

      <view class="agree-row" @click="agreed = !agreed">
        <view class="agree-box" :class="{ 'is-checked': agreed }">
          <text class="agree-check">{{ ICON.check }}</text>
        </view>
        <text class="agree-text">我已阅读并同意《用户协议》和《隐私政策》</text>
      </view>

      <button class="btn" :disabled="submitting || loading" @click="submit">
        {{ submitting ? "提交中..." : "同意并继续" }}
      </button>
      <text class="decline" @click="decline">暂不同意</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  padding: 48rpx 40rpx calc(48rpx + env(safe-area-inset-bottom));
  background: $color-bg-page;
}

.card {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  padding: $spacing-xl;
  border: 1rpx solid $color-border-subtle;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}

.title {
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-bold;
  line-height: 1.35;
}

.version {
  flex: none;
  padding: 4rpx 16rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
}

.lead {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.7;
}

.points {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  padding: $spacing-lg 0;
  border-top: 1rpx solid $color-border-subtle;
  border-bottom: 1rpx solid $color-border-subtle;
}

.point-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.5;
}

.point-body {
  display: block;
  margin-top: 6rpx;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.65;
}

.links {
  display: flex;
  flex-direction: column;
}

.link {
  min-height: 80rpx;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
  line-height: 80rpx;
  text-decoration: underline;
}

.error {
  padding: 20rpx 24rpx;
  border: 1rpx solid $color-semantic-danger-border;
  border-radius: $border-radius-xl;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger-strong;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.agree-row {
  display: flex;
  align-items: center;
  gap: 20rpx;
  min-height: 88rpx;
  padding: 0 8rpx;
}

.agree-box {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 40rpx;
  height: 40rpx;
  border: 3rpx solid $color-text-muted;
  border-radius: $border-radius-md;
  background: $color-bg-card;
}

.agree-box.is-checked {
  border-color: $color-brand-primary-dark;
  background: $color-brand-primary-dark;
}

.agree-check {
  color: $color-text-on-brand;
  font-family: remixicon;
  font-size: $typography-font-size-xs;
  line-height: 1;
  opacity: 0;
}

.agree-box.is-checked .agree-check {
  opacity: 1;
}

.agree-text {
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  line-height: 1.6;
}

.btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 96rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-bold;
  box-shadow: $shadow-md;
}

.btn[disabled] {
  opacity: 0.55;
  background: $gradient-brand;
  color: $color-text-on-brand;
}

.decline {
  min-height: 88rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 88rpx;
  text-align: center;
}
</style>
