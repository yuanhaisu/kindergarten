<script setup>
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { clearAuth, getRefreshToken, getToken, getUser } from "@/utils/auth"
import { ensureLogin } from "@/utils/login"
import { rawRequest } from "@/api/http"
import { fetchTeacherHomeOverview } from "@/api/teacher-home"
import PageShell from "@/components/PageShell.vue"
import BottomNav from "@/components/BottomNav.vue"

// 设计稿的线性图标在小程序里改为 remixicon 字形（App.vue 已加载字体）。
const ICONS = {
  building: "", // building-line
  ticket: "", // ticket-2-line
  shield: "", // shield-check-line
  settings: "", // settings-3-line
  logout: "", // logout-box-r-line
  emptyHome: "", // home-4-line
  alert: "" // error-warning-line
}

const ROLE_LABELS = {
  teacher: "主班老师",
  assistant: "配班老师",
  principal: "园长",
  doctor: "保健医"
}

const FEATURES = [
  { title: "邀请码", desc: "分享或管理邀请码", icon: ICONS.ticket },
  { title: "账号安全", desc: "修改手机号、绑定微信", icon: ICONS.shield },
  { title: "设置", desc: "关于、意见反馈", icon: ICONS.settings }
]

const user = ref(getUser() || {})
const overview = ref({})
// 今日数字区的四态：loading / normal / empty（未绑定园所）/ error
const statsState = ref("loading")
const dialogOpen = ref(false)
const loggingOut = ref(false)

const roles = computed(() => (Array.isArray(user.value?.roles) ? user.value.roles : []))

// 跨端同人是多条角色（parent / teacher / doctor），roles[0] 不一定是教师角色。
// 身份卡与班级/职务都必须以 teacher 角色为准，否则会拿到家长角色的空班级与错误职务。
function roleKey(item) {
  if (typeof item === "string") return item.trim()
  return String(item?.role || item?.code || "").trim()
}

const teacherRole = computed(
  () => roles.value.find((item) => roleKey(item) === "teacher") || null
)
const hasTeacherRole = computed(() => Boolean(teacherRole.value))
const userName = computed(() => String(user.value?.nickname || "").trim() || "老师")
const userInitial = computed(() => firstChar(userName.value) || "师")
// 教师角色没带园所名时退到任一角色的园所名（同账号跨端）；无 teacher 角色按未绑定处理
const kindergartenName = computed(() => {
  if (!hasTeacherRole.value) return "未绑定园所"
  const own = String(teacherRole.value?.kindergarten_name || "").trim()
  if (own) return own
  const fallback = roles.value.find(
    (item) => typeof item === "object" && String(item?.kindergarten_name || "").trim()
  )
  return (fallback && String(fallback.kindergarten_name).trim()) || "未绑定园所"
})
const hasKindergarten = computed(
  () => hasTeacherRole.value && kindergartenName.value !== "未绑定园所"
)
// 身份胶囊优先用 overview（与工作台同口径）；角色字段只在 overview 未返回时兜底
const className = computed(() =>
  String(
    overview.value?.class_name ||
      overview.value?.class_info?.name ||
      teacherRole.value?.class_name ||
      ""
  ).trim()
)
const roleLabel = computed(() => {
  const fromOverview = String(overview.value?.class_info?.title_label || "").trim()
  if (fromOverview) return fromOverview
  const fallback = String(teacherRole.value?.title_label || "").trim()
  if (fallback) return fallback
  return hasTeacherRole.value ? ROLE_LABELS[roleKey(teacherRole.value)] || "老师" : ""
})
// 未绑定园所时整页不再展示班级/职务；有教师角色但没班级（园长/保健医）时只展示职务
const showTags = computed(
  () => statsState.value !== "empty" && Boolean(className.value || roleLabel.value)
)

const stats = computed(() => ({
  present: countOf(overview.value?.attendance?.present),
  leave: countOf(overview.value?.attendance?.leave),
  pending: countOf(overview.value?.pending_approvals)
}))
const classMetaText = computed(() => {
  const count = countOf(overview.value?.student_count)
  return [className.value, `在园 ${count} 人`].filter(Boolean).join(" · ")
})
const statsLoading = computed(() => statsState.value === "loading")
const statsEmpty = computed(() => statsState.value === "empty")
const statsError = computed(() => statsState.value === "error")

function firstChar(text) {
  const name = String(text || "").trim()
  return name ? Array.from(name)[0] : ""
}

function countOf(value) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? Math.floor(n) : 0
}

