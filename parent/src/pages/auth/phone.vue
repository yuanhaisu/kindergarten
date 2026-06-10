<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getUser } from "@/utils/auth"

const loading = ref(false)
const user = ref(null)

async function handleLogin() {
  loading.value = true
  try {
    await ensureLogin()
    user.value = getUser()
    uni.showToast({ title: "登录成功", icon: "success" })
    setTimeout(() => {
      uni.reLaunch({ url: "/pages/index/index" })
    }, 800)
  } catch (err) {
    uni.showToast({ title: err?.message || "登录失败", icon: "none" })
  } finally {
    loading.value = false
  }
}

onLoad(() => {
  user.value = getUser()
})
</script>

<template>
  <view class="page">
    <view class="card">
      <text class="logo">养成星球</text>
      <text class="desc">幼儿园家园共育平台</text>

      <view v-if="user" class="user-info">
        <text class="welcome">欢迎回来</text>
        <text class="name">{{ user.nickname || "家长" }}</text>
      </view>

      <button class="login-btn" :disabled="loading" @click="handleLogin">
        {{ loading ? "登录中..." : "微信一键登录" }}
      </button>

      <text class="tip">点击登录即表示同意《用户协议》和《隐私政策》</text>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 48rpx;
  background: linear-gradient(180deg, #f0f4ff 0%, #f7f3ea 60%);
}

.card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: 600rpx;
  padding: 64rpx 48rpx;
  border-radius: 40rpx;
  background: $parent-card;
  box-shadow: $parent-shadow;
  text-align: center;
}

.logo {
  color: $parent-text;
  font-size: 52rpx;
  font-weight: 800;
}

.desc {
  margin-top: 16rpx;
  color: $parent-muted;
  font-size: 28rpx;
}

.user-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 48rpx;
}

.welcome {
  color: $parent-muted;
  font-size: 26rpx;
}

.name {
  margin-top: 8rpx;
  color: $parent-text;
  font-size: 36rpx;
  font-weight: 700;
}

.login-btn {
  width: 100%;
  height: 96rpx;
  margin-top: 56rpx;
  border-radius: 999rpx;
  background: $parent-blue;
  color: #fff;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 96rpx;
}

.tip {
  margin-top: 24rpx;
  color: $parent-muted;
  font-size: 22rpx;
}
</style>
