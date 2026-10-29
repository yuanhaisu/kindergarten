<script setup>
/**
 * 我的（家长端 tab 页）
 *
 * 身份卡从 GET /common/user/profile 取真实用户；请求失败时保留本地登录缓存兜底，
 * 缓存也没有才落到错误态。菜单是静态的，加载态只换骨架，不阻塞原生底栏高亮。
 */
import { computed, ref } from "vue"
import { onShow } from "@dcloudio/uni-app"
import { clearAuth, getUser, setUser } from "@/utils/auth"
import { ensureLogin } from "@/utils/login"
import { fetchCurrentUser } from "@/api/auth"
import { syncTabBar } from "@/utils/tabbar"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形（App.vue 已全局注册字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  user: "", // user-line
  alert: "", // error-warning-line
  chevron: "", // arrow-right-s-line
  message: "", // message-3-line
  task: "", // task-line
  points: "", // medal-line
  calendar: "", // calendar-line
  report: "", // file-chart-line
  leave: "", // calendar-check-line
  pickup: "", // car-line
  auth: "", // shield-user-line
  child: "", // emotion-happy-line
  klass: "", // home-4-line
  notice: "", // notification-3-line
  feedback: "", // feedback-line
  settings: "" // settings-3-line
}

// 四组菜单保持源码顺序；「消息」按 X00 决策补进「消息与设置」组首，
// 其余 12 项的相对顺序不变。
const menuGroups = [
  {
    key: "growth",
    title: "成长与任务",
    items: [
      { title: "亲子任务", glyph: ICON.task, path: "/pages/tasks/index" },
      { title: "积分成就", glyph: ICON.points, path: "/pages/points/index" },
      { title: "班级日历", glyph: ICON.calendar, path: "/pages/calendar/index" },
      { title: "成长报告", glyph: ICON.report, path: "/pages/growth/report" }
    ]
  },
  {
    key: "daily",
    title: "在园日常",
    items: [
      { title: "请假记录", glyph: ICON.leave, path: "/pages/leave/index" },
      { title: "接送记录", glyph: ICON.pickup, path: "/pages/pickup/index" },
      { title: "授权人管理", glyph: ICON.auth, path: "/pages/pickup/authorized" }
    ]
  },
  {
    key: "child",
    title: "孩子与班级",
    items: [
      { title: "我的孩子", glyph: ICON.child, path: "/pages/student/index" },
      { title: "班级信息", glyph: ICON.klass, path: "/pages/class/index" }
    ]
  },
  {
    key: "message",
    title: "消息与设置",
    items: [
      { title: "消息", glyph: ICON.message, path: "/pages/messages/index" },
      { title: "通知公告", glyph: ICON.notice, path: "/pages/notice/index" },
      { title: "意见反馈", glyph: ICON.feedback, path: "/pages/feedback/index" },
      { title: "设置", glyph: ICON.settings, path: "/pages/settings/index" }
    ]
  }
]

const user = ref(getUser())
// 初始即视为加载中：无缓存时首帧直接是骨架，不会闪一下「家长 / 微信用户」兜底卡。
const loading = ref(true)
const retrying = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")

// 每次 onShow 都刷新，快速切 tab 时只允许最后一次请求写状态。
let loadSeq = 0

const isFirstLoading = computed(
  () => loading.value && !hasLoaded.value && !user.value && !retrying.value
)
const showError = computed(() => Boolean(loadError.value) && !user.value)
const showEmpty = computed(() => hasLoaded.value && !showError.value && !user.value)

function firstChar(value, fallback) {
  const chars = Array.from(String(value || "").trim())
  return chars[0] || fallback
}

const displayName = computed(() => String(user.value?.nickname || "").trim() || "家长")
const displayPhone = computed(() => String(user.value?.phone || "").trim() || "微信用户")
// 家长角色条目的 kindergarten_name 是契约里唯一能确认「哪个园」的字段。
const parentRole = computed(() => {
  const roles = Array.isArray(user.value?.roles) ? user.value.roles : []
  return roles.find(role => role.role === "parent") || roles[0] || null
})
const displaySchool = computed(
  () => String(parentRole.value?.kindergarten_name || "").trim() || "未绑定园所"
)
const avatarInitial = computed(() => firstChar(user.value?.nickname, "家"))

