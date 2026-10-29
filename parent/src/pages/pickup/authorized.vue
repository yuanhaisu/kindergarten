<script setup>
/**
 * 授权接送人（家长端子页）
 *
 * 导航栏已承担标题，页内不再重复大标题；内容区是「可信名单」：
 * 安全提示 → 工具行 → 今日接走人横幅 → 人员卡 → 页脚说明。
 * 权限标签只在有效授权（review_status=1 且 status=1）上显示，
 * 待审核 / 已撤销只在状态徽标与降级按钮上表达，避免同卡自相矛盾。
 */
import { computed, ref } from "vue"
import { onShow } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId } from "@/utils/currentStudent"
import {
  fetchAuthorizedPersons,
  deleteAuthorizedPerson,
  designatePickupPerson,
  fetchDesignatedPerson
} from "@/api/parent-pickup"
import { relationLabel } from "@/utils/relation"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形（App.vue 已全局注册字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  shield: "", // shield-check-line
  add: "", // add-line
  check: "", // check-line
  alert: "", // error-warning-line（复用为提醒三角）
  trash: "", // delete-bin-line
  info: "", // information-line
  userAdd: "", // user-add-line
  error: "" // error-warning-line
}

const persons = ref([])
const loading = ref(false)
const retrying = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
// 当前指定的授权人 ID（null = 今日尚未指定）
const designatedId = ref(null)
// 指定/删除在途时忽略重复点击，避免并发请求互相覆盖
const actingId = ref(null)

function statusBadge(person) {
  if (Number(person.status) === 0) return { label: "已撤销", tone: "revoked" }
  if (Number(person.review_status) === 2) return { label: "已拒绝", tone: "rejected" }
  if (Number(person.review_status) === 0) return { label: "待审核", tone: "pending" }
  return null
}

// 权限标签只表达「已生效」的能力：待审核/已撤销不显示，避免绿色「可接送」
// 与「待审核」同卡出现。
function permissionTags(person) {
  if (Number(person.review_status) !== 1 || Number(person.status) !== 1) return []
  const tags = []
  if (person.allow_pickup) tags.push({ key: "pickup", label: "可接送" })
  if (person.allow_view_record) tags.push({ key: "record", label: "看记录" })
  if (person.allow_growth_view) tags.push({ key: "growth", label: "看成长" })
  return tags
}

const cards = computed(() =>
  persons.value.map(p => ({
    raw: p,
    id: p.id,
    name: p.name || "未命名",
    initial: String(p.name || "?").trim().slice(0, 1) || "?",
    authTypeLabel: p.auth_type_label || (p.auth_type === "temporary" ? "临时授权" : "长期授权"),
    relation: relationLabel(p.relation),
    phone: p.phone || "—",
    effective: Number(p.review_status) === 1 && Number(p.status) === 1,
    badge: statusBadge(p),
    designated: p.id === designatedId.value,
    perms: permissionTags(p)
  }))
)

const designatedCard = computed(() => cards.value.find(c => c.designated) || null)
const isFirstLoading = computed(() => loading.value && !hasLoaded.value)
const showError = computed(() => Boolean(loadError.value) && persons.value.length === 0)
const showEmpty = computed(() => hasLoaded.value && !showError.value && persons.value.length === 0)
const showContent = computed(() => !isFirstLoading.value && !showError.value && !showEmpty.value)

async function loadData() {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const sid = getCurrentStudentId()
    // 并行拉列表与当前指定；指定查询无副作用，失败不阻断列表
    const [listRes, desigRes] = await Promise.all([
      fetchAuthorizedPersons(sid),
      fetchDesignatedPerson(sid).catch(() => ({ data: null }))
    ])
    persons.value = listRes.data?.list || []
    designatedId.value = desigRes.data?.designated ? desigRes.data.authorization_id : null
    hasLoaded.value = true
  } catch (err) {
    console.warn("[pickup-authorized] load failed:", err?.message || err)
    if (persons.value.length === 0) {
      loadError.value = "授权接送人加载失败"
    } else {
      uni.showToast({ title: "刷新失败，请稍后再试", icon: "none" })
    }
  } finally {
    loading.value = false
  }
}

function confirmModal(title, content) {
  return new Promise(resolve => {
    uni.showModal({ title, content, success: s => resolve(Boolean(s.confirm)), fail: () => resolve(false) })
  })
}

