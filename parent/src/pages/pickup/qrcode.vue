<script setup>
import { ref, computed, nextTick, getCurrentInstance } from "vue"
import { onLoad, onShow, onHide } from "@dcloudio/uni-app"
import qrcode from "qrcode-generator"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId } from "@/utils/currentStudent"
import { relationLabel } from "@/utils/relation"
import { fetchPickupQRCode } from "@/api/parent-pickup"

const instance = getCurrentInstance()
const qrData = ref({})
const loading = ref(false)
const refreshing = ref(false)
const error = ref("")
const nowTick = ref(Date.now())
const expireAtMs = ref(0)
// 画布逻辑尺寸（px）；显示尺寸由 CSS 的 rpx 控制
const QR_SIZE = 200
// canvas 绘制 API 不接受 CSS 变量；这两个值与 token 的白色卡片 / text-primary 对齐。
const QR_BG = "#FFFFFF"
const QR_INK = "#1E293B"
let firstShow = true
let expiryTimer = 0

const isExpired = computed(() => expireAtMs.value > 0 && nowTick.value >= expireAtMs.value)
const noCredential = computed(() => error.value.includes("请先指定") || error.value === "未获取到接送凭证")
const statusLabel = computed(() => (refreshing.value ? "刷新中" : isExpired.value ? "已过期" : "有效"))
const statusClass = computed(() => (refreshing.value ? "is-info" : isExpired.value ? "is-danger" : "is-success"))
const studentName = computed(() => qrData.value.student?.name || "孩子")
const personName = computed(() => qrData.value.person?.name || "待指定")
// 关系统一走 utils/relation.js：兼容 9 个 code 与存量中文，未知值兜底「接走人」
const personRelation = computed(() => relationLabel(qrData.value.person?.relation, "接走人"))
const studentInitial = computed(() => (studentName.value || "孩")[0])
const personInitial = computed(() => (personName.value || "人")[0])
const expireLabel = computed(() => {
  if (!expireAtMs.value) return ""
  const text = formatDateTime(new Date(expireAtMs.value))
  return isExpired.value ? `已于 ${text} 过期` : `有效期至 ${text}`
})

function formatDateTime(date) {
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) return ""
  const m = date.getMonth() + 1
  const d = date.getDate()
  const hh = String(date.getHours()).padStart(2, "0")
  const mm = String(date.getMinutes()).padStart(2, "0")
  return `${m}月${d}日 ${hh}:${mm}`
}

