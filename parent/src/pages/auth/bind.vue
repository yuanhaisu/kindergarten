<script setup>
/**
 * 绑定孩子 · 把「填一个码」做成一次有把握的仪式
 *
 * 只有邀请码一个字段；手机号授权在家长首页，与本页无关。
 * 缺项 / 校验 / 提交 / 服务端失败四种句子都固定在同一块输入区播报，
 * 不用转瞬即逝的 toast 承载失败原因。
 */
import { computed, nextTick, ref } from "vue"
import { onUnload } from "@dcloudio/uni-app"
import PageShell from "@/components/PageShell.vue"
import { bindStudent } from "@/api/parent-student"
import { toBindError } from "@/utils/bindError"

// remixicon 字形（App.vue 已加载字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  camera: "\u{eb31}", // camera-line
  refresh: "\u{f064}", // refresh-line
  info: "\u{ee59}", // information-line
  alert: "\u{eca1}", // error-warning-line
  check: "\u{eb7b}" // check-line
}

const MIN_LENGTH = 6
const MAX_LENGTH = 20

const GUIDE = [
  {
    index: "01",
    title: "学号就是邀请码",
    // helper 已经交代来源，这里只补「学号」这层新信息，避免同一句说两遍
    text: "孩子在园里的学号就是邀请码。"
  },
  {
    index: "02",
    title: "可以绑多个孩子",
    text: "一个家长可绑定多个孩子，也不会替代孩子已有的主家长；之后可在「我的 → 我的孩子」查看、切换或解绑。"
  },
  {
    index: "03",
    title: "提示无效怎么办",
    text: "先确认是否已绑过这个孩子，再联系班级老师核对邀请码。"
  }
]

const inviteCode = ref("")
const loading = ref(false)
const success = ref(false)
// "" | "client"（位数不足）| "server"（后端拒绝）
const errorKind = ref("")
const errorMessage = ref("")
// 打字阶段的中性提示（还差 N 位），不算错误
const hintMessage = ref("")
const refillable = ref(false)
const focusInvite = ref(false)
const focused = ref(false)

let successTimer = null
let unloaded = false

const codeLength = computed(() => Array.from(inviteCode.value).length)
const locked = computed(() => loading.value || success.value)
const canSubmit = computed(() => !locked.value && codeLength.value >= MIN_LENGTH)
const showCtaHint = computed(() => !locked.value && codeLength.value === 0)
const showRefill = computed(() => errorKind.value === "server" && refillable.value)

const reportText = computed(() => {
  if (errorMessage.value) return errorMessage.value
  if (hintMessage.value) return hintMessage.value
  return "邀请码由班级老师提供，每位孩子一个"
})

function normalizeCode(value) {
  return String(value || "")
    .toUpperCase()
    .replace(/\s+/g, "")
    .slice(0, MAX_LENGTH)
}

function clearError() {
  errorKind.value = ""
  errorMessage.value = ""
  refillable.value = false
}

function showClientError() {
  errorKind.value = "client"
  errorMessage.value = `邀请码至少 ${MIN_LENGTH} 位`
  hintMessage.value = ""
  refillable.value = false
}

function onInput(e) {
  if (locked.value) return
  inviteCode.value = normalizeCode(e?.detail?.value)
  clearError()
  hintMessage.value =
    codeLength.value > 0 && codeLength.value < MIN_LENGTH
      ? `还差 ${MIN_LENGTH - codeLength.value} 位`
      : ""
}

function onFocus() {
  focused.value = true
  // focus 是「请求聚焦」的一次性开关，聚焦后复位，避免后续渲染再次抢焦点
  focusInvite.value = false
}

// 离开输入框才把「还没输完」判成错误：打字过程中一直判红会让用户以为码错了。
function onBlur() {
  focused.value = false
  if (locked.value) return
  if (!inviteCode.value || codeLength.value >= MIN_LENGTH) return
  showClientError()
}

async function submitCode(raw, { fromScan = false } = {}) {
  if (locked.value) return
  const code = normalizeCode(raw)
  inviteCode.value = code

  if (!code) {
    clearError()
    hintMessage.value = ""
    uni.showToast({ title: "请输入邀请码", icon: "none" })
    return
  }

  // 扫码结果直接提交：客户端最少位数只是防手滑，不拦老师给的合法短码。
  if (!fromScan && code.length < MIN_LENGTH) {
    showClientError()
    return
  }

  clearError()
  hintMessage.value = ""
  loading.value = true
  try {
    await bindStudent(code)
    if (unloaded) return
    success.value = true
    successTimer = setTimeout(() => {
      successTimer = null
      // 本页是 navigateTo 进来的子页；直接打开（无栈）时退回首页。
      uni.navigateBack({
        fail: () => uni.switchTab({ url: "/pages/index/index" })
      })
    }, 1000)
  } catch (err) {
    if (unloaded) return
    const mapped = toBindError(err)
    errorKind.value = mapped.kind
    errorMessage.value = mapped.message
    refillable.value = mapped.refill
  } finally {
    loading.value = false
  }
}

