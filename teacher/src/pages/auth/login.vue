<script setup>
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { bindPhoneLogin, toLoginError } from "@/utils/login"
import { getToken } from "@/utils/auth"
import BrandBlock from "@/components/BrandBlock.vue"

// remixicon PUA 字形（App.vue 已加载字体）。设计稿的微信气泡与对勾是内联 SVG，
// WXML 不允许内联 SVG，改用同一图标集的字体字形。
const ICON_WECHAT = "" // wechat-line
const ICON_CHECK = "" // check-line

const submitting = ref(false)
const agreed = ref(false)
// 内联错误态 { title, detail }；null 表示无错误
const loginError = ref(null)

// 一颗按钮承载三态文案：空闲 / 登录中 / 失败后重试
const buttonLabel = computed(() => {
  if (submitting.value) return "登录中..."
  if (loginError.value) return "重新登录"
  return "微信授权登录"
})

function toggleAgreed() {
  if (submitting.value) return
  agreed.value = !agreed.value
}

async function handleWxLogin() {
  if (submitting.value) return
  if (!agreed.value) {
    uni.showToast({ title: "请先阅读并同意用户协议和隐私政策", icon: "none" })
    return
  }

  // 重试前先清掉上一轮错误条，避免「登录中」与旧错误同屏
  loginError.value = null
  submitting.value = true

  try {
    await bindPhoneLogin()
    uni.showToast({ title: "登录成功", icon: "success" })
    uni.reLaunch({ url: "/pages/index/index" })
  } catch (err) {
    // 不再用 toast —— 1.5 秒就消失，失败原因读不到。
    // 改渲染成卡片内常驻错误条，按钮切「重新登录」。
    loginError.value = toLoginError(err)
  } finally {
    submitting.value = false
  }
}

onLoad(() => {
  // 已登录则直接进入工作台，不在登录页停留（与家长端一致）
  if (getToken()) {
    uni.reLaunch({ url: "/pages/index/index" })
  }
})
</script>

<template>
  <view class="page">
    <!-- 品牌区两端共用（BrandBlock），柔光只在本页作为背景装饰 -->
    <view class="hero">
      <view class="hero-halo" />
      <view class="hero-brand"><BrandBlock /></view>
    </view>

    <view class="login-card">
      <!-- 失败态：卡片内常驻错误条，文案按 toLoginError 的分支切换 -->
      <view v-if="loginError" class="error-strip" role="alert">
        <text class="error-title">{{ loginError.title }}</text>
        <text class="error-desc">{{ loginError.detail }}</text>
      </view>

      <button
        class="login-btn"
        :class="{ 'is-loading': submitting }"
        :disabled="submitting"
        hover-class="none"
        @click="handleWxLogin"
      >
        <text class="login-btn__icon">{{ ICON_WECHAT }}</text>
        <text class="login-btn__label">{{ buttonLabel }}</text>
      </button>

      <!-- 主动勾选：整行可点（88rpx 命中区）。协议名在协议页落地前保持纯文本，
           不做「页面开发中」的死链。 -->
      <view class="agree-row" :class="{ 'is-loading': submitting }" @click="toggleAgreed">
        <view class="agree-box" :class="{ 'is-checked': agreed }">
          <text class="agree-check">{{ ICON_CHECK }}</text>
        </view>
        <text class="agree-text">我已阅读并同意《用户协议》和《隐私政策》</text>
      </view>
    </view>

    <text class="footnote">仅对幼儿园老师开放，需园方统一开通账号</text>
  </view>
</template>

<style scoped lang="scss">
.page {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  // 设计稿 .screen：28px 24px + 底部安全区；间距 token 无这一档，按设计值换算
  padding: 56rpx 48rpx calc(56rpx + env(safe-area-inset-bottom));
  // 老师端页面底色是暖奶白（DESIGN.md「暖老师」），不是冷雾白
  background: $gradient-page;
}

/* ---------- 品牌区 ---------- */

.hero {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  // BrandBlock 自带 32rpx 底距，再补到设计稿的 64rpx
  margin-bottom: $spacing-xl;
}

