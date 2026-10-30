<script setup>
import { computed, reactive, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import {
  fetchHealthQueue,
  submitCheck,
  uploadHealthPhoto,
  addCheckMedia
} from "@/api/doctor-health"
import { formatDateLabel, formatWeekdayShort, todayKey } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// 五项检查的三态取值：0 正常 / 1、2 异常。手部/口腔/皮肤的 3=其他异常
// 在本次重做里没有对应 chip（设计稿只有三态），提交值因此只可能是 0–2。
const CHECKS = [
  { key: "throat_status", label: "喉咙", category: "throat", options: ["正常", "发红", "红肿"] },
  { key: "hand_status", label: "手部", category: "hand", options: ["正常", "红疹", "疱疹"] },
  { key: "mouth_status", label: "口腔", category: "mouth", options: ["正常", "溃疡", "疱疹"] },
  { key: "eye_status", label: "眼睛", category: "eye", options: ["正常", "发红", "分泌物"] },
  { key: "skin_status", label: "皮肤", category: "skin", options: ["正常", "红疹", "水疱"] }
]
const CHECK_BY_KEY = {}
CHECKS.forEach((item) => {
  CHECK_BY_KEY[item.key] = item
})

const date = ref(todayKey())
const queue = ref([])
const loading = ref(true)
const loadError = ref("")
const policy = ref({ temperature_required: false, fever_threshold: 37.3 })

const selectedClass = ref(0)
const openId = ref(0)
const completedOpen = ref(true)

// 草稿按记录 id 分开存：展开另一人不会冲掉上一个人已填的内容。
const drafts = reactive({})
const media = reactive({})
const uploading = reactive({})
const submittingId = ref(0)
const submittingStatus = ref(-1)
const submitError = reactive({ id: 0, message: "" })
let lastAttempt = null

const dateLabel = computed(() => formatDateLabel(date.value))
const weekdayLabel = computed(() => formatWeekdayShort(date.value))

const pendingAll = computed(() =>
  queue.value
    .filter((row) => Number(row.check_status) === 0)
    .slice()
    .sort((a, b) => String(a.pickup_time || "").localeCompare(String(b.pickup_time || "")))
)
const doneAll = computed(() =>
  queue.value
    .filter((row) => Number(row.check_status) !== 0)
    .slice()
    .sort((a, b) => String(a.pickup_time || "").localeCompare(String(b.pickup_time || "")))
)
// 空态只在「全园当日无待检」时出现；某个班先查完但别班还有待检时，
// 保持正常态并在待检区显示「本班的晨检都完成了」，避免把全班误报成全院完成。
const allDone = computed(() => !loading.value && !loadError.value && pendingAll.value.length === 0)

const classList = computed(() => {
  const seen = new Map()
  queue.value.forEach((row) => {
    if (row.class_id == null || seen.has(row.class_id)) return
    seen.set(row.class_id, row.class_name || "未分班")
  })
  return Array.from(seen, ([id, name]) => ({ id, name }))
})

const scopedRows = computed(() => {
  if (!selectedClass.value) return queue.value
  return queue.value.filter((row) => Number(row.class_id) === Number(selectedClass.value))
})
const pendingList = computed(() =>
  allDone.value
    ? pendingAll.value
    : pendingAll.value.filter((row) => !selectedClass.value || Number(row.class_id) === Number(selectedClass.value))
)
const completedRows = computed(() => {
  if (allDone.value) return doneAll.value
  return doneAll.value.filter(
    (row) => !selectedClass.value || Number(row.class_id) === Number(selectedClass.value)
  )
})

const stats = computed(() => {
  const rows = allDone.value ? queue.value : scopedRows.value
  const done = rows.filter((row) => Number(row.check_status) !== 0).length
  return { total: rows.length, done, pending: rows.length - done }
})
const pendingHot = computed(() => !loading.value && !loadError.value && !allDone.value && stats.value.pending > 0)

const policyText = computed(() =>
  policy.value.temperature_required
    ? `本园要求测温 · ${policy.value.fever_threshold}°C 预警`
    : "按需测温 · 体温选填"
)
const pendingHint = computed(() => (pendingList.value.length ? "点开逐项检查" : "本班已全部完成"))
const emptyAttention = computed(
  () => doneAll.value.filter((row) => Number(row.check_status) === 1 && Number(row.overall_status) !== 0).length
)

function draftFor(id) {
  if (!drafts[id]) {
    drafts[id] = {
      temperature: "",
      throat_status: 0,
      hand_status: 0,
      mouth_status: 0,
      eye_status: 0,
      skin_status: 0,
      remark: "",
      is_isolated: false
    }
  }
  return drafts[id]
}

function initialOf(name) {
  return String(name || "?").slice(0, 1)
}

// check_status 与 overall_status 是两套语义：2=已跳过，不能落到「健康」；
// overall 只在 check_status=1（确实检查过）时才代表结论。
function resultOf(row) {
  if (Number(row.check_status) === 2) return { label: "已跳过", cls: "is-skip" }
  const overall = Number(row.overall_status)
  if (overall === 1) return { label: "需观察", cls: "is-warning" }
  if (overall === 2) return { label: "劝返", cls: "is-danger" }
  return { label: "健康", cls: "is-success" }
}

function feverText(id) {
  const raw = String(draftFor(id).temperature || "").trim()
  if (!raw) return ""
  const value = Number(raw)
  const threshold = Number(policy.value.fever_threshold)
  if (!Number.isFinite(value) || !Number.isFinite(threshold) || value < threshold) return ""
  return `${value}°C 高于阈值 ${threshold}°C`
}

function remarkLength(id) {
  return String(draftFor(id).remark || "").length
}

function selectClass(id) {
  if (submittingId.value) return
  selectedClass.value = id
  openId.value = 0
}

function toggleChild(row) {
  if (submittingId.value) return
  if (openId.value === row.id) {
    openId.value = 0
    return
  }
  draftFor(row.id)
  if (!media[row.id]) media[row.id] = []
  openId.value = row.id
}

function selectCheck(id, field, value) {
  if (submittingId.value === id) return
  const draft = draftFor(id)
  // 再点一次已选中的异常值 = 取消回「正常」；点「正常」始终归零。
  draft[field] = value === 0 ? 0 : draft[field] === value ? 0 : value
}

function chipClass(id, field, value) {
  const active = draftFor(id)[field] === value
  return {
    "is-active": active,
    "is-normal": active && value === 0,
    "is-abnormal": active && value !== 0
  }
}

function isAbnormal(id, field) {
  return draftFor(id)[field] !== 0
}

function onTempInput(id, event) {
  draftFor(id).temperature = event.detail?.value ?? ""
}

function onRemarkInput(id, event) {
  draftFor(id).remark = event.detail?.value ?? ""
}

function toggleIsolation(id) {
  if (submittingId.value === id) return
  const draft = draftFor(id)
  draft.is_isolated = !draft.is_isolated
}

async function capturePhoto(id, category) {
  if (submittingId.value) return
  const key = `${id}:${category}`
  if (uploading[key]) return
  const meta = CHECK_BY_KEY[Object.keys(CHECK_BY_KEY).find((field) => CHECK_BY_KEY[field].category === category)]
  uni.chooseImage({
    count: 1,
    sourceType: ["camera"],
    success: async (res) => {
      const filePath = res.tempFilePaths && res.tempFilePaths[0]
      if (!filePath) return
      uploading[key] = true
      try {
        const up = await uploadHealthPhoto(filePath, { silent: true })
        const fileId = up.data && up.data.id
        if (!fileId) throw new Error("上传失败")
        const linked = await addCheckMedia(
          id,
          { file_id: fileId, category, ref_desc: `${meta?.label || category}异常` },
          { silent: true }
        )
        if (!linked.data) throw new Error("关联失败")
        if (!media[id]) media[id] = []
        media[id].push(linked.data)
        uni.showToast({ title: "照片已保存", icon: "success" })
      } catch (err) {
        uni.showToast({ title: err?.message || "拍照上传失败", icon: "none" })
      } finally {
        uploading[key] = false
      }
    }
  })
}

function buildPayload(id, status) {
  const draft = draftFor(id)
  const raw = String(draft.temperature || "").trim()
  if (!raw && policy.value.temperature_required) {
    uni.showToast({ title: "本园要求录入体温", icon: "none" })
    return null
  }
  if (raw && !Number.isFinite(Number(raw))) {
    uni.showToast({ title: "请输入正确的体温", icon: "none" })
    return null
  }
  // symptom_tags 由服务端派生；照片已通过 /media 单独关联。
  return {
    temperature: raw ? Number(raw) : null,
    throat_status: draft.throat_status,
    hand_status: draft.hand_status,
    mouth_status: draft.mouth_status,
    eye_status: draft.eye_status,
    skin_status: draft.skin_status,
    overall_status: status,
    is_isolated: draft.is_isolated,
    remark: String(draft.remark || "").trim()
  }
}

async function submit(id, status) {
  if (submittingId.value) return
  const payload = buildPayload(id, status)
  if (!payload) return
  submittingId.value = id
  submittingStatus.value = status
  submitError.id = 0
  submitError.message = ""
  lastAttempt = { id, status }
  try {
    const res = await submitCheck(id, payload, { silent: true })
    const data = res.data || {}
    if (Number(data.notify_failed_count) > 0) {
      // 通知失败不能静默：保健医需要知道家长没收到。
      uni.showModal({
        title: "已记录",
        content: `${data.notify_failed_count} 位家长通知失败，请线下告知`,
        showCancel: false
      })
    } else {
      const label = status === 0 ? "健康入园" : status === 1 ? "需观察" : "劝返就医"
      const extra = Number(data.notified_count) > 0 ? " · 已通知家长" : ""
      uni.showToast({ title: `${label}${extra}`, icon: "none" })
    }
    // 先就地移入已完成，队列上下文不闪断；随后后台校准服务端状态。
    const row = queue.value.find((item) => item.id === id)
    if (row) {
      row.check_status = 1
      row.overall_status = status
    }
    openId.value = 0
    delete drafts[id]
    delete media[id]
    // 不 await：后台校准不应延长按钮禁用时间；loadQueue 内部已兜住错误。
    loadQueue({ background: true })
  } catch {
    submitError.id = id
    submitError.message = "提交失败，请重试"
  } finally {
    submittingId.value = 0
    submittingStatus.value = -1
  }
}

function retrySubmit(id) {
  if (submittingId.value || !lastAttempt || lastAttempt.id !== id) return
  submit(id, lastAttempt.status)
}

function showCompleted() {
  completedOpen.value = true
  setTimeout(() => {
    const query = uni.createSelectorQuery()
    query.select("#completedSection").boundingClientRect()
    query.selectViewport().scrollOffset()
    query.exec((result) => {
      const rect = result && result[0]
      const viewport = result && result[1]
      if (!rect || !viewport) return
      uni.pageScrollTo({
        scrollTop: Math.max(0, viewport.scrollTop + rect.top - 36),
        duration: 300
      })
    })
  }, 30)
}

function goGardenCalendar() {
  uni.navigateTo({ url: "/pages/doctor/calendar" })
}

async function loadQueue({ background = false } = {}) {
  if (!background) loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const res = await fetchHealthQueue({ date: date.value }, { silent: true })
    queue.value = res.data?.list || []
    policy.value = res.data?.policy || { temperature_required: false, fever_threshold: 37.3 }
    if (!background) openId.value = 0
  } catch (err) {
    if (background) {
      // 提交已经成功，后台校准失败不该把整页打回错误态；保留本地乐观结果。
      uni.showToast({ title: "晨检数据刷新失败", icon: "none" })
      return
    }
    // 队列为空在这页的含义是「全部检查完毕」，请求失败时静默留空等于
    // 告诉保健医今天的孩子都查过了；必须让错误态压过空态。
    loadError.value = err?.message || "晨检数据加载失败"
    queue.value = []
  } finally {
    if (!background) loading.value = false
  }
}

