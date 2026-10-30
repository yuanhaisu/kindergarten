<script setup>
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { getUser } from "@/utils/auth"
import { ensureLogin } from "@/utils/login"
import { goTab } from "@/utils/nav"
import PageShell from "@/components/PageShell.vue"

// 老师端没有 tabBar（见 src/pages.json），首页工作台只放得下 8 个磁贴，
// 这里是完整入口的聚合页。
//
// 设计稿用内联 SVG，小程序 WXML 不允许，改用 remixicon 字形（App.vue 已加载字体）。
const ICONS = {
  megaphone: "", // megaphone-line
  noticeList: "", // file-list-3-line
  message: "", // message-3-line
  sprout: "", // seedling-line
  album: "", // image-2-line
  scan: "", // qr-scan-line
  clipboard: "", // clipboard-line
  calendar: "", // calendar-2-line
  approval: "", // file-check-line
  family: "", // parent-line
  health: "", // heart-pulse-line
  chevron: "", // arrow-right-s-line
  empty: "", // apps-line
  alert: "" // error-warning-line
}

const GROUPS = [
  {
    key: "communication",
    title: "通知与沟通",
    items: [
      { title: "发布通知", desc: "向班级家长发送通知", icon: ICONS.megaphone, path: "/pages/notice/create" },
      { title: "通知管理", desc: "查看已发通知与家长已读情况", icon: ICONS.noticeList, path: "/pages/notice/index" },
      { title: "消息中心", desc: "通知、请假与系统消息", icon: ICONS.message, path: "/pages/messages/index" }
    ]
  },
  {
    key: "growth",
    title: "成长与相册",
    items: [
      { title: "成长记录", desc: "查看与添加成长记录", icon: ICONS.sprout, path: "/pages/growth/index" },
      { title: "班级相册", desc: "班级照片集", icon: ICONS.album, path: "/pages/album/index" }
    ]
  },
  {
    key: "pickup",
    title: "接送管理",
    items: [
      { title: "接送管理", desc: "扫码或手动登记接送", icon: ICONS.scan, path: "/pages/pickup/index" },
      { title: "接送记录", desc: "查看历史接送明细", icon: ICONS.clipboard, path: "/pages/pickup/records" }
    ]
  },
  {
    key: "affairs",
    title: "班级事务",
    items: [
      { title: "班级日历", desc: "班级活动与安排", icon: ICONS.calendar, path: "/pages/calendar/index" },
      { title: "请假审批", desc: "处理家长请假申请", icon: ICONS.approval, path: "/pages/leave/index" },
      { title: "亲子任务", desc: "发布与查看亲子任务", icon: ICONS.family, path: "/pages/tasks/index" }
    ]
  },
  {
    key: "health",
    title: "保健",
    doctorOnly: true,
    items: [
      { title: "入园晨检", desc: "录入晨检结果", icon: ICONS.health, path: "/pages/doctor/checkin" }
    ]
  }
]

const loading = ref(true)
const loadError = ref(false)
const roles = ref([])

// 晨检只对 doctor 角色开放：Casbin 策略里 teacher 角色**没有**
// /api/v1/doctor/* 权限（见 server/configs/rbac_policy.csv），
// 给普通老师显示这个入口只会让他们撞 403；整组隐藏，不置灰。
const isDoctor = computed(() => roles.value.some((item) => item && item.role === "doctor"))
const groups = computed(() => GROUPS.filter((group) => !group.doctorOnly || isDoctor.value))

const viewState = computed(() => {
  if (loading.value) return "loading"
  if (loadError.value) return "error"
  if (roles.value.length === 0) return "empty"
  return "normal"
})

async function loadEntries() {
  loading.value = true
  loadError.value = false
  try {
    await ensureLogin()
    const user = getUser() || {}
    roles.value = Array.isArray(user.roles) ? user.roles : []
  } catch (err) {
    roles.value = []
    loadError.value = true
    console.warn("[more] load entries failed:", err?.message || err)
  } finally {
    loading.value = false
  }
}

function retryLoad() {
  loadEntries()
}

function go(path) {
  uni.navigateTo({ url: path })
}

function backToWorkbench() {
  goTab("/pages/index/index")
}

onLoad(loadEntries)
</script>

<template>
  <PageShell max-width="430px" flush-top>
    <view class="more-page">
      <view v-if="viewState === 'normal'" class="menu-groups">
        <view v-for="group in groups" :key="group.key" class="menu-group">
          <text class="group-title">{{ group.title }}</text>
          <view class="menu-card">
            <view
              v-for="item in group.items"
              :key="item.path"
              class="menu-row"
              @click="go(item.path)"
            >
              <view class="menu-icon"><text class="icon menu-glyph">{{ item.icon }}</text></view>
              <view class="menu-copy">
                <text class="menu-label">{{ item.title }}</text>
                <text class="menu-desc">{{ item.desc }}</text>
              </view>
              <text class="icon menu-chevron">{{ ICONS.chevron }}</text>
            </view>
          </view>
        </view>
      </view>

      <view v-else-if="viewState === 'loading'" class="menu-loading">
        <text class="sr-only">正在加载功能入口</text>
        <view v-for="group in 2" :key="'sk-group-' + group" class="skeleton-group">
          <view class="skeleton-heading" />
          <view class="skeleton-card">
            <view v-for="row in 2" :key="'sk-row-' + row" class="skeleton-row">
              <view class="skeleton-icon" />
              <view class="skeleton-copy">
                <view class="skeleton-line skeleton-line--title" />
                <view class="skeleton-line skeleton-line--desc" />
              </view>
              <view class="skeleton-tail" />
            </view>
          </view>
        </view>
      </view>

      <view v-else-if="viewState === 'empty'" class="menu-state">
        <view class="state-icon"><text class="icon state-glyph">{{ ICONS.empty }}</text></view>
        <text class="state-title">暂无可用的功能入口</text>
        <view class="state-action" @click="backToWorkbench"><text>返回工作台</text></view>
      </view>

      <view v-else class="menu-state" role="alert">
        <view class="state-icon state-icon--error"><text class="icon state-glyph">{{ ICONS.alert }}</text></view>
        <text class="state-title">功能入口加载失败</text>
        <view class="state-action" @click="retryLoad"><text>重试</text></view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
