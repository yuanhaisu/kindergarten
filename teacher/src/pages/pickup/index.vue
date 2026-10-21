<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import {
  fetchTeacherPickupOverview,
  fetchTeacherPickupStudents,
  manualTeacherPickup,
  scanTeacherPickup,
  notifyOverdueParent,
  fetchStudentAuthPersons
} from "@/api/teacher-pickup"

const overview = ref({})
const students = ref([])
const filterStatus = ref("not_picked_up") // not_picked_up | all | arrived | picked_up
const loadError = ref("")

// 手动登记弹窗状态
const showManual = ref(false)
const manualStudent = ref(null)
const manualMode = ref("authorized") // authorized | override
const manualRemark = ref("")
const manualOverrideName = ref("")
const submitting = ref(false)
// 选授权人状态
const authPersons = ref([])
const selectedAuthId = ref(null)
const authPersonsLoading = ref(false)

const todayDate = computed(() => {
  const now = new Date()
  return `${now.getFullYear()}年${now.getMonth() + 1}月${now.getDate()}日`
})

const overdueCount = computed(() => overview.value.overdue_not_picked || 0)

async function loadPickup() {
  loadError.value = ""
  try {
    const [overviewRes, studentsRes] = await Promise.all([
      fetchTeacherPickupOverview(),
      fetchTeacherPickupStudents({ status: filterStatus.value }),
    ])
    overview.value = overviewRes.data || {}
    students.value = studentsRes.data?.list || []
  } catch (err) {
    // 必须显式报错，不能留着空数组：这页空列表的语义是「都接走了」，
    // 请求失败时静默留空，等于告诉老师校内已经没有孩子了。
    loadError.value = err?.message || "接送数据加载失败"
    students.value = []
    overview.value = {}
  } finally {
    // 放进 finally，否则失败时下拉刷新圈永远不会停
    uni.stopPullDownRefresh()
  }
}

// 打开手动登记弹窗
async function openManual(student) {
  manualStudent.value = student
  manualMode.value = "authorized"
  manualRemark.value = ""
  manualOverrideName.value = ""
  selectedAuthId.value = null
  authPersons.value = []
  showManual.value = true
  // 懒加载该生授权人列表
  authPersonsLoading.value = true
  try {
    const res = await fetchStudentAuthPersons(student.student_id)
    authPersons.value = res.data?.list || []
  } catch {
    authPersons.value = []
  } finally {
    authPersonsLoading.value = false
  }
}

// 切换模式：清空选人/例外状态
function switchManualMode(mode) {
  manualMode.value = mode
  selectedAuthId.value = null
  manualOverrideName.value = ""
}

async function submitManual() {
  if (!manualStudent.value) return
  if (manualMode.value === "authorized" && !selectedAuthId.value) {
    uni.showToast({ title: "请先选择一位授权接送人", icon: "none" })
    return
  }
  if (manualMode.value === "override" && !manualOverrideName.value.trim()) {
    uni.showToast({ title: "请填写例外接送人姓名", icon: "none" })
    return
  }
  submitting.value = true
  try {
    await manualTeacherPickup({
      student_id: manualStudent.value.student_id,
      type: 2,
      authorization_id: manualMode.value === "authorized" ? selectedAuthId.value : null,
      is_authorized_override: manualMode.value === "override",
      override_name: manualMode.value === "override" ? manualOverrideName.value.trim() : "",
      remark: manualRemark.value.trim(),
    })
    uni.showToast({ title: "登记成功", icon: "success" })
    showManual.value = false
    loadPickup()
  } catch (err) {
    uni.showToast({ title: mapPickupError(err), icon: "none" })
  } finally {
    submitting.value = false
  }
}

// 送入（入园）一键登记，不走弹窗
async function handleDropOff(student) {
  try {
    await manualTeacherPickup({
      student_id: student.student_id,
      type: 1,
    })
    uni.showToast({ title: `${student.name} 已入园`, icon: "success" })
    loadPickup()
  } catch (err) {
    uni.showToast({ title: mapPickupError(err), icon: "none" })
  }
}

