<script setup>
import { ref } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchMessages, fetchUnreadCount } from "@/api/parent-messages"

const messages = ref([])
const unread = ref(0)
const loading = ref(false)
const loadError = ref("")

// 家长端 tabBar 只有这四页（pages.json → tabBar.list）。跳到 tabBar 页必须 switchTab，
// navigateTo 会直接失败——这个差异只在真实点击时暴露。
const TAB_PAGES = [
  "/pages/index/index",
  "/pages/growth/index",
  "/pages/pickup/index",
  "/pages/profile/index"
]

// action_path 由服务端写入，只允许小程序内部 `/pages/` 路径。
// 这里再挡一次 `://`：任何外部 URL 都不该被站内消息点开。
function isInternalPath(path) {
  return typeof path === "string" &&
    path.indexOf("/pages/") === 0 &&
    path.indexOf("://") === -1
}

function openMessage(item) {
  const path = item?.action_path
  if (!isInternalPath(path)) return
  const pathname = path.split("?")[0]
  if (TAB_PAGES.indexOf(pathname) !== -1) {
    uni.switchTab({ url: pathname })
    return
  }
  uni.navigateTo({
    url: path,
    fail: () => {
      // 任务被删除/页面未注册时不能静默：用户点了没反应会被当成卡死。
      uni.showToast({ title: "该内容已不存在", icon: "none" })
    }
  })
}

async function loadData() {
  if (messages.value.length === 0) loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const [m, u] = await Promise.all([fetchMessages(), fetchUnreadCount()])
    messages.value = m.data?.list || m.data || []
    unread.value = u.data?.count || 0
  } catch (err) {
    loadError.value = err?.message || "加载失败"
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

// onShow：从任务详情返回后未读数可能变了，onLoad 不会再跑。
onShow(loadData)
onPullDownRefresh(loadData)
</script>

<template>
  <view class="page">
    <view class="header">
      <text class="page-title">消息</text>
      <text v-if="unread" class="badge">{{ unread }}条未读</text>
    </view>
    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="loadError" class="state" @click="loadData">{{ loadError }}，点击重试</view>
    <view v-else-if="messages.length === 0" class="state">暂无消息</view>
    <view v-else class="list">
      <view
        v-for="item in messages"
        :key="item.id"
        class="card"
        :class="{ linkable: isInternalPath(item.action_path) }"
        @click="openMessage(item)"
      >
        <view class="card-head">
          <text class="title">{{ item.title }}</text>
          <text v-if="isInternalPath(item.action_path)" class="go">查看 ›</text>
        </view>
        <text class="content">{{ item.content }}</text>
        <text class="date">{{ item.created_at?.slice(0, 16) }}</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $component-page-padding-y $component-page-padding-x; background: $color-bg-page; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: $spacing-lg; }
.page-title { color: $color-text-primary; font-size: $typography-font-size-2xl; font-weight: $typography-font-weight-bold; }
.badge { padding: $spacing-xs 20rpx; border-radius: $border-radius-full; background: $color-semantic-danger; color: $color-text-on-brand; font-size: $typography-font-size-xs; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }
.list { display: flex; flex-direction: column; gap: 18rpx; }
.card { padding: 26rpx; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: $shadow-sm; }
.card.linkable:active { background: $color-brand-primary-bg; }
.card-head { display: flex; align-items: center; justify-content: space-between; gap: $spacing-md; }
.title { flex: 1; color: $color-text-primary; font-size: $typography-font-size-md; font-weight: $typography-font-weight-bold; }
.go { flex-shrink: 0; color: $color-brand-primary; font-size: $typography-font-size-xs; }
.content { display: block; margin-top: $spacing-xs; color: $color-text-muted; font-size: $typography-font-size-sm; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.date { display: block; margin-top: $spacing-sm; color: $color-text-muted; font-size: 20rpx; }
</style>
