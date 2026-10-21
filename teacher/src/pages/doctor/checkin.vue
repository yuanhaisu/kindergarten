<script setup>
import { ref, computed } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import {
  fetchHealthQueue,
  submitCheck,
  uploadHealthPhoto,
  addCheckMedia,
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
  remark: "",
})

const media = ref([])
const uploading = ref({})
const photoCategories = { throat: "喉咙", hand: "手部", mouth: "口腔", eye: "眼睛", skin: "皮肤" }

// 园所体温策略随队列一起下发；默认按需测温，兼容尚未下发 policy 的旧后端。
const policy = ref({ temperature_required: false, fever_threshold: 37.3 })

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
    policy.value = res.data?.policy || { temperature_required: false, fever_threshold: 37.3 }
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
    remark: "",
  }
  media.value = []
  uploading.value = {}
  showCard.value = true
}

function closeCard() {
  showCard.value = false
  currentStudent.value = null
}

// 异常部位拍照：先走统一上传通道拿 file_id，再关联到晨检记录。
// 照片是纠纷与传染病上报的证据，上传失败必须让保健医看到。
async function capturePhoto(field) {
  if (!currentStudent.value || uploading.value[field]) return
  uni.chooseImage({
    count: 1,
    sourceType: ["camera"],
    success: async (res) => {
      const filePath = res.tempFilePaths && res.tempFilePaths[0]
      if (!filePath) return
      uploading.value = { ...uploading.value, [field]: true }
      try {
        const up = await uploadHealthPhoto(filePath)
        const fileId = up.data && up.data.id
        if (!fileId) throw new Error("上传失败")
        const r = await addCheckMedia(currentStudent.value.id, {
          file_id: fileId,
          category: field,
          ref_desc: photoCategories[field] || ""
        })
        media.value = [...media.value, r.data]
        uni.showToast({ title: "照片已保存", icon: "success" })
      } catch (e) {
        uni.showToast({ title: e?.message || "拍照上传失败", icon: "none" })
      } finally {
        uploading.value = { ...uploading.value, [field]: false }
      }
    }
  })
}