// 后端错误 → 可操作提示（D5）
function mapPickupError(err) {
  const msg = err?.message || "登记失败"
  if (msg.includes("先登记入园") || msg.includes("未到园")) {
    return "该生今日尚未入园，请先登记入园（送入）后再登记接走"
  }
  if (msg.includes("重复接走") && msg.includes("理由")) {
    return "重复接走需填写理由"
  }
  if (msg.includes("不在有效授权")) {
    return "所选接送人授权已失效，请重新选择或标例外"
  }
  return msg
}

function goQrScan() {
  uni.scanCode({
    success: async (res) => {
      try {
        const qrCode = res.result || res.scanCode
        if (!qrCode) {
          uni.showToast({ title: "二维码识别失败", icon: "none" })
          return
        }
        // 只传凭证内容，学生与接走人由后端从码自解释
        const result = await scanTeacherPickup({ qr_code: qrCode })
        const data = result.data
        uni.showToast({
          title: `${data.student?.name || ""} ${data.type_label || "接送"}登记成功`,
          icon: "success",
        })
        loadPickup()
      } catch (err) {
        uni.showToast({ title: err?.message || "扫码登记失败", icon: "none" })
      }
    },
    fail: () => {
      uni.showToast({ title: "扫码已取消", icon: "none" })
    },
  })
}

// 一键通知滞留儿童家长
async function handleNotifyOverdue(student) {
  try {
    await notifyOverdueParent(student.student_id)
    uni.showToast({ title: `已通知 ${student.name} 家长`, icon: "success" })
  } catch (err) {
    uni.showToast({ title: err?.message || "通知失败", icon: "none" })
  }
}

function goManualRegister() {
  filterStatus.value = "all"
  loadPickup()
}

function goRecords() {
  uni.navigateTo({ url: "/pages/pickup/records" })
}

const filterOptions = [
  { key: "not_picked_up", label: "未接走" },
  { key: "all", label: "全部" },
  { key: "arrived", label: "已入园" },
  { key: "picked_up", label: "已接走" },
]
function toggleFilter() {
  const idx = filterOptions.findIndex((o) => o.key === filterStatus.value)
  filterStatus.value = filterOptions[(idx + 1) % filterOptions.length].key
  loadPickup()
}

// 标题从 filterStatus 派生，不再维护第二个状态变量 —— 之前 goManualRegister()
// 只改 filterStatus 不改索引，标题就和列表对不上了。
const studentSectionTitle = computed(() => {
  const opt = filterOptions.find((o) => o.key === filterStatus.value) || filterOptions[0]
  return `${opt.label}学生（${students.value.length}人）`
})

onLoad(loadPickup)
onPullDownRefresh(loadPickup)
</script>

