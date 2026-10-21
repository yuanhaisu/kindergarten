<script setup>
import { ref, nextTick, getCurrentInstance } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import qrcode from "qrcode-generator"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId } from "@/utils/currentStudent"
import { fetchPickupQRCode } from "@/api/parent-pickup"

const instance = getCurrentInstance()
const qrData = ref({})
const loading = ref(false)
const error = ref("")
// 画布尺寸（px，rpx 由 CSS 控制）
const QR_SIZE = 240

async function loadData() {
  loading.value = true
  error.value = ""
  try {
    await ensureLogin()
    const res = await fetchPickupQRCode(getCurrentStudentId())
    qrData.value = res.data || {}
    if (!qrData.value.qr_code) {
      error.value = "未获取到接送凭证"
      return
    }
    await nextTick()
    renderQR(qrData.value.qr_code)
  } catch (err) {
    // 后端已按错误类型返回可读提示，直接展示；兼容旧版"参数错"
    const msg = err?.message || "加载失败"
    if (msg.includes("参数") && msg.includes("指定")) {
      error.value = "请先指定「今日接走人」"
    } else if (msg.includes("授权已失效") || msg.includes("Forbidden")) {
      error.value = "所选接走人授权已失效，请重新指定"
    } else {
      error.value = msg
    }
  } finally {
    loading.value = false
  }
}

// 用 qrcode-generator 生成模块矩阵，再用 canvas 逐格绘制。
// 不走 toDataURL：小程序没有 document.createElement('canvas')。
function renderQR(text) {
  const qr = qrcode(0, "M")
  qr.addData(text)
  qr.make()

  const count = qr.getModuleCount()
  // 按画布尺寸换算每格像素，向取整避免半格模糊；留 1 格静区
  const cell = Math.max(1, Math.floor(QR_SIZE / (count + 2)))
  const drawSize = cell * (count + 2)
  const offset = Math.floor((QR_SIZE - drawSize) / 2)

  const ctx = uni.createCanvasContext("pickupQr", instance?.proxy)
  ctx.setFillStyle("#ffffff")
  ctx.fillRect(0, 0, QR_SIZE, QR_SIZE)
  ctx.setFillStyle("#1E293B")
  for (let r = 0; r < count; r++) {
    for (let c = 0; c < count; c++) {
      if (qr.isDark(r, c)) {
        ctx.fillRect(offset + (c + 1) * cell, offset + (r + 1) * cell, cell, cell)
      }
    }
  }
  ctx.draw()
}

function goDesignate() {
  uni.navigateTo({ url: "/pages/pickup/authorized" })
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>

    <template v-else>
      <!-- 未指定接走人 / 其它错误 -->
      <view v-if="error" class="card">
        <text class="title">接送二维码</text>
        <text class="err-text">{{ error }}</text>
        <button class="action-btn" @click="goDesignate">去指定今日接走人</button>
      </view>

      <view v-else class="card">
        <text class="title">{{ qrData.student?.name }} 的接送二维码</text>

        <!-- 人证对照：孩子 + 接走人 -->
        <view class="person-row">
          <view class="person-item">
            <image v-if="qrData.student?.photo" :src="qrData.student.photo" class="photo" mode="aspectFill" />
            <view v-else class="photo photo-placeholder" />
            <text class="person-name">{{ qrData.student?.name }}</text>
            <text class="person-tag">孩子</text>
          </view>
          <text class="person-arrow">→</text>
          <view class="person-item">
            <image v-if="qrData.person?.photo" :src="qrData.person.photo" class="photo" mode="aspectFill" />
            <view v-else class="photo photo-placeholder" />
            <text class="person-name">{{ qrData.person?.name || '待指定' }}</text>
            <text class="person-tag">{{ qrData.person?.relation || '接走人' }}</text>
          </view>
        </view>

        <!-- 真实二维码 -->
        <view class="qr-box">
          <canvas
            canvas-id="pickupQr"
            id="pickupQr"
            class="qr-canvas"
            :style="{ width: '240px', height: '240px' }"
          />
        </view>

        <text class="tip">请向老师出示此码，核验后一次有效</text>
        <text class="tip">有效期至 {{ (qrData.expire_at || '').slice(0, 16).replace('T', ' ') }}</text>
      </view>

      <button class="action-btn" @click="loadData">刷新二维码</button>
      <button class="action-btn outline" @click="goDesignate">更换今日接走人</button>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg $component-page-padding-x calc(150rpx + env(safe-area-inset-bottom)); background: $color-bg-page; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }

.card {
  display: flex; flex-direction: column; align-items: center;
  padding: $spacing-xl $spacing-xl; border-radius: $border-radius-xl;
  background: $color-bg-card; box-shadow: $shadow-sm; text-align: center;
}
.title { color: $color-text-primary; font-size: $typography-font-size-xl; font-weight: 700; margin-bottom: 24rpx; }
.err-text { color: $color-text-muted; font-size: $typography-font-size-md; margin-bottom: 24rpx; line-height: 1.6; }

/* 人证对照区 */
.person-row { display: flex; align-items: center; gap: $spacing-lg; margin-bottom: 28rpx; }
.person-item { display: flex; flex-direction: column; align-items: center; gap: $spacing-xs; }
.photo { width: 112rpx; height: 112rpx; border-radius: $border-radius-full; background: $color-semantic-info-bg; }
.photo-placeholder { background: $color-text-hint; opacity: 0.3; }
.person-name { font-size: $typography-font-size-sm; font-weight: 600; color: $color-text-primary; max-width: 160rpx; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.person-tag { font-size: $typography-font-size-xs; color: $color-text-muted; }
.person-arrow { font-size: $typography-font-size-xl; color: $color-text-hint; }

/* 二维码 */
.qr-box { padding: $spacing-md; border-radius: $border-radius-md; background: $color-bg-card; border: 2rpx solid rgba(106, 77, 42, 0.1); }
.qr-canvas { display: block; width: 240px; height: 240px; }

.tip { margin-top: 16rpx; color: $color-text-muted; font-size: $typography-font-size-sm; line-height: 1.6; }

.action-btn { width: 100%; height: 88rpx; margin-top: 24rpx; border-radius: $border-radius-full; background: $color-brand-primary; color: #fff; font-size: $typography-font-size-md; font-weight: 700; line-height: 88rpx; }
.action-btn.outline { background: transparent; color: $color-brand-primary; border: 2rpx solid $color-brand-primary; }
</style>
