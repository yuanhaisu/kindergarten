<script setup>
import { computed, ref } from "vue"
import { onLoad, onShow } from "@dcloudio/uni-app"
import { fetchTeacherCalendarEvent, deleteTeacherCalendarEvent } from "@/api/teacher-calendar"
import CalendarEventCard from "@/components/CalendarEventCard.vue"

const event = ref({})
const loading = ref(true)
const eventId = ref(0)
const showPreview = ref(false)

let skipNextShow = true

// 类型标签/颜色取自服务端按字典解析下发的 event_type_label / color（数据驱动色）
function typeStyle(e) {
  const color = e?.color || ""
  return color ? { backgroundColor: color + "1F", color } : {}
}
function typeLabel(e) { return e?.event_type_label || e?.event_type || "活动" }
function isGarden() { return event.value?.scope === "kindergarten" }

// 预览家长视角：映射成 CalendarEventCard 的归一化形状。该组件与家长端日历页共用，
// 所以这里渲染出来的就是家长实际看到的那张卡（surface="parent" 套家长端表面色）。
const previewEvent = computed(() => ({
  title: event.value.title || "",
  event_type_label: typeLabel(event.value),
  color: event.value.color || "",
  event_date: event.value.event_date || "",
  is_all_day: !!event.value.is_all_day,
  start_time: event.value.start_time || "",
  end_time: event.value.end_time || "",
  location: event.value.location || "",
  description: event.value.description || "",
}))

function goEdit() { uni.navigateTo({ url: `/pages/calendar/create?id=${event.value.id}` }) }

function confirmDelete() {
  uni.showModal({
    title: "删除活动",
    content: "确定要删除该活动吗？删除后家长端将不再可见。",
    success: async (r) => {
      if (!r.confirm) return
      try {
        await deleteTeacherCalendarEvent(event.value.id)
        uni.showToast({ title: "已删除", icon: "success" })
        setTimeout(() => uni.navigateBack(), 1000)
      } catch (err) {
        uni.showToast({ title: err?.message || "删除失败", icon: "none" })
      }
    }
  })
}

async function load() {
  if (!eventId.value) { loading.value = false; return }
  try {
    const res = await fetchTeacherCalendarEvent(eventId.value)
    event.value = res.data?.event || {}
  } catch {
    uni.showToast({ title: "加载失败", icon: "none" })
  } finally { loading.value = false }
}

onLoad((options) => {
  eventId.value = Number(options?.id || 0)
  load()
})
// 从编辑页（create?id=）返回时刷新
onShow(() => {
  if (skipNextShow) { skipNextShow = false; return }
  load()
})
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state-wrap"><text class="state-text">加载中...</text></view>
    <template v-else>
      <view class="info-card">
        <view class="tag-row">
          <view class="type-tag" :class="{ 'is-unknown': !event.color }" :style="typeStyle(event)">
            <text class="type-icon">{{ typeLabel(event)[0] }}</text>
            <text class="type-text">{{ typeLabel(event) }}</text>
          </view>
          <view class="status-tag" :class="event.status === 'draft' ? 'is-draft' : 'is-published'">
            <view class="status-dot" />
            <text>{{ event.status === "draft" ? "草稿（家长不可见）" : "已发布" }}</text>
          </view>
        </view>
        <text class="event-title">{{ event.title || "活动标题" }}</text>
        <text class="event-desc">{{ event.description || "暂无描述" }}</text>

        <view v-if="isGarden()" class="garden-note">
          <text class="garden-note-text">全园事件（由园长 / 保健医发布），老师端只读。</text>
        </view>

        <view class="info-list">
          <view class="info-row">
            <view class="info-icon-box is-blue"><text class="info-icon">日</text></view>
            <view class="info-content"><text class="info-label">活动日期</text><text class="info-value">{{ event.event_date || "未设置" }}</text></view>
          </view>
          <view class="info-row">
            <view class="info-icon-box is-orange"><text class="info-icon">时</text></view>
            <view class="info-content">
              <text class="info-label">活动时间</text>
              <text class="info-value">{{ event.is_all_day ? "全天" : ((event.start_time || "未设置") + (event.end_time ? " - " + event.end_time : "")) }}</text>
            </view>
          </view>
          <view class="info-row">
            <view class="info-icon-box is-green"><text class="info-icon">点</text></view>
            <view class="info-content"><text class="info-label">活动地点</text><text class="info-value">{{ event.location || "未设置" }}</text></view>
          </view>
        </view>
      </view>

      <!-- 预览家长视角：与家长端日历页共用 CalendarEventCard -->
      <view class="preview-btn" @click="showPreview = true">
        <text class="preview-text">预览家长视角</text>
      </view>

      <view v-if="!isGarden()" class="bottom-actions">
        <view class="edit-btn" @click="goEdit"><text class="edit-text">编辑活动</text></view>
        <view class="delete-btn" @click="confirmDelete"><text class="delete-text">删除</text></view>
      </view>
      <view class="bottom-safe" />

      <!-- 家长视角预览弹层 -->
      <view v-if="showPreview" class="sheet-mask" @click="showPreview = false">
        <view class="sheet" @click.stop>
          <view class="sheet-head">
            <text class="sheet-title">家长看到的样子</text>
            <text class="sheet-close" @click="showPreview = false">关闭</text>
          </view>
          <view v-if="event.status === 'draft'" class="sheet-note">
            <text class="sheet-note-text">草稿 · 家长暂时看不到</text>
          </view>
          <CalendarEventCard :event="previewEvent" surface="parent" />
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: $spacing-lg $component-page-padding-x;
  background: $color-bg-page;
}

