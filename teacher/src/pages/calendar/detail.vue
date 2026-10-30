<script setup>
import { computed, ref } from "vue"
import { onLoad, onShow } from "@dcloudio/uni-app"
import { deleteTeacherCalendarEvent, fetchTeacherCalendarEvent } from "@/api/teacher-calendar"
import {
  eventTimeText,
  formatDateCn,
  scopeText,
  splitParagraphs,
  statusDetailText,
  typeColorOf,
  typeLabelOf,
  typeTint
} from "@/utils/calendar"
import PageShell from "@/components/PageShell.vue"
import CalendarEventCard from "@/components/CalendarEventCard.vue"

const event = ref({})
const loading = ref(true)
const loadError = ref("")
const eventId = ref(0)
const showPreview = ref(false)
const deleteVisible = ref(false)
const deleting = ref(false)
let skipNextShow = true

const isGarden = computed(() => event.value?.scope === "kindergarten")
const typeLabel = computed(() => typeLabelOf(event.value, {}) || "活动")
const typeColor = computed(() => typeColorOf(event.value, {}))
const typeBadgeStyle = computed(() => {
  const color = typeColor.value
  return color ? { backgroundColor: typeTint(color), color } : {}
})
const paragraphs = computed(() => splitParagraphs(event.value?.description))
const dateText = computed(() => formatDateCn(event.value?.event_date) || "未设置")
const timeText = computed(() => eventTimeText(event.value) || "未设置")
const locationText = computed(() => event.value?.location || "未设置")

// 预览必须与家长端共用同一张卡（CalendarEventCard），否则「预览」会漂成另一种设计
const previewEvent = computed(() => ({
  title: event.value?.title || "",
  event_type_label: typeLabel.value,
  color: typeColor.value,
  event_date: event.value?.event_date || "",
  is_all_day: !!event.value?.is_all_day,
  start_time: event.value?.start_time || "",
  end_time: event.value?.end_time || "",
  location: event.value?.location || "",
  description: event.value?.description || ""
}))

async function load({ retry = false } = {}) {
  if (!eventId.value) {
    loading.value = false
    loadError.value = "活动加载失败"
    return
  }
  loading.value = true
  loadError.value = ""
  try {
    const res = await fetchTeacherCalendarEvent(eventId.value, { silent: true })
    event.value = res.data?.event || {}
    if (!event.value.id) loadError.value = "活动加载失败"
    if (retry && !loadError.value) uni.showToast({ title: "已重新加载", icon: "none" })
  } catch {
    event.value = {}
    loadError.value = "活动加载失败"
  } finally {
    loading.value = false
  }
}

function noop() {}
function retryLoad() {
  load({ retry: true })
}
function goEdit() {
  if (isGarden.value) return
  uni.navigateTo({ url: `/pages/calendar/create?id=${event.value.id}` })
}
function confirmDelete() {
  if (isGarden.value) return
  deleteVisible.value = true
}
function closeDelete() {
  if (deleting.value) return
  deleteVisible.value = false
}
async function doDelete() {
  if (deleting.value) return
  deleting.value = true
  try {
    await deleteTeacherCalendarEvent(event.value.id, { silent: true })
    deleteVisible.value = false
    uni.showToast({ title: "已删除", icon: "success" })
    setTimeout(() => uni.navigateBack(), 900)
  } catch (err) {
    uni.showToast({ title: err?.message || "删除失败", icon: "none" })
  } finally {
    deleting.value = false
  }
}

onLoad((options) => {
  eventId.value = Number(options?.id || 0)
  load()
})
// 从编辑页（create?id=）返回时刷新，避免详情停留在旧数据
onShow(() => {
  if (skipNextShow) {
    skipNextShow = false
    return
  }
  load()
})
</script>

