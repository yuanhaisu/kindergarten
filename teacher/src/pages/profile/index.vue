<script setup>
import { ref, computed } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { getUser, clearAuth } from "@/utils/auth"
import { ensureLogin } from "@/utils/login"
import { fetchTeacherHomeOverview } from "@/api/teacher-home"

const user = ref(getUser() || {})
const stats = ref({ present: 0, leave: 0, pending: 0 })
const loading = ref(false)

const userInitial = computed(() => {
  const name = user.value?.nickname || "老"
  return name[0]
})

const userName = computed(() => user.value?.nickname || "老师")

const schoolName = computed(() => {
  const role = user.value?.roles?.[0]
  return role?.kindergarten_name || "未绑定园所"
})

const className = computed(() => {
  const role = user.value?.roles?.[0]
  return role?.class_name || ""
})

const roleLabel = computed(() => {
  const role = user.value?.roles?.[0]
  if (!role) return ""
  const labels = { teacher: "主班老师", assistant: "配班老师", principal: "园长" }
  return labels[role.role] || "老师"
})

const hasRole = computed(() => user.value?.roles?.length > 0)

const featureItems = [
  { title: "邀请码", desc: "分享或管理邀请码", icon: "邀", bg: "#F0FDF4", fg: "#10B981", path: "" },
  { title: "账号安全", desc: "修改手机号、绑定微信", icon: "安", bg: "#FFF7ED", fg: "#F97316", path: "" },
  { title: "设置", desc: "关于、意见反馈", icon: "设", bg: "#F1F5F9", fg: "#64748B", path: "" }
]

const navItems = [
  { label: "首页", icon: "首", path: "/pages/index/index" },
  { label: "班级", icon: "班", path: "/pages/class/index" },
  { label: "消息", icon: "消", path: "/pages/messages/index" },
  { label: "我的", icon: "我", path: "/pages/profile/index", active: true }
]

async function loadProfile() {
  loading.value = true
  try {
    await ensureLogin()
    user.value = getUser() || {}
    const res = await fetchTeacherHomeOverview()
    const data = res.data || {}
    stats.value = {
      present: data.attendance?.present ?? 0,
      leave: data.attendance?.leave ?? 0,
      pending: data.pending_approvals ?? 0
    }
  } catch (e) {
    // silently fallback to zero stats
  } finally {
    loading.value = false
  }
}

function handleFeature(item) {
  if (!item.path) {
    uni.showToast({ title: "功能开发中", icon: "none" })
    return
  }
  uni.navigateTo({ url: item.path })
}

function navigateTo(path) {
  if (path === "/pages/profile/index") return
  uni.navigateTo({ url: path })
}

function confirmLogout() {
  uni.showModal({
    title: "退出登录",
    content: "确定要退出当前账号吗？",
    success(res) {
      if (res.confirm) {
        clearAuth()
        uni.reLaunch({ url: "/pages/index/index" })
      }
    }
  })
}

onLoad(() => {
  loadProfile()
})
</script>

<template>
  <view class="page">
    <!-- 个人信息头部 -->
    <view class="header-area">
      <view class="status-bar" :style="{ height: 'var(--status-bar-height)' }" />

      <view class="profile-row">
        <view class="avatar">
          <text class="avatar-text">{{ userInitial }}</text>
        </view>
        <view class="user-center">
          <text class="user-name">{{ userName }}</text>
          <view class="school-row">
            <text class="school-icon">园</text>
            <text class="school-name">{{ schoolName }}</text>
          </view>
        </view>
      </view>

      <view v-if="hasRole" class="tag-row">
        <view v-if="className" class="tag">
          <text class="tag-icon">班</text>
          <text class="tag-text">{{ className }}</text>
        </view>
        <view v-if="roleLabel" class="tag">
          <text class="tag-icon">师</text>
          <text class="tag-text">{{ roleLabel }}</text>
        </view>
      </view>
    </view>

    <!-- 数据概览卡片 -->
    <view class="stats-card">
      <view class="stat-item">
        <text class="stat-num is-blue">{{ stats.present }}</text>
        <text class="stat-label">今日出勤</text>
      </view>
      <view class="stat-divider" />
      <view class="stat-item">
        <text class="stat-num is-green">{{ stats.leave }}</text>
        <text class="stat-label">今日请假</text>
      </view>
      <view class="stat-divider" />
      <view class="stat-item">
        <text class="stat-num is-amber">{{ stats.pending }}</text>
        <text class="stat-label">待审批</text>
      </view>
    </view>

    <!-- 功能列表 -->
    <view class="feature-card">
      <view
        v-for="item in featureItems"
        :key="item.title"
        class="feature-item"
        @click="handleFeature(item)"
      >
        <view class="feature-icon-box" :style="{ backgroundColor: item.bg }">
          <text class="feature-icon" :style="{ color: item.fg }">{{ item.icon }}</text>
        </view>
        <view class="feature-body">
          <text class="feature-title">{{ item.title }}</text>
          <text class="feature-desc">{{ item.desc }}</text>
        </view>
        <text class="feature-arrow">></text>
      </view>
    </view>

    <!-- 退出登录 -->
    <view class="logout-card" @click="confirmLogout">
      <text class="logout-icon">退</text>
      <text class="logout-text">退出登录</text>
    </view>

    <!-- 底部占位 -->
    <view class="bottom-placeholder" />

    <!-- 底部导航栏 -->
    <view class="bottom-nav">
      <view
        v-for="item in navItems"
        :key="item.label"
        class="nav-item"
        :class="{ active: item.active }"
        @click="navigateTo(item.path)"
      >
        <text class="nav-icon" :class="{ 'active-icon': item.active }">{{ item.icon }}</text>
        <text class="nav-label" :class="{ 'active-label': item.active }">{{ item.label }}</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  box-sizing: border-box;
  width: 100%;
  max-width: 430px;
  margin: 0 auto;
  background: #F8FAFC;
}