async function loadProfile() {
  const seq = ++loadSeq
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchCurrentUser()
    if (seq !== loadSeq) return
    const profile = res?.data && res.data.id ? res.data : null
    user.value = profile
    setUser(profile)
    loadError.value = ""
    hasLoaded.value = true
  } catch (err) {
    if (seq !== loadSeq) return
    // 已有缓存时保留身份卡（缓存兜底），只有一次都没拿到才切错误态。
    // 清错误文案放在成功分支：重试期间错误卡与「重试中…」保持可见，不闪正常态。
    console.warn("[profile] load failed:", err?.message || err)
    if (!user.value) loadError.value = "网络开小差了，请稍后重试"
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

async function handleRetry() {
  if (retrying.value || loading.value) return
  retrying.value = true
  await loadProfile()
  retrying.value = false
  if (showError.value) {
    uni.showToast({ title: "仍无法加载，请稍后再试", icon: "none" })
  } else {
    uni.showToast({ title: "已重新加载", icon: "none" })
  }
}

function handleRelogin() {
  // 空态没有可展示的账号信息：清本地登录态，回登录页重新走微信授权。
  clearAuth()
  uni.reLaunch({ url: "/pages/auth/login" })
}

// 家长端有 tabBar（见 pages.json），跳 tabBar 页必须用 switchTab ——
// navigateTo 到 tabBar 页会静默失败，「接送记录」点了没有任何反应。
// 列表与 pages.json → tabBar.list[].pagePath 保持一致。
const tabBarPaths = ["/pages/index/index", "/pages/growth/index", "/pages/pickup/index", "/pages/profile/index"]

function goPage(path) {
  if (tabBarPaths.includes(path)) uni.switchTab({ url: path })
  else uni.navigateTo({ url: path })
}

// switchTab 切回本页不重跑 onLoad：高亮与资料刷新都必须放 onShow。
onShow(() => {
  syncTabBar(3)
  loadProfile()
})
</script>

<template>
  <PageShell bottom-space="140rpx" max-width="430px">
    <view class="profile-page">
      <!-- 身份卡 · 加载 -->
      <view v-if="isFirstLoading" class="identity-card identity-skeleton">
        <view class="sk sk-avatar" />
        <view class="skeleton-copy">
          <view class="sk sk-line sk-line--name" />
          <view class="sk sk-line sk-line--meta" />
        </view>
      </view>

      <!-- 身份卡 · 失败（菜单保留，重试只换身份卡） -->
      <view v-else-if="showError" class="identity-card identity-state">
        <view class="state-badge is-danger">
          <text class="badge-glyph">{{ ICON.alert }}</text>
        </view>
        <view class="state-copy">
          <text class="state-title">账号信息加载失败</text>
          <text class="state-desc">{{ loadError }}</text>
        </view>
        <button class="state-btn is-ghost" :disabled="retrying" @click="handleRetry">
          {{ retrying ? "重试中…" : "重试" }}
        </button>
      </view>

      <!-- 身份卡 · 空（接口成功但没有用户） -->
      <view v-else-if="showEmpty" class="identity-card identity-state">
        <view class="state-badge is-warm">
          <text class="badge-glyph">{{ ICON.user }}</text>
        </view>
        <view class="state-copy">
          <text class="state-title">还没有读取到家长信息</text>
        </view>
        <button class="state-btn is-primary" @click="handleRelogin">重新登录</button>
      </view>

      <!-- 身份卡 · 正常 -->
      <view v-else class="identity-card">
        <view class="identity-orbit">
          <view class="orbit-glow" />
          <view class="orbit-ring orbit-ring--outer" />
          <view class="orbit-ring orbit-ring--inner" />
        </view>
        <view class="identity-avatar">
          <text class="avatar-text">{{ avatarInitial }}</text>
        </view>
        <view class="identity-body">
          <view class="identity-name-row">
            <text class="identity-name">{{ displayName }}</text>
            <view class="identity-role">
              <text class="identity-role-text">家长</text>
            </view>
          </view>
          <view class="identity-meta">
            <text class="identity-phone">{{ displayPhone }}</text>
            <text class="identity-sep">·</text>
            <text class="identity-school">{{ displaySchool }}</text>
          </view>
        </view>
      </view>

      <!-- 菜单 · 加载骨架 -->
      <view v-if="isFirstLoading" class="menu-skeleton">
        <view v-for="i in 3" :key="i" class="menu-skeleton-row">
          <view class="sk sk-tile" />
          <view class="sk sk-line sk-line--menu" :class="{ 'is-short': i === 2 }" />
          <view class="sk sk-chevron" />
        </view>
      </view>

      <!-- 菜单 · 四组 -->
      <view v-else class="menu-groups">
        <view v-for="group in menuGroups" :key="group.key" class="menu-group">
          <text class="group-title">{{ group.title }}</text>
          <view class="menu-card">
            <view
              v-for="item in group.items"
              :key="item.path"
              class="menu-row"
              hover-class="menu-row--active"
              @click="goPage(item.path)"
            >
              <view class="menu-icon">
                <text class="menu-glyph">{{ item.glyph }}</text>
              </view>
              <text class="menu-label">{{ item.title }}</text>
              <text class="menu-chevron">{{ ICON.chevron }}</text>
            </view>
          </view>
        </view>
      </view>

      <text class="profile-footer">养成星球 · 家长端</text>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 我的 · 紧凑身份卡 + 四段菜单
// 冷雾白底；暖色只落在头像、角色胶囊、图标砖与右侧轨道装饰。
// ============================================================

.identity-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-height: 192rpx;
  padding: $spacing-xl;
  overflow: hidden;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

// ---- 轨道装饰（纯视觉，不承载信息） --------------------------

.identity-orbit {
  position: absolute;
  top: -148rpx;
  right: -28rpx;
  width: 308rpx;
  height: 308rpx;
  border-radius: $border-radius-full;
  pointer-events: none;
}

.orbit-glow {
  position: absolute;
  inset: 0;
  border-radius: $border-radius-full;
  background: $color-brand-primary-light;
  opacity: 0.1;
}

.orbit-ring {
  position: absolute;
  border: 3rpx solid $color-brand-primary-light;
  border-radius: $border-radius-full;
  opacity: 0.14;
}

.orbit-ring--outer {
  inset: 48rpx;
}

.orbit-ring--inner {
  inset: 104rpx;
}

// ---- 正常态 --------------------------------------------------

.identity-avatar {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 112rpx;
  height: 112rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-sm;
}

// 设计稿 22px，超出字号 token 档位上限（3xl=40rpx）；仅头像首字使用。
.avatar-text {
  color: $color-text-on-brand;
  font-size: 44rpx;
  font-weight: $typography-font-weight-bold;
  line-height: 1;
}

.identity-body {
  position: relative;
  z-index: 1;
  flex: 1;
  min-width: 0;
}

.identity-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10rpx $spacing-md;
  min-width: 0;
}