<template>
  <PageShell max-width="430px">
    <view class="detail-page">
      <template v-if="loading">
        <text class="loading-text">加载中…</text>
        <view class="skeleton" aria-hidden="true">
          <view class="sk-card">
            <view class="sk-badges">
              <view class="sk-chip" />
              <view class="sk-chip is-status" />
            </view>
            <view class="sk-line is-title" />
            <view class="sk-line is-title is-short" />
            <view class="sk-desc">
              <view class="sk-line" />
              <view class="sk-line is-wide" />
              <view class="sk-line is-mid" />
            </view>
          </view>
          <view class="sk-card is-info">
            <view v-for="i in 4" :key="i" class="sk-row">
              <view class="sk-icon" />
              <view class="sk-lines">
                <view class="sk-line is-label" />
                <view class="sk-line is-value" />
              </view>
            </view>
          </view>
          <view class="sk-cta" />
        </view>
      </template>

      <view v-else-if="loadError" class="error-block">
        <view class="error-icon"><text class="icon">&#xf2c2;</text></view>
        <text class="error-title">活动加载失败</text>
        <text class="error-desc">网络异常，未获取到活动详情。</text>
        <view class="error-btn" @click="retryLoad"><text>重新加载</text></view>
      </view>

      <template v-else>
        <view class="header-card">
          <view class="badge-row">
            <view
              class="type-badge"
              :class="{ 'is-unknown': !typeColor }"
              :style="typeBadgeStyle"
            >
              <text class="type-mono">{{ typeLabel[0] }}</text>
              <text class="type-label">{{ typeLabel }}</text>
            </view>
            <view
              class="status-badge"
              :class="event.status === 'draft' ? 'is-draft' : 'is-published'"
            >
              <view class="status-dot" />
              <text>{{ statusDetailText(event.status) }}</text>
            </view>
          </view>

          <text class="event-title">{{ event.title || "活动标题" }}</text>

          <view v-if="event.status === 'draft'" class="draft-note">
            <text class="icon note-icon">&#xecb7;</text>
            <text class="note-text">草稿 · 家长端暂时看不到这个活动，发布后才可见。</text>
          </view>

          <view v-if="paragraphs.length" class="event-desc">
            <text v-for="(paragraph, i) in paragraphs" :key="i" class="desc-paragraph">{{ paragraph }}</text>
          </view>
          <view v-else class="desc-empty">
            <view class="desc-icon"><text class="icon">&#xed0f;</text></view>
            <text class="desc-text">暂无活动说明</text>
            <view v-if="!isGarden" class="desc-btn" @click="goEdit">
              <text>补充说明</text>
              <text class="icon desc-chevron">&#xea6e;</text>
            </view>
          </view>

          <view v-if="isGarden" class="scope-note">
            <text class="icon note-icon">&#xeece;</text>
            <text class="note-text">全园事件（由园长 / 保健医发布），老师端只读。</text>
          </view>
        </view>

        <view class="info-card">
          <view class="info-row">
            <view class="info-icon"><text class="icon">&#xeb27;</text></view>
            <view class="info-body">
              <text class="info-label">活动日期</text>
              <text class="info-value">{{ dateText }}</text>
            </view>
          </view>
          <view class="info-row">
            <view class="info-icon"><text class="icon">&#xf20f;</text></view>
            <view class="info-body">
              <text class="info-label">活动时间</text>
              <text class="info-value is-time">{{ timeText }}</text>
            </view>
          </view>
          <view class="info-row">
            <view class="info-icon"><text class="icon">&#xef14;</text></view>
            <view class="info-body">
              <text class="info-label">活动地点</text>
              <text class="info-value">{{ locationText }}</text>
            </view>
          </view>
          <view class="info-row">
            <view class="info-icon"><text class="icon">&#xf1ee;</text></view>
            <view class="info-body">
              <text class="info-label">参与范围</text>
              <text class="info-value">{{ scopeText(event.scope) }}</text>
            </view>
          </view>
        </view>

        <view class="preview-btn" @click="showPreview = true">
          <text class="icon preview-icon">&#xecb5;</text>
          <text class="preview-text">预览家长视角</text>
        </view>

        <view v-if="!isGarden" class="bottom-actions">
          <view class="btn-edit" @click="goEdit"><text>编辑活动</text></view>
          <view class="btn-delete" @click="confirmDelete"><text>删除</text></view>
        </view>
      </template>
    </view>

    <!-- 家长视角预览：与家长端共用 CalendarEventCard -->
    <view v-if="showPreview" class="sheet-layer">
      <view class="sheet-scrim" @click="showPreview = false" @touchmove.stop.prevent="noop" />
      <view class="sheet">
        <view class="sheet-head">
          <text class="sheet-title">家长看到的样子</text>
          <view class="sheet-close" @click="showPreview = false"><text>关闭</text></view>
        </view>
        <view class="sheet-body">
          <view v-if="event.status === 'draft'" class="sheet-draft">
            <text>草稿 · 家长暂时看不到</text>
          </view>
          <CalendarEventCard :event="previewEvent" surface="parent" />
        </view>
      </view>
    </view>
    <!-- 删除确认：微信风格居中对话框 -->
    <view v-if="deleteVisible" class="dialog-layer">
      <view class="dialog-mask" @click="closeDelete" @touchmove.stop.prevent="noop" />
      <view class="dialog">
        <text class="dialog-title">删除活动</text>
        <text class="dialog-desc">确定要删除该活动吗？删除后家长端将不再可见。</text>
        <view class="dialog-actions">
          <view class="dialog-btn is-cancel" @click="closeDelete"><text>取消</text></view>
          <view class="dialog-btn is-danger" @click="doDelete">
            <text>{{ deleting ? "删除中…" : "确定" }}</text>
          </view>
        </view>
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