async function doSubmit(status) {
  if (submitting.value) return
  if (policy.value.temperature_required && !form.value.temperature) {
    // 服务端也会拒绝；这里先拦一次，避免保健医等一次无谓请求
    uni.showToast({ title: "本园要求录入体温", icon: "none" })
    return
  }
  const t = form.value.temperature ? parseFloat(form.value.temperature) : null
  // symptom_tags 由服务端派生；照片已通过 /media 单独关联。
  const body = {
    temperature: t,
    throat_status: form.value.throat,
    hand_status: form.value.hand,
    mouth_status: form.value.mouth,
    eye_status: form.value.eye,
    skin_status: form.value.skin,
    overall_status: status,
    is_isolated: form.value.is_isolated,
    remark: form.value.remark,
  }
  submitting.value = true
  try {
    const res = await submitCheck(currentStudent.value.id, body)
    const data = res.data || {}
    if ((data.notify_failed_count || 0) > 0) {
      // 通知失败不能静默：保健医需要知道家长没收到
      uni.showModal({
        title: "已记录",
        content: `${data.notify_failed_count} 位家长通知失败，请线下告知`,
        showCancel: false
      })
    } else {
      const label = status === 0 ? "健康入园" : status === 1 ? "需观察" : "已劝返"
      uni.showToast({
        title: (data.notified_count || 0) > 0 ? `${label}·已通知` : label,
        icon: "none"
      })
    }
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

function goGardenCalendar() { uni.navigateTo({ url: "/pages/doctor/calendar" }) }

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

    <!-- 园所体温策略：让保健医开始前就知道要不要逐个录体温。
         加载中/加载失败时不展示，避免在 required 园所短暂显示错误的「按需测温」。 -->
    <view v-if="!loading && !loadError" class="policy-note" :class="{ 'is-required': policy.temperature_required }">
      <text class="policy-text">
        {{ policy.temperature_required ? "本园要求全员测温" : "本园按需测温（体温选填）" }}
      </text>
    </view>

    <!-- 园级事件发布入口（保健医） -->
    <view class="garden-entry" @click="goGardenCalendar">
      <text class="garden-entry-icon">布</text>
      <text class="garden-entry-text">发布园级事件（体检等，全园可见）</text>
      <text class="garden-entry-arrow">▸</text>
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
          <text class="field-label">
            🌡️ 体温 (°C)
            <text class="field-hint" :class="{ 'is-required': policy.temperature_required }">{{ policy.temperature_required ? "必填" : "选填" }}</text>
          </text>
          <input v-model="form.temperature" type="digit" class="temp-input" placeholder="36.5" maxlength="4" />
          <text v-if="form.temperature && parseFloat(form.temperature) >= policy.fever_threshold" class="fever-warn">🔴 发烧</text>
        </view>

        <!-- Check items -->
        <view class="field">
          <text class="field-label">👄 喉咙</text>
          <view class="chip-row">
            <view class="chip" :class="{ on: form.throat === 0 }" @click="form.throat = 0">正常</view>
            <view class="chip" :class="{ abnormal: form.throat === 1 }" @click="toggleItem('throat', 1)">发红</view>
            <view class="chip" :class="{ abnormal: form.throat === 2 }" @click="toggleItem('throat', 2)">红肿</view>
            <view v-if="isAbnormal('throat')" class="photo-btn" @click="capturePhoto('throat')">
              <text class="photo-btn-text">{{ uploading.throat ? "…" : "拍" }}</text>
            </view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">✋ 手部</text>
          <view class="chip-row">
            <view class="chip" :class="{ on: form.hand === 0 }" @click="form.hand = 0">正常</view>
            <view class="chip" :class="{ abnormal: form.hand === 1 }" @click="toggleItem('hand', 1)">红疹</view>
            <view class="chip" :class="{ abnormal: form.hand === 2 }" @click="toggleItem('hand', 2)">疱疹</view>
            <view v-if="isAbnormal('hand')" class="photo-btn" @click="capturePhoto('hand')">
              <text class="photo-btn-text">{{ uploading.hand ? "…" : "拍" }}</text>
            </view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">👅 口腔</text>
          <view class="chip-row">
            <view class="chip" :class="{ on: form.mouth === 0 }" @click="form.mouth = 0">正常</view>
            <view class="chip" :class="{ abnormal: form.mouth === 1 }" @click="toggleItem('mouth', 1)">溃疡</view>
            <view class="chip" :class="{ abnormal: form.mouth === 2 }" @click="toggleItem('mouth', 2)">疱疹</view>
            <view v-if="isAbnormal('mouth')" class="photo-btn" @click="capturePhoto('mouth')">
              <text class="photo-btn-text">{{ uploading.mouth ? "…" : "拍" }}</text>
            </view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">👁️ 眼睛</text>
          <view class="chip-row">
            <view class="chip" :class="{ on: form.eye === 0 }" @click="form.eye = 0">正常</view>
            <view class="chip" :class="{ abnormal: form.eye === 1 }" @click="toggleItem('eye', 1)">发红</view>
            <view class="chip" :class="{ abnormal: form.eye === 2 }" @click="toggleItem('eye', 2)">分泌物</view>
            <view v-if="isAbnormal('eye')" class="photo-btn" @click="capturePhoto('eye')">
              <text class="photo-btn-text">{{ uploading.eye ? "…" : "拍" }}</text>
            </view>
          </view>
        </view>

        <view class="field">
          <text class="field-label">🖐️ 皮肤</text>
          <view class="chip-row">
            <view class="chip" :class="{ on: form.skin === 0 }" @click="form.skin = 0">正常</view>
            <view class="chip" :class="{ abnormal: form.skin === 1 }" @click="toggleItem('skin', 1)">红疹</view>
            <view class="chip" :class="{ abnormal: form.skin === 2 }" @click="toggleItem('skin', 2)">水疱</view>
            <view v-if="isAbnormal('skin')" class="photo-btn" @click="capturePhoto('skin')">
              <text class="photo-btn-text">{{ uploading.skin ? "…" : "拍" }}</text>
            </view>
          </view>
        </view>

        <!-- Remark -->
        <view class="field">
          <text class="field-label">📝 备注</text>
          <textarea v-model="form.remark" class="remark-input" placeholder="异常情况说明..." :maxlength="200" />
        </view>

        <!-- 已关联的异常照片（服务端落库后才显示） -->
        <view v-if="media.length" class="photo-list">
          <image
            v-for="m in media"
            :key="m.id"
            :src="m.thumb_url || m.file_url"
            class="photo-item"
            mode="aspectFill"
          />
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
.policy-note {
  display: flex; align-items: center;
  padding: $spacing-sm $spacing-lg;
  margin-bottom: 20rpx;
  border-radius: $border-radius-md;
  background: $color-semantic-info-bg;
}
.policy-note.is-required { background: $color-semantic-warning-bg; }
.policy-text { font-size: $typography-font-size-sm; color: $color-semantic-info; }
.policy-note.is-required .policy-text { color: $color-semantic-warning; font-weight: 600; }
.garden-entry { display: flex; align-items: center; gap: $spacing-md; padding: $spacing-lg $component-page-padding-x; background: #fff; border-radius: 20rpx; margin-bottom: 20rpx; }
.garden-entry-icon { display: flex; align-items: center; justify-content: center; width: 64rpx; height: 64rpx; border-radius: $border-radius-lg; background: #e0f2fe; color: #3b82f6; font-weight: 700; font-size: $typography-font-size-lg; }
.garden-entry-text { flex: 1; font-size: $typography-font-size-base; color: #333; font-weight: 500; }
.garden-entry-arrow { color: #ccc; font-size: $typography-font-size-sm; }
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
.field-hint { margin-left: $spacing-sm; font-size: $typography-font-size-xs; color: $color-text-hint; }
.field-hint.is-required { color: $color-semantic-danger; font-weight: 600; }
.temp-input { border: 2rpx solid #e5e7eb; border-radius: $border-radius-md; padding: $spacing-md 24rpx; font-size: $typography-font-size-3xl; text-align: center; width: 200rpx; }
.fever-warn { margin-left: 16rpx; font-size: $typography-font-size-md; color: #dc2626; }
.chip-row { display: flex; align-items: center; gap: $spacing-sm; }
.chip { padding: 10rpx 24rpx; border-radius: 999rpx; border: 2rpx solid #e5e7eb; font-size: $typography-font-size-sm; color: #666; transition: all .15s; }
.chip.on { background: #dcfce7; border-color: #22c55e; color: #16a34a; }
.chip.abnormal { background: #fee2e2; border-color: #ef4444; color: #dc2626; }
.photo-btn {
  display: flex; align-items: center; justify-content: center;
  width: 56rpx; height: 56rpx; margin-left: auto;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}
.photo-btn-text { font-size: $typography-font-size-sm; font-weight: 600; color: $color-brand-primary-dark; }
.remark-input { width: 100%; border: 2rpx solid #e5e7eb; border-radius: $border-radius-md; padding: $spacing-md; font-size: $typography-font-size-base; min-height: 100rpx; }
.photo-list { display: flex; flex-wrap: wrap; gap: $spacing-sm; margin-bottom: $spacing-lg; }
.photo-item { width: 120rpx; height: 120rpx; border-radius: $border-radius-md; background: $color-bg-page; }
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
