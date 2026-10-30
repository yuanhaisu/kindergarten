<script setup>
/**
 * 家长登录 · 全页只有一个问题：怎么进去
 *
 * 入口根页面（三个调用点全部 reLaunch），所以没有返回箭头；页面不在 tabBar 里。
 * 首屏协议默认未勾选，点击登录只做门禁提示，不进入 loading。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin, toLoginError } from "@/utils/login"
import { getToken } from "@/utils/auth"
import { resumePendingRoute } from "@/utils/pendingRoute"
import BrandBlock from "@/components/BrandBlock.vue"

// remixicon 字形（App.vue 已加载字体）。设计稿的微信气泡/对勾/感叹号是内联 SVG，
// WXML 不允许内联 SVG，改用同一图标集的字体字形。
const ICON = {
  wechat: "\u{f2b6}", // wechat-line
  check: "\u{eb7b}", // check-line
  alert: "\u{eca1}" // error-warning-line
}

const loading = ref(false)
// 内联错误态 { title, detail }；null 表示无错误
const loginError = ref(null)
// 主动勾选：两端一致，不用「登录即视为同意」的被动推定
const agreed = ref(false)
const nudged = ref(false)

let nudgeTimer = null

// 一颗按钮承载三态文案：空闲 / 登录中 / 失败后重试
const buttonLabel = computed(() => {
  if (loading.value) return "登录中..."
  if (loginError.value) return "重新登录"
  return "微信一键登录"
})

function toggleAgreed() {
  if (loading.value) return
  agreed.value = !agreed.value
  if (agreed.value) nudged.value = false
}

// 未勾选时轻抖一下协议行，让「你说的就是这一行」有落点；toast 一闪而过，
// 不是所有人都会把视线移到勾选框。
function nudgeAgree() {
  if (nudgeTimer) clearTimeout(nudgeTimer)
  nudged.value = false
  nudgeTimer = setTimeout(() => {
    nudged.value = true
    nudgeTimer = setTimeout(() => {
      nudged.value = false
      nudgeTimer = null
    }, 420)
  }, 16)
}

async function handleLogin() {
  if (loading.value) return
  if (!agreed.value) {
    nudgeAgree()
    uni.showToast({ title: "请先阅读并同意用户协议和隐私政策", icon: "none" })
    return
  }

  // 重试前先清掉上一轮错误条，避免「登录中」与旧错误同屏
  loginError.value = null
  loading.value = true
  try {
    await ensureLogin()
    uni.showToast({ title: "登录成功", icon: "success" })
    setTimeout(() => {
      // 从深链/401 过来时优先回到原页面；没有保留路径才回首页。
      if (!resumePendingRoute()) uni.reLaunch({ url: "/pages/index/index" })
    }, 800)
  } catch (err) {
    // 不用 toast —— 1.5 秒就消失，失败原因读不到，也没有第二次机会。
    // 改渲染成卡片内常驻错误条，按钮切「重新登录」。
    loginError.value = toLoginError(err)
  } finally {
    loading.value = false
  }
}

onLoad(() => {
  // 已登录则直接离开登录页，不在登录页停留；有保留路径先回跳。
  if (getToken()) {
    if (!resumePendingRoute()) uni.reLaunch({ url: "/pages/index/index" })
  }
})
</script>

<template>
  <view class="page">
    <!-- 品牌区两端共用（BrandBlock），标识/字阶/tagline 不再各写一份 -->
    <view class="hero"><BrandBlock /></view>

    <view class="login-card">
      <!-- 失败态：卡片内常驻错误条，文案按 toLoginError 的分支切换 -->
      <view v-if="loginError" class="error-strip" role="alert">
        <text class="error-icon">{{ ICON.alert }}</text>
        <view class="error-copy">
          <text class="error-title">{{ loginError.title }}</text>
          <text class="error-desc">{{ loginError.detail }}</text>
        </view>
      </view>

      <button
        class="login-btn"
        :class="{ 'is-loading': loading }"
        :disabled="loading"
        hover-class="none"
        @click="handleLogin"
      >
        <view v-if="loading" class="spinner" />
        <text v-else class="btn-icon">{{ ICON.wechat }}</text>
        <text class="btn-label">{{ buttonLabel }}</text>
      </button>

      <!-- 主动勾选：整行是命中区（96rpx），协议名在协议页落地前保持纯文本。 -->
      <view
        class="agree-row"
        :class="{ 'is-loading': loading, 'is-unchecked': !agreed, 'is-nudged': nudged }"
        @click="toggleAgreed"
      >
        <view class="agree-box" :class="{ 'is-checked': agreed }">
          <text class="agree-check">{{ ICON.check }}</text>
        </view>
        <text class="agree-text">我已阅读并同意《用户协议》和《隐私政策》</text>
      </view>
    </view>

    <!-- 前置告知：账号未开通的失败发生在首页绑卡，这里先说清楚要什么 -->
    <text class="footnote">登录后需绑定园所已开通的家长账号</text>
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
  // 设计稿 32px 20px + 底部安全区；间距 token 无这一档，按设计值换算
  padding: 64rpx 40rpx calc(64rpx + env(safe-area-inset-bottom));
  // 家长端页面底色是冷雾白（DESIGN.md「冷家长」）
  background: $color-bg-page;
}

/* ---------- 品牌区 ---------- */

.hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: 700rpx; // 设计稿 350px
  // BrandBlock 自带 32rpx 底距，再补到设计稿的 64rpx
  margin-bottom: $spacing-xl;
}

/* ---------- 登录卡 ---------- */

.login-card {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 700rpx;
  gap: $spacing-lg;
  // 设计稿 16px 内边距（--space-xl），与 AppCard 的 32rpx 同值
  padding: $spacing-xl;
  border: 1rpx solid $color-border-subtle;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

/* ---------- 内联错误条 ---------- */

.error-strip {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  padding: 20rpx 24rpx;
  border: 1rpx solid $color-semantic-danger-border;
  border-radius: $border-radius-xl;
  background: $color-semantic-danger-bg;
  animation: error-in $animation-duration-base $animation-easing-default both;
}

.error-icon {
  flex: none;
  margin-top: 2rpx;
  color: $color-semantic-danger-strong;
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1.2;
}

.error-copy {
  min-width: 0;
}

.error-title {
  display: block;
  color: $color-semantic-danger-strong;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  letter-spacing: 0.01em;
  overflow-wrap: anywhere;
}

.error-desc {
  display: block;
  margin-top: 4rpx;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-normal;
  line-height: 1.5;
  letter-spacing: 0.01em;
  overflow-wrap: anywhere;
}

/* ---------- 主按钮 ---------- */

.login-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  width: 100%;
  height: 96rpx; // 设计稿 48px
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-bold;
  line-height: 1;
  letter-spacing: 0.02em;
  box-shadow: $shadow-md;
  transition:
    opacity $animation-duration-fast $animation-easing-default,
    transform $animation-duration-fast $animation-easing-default,
    box-shadow $animation-duration-fast $animation-easing-default;
}

.login-btn:active {
  transform: scale(0.985);
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

.btn-icon {
  flex: none;
  font-family: remixicon;
  font-size: $typography-font-size-3xl; // 设计稿 20px
  line-height: 1;
}

.btn-label {
  min-width: 0;
}

.spinner {
  flex: none;
  width: 30rpx;
  height: 30rpx;
  border: 4rpx solid rgba($color-text-on-brand, 0.34);
  border-top-color: $color-text-on-brand;
  border-radius: $border-radius-full;
  animation: spin 0.8s linear infinite;
}

/* ---------- 协议行 ---------- */

.agree-row {
  display: flex;
  align-items: center;
  gap: 20rpx;
  width: 100%;
  min-height: 96rpx; // 设计稿 48px，高于 iOS 44pt 下限
  margin-top: 4rpx;
  padding: 0 16rpx;
  border-radius: $border-radius-xl;
  transition:
    opacity $animation-duration-fast $animation-easing-default,
    background-color $animation-duration-fast $animation-easing-default;
}

.agree-row:active {
  background: $color-brand-primary-bg;
}

.agree-row.is-nudged {
  animation: agree-nudge 360ms $animation-easing-default;
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
  border: 3rpx solid $color-text-muted;
  border-radius: $border-radius-md;
  background: $color-bg-card;
  transition:
    background $animation-duration-fast $animation-easing-default,
    border-color $animation-duration-fast $animation-easing-default;
}

/* 未勾选是首屏默认形态，边框归品牌深橙，提醒这里是待办项 */
.agree-row.is-unchecked .agree-box {
  border-color: $color-brand-primary-dark;
}

.agree-box.is-checked {
  border-color: $color-brand-primary-dark;
  background: $color-brand-primary-dark;
}

.agree-check {
  font-family: remixicon;
  font-style: normal;
  font-size: $typography-font-size-xs;
  line-height: 1;
  color: $color-text-on-brand;
  opacity: 0;
  transform: scale(0.7);
  transition:
    opacity $animation-duration-fast $animation-easing-default,
    transform $animation-duration-fast $animation-easing-default;
}

.agree-box.is-checked .agree-check {
  opacity: 1;
  transform: scale(1);
}

.agree-text {
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-normal;
  line-height: 1.65;
  letter-spacing: 0.01em;
  overflow-wrap: anywhere;
  transition: color $animation-duration-fast $animation-easing-default;
}

.agree-row.is-unchecked .agree-text {
  color: $color-text-primary;
}

/* ---------- 卡外脚注 ---------- */

.footnote {
  max-width: 700rpx;
  margin-top: 56rpx; // 设计稿 28px
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-normal;
  line-height: 1.65;
  letter-spacing: 0.01em;
  text-align: center;
  overflow-wrap: anywhere;
}

/* ---------- 动效 ---------- */

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes error-in {
  from {
    opacity: 0;
    transform: translateY(-8rpx);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes agree-nudge {
  0%,
  100% {
    transform: translateX(0);
  }
  22% {
    transform: translateX(-6rpx);
  }
  48% {
    transform: translateX(6rpx);
  }
  72% {
    transform: translateX(-2rpx);
  }
}

@media (prefers-reduced-motion: reduce) {
  .spinner {
    animation: none;
  }

  .error-strip,
  .agree-row.is-nudged {
    animation: none;
  }
}
</style>