onLoad(() => {
  loadQueue()
})
</script>

<template>
  <PageShell max-width="430px">
    <view class="checkin-page">
      <view class="today-card">
        <view class="today-top">
          <view class="today-date">
            <text class="today-date-main">{{ dateLabel }}</text>
            <text class="today-weekday">{{ weekdayLabel }}</text>
          </view>
          <view
            v-if="!loading && !loadError"
            class="policy-pill"
            :class="{ 'is-required': policy.temperature_required }"
          >
            <view class="policy-dot" />
            <text class="policy-text">{{ policyText }}</text>
          </view>
        </view>
        <view class="today-stats">
          <view class="stat">
            <view v-if="loading" class="stat-skel" />
            <text v-else class="stat-num">{{ loadError ? "—" : stats.total }}</text>
            <text class="stat-label">总入园</text>
          </view>
          <view class="stat" :class="{ 'is-hot': pendingHot }">
            <view v-if="loading" class="stat-skel" />
            <text v-else class="stat-num">{{ loadError ? "—" : stats.pending }}</text>
            <text class="stat-label">待检</text>
          </view>
          <view class="stat">
            <view v-if="loading" class="stat-skel" />
            <text v-else class="stat-num">{{ loadError ? "—" : stats.done }}</text>
            <text class="stat-label">已完成</text>
          </view>
        </view>
      </view>

      <view class="event-cell" @click="goGardenCalendar">
        <view class="event-icon"><text>园</text></view>
        <view class="event-copy">
          <text class="event-title">发布园级事件</text>
          <text class="event-sub">体检等 · 全园可见</text>
        </view>
        <text class="icon event-chev">&#xea6e;</text>
      </view>

      <view v-if="!loading && !loadError && !allDone" class="filter-wrap">
        <view class="class-chip" :class="{ 'is-active': selectedClass === 0 }" @click="selectClass(0)">
          <text class="class-chip-face">全部</text>
        </view>
        <view
          v-for="item in classList"
          :key="item.id"
          class="class-chip"
          :class="{ 'is-active': Number(selectedClass) === Number(item.id) }"
          @click="selectClass(item.id)"
        >
          <text class="class-chip-face">{{ item.name }}</text>
        </view>
      </view>

      <view v-if="loading" class="skeleton-list">
        <view v-for="i in 3" :key="i" class="sk-card">
          <view class="sk-avatar" />
          <view class="sk-lines">
            <view class="sk-line" :style="{ width: i === 1 ? '38%' : i === 2 ? '46%' : '34%' }" />
            <view class="sk-line is-sub" :style="{ width: i === 1 ? '64%' : i === 2 ? '58%' : '68%' }" />
          </view>
        </view>
      </view>

      <view v-else-if="loadError" class="state-block" role="alert">
        <view class="state-icon is-danger"><text class="icon state-glyph">&#xeca1;</text></view>
        <text class="state-title">晨检数据加载失败</text>
        <text class="state-desc">网络异常，或当前账号没有保健医权限</text>
        <view class="primary-btn state-cta" @click="loadQueue"><text>重试</text></view>
      </view>

      <view v-else-if="allDone" class="state-block">
        <view class="state-icon is-success"><text class="icon state-glyph">&#xeb7b;</text></view>
        <text class="state-title">今天的晨检都完成了</text>
        <text class="state-desc">
          <text class="state-strong">{{ queue.length }}</text> 名幼儿已完成，<text class="state-strong">{{ emptyAttention }}</text> 人需留意
        </text>
        <view class="primary-btn state-cta" @click="showCompleted"><text>查看已完成</text></view>
      </view>

      <view v-else class="pending-section">
        <view class="section-head">
          <view class="section-title-wrap">
            <text class="section-title">待检</text>
            <text class="section-count">{{ pendingList.length }} 人</text>
          </view>
          <text class="section-hint">{{ pendingHint }}</text>
        </view>

        <view v-if="pendingList.length" class="child-list">
          <view
            v-for="row in pendingList"
            :key="row.id"
            class="child-card"
            :class="{ 'is-open': openId === row.id }"
          >
            <view class="child-toggle" @click="toggleChild(row)">
              <image v-if="row.avatar" class="avatar" :src="row.avatar" mode="aspectFill" />
              <view v-else class="avatar">{{ initialOf(row.student_name) }}</view>
              <view class="child-main">
                <text class="child-name">{{ row.student_name }}</text>
                <text class="child-meta">{{ row.class_name }} · {{ row.pickup_time }} 入园</text>
              </view>
              <view class="child-trail">
                <text class="child-check">{{ openId === row.id ? "检查中" : "待晨检" }}</text>
                <view class="child-collapse">
                  <text>{{ openId === row.id ? "收起" : "展开" }}</text>
                  <text class="icon child-chev" :class="{ 'is-open': openId === row.id }">&#xea4e;</text>
                </view>
              </view>
            </view>

            <view
              v-if="openId === row.id && drafts[row.id]"
              class="check-panel"
              :class="{ 'is-locked': submittingId === row.id }"
            >
              <view class="field-row">
                <text class="field-label">
                  体温
                  <text class="field-note" :class="{ 'is-required': policy.temperature_required }">
                    {{ policy.temperature_required ? "必填" : "选填" }}
                  </text>
                </text>
                <view class="temp-field">
                  <input
                    class="temp-input"
                    type="digit"
                    maxlength="4"
                    placeholder="36.5"
                    :value="drafts[row.id].temperature"
                    :disabled="submittingId === row.id"
                    @input="onTempInput(row.id, $event)"
                  />
                  <text class="temp-unit">°C</text>
                </view>
              </view>
              <view v-if="feverText(row.id)" class="temp-hint">
                <view class="temp-hint-dot" />
                <text>{{ feverText(row.id) }}</text>
              </view>

              <view class="check-list">
                <view v-for="item in CHECKS" :key="item.key" class="check-row">
                  <text class="check-label">{{ item.label }}</text>
                  <view class="check-chips">
                    <view
                      v-for="(label, value) in item.options"
                      :key="value"
                      class="check-chip"
                      :class="chipClass(row.id, item.key, value)"
                      @click="selectCheck(row.id, item.key, value)"
                    >
                      <text class="check-chip-face">{{ label }}</text>
                    </view>
                  </view>
                  <view
                    v-if="isAbnormal(row.id, item.key)"
                    class="photo-mini"
                    @click="capturePhoto(row.id, item.category)"
                  >
                    <view class="photo-mini-face">
                      <text class="icon photo-icon">&#xeb31;</text>
                    </view>
                  </view>
                </view>
              </view>

              <view class="remark-block">
                <text class="field-label">备注</text>
                <textarea
                  class="remark-input"
                  maxlength="200"
                  placeholder="异常情况说明（选填）"
                  :value="drafts[row.id].remark"
                  :disabled="submittingId === row.id"
                  @input="onRemarkInput(row.id, $event)"
                />
                <view class="remark-foot"><text class="remark-count">{{ remarkLength(row.id) }}/200</text></view>
              </view>

              <view v-if="(media[row.id] || []).length" class="photos-block">
                <text class="photos-title">异常照片</text>
                <view class="photo-list">
                  <image
                    v-for="item in media[row.id]"
                    :key="item.id"
                    class="photo-item"
                    :src="item.thumb_url || item.file_url"
                    mode="aspectFill"
                  />
                </view>
              </view>

              <view
                class="isolation-row"
                :class="{ 'is-on': drafts[row.id].is_isolated }"
                @click="toggleIsolation(row.id)"
              >
                <text class="isolation-copy">已带到隔离室等待家长</text>
                <view class="switch-track"><view class="switch-knob" /></view>
              </view>

              <view class="submit-area">
                <view v-if="submitError.id === row.id" class="submit-error" role="alert">
                  <text class="submit-error-text">{{ submitError.message }}</text>
                  <view class="retry-link" @click="retrySubmit(row.id)"><text>重试</text></view>
                </view>
                <view class="submit-actions" :class="{ 'is-submitting': submittingId === row.id }">
                  <view
                    class="submit-btn submit-primary"
                    :aria-disabled="submittingId === row.id"
                    @click="submit(row.id, 0)"
                  >
                    <text>{{ submittingId === row.id && submittingStatus === 0 ? "提交中…" : "健康入园" }}</text>
                  </view>
                  <view class="submit-secondary">
                    <view
                      class="submit-btn submit-warn"
                      :aria-disabled="submittingId === row.id"
                      @click="submit(row.id, 1)"
                    >
                      <text>{{ submittingId === row.id && submittingStatus === 1 ? "提交中…" : "需观察" }}</text>
                    </view>
                    <view
                      class="submit-btn submit-danger"
                      :aria-disabled="submittingId === row.id"
                      @click="submit(row.id, 2)"
                    >
                      <text>{{ submittingId === row.id && submittingStatus === 2 ? "提交中…" : "劝返就医" }}</text>
                    </view>
                  </view>
                </view>
              </view>
            </view>
          </view>
        </view>

        <view v-else class="quiet-empty">
          <text class="icon quiet-icon">&#xeb7b;</text>
          <text>本班的晨检都完成了</text>
        </view>
      </view>

      <view v-if="!loading && !loadError && completedRows.length" id="completedSection" class="completed-section">
        <view class="completed-heading" @click="completedOpen = !completedOpen">
          <view class="section-title-wrap">
            <text class="section-title">已完成</text>
            <text class="section-count">{{ completedRows.length }}</text>
          </view>
          <view class="completed-action">
            <text>{{ completedOpen ? "收起" : "展开" }}</text>
            <text class="icon chev" :class="{ 'is-open': completedOpen }">&#xea4e;</text>
          </view>
        </view>
        <view v-if="completedOpen" class="completed-card">
          <view v-for="row in completedRows" :key="row.id" class="completed-row">
            <image v-if="row.avatar" class="avatar is-sm" :src="row.avatar" mode="aspectFill" />
            <view v-else class="avatar is-sm">{{ initialOf(row.student_name) }}</view>
            <view class="completed-main">
              <text class="completed-name">{{ row.student_name }}</text>
              <text class="completed-class">{{ row.class_name }}</text>
            </view>
            <view class="status-tag" :class="resultOf(row).cls">
              <view class="status-dot" />
              <text>{{ resultOf(row).label }}</text>
            </view>
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

