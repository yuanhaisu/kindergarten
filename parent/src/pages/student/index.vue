<script setup>
/**
 * 孩子信息 · 成长护照
 *
 * 身份带 + 2×2 档案事实格 + 被收进浅底块的健康备注 + 安静的晨检时间线 +
 * 已绑定孩子名单。档案与晨检各自维护错误态：晨检失败只降级晨检卡，
 * 不把整页拖成空白；档案失败才是整页错误。
 *
 * 缺失字段用「—」占位并给「刷新档案」，而不是 v-if 把整格藏掉——
 * 家长需要看见「这里还没填」，而不是以为档案本来就没有这一项。
 */
import { computed, ref } from "vue"
import { onLoad, onShow } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId, setCurrentStudentId } from "@/utils/currentStudent"
import { fetchStudentProfile, fetchStudents, unbindStudent } from "@/api/parent-student"
import { fetchHealthRecords } from "@/api/parent-health"
import { formatDateOnly, formatMonthDayWeekday } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  shield: "\u{f100}", // shield-check-line
  calendar: "\u{eb27}", // calendar-line
  user: "\u{f264}", // user-line
  alert: "\u{eca1}" // error-warning-line
}

const profile = ref({})
const students = ref([])
// 页面内唯一可信的「当前孩子」：来自本地存储并已用绑定列表校验。
const currentStudentId = ref(getCurrentStudentId())
const unbound = ref(false)
const loading = ref(false)
const retrying = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")

const healthRecords = ref([])
const healthLoading = ref(false)
const healthError = ref("")
const healthRetrying = ref(false)

const dialogStudent = ref(null)
const unbinding = ref(false)

// 首帧（onLoad 之前的第一次渲染）也走骨架：此时既没有数据也没有错误。
const showSkeleton = computed(() => !hasLoaded.value && !loadError.value)
const showError = computed(() => Boolean(loadError.value) && !loading.value)

const childName = computed(() => profile.value.name || "孩子")
const avatarInitial = computed(() => {
  const chars = Array.from(String(profile.value.name || "").trim())
  return chars.length ? chars[0] : "孩"
})
const classPill = computed(() => profile.value.class_name || "")
const healthNote = computed(() => String(profile.value.health_note || "").trim())

// 四个事实格都按「值 + 缺失占位」呈现，缺失检测与渲染用同一份数据。
const facts = computed(() => {
  const p = profile.value
  const gender = Number(p.gender) === 1 ? "男" : Number(p.gender) === 2 ? "女" : ""
  return [
    { key: "gender", label: "性别", value: gender, num: false },
    { key: "birthday", label: "生日", value: formatDateOnly(p.birthday), num: true },
    { key: "age", label: "年龄", value: p.age || "", num: false },
    { key: "enroll", label: "入园时间", value: formatDateOnly(p.enroll_date), num: true }
  ]
})
const hasMissing = computed(() => facts.value.some(fact => !fact.value))

const healthRows = computed(() =>
  (healthRecords.value || []).map((row, idx) => ({
    key: `${row.check_date || "row"}-${idx}`,
    date: formatMonthDayWeekday(row.check_date) || String(row.check_date || ""),
    status: row.overall_status_label || "已晨检",
    tone: healthTone(row.overall_status),
    abnormal: abnormalText(row.abnormal_items),
    temp: row.temperature === null || row.temperature === undefined
      ? ""
      : `${Number(row.temperature).toFixed(1)}°C`,
    isolated: Boolean(row.is_isolated),
    remark: row.remark || ""
  }))
)
const healthEmpty = computed(
  () => !healthLoading.value && !healthError.value && healthRows.value.length === 0
)

const unbindDesc = computed(() => {
  const student = dialogStudent.value
  if (!student) return ""
  const suffix = student.class_name ? `（${student.class_name}）` : ""
  return `确定解除与 ${student.name || "该孩子"}${suffix}的绑定吗？`
})

function healthTone(status) {
  if (Number(status) === 1) return "warn"
  if (Number(status) === 2) return "danger"
  return "ok"
}

function abnormalText(items) {
  if (!Array.isArray(items) || items.length === 0) return ""
  return items.map(item => `${item.part_label || ""}${item.status_label || ""}`).join("、")
}

