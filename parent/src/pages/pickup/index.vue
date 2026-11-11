<script setup>
/**
 * 接送记录（家长端 tab 页）
 *
 * 今天的记录聚合成一张「接送凭证」：状态、时间轴、二维码入口都在卡内；
 * 过去按日期分组退成流水。数据仍来自 GET /parent/pickup/records 一次拉取，
 * 没有新增接口 —— 三态（未入园 / 在园中 / 已接走）全部由当天记录派生。
 */
import { computed, ref } from "vue"
import { onShow, onHide, onUnload, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { syncTabBar } from "@/utils/tabbar"
import { fetchPickupRecords } from "@/api/parent-pickup"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { relationLabel } from "@/utils/relation"
import { formatClock, formatDateLabel } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// DNA 图标按色调导出（WXML 的 image 不能继承 currentColor）：
// pine 用于浅底，muted 用于次级，bone 只用于 pine 实底上的图标。
function iconSrc(name, tone = "pine") {
  return `/static/icons/dna/${tone}/icon-${name}.svg`
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
const sealIcon = computed(() => (dayState.value === "picked" ? "check" : dayState.value === "in" ? "sprout" : "clock"))
const sealTone = computed(() => (dayState.value === "pre" ? "muted" : "pine"))
const nowClock = ref(formatClock(new Date()))
// 「未入园」必须带时间上下文：早上 8 点看到是正常，上午 10 点看到才值得担心。
const statusSubText = computed(() => (dayState.value === "pre" ? `截至 ${nowClock.value || "--:--"} 未入园` : ""))

const inRecord = computed(() => todayRecords.value.find(r => Number(r.type) === 1) || null)
const outRecord = computed(() => todayRecords.value.find(r => Number(r.type) === 2) || null)

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
        name: r.pickup_name || "",
        relation: relationLabel(r.pickup_relation) || "其他关系",
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
    return false
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
  return true
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
let clockTimer = 0
function syncClock() {
  nowClock.value = formatClock(new Date())
}

onShow(() => {
  syncClock()
  if (!clockTimer) clockTimer = setInterval(syncClock, 60000)
  syncTabBar(2)
  loadData()
})

function stopClock() {
  if (clockTimer) {
    clearInterval(clockTimer)
    clockTimer = 0
  }
}

onHide(stopClock)
onUnload(stopClock)

// 刷新粒度是整页：接口只有整页 GET，历史日期组不再单独提供刷新。
onPullDownRefresh(loadData)
</script>

<template>
  <PageShell background="#ece6d7" bottom-space="140rpx" max-width="430px">
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
      <view class="error-art">
        <image class="error-illustration" src="/static/illustrations/pickup-error.svg" mode="aspectFit" />
      </view>
      <text class="error-title">接送记录加载失败</text>
      <text class="error-desc">网络好像不太稳定</text>
      <button class="retry-btn" :disabled="retrying" @click="handleRetry">
        {{ retrying ? "重试中…" : "重试" }}
      </button>
    </view>

    <!-- ===== 空：今天没记录且没有任何历史 ===== -->
    <view v-else-if="showEmpty" class="empty-state">
      <view class="empty-art">
        <image class="empty-illustration" src="/static/illustrations/pickup-empty.svg" mode="aspectFit" />
      </view>
      <text class="empty-title">今天还没有接送记录</text>
      <text class="empty-desc">老师核验接送码后，记录会出现在这里</text>
      <button class="qr-cta empty-cta" @click="goQRCode">
        <image class="qr-cta-icon" src="/static/icons/dna/bone/icon-qr.svg" mode="aspectFit" />
        <text>出示接送二维码</text>
      </button>
    </view>

    <!-- ===== 正常 ===== -->
    <template v-else>
      <!-- 大标题与日期：DNA 里标题承担层级，不再放进卡片 -->
      <view class="pickup-head">
        <view class="pickup-head-copy">
          <text class="pickup-overline">{{ todayDateText }}<text v-if="todayWeekday"> · {{ todayWeekday }}</text></text>
          <text class="pickup-title">接送</text>
        </view>
        <button class="head-cta" @click="goQRCode">接送码</button>
      </view>

      <!-- 今日接送凭证：白面 + 发丝线，状态与进度轨都在卡内 -->
      <view class="pass">
        <view class="pass-head">
          <view class="pass-copy">
            <text class="pass-eyebrow">今日接送凭证</text>
            <text class="pass-state">{{ statusMain }}</text>
            <text v-if="dayState === 'pre'" class="pass-sub">{{ statusSubText }}</text>
            <text v-else-if="childLabel" class="pass-sub">{{ childLabel }}</text>
          </view>
          <view class="seal" :class="'seal--' + dayState">
            <image class="seal-glyph" :src="iconSrc(sealIcon, sealTone)" mode="aspectFit" />
          </view>
        </view>

        <view v-if="dayState !== 'pre'" class="pass-rail" aria-label="今日接送进度">
          <view class="rail-step rail-step--in is-done">
            <view class="rail-dot">
              <image class="rail-glyph" src="/static/icons/dna/bone/icon-check.svg" mode="aspectFit" />
            </view>
            <view class="rail-copy">
              <text class="rail-label">送入</text>
              <text class="rail-meta">
                {{ formatClock(inRecord && inRecord.recorded_at) }} · {{ (inRecord && inRecord.pickup_name) || "接送人未记录" }}
              </text>
            </view>
          </view>
          <view class="rail-line" :class="{ 'is-done': dayState === 'picked' }" />
          <view class="rail-step rail-step--out" :class="{ 'is-done': dayState === 'picked' }">
            <view class="rail-dot">
              <image
                class="rail-glyph"
                :src="iconSrc(dayState === 'picked' ? 'check' : 'clock', dayState === 'picked' ? 'bone' : 'muted')"
                mode="aspectFit"
              />
            </view>
            <view class="rail-copy">
              <text class="rail-label">接走</text>
              <text v-if="dayState === 'picked'" class="rail-meta">
                {{ formatClock(outRecord && outRecord.recorded_at) }} · {{ (outRecord && outRecord.pickup_name) || "接送人未记录" }}
              </text>
              <text v-else class="rail-meta">等待接走</text>
            </view>
          </view>
        </view>

        <view class="pass-foot">
          <!-- 已接走是终态：再出示二维码会与「今日接送已完成」自相矛盾 -->
          <view v-if="dayState === 'picked'" class="done-note">
            <image class="done-icon" src="/static/icons/dna/pine/icon-check.svg" mode="aspectFit" />
            <text>
              今日接送已完成 · {{ formatClock(lastToday && lastToday.recorded_at) }}
              {{ (lastToday && lastToday.pickup_name) || "接送人未记录" }} 接走
            </text>
          </view>
          <template v-else>
            <button class="qr-cta" :disabled="loading" @click="goQRCode">
              <image class="qr-cta-icon" src="/static/icons/dna/bone/icon-qr.svg" mode="aspectFit" />
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
              <view class="rec-type" :class="row.isOut ? 'rec-type--out' : 'rec-type--in'">
                <image
                  class="rec-type-icon"
                  :src="row.isOut ? '/static/icons/dna/pine/icon-logout.svg' : '/static/icons/dna/pine/icon-login.svg'"
                  mode="aspectFit"
                />
                <text>{{ row.typeLabel }}</text>
              </view>
              <view class="rec-main">
                <view class="rec-name-row">
                  <text class="rec-name" :class="{ 'is-missing': !row.name }">{{ row.name || "接送人未记录" }}</text>
                  <text v-if="row.abnormal" class="abnormal-tag">异常</text>
                </view>
                <text v-if="row.relation && row.relation !== row.name" class="rec-rel">{{ row.relation }}</text>
              </view>
              <view class="rec-end">
                <text class="rec-time">{{ row.time }}</text>
              </view>
            </view>
          </view>
        </view>
      </view>
    </template>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 接送记录 · Kinderly DNA 迁移验证
// 暖纸底 + 深松绿；白面靠 1rpx 发丝线分层，不用阴影。
// 数值按 750rpx 设计基准（1px = 2rpx），字体只用系统可用族。
// ============================================================

$dna-bone: #ece6d7;
$dna-card: #ffffff;
$dna-card-soft: #fcfaf4;
$dna-pine: #173220;
$dna-ink-mid: #3d5141;
$dna-ink-soft: #728173;
$dna-sage-deep: #5c7359;
$dna-sage-soft: #cfd9c5;
$dna-marigold-soft: #efd485;
$dna-terracotta: #c36540;
$dna-terracotta-soft: #e5bca1;
$dna-hair: rgba(23, 50, 32, 0.1);
$dna-hair-strong: rgba(23, 50, 32, 0.22);
// 小程序拿不到 Instrument Serif；中文回落宋体，Android 无宋体时退到无衬线。
$dna-serif: "Noto Serif SC", "Songti SC", "STSong", serif;
$dna-sans: -apple-system, "PingFang SC", "Microsoft YaHei", sans-serif;

// ---- 页头 ----------------------------------------------------
.pickup-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24rpx;
  margin: 8rpx 0 28rpx;
}

.pickup-head-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.pickup-overline {
  color: $dna-ink-soft;
  font-family: $dna-sans;
  font-size: 22rpx;
  font-weight: 600;
  letter-spacing: 0.14em;
  line-height: 1.4;
}

.pickup-title {
  margin-top: 8rpx;
  color: $dna-pine;
  font-family: $dna-serif;
  font-size: 60rpx;
  font-weight: 400;
  letter-spacing: 0;
  line-height: 1.35;
}

.head-cta {
  flex: none;
  height: 64rpx;
  padding: 0 24rpx;
  margin: 0;
  border: 1rpx solid $dna-hair-strong;
  border-radius: 999rpx;
  background: $dna-card;
  color: $dna-pine;
  font-family: $dna-sans;
  font-size: 26rpx;
  font-weight: 500;
  line-height: 62rpx;
}

.head-cta::after {
  border: none;
}

// ---- 凭证卡 --------------------------------------------------
.pass {
  margin-bottom: 28rpx;
  overflow: hidden;
  border: 1rpx solid $dna-hair;
  border-radius: 40rpx;
  background: $dna-card;
}

.pass-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24rpx;
  padding: 32rpx 32rpx 0;
}

