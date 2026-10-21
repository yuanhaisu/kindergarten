<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { bindPhoneLogin, toLoginError } from "@/utils/login"
import { getToken } from "@/utils/auth"
import AppCard from "@/components/AppCard.vue"
import AppButton from "@/components/AppButton.vue"
import BrandBlock from "@/components/BrandBlock.vue"

const submitting = ref(false)
const agreed = ref(false)
// 内联错误态 { title, detail }；null 表示无错误
const loginError = ref(null)

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
    <!-- 品牌区两端共用（BrandBlock），标识/字阶/tagline 不再各写一份 -->
    <BrandBlock />

    <AppCard custom-class="login-card">
      <view v-if="loginError" class="error-box">
        <text class="error-title">{{ loginError.title }}</text>
        <text class="error-detail">{{ loginError.detail }}</text>
      </view>

      <AppButton size="lg" :disabled="submitting" @click="handleWxLogin">
        {{ submitting ? "登录中..." : loginError ? "重新登录" : "微信授权登录" }}
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
  padding: $component-page-padding-y $component-page-padding-x;
  // 老师端页面底色是暖奶白（DESIGN.md「暖老师」），不是冷雾白
  background: $gradient-page;
}

// ===== 登录卡（AppCard 管圆角/阴影/底色） =====
.login-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: 640rpx;
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

// ===== 同意勾选：整行是命中区，勾选框只是视觉标记 =====
.agreement-row {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
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

.bottom-hint {
  margin-top: $spacing-xl;
  text-align: center;
}

.bottom-hint text {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}
</style>