async function loadHealth(options = {}) {
  const keepError = Boolean(options.keepError)
  const sid = currentStudentId.value || getCurrentStudentId()
  // 重试时保留错误块，让「重试中…」与 spinner 有可见的宿主；成功后统一清空。
  if (!keepError) healthError.value = ""
  if (!sid) {
    healthRecords.value = []
    healthLoading.value = false
    return
  }
  healthLoading.value = true
  try {
    const res = await fetchHealthRecords(sid)
    healthRecords.value = Array.isArray(res.data?.list) ? res.data.list : []
    healthError.value = ""
  } catch (err) {
    // 晨检失败不拖垮档案，但必须显式提示，不能静默留空。
    console.warn("[student-index] health load failed:", err?.message || err)
    healthError.value = "晨检记录加载失败"
    healthRecords.value = []
  } finally {
    healthLoading.value = false
  }
}

/**
 * 当前孩子只认本地选择，并用绑定列表校验：
 * GET /parent/students 不带 student_id，服务端的 is_current 只是「无参数时的
 * 回退孩子」（后端 ResolveContext 回退到 students[0]），不代表家长的选择。
 * 因此本地 id 仍在列表里时绝不覆盖；只有本地没选或已解绑（孤儿）时才采用
 * 服务端的 is_current / 第一个孩子回退。否则会把 A 的晨检渲染在 B 的档案下。
 */
function resolveCurrentStudentId() {
  const localId = getCurrentStudentId()
  if (localId && students.value.some(item => item.id === localId)) return localId
  const fallback = students.value.find(item => item.is_current) || students.value[0]
  return fallback ? fallback.id : null
}

async function loadData(options = {}) {
  const keepView = Boolean(options.keepView)
  if (!keepView) {
    loading.value = true
    loadError.value = ""
  }
  try {
    await ensureLogin()
    let studentsRes
    try {
      studentsRes = await fetchStudents()
    } catch (err) {
      // 零绑定孩子时后端返回 40300「未绑定孩子信息」，这不是网络错误：
      // 解绑最后一个孩子后会走到这里，必须给可理解的空态而不是「网络不稳定」。
      if (err?.code === 40300) {
        students.value = []
        profile.value = {}
        healthRecords.value = []
        healthError.value = ""
        currentStudentId.value = null
        setCurrentStudentId(null)
        unbound.value = true
        hasLoaded.value = true
        loadError.value = ""
        return
      }
      throw err
    }
    students.value = Array.isArray(studentsRes.data?.list) ? studentsRes.data.list : []
    unbound.value = students.value.length === 0
    if (unbound.value) {
      profile.value = {}
      healthRecords.value = []
      healthError.value = ""
      currentStudentId.value = null
      setCurrentStudentId(null)
      hasLoaded.value = true
      loadError.value = ""
      return
    }
    const activeId = resolveCurrentStudentId()
    currentStudentId.value = activeId
    if (activeId !== getCurrentStudentId()) setCurrentStudentId(activeId)
    const profileRes = await fetchStudentProfile(activeId)
    profile.value = profileRes.data || {}
    hasLoaded.value = true
    loadError.value = ""
    await loadHealth()
  } catch (err) {
    console.warn("[student-index] load failed:", err?.message || err)
    profile.value = {}
    students.value = []
    healthRecords.value = []
    healthError.value = ""
    unbound.value = false
    loadError.value = "档案加载失败"
  } finally {
    loading.value = false
  }
}

async function handleRetry() {
  if (retrying.value || loading.value) return
  retrying.value = true
  await loadData({ keepView: true })
  retrying.value = false
  const title = loadError.value
    ? "仍无法加载，请稍后再试"
    : unbound.value
      ? "还没有绑定孩子"
      : "已重新加载"
  uni.showToast({ title, icon: "none" })
}

async function handleHealthRetry() {
  if (healthRetrying.value || healthLoading.value) return
  healthRetrying.value = true
  await loadHealth({ keepError: true })
  healthRetrying.value = false
  uni.showToast({
    title: healthError.value ? "仍无法加载，请稍后再试" : "晨检已更新",
    icon: "none"
  })
}

function openUnbind(student) {
  if (unbinding.value) return
  dialogStudent.value = student
}