// ===== 个人信息头部 =====
.header-area {
  padding: 0 46rpx 62rpx;
  background: linear-gradient(135deg, $color-semantic-info 0%, $color-semantic-info 100%);
}

.status-bar {
  width: 100%;
}

.profile-row {
  display: flex;
  align-items: center;
  gap: 31rpx;
  margin-top: 46rpx;
}

.avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 123rpx;
  height: 123rpx;
  border-radius: $border-radius-full;
  background: rgba(255, 255, 255, 0.2);
  border: 6rpx solid rgba(255, 255, 255, 0.4);
}

.avatar-text {
  font-size: 54rpx;
  font-weight: 700;
  color: #fff;
}

.user-center {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
}

.user-name {
  font-size: 42rpx;
  font-weight: 700;
  color: #fff;
}

.school-row {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.school-icon {
  font-size: 27rpx;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.8);
}

.school-name {
  font-size: 25rpx;
  color: rgba(255, 255, 255, 0.85);
}

.tag-row {
  display: flex;
  align-items: center;
  gap: 15rpx;
  margin-top: 31rpx;
}

.tag {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  padding: $spacing-sm 23rpx;
  border-radius: $border-radius-full;
  background: rgba(255, 255, 255, 0.2);
}

.tag-icon {
  font-size: 27rpx;
  font-weight: 600;
  color: #fff;
}

.tag-text {
  font-size: 25rpx;
  font-weight: 500;
  color: #fff;
}

// ===== 数据概览卡片 =====
.stats-card {
  display: flex;
  align-items: center;
  margin: 38rpx 46rpx;
  padding: 38rpx;
  border-radius: 31rpx;
  background: #fff;
  box-shadow: 0 4rpx 23rpx rgba(0, 0, 0, 0.06);
}

.stat-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-xs;
}

.stat-num {
  font-size: 54rpx;
  font-weight: 800;
}

.is-blue { color: $color-semantic-info; }
.is-green { color: #10B981; }
.is-amber { color: #F59E0B; }

.stat-label {
  font-size: 23rpx;
  color: #94A3B8;
}

.stat-divider {
  width: 4rpx;
  height: 92rpx;
  background: #E2E8F0;
  border-radius: 2rpx;
}

// ===== 功能列表 =====
.feature-card {
  margin: 0 46rpx;
  border-radius: 31rpx;
  background: #fff;
  box-shadow: 0 4rpx 23rpx rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.feature-item {
  display: flex;
  align-items: center;
  padding: 31rpx 38rpx;
  border-bottom: 1rpx solid #F1F5F9;

  &:last-child {
    border-bottom: 0;
  }
}

.feature-icon-box {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 77rpx;
  height: 77rpx;
  border-radius: 23rpx;
}

.feature-icon {
  font-size: 38rpx;
  font-weight: 700;
}

.feature-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  margin-left: 23rpx;
}

.feature-title {
  font-size: 29rpx;
  font-weight: 500;
  color: $color-text-primary;
}

.feature-desc {
  font-size: 23rpx;
  color: #94A3B8;
}

.feature-arrow {
  font-size: 38rpx;
  color: #CBD5E1;
  font-weight: 300;
}

// ===== 退出登录 =====
.logout-card {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  margin: 62rpx 46rpx 0;
  padding: 27rpx 0;
  border-radius: 31rpx;
  background: #fff;
  box-shadow: 0 4rpx 23rpx rgba(0, 0, 0, 0.06);
}

.logout-icon {
  font-size: 35rpx;
  font-weight: 600;
  color: #EF4444;
}

.logout-text {
  font-size: 29rpx;
  font-weight: 500;
  color: #EF4444;
}

// ===== 底部占位 =====
.bottom-placeholder {
  height: 140rpx;
}

// ===== 底部导航栏 =====
.bottom-nav {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  display: flex;
  justify-content: space-between;
  max-width: 430px;
  margin: 0 auto;
  padding: 23rpx 46rpx calc(23rpx + env(safe-area-inset-bottom));
  background: #fff;
  box-shadow: 0 -2rpx 20rpx rgba(0, 0, 0, 0.06);
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
  padding: $spacing-xs 0;
}

.nav-icon {
  font-size: 48rpx;
  font-weight: 600;
  color: #94A3B8;
}

.nav-label {
  font-size: $typography-font-size-xs;
  color: #94A3B8;
}

.active-icon {
  color: $color-semantic-info;
}

.active-label {
  color: $color-semantic-info;
  font-weight: 600;
}
</style>