async function loadData(isRefresh = false) {
  if (loading.value || refreshing.value) return
  if (isRefresh) {
    refreshing.value = true
  } else {
    loading.value = true
  }
  error.value = ""
  try {
    await ensureLogin()
    const res = await fetchPickupQRCode(getCurrentStudentId())
    const data = res.data || {}
    if (!data.qr_code) {
      qrData.value = {}
      error.value = "未获取到接送凭证"
      return
    }
    qrData.value = data
    const parsed = data.expire_at ? new Date(data.expire_at).getTime() : 0
    expireAtMs.value = Number.isNaN(parsed) ? 0 : parsed
    nowTick.value = Date.now()
    await nextTick()
    renderQR(data.qr_code)
    if (isRefresh) uni.showToast({ title: "二维码已刷新", icon: "none" })
  } catch (err) {
    const msg = err?.message || "加载失败"
    const friendly = msg.includes("参数") && msg.includes("指定")
      ? "请先指定「今日接走人」"
      : msg.includes("授权已失效") || msg.includes("Forbidden")
        ? "所选接走人授权已失效，请重新指定"
        : msg
    if (isRefresh && qrData.value?.qr_code) {
      uni.showToast({ title: friendly, icon: "none" })
    } else {
      qrData.value = {}
      expireAtMs.value = 0
      error.value = friendly
    }
  } finally {
    loading.value = false
    refreshing.value = false
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
  ctx.setFillStyle(QR_BG)
  ctx.fillRect(0, 0, QR_SIZE, QR_SIZE)
  ctx.setFillStyle(QR_INK)
  for (let r = 0; r < count; r++) {
    for (let c = 0; c < count; c++) {
      if (qr.isDark(r, c)) {
        ctx.fillRect(offset + (c + 1) * cell, offset + (r + 1) * cell, cell, cell)
      }
    }
  }
  ctx.draw()
}

function refresh() {
  loadData(true)
}

function goDesignate() {
  uni.navigateTo({ url: "/pages/pickup/authorized" })
}

onLoad(() => {
  loadData(false)
})

onShow(() => {
  nowTick.value = Date.now()
  if (!expiryTimer) {
    expiryTimer = setInterval(() => {
      nowTick.value = Date.now()
    }, 30000)
  }
  // 首次 onShow 紧跟在 onLoad 后面，不重复请求；从授权人页返回时重新拉取，
  // 否则换了接走人，页面仍显示旧凭证。
  if (firstShow) {
    firstShow = false
    return
  }
  loadData(false)
})

onHide(() => {
  if (expiryTimer) {
    clearInterval(expiryTimer)
    expiryTimer = 0
  }
})
</script>

<template>
  <view class="page">
    <!-- 首屏加载 -->
    <view v-if="loading" class="pass-card skeleton">
      <view class="sk-pass-head">
        <view class="sk sk-kicker" />
        <view class="sk sk-pill" />
      </view>
      <view class="sk sk-title" />
      <view class="sk-persons">
        <view class="sk sk-avatar" />
        <view class="sk sk-arrow" />
        <view class="sk sk-avatar" />
      </view>
      <view class="sk sk-qr" />
      <view class="sk sk-line" />
      <view class="sk sk-line sk-line--short" />
    </view>

    <!-- 未指定今日接走人 -->
    <view v-else-if="noCredential" class="state-panel">
      <view class="state-orb"><text>码</text></view>
      <text class="state-title">还没有接送凭证</text>
      <text class="state-copy">请先指定今日接走人，指定后即可获取二维码</text>
      <button class="primary-btn" @click="goDesignate">去指定今日接走人</button>
    </view>

    <!-- 失败 -->
    <view v-else-if="error" class="state-panel">
      <view class="state-orb state-orb--danger"><text>!</text></view>
      <text class="state-title">凭证加载失败</text>
      <text class="state-copy">{{ error }}</text>
      <button class="primary-btn" @click="loadData(false)">重试</button>
      <button class="outline-btn" @click="goDesignate">更换今日接走人</button>
      <text class="text-btn" @click="goDesignate">去指定今日接走人</text>
    </view>

    <!-- 加载/网络失败时仍保留使用说明；未指定接走人不显示 -->
    <view v-if="loading || (error && !noCredential)" class="usage-card">
      <text class="usage-title">使用说明</text>
      <view class="usage-row">
        <text class="usage-num">1</text>
        <text class="usage-text">接孩子时向老师出示此二维码</text>
      </view>
      <view class="usage-row">
        <text class="usage-num">2</text>
        <text class="usage-text">老师核验后立即失效，不能重复使用</text>
      </view>
      <view class="usage-row">
        <text class="usage-num">3</text>
        <text class="usage-text">凭证当日 23:59 过期，换人或过期后请刷新</text>
      </view>
    </view>

    <!-- 凭证卡 -->
    <template v-else-if="qrData.qr_code">
      <view class="pass-card" :class="{ 'is-expired': isExpired }">
        <view class="pass-head">
          <text class="pass-kicker">接送凭证</text>
          <text class="status-pill" :class="statusClass">{{ statusLabel }}</text>
        </view>
        <text class="pass-title">{{ studentName }}的接送二维码</text>

        <view class="person-row">
          <view class="person-item">
            <view class="avatar">
              <image v-if="qrData.student?.photo" :src="qrData.student.photo" class="avatar-img" mode="aspectFill" />
              <text v-else>{{ studentInitial }}</text>
            </view>
            <text class="person-name">{{ studentName }}</text>
            <text class="person-tag">孩子</text>
          </view>
          <text class="person-arrow">→</text>
          <view class="person-item">
            <view class="avatar avatar--person">
              <image v-if="qrData.person?.photo" :src="qrData.person.photo" class="avatar-img" mode="aspectFill" />
              <text v-else>{{ personInitial }}</text>
            </view>
            <text class="person-name">{{ personName }}</text>
            <text class="person-tag">{{ personRelation }}</text>
          </view>
        </view>

        <view class="tear">
          <view class="tear-line" />
          <view class="tear-notch tear-notch--left" />
          <view class="tear-notch tear-notch--right" />
        </view>

        <view class="qr-panel">
          <canvas canvas-id="pickupQr" id="pickupQr" class="qr-canvas" />
          <view v-if="isExpired" class="qr-mask">
            <text class="qr-mask-text">已过期 · 刷新后可用</text>
          </view>
          <view v-if="refreshing" class="qr-mask qr-mask--refresh">
            <view class="spinner" />
          </view>
        </view>

        <text class="expiry" :class="{ danger: isExpired }">{{ expireLabel }}</text>
        <text class="pass-tip">请向老师出示此码，核验后一次有效</text>
      </view>

      <view class="actions">
        <text v-if="isExpired" class="refresh-hint">刷新后即可继续使用</text>
        <button class="primary-btn" :disabled="refreshing" @click="refresh">
          {{ refreshing ? "刷新中…" : "刷新二维码" }}
        </button>
        <button class="outline-btn" @click="goDesignate">更换今日接走人</button>
      </view>

      <view class="usage-card">
        <text class="usage-title">使用说明</text>
        <view class="usage-row">
          <text class="usage-num">1</text>
          <text class="usage-text">接孩子时向老师出示此二维码</text>
        </view>
        <view class="usage-row">
          <text class="usage-num">2</text>
          <text class="usage-text">老师核验后立即失效，不能重复使用</text>
        </view>
        <view class="usage-row">
          <text class="usage-num">3</text>
          <text class="usage-text">凭证当日 23:59 过期，换人或过期后请刷新</text>
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  padding: $component-page-padding-y $component-page-padding-x calc(56rpx + env(safe-area-inset-bottom));
  background: $color-bg-page;
}