.detail-page {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  padding-bottom: $spacing-xl;
}

/* 类型色来自接口（字典解析后的 color），仅做透明度派生，不引入本地类型色表 */
.type-badge {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 10rpx;
  height: 56rpx;
  padding: 0 20rpx 0 10rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1.3;
  letter-spacing: 0.02em;
}
.type-badge.is-unknown {
  background: $color-bg-page;
  color: $color-text-muted;
}
.type-mono {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36rpx;
  height: 36rpx;
  border-radius: $border-radius-md;
  font-size: $typography-font-size-xs;
  font-weight: 600;
  line-height: 1;
  background: rgba($color-text-on-brand, 0.55);
}
.type-label {
  padding-right: 2rpx;
}

/* ---------- 头部信息卡 ---------- */
.header-card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}
.badge-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10rpx 24rpx;
}
.status-badge {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 10rpx;
  min-height: 56rpx;
  margin-left: auto;
  padding: 6rpx $component-badge-padding-x;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1.3;
  letter-spacing: 0.02em;
}
.status-badge.is-published {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}
.status-badge.is-draft {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}
.status-dot {
  flex: none;
  width: 12rpx;
  height: 12rpx;
  border-radius: $border-radius-full;
  background: currentColor;
}
.event-title {
  display: block;
  margin-top: 20rpx;
  font-size: $typography-font-size-3xl;
  font-weight: 600;
  line-height: 1.3;
  color: $color-text-primary;
  overflow-wrap: anywhere;
}

.draft-note,
.scope-note {
  display: flex;
  align-items: flex-start;
  gap: 14rpx;
  margin-top: 20rpx;
  padding: 18rpx 24rpx;
  border-radius: $border-radius-lg;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  line-height: 1.65;
}
.draft-note {
  background: $color-semantic-warning-bg;
  color: $color-text-body;
}
.scope-note {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}
.note-icon {
  flex: none;
  margin-top: 2rpx;
  font-size: $typography-font-size-base;
}
.draft-note .note-icon {
  color: $color-semantic-warning;
}
.note-text {
  min-width: 0;
  overflow-wrap: anywhere;
}

.event-desc {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  margin-top: 24rpx;
  font-size: $typography-font-size-base;
  line-height: 1.75;
  color: $color-text-body;
}
.desc-paragraph {
  overflow-wrap: anywhere;
}

.desc-empty {
  display: flex;
  align-items: center;
  gap: 20rpx;
  margin-top: 24rpx;
  padding: 14rpx 20rpx 14rpx 24rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}
.desc-icon {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  color: $color-text-muted;
  font-size: $typography-font-size-xl;
}
.desc-text {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  line-height: 1.5;
  color: $color-text-body;
}
.desc-btn {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 4rpx;
  min-width: 88rpx;
  min-height: 88rpx;
  padding: 0 8rpx;
  font-size: $typography-font-size-sm;
  font-weight: 600;
  color: $color-text-body;
}
.desc-chevron {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

/* ---------- 信息卡 ---------- */
.info-card {
  padding: 10rpx $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}
.info-row {
  display: flex;
  align-items: flex-start;
  gap: $spacing-lg;
  min-width: 0;
  padding: 22rpx 0;
}
.info-row + .info-row {
  border-top: 2rpx solid $color-border-subtle;
}
.info-icon {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-xl;
}
.info-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
  padding-top: 2rpx;
}
.info-label {
  font-size: $typography-font-size-xs;
  font-weight: 500;
  line-height: 1.3;
  letter-spacing: 0.02em;
  color: $color-text-muted;
}
.info-value {
  font-size: $typography-font-size-base;
  font-weight: 500;
  line-height: 1.55;
  color: $color-text-primary;
  overflow-wrap: anywhere;
}
.info-value.is-time {
  font-variant-numeric: tabular-nums;
}

/* ---------- 预览家长视角 ---------- */
.preview-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  width: 100%;
  min-height: 88rpx;
  padding: 0 $spacing-xl;
  border: 2rpx solid $color-brand-primary-light;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
  font-weight: 500;
  letter-spacing: 0.02em;
  box-shadow: $shadow-sm;
}
.preview-btn:active {
  transform: scale(0.985);
  box-shadow: none;
}
.preview-icon {
  font-size: $typography-font-size-lg;
}

/* ---------- 底部操作 ---------- */
.bottom-actions {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 216rpx;
  gap: 20rpx;
  margin-top: 8rpx;
}
.btn-edit {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 96rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-lg;
  font-weight: 600;
  letter-spacing: 0.02em;
  box-shadow: $shadow-md;
}
.btn-delete {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 96rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger-strong;
  font-size: $typography-font-size-lg;
  font-weight: 500;
  letter-spacing: 0.02em;
}
.btn-edit:active,
.btn-delete:active {
  transform: scale(0.97);
}

