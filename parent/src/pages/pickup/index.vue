<script setup>
/**
 * 接送记录（家长端 tab 页）
 *
 * 今天的记录聚合成一张「接送凭证」：状态、时间轴、二维码入口都在卡内；
 * 过去按日期分组退成流水。数据仍来自 GET /parent/pickup/records 一次拉取，
 * 没有新增接口 —— 三态（未入园 / 在园中 / 已接走）全部由当天记录派生。
 */
import { computed, ref } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { syncTabBar } from "@/utils/tabbar"
import { fetchPickupRecords } from "@/api/parent-pickup"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { relationLabel } from "@/utils/relation"
import { formatClock, formatDateLabel } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形（App.vue 已全局注册字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  clock: "", // time-line
  sprout: "", // seedling-line
  check: "", // check-line
  qr: "", // qr-code-line
  error: "" // error-warning-line
}

const WEEKDAYS = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"]

const records = ref([])
const loading = ref(false)
const retrying = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const student = ref({ name: "", class_name: "" })
// 服务端 overview 的 date 是园所时区的「今天」；设备时区一致时与本地等价，
// 拿不到时退回设备日期，避免整页卡在 loading。
const overviewDate = ref("")

function pad(n) {
  return String(n).padStart(2, "0")
}

function localDateKey(date) {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function parseDateParts(value) {
  const m = String(value || "").match(/^(\d{4})-(\d{2})-(\d{2})/)
  return m ? { y: Number(m[1]), m: Number(m[2]), d: Number(m[3]) } : null
}

function weekdayOf(value) {
  const p = parseDateParts(value)
  if (!p) return ""
  return WEEKDAYS[new Date(p.y, p.m - 1, p.d).getDay()] || ""
}

function timestampOf(value) {
  const t = new Date(value).getTime()
  return Number.isNaN(t) ? 0 : t
}

const todayKey = computed(() => overviewDate.value || localDateKey(new Date()))
const todayDateText = computed(() => formatDateLabel(todayKey.value) || todayKey.value)
const todayWeekday = computed(() => weekdayOf(todayKey.value))
const childLabel = computed(() => [student.value.name, student.value.class_name].filter(Boolean).join(" · "))

// 记录接口按 recorded_at DESC 返回；时间轴/历史行都要正序展示（入园在前）。
const todayRecords = computed(() =>
  records.value
    .filter(r => r.date === todayKey.value)
    .slice()
    .sort((a, b) => timestampOf(a.recorded_at) - timestampOf(b.recorded_at))
)

const lastToday = computed(() => todayRecords.value[todayRecords.value.length - 1] || null)

// 三态：当天没有记录 = 未入园；最后一条是接走 = 已接走；否则是在园中。
const dayState = computed(() => {
  if (!lastToday.value) return "pre"
  return Number(lastToday.value.type) === 2 ? "picked" : "in"
})

const statusMain = computed(() => ({ pre: "未入园", in: "在园中", picked: "已接走" }[dayState.value]))

const timelineRecords = computed(() => {
  if (dayState.value === "pre") return []
  // 「在园中」不带接走节点，避免尚未发生的动作出现在轨迹里。
  if (dayState.value === "in") return todayRecords.value.filter(r => Number(r.type) !== 2)
  return todayRecords.value
})

const historyGroups = computed(() => {
  const groups = []
  const seen = new Map()
  for (const r of records.value) {
    if (r.date === todayKey.value) continue
    const key = String(r.date || "") || "unknown"
    if (!seen.has(key)) {
      const group = {
        key,
        label: formatDateLabel(key) || "日期未知",
        weekday: weekdayOf(key),
        rows: []
      }
      seen.set(key, group)
      groups.push(group)
    }
    seen.get(key).rows.push(r)
  }
  return groups.map(group => ({
    ...group,
    rows: group.rows
      .slice()
      .sort((a, b) => timestampOf(a.recorded_at) - timestampOf(b.recorded_at))
      .map(r => ({
        id: r.id,
        typeLabel: r.type_label || (Number(r.type) === 2 ? "接走" : "入园"),
        isOut: Number(r.type) === 2,
        name: r.pickup_name || "—",
        relation: relationLabel(r.pickup_relation),
        time: formatClock(r.recorded_at),
        abnormal: Boolean(r.is_abnormal)
      }))
  }))
})

const isFirstLoading = computed(() => loading.value && !hasLoaded.value)
const showError = computed(() => Boolean(loadError.value) && records.value.length === 0)
const showEmpty = computed(() => hasLoaded.value && !showError.value && records.value.length === 0)

async function loadData() {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const sid = getCurrentStudentId()
    const res = await fetchPickupRecords({ student_id: sid || undefined })
    records.value = res.data?.list || []
    hasLoaded.value = true
    // 姓名/班级与「今天」只服务凭证卡头部：异步补齐，失败不阻断记录本身，
    // 也不让整页骨架等这次请求。
    fetchParentHomeOverview(sid)
      .then(overviewRes => {
        const data = overviewRes?.data
        if (!data) return
        syncCurrentStudentFromOverview(data)
        const stu = data.current_student
        if (stu) {
          student.value = { name: stu.name || "", class_name: stu.class_name || "" }
        }
        const date = String(data.date || "")
        if (/^\d{4}-\d{2}-\d{2}$/.test(date)) overviewDate.value = date
      })
      .catch(() => {})
  } catch (err) {
    console.warn("[pickup-index] load failed:", err?.message || err)
    // 已加载过内容时保留旧数据，只有一次都没拿到才切整页错误态。
    if (records.value.length === 0) loadError.value = "接送记录加载失败"
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

async function handleRetry() {
  if (loading.value) return
  retrying.value = true
  await loadData()
  retrying.value = false
  if (loadError.value) {
    uni.showToast({ title: "仍无法加载，请稍后再试", icon: "none" })
  } else {
    uni.showToast({ title: "已重新加载", icon: "none" })
  }
}

function goQRCode() {
  if (loading.value) return
  uni.navigateTo({ url: "/pages/pickup/qrcode" })
}

// tabBar 页必须用 onShow：switchTab 切回来不重跑 onLoad，切完孩子会看不到变化。
// 同时把原生底栏高亮校正到「接送」（index 2）——组件内算不准当前 tab，见 utils/tabbar.js。
onShow(() => {
  syncTabBar(2)
  loadData()
})

// 刷新粒度是整页：接口只有整页 GET，历史日期组不再单独提供刷新。
onPullDownRefresh(loadData)
</script>

<template>
  <PageShell bottom-space="140rpx" max-width="430px">
    <!-- ===== 加载：凭证卡骨架 + 3 行列表骨架 ===== -->
    <view v-if="isFirstLoading" class="loading-state">
      <view class="sk-pass">
        <view class="sk-top">
          <view class="sk sk-chip" />
          <view class="sk sk-chip sk-chip--sm" />
        </view>
        <view class="sk sk-title" />
        <view class="sk-status">
          <view class="sk sk-seal" />
          <view class="sk-lines">
            <view class="sk sk-line sk-line--lg" />
            <view class="sk sk-line" />
          </view>
        </view>
        <view class="sk sk-kicker" />
        <view class="sk-track">
          <view class="sk sk-dot" />
          <view class="sk sk-line sk-line--track" />
        </view>
        <view class="sk-track">
          <view class="sk sk-dot" />
          <view class="sk sk-line sk-line--track" />
        </view>
        <view class="sk sk-cta" />
      </view>
      <view class="sk-list">
        <view v-for="i in 3" :key="i" class="sk-item">
          <view class="sk sk-pill" />
          <view class="sk sk-line sk-line--name" />
          <view class="sk sk-time" />
        </view>
      </view>
    </view>

    <!-- ===== 错误：先于空态判定，避免把加载失败误读成「没有记录」 ===== -->
    <view v-else-if="showError" class="error-state">
      <view class="error-icon"><text class="error-glyph">{{ ICON.error }}</text></view>
      <text class="error-title">接送记录加载失败</text>
      <text class="error-desc">网络好像不太稳定</text>
      <button class="retry-btn" :disabled="retrying" @click="handleRetry">
        {{ retrying ? "重试中…" : "重试" }}
      </button>
    </view>

    <!-- ===== 空：今天没记录且没有任何历史 ===== -->
    <view v-else-if="showEmpty" class="empty-state">
      <view class="empty-art"><text class="empty-glyph">{{ ICON.qr }}</text></view>
      <text class="empty-title">今天还没有接送记录</text>
      <text class="empty-desc">老师核验接送码后，记录会出现在这里</text>
      <button class="qr-cta empty-cta" @click="goQRCode">
        <text class="qr-cta-icon">{{ ICON.qr }}</text>
        <text>出示接送二维码</text>
      </button>
    </view>

    <!-- ===== 正常 ===== -->
    <template v-else>
      <!-- 今日接送凭证：全页唯一的大面积暖色块 -->
      <view class="pass">
        <view class="pass-head">
          <view class="pass-head-top">
            <text class="pass-eyebrow">今日接送凭证</text>
            <view v-if="childLabel" class="child-chip"><text>{{ childLabel }}</text></view>
          </view>
          <view class="pass-date-row">
            <text class="pass-date">{{ todayDateText }}</text>
            <text v-if="todayWeekday" class="pass-week">{{ todayWeekday }}</text>
          </view>

          <view class="pass-status">
            <view class="seal" :class="'seal--' + dayState">
              <text class="seal-glyph">
                {{ dayState === "picked" ? ICON.check : dayState === "in" ? ICON.sprout : ICON.clock }}
              </text>
            </view>
            <view class="status-copy">
              <text class="status-main">{{ statusMain }}</text>
              <view class="status-sub">
                <template v-if="dayState !== 'pre'">
                  <text class="status-time">{{ formatClock(lastToday && lastToday.recorded_at) }}</text>
                  <text class="status-dot">·</text>
                  <text class="status-who">
                    {{ (lastToday && lastToday.pickup_name) || "—" }}{{ dayState === "in" ? " 送入" : "" }}
                  </text>
                </template>
                <text v-else class="status-who">今天还没有入园记录</text>
              </view>
            </view>
          </view>
        </view>

        <view v-if="dayState !== 'pre'" class="pass-body">
          <text class="track-label">今日轨迹</text>
          <view class="track">
            <view v-for="item in timelineRecords" :key="item.id" class="track-item">
              <view class="track-mark">
                <view class="track-dot" :class="Number(item.type) === 2 ? 'track-dot--out' : 'track-dot--in'" />
              </view>
              <view class="track-content">
                <view class="track-top">
                  <text class="track-type" :class="Number(item.type) === 2 ? 'track-type--out' : 'track-type--in'">
                    {{ item.type_label || (Number(item.type) === 2 ? "接走" : "入园") }}
                  </text>
                  <text class="track-time">{{ formatClock(item.recorded_at) }}</text>
                  <text class="track-rel">{{ relationLabel(item.pickup_relation) }}</text>
                </view>
                <text class="track-who">{{ item.pickup_name || "—" }}</text>
              </view>
            </view>
          </view>
        </view>

        <view class="pass-foot">
          <!-- 已接走是终态：再出示二维码会与「今日接送已完成」自相矛盾 -->
          <view v-if="dayState === 'picked'" class="done-note">
            <text class="done-icon">{{ ICON.check }}</text>
            <text>今日接送已完成</text>
          </view>
          <template v-else>
            <button class="qr-cta" :disabled="loading" @click="goQRCode">
              <text class="qr-cta-icon">{{ ICON.qr }}</text>
              <text>出示接送二维码</text>
            </button>
            <view class="pass-note">
              <text>核验后</text>
              <text>一次有效</text>
            </view>
          </template>
        </view>
      </view>

      <!-- 历史：按日期分组，今天的两条不在这里重复出现 -->
      <view v-if="historyGroups.length" class="history">
        <view class="history-head">
          <text class="history-title">历史记录</text>
        </view>
        <view class="ledger">
          <view v-for="group in historyGroups" :key="group.key" class="day">
            <view class="day-head">
              <text class="day-title">{{ group.label }}</text>
              <text v-if="group.weekday" class="day-week">{{ group.weekday }}</text>
            </view>
            <view v-for="row in group.rows" :key="row.id" class="rec-row">
              <text class="rec-type" :class="row.isOut ? 'rec-type--out' : 'rec-type--in'">{{ row.typeLabel }}</text>
              <view class="rec-main">
                <view class="rec-name-row">
                  <text class="rec-name">{{ row.name }}</text>
                  <text v-if="row.abnormal" class="abnormal-tag">异常</text>
                </view>
                <text class="rec-rel">{{ row.relation }}</text>
              </view>
              <text class="rec-time">{{ row.time }}</text>
            </view>
          </view>
        </view>
      </view>
    </template>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 接送记录 · 今日凭证 + 历史流水
// 冷雾白底；暖色只落在凭证头部、主按钮与「接走」节点上。
// ============================================================

.pass {
  margin-top: $spacing-lg;
  overflow: hidden;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.pass-head {
  padding: 28rpx 32rpx 32rpx;
  background: $gradient-brand-soft;
}

.pass-head-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  min-width: 0;
}

.pass-eyebrow {
  flex: 1;
  min-width: 0;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.child-chip {
  flex: none;
  max-width: 300rpx;
  padding: 6rpx 18rpx;
  overflow: hidden;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
  box-shadow: $shadow-sm;
}

.pass-date-row {
  display: flex;
  align-items: baseline;
  gap: $spacing-sm;
  margin-top: $spacing-sm;
}

.pass-date {
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.pass-week {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.pass-status {
  display: flex;
  align-items: flex-start;
  gap: $spacing-lg;
  margin-top: $spacing-lg;
}

.seal {
  flex: none;
  width: 88rpx;
  height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-brand-primary-dark;
  box-shadow: $shadow-sm;
}

.seal--pre {
  color: $color-text-muted;
}

.seal-glyph {
  font-family: remixicon;
  font-size: 44rpx;
  line-height: 1;
}

.status-copy {
  min-width: 0;
  padding-top: 2rpx;
}

// 设计稿 24–26px，超出 token 上限 40rpx；仅此一处状态标题使用。
.status-main {
  display: block;
  color: $color-text-primary;
  font-size: 48rpx;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.status-sub {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: $spacing-sm;
  margin-top: 6rpx;
}

.status-time {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.status-dot {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1;
}

.status-who {
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.pass-body {
  padding: 28rpx 32rpx 0;
}

.track-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.track {
  margin-top: $spacing-xs;
}

.track-item {
  display: grid;
  grid-template-columns: 28rpx minmax(0, 1fr);
  column-gap: 20rpx;
}

.track-mark {
  position: relative;
  display: flex;
  justify-content: center;
  padding-top: 16rpx;
}

.track-mark::after {
  content: "";
  position: absolute;
  top: 34rpx;
  bottom: -24rpx;
  left: 50%;
  width: 1rpx;
  background: $color-border-hairline;
  transform: translateX(-50%);
}

.track-item:last-child .track-mark::after {
  display: none;
}

.track-dot {
  position: relative;
  z-index: 1;
  width: 18rpx;
  height: 18rpx;
  border-radius: $border-radius-full;
}

.track-dot--in {
  background: $color-semantic-success;
}

.track-dot--out {
  background: $color-brand-primary;
}

.track-content {
  min-width: 0;
  padding: 4rpx 0 28rpx;
}

.track-item:last-child .track-content {
  padding-bottom: 0;
}

.track-top {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-md;
  min-width: 0;
}

.track-type {
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.track-type--in {
  color: $color-semantic-success;
}

.track-type--out {
  color: $color-brand-primary-dark;
}

.track-time {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.track-rel {
  margin-left: auto;
  padding: 2rpx 16rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

.track-who {
  display: block;
  margin-top: 4rpx;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.pass-foot {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: $spacing-sm;
  margin-top: 28rpx;
  padding: 28rpx 32rpx 32rpx;
  border-top: 1rpx solid $color-border-subtle;
}

.qr-cta {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  width: 100%;
  height: $component-input-height;
  padding: 0 28rpx;
  margin: 0;
  border: none;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  box-shadow: $shadow-md;
}

.qr-cta::after {
  border: none;
}

.qr-cta[disabled] {
  opacity: 0.45;
  box-shadow: none;
}

.qr-cta-icon {
  font-family: remixicon;
  font-size: 38rpx;
  line-height: 1;
}

.pass-note {
  display: flex;
  flex-direction: column;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.45;
}

.done-note {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  min-height: $component-input-height;
  border-radius: $border-radius-full;
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.done-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

// ---- 历史 ----------------------------------------------------
.history {
  margin-top: 40rpx;
}

.history-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  padding: 0 4rpx 16rpx;
}

.history-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.ledger {
  overflow: hidden;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.day + .day {
  border-top: 1rpx solid $color-border-hairline;
}

.day-head {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-height: $component-input-height;
  padding: 4rpx $spacing-md 4rpx 32rpx;
}

.day-title {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.day-week {
  padding: 4rpx 14rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

.rec-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  column-gap: $spacing-lg;
  min-height: 116rpx;
  padding: 18rpx 44rpx 18rpx 32rpx;
  border-top: 1rpx solid $color-border-hairline;
}

.rec-type {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 88rpx;
  height: 48rpx;
  padding: 0 16rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.rec-type--in {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

// 接走在今日轨迹与历史里统一用品牌色，避免同一动作两套语义色。
.rec-type--out {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.rec-main {
  display: flex;
  flex-direction: column;
  gap: 2rpx;
  min-width: 0;
}

.rec-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-sm;
  min-width: 0;
}

.rec-name {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.rec-rel {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

.abnormal-tag {
  flex: none;
  padding: 2rpx 12rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.5;
}

.rec-time {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

// ---- 空态 / 错误态 ------------------------------------------
.empty-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 168rpx 32rpx 80rpx;
  text-align: center;
}

.empty-art {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 192rpx;
  height: 192rpx;
  margin-bottom: 36rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand-soft;
  color: $color-brand-primary-dark;
  box-shadow: $shadow-sm;
}

.empty-glyph {
  font-family: remixicon;
  font-size: 116rpx;
  line-height: 1;
}

.empty-title {
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.empty-desc {
  max-width: 520rpx;
  margin-top: $spacing-md;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.7;
}

.empty-cta {
  width: auto;
  min-width: 400rpx;
  margin-top: 44rpx;
  padding: 0 56rpx;
}

.error-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 104rpx;
  height: 104rpx;
  margin-bottom: 28rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.error-glyph {
  font-family: remixicon;
  font-size: 52rpx;
  line-height: 1;
}

.error-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.error-desc {
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.retry-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 320rpx;
  height: $component-input-height;
  margin-top: 40rpx;
  padding: 0 48rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
  box-shadow: $shadow-sm;
}

.retry-btn::after {
  border: none;
}

.retry-btn[disabled] {
  opacity: 0.55;
  box-shadow: none;
}

// ---- 骨架 ----------------------------------------------------
.sk-pass {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  padding: $spacing-xl;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sk-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}

.sk {
  display: block;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
  animation: pickup-sk 1.5s $animation-easing-default infinite;
}

.sk-chip {
  width: 168rpx;
  height: 24rpx;
}

.sk-chip--sm {
  width: 128rpx;
}

.sk-title {
  width: 264rpx;
  height: 44rpx;
  margin-top: 2rpx;
}

.sk-status {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
}

.sk-seal {
  flex: none;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
}

.sk-lines {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
}

.sk-line {
  width: 100%;
  height: 24rpx;
}

.sk-line--lg {
  width: 60%;
  height: 40rpx;
}

.sk-kicker {
  width: 96rpx;
  height: 20rpx;
}

.sk-track {
  display: flex;
  align-items: center;
  gap: 20rpx;
}

.sk-track .sk-line {
  width: 62%;
}

.sk-line--track {
  height: 24rpx;
}

.sk-dot {
  flex: none;
  width: 18rpx;
  height: 18rpx;
  border-radius: $border-radius-full;
}

.sk-cta {
  width: 100%;
  height: $component-input-height;
  margin-top: 4rpx;
  border-radius: $border-radius-full;
}

.sk-list {
  margin-top: $spacing-lg;
  overflow: hidden;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sk-item {
  display: grid;
  grid-template-columns: 88rpx minmax(0, 1fr) 88rpx;
  align-items: center;
  gap: $spacing-lg;
  min-height: 116rpx;
  padding: 20rpx 32rpx;
}

.sk-item + .sk-item {
  border-top: 1rpx solid $color-border-subtle;
}

.sk-pill {
  width: 88rpx;
  height: 44rpx;
}

.sk-line--name {
  width: 72%;
  height: 24rpx;
}

.sk-time {
  width: 80rpx;
  height: 24rpx;
  justify-self: end;
}

@keyframes pickup-sk {
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