.checkin-page {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

/* ---------- 今日卡 ---------- */
.today-card {
  padding: 28rpx 32rpx 24rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.today-top {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md $spacing-lg;
}
.today-date {
  display: inline-flex;
  align-items: baseline;
  min-width: 0;
}
.today-date-main {
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
  color: $color-text-primary;
}
.today-weekday {
  margin-left: $spacing-sm;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
}
.policy-pill {
  display: inline-flex;
  align-items: center;
  gap: $spacing-sm;
  max-width: 100%;
  min-height: 52rpx;
  padding: 0 $component-badge-padding-x;
  border-radius: $border-radius-full;
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
  font-size: $component-badge-font-size;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}
.policy-pill.is-required {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}
.policy-dot {
  flex: none;
  width: 10rpx;
  height: 10rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-info;
}
.policy-pill.is-required .policy-dot {
  background: $color-semantic-warning;
}
.policy-text {
  min-width: 0;
}
.today-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-top: $spacing-lg;
  padding-top: $spacing-lg;
  border-top: 2rpx solid $color-border-subtle;
}
.stat {
  min-width: 0;
  padding: 0 20rpx;
}
.stat:first-child {
  padding-left: 0;
}
.stat:last-child {
  padding-right: 0;
}
.stat + .stat {
  border-left: 2rpx solid $color-border-subtle;
}
.stat-num {
  display: block;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
  color: $color-text-primary;
}
.stat.is-hot .stat-num {
  color: $color-brand-primary-dark;
}
.stat-label {
  display: block;
  margin-top: 4rpx;
  font-size: $typography-font-size-sm;
  line-height: 1.2;
  color: $color-text-muted;
}
.stat-skel {
  width: 72rpx;
  height: 36rpx;
  margin: 6rpx 0;
  border-radius: $border-radius-sm;
  background: $color-brand-primary-bg;
  animation: sk-pulse 1.4s $animation-easing-default infinite;
}

/* ---------- 园级事件入口 ---------- */
.event-cell {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-height: 120rpx;
  margin-top: 36rpx;
  padding: $spacing-lg 28rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.event-cell:active {
  opacity: 0.86;
}
.event-icon {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 56rpx;
  height: 56rpx;
  border-radius: $border-radius-lg;
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}
.event-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2rpx;
  min-width: 0;
}
.event-title {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
  color: $color-text-primary;
}
.event-sub {
  font-size: $typography-font-size-sm;
  line-height: 1.35;
  color: $color-text-muted;
}
.event-chev {
  flex: none;
  font-size: $typography-font-size-md;
  color: $color-text-muted;
}

/* ---------- 班级筛选 ---------- */
.filter-wrap {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0 $spacing-md;
  margin-top: $spacing-lg;
}
.class-chip {
  display: inline-flex;
  align-items: center;
  padding: $spacing-sm 0;
  border-radius: $border-radius-full;
}
.class-chip-face {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 64rpx;
  padding: 0 28rpx;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}
.class-chip.is-active .class-chip-face {
  border-color: $color-brand-primary;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}
.class-chip:active .class-chip-face {
  opacity: 0.82;
}

/* ---------- 待检列表 ---------- */
.pending-section {
  margin-top: $spacing-lg;
  min-width: 0;
}
.section-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-xs $spacing-lg;
  min-height: 48rpx;
  padding: 0 4rpx;
}
.section-title-wrap {
  display: inline-flex;
  align-items: baseline;
  min-width: 0;
}
.section-title {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
  color: $color-text-primary;
}
.section-count {
  margin-left: $spacing-xs;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
}
.section-hint {
  font-size: $typography-font-size-sm;
  line-height: 1.25;
  color: $color-text-muted;
}
.child-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  margin-top: $spacing-md;
}
.child-card {
  border: 2rpx solid transparent;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.child-card.is-open {
  border-color: $color-brand-primary-light;
  box-shadow: $shadow-md;
}
.child-toggle {
  display: flex;
  align-items: center;
  gap: 20rpx;
  width: 100%;
  min-height: 120rpx;
  padding: 20rpx 24rpx;
  box-sizing: border-box;
  border-radius: $component-card-radius;
}
.avatar {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}
.avatar.is-sm {
  width: 56rpx;
  height: 56rpx;
  font-size: $typography-font-size-xs;
}
.child-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}
.child-name {
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
  color: $color-text-primary;
}
.child-meta {
  font-size: $typography-font-size-sm;
  line-height: 1.25;
  color: $color-text-muted;
}
.child-trail {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 4rpx;
}
.child-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 56rpx;
  padding: 0 24rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}