/* ---------- 骨架屏 ---------- */
.loading-text {
  position: absolute;
  width: 1rpx;
  height: 1rpx;
  overflow: hidden;
  opacity: 0;
}
.skeleton {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}
.sk-card {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.sk-badges {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  margin-bottom: 4rpx;
}
.sk-chip {
  width: 128rpx;
  height: 52rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  animation: sk-pulse 1.6s ease-in-out infinite;
}
.sk-chip.is-status {
  width: 152rpx;
}
.sk-line {
  display: block;
  height: 20rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  animation: sk-pulse 1.6s ease-in-out infinite;
}
.sk-line.is-title {
  width: 84%;
  height: 36rpx;
  border-radius: $border-radius-md;
}
.sk-line.is-title.is-short {
  width: 52%;
}
.sk-line.is-wide {
  width: 94%;
}
.sk-line.is-mid {
  width: 66%;
}
.sk-desc {
  display: flex;
  flex-direction: column;
  gap: 14rpx;
  margin-top: 4rpx;
}
.sk-card.is-info {
  padding: 10rpx $spacing-xl;
}
.sk-row {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-width: 0;
  padding: 22rpx 0;
}
.sk-row + .sk-row {
  border-top: 2rpx solid $color-border-subtle;
}
.sk-icon {
  flex: none;
  width: 72rpx;
  height: 72rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}
.sk-lines {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 14rpx;
  min-width: 0;
}
.sk-line.is-label {
  width: 34%;
  height: 16rpx;
}
.sk-line.is-value {
  width: 62%;
  height: 24rpx;
}
.sk-cta {
  display: block;
  width: 100%;
  height: 88rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  animation: sk-pulse 1.6s ease-in-out infinite;
}
@keyframes sk-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.55;
  }
}

/* ---------- 失败态 ---------- */
.error-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14rpx;
  padding: 60rpx 40rpx 52rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}
.error-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 104rpx;
  height: 104rpx;
  margin-bottom: 4rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-3xl;
}
.error-title {
  font-size: $typography-font-size-md;
  font-weight: 600;
  line-height: 1.3;
  color: $color-text-primary;
}
.error-desc {
  font-size: $typography-font-size-sm;
  line-height: 1.7;
  color: $color-text-muted;
}
.error-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: 12rpx;
  padding: 0 48rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-md;
  font-weight: 600;
  letter-spacing: 0.02em;
  box-shadow: $shadow-sm;
}

/* ---------- 家长视角弹层 ---------- */
/* 遮罩色是刻意的一次性值：design token 没有 overlay 语义，全端仅此几处使用。 */
.sheet-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 300;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.sheet-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba(15, 23, 42, 0.45);
}
.sheet {
  position: relative;
  width: 100%;
  max-width: 430px;
  padding: 28rpx 32rpx calc(40rpx + env(safe-area-inset-bottom));
  border-radius: $border-radius-2xl $border-radius-2xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}
.sheet-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  min-height: 88rpx;
}
.sheet-title {
  font-size: $typography-font-size-xl;
  font-weight: 600;
  line-height: 1.4;
  color: $color-text-primary;
}
.sheet-close {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  min-width: 88rpx;
  min-height: 88rpx;
  font-size: $typography-font-size-base;
  font-weight: 500;
  color: $color-text-body;
}
.sheet-body {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  margin-top: 12rpx;
}
.sheet-draft {
  padding: 18rpx 24rpx;
  border-radius: $border-radius-lg;
  background: $color-semantic-warning-bg;
  font-size: $typography-font-size-sm;
  font-weight: 500;
  line-height: 1.6;
  color: $color-text-body;
}

/* ---------- 删除确认（居中，微信风格） ---------- */
.dialog-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 320;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48rpx;
}
.dialog-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: rgba(15, 23, 42, 0.45);
}
.dialog {
  position: relative;
  width: 100%;
  max-width: 600rpx;
  padding: 44rpx 40rpx 36rpx;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
  text-align: center;
}
.dialog-title {
  display: block;
  font-size: $typography-font-size-xl;
  font-weight: 600;
  line-height: 1.4;
  color: $color-text-primary;
}
.dialog-desc {
  display: block;
  margin-top: 16rpx;
  font-size: $typography-font-size-base;
  line-height: 1.7;
  color: $color-text-body;
}
.dialog-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20rpx;
  margin-top: 40rpx;
}
.dialog-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-md;
  font-weight: 500;
  letter-spacing: 0.02em;
}
.dialog-btn.is-cancel {
  background: $color-bg-page;
  color: $color-text-body;
}
.dialog-btn.is-danger {
  background: $color-semantic-danger;
  color: $color-text-on-brand;
  font-weight: 600;
}
.dialog-btn:active {
  transform: scale(0.97);
}
</style>