<template>
  <view class="page">
    <!-- 今日接送统计卡片 -->
    <view class="stats-card">
      <view class="stats-head">
        <text class="stats-title">今日接送</text>
        <text class="stats-date">{{ todayDate }}</text>
      </view>
      <view class="stats-grid">
        <view class="stat-item">
          <view class="stat-icon-box is-green">
            <text class="stat-icon-text">入</text>
          </view>
          <view class="stat-text">
            <text class="stat-num">{{ overview.arrived || 0 }}</text>
            <text class="stat-label">已入园</text>
          </view>
        </view>
        <view class="stat-item">
          <view class="stat-icon-box is-blue">
            <text class="stat-icon-text">接</text>
          </view>
          <view class="stat-text">
            <text class="stat-num">{{ overview.picked_up || 0 }}</text>
            <text class="stat-label">已接走</text>
          </view>
        </view>
        <view class="stat-item">
          <view class="stat-icon-box is-amber">
            <text class="stat-icon-text">待</text>
          </view>
          <view class="stat-text">
            <text class="stat-num">{{ overview.not_picked || (overview.total - overview.picked_up) || 0 }}</text>
            <text class="stat-label">未接走</text>
          </view>
        </view>
        <view class="stat-item">
          <view class="stat-icon-box is-danger">
            <text class="stat-icon-text">异</text>
          </view>
          <view class="stat-text">
            <text class="stat-num">{{ overview.abnormal || 0 }}</text>
            <text class="stat-label">异常</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 功能操作区 -->
    <view class="action-card">
      <view class="action-grid">
        <view class="action-item" @click="goQrScan">
          <view class="action-icon-box is-green">
            <text class="action-icon-text">扫</text>
          </view>
          <text class="action-label">扫码接送</text>
        </view>
        <view class="action-item" @click="goManualRegister">
          <view class="action-icon-box is-blue">
            <text class="action-icon-text">记</text>
          </view>
          <text class="action-label">手动登记</text>
        </view>
        <view class="action-item" @click="goRecords">
          <view class="action-icon-box is-purple">
            <text class="action-icon-text">录</text>
          </view>
          <text class="action-label">接送记录</text>
        </view>
      </view>
      <view class="action-hint">
        <text class="hint-icon">提</text>
        <text class="hint-text">点击「扫码接送」，扫描家长出示的接送二维码即可完成登记，操作仅需3秒</text>
      </view>
    </view>

    <!-- 学生列表 -->
    <view class="student-section">
      <view class="student-head">
        <text class="student-title">{{ studentSectionTitle }}</text>
        <view class="filter-btn" @click="toggleFilter">
          <text>筛选</text>
        </view>
      </view>

      <!-- 错误态必须排在空态之前：否则请求失败会被当成「没有学生」 -->
      <view v-if="loadError" class="empty-state">
        <text class="empty-text">{{ loadError }}</text>
        <view class="retry-btn" @click="loadPickup">重试</view>
      </view>
      <view v-else-if="students.length === 0" class="empty-state">
        <text class="empty-text">当前筛选下没有学生</text>
      </view>
      <view v-else class="student-list">
        <view
          v-for="student in students"
          :key="student.student_id"
          :class="['student-card', student.is_overdue ? 'card-overdue' : '']"
        >
          <view class="student-info">
            <view class="student-avatar">
              <image
                v-if="student.avatar"
                :src="student.avatar"
                mode="aspectFill"
                class="avatar-img"
              />
              <text v-else class="avatar-placeholder">{{ student.name?.[0] || "生" }}</text>
            </view>
            <view class="student-text">
              <view class="name-row">
                <text class="student-name">{{ student.name }}</text>
                <text v-if="student.is_overdue" class="overdue-tag">滞留</text>
                <text v-if="student.is_authorized_override" class="override-tag">例外</text>
              </view>
              <text class="student-auth">
                授权接送人：{{ student.authorized_persons || "暂无" }}
              </text>
              <text v-if="student.is_overdue && student.pickup_deadline" class="overdue-hint">
                放学时间 {{ student.pickup_deadline }}，请尽快联系家长
              </text>
            </view>
          </view>
          <view class="action-col">
            <view
              v-if="student.pickup_status === 'not_arrived'"
              class="dropoff-btn"
              @click="handleDropOff(student)"
            >
              <text>入园</text>
            </view>
            <view class="register-btn" @click="openManual(student)">
              <text>{{ student.pickup_status === 'picked_up' ? '再接走' : '接走' }}</text>
            </view>
            <view
              v-if="student.is_overdue"
              class="notify-btn"
              @click="handleNotifyOverdue(student)"
            >
              <text>通知</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <!-- 手动登记弹窗 -->
    <view v-if="showManual" class="modal-mask" @click="showManual = false">
      <view class="modal-box" @click.stop>
        <text class="modal-title">手动登记 — {{ manualStudent?.name }}</text>

        <!-- 二选一：选授权人 / 标例外 -->
        <view class="mode-switch">
          <view
            :class="['mode-btn', manualMode === 'authorized' ? 'mode-active' : '']"
            @click="switchManualMode('authorized')"
          >
            <text>选授权人</text>
          </view>
          <view
            :class="['mode-btn', manualMode === 'override' ? 'mode-active' : '']"
            @click="switchManualMode('override')"
          >
            <text>标例外</text>
          </view>
        </view>

        <!-- 选授权人：radio 卡片列表 -->
        <view v-if="manualMode === 'authorized'" class="auth-list">
          <view v-if="authPersonsLoading" class="auth-loading">
            <text>加载授权人...</text>
          </view>
          <view v-else-if="authPersons.length === 0" class="auth-empty">
            <text>暂无有效授权接送人</text>
            <text class="auth-empty-hint">可切换到「标例外」模式登记</text>
          </view>
          <view v-else class="auth-cards">
            <view
              v-for="p in authPersons"
              :key="p.id"
              :class="['auth-card', selectedAuthId === p.id ? 'auth-card-selected' : '']"
              @click="selectedAuthId = p.id"
            >
              <view class="auth-radio">
                <view v-if="selectedAuthId === p.id" class="auth-radio-dot" />
              </view>
              <view class="auth-avatar">
                <image v-if="p.avatar" :src="p.avatar" class="auth-avatar-img" mode="aspectFill" />
                <text v-else class="auth-avatar-placeholder">{{ p.name?.[0] || "人" }}</text>
              </view>
              <view class="auth-info">
                <text class="auth-name">{{ p.name }}</text>
                <text class="auth-relation">{{ p.relation }} · {{ p.phone }}</text>
              </view>
            </view>
          </view>
        </view>

        <!-- 例外模式：填写来接者姓名 -->
        <view v-if="manualMode === 'override'" class="form-row">
          <text class="form-label">来接者姓名</text>
          <input
            v-model="manualOverrideName"
            class="form-input"
            placeholder="如：邻居王五"
            maxlength="50"
          />
        </view>

        <!-- 理由（重复接走/提前接走必填，此处统一提供） -->
        <view class="form-row">
          <text class="form-label">理由</text>
          <textarea
            v-model="manualRemark"
            class="form-textarea"
            placeholder="重复接走/提前接走时必填"
            maxlength="200"
          />
        </view>

        <view class="modal-actions">
          <view class="modal-btn cancel" @click="showManual = false">
            <text>取消</text>
          </view>
          <view class="modal-btn confirm" @click="submitManual">
            <text>{{ submitting ? "提交中..." : "确认登记" }}</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 底部留白 -->
    <view class="bottom-safe" />
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: $spacing-lg $component-page-padding-x;
  background: $color-bg-page;
}