/* 全页唯一的装饰：logo 背后一圈极淡的琥珀柔光 */
.hero-halo {
  position: absolute;
  top: 80rpx; // logo 160rpx 的圆心
  left: 50%;
  z-index: 0;
  width: 360rpx;
  height: 360rpx;
  border-radius: $border-radius-full;
  background: radial-gradient(
    circle,
    rgba($color-brand-primary, 0.16) 0%,
    rgba($color-brand-primary, 0) 68%
  );
  transform: translate(-50%, -50%);
  pointer-events: none;
}

// 定位后压在柔光之上，logo 与文字不被琥珀色染层覆盖
.hero-brand {
  position: relative;
  z-index: 1;
  width: 100%;
}

/* ---------- 登录卡 ---------- */

.login-card {
  display: flex;
  flex-direction: column;
  width: 100%;
  // 登录卡是独立视觉块：设计稿 20px 内边距，组件卡 token（32rpx）不适用
  padding: 40rpx;
  gap: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

/* 文字走前景色而非红色：#EF4444 压 #FEE2E2 只有 3.1:1，不达正文 4.5:1。
   错误语义由浅红底承担，文字保证可读。 */
.error-strip {
  display: flex;
  flex-direction: column;
  width: 100%;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
  text-align: center;
}

.error-title {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  color: $color-text-primary;
}

.error-desc {
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-normal;
  line-height: 1.4;
  color: $color-text-primary;
}

/* ---------- 主按钮 ---------- */

.login-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100rpx; // 设计稿 50px
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-bold;
  line-height: 1;
  letter-spacing: 0.02em;
  box-shadow: $shadow-md;
  transition: opacity $animation-duration-fast $animation-easing-default,
    transform $animation-duration-fast $animation-easing-default,
    box-shadow $animation-duration-fast $animation-easing-default;
}

.login-btn:active {
  transform: scale(0.98);
  box-shadow: $shadow-sm;
}

/* 登录中：按钮禁用 + 降透明度（设计稿 .55）。
   微信原生 disabled 会给 button 铺灰底，这里显式保住渐变，只降透明度。 */
.login-btn.is-loading {
  opacity: 0.55;
  box-shadow: $shadow-sm;
}

.login-btn[disabled] {
  background: $gradient-brand;
  color: $color-text-on-brand;
}

.login-btn__icon {
  flex: none;
  margin-right: $spacing-md;
  font-family: remixicon;
  font-style: normal;
  font-size: $typography-font-size-3xl; // 设计稿 20px
  line-height: 1;
}

.login-btn__label {
  min-width: 0;
}

/* ---------- 协议行 ---------- */

.agree-row {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  gap: $spacing-md;
  width: 100%;
  min-height: 88rpx; // iOS 44pt 触摸下限
  padding: $spacing-sm 0;
  border-radius: $border-radius-xl;
  transition: opacity $animation-duration-fast $animation-easing-default;
}

/* 登录中协议行同步降级，且不可再切换勾选 */
.agree-row.is-loading {
  opacity: 0.55;
  pointer-events: none;
}

.agree-box {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 40rpx; // 设计稿 20px
  height: 40rpx;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-sm;
  background: $color-bg-card;
  transition: background $animation-duration-fast $animation-easing-default,
    border-color $animation-duration-fast $animation-easing-default;
}

/* 选中态归暖阳橙（Warm-Dominance：选中/激活不走语义蓝） */
.agree-box.is-checked {
  border-color: $color-brand-primary;
  background: $color-brand-primary;
}

.agree-check {
  font-family: remixicon;
  font-style: normal;
  font-size: $typography-font-size-sm;
  line-height: 1;
  color: $color-text-on-brand;
  opacity: 0;
  transform: scale(0.7);
  transition: opacity $animation-duration-fast $animation-easing-default,
    transform $animation-duration-fast $animation-easing-default;
}

.agree-box.is-checked .agree-check {
  opacity: 1;
  transform: scale(1);
}

.agree-text {
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-normal;
  line-height: 1.65;
  color: $color-text-muted;
}

/* ---------- 脚注 ---------- */

.footnote {
  max-width: 560rpx; // 设计稿 280px
  margin-top: 56rpx; // 设计稿 28px
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-normal;
  line-height: 1.65;
  // --hint（#94A3B8）压暖奶白只有 2.3:1，退到 --muted（4.3:1）
  color: $color-text-muted;
  text-align: center;
}
</style>