.pass-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.pass-eyebrow {
  color: $dna-ink-soft;
  font-family: $dna-sans;
  font-size: 22rpx;
  font-weight: 600;
  letter-spacing: 0.12em;
  line-height: 1.4;
}

.pass-state {
  margin-top: 10rpx;
  color: $dna-pine;
  font-family: $dna-serif;
  font-size: 60rpx;
  font-weight: 400;
  letter-spacing: 0;
  line-height: 1.35;
}

.pass-sub {
  margin-top: 8rpx;
  color: $dna-ink-soft;
  font-family: $dna-sans;
  font-size: 26rpx;
  line-height: 1.5;
}

.seal {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 96rpx;
  height: 96rpx;
  border-radius: 999rpx;
  background: $dna-sage-soft;
  color: $dna-sage-deep;
}

.seal--pre {
  background: $dna-bone;
  color: $dna-ink-soft;
}

.seal--picked {
  background: $dna-marigold-soft;
  color: $dna-pine;
}

.seal-glyph {
  width: 40rpx;
  height: 40rpx;
}

// ---- 进度轨 --------------------------------------------------
.pass-rail {
  display: flex;
  align-items: flex-start;
  gap: 12rpx;
  padding: 32rpx 32rpx 8rpx;
}

.rail-step {
  display: flex;
  align-items: flex-start;
  flex: 1 1 0;
  gap: 14rpx;
  min-width: 0;
}