.child-collapse {
  display: inline-flex;
  align-items: center;
  gap: 4rpx;
  padding: $spacing-md $spacing-xs;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  color: $color-brand-primary-dark;
}
.child-chev {
  font-size: $typography-font-size-sm;
  transition: transform $animation-duration-fast $animation-easing-default;
}
.child-chev.is-open {
  transform: rotate(180deg);
}

/* ---------- 展开面板 ---------- */
.check-panel {
  padding: $spacing-lg $spacing-lg 28rpx;
  border-top: 2rpx solid $color-border-subtle;
}
.check-panel.is-locked .check-chip,
.check-panel.is-locked .photo-mini,
.check-panel.is-locked .isolation-row {
  opacity: 0.55;
}
.field-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  min-height: 88rpx;
}
.field-label {
  display: inline-flex;
  align-items: baseline;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
  color: $color-text-muted;
}
.field-note {
  margin-left: $spacing-sm;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
}
.field-note.is-required {
  color: $color-semantic-danger;
}
.temp-field {
  position: relative;
  flex: none;
  width: 192rpx;
  height: 80rpx;
}
.temp-input {
  box-sizing: border-box;
  width: 192rpx;
  height: 80rpx;
  padding: 0 56rpx 0 24rpx;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
}
.temp-unit {
  position: absolute;
  top: 50%;
  right: 20rpx;
  transform: translateY(-50%);
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}
.temp-hint {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  margin-top: $spacing-sm;
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius-md;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.3;
}
.temp-hint-dot {
  flex: none;
  width: 10rpx;
  height: 10rpx;
  margin-top: 8rpx;
  border-radius: $border-radius-full;
  background: $color-semantic-danger;
}
.check-list {
  margin-top: 20rpx;
}
.check-row {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm $spacing-md;
  min-height: 88rpx;
  padding: 4rpx 0;
}
.check-label {
  flex: none;
  width: 96rpx;
  padding-top: 22rpx;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
  color: $color-text-muted;
}
.check-chips {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  gap: 0 $spacing-sm;
  min-width: 0;
}
.check-chip {
  display: inline-flex;
  align-items: center;
  padding: $spacing-sm 0;
  border-radius: $border-radius-full;
}
.check-chip-face {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 64rpx;
  padding: 0 $spacing-lg;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}
