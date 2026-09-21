<script setup>
import { ref, computed } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import {
  fetchHealthQueue,
  submitCheck,
} from "@/api/doctor-health"

function todayStr() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`
}

const date = ref(todayStr())
const queue = ref([])
const loading = ref(false)
const loadError = ref("")
const submitting = ref(false)
const selectedClass = ref(0) // 0 = all

// Check card state
const showCard = ref(false)
const currentStudent = ref(null)
const form = ref({
  temperature: "",
  throat: 0, hand: 0, mouth: 0, eye: 0, skin: 0,
  overall: 0,
  is_isolated: false,
  symptom_tags: "",
  remark: "",
})

const pendingList = computed(() => queue.value.filter(q => q.check_status === 0))
const doneList = computed(() => queue.value.filter(q => q.check_status !== 0))

const stats = computed(() => {
  const total = queue.value.length
  const done = doneList.value.length
  return { total, done, pending: total - done }
})

const classList = computed(() => {
  const seen = new Map()
  queue.value.forEach(q => {
    if (!seen.has(q.class_id)) seen.set(q.class_id, q.class_name)
  })
  return Array.from(seen, ([id, name]) => ({ id, name }))
})

async function loadQueue() {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const params = { date: date.value }
    if (selectedClass.value > 0) params.class_id = selectedClass.value
    const res = await fetchHealthQueue(params)
    queue.value = res.data?.list || []
  } catch (err) {
    // 必须记错误态：队列为空在这页的含义是「全部检查完毕」，
    // 请求失败时静默留空等于告诉保健医今天的孩子都查过了。
    // 老师角色调用还会撞 403（Casbin 未给 teacher 开放 /doctor/*）。
    loadError.value = err?.message || "晨检数据加载失败"
    queue.value = []
  } finally {
    loading.value = false
  }
}

function openCheck(student) {
  currentStudent.value = student
  form.value = {
    temperature: "",
    throat: 0, hand: 0, mouth: 0, eye: 0, skin: 0,
    overall: 0,
    is_isolated: false,
    symptom_tags: "",
    remark: "",
  }
  showCard.value = true
}

function closeCard() {
  showCard.value = false
  currentStudent.value = null
}

// Build symptom tags from check items
function buildSymptomTags() {
  const tags = []
  const t = parseFloat(form.value.temperature)
  if (!isNaN(t) && t >= 37.3) tags.push("fever")
  if (form.value.throat === 1) tags.push("red_throat")
  if (form.value.throat === 2) tags.push("swollen_throat")
  if (form.value.hand === 1) tags.push("hand_rash")
  if (form.value.hand === 2) tags.push("hand_blister")
  if (form.value.mouth === 1) tags.push("mouth_ulcer")
  if (form.value.mouth === 2) tags.push("mouth_blister")
  if (form.value.eye === 1) tags.push("red_eye")
  if (form.value.eye === 2) tags.push("eye_discharge")
  if (form.value.skin === 1) tags.push("skin_rash")
  if (form.value.skin === 2) tags.push("skin_blister")
  return tags.join(",")
}

async function doSubmit(status) {
  if (submitting.value) return
  const t = form.value.temperature ? parseFloat(form.value.temperature) : null
  const body = {
    temperature: t,
    throat_status: form.value.throat,
    hand_status: form.value.hand,
    mouth_status: form.value.mouth,
    eye_status: form.value.eye,
    skin_status: form.value.skin,
    overall_status: status,
    is_isolated: form.value.is_isolated,
    symptom_tags: buildSymptomTags(),
    remark: form.value.remark,
    media_ids: [],
  }
  submitting.value = true
  try {
    await submitCheck(currentStudent.value.id, body)
    uni.showToast({ title: status === 0 ? "✅ 健康入园" : status === 1 ? "⚠️ 需观察" : "🚫 已劝返", icon: "none" })
    closeCard()
    await loadQueue()
  } catch (e) {
    uni.showToast({ title: e?.message || "提交失败", icon: "none" })
  } finally { submitting.value = false }
}

function healthSubmit() { doSubmit(0) }
function observeSubmit() { doSubmit(1) }
function rejectSubmit() { doSubmit(2) }

function toggleItem(field, val) {
  form.value[field] = form.value[field] === val ? 0 : val
}

function isAbnormal(field) { return form.value[field] !== 0 }

onLoad(() => { loadQueue() })
</script>

<template>
  <view class="page">
    <!-- Stats bar -->
    <view class="stats-bar">
      <view class="stat"><text class="stat-num">{{ stats.total }}</text><text class="stat-label">总入园</text></view>
      <view class="stat"><text class="stat-num accent">{{ stats.pending }}</text><text class="stat-label">待检查</text></view>
      <view class="stat"><text class="stat-num done">{{ stats.done }}</text><text class="stat-label">已完成</text></view>
    </view>

    <!-- Class filter -->
    <scroll-view scroll-x class="filter-bar">
      <view
        class="filter-chip" :class="{ active: selectedClass === 0 }"
        @click="selectedClass = 0; loadQueue()"
      >全部</view>
      <view
        v-for="c in classList" :key="c.id"
        class="filter-chip" :class="{ active: selectedClass === c.id }"
        @click="selectedClass = c.id; loadQueue()"
      >{{ c.name }}</view>
    </scroll-view>

    <!-- Pending list -->
    <view class="section-title">⏳ 待检查 ({{ pendingList.length }})</view>
    <view v-if="loading" class="state">加载中...</view>
    <!-- 错误态必须排在空态之前：否则加载失败会被当成「今天都查完了」 -->
    <view v-else-if="loadError" class="state">
      <text class="state-text">{{ loadError }}</text>
      <view class="retry-btn" @click="loadQueue">重试</view>
    </view>
    <view v-else-if="pendingList.length === 0" class="state">全部检查完毕 ✅</view>
    <view v-else class="list">
      <view v-for="s in pendingList" :key="s.id" class="student-row" @click="openCheck(s)">
        <image v-if="s.avatar" :src="s.avatar" class="avatar" mode="aspectFill" />
        <view v-else class="avatar-empty">{{ s.student_name[0] }}</view>
        <view class="info">
          <text class="name">{{ s.student_name }}</text>
          <text class="meta">{{ s.class_name }} · {{ s.pickup_time }}入园</text>
        </view>
        <text class="arrow">▶</text>
      </view>
    </view>

    <!-- Done list (collapsed summary) -->
    <view v-if="doneList.length > 0" class="section-title">✅ 已完成 ({{ doneList.length }})</view>
    <view v-if="doneList.length > 0" class="list done-list">
      <view v-for="s in doneList" :key="s.id" class="student-row done">
        <image v-if="s.avatar" :src="s.avatar" class="avatar small" mode="aspectFill" />
        <view v-else class="avatar-empty small">{{ s.student_name[0] }}</view>
        <view class="info">
          <text class="name">{{ s.student_name }}</text>
          <text class="meta">{{ s.class_name }}</text>
        </view>
        <text class="status-tag" :class="s.overall_status === 0 ? 'healthy' : s.overall_status === 1 ? 'observe' : 'reject'">
          {{ s.overall_status === 0 ? '✅健康' : s.overall_status === 1 ? '⚠️观察' : '🚫劝返' }}
        </text>
      </view>
    </view>

    <!-- Check card (modal) -->
    <view v-if="showCard" class="modal-mask" @click="closeCard">
      <view class="card" @click.stop>
        <view class="card-header">
          <text class="card-title">{{ currentStudent?.student_name }}</text>
          <text class="card-sub">{{ currentStudent?.class_name }} · {{ currentStudent?.pickup_time }}入园</text>
          <view class="close-btn" @click="closeCard">✕</view>
        </view>

        <!-- Temperature -->
        <view class="field">
          <text class="field-label">🌡️ 体温 (°C)</text>
          <input v-model="form.temperature" type="digit" class="temp-input" placeholder="36.5" maxlength="4" />
          <text v-if="form.temperature && parseFloat(form.temperature) >= 37.3" class="fever-warn">🔴 发烧</text>
        </view>

        <!-- Check items -->
        <view class="field">
          <text class="field-label">👄 喉咙</text>
          <view class="chip-row">
            <view class="chip" :class="{ on: form.throat === 0 }" @click="form.throat = 0">正常</view>
            <view class="chip" :class="{ abnormal: form.throat === 1 }" @click="toggleItem('throat', 1)">发红</view>
            <view class="chip" :class="{ abnormal: form.throat === 2 }" @click="toggleItem('throat', 2)">红肿</view>
            <text v-if="isAbnormal('throat')" class="camera-hint" @click="uni.chooseImage({ count:1, sourceType:['camera'], success(r){} })">📷</text>
          </view>
        </view>

        <view class="field">
          <text class="field-label">✋ 手部</text>
          <view class="chip-row">
            <view class="chip" :class="{ on: form.hand === 0 }" @click="form.hand = 0">正常</view>
            <view class="chip" :class="{ abnormal: form.hand === 1 }" @click="toggleItem('hand', 1)">红疹</view>
            <view class="chip" :class="{ abnormal: form.hand === 2 }" @click="toggleItem('hand', 2)">疱疹</view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">👅 口腔</text>
          <view class="chip-row">
            <view class="chip" :class="{ on: form.mouth === 0 }" @click="form.mouth = 0">正常</view>
            <view class="chip" :class="{ abnormal: form.mouth === 1 }" @click="toggleItem('mouth', 1)">溃疡</view>
            <view class="chip" :class="{ abnormal: form.mouth === 2 }" @click="toggleItem('mouth', 2)">疱疹</view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">👁️ 眼睛</text>
          <view class="chip-row">
            <view class="chip" :class="{ on: form.eye === 0 }" @click="form.eye = 0">正常</view>
            <view class="chip" :class="{ abnormal: form.eye === 1 }" @click="toggleItem('eye', 1)">发红</view>
            <view class="chip" :class="{ abnormal: form.eye === 2 }" @click="toggleItem('eye', 2)">分泌物</view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">🖐️ 皮肤</text>
          <view class="chip-row">
            <view class="chip" :class="{ on: form.skin === 0 }" @click="form.skin = 0">正常</view>
            <view class="chip" :class="{ abnormal: form.skin === 1 }" @click="toggleItem('skin', 1)">红疹</view>
            <view class="chip" :class="{ abnormal: form.skin === 2 }" @click="toggleItem('skin', 2)">水疱</view>
          </view>
        </view>

        <!-- Remark -->
        <view class="field">
          <text class="field-label">📝 备注</text>
          <textarea v-model="form.remark" class="remark-input" placeholder="异常情况说明..." :maxlength="200" />
        </view>

        <!-- Action buttons -->
        <view class="actions">
          <button class="btn healthy" :loading="submitting" @click="healthSubmit">✅ 健康入园</button>
          <view class="row-actions">
            <button class="btn observe" :loading="submitting" @click="observeSubmit">⚠️ 需观察</button>
            <button class="btn reject" :loading="submitting" @click="rejectSubmit">🚫 劝返就医</button>
          </view>
          <view class="isolate-row" @click="form.is_isolated = !form.is_isolated">
            <view class="checkbox" :class="{ checked: form.is_isolated }">{{ form.is_isolated ? '☑' : '☐' }}</view>
            <text>🏥 已带到隔离室等待家长</text>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg 28rpx 120rpx; background: #f5f5f5; }
.stats-bar { display: flex; justify-content: space-around; padding: 28rpx 0; background: #fff; border-radius: 20rpx; margin-bottom: 20rpx; }
.stat { text-align: center; }
.stat-num { display: block; font-size: 44rpx; font-weight: 800; color: #333; }
.stat-num.accent { color: #f59e0b; }
.stat-num.done { color: #22c55e; }
.stat-label { font-size: $typography-font-size-sm; color: #999; }
.filter-bar { white-space: nowrap; padding: 0 0 16rpx; }
.filter-chip { display: inline-block; padding: 10rpx 28rpx; margin-right: 16rpx; border-radius: 999rpx; background: #fff; font-size: $typography-font-size-base; color: #666; }
.filter-chip.active { background: #3b82f6; color: #fff; }
.section-title { font-size: $typography-font-size-md; font-weight: 700; color: #333; margin: 20rpx 0 12rpx; }
.state { display: flex; flex-direction: column; align-items: center; gap: $spacing-lg; text-align: center; padding: 120rpx 0; color: #999; font-size: $typography-font-size-md; }
.state-text { font-size: $typography-font-size-md; color: $color-text-hint; }
.retry-btn { padding: $spacing-sm $spacing-xl; border-radius: $border-radius-full; background: $color-semantic-info; color: #fff; font-size: $typography-font-size-base; }
.list { display: flex; flex-direction: column; gap: $spacing-sm; }
.student-row { display: flex; align-items: center; padding: 20rpx; background: #fff; border-radius: $border-radius-lg; }
.student-row.done { opacity: 0.7; }
.avatar, .avatar-empty { width: 72rpx; height: 72rpx; border-radius: 50%; margin-right: 20rpx; }
.avatar-empty { display: flex; align-items: center; justify-content: center; background: #e0f2fe; color: #3b82f6; font-size: $typography-font-size-xl; font-weight: 700; }
.avatar-empty.small, .avatar.small { width: 56rpx; height: 56rpx; font-size: $typography-font-size-sm; }
.info { flex: 1; }
.name { display: block; font-size: $typography-font-size-md; font-weight: 600; color: #333; }
.meta { font-size: $typography-font-size-xs; color: #999; }
.arrow { color: #ccc; font-size: $typography-font-size-sm; }
.status-tag { font-size: $typography-font-size-xs; padding: 4rpx 12rpx; border-radius: 999rpx; }
.status-tag.healthy { background: #dcfce7; color: #16a34a; }
.status-tag.observe { background: #fef9c3; color: #ca8a04; }
.status-tag.reject { background: #fee2e2; color: #dc2626; }

/* Check card */
.modal-mask { position: fixed; inset: 0; background: rgba(0,0,0,0.5); z-index: 999; display: flex; align-items: flex-end; }
.card { width: 100%; max-height: 90vh; background: #fff; border-radius: $border-radius-xl 32rpx 0 0; padding: $spacing-xl; overflow-y: auto; }
.card-header { position: relative; margin-bottom: 24rpx; }
.card-title { font-size: $typography-font-size-2xl; font-weight: 800; display: block; }
.card-sub { font-size: $typography-font-size-sm; color: #999; }
.close-btn { position: absolute; right: 0; top: 0; width: 48rpx; height: 48rpx; display: flex; align-items: center; justify-content: center; font-size: $typography-font-size-xl; color: #999; }
.field { margin-bottom: 20rpx; }
.field-label { display: block; font-size: $typography-font-size-base; color: #666; margin-bottom: 8rpx; }
.temp-input { border: 2rpx solid #e5e7eb; border-radius: $border-radius-md; padding: $spacing-md 24rpx; font-size: $typography-font-size-3xl; text-align: center; width: 200rpx; }
.fever-warn { margin-left: 16rpx; font-size: $typography-font-size-md; color: #dc2626; }
.chip-row { display: flex; align-items: center; gap: $spacing-sm; }
.chip { padding: 10rpx 24rpx; border-radius: 999rpx; border: 2rpx solid #e5e7eb; font-size: $typography-font-size-sm; color: #666; transition: all .15s; }
.chip.on { background: #dcfce7; border-color: #22c55e; color: #16a34a; }
.chip.abnormal { background: #fee2e2; border-color: #ef4444; color: #dc2626; }
.camera-hint { font-size: $typography-font-size-xl; }
.remark-input { width: 100%; border: 2rpx solid #e5e7eb; border-radius: $border-radius-md; padding: $spacing-md; font-size: $typography-font-size-base; min-height: 100rpx; }
.actions { margin-top: 28rpx; }
.btn { width: 100%; padding: 22rpx; border-radius: $border-radius-lg; font-size: $typography-font-size-lg; font-weight: 700; text-align: center; margin-bottom: 12rpx; border: none; }
.btn.healthy { background: #22c55e; color: #fff; }
.row-actions { display: flex; gap: $spacing-sm; }
.row-actions .btn { flex: 1; }
.btn.observe { background: #f59e0b; color: #fff; }
.btn.reject { background: #ef4444; color: #fff; }
.isolate-row { display: flex; align-items: center; gap: $spacing-sm; padding: $spacing-md 0; font-size: $typography-font-size-base; color: #666; }
.checkbox { font-size: $typography-font-size-2xl; }
</style>