async function handleDesignate(card) {
  if (actingId.value) return
  const person = card.raw
  // 点按仍给出原因，不让用户靠试探猜「为什么按钮是灰的」
  if (Number(person.review_status) !== 1) {
    uni.showToast({ title: "该接送人尚未通过园所审核", icon: "none" })
    return
  }
  if (Number(person.status) !== 1) {
    uni.showToast({ title: "该接送人授权已撤销", icon: "none" })
    return
  }
  if (card.designated) return

  // 换人会作废今日已生成的凭证 —— 必须讲清楚
  const tip = designatedId.value
    ? `换为 ${person.name} 后，今日已生成的接送二维码将全部失效，需要重新获取。确定继续？`
    : `确定指定 ${person.name} 为今日接走人吗？`
  const ok = await confirmModal("设为今日接走人", tip)
  if (!ok) return

  actingId.value = person.id
  try {
    await designatePickupPerson(person.id, getCurrentStudentId())
    designatedId.value = person.id
    uni.showToast({ title: `今日接走人：${person.name}`, icon: "success" })
  } catch (err) {
    console.warn("[pickup-authorized] designate failed:", err?.message || err)
    uni.showToast({ title: err?.message || "指定失败", icon: "none" })
  } finally {
    actingId.value = null
  }
}

