<script setup>
/**
 * 添加授权接送人（家长端子页）
 *
 * 「核验登记表」：先回答「这个人是谁」，再回答「他能做什么」。
 * 必填项在提交前阻断并就地标出；关系走 9 值枚举的底部面板，提交 code 而非中文；
 * 审核规则提前说明 —— 提交 ≠ 生效，审核通过前不能接走孩子。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { createAuthorizedPerson } from "@/api/parent-pickup"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { RELATION_OPTIONS, relationCode, relationLabel } from "@/utils/relation"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形（App.vue 已全局注册字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  chevron: "", // arrow-right-s-line
  check: "", // check-line
  shield: "" // shield-check-line
}

// 字段严格对齐后端的 CreateAuthorizedPersonInput；开关在视图层用短 key，
// 提交时映射回 allow_* 字段。
const PERM_ROWS = [
  { key: "allowPickup", label: "可接送孩子", sub: "允许门卫核验身份后接走孩子" },
  { key: "allowViewRecord", label: "可查看接送记录", sub: "查看孩子每次被接走的时间" },
  { key: "allowViewStatus", label: "可查看在园状态", sub: "查看孩子此刻是否在园" },
  { key: "allowGrowthView", label: "可查看成长记录", sub: "查看老师发布的成长记录" },
  { key: "notifyOnPickup", label: "接送时通知我", sub: "孩子被接走时通知本账号" }
]

const form = ref({
  name: "",
  phone: "",
  // 与源码一致：原生 picker 有默认值，关系始终有值，不会出现空关系提交
  relation: "father",
  idCard: "",
  allowPickup: true,
  allowViewRecord: false,
  allowViewStatus: false,
  allowGrowthView: false,
  notifyOnPickup: true
})

// 本页唯一的首屏异步依赖：孩子姓名/班级与当前孩子落盘。
// 失败只影响上下文条的姓名，不阻断填写。
const contextLoading = ref(true)
const studentName = ref("")
const studentInitial = computed(() => (studentName.value || "孩").slice(0, 1))

const submitting = ref(false)
// 成功后到 navigateBack 之间仍有 900ms，锁住表单避免重复提交出两条授权
const submitted = ref(false)
const submitError = ref("")
const submitAttempted = ref(false)
const touched = ref({ name: false, phone: false })
const focusedField = ref("")
const sheetOpen = ref(false)

const relationLabelText = computed(() => relationLabel(form.value.relation))
const nameMissing = computed(() => !form.value.name.trim())
const phoneMissing = computed(() => form.value.phone.length !== 11)
const missingLabels = computed(() => {
  const labels = []
  if (nameMissing.value) labels.push("姓名")
  if (phoneMissing.value) labels.push("手机号")
  return labels
})
const submitBlocked = computed(() => missingLabels.value.length > 0 || !relationCode(form.value.relation))
const formLocked = computed(() => contextLoading.value || submitting.value || submitted.value)
const nameError = computed(() => (touched.value.name || submitAttempted.value) && nameMissing.value)
const phoneError = computed(() => (touched.value.phone || submitAttempted.value) && phoneMissing.value)

const submitNote = computed(() => {
  if (contextLoading.value) return "正在加载…"
  if (submitting.value) return "正在提交，请稍候"
  if (submitted.value) return "已提交，待园所审核"
  if (missingLabels.value.length > 0) return `还需填写：${missingLabels.value.join("、")}`
  return "提交后需园所审核，通过后才可接送"
})

// 失败条展开时底栏更高，页面预留同步加高，避免最后一张卡被盖住
const bottomSpace = computed(() => (submitError.value ? "344rpx" : "240rpx"))

function onNameBlur() {
  touched.value.name = true
  focusedField.value = ""
}

function onPhoneInput(e) {
  const digits = String(e.detail?.value || "").replace(/\D/g, "").slice(0, 11)
  form.value.phone = digits
  return digits
}

function onPhoneBlur() {
  touched.value.phone = true
  focusedField.value = ""
}

function onIdInput(e) {
  const value = String(e.detail?.value || "")
    .replace(/[^0-9xX]/g, "")
    .slice(0, 18)
    .replace(/x/g, "X")
  form.value.idCard = value
  return value
}

function togglePerm(key) {
  if (formLocked.value) return
  form.value[key] = !form.value[key]
}

function openRelation() {
  if (formLocked.value) return
  // 输入框聚焦时先收键盘，否则原生键盘会盖住底部面板
  uni.hideKeyboard()
  sheetOpen.value = true
}

function closeRelation() {
  sheetOpen.value = false
}

function selectRelation(code) {
  form.value.relation = code
  sheetOpen.value = false
}

async function loadContext() {
  try {
    await ensureLogin()
    const res = await fetchParentHomeOverview(getCurrentStudentId() || undefined)
    syncCurrentStudentFromOverview(res?.data)
    const stu = res?.data?.current_student
    if (stu && stu.name) studentName.value = stu.name
  } catch (err) {
    // 姓名只是辅助信息；拿不到时上下文条退回「当前孩子」，表单仍可提交。
    console.warn("[pickup-authorized-create] context failed:", err?.message || err)
  } finally {
    contextLoading.value = false
  }
}

async function handleSubmit() {
  if (formLocked.value) return
  // 设计稿口径：必填未完成时点击不提交、不弹 toast，只把缺项标在原地
  if (submitBlocked.value) {
    submitAttempted.value = true
    return
  }
  // 失败态下再次提交先收起失败条，避免「提交失败」与成功 toast 同屏
  submitError.value = ""
  submitting.value = true
  try {
    await ensureLogin()
    const sid = getCurrentStudentId()
    // 没选孩子时没有可提交的归属，直接说明原因，不要落成一个无法解释的失败条
    if (!sid) {
      uni.showToast({ title: "请先在首页选择孩子", icon: "none" })
      return
    }
    await createAuthorizedPerson(
      {
        name: form.value.name.trim(),
        phone: form.value.phone,
        relation: form.value.relation,
        id_card_number: form.value.idCard,
        allow_pickup: form.value.allowPickup,
        allow_view_record: form.value.allowViewRecord,
        allow_view_status: form.value.allowViewStatus,
        allow_growth_view: form.value.allowGrowthView,
        notify_on_pickup: form.value.notifyOnPickup
      },
      sid
    )
    uni.showToast({ title: "已提交，待园所审核", icon: "none" })
    submitted.value = true
    setTimeout(() => uni.navigateBack(), 900)
  } catch (err) {
    // http.js 的拦截器会 toast 具体原因；这里保留填写内容并提供可重试失败条
    console.warn("[pickup-authorized-create] submit failed:", err?.message || err)
    submitError.value = "提交失败，请重试"
  } finally {
    submitting.value = false
  }
}

onLoad(loadContext)
</script>

<template>
  <PageShell :bottom-space="bottomSpace" max-width="430px">
    <!-- ===== 首屏加载：上下文 + 表单骨架 ===== -->
    <view v-if="contextLoading" class="skeleton-area">
      <view class="sk-context">
        <view class="sk sk-avatar" />
        <view class="sk sk-context-line" />
      </view>
      <view class="sk-card">
        <view v-for="i in 3" :key="i" class="sk sk-input" />
      </view>
      <view class="sk-card">
        <view v-for="i in 3" :key="i" class="sk sk-row" />
      </view>
    </view>

    <!-- ===== 表单 ===== -->
    <view v-else class="form-area" :class="{ 'is-busy': submitting }">
      <!-- 孩子上下文：只读，无切换入口 -->
      <view class="child-context">
        <view class="child-avatar"><text>{{ studentInitial }}</text></view>
        <text class="child-text">
          为 <text class="child-name">{{ studentName || "当前孩子" }}</text> 添加接送人
        </text>
      </view>

      <view class="section-head">
        <text class="section-title">基本信息</text>
        <view class="section-rule" />
        <text class="section-meta">3 项必填</text>
      </view>

      <view class="card form-card">
        <view class="field" :class="{ 'is-error': nameError }">
          <text class="field-label">姓名<text class="req">*</text></text>
          <input
            v-model="form.name"
            class="field-input"
            :class="{ 'is-focus': focusedField === 'name' }"
            placeholder="请输入姓名"
            :maxlength="20"
            :disabled="formLocked"
            @focus="focusedField = 'name'"
            @blur="onNameBlur"
          />
          <text v-if="nameError" class="field-error">请填写姓名</text>
        </view>

        <view class="field" :class="{ 'is-error': phoneError }">
          <text class="field-label">手机号<text class="req">*</text></text>
          <input
            :value="form.phone"
            class="field-input"
            :class="{ 'is-focus': focusedField === 'phone' }"
            type="number"
            :maxlength="11"
            placeholder="请输入 11 位手机号"
            :disabled="formLocked"
            @input="onPhoneInput"
            @focus="focusedField = 'phone'"
            @blur="onPhoneBlur"
          />
          <text v-if="phoneError" class="field-error">请填写 11 位手机号</text>
        </view>

        <view class="field">
          <text class="field-label">与孩子关系<text class="req">*</text></text>
          <view class="relation-row" :class="{ 'is-locked': formLocked }" @click="openRelation">
            <text class="relation-value">{{ relationLabelText }}</text>
            <text class="relation-chevron">{{ ICON.chevron }}</text>
          </view>
        </view>

        <view class="field">
          <view class="field-label-row">
            <text class="field-label">身份证号</text>
            <text class="field-optional">选填</text>
          </view>
          <input
            :value="form.idCard"
            class="field-input"
            :class="{ 'is-focus': focusedField === 'idCard' }"
            :maxlength="18"
            placeholder="便于门卫核验"
            :disabled="formLocked"
            @input="onIdInput"
            @focus="focusedField = 'idCard'"
            @blur="focusedField = ''"
          />
        </view>
      </view>

      <view class="section-head">
        <text class="section-title">权限设置</text>
        <view class="section-rule" />
        <text class="section-meta">按需开启</text>
      </view>

      <view class="card perm-card">
        <view
          v-for="row in PERM_ROWS"
          :key="row.key"
          class="perm-row"
          :class="{ 'is-on': form[row.key], 'is-locked': formLocked }"
          @click="togglePerm(row.key)"
        >
          <view class="perm-copy">
            <text class="perm-label">{{ row.label }}</text>
            <text class="perm-sub">{{ row.sub }}</text>
          </view>
          <view class="switch"><view class="switch-knob" /></view>
        </view>
      </view>

      <view class="notice">
        <text class="notice-icon">{{ ICON.shield }}</text>
        <text class="notice-text">提交后需园所审核通过，授权才会生效。审核通过前，该接送人不能接走孩子。</text>
      </view>
    </view>

    <!-- ===== 固定提交栏 ===== -->
    <template #bottom>
      <view class="submit-bar">
        <view v-if="submitError" class="fail-bar">
          <text class="fail-text">提交失败，请重试</text>
          <view class="fail-retry" @click="handleSubmit"><text>重试</text></view>
        </view>
        <text class="submit-note" :class="{ 'is-blocked': submitBlocked }">{{ submitNote }}</text>
        <button
          class="submit-btn"
          :class="{ 'is-disabled': (submitBlocked || contextLoading || submitted) && !submitting, 'is-busy': submitting }"
          :disabled="formLocked"
          @click="handleSubmit"
        >
          <template v-if="submitting">
            <view class="btn-spinner" />
            <text>提交中…</text>
          </template>
          <text v-else>提交</text>
        </button>
      </view>
    </template>

    <!-- ===== 关系选择面板：9 个选项一屏可读 ===== -->
    <view v-if="sheetOpen" class="sheet-layer">
      <view class="sheet-scrim" @click="closeRelation" @touchmove.stop.prevent />
      <view class="sheet-panel">
        <view class="sheet-handle" />
        <text class="sheet-title">与孩子关系</text>
        <scroll-view scroll-y class="sheet-options">
          <view
            v-for="opt in RELATION_OPTIONS"
            :key="opt.code"
            class="sheet-option"
            :class="{ 'is-selected': form.relation === opt.code }"
            @click="selectRelation(opt.code)"
          >
            <text class="sheet-option-label">{{ opt.label }}</text>
            <text v-if="form.relation === opt.code" class="sheet-check">{{ ICON.check }}</text>
          </view>
        </scroll-view>
        <view class="sheet-cancel" @click="closeRelation"><text>取消</text></view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 添加授权接送人 · 核验登记表
// 橙色只落在开启的开关、关系值与主按钮上。
// ============================================================

.form-area {
  transition: opacity $animation-duration-base $animation-easing-default;
}

// 提交中整块变暗并拒绝输入；底栏不在 form-area 内，保持可读
.form-area.is-busy {
  opacity: 0.62;
  pointer-events: none;
}

// ---- 孩子上下文 ----------------------------------------------
.child-context {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-height: 80rpx;
  padding: 0 4rpx;
}

.child-avatar {
  flex: none;
  width: 56rpx;
  height: 56rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.child-text {
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.4;
}

.child-name {
  color: $color-text-primary;
  font-weight: $typography-font-weight-semibold;
}

// ---- 分组标题 ------------------------------------------------
.section-head {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  margin: $spacing-lg 0 $spacing-md;
  padding: 0 4rpx;
}

.section-title {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.section-rule {
  flex: 1;
  min-width: 32rpx;
  height: 1rpx;
  background: $color-border-subtle;
}

.section-meta {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.35;
}

// ---- 卡片与字段 ----------------------------------------------
.card {
  padding: 28rpx 32rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.field + .field {
  margin-top: $spacing-lg;
}

.field-label {
  display: block;
  margin-bottom: $spacing-sm;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.field-label-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: $spacing-sm;
}

.field-label-row .field-label {
  margin-bottom: 0;
}

.req {
  margin-left: 2rpx;
  color: $color-semantic-danger;
  font-weight: $typography-font-weight-semibold;
}

.field-optional {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.35;
}

.field-input {
  display: block;
  box-sizing: border-box;
  width: 100%;
  height: $component-input-height;
  padding: 0 $spacing-lg;
  border: none;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  line-height: 1.25;
  box-shadow: inset 0 0 0 1rpx $color-border-subtle;
}

.field-input.is-focus {
  box-shadow: inset 0 0 0 1rpx $color-brand-primary;
}

.field.is-error .field-input,
.field.is-error .field-input.is-focus {
  box-shadow: inset 0 0 0 1rpx $color-semantic-danger;
}

.field-error {
  display: block;
  margin-top: $spacing-sm;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  line-height: 1.35;
}

// ---- 关系行 --------------------------------------------------
.relation-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  box-sizing: border-box;
  width: 100%;
  min-height: 104rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  box-shadow: inset 0 0 0 1rpx $color-border-subtle;
}

.relation-row.is-locked {
  opacity: 0.6;
}

.relation-value {
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.relation-chevron {
  flex: none;
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  line-height: 1;
}

// ---- 权限行与开关 --------------------------------------------
.perm-card {
  padding-top: 8rpx;
  padding-bottom: 8rpx;
}

.perm-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  min-height: 112rpx;
  padding: 20rpx 0;
}

.perm-row + .perm-row {
  border-top: 1rpx solid $color-border-subtle;
}

.perm-row.is-locked {
  opacity: 0.6;
}

.perm-copy {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2rpx;
}

.perm-label {
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

// 设计稿副文案 11px，低于批次正文 13px 下限；这里按可读性取 13px（审查 P2-4）
.perm-sub {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.35;
}

.switch {
  position: relative;
  flex: none;
  width: 102rpx;
  height: 62rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
  transition: background $animation-duration-base $animation-easing-default;
}

.switch-knob {
  position: absolute;
  top: 4rpx;
  left: 4rpx;
  width: 54rpx;
  height: 54rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-base $animation-easing-default;
}

.perm-row.is-on .switch {
  background: $color-brand-primary;
}

.perm-row.is-on .switch-knob {
  transform: translateX(40rpx);
}

// ---- 审核说明 ------------------------------------------------
.notice {
  display: flex;
  align-items: flex-start;
  gap: 20rpx;
  margin-top: $spacing-lg;
  padding: 20rpx 28rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-semantic-warning-bg;
}

.notice-icon {
  flex: none;
  margin-top: 2rpx;
  color: $color-semantic-warning;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.notice-text {
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.55;
}

// ---- 骨架 ----------------------------------------------------
.sk-context {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  height: 80rpx;
}

.sk {
  display: block;
  background: $color-border-subtle;
  border-radius: $border-radius-xl;
  animation: create-sk 1.4s $animation-easing-default infinite;
}

.sk-avatar {
  flex: none;
  width: 56rpx;
  height: 56rpx;
  border-radius: $border-radius-full;
}

.sk-context-line {
  width: 256rpx;
  height: 24rpx;
  border-radius: $border-radius-full;
}

.sk-card {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  margin-top: $spacing-lg;
  padding: 28rpx 32rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sk-input {
  height: $component-input-height;
}

.sk-row {
  height: 104rpx;
}

@keyframes create-sk {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

// ---- 固定提交栏 ----------------------------------------------
.submit-bar {
  padding: 20rpx 28rpx calc(24rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid $color-border-hairline;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.fail-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
  padding: 4rpx 4rpx 4rpx 24rpx;
  border-radius: $border-radius-xl;
  background: $color-semantic-danger-bg;
}

.fail-text {
  min-width: 0;
  color: $color-semantic-danger;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.fail-retry {
  flex: none;
  min-width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: $color-semantic-danger;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
}

.submit-note {
  display: block;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.submit-note.is-blocked {
  color: $color-semantic-danger;
}

.submit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  box-sizing: border-box;
  width: 100%;
  height: 96rpx;
  margin-top: $spacing-md;
  padding: 0 32rpx;
  border: none;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  box-shadow: $shadow-sm;
}

.submit-btn::after {
  border: none;
}

.submit-btn.is-disabled {
  background: $color-bg-page;
  color: $color-text-muted;
  box-shadow: inset 0 0 0 1rpx $color-border-hairline;
}

.submit-btn.is-busy {
  background: $gradient-brand;
  color: $color-text-on-brand;
  box-shadow: $shadow-sm;
}

.btn-spinner {
  width: 32rpx;
  height: 32rpx;
  border: 4rpx solid $color-brand-tint;
  border-top-color: $color-text-on-brand;
  border-radius: $border-radius-full;
  animation: btn-spin 0.8s linear infinite;
}

@keyframes btn-spin {
  to {
    transform: rotate(360deg);
  }
}

// ---- 关系选择面板 --------------------------------------------
.sheet-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 50;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

// 遮罩用半透明黑叠层（CSS-COMPAT 认可用法），不用 color-mix
.sheet-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba(0, 0, 0, 0.45);
}

.sheet-panel {
  position: relative;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  width: 100%;
  max-width: 430px;
  margin: 0 auto;
  padding: $spacing-sm 28rpx calc(20rpx + env(safe-area-inset-bottom));
  border-radius: $border-radius-2xl $border-radius-2xl $border-radius-sm $border-radius-sm;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.sheet-handle {
  flex: none;
  width: 72rpx;
  height: 8rpx;
  margin: 4rpx auto 20rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
}

.sheet-title {
  flex: none;
  display: block;
  margin-bottom: $spacing-md;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  text-align: center;
}

// scroll-view 必须拿到确定高度才能滚动（CSS-COMPAT §2）
.sheet-options {
  height: 56vh;
}

.sheet-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  height: 96rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-xl;
}

.sheet-option.is-selected {
  background: $color-brand-primary-bg;
}

.sheet-option-label {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.sheet-option.is-selected .sheet-option-label {
  font-weight: $typography-font-weight-semibold;
}

.sheet-check {
  flex: none;
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.sheet-cancel {
  flex: none;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: $spacing-md;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

@media (prefers-reduced-motion: reduce) {
  .sk,
  .btn-spinner {
    animation: none;
  }
}
</style>
