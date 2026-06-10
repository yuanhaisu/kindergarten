<script setup>
import { ref } from "vue"
import { bindStudent } from "@/api/parent-student"

const inviteCode = ref("")
const loading = ref(false)

async function handleBind() {
  const code = inviteCode.value.trim()
  if (!code) {
    uni.showToast({ title: "请输入邀请码", icon: "none" })
    return
  }

  loading.value = true
  try {
    const res = await bindStudent(code)
    uni.showToast({ title: "绑定成功", icon: "success" })
    setTimeout(() => {
      uni.navigateBack()
    }, 1000)
  } catch (err) {
    uni.showToast({ title: err?.message || "绑定失败", icon: "none" })
  } finally {
    loading.value = false
  }
}

function handleScan() {
  uni.scanCode({
    success(res) {
      inviteCode.value = res.result
      handleBind()
    }
  })
}
</script>

<template>
  <view class="page">
    <view class="card">
      <text class="title">绑定孩子</text>
      <text class="desc">输入老师提供的邀请码，或扫描二维码绑定</text>

      <input
        v-model="inviteCode"
        class="input"
        placeholder="请输入邀请码"
        maxlength="20"
      />

      <button class="bind-btn" :disabled="loading" @click="handleBind">
        {{ loading ? "绑定中..." : "确认绑定" }}
      </button>

      <button class="scan-btn" @click="handleScan">
        扫描二维码绑定
      </button>
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
  background: $parent-bg;
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
}

.title {
  color: $parent-text;
  font-size: 40rpx;
  font-weight: 800;
}

.desc {
  margin-top: 16rpx;
  color: $parent-muted;
  font-size: 26rpx;
  text-align: center;
}

.input {
  width: 100%;
  height: 96rpx;
  margin-top: 48rpx;
  padding: 0 28rpx;
  border: 2rpx solid rgba(106, 77, 42, 0.12);
  border-radius: 20rpx;
  background: #fff;
  font-size: 28rpx;
  box-sizing: border-box;
}

.bind-btn {
  width: 100%;
  height: 96rpx;
  margin-top: 32rpx;
  border-radius: 999rpx;
  background: $parent-blue;
  color: #fff;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 96rpx;
}

.scan-btn {
  width: 100%;
  height: 96rpx;
  margin-top: 20rpx;
  border-radius: 999rpx;
  background: transparent;
  color: $parent-blue;
  font-size: 28rpx;
  line-height: 96rpx;
}
</style>
