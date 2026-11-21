<script setup>
/**
 * 我的园所（老师端 · #169）
 *
 * 列表来自 GET /common/user/kindergartens：后端已按「当前客户端角色族 ∩ 园所通道」
 * 过滤，前端不再拼接别处的园所，也不提供搜索/目录入口。
 *
 * 切换走 POST /auth/switch-kindergarten，成功后 api 层已换发并落盘双 token、
 * 清掉授权缓存；本页回到工作台触发一次干净的重载（老师端没有 tabBar，用 reLaunch）。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { getUser } from "@/utils/auth"
import { ensureLogin } from "@/utils/login"
import { fetchMyKindergartens, switchKindergarten } from "@/api/auth"
import { loadCurrentKindergartenBrand } from "@/utils/brand"
import { normalizeBrandColor, tintBrandColor } from "@/utils/kindergartenBrand.mjs"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形（App.vue 已全局注册字体）；小程序 WXML 不允许内联 SVG。
const ICON = {
  building: "\u{eb0f}", // building-line
  check: "\u{eb7b}", // check-line
  alert: "\u{eca1}", // error-warning-line
  inbox: "\u{ee4f}", // inbox-line
  chevron: "\u{ea6e}" // arrow-right-s-line
}

const ROLE_LABELS = {
  teacher: "老师",
  assistant: "配班老师",
  principal: "园长",
  doctor: "保健医",
  parent: "家长"
}

const state = ref("loading") // loading / error / empty / ok
const items = ref([])
const user = ref(getUser())
const switchingID = ref(0)
const loadError = ref("")
const brand = ref(null)
const logoFailed = ref(false)

const switching = computed(() => switchingID.value > 0)

// 当前园以本地用户资料里的教师角色为准：登录/切换返回的 roles 已裁剪到当前园。
const currentKindergartenID = computed(() => {
  const roles = Array.isArray(user.value?.roles) ? user.value.roles : []
  const role = roles.find((item) => item && item.role === "teacher") || roles[0] || null
  const id = Number(role?.kindergarten_id)
  return id > 0 ? id : 0
})

const currentKindergarten = computed(() => {
  const id = currentKindergartenID.value
  return items.value.find((item) => item.kindergarten_id === id) || null
})

const currentName = computed(
  () => currentKindergarten.value?.name || "未读取到当前园所"
)

// 品牌色只驱动本页「当前园」这一面：顶部色条 + 同色柔底，其余样式仍走 token。
// 全站换肤要求页面样式改为消费 CSS 变量（目前两端还没有消费方），属独立设计变更。
const currentBrandColor = computed(() => normalizeBrandColor(brand.value?.brand_color))
const heroTint = computed(() => tintBrandColor(currentBrandColor.value))
// logo 是 10 分钟签名 URL，加载失败就退回字形，不留破图。
const currentLogo = computed(() => {
  if (logoFailed.value) return ""
  return String(brand.value?.logo_url || "").trim()
})

function onLogoError() {
  logoFailed.value = true
}

function roleLabel(item) {
  const role = String(item?.role || "").trim()
  return ROLE_LABELS[role] || role || "成员"
}

function initialOf(name) {
  const chars = Array.from(String(name || "").trim())
  return chars[0] || "园"
}

function isCurrent(item) {
  return item.kindergarten_id === currentKindergartenID.value
}

async function load() {
  state.value = "loading"
  loadError.value = ""
  try {
    await ensureLogin()
    user.value = getUser()
    const res = await fetchMyKindergartens()
    items.value = Array.isArray(res?.data) ? res.data : []
    state.value = items.value.length ? "ok" : "empty"
    brand.value = null
    logoFailed.value = false
    // 品牌是装饰层：loader 内部已兜底，失败只会退回 token 样式，不影响列表。
    if (items.value.length) {
      brand.value = await loadCurrentKindergartenBrand(items.value, currentKindergartenID.value)
    }
  } catch (err) {
    // 失败不静默：整页错误态 + 重试；列表接口 403 时文案由服务端给出。
    console.warn("[kindergartens] load failed:", err?.message || err)
    loadError.value = err?.message || "网络开小差了，请稍后再试"
    state.value = "error"
  }
}

function retry() {
  if (state.value === "loading") return
  load()
}

function confirmSwitch(item) {
  if (switching.value || isCurrent(item)) return
  uni.showModal({
    title: "切换园所",
    content: `切换后将只显示「${item.name}」的数据，未保存的操作会失效。`,
    confirmText: "切换",
    success(result) {
      if (!result.confirm) return
      doSwitch(item)
    }
  })
}

async function doSwitch(item) {
  if (switching.value) return
  switchingID.value = item.kindergarten_id
  try {
    await switchKindergarten(item.kindergarten_id)
    uni.showToast({ title: `已切换到${item.name}`, icon: "none" })
    // 等 toast 露出再走：reLaunch 会销毁当前页，立即跳转会让结果一闪而过。
    setTimeout(() => {
      uni.reLaunch({ url: "/pages/index/index" })
    }, 600)
  } catch (err) {
    if (err?.code !== 40100) {
      uni.showToast({ title: err?.message || "切换失败，请稍后再试", icon: "none" })
    }
  } finally {
    switchingID.value = 0
  }
}

onLoad(load)
</script>

<template>
  <PageShell max-width="430px">
    <view class="kg-page">
      <!-- 当前园所 -->
      <view class="kg-hero" :style="heroTint ? { background: heroTint } : {}">
        <view v-if="currentBrandColor" class="hero-accent" :style="{ background: currentBrandColor }" />
        <view class="hero-icon">
          <image
            v-if="currentLogo"
            class="hero-logo"
            :src="currentLogo"
            mode="aspectFill"
            @error="onLogoError"
          />
          <text v-else class="hero-glyph">{{ ICON.building }}</text>
        </view>
        <view class="hero-copy">
          <text class="hero-label">当前园所</text>
          <text class="hero-name">{{ currentName }}</text>
        </view>
      </view>

      <text class="kg-tip">
        仅展示你在本小程序中已加入的园所。切换后，工作台、班级与接送都会按目标园重新加载。
      </text>

      <!-- 加载 -->
      <view v-if="state === 'loading'" class="kg-list">
        <view v-for="i in 2" :key="i" class="kg-card kg-skeleton">
          <view class="sk sk-avatar" />
          <view class="sk-copy">
            <view class="sk sk-line sk-line--name" />
            <view class="sk sk-line sk-line--meta" />
          </view>
        </view>
      </view>

      <!-- 失败 -->
      <view v-else-if="state === 'error'" class="kg-card kg-state">
        <view class="state-badge is-danger">
          <text class="state-glyph">{{ ICON.alert }}</text>
        </view>
        <text class="state-title">园所列表加载失败</text>
        <text class="state-desc">{{ loadError }}</text>
        <button class="state-btn" @click="retry">重试</button>
      </view>

      <!-- 空 -->
      <view v-else-if="state === 'empty'" class="kg-card kg-state">
        <view class="state-badge">
          <text class="state-glyph">{{ ICON.inbox }}</text>
        </view>
        <text class="state-title">还没有加入任何园所</text>
        <text class="state-desc">园所由园方邀请加入，开通后重新进入本页即可看到。</text>
      </view>

      <!-- 列表 -->
      <view v-else class="kg-list">
        <text class="list-title">我在的园所</text>
        <view
          v-for="item in items"
          :key="item.kindergarten_id"
          class="kg-card"
          :class="{ 'is-current': isCurrent(item), 'is-switching': switchingID === item.kindergarten_id }"
          @click="confirmSwitch(item)"
        >
          <view class="kg-avatar">
            <text class="kg-avatar-text">{{ initialOf(item.name) }}</text>
          </view>
          <view class="kg-copy">
            <text class="kg-name">{{ item.name }}</text>
            <view class="kg-meta">
              <text class="role-pill">{{ roleLabel(item) }}</text>
              <text v-if="isCurrent(item)" class="current-hint">当前使用中</text>
            </view>
          </view>
          <view v-if="isCurrent(item)" class="current-mark">
            <text class="current-glyph">{{ ICON.check }}</text>
          </view>
          <view v-else class="switch-tail">
            <text class="switch-text">{{ switchingID === item.kindergarten_id ? "切换中…" : "切换" }}</text>
            <text class="switch-chevron">{{ ICON.chevron }}</text>
          </view>
        </view>
        <text v-if="items.length === 1" class="list-foot">
          你目前只在一个园所。如需加入其它园所，请联系园长开通。
        </text>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
.kg-page {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

/* 当前园所：暖色柔光定位「我现在在哪」；有品牌色时由 inline style 覆盖底色 */
.kg-hero {
  position: relative;
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  overflow: hidden;
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $gradient-brand-soft;
  box-shadow: $shadow-sm;
}