.check-chip.is-normal .check-chip-face {
  border-color: $color-semantic-success;
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}
.check-chip.is-abnormal .check-chip-face {
  border-color: $color-semantic-danger;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}
.check-chip:active .check-chip-face {
  opacity: 0.82;
}
.photo-mini {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
}
.photo-mini-face {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56rpx;
  height: 56rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
  box-shadow: $shadow-sm;
}
.photo-icon {
  font-size: $typography-font-size-base;
}

.remark-block {
  margin-top: $spacing-lg;
}
.remark-input {
  display: block;
  box-sizing: border-box;
  width: 100%;
  min-height: 128rpx;
  margin-top: $spacing-sm;
  padding: 18rpx 24rpx;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}
.remark-foot {
  display: flex;
  justify-content: flex-end;
  margin-top: 4rpx;
}
.remark-count {
  font-size: $typography-font-size-xs;
  color: $color-text-muted;
}

.photos-block {
  margin-top: $spacing-lg;
}
.photos-title {
  display: block;
  margin-bottom: $spacing-sm;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
  color: $color-text-muted;
}
.photo-list {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;
}
.photo-item {
  width: 104rpx;
  height: 104rpx;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.isolation-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  width: 100%;
  box-sizing: border-box;
  min-height: 96rpx;
  margin-top: 28rpx;
  padding: $spacing-md $spacing-lg;
  border: 2rpx solid $color-border-hairline;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  color: $color-text-body;
}
.isolation-row.is-on {
  border-color: $color-semantic-danger;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}
