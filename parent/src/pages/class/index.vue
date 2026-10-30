<script setup>
/**
 * 班级信息 · 班级名片
 *
 * 一屏稀疏数据做成一张名片：徽记 + 班级名 + 学年 + 在班人数，
 * 下面是一份安静的老师名单。班级信息由园所维护，本页只读；
 * 唯一可恢复的动作是「重试 / 重新加载」，所以四态必须都落到真实数据分支上：
 * 空态 = 老师列表为空（班级本身存在），不是接口失败。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId } from "@/utils/currentStudent"
import { fetchClassInfo } from "@/api/parent-class"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  user: "\u{f264}", // user-line
  album: "\u{ee4b}", // image-line
  notice: "\u{ef94}", // notification-3-line
  chevron: "\u{ea6e}", // arrow-right-s-line
  alert: "\u{eca1}" // error-warning-line
}

const info = ref({})
const loading = ref(false)
const retrying = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")

const teachers = computed(() => info.value.teachers || [])
const className = computed(() => info.value.name || "班级")
const emblem = computed(() => {
  const chars = Array.from(String(info.value.name || "").trim())
  return chars.slice(0, 2).join("") || "班"
})
// 年级与学年都可能为空：缺一项就少一段，不留孤零零的分隔符。
const metaLine = computed(() => {
  const parts = []
  if (info.value.grade) parts.push(info.value.grade)
  if (info.value.academic_year) parts.push(`${info.value.academic_year} 学年`)
  return parts.join(" · ")
})
// 人数缺失时用「—」而不是 0：把「没拿到」读成「真的没人」是两种不同的错误。
const countText = computed(() => {
  const raw = info.value.student_count
  return raw === 0 || raw ? String(raw) : "—"
})

// 首帧（onLoad 之前的第一次渲染）也走骨架：此时既没有数据也没有错误。
const showSkeleton = computed(() => !hasLoaded.value && !loadError.value)
const showError = computed(() => Boolean(loadError.value) && !loading.value)

function initialOf(name) {
  const chars = Array.from(String(name || "").trim())
  return chars.length ? chars[0] : "师"
}

async function loadData(options = {}) {
  const keepView = Boolean(options.keepView)
  if (!keepView) {
    loading.value = true
    loadError.value = ""
  }
  try {
    await ensureLogin()
    const res = await fetchClassInfo(getCurrentStudentId() || undefined)
    info.value = res.data || {}
    hasLoaded.value = true
    // 重试失败时保留原错误文案，只有真正拿到数据才清掉整页错误态。
    loadError.value = ""
  } catch (err) {
    console.warn("[class-index] load failed:", err?.message || err)
    loadError.value = "班级信息加载失败"
  } finally {
    loading.value = false
  }
}

async function handleRetry() {
  if (retrying.value || loading.value) return
  retrying.value = true
  await loadData({ keepView: true })
  retrying.value = false
  uni.showToast({
    title: loadError.value ? "仍无法加载，请稍后再试" : "已重新加载",
    icon: "none"
  })
}

// 两个入口都不是 tabBar 页（tabBar 只有首页/成长/接送/我的），用 navigateTo。
function goAlbum() {
  uni.navigateTo({ url: "/pages/growth/album" })
}

function goNotice() {
  uni.navigateTo({ url: "/pages/notice/index" })
}

onLoad(() => {
  loadData()
})
</script>

<template>
  <PageShell max-width="430px" bottom-space="48rpx">
    <!-- ===== 加载：名片骨架 + 3 行老师骨架，入口隐藏 ===== -->
    <template v-if="showSkeleton">
      <view class="identity-card">
        <view class="identity-skeleton">
          <view class="sk sk-seal" />
          <view class="sk-copy">
            <view class="sk sk-line sk-line--title" />
            <view class="sk sk-line sk-line--meta" />
          </view>
          <view class="sk sk-number" />
        </view>
      </view>

      <view class="section">
        <view class="section-head">
          <text class="section-title">班级老师</text>
        </view>
        <view class="group-card">
          <view v-for="i in 3" :key="i" class="teacher-skeleton-row">
            <view class="sk sk-avatar" />
            <view class="sk-teacher-copy">
              <view class="sk sk-line sk-line--name" />
              <view class="sk sk-line sk-line--phone" />
            </view>
          </view>
        </view>
      </view>

      <text class="page-footer">班级信息由园所统一维护</text>
    </template>

    <!-- ===== 整页失败 ===== -->
    <view v-else-if="showError" class="error-state">
      <view class="error-icon">
        <text class="error-glyph">{{ ICON.alert }}</text>
      </view>
      <text class="error-title">班级信息加载失败</text>
      <text class="error-desc">网络不太稳定，请稍后再试</text>
      <view
        class="state-action"
        :class="{ 'is-busy': retrying }"
        :aria-disabled="retrying ? 'true' : 'false'"
        @click="handleRetry"
      >
        <text class="state-action-text">{{ retrying ? "重试中…" : "重试" }}</text>
      </view>
    </view>

    <!-- ===== 正常 / 老师为空 ===== -->
    <template v-else>
      <view class="identity-card">
        <view class="identity-real">
          <view class="identity-lead">
            <view class="class-emblem">
              <text class="class-emblem-text">{{ emblem }}</text>
            </view>
            <view class="identity-copy">
              <text class="class-name">{{ className }}</text>
              <text v-if="metaLine" class="class-meta">{{ metaLine }}</text>
            </view>
          </view>
          <view class="class-size">
            <text class="class-size-label">在班</text>
            <text class="class-size-value">{{ countText }}<text class="class-size-unit">人</text></text>
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-head">
          <text class="section-title">班级老师</text>
          <text v-if="teachers.length" class="section-meta">{{ teachers.length }} 位</text>
        </view>
        <view class="group-card">
          <view v-if="teachers.length" class="teacher-list">
            <view v-for="t in teachers" :key="t.id" class="teacher-row">
              <view class="teacher-avatar">
                <image v-if="t.avatar" class="teacher-avatar-img" :src="t.avatar" mode="aspectFill" />
                <text v-else class="teacher-avatar-initial">{{ initialOf(t.nickname) }}</text>
              </view>
              <view class="teacher-copy">
                <view class="teacher-head">
                  <text class="teacher-name">{{ t.nickname || "老师" }}</text>
                  <view
                    v-if="t.title_label"
                    class="role-pill"
                    :class="{ 'is-lead': t.title_label === '主班老师' }"
                  >
                    <text class="role-pill-text">{{ t.title_label }}</text>
                  </view>
                </view>
                <!-- 契约允许 phone 为空：空字段整行不显示，不留空白行 -->
                <text v-if="t.phone" class="teacher-phone">{{ t.phone }}</text>
              </view>
            </view>
          </view>

          <view v-else class="teacher-empty">
            <view class="empty-icon">
              <text class="empty-glyph">{{ ICON.user }}</text>
            </view>
            <text class="empty-title">暂无老师信息</text>
            <text class="empty-desc">班级老师尚未配置，请稍后重新加载</text>
            <view
              class="state-action"
              :class="{ 'is-busy': retrying }"
              :aria-disabled="retrying ? 'true' : 'false'"
              @click="handleRetry"
            >
              <text class="state-action-text">{{ retrying ? "重试中…" : "重新加载" }}</text>
            </view>
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-head">
          <text class="section-title">班级相关</text>
        </view>
        <view class="group-card">
          <view class="entry-cell" @click="goAlbum">
            <view class="entry-icon">
              <text class="entry-glyph">{{ ICON.album }}</text>
            </view>
            <text class="entry-title">成长相册</text>
            <text class="entry-arrow">{{ ICON.chevron }}</text>
          </view>
          <view class="entry-cell" @click="goNotice">
            <view class="entry-icon">
              <text class="entry-glyph">{{ ICON.notice }}</text>
            </view>
            <text class="entry-title">通知公告</text>
            <text class="entry-arrow">{{ ICON.chevron }}</text>
          </view>
        </view>
      </view>

      <text class="page-footer">班级信息由园所统一维护</text>
    </template>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 班级名片：白卡 + 标题层级；暖色只落在徽记、主班胶囊、
// 入口箭头与主按钮上，电话降为中性灰。
// ============================================================

// ---- 班级名片 ------------------------------------------------
.identity-card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.identity-real {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  min-width: 0;
}

.identity-lead {
  display: flex;
  flex: 1;
  align-items: center;
  gap: $spacing-lg;
  min-width: 0;
}

// 徽记取班级名前两字：与班级名形成主次，不是第二份名称。
.class-emblem {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 112rpx;
  height: 112rpx;
  border: 1rpx solid $color-brand-primary-light;
  border-radius: $border-radius-2xl;
  background: $gradient-brand-soft;
}

.class-emblem-text {
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.identity-copy {
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
  min-width: 0;
}

.class-name {
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.class-meta {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.class-size {
  display: flex;
  flex: none;
  align-self: stretch;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-width: 112rpx;
  padding-left: $spacing-lg;
  border-left: 1rpx solid $color-border-hairline;
}

.class-size-label {
  margin-bottom: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.class-size-value {
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  font-variant-numeric: tabular-nums;
  line-height: 1.1;
}

.class-size-unit {
  margin-left: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

// ---- 区块标题 ------------------------------------------------
.section {
  margin-top: calc(#{$spacing-xl} + #{$spacing-md});
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  margin: 0 4rpx $spacing-md;
}

.section-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.section-meta {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

// ---- 老师名单 ------------------------------------------------
.group-card {
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  overflow: hidden;
}

.teacher-row {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-width: 0;
  padding: $spacing-xl;
}

.teacher-row + .teacher-row {
  border-top: 1rpx solid $color-border-hairline;
}

.teacher-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: $component-avatar-md;
  height: $component-avatar-md;
  overflow: hidden;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.teacher-avatar-img {
  width: 100%;
  height: 100%;
}

.teacher-avatar-initial {
  color: $color-text-body;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.teacher-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-sm;
  min-width: 0;
}

.teacher-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-md;
  min-width: 0;
}

.teacher-name {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.role-pill {
  display: inline-flex;
  align-items: center;
  flex: none;
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.role-pill-text {
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

// 主班用极浅暖底 + 小圆点区分，不靠颜色说「等级」。
.role-pill.is-lead {
  gap: $spacing-xs;
  background: $color-brand-primary-bg;
}

.role-pill.is-lead .role-pill-text {
  color: $color-text-primary;
}

.role-pill.is-lead::before {
  width: 8rpx;
  height: 8rpx;
  flex: none;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
  content: "";
}

.teacher-phone {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  font-variant-numeric: tabular-nums;
  line-height: 1.4;
  overflow-wrap: break-word;
}

// ---- 班级相关入口 --------------------------------------------
.entry-cell {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-height: 112rpx;
  padding: $spacing-md $spacing-xl;

  &:active {
    background: $color-bg-page;
  }
}

.entry-cell + .entry-cell {
  border-top: 1rpx solid $color-border-hairline;
}

.entry-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 56rpx;
  height: 56rpx;
}

.entry-glyph {
  color: $color-text-body;
  font-family: remixicon;
  font-size: $typography-font-size-3xl;
  line-height: 1;
}

.entry-title {
  flex: 1;
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.entry-arrow {
  flex: none;
  color: $color-brand-primary-dark;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

// ---- 页脚 ----------------------------------------------------
.page-footer {
  display: block;
  margin-top: calc(#{$spacing-xl} + #{$spacing-xs});
  padding: 0 $spacing-xl;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.6;
  text-align: center;
}

// ---- 老师为空 ------------------------------------------------
.teacher-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: calc(#{$spacing-xl} + #{$spacing-lg}) $spacing-xl $spacing-xl;
  text-align: center;
}

.empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  margin-bottom: $spacing-lg;
  border: 1rpx solid $color-border-hairline;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.empty-glyph {
  color: $color-text-muted;
  font-family: remixicon;
  font-size: 52rpx;
  line-height: 1;
}

.empty-title {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.empty-desc {
  max-width: 460rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

// ---- 主按钮（重试 / 重新加载） --------------------------------
.state-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 224rpx;
  min-height: 88rpx;
  margin-top: $spacing-xl;
  padding: 0 calc(#{$spacing-xl} + #{$spacing-xs});
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.98);
  }
}

// 白字压品牌橙不足 4.5:1，主按钮按设计稿用深色字。
.state-action-text {
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.state-action.is-busy {
  background: $color-border-hairline;
  box-shadow: none;

  &:active {
    transform: none;
  }
}

.state-action.is-busy .state-action-text {
  color: $color-text-hint;
}

// ---- 加载骨架 ------------------------------------------------
.sk {
  display: block;
  border-radius: $border-radius-full;
  background: $color-border-hairline;
  animation: sk-breathe 1.5s $animation-easing-default infinite alternate;
}

.identity-skeleton {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  min-width: 0;
}

.sk-seal {
  flex: none;
  width: 112rpx;
  height: 112rpx;
  border-radius: $border-radius-2xl;
}

.sk-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-md;
  min-width: 0;
}

.sk-line {
  height: 20rpx;
}

.sk-line--title {
  width: 58%;
  max-width: 224rpx;
  height: 28rpx;
}

.sk-line--meta {
  width: 78%;
  max-width: 296rpx;
}

.sk-number {
  flex: none;
  width: 104rpx;
  height: 104rpx;
  border-radius: $border-radius-xl;
}

.teacher-skeleton-row {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: $spacing-xl;
}

.teacher-skeleton-row + .teacher-skeleton-row {
  border-top: 1rpx solid $color-border-hairline;
}

.sk-avatar {
  flex: none;
  width: $component-avatar-md;
  height: $component-avatar-md;
  border-radius: $border-radius-full;
}

.sk-teacher-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: $spacing-md;
  min-width: 0;
}

.sk-line--name {
  width: 144rpx;
  max-width: 40%;
  height: 24rpx;
}

.sk-line--phone {
  width: 208rpx;
  max-width: 56%;
  height: 18rpx;
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
  width: 128rpx;
  height: 128rpx;
  margin-bottom: $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
}

.error-glyph {
  color: $color-semantic-danger;
  font-family: remixicon;
  font-size: 56rpx;
  line-height: 1;
}

.error-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.error-desc {
  max-width: 480rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.error-state .state-action {
  margin-top: $spacing-xl;
}

@keyframes sk-breathe {
  from {
    opacity: 0.45;
  }
  to {
    opacity: 0.9;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
  }

  .entry-cell,
  .state-action {
    transition: none;
  }
}
</style>
