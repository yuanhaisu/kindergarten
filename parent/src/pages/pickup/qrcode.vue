<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId } from "@/utils/currentStudent"
import { fetchPickupQRCode } from "@/api/parent-pickup"

const qrData = ref({})
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchPickupQRCode(getCurrentStudentId())
    qrData.value = res.data || {}
  } catch (err) {
    uni.showToast({ title: err?.message || "加载失败", icon: "none" })
  } finally { loading.value = false }
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>
    <template v-else>
      <view class="card">
        <text class="title">{{ qrData.student?.name }} 的接送二维码</text>
        <view class="qr-placeholder">
          <text class="qr-text">{{ qrData.qr_code || "二维码加载中" }}</text>
        </view>
        <text class="tip">有效期至 {{ qrData.expire_at?.slice(0, 16) }}</text>
      </view>
      <button class="action-btn" @click="loadData">刷新二维码</button>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg 28rpx; background: $color-bg-page; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }
.card { display: flex; flex-direction: column; align-items: center; padding: 48rpx 32rpx; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); text-align: center; }
.title { color: $color-text-primary; font-size: $typography-font-size-xl; font-weight: 800; margin-bottom: 28rpx; }
.qr-placeholder { display: flex; align-items: center; justify-content: center; width: 400rpx; height: 400rpx; border: 2rpx dashed rgba(106,77,42,0.15); border-radius: $border-radius-xl; background: #fff; padding: $spacing-lg; }
.qr-text { color: $color-text-muted; font-size: $typography-font-size-xs; word-break: break-all; }
.tip { margin-top: 20rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
.action-btn { width: 100%; height: 88rpx; margin-top: 28rpx; border-radius: 999rpx; background: $parent-blue; color: #fff; font-size: $typography-font-size-md; font-weight: 700; line-height: 88rpx; }
</style>