.state-wrap { display: flex; align-items: center; justify-content: center; min-height: 60vh; }
.state-text { font-size: $typography-font-size-md; color: $color-text-hint; }

.info-card { padding: $spacing-xl; border-radius: $component-card-radius; background: $color-bg-card; box-shadow: $shadow-sm; margin-bottom: $spacing-lg; }
.tag-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: $spacing-lg; }

.type-tag { display: flex; align-items: center; gap: $spacing-sm; padding: $spacing-xs $spacing-lg; border-radius: $border-radius-md; }
.type-tag.is-unknown { background: $color-semantic-info-bg; color: $color-semantic-info; }
.type-icon { font-size: $typography-font-size-base; font-weight: 700; }
.type-text { font-size: $typography-font-size-sm; font-weight: 500; }

.status-tag { display: flex; align-items: center; gap: $spacing-xs; padding: $spacing-xs $spacing-lg; border-radius: $border-radius-md; }
.status-dot { width: 16rpx; height: 16rpx; border-radius: $border-radius-full; }
.status-tag.is-published { background: $color-semantic-success-bg; }
.status-tag.is-published .status-dot { background: $color-semantic-success; }
.status-tag.is-published text { color: $color-semantic-success; }
.status-tag.is-draft { background: $color-semantic-warning-bg; }
.status-tag.is-draft .status-dot { background: $color-semantic-warning; }
.status-tag.is-draft text { color: $color-semantic-warning; }
.status-tag text { font-size: $typography-font-size-sm; font-weight: 500; }

.event-title { display: block; font-size: $typography-font-size-3xl; font-weight: 800; color: $color-text-primary; margin-bottom: $spacing-lg; }
.event-desc { display: block; font-size: $typography-font-size-md; color: $color-text-muted; line-height: 1.6; margin-bottom: $spacing-xl; }

.garden-note { margin-bottom: $spacing-lg; padding: $spacing-md $spacing-lg; border-radius: $border-radius-md; background: $color-semantic-info-bg; }
.garden-note-text { font-size: $typography-font-size-sm; color: $color-semantic-info; }

.info-list { display: flex; flex-direction: column; gap: $spacing-lg; padding-top: $spacing-xl; border-top: 2rpx solid $color-bg-page; }
.info-row { display: flex; align-items: flex-start; gap: $spacing-lg; }
.info-icon-box { display: flex; flex-shrink: 0; align-items: center; justify-content: center; width: 72rpx; height: 72rpx; border-radius: $border-radius-xl; }
.info-icon-box.is-blue   { background: $color-semantic-info-bg; }
.info-icon-box.is-orange { background: $color-brand-primary-bg; }
.info-icon-box.is-green  { background: $color-semantic-success-bg; }

.info-icon { font-size: $typography-font-size-xl; font-weight: 700; }
.info-icon-box.is-blue   .info-icon { color: $color-semantic-info; }
.info-icon-box.is-orange .info-icon { color: $color-brand-primary-dark; }
.info-icon-box.is-green  .info-icon { color: $color-semantic-success; }

.info-content { flex: 1; display: flex; flex-direction: column; gap: 4rpx; }
.info-label { font-size: $typography-font-size-sm; color: $color-text-hint; }
.info-value { font-size: $typography-font-size-md; font-weight: 500; color: $color-text-primary; }

// ===== 预览家长视角 =====
.preview-btn {
  display: flex; align-items: center; justify-content: center;
  height: 88rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}
.preview-text { font-size: $typography-font-size-base; font-weight: 600; color: $color-brand-primary-dark; }

// 底部弹层（小程序惯例）。遮罩色是刻意的一次性值：design token 里没有 overlay 语义，
// 且全端只此一处。
.sheet-mask {
  position: fixed; top: 0; right: 0; bottom: 0; left: 0;
  z-index: 200;
  display: flex; align-items: flex-end;
  background: rgba(30, 41, 59, 0.45);
}
.sheet {
  width: 100%;
  padding: $spacing-xl $component-page-padding-x;
  padding-bottom: calc(#{$spacing-xl} + env(safe-area-inset-bottom));
  border-top-left-radius: $border-radius-2xl;
  border-top-right-radius: $border-radius-2xl;
  background: $color-bg-page;
  box-shadow: $shadow-lg;
}
.sheet-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: $spacing-lg; }
.sheet-title { font-size: $typography-font-size-xl; font-weight: 700; color: $color-text-primary; }
.sheet-close { font-size: $typography-font-size-base; color: $color-text-muted; }
.sheet-note {
  margin-bottom: $spacing-md;
  padding: $spacing-sm $spacing-lg;
  border-radius: $border-radius-md;
  background: $color-semantic-warning-bg;
}
.sheet-note-text { font-size: $typography-font-size-sm; font-weight: 600; color: $color-semantic-warning; }

.bottom-actions { display: flex; gap: $spacing-lg; }
.edit-btn {
  flex: 1.4; display: flex; align-items: center; justify-content: center;
  height: 96rpx; border-radius: $border-radius-full;
  background: $gradient-brand; box-shadow: $shadow-md;
}
.edit-text { font-size: $typography-font-size-lg; font-weight: 700; color: $color-text-on-brand; }
.delete-btn {
  flex: 1; display: flex; align-items: center; justify-content: center;
  height: 96rpx; border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
}
.delete-text { font-size: $typography-font-size-lg; font-weight: 600; color: $color-semantic-danger; }
.bottom-safe { height: calc(60rpx + env(safe-area-inset-bottom)); }
</style>
