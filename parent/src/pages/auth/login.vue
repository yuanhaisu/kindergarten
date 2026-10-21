<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin, toLoginError } from "@/utils/login"
import { getToken } from "@/utils/auth"
import AppCard from "@/components/AppCard.vue"
import AppButton from "@/components/AppButton.vue"
import BrandBlock from "@/components/BrandBlock.vue"

const loading = ref(false)
// 内联错误态 { title, detail }；null 表示无错误
const loginError = ref(null)
// 主动勾选：两端一致，不用「登录即视为同意」的被动推定
const agreed = ref(false)

function toggleAgreed() {
  agreed.value = !agreed.value
}

async function handleLogin() {
  if (loading.value) return
  if (!agreed.value) {
    uni.showToast({ title: "请先阅读并同意用户协议和隐私政策", icon: "none" })
    return
  }
  loginError.value = null
  loading.value = true
  try {
    await ensureLogin()
    uni.showToast({ title: "登录成功", icon: "success" })
    setTimeout(() => {
      uni.reLaunch({ url: "/pages/index/index" })
    }, 800)
  } catch (err) {
    // 不再用 toast —— 1.5 秒就消失，失败原因读不到，也没有第二次机会。
    // 改渲染成卡片内常驻错误条，按钮切「重新登录」。
    loginError.value = toLoginError(err)
  } finally {
    loading.value = false
  }
}

onLoad(() => {
  // 已登录则直接进入首页，不在登录页停留
  if (getToken()) {
    uni.reLaunch({ url: "/pages/index/index" })
  }
})
</script>

<template>
  <view class="page">
    <!-- 品牌区两端共用（BrandBlock），标识/字阶/tagline 不再各写一份 -->
    <BrandBlock />

    <AppCard custom-class="login-card">
      <view v-if="loginError" class="error-box">
        <text class="error-title">{{ loginError.title }}</text>
        <text class="error-detail">{{ loginError.detail }}</text>
      </view>

      <AppButton size="lg" :disabled="loading" @click="handleLogin">
        {{ loading ? "登录中..." : loginError ? "重新登录" : "微信一键登录" }}
      </AppButton>

      <!-- 主动勾选：整行可点（命中区 88rpx），协议名在协议页落地前先作纯文本，
           不做「页面开发中」的死链。 -->
      <view class="agreement-row" @click="toggleAgreed">
        <view class="checkbox" :class="{ checked: agreed }">
          <text v-if="agreed" class="check-mark">✓</text>
        </view>
        <text class="agree-text">我已阅读并同意《用户协议》和《隐私政策》</text>
      </view>
    </AppCard>

    <!-- 前置告知：账号未开通的失败发生在首页绑卡（40031），这里先说清楚要什么 -->
    <view class="bottom-hint">
      <text>登录后需绑定园所已开通的家长账号</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: $spacing-xl $component-page-padding-x;
  // 家长端页面底色是冷雾白（DESIGN.md「冷家长」），不是共享暖渐变
  background: $color-bg-page;
}

// ===== 登录卡（AppCard 管圆角/阴影/底色） =====
.login-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: 600rpx;
  gap: $spacing-lg;
}

// ===== 登录失败的内联错误态（取代 toast） =====
// 文字走前景色而非 danger-rose：#EF4444 压 #FEE2E2 只有 3.1:1，不达正文 4.5:1。
// 错误语义由浅红底承担，文字保证可读。若日后 tokens.json 补了 danger 深色，
// 再把标题换回那一档。
.error-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-xs;
  width: 100%;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-md;
  background: $color-semantic-danger-bg;
}

.error-title {
  font-size: $typography-font-size-md;
  font-weight: 600;
  color: $color-text-primary;
}

.error-detail {
  font-size: $typography-font-size-xs;
  color: $color-text-primary;
  text-align: center;
  line-height: 1.6;
}

.bottom-hint {
  margin-top: $spacing-xl;
  text-align: center;
}

.bottom-hint text {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

// ===== 同意勾选：整行是命中区，勾选框只是视觉标记 =====
.agreement-row {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx; // iOS 44pt 触摸下限
  gap: $spacing-xs;
}

.checkbox {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40rpx;
  height: 40rpx;
  border-radius: $border-radius-sm;
  border: 2rpx solid $color-border-hairline;
  background: $color-bg-card;
  flex-shrink: 0;
}

// 选中态归暖阳橙（Warm-Dominance：选中/激活不走语义蓝）
.checkbox.checked {
  background: $color-brand-primary;
  border-color: $color-brand-primary;
}

.check-mark {
  font-size: $typography-font-size-sm;
  font-weight: 700;
  color: $color-text-on-brand;
}

.agree-text {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  line-height: 1.6;
}
</style>