/* 凭证卡 */
.pass-card {
  position: relative;
  padding: 32rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
  overflow: hidden;
}

.pass-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}

.pass-kicker {
  font-size: $typography-font-size-sm;
  font-weight: 500;
  color: $color-text-muted;
}

.status-pill {
  min-height: 48rpx;
  padding: 0 20rpx;
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: 600;
}

.status-pill.is-success {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.status-pill.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.status-pill.is-info {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.pass-title {
  display: block;
  margin-top: 16rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: 600;
  line-height: 1.4;
}

.person-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-lg;
  margin-top: 28rpx;
}

.person-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8rpx;
  min-width: 0;
}

.avatar {
  width: 96rpx;
  height: 96rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  line-height: 1;
  overflow: hidden;
}

.avatar--person {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.avatar-img {
  width: 100%;
  height: 100%;
}

.person-name {
  max-width: 200rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  font-weight: 600;
}

.person-tag {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.person-arrow {
  color: $color-text-hint;
  font-size: $typography-font-size-xl;
}

/* 票根缺口 */
.tear {
  position: relative;
  height: 2rpx;
  margin: 28rpx -32rpx;
  background: repeating-linear-gradient(
    90deg,
    $color-border-hairline 0,
    $color-border-hairline 10rpx,
    transparent 10rpx,
    transparent 20rpx
  );
}

.tear-notch {
  position: absolute;
  top: -26rpx;
  width: 52rpx;
  height: 52rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.tear-notch--left {
  left: -26rpx;
}

.tear-notch--right {
  right: -26rpx;
}

/* 二维码 */
.qr-panel {
  position: relative;
  // 设计稿 232px 面板 + 200px 码；token 无二维码专用尺寸。
  width: 464rpx;
  height: 464rpx;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  overflow: hidden;
}

.qr-canvas {
  width: 400rpx;
  height: 400rpx;
  display: block;
}

.pass-card.is-expired .qr-canvas {
  opacity: 0.16;
}

.qr-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.qr-mask-text {
  padding: 12rpx 24rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  font-weight: 600;
  box-shadow: $shadow-sm;
}

.qr-mask--refresh {
  background: $color-bg-card;
}

.spinner {
  width: 40rpx;
  height: 40rpx;
  border: 4rpx solid $color-border-hairline;
  border-top-color: $color-brand-primary;
  border-radius: $border-radius-full;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.expiry {
  display: block;
  margin-top: 20rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  text-align: center;
}

.expiry.danger {
  color: $color-semantic-danger;
}

.pass-tip {
  display: block;
  margin-top: 8rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  text-align: center;
}

/* 操作 */
.actions {
  margin-top: 28rpx;
}

.refresh-hint {
  display: block;
  margin-bottom: 12rpx;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  text-align: center;
}

.primary-btn,
.outline-btn {
  width: 100%;
  min-height: 96rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-md;
  font-weight: 600;
  line-height: 1;
}

.primary-btn {
  border: 1rpx solid transparent;
  background: $gradient-brand;
  color: $color-text-on-brand;
  box-shadow: $shadow-md;
}

.primary-btn[disabled] {
  opacity: 0.55;
}

.outline-btn {
  margin-top: 16rpx;
  border: 1rpx solid $color-brand-primary;
  background: transparent;
  color: $color-brand-primary-dark;
}

.primary-btn::after,
.outline-btn::after {
  border: none;
}

/* 使用说明 */
.usage-card {
  margin-top: 28rpx;
  padding: 28rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.usage-title {
  display: block;
  margin-bottom: 20rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: 600;
}

.usage-row {
  display: flex;
  align-items: flex-start;
  gap: 16rpx;
  margin-top: 16rpx;
}

.usage-num {
  flex: none;
  width: 40rpx;
  height: 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: 600;
}

.usage-text {
  flex: 1;
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

/* 状态 */
.state-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 120rpx 32rpx 80rpx;
  text-align: center;
}

.state-orb {
  width: 192rpx;
  height: 192rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 32rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: 56rpx;
  font-weight: 600;
}

.state-orb--danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.state-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: 600;
  line-height: 1.4;
}

.state-copy {
  max-width: 560rpx;
  margin: 16rpx 0 40rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.text-btn {
  margin-top: 24rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
}

/* 骨架 */
.skeleton {
  padding: 32rpx;
}

.sk-pass-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
}

.sk {
  display: block;
  background: $color-border-hairline;
  border-radius: $border-radius-full;
  opacity: 0.45;
  animation: skPulse 1.6s $animation-easing-default infinite;
}

.sk-kicker {
  width: 120rpx;
  height: 20rpx;
}

.sk-pill {
  width: 96rpx;
  height: 40rpx;
}

.sk-title {
  width: 58%;
  height: 32rpx;
  margin-top: 24rpx;
}

.sk-persons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 24rpx;
  margin-top: 32rpx;
}

.sk-avatar {
  width: 96rpx;
  height: 96rpx;
  border-radius: $border-radius-full;
}

.sk-arrow {
  width: 40rpx;
  height: 16rpx;
}

.sk-qr {
  width: 464rpx;
  height: 464rpx;
  margin: 32rpx auto 24rpx;
  border-radius: $border-radius-xl;
}

.sk-line {
  width: 100%;
  height: 20rpx;
  margin-top: 16rpx;
}

.sk-line--short {
  width: 60%;
  margin-left: auto;
  margin-right: auto;
}

@keyframes skPulse {
  0%, 100% { opacity: 0.45; }
  50% { opacity: 0.22; }
}

@media (prefers-reduced-motion: reduce) {
  .sk,
  .spinner {
    animation: none;
  }
}
</style>