// ===== 统计卡片（紧凑风格：白底 + 语义色数字） =====
.stats-card {
  padding: $component-card-padding;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  margin-bottom: $spacing-lg;
}

.stats-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: $spacing-lg;
}

.stats-title {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
}

.stats-date {
  font-size: $typography-font-size-sm;
  color: $color-text-hint;
}

.stats-grid {
  display: flex;
  justify-content: space-between;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.stat-icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  border-radius: $border-radius-md;
  flex-shrink: 0;
}

.stat-icon-box.is-green  { background: $color-semantic-success-bg; }
.stat-icon-box.is-blue   { background: $color-semantic-info-bg; }
.stat-icon-box.is-amber  { background: $color-semantic-warning-bg; }
.stat-icon-box.is-danger { background: $color-semantic-danger-bg; }

.stat-icon-text {
  font-size: $typography-font-size-sm;
  font-weight: 600;
}

.stat-icon-box.is-green  .stat-icon-text { color: $color-semantic-success; }
.stat-icon-box.is-blue   .stat-icon-text { color: $color-semantic-info; }
.stat-icon-box.is-amber  .stat-icon-text { color: $color-semantic-warning; }
.stat-icon-box.is-danger .stat-icon-text { color: $color-semantic-danger; }

.stat-text {
  display: flex;
  flex-direction: column;
  gap: 2rpx;
}