function closeUnbind() {
  if (unbinding.value) return
  dialogStudent.value = null
}

async function confirmUnbind() {
  const student = dialogStudent.value
  if (!student || unbinding.value) return
  unbinding.value = true
  try {
    await unbindStudent(student.id)
    const wasCurrent = student.id === currentStudentId.value
    if (wasCurrent) {
      // 解绑当前孩子后必须把本地 id 切到下一个（或清空）：
      // 留下孤儿 id 时后端会静默回退到第一个孩子，界面正常但身份已变。
      const next = students.value.find(item => item.id !== student.id)
      currentStudentId.value = next ? next.id : null
      setCurrentStudentId(currentStudentId.value)
    }
    dialogStudent.value = null
    uni.showToast({ title: "已解绑", icon: "none" })
    if (wasCurrent) hasLoaded.value = false
    await loadData(wasCurrent ? {} : { keepView: true })
  } catch (err) {
    console.warn("[student-index] unbind failed:", err?.message || err)
    uni.showToast({ title: err?.message || "解绑失败", icon: "none" })
  } finally {
    unbinding.value = false
  }
}

// 从其它页返回时静默刷新晨检；首次 onShow 紧随 onLoad，跳过。
let skipNextShow = true
onShow(() => {
  if (skipNextShow) {
    skipNextShow = false
    return
  }
  if (loading.value || retrying.value || loadError.value) return
  // 解绑到没有孩子后若在别处重新绑定，返回本页要能自愈，而不是停在空态。
  if (unbound.value) {
    loadData({ keepView: true })
    return
  }
  loadHealth()
})

onLoad(() => {
  loadData()
})
</script>