function handleSubmit() {
  submitCode(inviteCode.value)
}

function handleScan() {
  if (locked.value) return
  // 设计稿的扫码层是「模拟扫码」原型（notes 已注明不申请相机）；真机上用
  // uni.scanCode 拉起微信原生扫码，关闭/取消由原生界面提供，这里补齐三条 fail 分支。
  uni.scanCode({
    success: res => {
      const code = normalizeCode(res?.result || res?.scanCode || "")
      if (!code) {
        uni.showToast({ title: "二维码识别失败", icon: "none" })
        return
      }
      // 扫到即填、即提交，没有二次确认。
      submitCode(code, { fromScan: true })
    },
    fail: err => {
      const msg = String(err?.errMsg || err?.message || "")
      if (/cancel/i.test(msg)) return
      if (/auth|permission|deny|denied|forbidden/i.test(msg)) {
        uni.showToast({ title: "未获得相机权限，请在微信设置中开启后重试", icon: "none" })
        return
      }
      uni.showToast({ title: "扫码失败，请重试", icon: "none" })
    }
  })
}

function refill() {
  if (locked.value) return
  inviteCode.value = ""
  clearError()
  hintMessage.value = ""
  focusInvite.value = false
  nextTick(() => {
    focusInvite.value = true
  })
}

onUnload(() => {
  unloaded = true
  if (successTimer) {
    clearTimeout(successTimer)
    successTimer = null
  }
})
</script>

<template>
  <PageShell max-width="430px" bottom-space="184rpx">
    <view class="bind-page">
      <!-- ===== Hero：回答「这是什么」 ===== -->
      <view class="bind-hero">
        <view class="hero-copy">
          <text class="hero-title">认领孩子</text>
          <text class="hero-desc">输入老师提供的邀请码，或扫描二维码绑定</text>
        </view>
        <!-- 设计稿的亲子连接图是内联 SVG；WXML 不允许，用同心轨道 + 星核转写 -->
        <view class="hero-art" aria-hidden="true">
          <view class="art-orbit art-orbit--outer" />
          <view class="art-orbit art-orbit--inner" />
          <view class="art-core" />
          <view class="art-satellite" />
        </view>
      </view>

      <!-- ===== 表单卡：邀请码是唯一动作 ===== -->
      <view class="bind-card">
        <view class="field-block" :class="{ 'is-busy': loading }">
          <view class="field-heading">
            <text class="field-label">邀请码</text>
            <text class="field-count">{{ codeLength }}/{{ MAX_LENGTH }}</text>
          </view>

          <input
            class="invite-input"
            :class="{ 'is-error': errorKind !== '', 'is-focused': focused }"
            :value="inviteCode"
            placeholder="请输入邀请码"
            placeholder-class="invite-placeholder"
            :maxlength="MAX_LENGTH"
            :disabled="locked"
            :focus="focusInvite"
            confirm-type="done"
            @input="onInput"
            @blur="onBlur"
            @focus="onFocus"
            @confirm="handleSubmit"
          />

          <view class="field-report" :class="{ 'is-error': errorKind !== '' }">
            <text class="report-icon">{{ errorKind ? ICON.alert : ICON.info }}</text>
            <text class="report-text">{{ reportText }}</text>
          </view>
        </view>

        <view class="submit-block">
          <view
            class="submit-btn"
            :class="{ 'is-disabled': !canSubmit, 'is-loading': loading }"
            @click="handleSubmit"
          >
            <view v-if="loading" class="spinner" />
            <text>{{ loading ? "绑定中..." : "确认绑定" }}</text>
          </view>

          <view v-if="!loading" class="submit-meta">
            <view v-if="showCtaHint" class="cta-hint">
              <view class="cta-dot" />
              <text>还差邀请码</text>
            </view>
            <view v-if="showRefill" class="refill-btn" @click="refill">
              <text class="refill-icon">{{ ICON.refresh }}</text>
              <text>重新填写</text>
            </view>
          </view>
        </view>

        <view class="or-divider" :class="{ 'is-busy': loading }">
          <view class="or-line" />
          <text class="or-label">或</text>
          <view class="or-line" />
        </view>

        <view class="scan-btn" :class="{ 'is-disabled': locked }" @click="handleScan">
          <view class="scan-icon-box">
            <text class="camera-icon">{{ ICON.camera }}</text>
          </view>
          <text>扫描二维码绑定</text>
        </view>
      </view>

      <!-- ===== 说明卡：解释关系，而不是重复 helper ===== -->
      <view class="guide-card">
        <text class="guide-title">关于邀请码</text>
        <view class="guide-list">
          <view v-for="item in GUIDE" :key="item.index" class="guide-item">
            <text class="guide-index">{{ item.index }}</text>
            <view class="guide-copy">
              <text class="guide-item-title">{{ item.title }}</text>
              <text class="guide-item-text">{{ item.text }}</text>
            </view>
          </view>
        </view>
      </view>

      <view class="page-foot">
        <text class="foot-icon">{{ ICON.info }}</text>
        <text class="foot-text">家里另一位家长也能用自己的账号绑定同一个孩子。</text>
      </view>
    </view>

    <!-- ===== 成功确认：居中深色卡片，1 秒后返回 ===== -->
    <view v-if="success" class="success-layer" role="status">
      <view class="success-card">
        <view class="success-mark">
          <text class="success-check">{{ ICON.check }}</text>
        </view>
        <text class="success-title">绑定成功</text>
        <text class="success-sub">正在返回…</text>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 绑定孩子 · 冷雾白底 + 白色软卡；暖橙只落在主按钮、星核与扫码图标。