.identity-name {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  overflow-wrap: anywhere;
}

.identity-role {
  display: inline-flex;
  align-items: center;
  flex: none;
  height: 40rpx;
  padding: 0 14rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.identity-role-text {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.identity-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2rpx 12rpx;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.identity-sep {
  color: $color-text-hint;
}

.identity-phone,
.identity-school {
  overflow-wrap: anywhere;
}

// ---- 空态 / 失败态 ------------------------------------------

.identity-state {
  flex-wrap: wrap;
  row-gap: $spacing-md;
}

.state-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 72rpx;
  height: 72rpx;
  border-radius: $border-radius-full;
}

.state-badge.is-warm {
  background: $color-brand-primary-bg;
}

.state-badge.is-danger {
  background: $color-semantic-danger-bg;
}

.badge-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.is-warm .badge-glyph {
  color: $color-brand-primary-dark;
}

.is-danger .badge-glyph {
  color: $color-semantic-danger;
}

.state-copy {
  flex: 1 1 240rpx;
  min-width: 0;
}

.state-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.state-desc {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
}

.state-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  min-width: 160rpx;
  min-height: $component-input-height;
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.state-btn.is-ghost {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.state-btn.is-primary {
  background: $gradient-brand;
  color: $color-text-on-brand;
  box-shadow: $shadow-sm;
}

.state-btn[disabled] {
  opacity: 0.6;
}

// ---- 四组菜单 ------------------------------------------------

.menu-groups {
  display: flex;
  flex-direction: column;
  gap: $spacing-xl;
  margin-top: $spacing-xl;
}

.group-title {
  display: block;
  margin: 0 0 $spacing-md 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.menu-card {
  overflow: hidden;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.menu-row {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-height: 104rpx;
  padding: 0 $spacing-xl;
  background: $color-bg-card;
}

.menu-row + .menu-row {
  border-top: 1rpx solid $color-border-subtle;
}

.menu-row--active {
  background: $color-brand-primary-bg;
}

.menu-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 60rpx;
  height: 60rpx;
  border-radius: $border-radius-lg;
  background: $color-brand-primary-bg;
}

.menu-glyph {
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.menu-label {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.menu-chevron {
  flex: none;
  color: $color-text-muted;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

// ---- 骨架 ----------------------------------------------------

.sk {
  display: block;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  animation: profile-pulse 1.6s $animation-easing-default infinite;
}

.identity-skeleton {
  align-items: center;
}

.sk-avatar {
  flex: none;
  width: 112rpx;
  height: 112rpx;
  border-radius: $border-radius-full;
}

.skeleton-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-md;
  min-width: 0;
}

.sk-line {
  width: 100%;
  height: 24rpx;
}

.sk-line--name {
  width: 46%;
  height: 28rpx;
}

.sk-line--meta {
  width: 70%;
}

.menu-skeleton {
  margin-top: $spacing-xl;
  padding: 0 $spacing-lg;
  overflow: hidden;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.menu-skeleton-row {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  height: 104rpx;
}

.menu-skeleton-row + .menu-skeleton-row {
  border-top: 1rpx solid $color-bg-page;
}

.sk-tile {
  flex: none;
  width: 60rpx;
  height: 60rpx;
  border-radius: $border-radius-lg;
}

.sk-line--menu {
  flex: 0 1 42%;
  width: 42%;
  height: 24rpx;
}

.sk-line--menu.is-short {
  flex-basis: 32%;
  width: 32%;
}

.sk-chevron {
  flex: none;
  width: 28rpx;
  height: 28rpx;
  margin-left: auto;
  border-radius: $border-radius-sm;
}

// ---- 页脚 ----------------------------------------------------

.profile-footer {
  display: block;
  margin-top: $spacing-lg;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  line-height: 1.5;
  text-align: center;
}

@keyframes profile-pulse {
  0%,
  100% {
    opacity: 0.45;
  }
  50% {
    opacity: 0.95;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
    opacity: 0.7;
  }
}
</style>