<template>
  <PageShell max-width="430px" bottom-space="48rpx">
    <!-- ===== 加载：档案 + 晨检 + 绑定三段骨架 ===== -->
    <template v-if="showSkeleton">
      <view class="card profile-card">
        <view class="sk-identity">
          <view class="sk sk-avatar" />
          <view class="sk-identity-lines">
            <view class="sk sk-line sk-line-name" />
            <view class="sk sk-line sk-line-pill" />
          </view>
        </view>
        <view class="sk-facts">
          <view
            v-for="i in 4"
            :key="i"
            class="sk-fact"
            :class="{ 'is-right': i % 2 === 0, 'is-bottom': i > 2 }"
          >
            <view class="sk sk-line sk-line-label" />
            <view class="sk sk-line sk-line-value" />
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-head">
          <text class="section-title">最近晨检</text>
        </view>
        <view class="card">
          <view v-for="i in 3" :key="i" class="sk-check-row">
            <view class="sk-check-head">
              <view class="sk sk-check-date" />
              <view class="sk sk-check-pill" />
            </view>
            <view class="sk sk-check-line" />
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-head">
          <text class="section-title">已绑定的孩子</text>
        </view>
        <view class="card">
          <view v-for="i in 2" :key="i" class="sk-bound-row">
            <view class="sk-bound-main">
              <view class="sk sk-bound-name" />
              <view class="sk sk-bound-class" />
            </view>
            <view class="sk sk-bound-btn" />
          </view>
        </view>
      </view>
    </template>

    <!-- ===== 整页失败 ===== -->
    <view v-else-if="showError" class="error-state">
      <view class="error-icon">
        <text class="error-glyph">{{ ICON.alert }}</text>
      </view>
      <text class="error-title">档案加载失败</text>
      <text class="error-desc">网络不太稳定，请稍后再试</text>
      <view
        class="primary-button"
        :class="{ 'is-busy': retrying }"
        :aria-disabled="retrying ? 'true' : 'false'"
        @click="handleRetry"
      >
        <text class="primary-button-text">{{ retrying ? "重试中…" : "重试" }}</text>
      </view>
    </view>

    <!-- ===== 零绑定孩子（解绑最后一个孩子后） ===== -->
    <view v-else-if="unbound" class="error-state">
      <view class="error-icon error-icon--empty">
        <text class="error-glyph error-glyph--empty">{{ ICON.user }}</text>
      </view>
      <text class="error-title">还没有绑定孩子</text>
      <text class="error-desc">请先在首页绑定孩子，再查看档案信息</text>
      <view
        class="primary-button"
        :class="{ 'is-busy': retrying }"
        :aria-disabled="retrying ? 'true' : 'false'"
        @click="handleRetry"
      >
        <text class="primary-button-text">{{ retrying ? "重试中…" : "重新加载" }}</text>
      </view>
    </view>

    <!-- ===== 正常 / 缺字段 ===== -->
    <view v-else class="page-body" :aria-hidden="dialogStudent ? 'true' : 'false'">
      <view class="card profile-card">
        <view class="identity">
          <view class="avatar">
            <image v-if="profile.avatar" class="avatar-img" :src="profile.avatar" mode="aspectFill" />
            <text v-else class="avatar-initial">{{ avatarInitial }}</text>
          </view>
          <view class="identity-main">
            <text class="identity-name">{{ childName }}</text>
            <text v-if="classPill" class="class-pill">{{ classPill }}</text>
          </view>
        </view>

        <view class="facts">
          <view
            v-for="(fact, idx) in facts"
            :key="fact.key"
            class="fact"
            :class="{ 'is-right': idx % 2 === 1, 'is-bottom': idx > 1 }"
          >
            <text class="fact-label">{{ fact.label }}</text>
            <text
              class="fact-value"
              :class="{ 'is-num': fact.num, 'is-empty': !fact.value }"
            >{{ fact.value || "—" }}</text>
          </view>
        </view>

        <!-- 健康备注是敏感信息：弱化排版，不用告警色 -->
        <view v-if="healthNote" class="health-note">
          <text class="health-note-icon">{{ ICON.shield }}</text>
          <view class="health-note-body">
            <text class="health-note-label">健康备注</text>
            <text class="health-note-text">{{ healthNote }}</text>
          </view>
        </view>
      </view>

      <view v-if="hasMissing" class="partial-hint">
        <text class="partial-hint-text">部分档案信息暂未填写</text>
        <view
          class="text-action"
          :class="{ 'is-busy': retrying }"
          :aria-disabled="retrying ? 'true' : 'false'"
          @click="handleRetry"
        >
          <text class="text-action-text">{{ retrying ? "重试中…" : "刷新档案" }}</text>
        </view>
      </view>

      <view class="section">
        <view class="section-head">
          <text class="section-title">最近晨检</text>
          <text v-if="!healthEmpty" class="section-meta">近 7 天</text>
        </view>
        <view class="card health-card">
          <!-- 重试中：错误块作为宿主保留，「重试中…」与 spinner 才可见 -->
          <view v-if="healthRetrying" class="health-inline-error">
            <text class="health-inline-title">晨检记录加载失败</text>
            <view class="health-inline-actions">
              <view class="text-action text-action--quiet is-busy" aria-disabled="true">
                <text class="text-action-text">重试中…</text>
              </view>
              <view class="inline-spinner" />
            </view>
          </view>

          <!-- 返回页面时会重载晨检：先给行内骨架，不整页闪白 -->
          <view v-else-if="healthLoading" class="health-skeleton">
            <view v-for="i in 3" :key="i" class="sk-check-row">
              <view class="sk-check-head">
                <view class="sk sk-check-date" />
                <view class="sk sk-check-pill" />
              </view>
              <view class="sk sk-check-line" />
            </view>
          </view>

          <view v-else-if="healthError" class="health-inline-error">
            <text class="health-inline-title">晨检记录加载失败</text>
            <view class="health-inline-actions">
              <view
                class="text-action text-action--quiet"
                aria-disabled="false"
                @click="handleHealthRetry"
              >
                <text class="text-action-text">点击重试</text>
              </view>
            </view>
          </view>

          <view v-else-if="healthEmpty" class="health-empty">
            <text class="health-empty-icon">{{ ICON.calendar }}</text>
            <text class="health-empty-text">近 7 天暂无晨检记录</text>
          </view>

          <view v-else class="health-records">
            <view v-for="row in healthRows" :key="row.key" class="check-row">
              <view class="check-head">
                <text class="check-date">{{ row.date }}</text>
                <text class="status-pill" :class="'status--' + row.tone">{{ row.status }}</text>
              </view>
              <text v-if="row.abnormal" class="check-abnormal">异常：{{ row.abnormal }}</text>
              <text v-if="row.temp" class="check-temp">体温 <text class="check-num">{{ row.temp }}</text></text>
              <text v-if="row.isolated" class="check-isolated">已在隔离室等待家长</text>
              <text v-if="row.remark" class="check-remark">{{ row.remark }}</text>
            </view>
          </view>
        </view>
      </view>

      <view v-if="students.length > 1" class="section">
        <view class="section-head">
          <text class="section-title">已绑定的孩子</text>
          <text class="section-meta">{{ students.length }} 位</text>
        </view>
        <view class="card bound-card">
          <view v-for="student in students" :key="student.id" class="bound-row">
            <view class="bound-main">
              <text class="bound-name">{{ student.name }}</text>
              <text v-if="student.class_name" class="bound-class">{{ student.class_name }}</text>
              <!-- 当前标记只认本地选择：服务端 is_current 是「无 student_id 时的回退孩子」 -->
              <text v-if="student.id === currentStudentId" class="current-pill">当前</text>
            </view>
            <view class="unbind" @click="openUnbind(student)">
              <text class="unbind-text">解绑</text>
            </view>
          </view>
        </view>
      </view>

      <text class="page-foot">档案信息由园所统一维护</text>
    </view>

    <!-- ===== 解绑确认弹层 ===== -->
    <view v-if="dialogStudent" class="dialog-layer" @touchmove.stop.prevent>
      <view class="dialog-mask" @click="closeUnbind" />
      <view class="dialog" aria-role="dialog" aria-modal="true" aria-label="确认解绑">
        <text class="dialog-title">确认解绑</text>
        <text class="dialog-desc">{{ unbindDesc }}</text>
        <view class="dialog-actions">
          <view class="dialog-btn" @click="closeUnbind">
            <text class="dialog-btn-text">取消</text>
          </view>
          <view
            class="dialog-btn dialog-confirm"
            :class="{ 'is-busy': unbinding }"
            :aria-disabled="unbinding ? 'true' : 'false'"
            @click="confirmUnbind"
          >
            <text class="dialog-btn-text dialog-confirm-text">{{ unbinding ? "解绑中…" : "确定" }}</text>
          </view>
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 成长护照：白卡 + 冷雾底；暖色只落在头像底、刷新/重试文字
// 与主按钮。晨检用功能色系，健康备注保持中性。
// ============================================================