// ============================================================

.bind-page {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

/* ---------- compact hero ---------- */

.bind-hero {
  display: flex;
  align-items: center;
  gap: 28rpx;
  padding: 12rpx 4rpx 36rpx;
}

.hero-copy {
  flex: 1;
  min-width: 0;
}

.hero-title {
  display: block;
  color: $color-text-primary;
  font-size: 56rpx; // 设计稿 clamp(26px, 7.2vw, 30px)
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.hero-desc {
  display: block;
  margin-top: 12rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-normal;
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.hero-art {
  position: relative;
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 160rpx; // 设计稿 clamp(80px, 22vw, 90px)
  height: 160rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand-soft;
  box-shadow: $shadow-md;
}

.art-orbit {
  position: absolute;
  border-radius: $border-radius-full;
}

.art-orbit--outer {
  top: 24rpx;
  right: 24rpx;
  bottom: 24rpx;
  left: 24rpx;
  border: 1rpx solid $color-brand-primary-light;
  opacity: 0.55;
}

.art-orbit--inner {
  top: 50rpx;
  right: 50rpx;
  bottom: 50rpx;
  left: 50rpx;
  border: 1rpx dashed $color-brand-primary-light;
  opacity: 0.4;
}

.art-core {
  width: 56rpx;
  height: 56rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-sm;
}

.art-satellite {
  position: absolute;
  top: 30rpx;
  right: 36rpx;
  width: 16rpx;
  height: 16rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

/* ---------- form card ---------- */

.bind-card {
  box-sizing: border-box;
  width: 100%;
  padding: $spacing-xl;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.field-block {
  min-width: 0;
  transition: opacity $animation-duration-fast $animation-easing-default;
}

.field-block.is-busy {
  opacity: 0.8;
}

.or-divider.is-busy {
  opacity: 0.8;
}

.field-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
}

.field-label {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  letter-spacing: 0.02em;
}

.field-count {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  letter-spacing: 0.02em;
}

.invite-input {
  box-sizing: border-box;
  display: block;
  width: 100%;
  height: 108rpx; // 设计稿 54px
  padding: 0 36rpx;
  border: 2rpx solid $color-text-muted;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  letter-spacing: 0.12em;
  caret-color: $color-brand-primary-dark;
  transition: border-color $animation-duration-fast $animation-easing-default;
}

.invite-input.is-focused {
  border-color: $color-brand-primary-dark;
}

.invite-input.is-error {
  border-color: $color-semantic-danger;
}

.invite-input[disabled] {
  background: $color-bg-page;
  color: $color-text-body;
  opacity: 0.65;
}

.invite-placeholder {
  color: $color-text-muted;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-normal;
  letter-spacing: 0.01em;
}

.field-report {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  min-height: 40rpx; // 设计稿 20px
  margin: $spacing-md 2rpx 0;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-normal;
  line-height: 1.55;
  transition: color $animation-duration-fast $animation-easing-default;
}

.field-report.is-error {
  padding: 12rpx 18rpx;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
  color: $color-text-primary;
  font-weight: $typography-font-weight-medium;
}

.report-icon {
  flex: none;
  margin-top: 2rpx;
  color: $color-text-muted;
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1.4;
}

.field-report.is-error .report-icon {
  color: $color-semantic-danger-strong;
}

.report-text {
  min-width: 0;
  overflow-wrap: anywhere;
}

/* ---------- primary action ---------- */

.submit-block {
  margin-top: $spacing-lg;
}

.submit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  width: 100%;
  min-height: 104rpx; // 设计稿 52px
  padding: 0 40rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  // 设计稿主按钮用深色字压橙，比白字可读；与登录页白字是刻意不同
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  letter-spacing: 0.02em;
  box-shadow: $shadow-md;
  transition:
    opacity $animation-duration-fast $animation-easing-default,
    transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.985);
  }
}

.submit-btn.is-disabled {
  border: 1rpx solid transparent;
  // 设计稿：信息不全时主按钮灰底灰字，不再保留品牌渐变
  background: $color-border-hairline;
  color: $color-text-muted;
  box-shadow: none;
}

/* 提交中保留品牌渐变，只轻微降饱和，让「正在忙」而不是「不可点」 */
.submit-btn.is-disabled.is-loading {
  border-color: transparent;
  background: $gradient-brand;
  color: $color-text-primary;
  opacity: 0.9;
  box-shadow: $shadow-md;
}

.spinner {
  flex: none;
  width: 34rpx;
  height: 34rpx;
  border: 4rpx solid currentColor;
  border-right-color: transparent;
  border-radius: $border-radius-full;
  animation: spin 0.7s linear infinite;
}

.submit-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  min-height: 48rpx;
  margin-top: $spacing-sm;
}