.rail-dot {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 44rpx;
  height: 44rpx;
  margin-top: 0;
  border-radius: 999rpx;
  border: 2rpx solid $dna-hair-strong;
  background: $dna-card;
  color: $dna-card;
}

.rail-step.is-done .rail-dot {
  border-color: $dna-sage-deep;
  background: $dna-sage-deep;
}

.rail-glyph {
  width: 28rpx;
  height: 28rpx;
}

.rail-line {
  flex: 0 0 48rpx;
  height: 0;
  margin-top: 21rpx;
  border-top: 2rpx dashed $dna-hair-strong;
}

.rail-line.is-done {
  border-top-style: solid;
  border-top-color: $dna-sage-deep;
}

.rail-copy {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}

.rail-label {
  color: $dna-pine;
  font-family: $dna-sans;
  font-size: 24rpx;
  font-weight: 600;
  line-height: 1.4;
}

.rail-meta {
  color: $dna-ink-soft;
  font-family: $dna-sans;
  font-size: 24rpx;
  line-height: 1.4;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

// ---- 卡底操作 ------------------------------------------------
.pass-foot {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 12rpx;
  margin-top: 24rpx;
  padding: 24rpx 32rpx 32rpx;
  border-top: 1rpx solid $dna-hair;
}

.qr-cta {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  width: 100%;
  height: 96rpx;
  padding: 0 28rpx;
  margin: 0;
  border: none;
  border-radius: 999rpx;
  background: $dna-pine;
  color: $dna-bone;
  font-family: $dna-sans;
  font-size: 29rpx;
  font-weight: 500;
  line-height: 1.3;
}

.qr-cta::after {
  border: none;
}

.qr-cta[disabled] {
  opacity: 0.45;
}

.qr-cta-icon {
  width: 34rpx;
  height: 34rpx;
}

.pass-note {
  display: flex;
  justify-content: center;
  gap: 12rpx;
  color: $dna-ink-soft;
  font-family: $dna-sans;
  font-size: 22rpx;
  line-height: 1.5;
}

.done-note {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  min-height: 96rpx;
  border-radius: 999rpx;
  background: rgba(141, 163, 134, 0.18);
  color: $dna-pine;
  font-family: $dna-sans;
  font-size: 26rpx;
  font-weight: 500;
  line-height: 1.4;
  text-align: center;
}

.done-icon {
  width: 30rpx;
  height: 30rpx;
}

// ---- 历史 ----------------------------------------------------
.history {
  margin-top: 8rpx;
}

.history-head {
  padding: 0 4rpx 16rpx;
}

.history-title {
  color: $dna-pine;
  font-family: $dna-serif;
  font-size: 40rpx;
  font-weight: 400;
  letter-spacing: 0;
  line-height: 1.35;
}

.ledger {
  overflow: hidden;
  border: 1rpx solid $dna-hair;
  border-radius: 40rpx;
  background: $dna-card;
}

.day + .day {
  border-top: 1rpx solid $dna-hair;
}

.day-head {
  display: flex;
  align-items: center;
  gap: 14rpx;
  min-height: 88rpx;
  padding: 8rpx 32rpx;
}

.day-title {
  color: $dna-pine;
  font-family: $dna-serif;
  font-size: 32rpx;
  font-weight: 400;
  line-height: 1.35;
}

.day-week {
  padding: 4rpx 14rpx;
  border-radius: 999rpx;
  background: $dna-card-soft;
  color: $dna-ink-soft;
  font-family: $dna-sans;
  font-size: 22rpx;
  line-height: 1.5;
}

.rec-row {
  display: flex;
  align-items: center;
  gap: 20rpx;
  min-height: 116rpx;
  padding: 18rpx 32rpx;
  border-top: 1rpx solid $dna-hair;
}

.rec-type {
  display: inline-flex;
  align-items: center;
  flex: none;
  gap: 6rpx;
  padding: 6rpx 16rpx;
  border-radius: 999rpx;
  background: $dna-sage-soft;
  color: $dna-pine;
  font-family: $dna-sans;
  font-size: 22rpx;
  font-weight: 500;
  line-height: 1.2;
}

.rec-type--out {
  background: $dna-marigold-soft;
}

.rec-type-icon {
  width: 28rpx;
  height: 28rpx;
}

.rec-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2rpx;
  min-width: 0;
}