.card {
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

// ---- 身份带 --------------------------------------------------
.profile-card {
  padding: $spacing-xl;
}

.identity {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-width: 0;
}

.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: $component-avatar-lg;
  height: $component-avatar-lg;
  overflow: hidden;
  border: 1rpx solid $color-brand-primary-light;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.avatar-img {
  width: 100%;
  height: 100%;
}

.avatar-initial {
  color: $color-brand-primary-dark;
  font-size: 52rpx;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.identity-main {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: $spacing-sm;
  min-width: 0;
}

.identity-name {
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.class-pill {
  display: inline-block;
  padding: 4rpx $component-badge-padding-x;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

// ---- 2×2 档案事实格 ------------------------------------------
// 负边距让分隔线对齐卡片外缘（设计审查 P1-1）：不要加 max-width 抵消出血。
.facts {
  display: flex;
  flex-wrap: wrap;
  margin: $spacing-xl (-$spacing-xl) 0;
  border-top: 1rpx solid $color-border-hairline;
}

.fact {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  box-sizing: border-box;
  width: 50%;
  min-width: 0;
  padding: $spacing-lg $spacing-xl $spacing-xl;
}

.fact.is-right {
  border-left: 1rpx solid $color-border-hairline;
}

.fact.is-bottom {
  border-top: 1rpx solid $color-border-hairline;
}

.fact-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  letter-spacing: 0.04em;
  line-height: 1.3;
}

.fact-value {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.fact-value.is-num {
  font-variant-numeric: tabular-nums;
}

.fact-value.is-empty {
  color: $color-text-hint;
}

// ---- 健康备注（安静的敏感信息） ------------------------------
.health-note {
  display: flex;
  align-items: flex-start;
  gap: $spacing-md;
  margin-top: $spacing-xl;
  padding: $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}

.health-note-icon {
  flex: none;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.health-note-body {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}

.health-note-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  letter-spacing: 0.08em;
  line-height: 1.3;
}

.health-note-text {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

// ---- 局部缺失提示 --------------------------------------------
.partial-hint {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0 $spacing-md;
  margin-top: $spacing-xs;
  padding: 0 $spacing-xs;
}

.partial-hint-text {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.text-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 176rpx;
  min-height: 88rpx;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;

  &:active {
    background: $color-bg-page;
  }
}

.text-action-text {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.text-action--quiet .text-action-text {
  color: $color-text-muted;
}

.text-action.is-busy {
  background: $color-border-hairline;

  &:active {
    background: $color-border-hairline;
  }
}

.text-action.is-busy .text-action-text {
  color: $color-text-hint;
}

// ---- 区块标题 ------------------------------------------------
.section {
  margin-top: calc(#{$spacing-xl} + #{$spacing-md});
}

.section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: $spacing-md;
  padding: 0 $spacing-xs $spacing-md;
}

.section-title {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.section-meta {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
}

// ---- 最近晨检 ------------------------------------------------
.health-card {
  overflow: hidden;
}

.check-row {
  padding: $spacing-lg $spacing-xl;
}

.check-row + .check-row {
  border-top: 1rpx solid $color-border-hairline;
}

.check-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: $spacing-sm $spacing-md;
  min-width: 0;
}

.check-date {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.status-pill {
  display: inline-block;
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

.status--ok {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.status--warn {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}

.status--danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.check-abnormal {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.check-temp {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.check-num {
  font-variant-numeric: tabular-nums;
}

.check-isolated {
  display: block;
  margin-top: $spacing-sm;
  color: $color-semantic-warning;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.check-remark {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.health-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-md;
  padding: calc(#{$spacing-xl} + #{$spacing-lg}) $spacing-xl;
  text-align: center;
}

.health-empty-icon {
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-3xl;
  line-height: 1;
}

.health-empty-text {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.health-inline-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: $spacing-xs $spacing-md;
  min-height: 144rpx;
  padding: $spacing-lg $spacing-xl;
}

.health-inline-title {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.health-inline-actions {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.inline-spinner {
  width: 28rpx;
  height: 28rpx;
  border: 4rpx solid $color-border-hairline;
  border-top-color: $color-text-muted;
  border-radius: $border-radius-full;
  animation: ring-spin 700ms linear infinite;
}

// ---- 已绑定的孩子 --------------------------------------------
.bound-card {
  overflow: hidden;
}

.bound-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  min-height: 128rpx;
  padding: $spacing-sm $spacing-xl;
}

.bound-row + .bound-row {
  border-top: 1rpx solid $color-border-hairline;
}

.bound-main {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: $spacing-sm;
  min-width: 0;
}

.bound-name {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.bound-class {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
}

.current-pill {
  display: inline-block;
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}

.unbind {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  min-width: 112rpx;
  min-height: 88rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;

  &:active {
    background: $color-semantic-danger;

    .unbind-text {
      color: $color-text-on-brand;
    }
  }
}

.unbind-text {
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

// ---- 页脚 ----------------------------------------------------
.page-foot {
  display: block;
  margin-top: calc(#{$spacing-xl} + #{$spacing-lg});
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  letter-spacing: 0.02em;
  line-height: 1.6;
  text-align: center;
}

// ---- 加载骨架 ------------------------------------------------
.sk {
  display: block;
  background: $color-border-hairline;
  border-radius: $border-radius-sm;
  opacity: 0.5;
  animation: sk-breathe 1.5s $animation-easing-default infinite;
}

.sk-identity {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-width: 0;
}

.sk-avatar {
  flex: none;
  width: $component-avatar-lg;
  height: $component-avatar-lg;
  border-radius: $border-radius-full;
}

.sk-identity-lines {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-md;
  min-width: 0;
}

.sk-line {
  height: 20rpx;
}

.sk-line-name {
  width: 44%;
  height: 36rpx;
}

.sk-line-pill {
  width: 24%;
  height: 40rpx;
  border-radius: $border-radius-full;
}

.sk-facts {
  display: flex;
  flex-wrap: wrap;
  margin-top: $spacing-xl;
  border-top: 1rpx solid $color-border-hairline;
}

.sk-fact {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  box-sizing: border-box;
  width: 50%;
  min-width: 0;
  padding: $spacing-lg $spacing-lg $spacing-lg 0;
}

.sk-fact.is-right {
  padding-left: $spacing-lg;
  border-left: 1rpx solid $color-border-hairline;
}

.sk-fact.is-bottom {
  border-top: 1rpx solid $color-border-hairline;
}

.sk-line-label {
  width: 30%;
  height: 18rpx;
}

.sk-line-value {
  width: 68%;
  height: 26rpx;
}

.sk-check-row {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  padding: $spacing-lg $spacing-xl;
}

.sk-check-row + .sk-check-row {
  border-top: 1rpx solid $color-border-hairline;
}

.sk-check-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}

.sk-check-date {
  width: 42%;
  height: 26rpx;
}

.sk-check-pill {
  flex: none;
  width: 108rpx;
  height: 40rpx;
  border-radius: $border-radius-full;
}

.sk-check-line {
  width: 56%;
  height: 20rpx;
}

.sk-bound-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  min-height: 128rpx;
  padding: $spacing-lg $spacing-xl;
}

.sk-bound-row + .sk-bound-row {
  border-top: 1rpx solid $color-border-hairline;
}

.sk-bound-main {
  display: flex;
  align-items: center;
  flex: 1;
  gap: $spacing-sm;
  min-width: 0;
}

.sk-bound-name {
  width: 88rpx;
  height: 26rpx;
}

.sk-bound-class {
  width: 72rpx;
  height: 20rpx;
}

.sk-bound-btn {
  flex: none;
  width: 112rpx;
  height: 56rpx;
  border-radius: $border-radius-full;
}

// ---- 整页失败 ------------------------------------------------
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 72vh;
  padding: $spacing-xl;
  text-align: center;
}

.error-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
}

.error-glyph {
  color: $color-semantic-danger;
  font-family: remixicon;
  font-size: 48rpx;
  line-height: 1;
}

// 「还没有绑定孩子」是中性事实，不用告警红
.error-icon--empty {
  background: $color-bg-page;
}

.error-glyph--empty {
  color: $color-text-muted;
}

.error-title {
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.error-desc {
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: break-word;
}

.primary-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 240rpx;
  min-height: 88rpx;
  margin-top: calc(#{$spacing-xl} + #{$spacing-xs});
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
  box-shadow: $shadow-sm;

  &:active {
    transform: scale(0.98);
  }
}

.primary-button-text {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.primary-button.is-busy {
  background: $color-border-hairline;
  box-shadow: none;

  &:active {
    transform: none;
  }
}

.primary-button.is-busy .primary-button-text {
  color: $color-text-hint;
}

// ---- 解绑确认弹层 --------------------------------------------
.dialog-layer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: $spacing-xl;
}

.dialog-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: $color-text-primary;
  opacity: 0.34;
}

.dialog {
  position: relative;
  width: 100%;
  max-width: 640rpx;
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.dialog-title {
  display: block;
  padding: $spacing-xl $spacing-xl $spacing-md;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  text-align: center;
}

.dialog-desc {
  display: block;
  padding: 0 $spacing-xl $spacing-xl;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.6;
  text-align: center;
  overflow-wrap: break-word;
}

.dialog-actions {
  display: flex;
  border-top: 1rpx solid $color-border-hairline;
}

.dialog-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 96rpx;

  &:active {
    background: $color-bg-page;
  }
}

.dialog-btn + .dialog-btn {
  border-left: 1rpx solid $color-border-hairline;
}

.dialog-btn-text {
  color: $color-text-body;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.dialog-confirm-text {
  color: $color-semantic-danger;
}

.dialog-confirm.is-busy .dialog-confirm-text {
  color: $color-text-hint;
}

@keyframes sk-breathe {
  0%,
  100% {
    opacity: 0.35;
  }
  50% {
    opacity: 0.75;
  }
}

@keyframes ring-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .sk,
  .inline-spinner {
    animation: none;
  }

  .text-action,
  .unbind,
  .primary-button,
  .dialog-btn {
    transition: none;
  }
}
</style>