.cta-hint {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  letter-spacing: 0.02em;
}

.cta-dot {
  flex: none;
  width: 10rpx;
  height: 10rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-light;
}

.refill-btn {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  min-height: 88rpx; // 触控 44px
  margin: -20rpx -10rpx -20rpx auto;
  padding: 0 10rpx;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1;

  &:active {
    background: $color-bg-page;
  }
}

.refill-icon {
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

/* ---------- divider ---------- */

.or-divider {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin: $spacing-lg 0;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  letter-spacing: 0.06em;
}

.or-line {
  flex: 1;
  height: 1rpx;
  background: $color-border-hairline;
}

.or-label {
  flex: none;
}

/* ---------- scan action ---------- */

.scan-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20rpx;
  width: 100%;
  min-height: 100rpx; // 设计稿 50px
  padding: 0 36rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  letter-spacing: 0.02em;

  &:active {
    border-color: $color-brand-primary-light;
    background: $color-brand-primary-bg;
  }
}

.scan-icon-box {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 56rpx;
  height: 56rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.camera-icon {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.scan-btn.is-disabled {
  border-color: $color-border-hairline;
  background: $color-bg-page;
  color: $color-text-muted;
  pointer-events: none;
}

.scan-btn.is-disabled .scan-icon-box {
  background: $color-bg-page;
}

.scan-btn.is-disabled .camera-icon {
  color: $color-text-muted;
}

/* ---------- guidance ---------- */

.guide-card {
  box-sizing: border-box;
  width: 100%;
  margin-top: $spacing-xl;
  padding: $spacing-xl $spacing-xl $spacing-md;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
}

.guide-title {
  display: block;
  margin-bottom: $spacing-lg;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  letter-spacing: 0.01em;
}

.guide-item {
  display: flex;
  gap: 20rpx;
  padding: $spacing-lg 0;
}

.guide-item + .guide-item {
  border-top: 1rpx solid $color-border-hairline;
}

.guide-index {
  flex: none;
  width: 44rpx; // 设计稿 22px
  padding-top: 2rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.7;
  letter-spacing: 0.04em;
}

.guide-copy {
  flex: 1;
  min-width: 0;
}

.guide-item-title {
  display: block;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.6;
}

.guide-item-text {
  display: block;
  margin-top: 2rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-normal;
  line-height: 1.7;
  overflow-wrap: anywhere;
}

/* ---------- closing note ---------- */

.page-foot {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  margin: $spacing-xl 4rpx 0;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-normal;
  line-height: 1.6;
}

.foot-icon {
  flex: none;
  margin-top: 2rpx;
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1.4;
}

.foot-text {
  min-width: 0;
  overflow-wrap: anywhere;
}

/* ---------- success ---------- */

.success-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 95;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.success-card {
  width: 328rpx; // 设计稿 164px
  padding: 40rpx 32rpx 34rpx;
  border-radius: $border-radius-xl;
  background: $color-text-primary;
  box-shadow: $shadow-lg;
  text-align: center;
  animation: success-in $animation-duration-base $animation-easing-default both;
}

.success-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  margin: 0 auto $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-bg-card;
}

.success-check {
  color: $color-semantic-success;
  font-family: remixicon;
  font-size: 44rpx;
  line-height: 1;
}

.success-title {
  display: block;
  color: $color-text-on-brand;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  letter-spacing: 0.02em;
}

.success-sub {
  display: block;
  margin-top: $spacing-xs;
  color: $color-border-hairline;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-normal;
  line-height: 1.4;
}

/* ---------- motion ---------- */

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes success-in {
  from {
    opacity: 0;
    transform: translateY(12rpx) scale(0.94);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .spinner,
  .success-card {
    animation: none;
  }

  .submit-btn,
  .scan-btn {
    transition: none;
  }
}
</style>