.rec-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12rpx;
  min-width: 0;
}

.rec-name {
  color: $dna-pine;
  font-family: $dna-sans;
  font-size: 28rpx;
  font-weight: 500;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.rec-name.is-missing {
  color: $dna-terracotta;
  font-weight: 600;
}

.rec-rel {
  color: $dna-ink-soft;
  font-family: $dna-sans;
  font-size: 22rpx;
  line-height: 1.5;
}

.abnormal-tag {
  flex: none;
  padding: 2rpx 12rpx;
  border-radius: 999rpx;
  background: $dna-terracotta-soft;
  color: $dna-pine;
  font-family: $dna-sans;
  font-size: 22rpx;
  font-weight: 500;
  line-height: 1.5;
}

.rec-end {
  display: flex;
  align-items: center;
  flex: none;
}

.rec-time {
  color: $dna-ink-mid;
  font-family: $dna-sans;
  font-size: 26rpx;
  font-weight: 500;
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

.empty-art,
.error-art {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 176rpx;
  height: 176rpx;
  margin-bottom: 36rpx;
  border-radius: 999rpx;
  background: $dna-card-soft;
}

.empty-illustration,
.error-illustration {
  width: 148rpx;
  height: 148rpx;
}

.empty-title,
.error-title {
  color: $dna-pine;
  font-family: $dna-serif;
  font-size: 40rpx;
  font-weight: 400;
  letter-spacing: 0;
  line-height: 1.35;
}

.empty-desc,
.error-desc {
  margin-top: 10rpx;
  color: $dna-ink-soft;
  font-family: $dna-sans;
  font-size: 26rpx;
  line-height: 1.5;
}

.empty-cta {
  width: auto;
  min-width: 360rpx;
  margin-top: 32rpx;
  padding: 0 40rpx;
}

.retry-btn {
  height: 88rpx;
  margin-top: 32rpx;
  padding: 0 56rpx;
  border: none;
  border-radius: 999rpx;
  background: $dna-pine;
  color: $dna-bone;
  font-family: $dna-sans;
  font-size: 28rpx;
  line-height: 88rpx;
}

.retry-btn::after {
  border: none;
}

.retry-btn[disabled] {
  opacity: 0.5;
}

// ---- 骨架 ----------------------------------------------------
.loading-state {
  padding-top: 8rpx;
}

.sk {
  border-radius: 999rpx;
  background: rgba(23, 50, 32, 0.07);
}

.sk-pass {
  padding: 32rpx;
  border: 1rpx solid $dna-hair;
  border-radius: 40rpx;
  background: $dna-card;
}

.sk-top {
  display: flex;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.sk-chip {
  width: 120rpx;
  height: 20rpx;
}

.sk-chip--sm {
  width: 80rpx;
}

.sk-title {
  width: 300rpx;
  height: 56rpx;
  margin-bottom: 28rpx;
}

.sk-status {
  display: flex;
  align-items: center;
  gap: 24rpx;
  margin-bottom: 32rpx;
}

.sk-seal {
  width: 96rpx;
  height: 96rpx;
  border-radius: 999rpx;
}

.sk-lines {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 12rpx;
}

.sk-line {
  width: 100%;
  height: 20rpx;
}

.sk-line--lg {
  width: 60%;
  height: 28rpx;
}

.sk-kicker {
  width: 40%;
  height: 16rpx;
  margin-bottom: 16rpx;
}

.sk-track {
  display: flex;
  align-items: center;
  gap: 16rpx;
  margin-bottom: 28rpx;
}

.sk-dot {
  width: 24rpx;
  height: 24rpx;
  border-radius: 999rpx;
}

.sk-line--track {
  width: 120rpx;
  height: 4rpx;
  border-radius: 0;
}

.sk-cta {
  width: 100%;
  height: 96rpx;
}

.sk-list {
  margin-top: 40rpx;
  padding: 0 32rpx;
  border: 1rpx solid $dna-hair;
  border-radius: 40rpx;
  background: $dna-card;
}

.sk-item {
  display: flex;
  align-items: center;
  gap: 20rpx;
  min-height: 112rpx;
}

.sk-item + .sk-item {
  border-top: 1rpx solid $dna-hair;
}

.sk-pill {
  width: 88rpx;
  height: 34rpx;
}

.sk-line--name {
  width: 40%;
  height: 20rpx;
}

.sk-time {
  width: 60rpx;
  height: 20rpx;
  margin-left: auto;
}
</style>