.isolation-copy {
  flex: 1;
  min-width: 0;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.5;
}
.switch-track {
  position: relative;
  flex: none;
  width: 88rpx;
  height: 52rpx;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
  transition: background $animation-duration-fast $animation-easing-default;
}
.switch-knob {
  position: absolute;
  top: 6rpx;
  left: 6rpx;
  width: 40rpx;
  height: 40rpx;
  border-radius: $border-radius-full;
  background: $color-text-on-brand;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast $animation-easing-default;
}
.isolation-row.is-on .switch-track {
  background: $color-semantic-danger;
}
.isolation-row.is-on .switch-knob {
  transform: translateX(36rpx);
}

.submit-area {
  margin-top: 28rpx;
  padding-top: $spacing-lg;
  border-top: 2rpx solid $color-border-subtle;
}
.submit-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-sm;
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius-lg;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}
.submit-error-text {
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}
.retry-link {
  display: inline-flex;
  align-items: center;
  min-height: 88rpx;
  padding: 0 $spacing-md;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}
.submit-actions {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  margin-top: $spacing-md;
  transition: opacity $animation-duration-fast $animation-easing-default;
}
.submit-actions.is-submitting {
  opacity: 0.55;
}
.submit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
}
.submit-btn:active {
  opacity: 0.86;
}
.submit-primary {
  background: $gradient-brand;
  color: $color-text-on-brand;
  box-shadow: $shadow-md;
}
.submit-secondary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: $spacing-md;
}
.submit-warn {
  border: 2rpx solid $color-semantic-warning;
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}
.submit-danger {
  border: 2rpx solid $color-semantic-danger;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.quiet-empty {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-md;
  padding: $spacing-lg;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.25;
}
.quiet-icon {
  font-size: $typography-font-size-base;
  color: $color-semantic-success;
}

/* ---------- 骨架屏 ---------- */
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
  margin-top: 28rpx;
}
.sk-card {
  display: flex;
  align-items: center;
  gap: 20rpx;
  min-height: 120rpx;
  padding: 20rpx 24rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.sk-avatar {
  flex: none;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  animation: sk-pulse 1.4s $animation-easing-default infinite;
}
.sk-lines {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-md;
  min-width: 0;
}
.sk-line {
  height: 20rpx;
  border-radius: $border-radius-sm;
  background: $color-brand-primary-bg;
  animation: sk-pulse 1.4s $animation-easing-default infinite;
}
.sk-line.is-sub {
  height: 16rpx;
  background: $color-border-subtle;
}
@keyframes sk-pulse {
  0%,
  100% {
    opacity: 0.42;
  }
  50% {
    opacity: 0.78;
  }
}

/* ---------- 空态 / 错误态 ---------- */
.state-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 28rpx;
  padding: 52rpx 36rpx 44rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}