async function handleDelete(card) {
  if (actingId.value) return
  const person = card.raw
  const isDesignated = card.designated
  // 删除今日指定的接走人会让已发给门卫的二维码失效，必须提前说明
  const content = isDesignated
    ? `删除 ${person.name} 后，今日已生成的接送二维码将失效，需要重新指定。确定删除吗？`
    : `确定删除 ${person.name} 吗？`
  const ok = await confirmModal("确认删除", content)
  if (!ok) return

  actingId.value = person.id
  try {
    await deleteAuthorizedPerson(person.id)
    if (isDesignated) designatedId.value = null
    uni.showToast({ title: "已删除", icon: "success" })
    loadData()
  } catch (err) {
    console.warn("[pickup-authorized] delete failed:", err?.message || err)
    uni.showToast({ title: err?.message || "删除失败", icon: "none" })
  } finally {
    actingId.value = null
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

function goAdd() {
  uni.navigateTo({ url: "/pages/pickup/authorized-create" })
}

// 用 onShow 而非 onLoad：从新增页返回时要重新拉列表，否则刚添加的人看不见。
onShow(loadData)
</script>

<template>
  <PageShell bottom-space="56rpx" max-width="430px">
    <!-- 安全提示：白卡 + 浅黄图标块，正文用正文色保证可读 -->
    <view class="safety-card">
      <view class="safety-icon"><text class="safety-glyph">{{ ICON.shield }}</text></view>
      <view class="safety-copy">
        <text class="safety-title">安全提示</text>
        <text class="safety-text">授权接送人信息仅供门卫核验身份，请确保信息真实有效。</text>
      </view>
    </view>

    <!-- 工具行：失败态不谎报数量，空态整行隐藏（空态自带添加入口） -->
    <view v-if="!showEmpty" class="toolbar">
      <view v-if="isFirstLoading" class="toolbar-skeleton" />
      <text v-else-if="!showError" class="toolbar-count">{{ cards.length }} 位授权接送人</text>
      <view class="add-button" @click="goAdd">
        <text class="add-icon">{{ ICON.add }}</text>
        <text>添加</text>
      </view>
    </view>

    <!-- 今日接走人：已指定 / 未指定两态 -->
    <view v-if="isFirstLoading" class="banner-skeleton" />
    <template v-else-if="showContent">
      <view v-if="designatedCard" class="banner banner--assigned">
        <text class="banner-icon">{{ ICON.check }}</text>
        <view class="banner-copy">
          <text class="banner-kicker">今日接走人</text>
          <text class="banner-name">{{ designatedCard.name }}</text>
        </view>
      </view>
      <view v-else class="banner banner--unassigned">
        <text class="banner-icon">{{ ICON.alert }}</text>
        <view class="banner-copy">
          <text class="banner-main">今日尚未指定接走人</text>
          <text class="banner-sub">指定后才能获取接送二维码</text>
        </view>
      </view>
    </template>

    <!-- 加载：3 张骨架卡 -->
    <view v-if="isFirstLoading" class="roster-skeleton">
      <view v-for="i in 3" :key="i" class="skeleton-card">
        <view class="sk-avatar" />
        <view class="skeleton-body">
          <view class="sk-line sk-line--name" />
          <view class="sk-line sk-line--meta" />
          <view class="skeleton-tags">
            <view class="sk-tag" />
            <view class="sk-tag" />
          </view>
          <view class="skeleton-action">
            <view class="sk-line sk-line--action" />
          </view>
        </view>
      </view>
    </view>

    <!-- 错误：先于空态判定 -->
    <view v-else-if="showError" class="state-panel">
      <view class="state-icon state-icon--error"><text class="state-glyph">{{ ICON.error }}</text></view>
      <text class="state-title">授权接送人加载失败</text>
      <text class="state-copy">网络不太稳定，请稍后再试</text>
      <button class="state-button" :disabled="retrying" @click="handleRetry">
        {{ retrying ? "重试中…" : "重试" }}
      </button>
    </view>

    <!-- 空 -->
    <view v-else-if="showEmpty" class="state-panel">
      <view class="state-icon state-icon--empty"><text class="state-glyph">{{ ICON.userAdd }}</text></view>
      <text class="state-title">还没有授权接送人</text>
      <text class="state-copy">添加家人后，才能指定今日接走人并获取接送二维码</text>
      <button class="state-button" @click="goAdd">添加接送人</button>
    </view>

    <!-- 名单 -->
    <view v-else class="roster">
      <view
        v-for="card in cards"
        :key="card.id"
        class="person-card"
        :class="{ 'is-designated': card.designated }"
      >
        <view class="person-avatar" :class="{ 'is-designated': card.designated }">
          <text>{{ card.initial }}</text>
        </view>
        <view class="person-body">
          <view class="person-name-row">
            <text class="person-name">{{ card.name }}</text>
            <text class="badge badge--type">{{ card.authTypeLabel }}</text>
            <text v-if="card.badge" class="badge" :class="'badge--' + card.badge.tone">{{ card.badge.label }}</text>
            <text v-if="card.designated" class="badge badge--today">今日接走人</text>
          </view>
          <text class="person-meta">{{ card.relation }} · {{ card.phone }}</text>
          <view v-if="card.perms.length" class="permission-tags">
            <text
              v-for="perm in card.perms"
              :key="perm.key"
              class="permission-tag"
              :class="'permission-tag--' + perm.key"
            >{{ perm.label }}</text>
          </view>
          <view class="card-actions">
            <view
              v-if="!card.designated"
              class="designate-button"
              :class="{ 'is-unavailable': !card.effective }"
              @click="handleDesignate(card)"
            >设为今日接走人</view>
            <view class="delete-button" @click="handleDelete(card)">
              <text class="delete-icon">{{ ICON.trash }}</text>
              <text>删除</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <!-- 页脚说明：加载/正常保留，空态与错误态隐藏 -->
    <view v-if="!showEmpty && !showError" class="footer-note">
      <text class="footer-icon">{{ ICON.info }}</text>
      <text class="footer-text">新添加的接送人需园所审核通过，才能指定为今日接走人。</text>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 授权接送人 · 可信名单
// 橙色只落在「可操作」与「被选中」上，危险色只出现在删除确认里。
// ============================================================

// ---- 安全提示 ------------------------------------------------
.safety-card {
  display: flex;
  align-items: center;
  gap: 20rpx;
  margin-bottom: $spacing-lg;
  padding: $spacing-lg 28rpx;
  border: 1rpx solid $color-border-hairline;
  border-radius: $component-card-radius;
  background: $color-bg-card;
}

.safety-icon {
  flex: none;
  width: 60rpx;
  height: 60rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}

.safety-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.safety-copy {
  min-width: 0;
}

.safety-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

// 正文不用 hint：门卫核验场景下这行必须可读（设计审查 P2-5）
.safety-text {
  display: block;
  margin-top: 4rpx;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

// ---- 工具行 --------------------------------------------------
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  min-height: 88rpx;
  margin-bottom: $spacing-lg;
}

.toolbar-count {
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.toolbar-skeleton {
  width: 176rpx;
  height: 28rpx;
  flex: none;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  animation: authorized-sk 1.4s $animation-easing-default infinite alternate;
}

.add-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6rpx;
  height: 88rpx;
  padding: 0 32rpx;
  margin-left: auto;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  box-shadow: $shadow-sm;
}

.add-icon {
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

// ---- 今日接走人横幅 ------------------------------------------
.banner {
  display: flex;
  align-items: center;
  gap: 20rpx;
  margin-bottom: $spacing-lg;
  padding: $spacing-lg 28rpx;
  border-radius: $component-card-radius;
}

.banner--assigned {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.banner--unassigned {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}

.banner-icon {
  flex: none;
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.banner-copy {
  min-width: 0;
}

.banner-kicker {
  display: block;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.banner-name {
  display: block;
  margin-top: 2rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.banner-main {
  display: block;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.banner-sub {
  display: block;
  margin-top: 2rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

// ---- 人员卡 --------------------------------------------------
.roster {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.person-card {
  display: grid;
  grid-template-columns: 80rpx minmax(0, 1fr);
  align-items: start;
  gap: $spacing-lg;
  padding: 28rpx 32rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.person-card.is-designated {
  box-shadow: inset 0 0 0 2rpx $color-brand-primary, $shadow-md;
}

.person-avatar {
  width: 80rpx;
  height: 80rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.person-avatar.is-designated {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.person-body {
  min-width: 0;
}

.person-name-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4rpx $spacing-sm;
  min-width: 0;
}

.person-name {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 40rpx;
  max-width: 100%;
  padding: 0 16rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.badge--type {
  background: $color-bg-page;
  color: $color-text-body;
}

.badge--pending {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}

.badge--rejected {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.badge--revoked {
  background: $color-bg-page;
  color: $color-text-muted;
}

.badge--today {
  background: $color-brand-primary-dark;
  color: $color-text-on-brand;
}

// 关系/脱敏手机号是需要被门卫核对的正文，用 13px 而非 12px（设计审查 P2-5）
.person-meta {
  display: block;
  margin-top: 10rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.permission-tags {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
  margin-top: $spacing-md;
  min-width: 0;
}

.permission-tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 40rpx;
  padding: 0 16rpx;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.permission-tag--pickup {
  background: $color-semantic-success-bg;
  color: $color-semantic-success;
}

.permission-tag--record {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.permission-tag--growth {
  background: $color-bg-page;
  color: $color-text-body;
}

.card-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: $spacing-sm;
  margin-top: $spacing-lg;
  padding-top: $spacing-md;
  border-top: 1rpx solid $color-border-subtle;
}

.designate-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  padding: 0 32rpx;
  border: 1rpx solid $color-brand-primary;
  border-radius: $border-radius-full;
  background: transparent;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.designate-button.is-unavailable {
  border-color: transparent;
  background: $color-bg-page;
  color: $color-text-hint;
}

.delete-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  min-height: 88rpx;
  padding: 0 16rpx;
  border-radius: $border-radius-full;
  background: transparent;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.delete-icon {
  font-family: remixicon;
  font-size: $typography-font-size-lg;
  line-height: 1;
}

// ---- 页脚说明 ------------------------------------------------
.footer-note {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  margin-top: $spacing-sm;
  padding: 0 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.55;
}

.footer-icon {
  flex: none;
  margin-top: 2rpx;
  font-family: remixicon;
  font-size: $typography-font-size-base;
  line-height: 1;
}

.footer-text {
  min-width: 0;
  overflow-wrap: anywhere;
}

// ---- 空态 / 错误态 ------------------------------------------
.state-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 96rpx 24rpx 32rpx;
  text-align: center;
}

.state-icon {
  width: 112rpx;
  height: 112rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: $border-radius-full;
}

.state-icon--empty {
  background: $color-bg-page;
  color: $color-text-hint;
}

.state-icon--error {
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.state-glyph {
  font-family: remixicon;
  font-size: 52rpx;
  line-height: 1;
}

.state-title {
  margin-top: 28rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.state-copy {
  max-width: 580rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.state-button {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 320rpx;
  min-height: 88rpx;
  margin-top: 40rpx;
  padding: 0 40rpx;
  border: none;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-brand;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  box-shadow: $shadow-sm;
}

.state-button::after {
  border: none;
}

.state-button[disabled] {
  background: $color-bg-page;
  color: $color-text-muted;
  box-shadow: none;
}

// ---- 骨架 ----------------------------------------------------
.sk-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: $border-radius-full;
}

.sk-line {
  background: $color-bg-page;
  border-radius: $border-radius-full;
  animation: authorized-sk 1.4s $animation-easing-default infinite alternate;
}

.banner-skeleton {
  display: flex;
  align-items: center;
  min-height: 108rpx;
  margin-bottom: $spacing-lg;
  padding: $spacing-lg 28rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
}

.banner-skeleton::after {
  content: "";
  display: block;
  width: 58%;
  height: 28rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  animation: authorized-sk 1.4s $animation-easing-default infinite alternate;
}

.roster-skeleton {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.skeleton-card {
  display: grid;
  grid-template-columns: 80rpx minmax(0, 1fr);
  gap: $spacing-lg;
  padding: 28rpx 32rpx;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.skeleton-card .sk-avatar {
  background: $color-bg-page;
  animation: authorized-sk 1.4s $animation-easing-default infinite alternate;
}

.skeleton-body {
  min-width: 0;
}

.sk-line--name {
  width: 116rpx;
  height: 24rpx;
}

.sk-line--meta {
  width: 256rpx;
  height: 20rpx;
  margin-top: $spacing-md;
}

.skeleton-tags {
  display: flex;
  gap: $spacing-sm;
  margin-top: 20rpx;
}

.sk-tag {
  width: 92rpx;
  height: 40rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  animation: authorized-sk 1.4s $animation-easing-default infinite alternate;
}

.skeleton-action {
  display: flex;
  justify-content: flex-end;
  margin-top: $spacing-lg;
  padding-top: $spacing-md;
  border-top: 1rpx solid $color-border-subtle;
}

.sk-line--action {
  width: 208rpx;
  height: 32rpx;
}

@keyframes authorized-sk {
  0% {
    opacity: 0.45;
  }
  100% {
    opacity: 0.95;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sk-line,
  .sk-avatar,
  .sk-tag,
  .toolbar-skeleton,
  .banner-skeleton::after {
    animation: none;
    opacity: 0.7;
  }
}
</style>