async function loadProfile() {
  statsState.value = "loading"
  try {
    await ensureLogin()
    user.value = getUser() || {}
  } catch (err) {
    // 登录态校验失败时无法判断是否绑定园所：按失败处理，不能冒充「未绑定」
    statsState.value = "error"
    console.warn("[profile] ensure login failed:", err?.message || err)
    return
  }

  if (!hasKindergarten.value) {
    statsState.value = "empty"
    return
  }

  try {
    const res = await fetchTeacherHomeOverview()
    overview.value = res.data || {}
    statsState.value = "normal"
  } catch (err) {
    // 取数失败不能把三项数字静默归零——那会把「请求失败」显示成「今天没人来」
    statsState.value = "error"
    console.warn("[profile] load overview failed:", err?.message || err)
  }
}

function retryOverview() {
  loadProfile()
}

function refreshProfile() {
  loadProfile()
}

function onFeature() {
  // 三个设置入口都没有落地页：用「开发中」胶囊说明真实状态，点击只提示，不置灰
  uni.showToast({ title: "功能开发中", icon: "none" })
}

function openLogoutDialog() {
  if (loggingOut.value) return
  dialogOpen.value = true
}

function closeLogoutDialog() {
  if (loggingOut.value) return
  dialogOpen.value = false
}

async function confirmLogout() {
  if (loggingOut.value) return
  loggingOut.value = true
  const token = getToken()
  try {
    if (token) {
      // 用 rawRequest + 显式 Bearer：requestWithAuth 的 401 分支会抢先 reLaunch 到登录页，
      // 与「退出后回工作台」的既定去向冲突。
      await rawRequest({
        url: "/auth/logout",
        method: "POST",
        data: { refresh_token: getRefreshToken() || undefined },
        header: { Authorization: `Bearer ${token}` },
        silent: true
      })
    }
  } catch (err) {
    // 退出接口失败也清本地：用户明确要退出，不能因为网络把登录态留在本机
    console.warn("[profile] logout request failed:", err?.message || err)
  } finally {
    clearAuth()
    loggingOut.value = false
    dialogOpen.value = false
    uni.reLaunch({ url: "/pages/index/index" })
  }
}

function noop() {}

onLoad(loadProfile)
</script>

<template>
  <view class="profile-root">
    <PageShell max-width="430px" flush-top bottom-space="180rpx">
      <view class="profile-page">
        <!-- 1. 工牌式身份卡：全页唯一渐变 -->
        <view class="id-card">
          <view class="id-avatar"><text>{{ userInitial }}</text></view>
          <view class="id-body">
            <text class="id-name">{{ userName }}</text>
            <view class="id-org">
              <text class="icon id-org-icon">{{ ICONS.building }}</text>
              <text class="id-org-name">{{ kindergartenName }}</text>
            </view>
            <view v-if="showTags" class="id-tags">
              <text v-if="className" class="id-tag">{{ className }}</text>
              <text v-if="roleLabel" class="id-tag">{{ roleLabel }}</text>
            </view>
          </view>
        </view>

        <!-- 2. 今日班级 -->
        <view class="today">
          <view class="today-head">
            <text class="today-title">今日班级</text>
            <text v-if="statsState === 'normal' && classMetaText" class="today-meta">{{ classMetaText }}</text>
            <text v-else-if="statsLoading" class="today-meta">同步中…</text>
          </view>

          <view v-if="statsError" class="stat-error" role="alert">
            <view class="stat-error-ico"><text class="icon stat-error-glyph">{{ ICONS.alert }}</text></view>
            <text class="stat-error-text">今日数据加载失败</text>
            <view class="btn-soft" @click="retryOverview"><text>重试</text></view>
          </view>

          <template v-else>
            <view class="stat-card">
              <view class="stat">
                <view class="stat-v">
                  <view v-if="statsLoading" class="sk sk-num" />
                  <text v-else-if="statsEmpty" class="stat-dash">—</text>
                  <text v-else class="stat-num">{{ stats.present }}</text>
                </view>
                <text class="stat-k">今日出勤</text>
              </view>
              <view class="stat">
                <view class="stat-v">
                  <view v-if="statsLoading" class="sk sk-num" />
                  <text v-else-if="statsEmpty" class="stat-dash">—</text>
                  <text v-else class="stat-num">{{ stats.leave }}</text>
                </view>
                <text class="stat-k">今日请假</text>
              </view>
              <view class="stat">
                <view class="stat-v">
                  <view v-if="statsLoading" class="sk sk-num" />
                  <text v-else-if="statsEmpty" class="stat-dash">—</text>
                  <text v-else class="stat-num stat-num--pill">{{ stats.pending }}</text>
                </view>
                <text class="stat-k">待审批</text>
              </view>
            </view>

            <view v-if="statsEmpty" class="stat-empty">
              <view class="stat-empty-ico"><text class="icon stat-empty-glyph">{{ ICONS.emptyHome }}</text></view>
              <text class="stat-empty-text">还没有绑定园所，今日班级数据暂不可用</text>
              <view class="btn-soft" @click="refreshProfile"><text>刷新资料</text></view>
            </view>
          </template>
        </view>

        <!-- 3. 账号与设置 -->
        <view class="settings">
          <text class="group-title">账号与设置</text>
          <view class="cell-card">
            <view
              v-for="item in FEATURES"
              :key="item.title"
              class="cell"
              @click="onFeature"
            >
              <view class="cell-ico"><text class="icon cell-glyph">{{ item.icon }}</text></view>
              <view class="cell-main">
                <text class="cell-title">{{ item.title }}</text>
                <text class="cell-desc">{{ item.desc }}</text>
              </view>
              <text class="cell-badge">开发中</text>
            </view>
          </view>
        </view>

        <!-- 4. 退出登录 -->
        <view class="logout" @click="openLogoutDialog">
          <text class="icon logout-icon">{{ ICONS.logout }}</text>
          <text>退出登录</text>
        </view>
      </view>

      <template #bottom>
        <BottomNav :active="3" />
      </template>
    </PageShell>

    <!-- 居中确认框 -->
    <view
      v-if="dialogOpen"
      class="logout-overlay"
      @click="closeLogoutDialog"
      @touchmove.stop.prevent="noop"
    >
      <view class="logout-dialog" @click.stop>
        <text class="dialog-title">退出登录</text>
        <text class="dialog-sub">确定要退出当前账号吗？</text>
        <view class="dialog-actions">
          <view class="dialog-btn" @click="closeLogoutDialog"><text>取消</text></view>
          <view class="dialog-btn dialog-btn--danger" @click="confirmLogout">
            <text>{{ loggingOut ? "退出中…" : "退出登录" }}</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.icon {
  font-family: remixicon;
  font-style: normal;
  line-height: 1;
}

