<script setup>
import { ref, computed } from "vue"
import { onLoad, onReachBottom, onPullDownRefresh } from "@dcloudio/uni-app"
import BottomNav from "@/components/BottomNav.vue"
import { fetchTeacherMessages } from "@/api/messages"
import { fetchTeacherHomeOverview } from "@/api/teacher-home"

const activeTab = ref("全部")
const tabs = ["全部", "通知", "审批", "系统"]

const loading = ref(false)
const allMessages = ref([])
const page = ref(1)
const total = ref(0)
const overview = ref({})
const hasMore = computed(() => allMessages.value.length < total.value)

// ---- 字段映射 ----

const TYPE_CONFIG = {
  notice:  { sender: "班级通知", iconBg: "#DBEAFE", iconColor: "#3B82F6", tagType: "notice" },
  leave:   { sender: "审批提醒", iconBg: "#FEF9C3", iconColor: "#CA8A04", tagType: "approval" },
  system:  { sender: "系统消息", iconBg: "#F1F5F9", iconColor: "#64748B", tagType: "system" },
}

function formatRelativeTime(iso) {
  if (!iso) return ""
  const now = Date.now()
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return iso
  const diff = now - then
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return "刚刚"
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}天前`
  const d = new Date(then)
  return `${d.getMonth() + 1}月${d.getDate()}日`
}

function mapMessage(raw) {
  const cfg = TYPE_CONFIG[raw.type] || TYPE_CONFIG.system
  return {
    id:         raw.reference_id || 0,
    _type:      raw.type,
    type:       raw.type,
    category:   (raw.extra && raw.extra.category_label) || (raw.extra && raw.extra.status_label) || "",
    sender:     cfg.sender,
    time:       formatRelativeTime(raw.created_at),
    preview:    (raw.content || "").slice(0, 60),
    // 系统消息未读在后端是 is_read 字段，通知类没有 is_read
    isRead:     true,
    unread:     0,
    icon:       raw.type === "notice" ? "notify" : raw.type === "leave" ? "approve" : "system",
    iconBg:     cfg.iconBg,
    iconColor:  cfg.iconColor,
    tagType:    cfg.tagType,
    // 保留原始数据用于后续扩展
    raw,
  }
}

// ---- 列表过滤 ----

const tabTypeMap = { "通知": "notice", "审批": "leave", "系统": "system" }

const messages = computed(() => {
  if (activeTab.value === "全部") return allMessages.value
  const t = tabTypeMap[activeTab.value]
  return allMessages.value.filter(m => m._type === t)
})

const unreadTotal = computed(() => Number(overview.value.unread_notices || 0))

// ---- Tab 切换 ----

function switchTab(tab) {
  activeTab.value = tab
}

// ---- 加载 ----

async function loadMessages(reset = false) {
  loading.value = true
  if (reset) {
    page.value = 1
  }
  try {
    const res = await fetchTeacherMessages({ page: page.value, page_size: 20 })
    const data = res.data || res
    const list = (data.list || []).map(mapMessage)
    if (reset) {
      allMessages.value = list
    } else {
      allMessages.value.push(...list)
    }
    total.value = data.total || 0
  } catch (err) {
    if (reset) allMessages.value = []
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

async function loadMore() {
  if (loading.value || !hasMore.value) return
  page.value++
  await loadMessages(false)
}

async function loadUnread() {
  try {
    const res = await fetchTeacherHomeOverview()
    overview.value = res.data || {}
  } catch {
    // 角标是增强信息，失败不影响列表
  }
}

// ---- 操作 ----

function markAllRead() {
  uni.showToast({ title: "已全部标为已读", icon: "none" })
}

function goDetail(item) {
  const t = item.raw && item.raw.type
  if (t === "leave") {
    uni.navigateTo({ url: "/pages/leave/index" })
    return
  }
  uni.showToast({ title: `查看: ${item.sender}`, icon: "none" })
}

// ---- 生命周期 ----

onLoad(() => {
  loadMessages(true)
  loadUnread()
})
onPullDownRefresh(() => {
  loadMessages(true)
  loadUnread()
})
onReachBottom(loadMore)
</script>

<template>
  <view class="page">
    <!-- 消息分类标签 -->
    <view class="tabs-bar">
      <scroll-view scroll-x class="tabs-scroll" :show-scrollbar="false">
        <view class="tabs-row">
          <view
            v-for="tab in tabs"
            :key="tab"
            class="tab-pill"
            :class="{ active: activeTab === tab }"
            @click="switchTab(tab)"
          >
            <text>{{ tab }}</text>
          </view>
        </view>
      </scroll-view>
    </view>

    <!-- 消息列表 -->
    <scroll-view scroll-y class="msg-list" :refresher-enabled="true" @refresherrefresh="loadMessages">
      <view v-if="loading" class="state-wrap">
        <text class="state-text">加载中...</text>
      </view>

      <view v-else-if="messages.length === 0" class="state-wrap">
        <text class="state-text">暂无消息</text>
      </view>

      <view v-else>
        <view
          v-for="msg in messages"
          :key="msg.id"
          class="msg-card"
          @click="goDetail(msg)"
        >
          <!-- 左侧头像/图标区 -->
          <view class="avatar-zone">
            <!-- 未读红点 -->
            <view v-if="!msg.isRead && msg.unread > 0" class="dot-unread" />

            <!-- 图标 -->
            <view class="icon-box" :style="{ backgroundColor: msg.iconBg }">
              <text class="icon-text" :style="{ color: msg.iconColor }">&#xef94;</text>
            </view>
          </view>

          <!-- 右侧消息内容 -->
          <view class="msg-content">
            <!-- 头部: 名称 + 时间 -->
            <view class="msg-head">
              <text class="msg-sender" :class="{ bold: !msg.isRead }">{{ msg.sender }}</text>
              <text class="msg-time">{{ msg.time }}</text>
            </view>

            <!-- 消息预览 -->
            <text class="msg-preview">{{ msg.preview }}</text>

            <!-- 底部: 标签 + 未读数 -->
            <view class="msg-foot">
              <view class="msg-tag" :class="`tag-${msg.tagType || msg.type}`">
                <text>{{ msg.category }}</text>
              </view>
              <view v-if="!msg.isRead && msg.unread > 0" class="badge-count">
                <text>{{ msg.unread }}</text>
              </view>
            </view>
          </view>
        </view>
      </view>

      <view class="bottom-safe" />
    </scroll-view>

    <!-- 底部导航栏 -->
    <BottomNav :active="2" :unread-count="unreadTotal" />
  </view>
</template>

<style scoped lang="scss">

.page {
  min-height: 100vh;
  background: $color-bg-page;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow-x: hidden;
}

// ===== 分类标签 =====
.tabs-bar {
  background: #fff;
  padding: 0 28rpx;
  box-sizing: border-box;
  width: 100%;
}

.tabs-scroll {
  white-space: nowrap;
}

.tabs-row {
  display: flex;
  gap: $spacing-xl;
  padding: $spacing-lg 0;
}

.tab-pill {
  flex-shrink: 0;
  padding: 28rpx 32rpx;

  text {
    font-size: $typography-font-size-md;
    color: $color-text-muted;
  }
}

.tab-pill.active {
  text {
    color: $color-semantic-info;
    font-weight: 600;
  }
}

// ===== 消息列表 =====
.msg-list {
  flex: 1;
  padding: 0 28rpx;
  height: 0;
  box-sizing: border-box;
  width: 100%;
}

.state-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 200rpx 0;
}

.state-text {
  font-size: $typography-font-size-md;
  color: $color-text-hint;
}

.msg-card {
  background: #fff;
  border-radius: $border-radius-xl;
  padding: 28rpx 32rpx;
  display: flex;
  gap: $spacing-lg;
  box-shadow: 0 2rpx 8rpx rgba(120, 53, 15, 0.04);
  margin-bottom: 20rpx;
  box-sizing: border-box;
  width: 100%;
}

// ===== 头像区 =====
.avatar-zone {
  position: relative;
  width: 96rpx;
  height: 96rpx;
  flex-shrink: 0;
}

.dot-unread {
  position: absolute;
  top: -4rpx;
  right: -4rpx;
  width: 22rpx;
  height: 20rpx;
  background: $color-semantic-danger;
  border-radius: 9999rpx;
  border: 4rpx solid #fff;
  z-index: 2;
}

.icon-box {
  width: 96rpx;
  height: 96rpx;
  border-radius: 28rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.icon-text {
  // Remix Icon 的 PUA 码位，缺字体时会渲染成方块
  font-family: remixicon;
  font-size: 48rpx;
}

// ===== 消息内容区 =====
.msg-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
  overflow: hidden;
}

.msg-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.msg-sender {
  font-size: $typography-font-size-lg;
  color: $color-text-primary;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.msg-sender.bold {
  font-weight: 600;
}

.msg-time {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
  flex-shrink: 0;
  margin-left: 16rpx;
}

.msg-preview {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.msg-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.msg-tag {
  padding: 4rpx 16rpx;
  border-radius: $border-radius-sm;
  text {
    font-size: $typography-font-size-xs;
    font-weight: 500;
  }
}

// Tag colors by type
.tag-notice {
  background: $color-semantic-info-bg;
  text { color: $color-semantic-info; }
}

.tag-approval {
  background: $color-semantic-warning-bg;
  text { color: $color-semantic-warning; }
}

.tag-system {
  background: $color-bg-card;
  text { color: $color-text-muted; }
}

.badge-count {
  width: 40rpx;
  height: 40rpx;
  border-radius: 9999rpx;
  background: $color-semantic-danger;
  display: flex;
  align-items: center;
  justify-content: center;
  text {
    font-size: $typography-font-size-xs;
    font-weight: 600;
    color: #fff;
  }
}

.bottom-safe {
  height: calc(80rpx + env(safe-area-inset-bottom));
}

</style>
