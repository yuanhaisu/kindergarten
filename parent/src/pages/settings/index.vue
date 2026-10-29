<script setup>
/**
 * 设置（家长端子页）
 *
 * 8 个开关（6 通知 + 2 隐私）真实读写 GET/PUT /parent/settings；切换采用乐观更新，
 * 行内「同步中…」期间禁点，PUT 失败回滚并 toast。退出登录只在本页：先调
 * POST /auth/logout，再清本地并 reLaunch 登录页，失败也清本地并提示。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { clearAuth } from "@/utils/auth"
import { fetchSettings, normalizeSettings, updateSettings } from "@/api/parent-settings"
import { logout } from "@/api/parent-user"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形（App.vue 已全局注册字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  alert: "" // error-warning-line
}

// 原生 switch 的 color 只能传色值，不认 CSS 变量；对应 $color-brand-primary（parent）。
const SWITCH_COLOR = "#F97316"

const NOTIFICATION_ROWS = [
  { key: "notice_push", label: "通知推送" },
  { key: "leave_push", label: "请假通知" },
  { key: "pickup_push", label: "接送通知" },
  { key: "growth_push", label: "成长记录" },
  { key: "task_push", label: "任务提醒" },
  { key: "announcement_push", label: "公告通知" }
]

const PRIVACY_ROWS = [
  { key: "show_phone_to_teacher", label: "向老师展示手机号" },
  { key: "allow_growth_share", label: "允许展示成长记录" }
]

const settings = ref(normalizeSettings(null))
const syncing = ref({})
const loading = ref(true)
const retrying = ref(false)
const loggingOut = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")

const isFirstLoading = computed(
  () => loading.value && !hasLoaded.value && !retrying.value
)
const isSyncing = key => Boolean(syncing.value[key])

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchSettings()
    settings.value = normalizeSettings(res?.data)
    loadError.value = ""
    hasLoaded.value = true
  } catch (err) {
    // 失败不静默：整页错误态 + 重试；退出卡仍渲染，用户不会被困在本页。
    // 清错误文案放在成功分支：重试期间错误卡与「重试中…」保持可见，不闪默认值。
    console.warn("[settings] load failed:", err?.message || err)
    loadError.value = "网络不太稳定，请稍后再试"
  } finally {
    loading.value = false
  }
}

async function handleRetry() {
  if (retrying.value || loading.value) return
  retrying.value = true
  await loadData()
  retrying.value = false
  if (loadError.value) {
    uni.showToast({ title: "仍无法加载，请稍后再试", icon: "none" })
  } else {
    uni.showToast({ title: "设置已刷新", icon: "none" })
  }
}

function markSyncing(key, value) {
  const next = { ...syncing.value }
  if (value) next[key] = true
  else delete next[key]
  syncing.value = next
}

// 只提交本次改动的键（契约支持部分更新）：同一分组里其他键的并发切换
// 不会因为整组回传而互相覆盖。
async function applySwitch(group, key, nextValue) {
  if (isSyncing(key)) return
  const previous = Boolean(settings.value[group][key])
  if (nextValue === previous) return

  settings.value = {
    ...settings.value,
    [group]: { ...settings.value[group], [key]: nextValue }
  }
  markSyncing(key, true)
  try {
    await updateSettings({ [group]: { [key]: nextValue } })
    uni.showToast({ title: "已更新", icon: "none" })
  } catch (err) {
    settings.value = {
      ...settings.value,
      [group]: { ...settings.value[group], [key]: previous }
    }
    uni.showToast({ title: err?.message || "更新失败", icon: "none" })
  } finally {
    markSyncing(key, false)
  }
}

function toggleRow(group, key) {
  applySwitch(group, key, !settings.value[group][key])
}

function onSwitchChange(group, key, event) {
  applySwitch(group, key, Boolean(event?.detail?.value))
}

function confirmLogout() {
  return new Promise(resolve => {
    uni.showModal({
      title: "退出登录",
      content: "退出后需重新微信授权登录。",
      success: res => resolve(Boolean(res.confirm)),
      fail: () => resolve(false)
    })
  })
}

async function handleLogout() {
  if (loggingOut.value) return
  if (!(await confirmLogout())) return

  loggingOut.value = true
  let synced = true
  try {
    await logout()
  } catch (err) {
    // token 可能已过期：服务端确认失败不阻止本地退出。
    synced = false
    console.warn("[settings] logout failed:", err?.message || err)
  }
  clearAuth()
  uni.reLaunch({
    url: "/pages/auth/login",
    complete() {
      if (!synced) {
        // 导航会结束当前 toast，等登录页接管后再提示，避免提示被吞掉。
        setTimeout(() => {
          uni.showToast({ title: "已清除本地登录，请重新登录", icon: "none" })
        }, 400)
      }
    }
  })
}

onLoad(loadData)
</script>

<template>
  <PageShell bottom-space="48rpx" max-width="430px">
    <view class="settings-page">
      <!-- 加载：组标题条 + 三行骨架，退出卡不渲染（与源码 loading 分支一致） -->
      <view v-if="isFirstLoading" class="loading-state">
        <view class="sk sk-heading" />
        <view class="settings-card">
          <view v-for="i in 3" :key="i" class="skeleton-row">
            <view
              class="sk skeleton-bar"
              :class="i === 2 ? 'is-short' : i === 3 ? 'is-wide' : 'is-medium'"
            />
            <view class="sk skeleton-switch" />
          </view>
        </view>
      </view>

      <!-- 失败：重试；退出卡仍在下方 -->
      <view v-else-if="loadError" class="error-state">
        <view class="error-icon">
          <text class="error-glyph">{{ ICON.alert }}</text>
        </view>
        <text class="error-title">设置加载失败</text>
        <text class="error-desc">{{ loadError }}</text>
        <button class="retry-btn" :disabled="retrying" @click="handleRetry">
          {{ retrying ? "重试中…" : "重试" }}
        </button>
      </view>

      <!-- 正常 -->
      <template v-else>
        <view class="settings-group">
          <text class="group-title">消息通知</text>
          <view class="settings-card">
            <view
              v-for="row in NOTIFICATION_ROWS"
              :key="row.key"
              class="switch-row"
              hover-class="switch-row--active"
              @click="toggleRow('notification', row.key)"
            >
              <text class="row-label" :class="{ 'is-muted': isSyncing(row.key) }">
                {{ row.label }}
              </text>
              <!-- 开关自己处理 tap：阻止冒泡到整行，避免一次点击翻转两次 -->
              <view class="row-tail" @click.stop>
                <text v-if="isSyncing(row.key)" class="row-sync">同步中…</text>
                <switch
                  class="row-switch"
                  :checked="settings.notification[row.key]"
                  :color="SWITCH_COLOR"
                  :disabled="isSyncing(row.key)"
                  @change="onSwitchChange('notification', row.key, $event)"
                />
              </view>
            </view>
          </view>
          <text class="group-note">关闭后消息仍会保留在消息列表，只是不再提醒。</text>
        </view>

        <view class="settings-group">
          <text class="group-title">隐私</text>
          <view class="settings-card">
            <view
              v-for="row in PRIVACY_ROWS"
              :key="row.key"
              class="switch-row"
              hover-class="switch-row--active"
              @click="toggleRow('privacy', row.key)"
            >
              <text class="row-label" :class="{ 'is-muted': isSyncing(row.key) }">
                {{ row.label }}
              </text>
              <view class="row-tail" @click.stop>
                <text v-if="isSyncing(row.key)" class="row-sync">同步中…</text>
                <switch
                  class="row-switch"
                  :checked="settings.privacy[row.key]"
                  :color="SWITCH_COLOR"
                  :disabled="isSyncing(row.key)"
                  @change="onSwitchChange('privacy', row.key, $event)"
                />
              </view>
            </view>
          </view>
          <text class="group-note">仅影响老师和同班家长看到的信息。</text>
        </view>
      </template>

      <!-- 退出登录：加载态不渲染（危险按钮不该悬在骨架后面），错误态保留 -->
      <view v-if="!isFirstLoading" class="logout-wrap">
        <button class="logout-button" :disabled="loggingOut" @click="handleLogout">
          {{ loggingOut ? "退出中…" : "退出登录" }}
        </button>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 设置 · 安静的开关板
// 橙色只做「开着」的信号；退出登录单独一张卡，红色文字不抢主视觉。
// ============================================================

.settings-page {
  display: flex;
  flex-direction: column;
}

.settings-group + .settings-group {
  // 设计稿 24px，超出 spacing token 上限（xl=32rpx）；仅用于两个分组之间。
  margin-top: 48rpx;
}

.group-title {
  display: block;
  padding: 0 $spacing-xl;
  margin: 0 0 $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.settings-card {
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.group-note {
  display: block;
  padding: 0 $spacing-xl;
  margin-top: $spacing-md;
  color: $color-text-muted;
  // 说明性正文不低于 13px（设计审查 P2-3）
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

// ---- 开关行 --------------------------------------------------

.switch-row {
  position: relative;
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-height: 112rpx;
  padding: $spacing-md $spacing-xl;
}

.row-label {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.row-label.is-muted {
  color: $color-text-muted;
}

.row-tail {
  display: flex;
  align-items: center;
  flex: none;
  gap: $spacing-md;
}

.row-sync {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.row-switch {
  flex: none;
}

.switch-row + .switch-row::before {
  content: "";
  position: absolute;
  top: 0;
  right: $spacing-xl;
  left: $spacing-xl;
  height: 1rpx;
  background: $color-border-subtle;
}

.switch-row--active {
  background: $color-brand-primary-bg;
}

// ---- 退出登录卡 ----------------------------------------------

.logout-wrap {
  margin-top: 48rpx;
}

.logout-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 112rpx;
  padding: $spacing-md $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  // 浅红底上的 #EF4444 对比度不足，危险语义由文案承担，文字用更深的红。
  color: $color-semantic-danger-strong;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
  text-align: center;
}

.logout-button[disabled] {
  opacity: 0.6;
}

// ---- 错误态 --------------------------------------------------

.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 120rpx $spacing-xl $spacing-xl;
  text-align: center;
}

.error-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 128rpx;
  height: 128rpx;
  margin-bottom: $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
}

.error-glyph {
  color: $color-semantic-danger;
  font-family: remixicon;
  font-size: 48rpx;
  line-height: 1;
}

.error-title {
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.error-desc {
  max-width: 500rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.retry-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 264rpx;
  min-height: $component-input-height;
  margin-top: 40rpx;
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-md;
  // 橙底深字（约 5.4:1），与设计稿一致
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.retry-btn[disabled] {
  opacity: 0.68;
  box-shadow: none;
}

// ---- 骨架 ----------------------------------------------------

.sk {
  display: block;
  background: $color-border-subtle;
  border-radius: $border-radius-full;
  animation: settings-pulse 1.6s $animation-easing-default infinite;
}

.sk-heading {
  width: 128rpx;
  height: 24rpx;
  margin: 0 0 $spacing-md $spacing-xl;
}

.skeleton-row {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-xl;
  min-height: 112rpx;
  padding: 0 $spacing-xl;
}

.skeleton-row + .skeleton-row::before {
  content: "";
  position: absolute;
  top: 0;
  right: $spacing-xl;
  left: $spacing-xl;
  height: 1rpx;
  background: $color-border-subtle;
}

.skeleton-bar {
  height: 24rpx;
}

.skeleton-bar.is-short {
  width: 136rpx;
}

.skeleton-bar.is-medium {
  width: 176rpx;
}

.skeleton-bar.is-wide {
  width: 208rpx;
}

.skeleton-switch {
  flex: none;
  width: 96rpx;
  height: 60rpx;
}

@keyframes settings-pulse {
  0%,
  100% {
    opacity: 0.44;
  }
  50% {
    opacity: 0.92;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
    opacity: 0.7;
  }
}
</style>