.profile-root {
  min-height: 100vh;
}

.profile-page {
  padding-top: $spacing-xs;
  padding-bottom: $spacing-xl;
}

/* ---------- 1. 身份卡 ---------- */
.id-card {
  display: flex;
  align-items: center;
  padding: $spacing-xl;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-2xl;
  background: $gradient-brand-soft;
  box-shadow: $shadow-sm;
}

.id-avatar {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  margin-right: $spacing-lg;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.id-body {
  flex: 1;
  min-width: 0;
}

.id-name {
  display: block;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  color: $color-text-primary;
}

.id-org {
  display: flex;
  align-items: center;
  margin-top: 4rpx;
}

.id-org-icon {
  flex: none;
  margin-right: $spacing-xs;
  font-size: $typography-font-size-base;
  color: $color-text-muted;
}

.id-org-name {
  min-width: 0;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  color: $color-text-muted;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.id-tags {
  display: flex;
  flex-wrap: wrap;
  margin-top: $spacing-md;
}

.id-tag {
  padding: 6rpx 20rpx;
  margin-right: $spacing-sm;
  margin-bottom: 4rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
  color: $color-text-body;
}

/* ---------- 2. 今日班级 ---------- */
.today {
  margin-top: $spacing-xl;
}

.today-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 64rpx;
}

.today-title {
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  color: $color-text-primary;
}

.today-meta {
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  color: $color-text-muted;
}

.stat-card {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-top: 4rpx;
  padding: $spacing-lg 0;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.stat {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0 4rpx;
}

.stat + .stat::before {
  content: "";
  position: absolute;
  top: 50%;
  left: 0;
  width: 1rpx;
  height: 52rpx;
  background: $color-border-subtle;
  transform: translateY(-50%);
}

.stat-v {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 56rpx;
}

.stat-num {
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  color: $color-text-primary;
}

.stat-num--pill {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 68rpx;
  height: 56rpx;
  padding: 0 20rpx;
  border-radius: $border-radius-full;
  background: rgba($color-brand-primary, 0.18);
  color: $color-brand-primary-dark;
}

.stat-dash {
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  color: $color-text-hint;
}

.stat-k {
  margin-top: 4rpx;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  color: $color-text-muted;
}

/* 空态：未绑定园所 */
.stat-empty {
  display: flex;
  align-items: center;
  margin-top: $spacing-md;
  padding: $spacing-md $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
}

.stat-empty-ico {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 56rpx;
  height: 56rpx;
  margin-right: $spacing-md;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
}

.stat-empty-glyph {
  font-size: $typography-font-size-xl;
}

.stat-empty-text {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-text-muted;
}

/* 失败态 */
.stat-error {
  display: flex;
  align-items: center;
  margin-top: 4rpx;
  padding: $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.stat-error-ico {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  margin-right: $spacing-md;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.stat-error-glyph {
  font-size: $typography-font-size-2xl;
}

.stat-error-text {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
  color: $color-text-body;
}

/* 软按钮：accent 浅底 + accent-deep 字（accent-wash 与 surface 同色，需派生一层） */
.btn-soft {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  padding: 0 28rpx;
  margin-left: $spacing-md;
  border: 1rpx solid rgba($color-brand-primary, 0.3);
  border-radius: $border-radius-full;
  background: rgba($color-brand-primary, 0.18);
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  color: $color-brand-primary-dark;
}

.btn-soft:active {
  transform: scale(0.96);
}

/* 加载骨架：只替换三个数字，身份卡与目录保持可用 */
.sk {
  display: block;
  border-radius: $border-radius-sm;
  background: $color-border-subtle;
  opacity: 0.45;
  animation: profile-skeleton-pulse 1.6s $animation-easing-default infinite;
}

.sk-num {
  width: 60rpx;
  height: 40rpx;
  border-radius: $border-radius-md;
}

@keyframes profile-skeleton-pulse {
  0%,
  100% {
    opacity: 0.45;
  }

  50% {
    opacity: 0.22;
  }
}

/* ---------- 3. 账号与设置 ---------- */
.settings {
  margin-top: $spacing-xl;
}

.group-title {
  display: block;
  margin: 0 0 $spacing-md $spacing-xs;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  color: $color-text-muted;
}

.cell-card {
  overflow: hidden;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.cell {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 124rpx;
  padding: 20rpx $spacing-lg;
}

/* 分隔线内缩到文字起点，微信设置页同款 */
.cell + .cell::before {
  content: "";
  position: absolute;
  top: 0;
  left: 116rpx;
  right: $spacing-lg;
  height: 1rpx;
  background: $color-border-hairline;
}

/* 设计稿用 --surface-warm 作按下态，但它与卡片同色 → 改用页面底色才可见 */
.cell:active {
  background: $color-bg-page;
}

.cell-ico {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 68rpx;
  height: 68rpx;
  margin-right: $spacing-lg;
  border-radius: $border-radius-xl;
  background: rgba($color-brand-primary, 0.18);
  color: $color-brand-primary-dark;
}

.cell-glyph {
  font-size: $typography-font-size-2xl;
}

.cell-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.cell-title {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.45;
  color: $color-text-primary;
}

.cell-desc {
  font-size: $typography-font-size-sm;
  line-height: 1.45;
  color: $color-text-muted;
}

.cell-badge {
  flex: none;
  padding: 6rpx 16rpx;
  margin-left: $spacing-md;
  border: 1rpx solid rgba($color-text-body, 0.16);
  border-radius: $border-radius-full;
  background: rgba($color-text-body, 0.08);
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  color: $color-text-muted;
}

/* ---------- 4. 退出登录 ---------- */
.logout {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 108rpx;
  margin-top: $spacing-xl;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  color: $color-semantic-danger;
}

.logout:active {
  background: $color-bg-page;
  transform: scale(0.99);
}

.logout-icon {
  margin-right: $spacing-md;
  font-size: $typography-font-size-2xl;
}

/* ---------- 居中确认框 ---------- */
.logout-overlay {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 56rpx;
  background: rgba($color-text-primary, 0.38);
}

.logout-dialog {
  width: 100%;
  max-width: 600rpx;
  padding: $spacing-xl $spacing-xl 0;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.dialog-title {
  display: block;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  text-align: center;
  color: $color-text-primary;
}

.dialog-sub {
  display: block;
  margin-top: $spacing-md;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  text-align: center;
  color: $color-text-muted;
}

.dialog-actions {
  display: flex;
  margin: $spacing-xl (-$spacing-xl) 0;
  border-top: 1rpx solid $color-border-hairline;
}

.dialog-btn {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  height: 96rpx;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
}

.dialog-btn + .dialog-btn {
  border-left: 1rpx solid $color-border-hairline;
}

.dialog-btn--danger {
  font-weight: $typography-font-weight-semibold;
  color: $color-semantic-danger;
}

.dialog-btn:active {
  background: $color-bg-page;
}
</style>