.stat-num {
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
  line-height: 1.2;
}

.stat-label {
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
  line-height: 1.2;
}

// ===== 操作区 =====
.action-card {
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  margin-bottom: $spacing-lg;
}

.action-grid {
  display: flex;
  justify-content: space-around;
  margin-bottom: $spacing-xl;
}

.action-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18rpx;
}

.action-icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-xl;
}

.is-green  { background: $color-semantic-success-bg; }
.is-blue   { background: $color-semantic-info-bg; }
.is-purple { background: #F3E8FF; }

.action-icon-text {
  font-size: $typography-font-size-2xl;
  font-weight: 700;
}

.is-green  .action-icon-text { color: $color-semantic-success; }
.is-blue   .action-icon-text { color: $color-semantic-info; }
.is-purple .action-icon-text { color: #9333EA; }

.action-label {
  font-size: $typography-font-size-base;
  font-weight: 500;
  color: $color-text-body;
}

.action-hint {
  display: flex;
  align-items: flex-start;
  gap: $spacing-lg;
  padding: $spacing-lg;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
}

.hint-icon {
  flex-shrink: 0;
  font-size: $typography-font-size-md;
  font-weight: 700;
  color: $color-semantic-info;
}

.hint-text {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
  line-height: 1.5;
}

// ===== 学生列表 =====
.student-section {
  margin-bottom: $spacing-lg;
}

.student-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: $spacing-lg;
}

.student-title {
  font-size: $typography-font-size-xl;
  font-weight: 700;
  color: $color-text-primary;
}

.filter-btn {
  display: flex;
  align-items: center;
  gap: 4rpx;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-full;

  text {
    font-size: $typography-font-size-md;
    color: $color-semantic-info;
  }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-lg;
  padding: 80rpx 0;
  text-align: center;
}

.empty-text {
  font-size: $typography-font-size-md;
  color: $color-text-hint;
}

.retry-btn {
  padding: $spacing-md $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-semantic-info;
  font-size: $typography-font-size-base;
  color: #fff;
}

.student-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.student-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.student-info {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  flex: 1;
  min-width: 0;
}

.student-avatar {
  flex-shrink: 0;
  width: 96rpx;
  height: 96rpx;
  border-radius: $border-radius-full;
  overflow: hidden;
}

.avatar-img {
  width: 100%;
  height: 100%;
}

.avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: $color-semantic-info-bg;
  font-size: $typography-font-size-2xl;
  font-weight: 700;
  color: $color-semantic-info;
}

.student-text {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
}

.student-name {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
}