.state-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 128rpx;
  height: 128rpx;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-full;
}
.state-icon.is-success {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}
.state-icon.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}
.state-glyph {
  font-size: $typography-font-size-3xl;
}
.state-title {
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.25;
  color: $color-text-primary;
}
.state-desc {
  max-width: 30em;
  margin-top: $spacing-sm;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  color: $color-text-muted;
}
.state-strong {
  font-weight: $typography-font-weight-semibold;
  color: $color-text-body;
}
.primary-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: $spacing-xl;
  padding: 0 48rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  box-shadow: $shadow-md;
}
.primary-btn:active {
  opacity: 0.86;
}

/* ---------- 已完成 ---------- */
.completed-section {
  margin-top: 36rpx;
  min-width: 0;
}
.completed-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  min-height: 88rpx;
  padding: 0 4rpx;
}
.completed-action {
  display: inline-flex;
  align-items: center;
  gap: 4rpx;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  color: $color-text-muted;
}
.chev {
  font-size: $typography-font-size-sm;
  transition: transform $animation-duration-fast $animation-easing-default;
}
.chev.is-open {
  transform: rotate(180deg);
}
.completed-card {
  margin-top: $spacing-md;
  overflow: hidden;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.completed-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-height: 96rpx;
  padding: $spacing-md $spacing-lg;
}
.completed-row + .completed-row {
  border-top: 2rpx solid $color-border-subtle;
}
.completed-main {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  align-items: center;
  gap: 4rpx $spacing-sm;
  min-width: 0;
}
.completed-name {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
  color: $color-text-body;
}
.completed-class {
  font-size: $typography-font-size-sm;
  line-height: 1.25;
  color: $color-text-muted;
}
.status-tag {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: $spacing-xs;
  min-height: 44rpx;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
}
.status-dot {
  flex: none;
  width: 8rpx;
  height: 8rpx;
  border-radius: $border-radius-full;
}
.status-tag.is-success {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}
.status-tag.is-success .status-dot {
  background: $color-semantic-success;
}
.status-tag.is-warning {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}
.status-tag.is-warning .status-dot {
  background: $color-semantic-warning;
}
.status-tag.is-danger {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}
.status-tag.is-danger .status-dot {
  background: $color-semantic-danger;
}
.status-tag.is-skip {
  background: $color-bg-page;
  color: $color-text-muted;
}
.status-tag.is-skip .status-dot {
  background: $color-text-hint;
}
</style>