.icon {
  font-family: remixicon;
  font-style: normal;
  line-height: 1;
}

.more-page {
  padding-top: $spacing-md;
}

/* ---------- 分组目录 ---------- */
.menu-group + .menu-group {
  margin-top: $spacing-xl;
}

.group-title {
  display: block;
  margin-bottom: $spacing-md;
  padding-left: $spacing-xs;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  letter-spacing: 1rpx;
  color: $color-text-muted;
}

.menu-card {
  overflow: hidden;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.menu-row {
  position: relative;
  display: grid;
  grid-template-columns: 72rpx minmax(0, 1fr) 32rpx;
  align-items: center;
  column-gap: $spacing-lg;
  min-height: 112rpx;
  padding: $spacing-md $spacing-lg $spacing-md 28rpx;
}

/* 行间分割线内缩到文字起点（14px + 36px + 12px），与设计稿一致 */
.menu-row + .menu-row::before {
  content: "";
  position: absolute;
  top: 0;
  right: $spacing-lg;
  left: 124rpx;
  height: 1rpx;
  background: $color-border-hairline;
  opacity: 0.6;
}

.menu-row:active {
  background: $color-bg-page;
}

.menu-row:active .menu-icon {
  transform: scale(0.94);
}

.menu-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  transition: transform $animation-duration-fast $animation-easing-default;
}

.menu-glyph {
  font-size: $typography-font-size-3xl;
}

.menu-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.menu-label {
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  color: $color-text-primary;
  word-break: break-all;
}

.menu-desc {
  margin-top: 2rpx;
  font-size: $typography-font-size-sm;
  line-height: 1.45;
  color: $color-text-muted;
  word-break: break-all;
}

.menu-chevron {
  justify-self: end;
  font-size: $typography-font-size-xl;
  color: $color-text-hint;
}

/* ---------- 空态 / 错误 ---------- */
.menu-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 640rpx;
  padding: $spacing-xl $spacing-lg 0;
  text-align: center;
}

.state-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-muted;
  box-shadow: $shadow-sm;
}

.state-icon--error {
  border: 0;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.state-glyph {
  font-size: 52rpx;
}

.state-title {
  margin-top: $spacing-lg;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  color: $color-text-primary;
}

.state-action {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 208rpx;
  min-height: $component-input-height;
  margin-top: $spacing-xl;
  padding: 0 40rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  color: $color-text-primary;
  box-shadow: $shadow-sm;
}

.state-action:active {
  transform: scale(0.97);
}

/* ---------- 加载骨架 ---------- */
.menu-loading {
  display: flex;
  flex-direction: column;
}

/* 设计稿把加载文案放在无障碍层（sr-only），屏幕只留骨架 */
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  margin: -1px;
  padding: 0;
  overflow: hidden;
  clip-path: inset(50%);
  border: 0;
}

.skeleton-group {
  animation: menu-skeleton-pulse 1.5s $animation-easing-default infinite;
}

.skeleton-group + .skeleton-group {
  margin-top: $spacing-xl;
  animation-delay: 0.3s;
}

.skeleton-heading {
  display: block;
  width: 136rpx;
  height: 22rpx;
  margin: 4rpx 0 $spacing-md $spacing-xs;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  opacity: 0.7;
}

.skeleton-card {
  overflow: hidden;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.skeleton-row {
  display: grid;
  grid-template-columns: 72rpx minmax(0, 1fr) 16rpx;
  align-items: center;
  column-gap: $spacing-lg;
  min-height: 112rpx;
  padding: $spacing-md $spacing-lg $spacing-md 28rpx;
}

.skeleton-row + .skeleton-row {
  position: relative;
}

.skeleton-row + .skeleton-row::before {
  content: "";
  position: absolute;
  top: 0;
  right: $spacing-lg;
  left: 124rpx;
  height: 1rpx;
  background: $color-border-hairline;
  opacity: 0.6;
}

.skeleton-icon {
  width: 72rpx;
  height: 72rpx;
  border-radius: $border-radius-xl;
  background: $color-border-subtle;
  opacity: 0.5;
}

.skeleton-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.skeleton-line {
  display: block;
  height: 16rpx;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
}

.skeleton-line--title {
  width: 34%;
  height: 20rpx;
  opacity: 0.58;
}

.skeleton-line--desc {
  width: 66%;
  margin-top: $spacing-sm;
  opacity: 0.36;
}

.skeleton-tail {
  justify-self: end;
  width: 14rpx;
  height: 14rpx;
  border-top: 3rpx solid $color-border-subtle;
  border-right: 3rpx solid $color-border-subtle;
  border-radius: $border-radius-sm;
  opacity: 0.55;
  transform: rotate(45deg);
}

@keyframes menu-skeleton-pulse {
  0%,
  100% {
    opacity: 0.55;
  }

  50% {
    opacity: 1;
  }
}
</style>