.student-auth {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.register-btn {
  flex-shrink: 0;
  padding: $spacing-md $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-semantic-info-bg;

  text {
    font-size: $typography-font-size-base;
    font-weight: 500;
    color: $color-semantic-info;
  }
}

.bottom-safe {
  height: calc(60rpx + env(safe-area-inset-bottom));
}

// ===== 滞留高亮 =====
.student-card.card-overdue {
  border: 2rpx solid $color-semantic-danger;
  background: $color-semantic-danger-bg;
}

.name-row {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.overdue-tag {
  padding: 2rpx 12rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
  color: #fff;
  font-size: $typography-font-size-xs;
  font-weight: 600;
}

.override-tag {
  padding: 2rpx 12rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
  font-size: $typography-font-size-xs;
  font-weight: 600;
}

.overdue-hint {
  font-size: $typography-font-size-xs;
  color: $color-semantic-danger;
  margin-top: 2rpx;
}

.action-col {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  flex-shrink: 0;
}

.notify-btn {
  padding: $spacing-md $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
  text-align: center;

  text {
    font-size: $typography-font-size-base;
    font-weight: 500;
    color: #fff;
  }
}

// ===== 手动登记弹窗 =====
.modal-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-box {
  width: 640rpx;
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
}

.modal-title {
  font-size: $typography-font-size-lg;
  font-weight: 700;
  color: $color-text-primary;
  margin-bottom: $spacing-lg;
}

.mode-switch {
  display: flex;
  gap: $spacing-md;
  margin-bottom: $spacing-lg;
}

.mode-btn {
  flex: 1;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  text-align: center;

  text {
    font-size: $typography-font-size-base;
    color: $color-text-muted;
  }
}

.mode-btn.mode-active {
  background: $color-semantic-info;

  text {
    color: #fff;
    font-weight: 600;
  }
}

.form-row {
  margin-bottom: $spacing-lg;
}

.form-label {
  display: block;
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
  margin-bottom: $spacing-xs;
}

.form-input {
  width: 100%;
  height: $component-input-height;
  padding: 0 $spacing-md;
  border-radius: $component-input-radius;
  background: $color-bg-page;
  font-size: $typography-font-size-base;
  box-sizing: border-box;
}

.form-textarea {
  width: 100%;
  height: 140rpx;
  padding: $spacing-md;
  border-radius: $component-input-radius;
  background: $color-bg-page;
  font-size: $typography-font-size-base;
  box-sizing: border-box;
}

.modal-actions {
  display: flex;
  gap: $spacing-md;
  margin-top: $spacing-xl;
}

.modal-btn {
  flex: 1;
  padding: $spacing-md $spacing-lg;
  border-radius: $border-radius-full;
  text-align: center;

  text {
    font-size: $typography-font-size-base;
    font-weight: 600;
  }
}

.modal-btn.cancel {
  background: $color-bg-page;

  text { color: $color-text-muted; }
}

.modal-btn.confirm {
  background: $color-brand-primary;

  text { color: #fff; }
}

// ===== 选授权人列表 =====
.auth-list {
  margin-bottom: $spacing-lg;
}

.auth-loading {
  padding: $spacing-xl;
  text-align: center;

  text {
    font-size: $typography-font-size-sm;
    color: $color-text-hint;
  }
}

.auth-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-xs;
  padding: $spacing-xl;

  text {
    font-size: $typography-font-size-sm;
    color: $color-text-muted;
  }
}

.auth-empty-hint {
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.auth-cards {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  max-height: 400rpx;
  overflow-y: auto;
}

.auth-card {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  padding: $spacing-md;
  border-radius: $border-radius-md;
  background: $color-bg-page;
  border: 2rpx solid transparent;
}

.auth-card-selected {
  border-color: $color-brand-primary;
  background: rgba(249, 115, 22, 0.05);
}

.auth-radio {
  width: 36rpx;
  height: 36rpx;
  border-radius: 9999rpx;
  border: 3rpx solid $color-text-hint;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.auth-card-selected .auth-radio {
  border-color: $color-brand-primary;
}

.auth-radio-dot {
  width: 20rpx;
  height: 20rpx;
  border-radius: 9999rpx;
  background: $color-brand-primary;
}

.auth-avatar {
  width: 72rpx;
  height: 72rpx;
  border-radius: 9999rpx;
  overflow: hidden;
  background: $color-semantic-info-bg;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.auth-avatar-img {
  width: 100%;
  height: 100%;
}

.auth-avatar-placeholder {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-semantic-info;
}

.auth-info {
  display: flex;
  flex-direction: column;
  gap: 2rpx;
  min-width: 0;
  flex: 1;
}

.auth-name {
  font-size: $typography-font-size-base;
  font-weight: 600;
  color: $color-text-primary;
}

.auth-relation {
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

// ===== 送入按钮 =====
.dropoff-btn {
  padding: $spacing-md $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-semantic-success-bg;
  text-align: center;

  text {
    font-size: $typography-font-size-base;
    font-weight: 500;
    color: $color-semantic-success;
  }
}
</style>