/* 品牌色条：颜色来自公开品牌接口，宽度固定，不改变布局高度 */
.hero-accent {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 6rpx;
}

.hero-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: $component-avatar-lg;
  height: $component-avatar-lg;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.hero-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  color: $color-brand-primary-dark;
}

.hero-logo {
  width: 100%;
  height: 100%;
  border-radius: $border-radius-full;
}

.hero-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-xs;
  min-width: 0;
}

.hero-label {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.hero-name {
  overflow: hidden;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-bold;
  color: $color-text-primary;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kg-tip {
  font-size: $typography-font-size-sm;
  line-height: 1.6;
  color: $color-text-muted;
}

.kg-list {
  display: flex;
  flex-direction: column;
  gap: $component-card-gap;
}

.list-title {
  padding-left: $spacing-xs;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
}

.kg-card {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  padding: $component-card-padding;
  border: 1rpx solid transparent;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.kg-card.is-current {
  border-color: $color-brand-primary-light;
  background: $color-brand-primary-bg;
}

.kg-card.is-switching {
  opacity: 0.65;
}

.kg-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: $component-avatar-md;
  height: $component-avatar-md;
  border-radius: $border-radius-xl;
  background: $color-brand-primary-bg;
}

.kg-avatar-text {
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-bold;
  color: $color-brand-primary-dark;
}

.kg-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-xs;
  min-width: 0;
}

.kg-name {
  overflow: hidden;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  color: $color-text-primary;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kg-meta {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.role-pill {
  padding: 2rpx $component-badge-padding-x;
  font-size: $component-badge-font-size;
  color: $color-text-body;
  background: $color-border-subtle;
  border-radius: $component-badge-radius;
}

.current-hint {
  font-size: $typography-font-size-xs;
  color: $color-brand-primary-dark;
}

.current-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 48rpx;
  height: 48rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.current-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  color: $color-bg-card;
}

.switch-tail {
  display: flex;
  align-items: center;
  gap: 2rpx;
  flex-shrink: 0;
}

.switch-text {
  font-size: $typography-font-size-sm;
  color: $color-brand-primary-dark;
}

.switch-chevron {
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  color: $color-brand-primary-dark;
}

.list-foot {
  padding: $spacing-xs $spacing-xs 0;
  font-size: $typography-font-size-xs;
  line-height: 1.6;
  color: $color-text-muted;
}

/* 加载骨架 */
.kg-skeleton {
  align-items: center;
}

.sk {
  position: relative;
  overflow: hidden;
  background: $color-border-subtle;
}

.sk::after {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: linear-gradient(90deg, transparent, $color-bg-card, transparent);
  content: "";
  animation: sk-shine 1.4s infinite;
}

.sk-avatar {
  flex-shrink: 0;
  width: $component-avatar-md;
  height: $component-avatar-md;
  border-radius: $border-radius-xl;
}

.sk-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-sm;
}

.sk-line {
  height: 28rpx;
  border-radius: $border-radius-sm;
}

.sk-line--name {
  width: 45%;
}

.sk-line--meta {
  width: 25%;
  height: 22rpx;
}

@keyframes sk-shine {
  from {
    transform: translateX(-100%);
  }

  to {
    transform: translateX(100%);
  }
}

/* 失败 / 空 */
.kg-state {
  flex-direction: column;
  gap: $spacing-sm;
  padding: $spacing-2xl $spacing-xl;
  text-align: center;
}

.state-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 96rpx;
  height: 96rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
}

.state-badge.is-danger {
  background: $color-semantic-danger-bg;
}

.state-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  color: $color-brand-primary-dark;
}

.state-badge.is-danger .state-glyph {
  color: $color-semantic-danger;
}

.state-title {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  color: $color-text-primary;
}

.state-desc {
  font-size: $typography-font-size-sm;
  line-height: 1.6;
  color: $color-text-muted;
}

.state-btn {
  margin-top: $spacing-sm;
  padding: 0 $spacing-xl;
  font-size: $typography-font-size-sm;
  line-height: 68rpx;
  color: $color-bg-card;
  background: $color-brand-primary;
  border-radius: $border-radius-full;
}

.state-btn::after {
  border: none;
}
</style>
