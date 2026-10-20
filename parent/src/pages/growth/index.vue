<script setup>
import { computed, ref } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchGrowthFeed, likeGrowth, unlikeGrowth } from "@/api/parent-growth"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { fetchAchievements } from "@/api/parent-points"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import BottomNav from "@/components/BottomNav.vue"
import StateBlock from "@/components/StateBlock.vue"

const feeds = ref([])
const loading = ref(false)
const loadError = ref("")
const viewMode = ref("child")
const currentStudentId = ref(getCurrentStudentId())

const childInfo = ref({
  name: "",
  class: "",
  age: "",
  avatar: "",
  // stats 各项来自真实接口：records=feed.total，medals=成就 earned_count。
  // photos / attendance 尚无可靠来源，暂留 0，不要在这里写死假数字。
  stats: { records: 0, photos: 0, medals: 0, attendance: "" },
})

function goDetail(id) { uni.navigateTo({ url: `/pages/growth/detail?id=${id}` }) }
function goAlbum() { uni.navigateTo({ url: "/pages/growth/album" }) }
function goReport() { uni.navigateTo({ url: "/pages/growth/report" }) }

// 月份分组由 item.date 派生（design D4）：接口不返回 month 字段，避免冗余派生值。
// 后端已按 date DESC 排序，相邻同月记录自然聚在一起。
function monthLabelOf(ym) {
  const [y, m] = ym.split("-")
  return `${y}年${Number(m)}月`
}

// 列表提供 scope（individual/class）。个体记录的所有照片都属于当前孩子，
// 不按照片点名过滤；全班记录才按 media[].student_ids 过滤。
// scope 缺失（旧后端/回滚）时退回照片点名过滤，保持可用。
function mediaBelongsToChild(media) {
  const sid = currentStudentId.value
  if (!sid || !media) return false
  return Array.isArray(media.student_ids) && media.student_ids.includes(sid)
}

function mediaFor(item, mode = viewMode.value) {
  const media = Array.isArray(item.media) ? item.media : []
  if (mode === "all" || media.length === 0) return media
  if (item.scope === "individual") return media
  // 当前孩子未知时不过滤，避免整页被误判为空
  if (!currentStudentId.value) return media
  return media.filter(mediaBelongsToChild)
}

function isVisibleInChildView(item) {
  if (item.scope === "individual") return true
  const media = Array.isArray(item.media) ? item.media : []
  // 纯文字记录没有照片归属问题，默认视图保留
  if (media.length === 0) return true
  if (!currentStudentId.value) return true
  return media.some(mediaBelongsToChild)
}

const visibleFeeds = computed(() => {
  if (viewMode.value === "all") return feeds.value
  return feeds.value.filter(isVisibleInChildView)
})

const childFeedCount = computed(() => feeds.value.filter(isVisibleInChildView).length)
const allFeedCount = computed(() => feeds.value.length)
const childPhotoCount = computed(() =>
  feeds.value.reduce((total, item) => total + mediaFor(item, "child").length, 0)
)
const allPhotoCount = computed(() =>
  feeds.value.reduce((total, item) => total + mediaFor(item, "all").length, 0)
)

const viewHint = computed(() =>
  viewMode.value === "child"
    ? `只显示与${childInfo.value.name || "孩子"}确认相关的内容；切换「看全部」可浏览班级动态`
    : "包含班级其他孩子的动态；切换「只看孩子」可聚焦当前孩子"
)

const modeEmptyTitle = computed(() =>
  viewMode.value === "child"
    ? `还没有与${childInfo.value.name || "孩子"}相关的成长记录`
    : "班级还没有发布成长记录"
)

const modeEmptyText = computed(() =>
  viewMode.value === "child"
    ? "可以先看看班级动态，或等老师发布新的记录。"
    : "老师发布后会出现在这里。"
)

function switchView(mode) {
  if (viewMode.value === mode) return
  viewMode.value = mode
}

// 个体记录由列表 scope 明确，不再靠照片点名猜测；全班记录才按照片点名标归属。
function ownerKind(item) {
  if (item.scope === "individual") return "individual"
  if (!currentStudentId.value) return "plain"
  const media = Array.isArray(item.media) ? item.media : []
  if (media.length === 0) return "plain"
  // 没有点名信息的全班照片不能断言是「其他孩子」。
  const hasAnyTag = media.some(
    (m) => Array.isArray(m.student_ids) && m.student_ids.length > 0
  )
  if (!hasAnyTag) return "plain"
  const tagged = media.filter(mediaBelongsToChild)
  if (tagged.length === 0) return "other"
  if (tagged.length === media.length) return "personal"
  return "collective"
}

function ownerLabel(item) {
  const kind = ownerKind(item)
  if (kind === "individual") return "专属记录"
  if (kind === "personal") return "含孩子"
  if (kind === "collective") return "集体 · 含孩子"
  if (kind === "other") return "其他孩子"
  return "班级记录"
}

function hiddenPhotoCount(item) {
  if (viewMode.value !== "child" || item.scope === "individual") return 0
  const all = Array.isArray(item.media) ? item.media.length : 0
  return Math.max(0, all - mediaFor(item, "child").length)
}

// 月份标记按 id 单独映射，模板遍历 visibleFeeds。
// visibleFeeds 返回源对象本身（不复制），点赞等原地更新仍可渲染；
// 旧写法返回 { ...item } 副本导致点赞不刷新，不要再改回去。
const monthLabels = computed(() => {
  const labels = {}
  let lastMonth = ""
  for (const item of visibleFeeds.value) {
    const ym = (item.date || "").slice(0, 7)
    labels[item.id] = ym !== "" && ym !== lastMonth ? monthLabelOf(ym) : ""
    lastMonth = ym
  }
  return labels
})

// 与详情页 handleLike 同口径：调接口后以服务端返回的 is_liked 为准，不做乐观翻转。
// 列表契约不含 likes_count（2026-10 随字段审查移除），故只回写 is_liked。
async function toggleLike(item) {
  try {
    const r = item.is_liked ? await unlikeGrowth(item.id) : await likeGrowth(item.id)
    // 模板直接遍历 feeds，但小程序渲染层对「深层属性原地赋值」的 setData diff 不可靠。
    // 整体替换数组最稳：计算属性必然重算、渲染层必然拿到新值。
    feeds.value = feeds.value.map(f =>
      f.id === item.id ? { ...f, is_liked: r.data?.is_liked } : f
    )
  } catch (err) {
    uni.showToast({ title: err?.message || "操作失败", icon: "none" })
  }
}

// 徽章配色按 color_kind 映射到 design token 语义色（design D3：色值不落库）。
// 只认 warning/info 两族，其余回退 info——避免未知值渲染成无样式。
function badgeKind(badge) {
  return badge && badge.color_kind === "warning" ? "warning" : "info"
}

async function loadData() {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const sid = getCurrentStudentId()
    currentStudentId.value = sid || null
    // 成就数是次要指标，不能让它失败连累 feed 加载——单独吞错，失败时按 0 显示。
    const achievementsPromise = fetchAchievements(sid || undefined).catch(() => null)
    const [overviewRes, feedRes, achRes] = await Promise.all([
      fetchParentHomeOverview(sid),
      fetchGrowthFeed({ page: 1, page_size: 20, student_id: sid || undefined }),
      achievementsPromise,
    ])
    // 与首页同样承担自愈：存储为空时把服务端认定的当前孩子落盘
    syncCurrentStudentFromOverview(overviewRes.data)
    currentStudentId.value = getCurrentStudentId() || null
    const stu = overviewRes.data?.current_student
    if (stu) {
      childInfo.value = {
        name: stu.name || "",
        class: stu.class_name || "",
        age: "",
        avatar: stu.avatar || "",
        stats: {
          ...childInfo.value.stats,
          // 只填有真实来源的项，不留假数字
          records: feedRes.data?.total ?? 0,
          medals: achRes?.data?.earned_count ?? 0,
        },
      }
    }
    feeds.value = feedRes.data?.list || []
    // 原生导航栏即可承载「小明的成长时光」，不引入 navigationStyle: custom
    uni.setNavigationBarTitle({
      title: childInfo.value.name ? `${childInfo.value.name}的成长时光` : "成长时光",
    })
  } catch (err) {
    feeds.value = []
    // 错误态交给下方 StateBlock 内联渲染（http.js 已弹过一次全局 toast，此处不重复）
    loadError.value = err?.message || "加载成长时光失败"
    uni.setNavigationBarTitle({ title: "成长时光" })
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

// tabBar 页必须用 onShow：switchTab 切回来不重跑 onLoad，切完孩子会看不到变化
onShow(loadData)
onPullDownRefresh(loadData)
</script>

<template>
  <view class="page">
    <scroll-view scroll-y class="content-scroll" :refresher-enabled="true" @refresherrefresh="loadData">
      <!-- 孩子信息卡片 -->
      <view class="child-card">
        <view class="child-top">
          <view class="child-left">
            <image
              v-if="childInfo.avatar"
              :src="childInfo.avatar"
              class="child-avatar"
              mode="aspectFill"
            />
            <view v-else class="child-avatar-placeholder">
              <text>王</text>
            </view>
            <view class="child-meta">
              <text class="child-name">{{ childInfo.name }}</text>
              <text class="child-sub">{{ childInfo.class }} · {{ childInfo.age }}</text>
            </view>
          </view>
          <view class="export-btn">
            <text>导出相册</text>
          </view>
        </view>
        <view class="child-stats">
          <view class="stat-item">
            <text class="stat-num">{{ childInfo.stats.records }}</text>
            <text class="stat-label">成长记录</text>
          </view>
          <view class="stat-item">
            <text class="stat-num">{{ childInfo.stats.photos }}</text>
            <text class="stat-label">精彩照片</text>
          </view>
          <view class="stat-item">
            <text class="stat-num">{{ childInfo.stats.medals }}</text>
            <text class="stat-label">获得奖章</text>
          </view>
          <view class="stat-item">
            <text class="stat-num">{{ childInfo.stats.attendance }}</text>
            <text class="stat-label">出勤率</text>
          </view>
        </view>
      </view>

      <!-- 查看方式：只看孩子（默认）/ 看全部 -->
      <view class="view-switch">
        <view class="view-tabs" role="tablist" aria-label="查看方式">
          <view
            class="view-tab"
            :class="{ 'view-tab--active': viewMode === 'child' }"
            role="tab"
            :aria-selected="viewMode === 'child'"
            @click="switchView('child')"
          >
            <text class="view-tab-label">只看孩子</text>
            <text class="view-tab-count">{{ childFeedCount }} 条 · {{ childPhotoCount }} 张</text>
          </view>
          <view
            class="view-tab"
            :class="{ 'view-tab--active': viewMode === 'all' }"
            role="tab"
            :aria-selected="viewMode === 'all'"
            @click="switchView('all')"
          >
            <text class="view-tab-label">看全部</text>
            <text class="view-tab-count">{{ allFeedCount }} 条 · {{ allPhotoCount }} 张</text>
          </view>
        </view>
        <text class="view-hint">{{ viewHint }}</text>
      </view>

      <!-- 时光轴标题 -->
      <view class="timeline-bar">
        <view class="tl-head">
          <view class="tl-title-row">
            <text class="tl-title">时光轴</text>
            <view class="tl-year-badge">
              <text>2026年</text>
            </view>
          </view>
          <text class="tl-count">{{ visibleFeeds.length }} 条</text>
        </view>
      </view>

      <!-- 成长记录列表 -->
      <!-- 状态：加载/错误/空三态合一。「错误先于空」的判定顺序由 StateBlock 内部固化 -->
      <StateBlock
        v-if="loading || loadError"
        :loading="loading"
        :error="loadError"
        loading-text="加载成长时光..."
        empty-text="老师还没有发布成长记录"
        @retry="loadData"
      />
      <StateBlock
        v-else-if="feeds.length === 0"
        :loading="false"
        :error="''"
        empty-text="老师还没有发布成长记录"
        :retryable="false"
      />
      <view v-else-if="visibleFeeds.length === 0" class="mode-empty">
        <text class="mode-empty-title">{{ modeEmptyTitle }}</text>
        <text class="mode-empty-text">{{ modeEmptyText }}</text>
        <view v-if="viewMode === 'child'" class="mode-empty-btn" @click="switchView('all')">
          <text>看全部班级动态</text>
        </view>
      </view>
      <view v-else class="record-list">
        <template v-for="item in visibleFeeds" :key="item.id">
          <!-- 月份分隔：由 item.date 派生（模板持有源对象，点赞等原地更新才可渲染） -->
          <view v-if="monthLabels[item.id]" class="month-marker">
            <view class="month-dot" />
            <text class="month-text">{{ monthLabels[item.id] }}</text>
          </view>

          <!-- 记录卡片 -->
          <view class="record-card" @click="goDetail(item.id)">
            <!-- 卡片头部 -->
            <view class="rc-head">
              <image
                v-if="item.creator && item.creator.avatar"
                :src="item.creator.avatar"
                class="rc-teacher-avatar"
                mode="aspectFill"
              />
              <view v-else class="rc-teacher-avatar rc-teacher-avatar--placeholder">
                <text class="rc-ta-text">&#xf256;</text>
              </view>
              <view class="rc-teacher-meta">
                <text class="rc-teacher-name">{{ item.creator && item.creator.nickname }}</text>
                <!-- 时间格式与 detail.vue / pickup 一致：date + created_at 的 HH:mm -->
                <text class="rc-time">{{ item.date }} · {{ item.created_at?.slice(11, 16) || "" }}</text>
              </view>
              <text class="rc-owner" :class="'rc-owner--' + ownerKind(item)">{{ ownerLabel(item) }}</text>
            </view>

            <!-- 获奖徽章：仅当本记录是某次获奖的见证时渲染（design D1：见证可空） -->
            <view v-if="item.badge" class="rc-badge-row">
              <view class="rc-badge" :class="'rc-badge--' + badgeKind(item.badge)">
                <image v-if="item.badge.icon" :src="item.badge.icon" class="rc-badge-icon" mode="aspectFit" />
                <text class="rc-badge-text">{{ item.badge.name }}</text>
              </view>
            </view>

            <!-- 内容 -->
            <text class="rc-content">{{ item.content }}</text>

            <!-- 照片网格：只看孩子时只展示已点名到当前孩子的照片 -->
            <view v-if="mediaFor(item).length > 0" class="rc-photos">
              <view
                v-for="m in mediaFor(item)"
                :key="m.id"
                class="rc-photo-box"
                :class="{ 'rc-photo-full': mediaFor(item).length === 1 }"
              >
                <image class="rc-photo-placeholder" :src="m.thumb_url || m.file_url" mode="aspectFill" />
              </view>
            </view>
            <view v-if="hiddenPhotoCount(item) > 0" class="rc-photo-note">
              <text>还有 {{ hiddenPhotoCount(item) }} 张班级照片，切到「看全部」查看</text>
            </view>

            <!-- 互动栏 -->
            <view class="rc-actions">
              <view class="rca-left">
                <view class="rca-item" @click.stop="toggleLike(item)">
                  <text class="rca-icon" :class="{ liked: item.is_liked }">
                    {{ item.is_liked ? '&#xee0f;' : '&#xee0e;' }}
                  </text>
                  <text class="rca-num" :class="{ liked: item.is_liked }">
                    {{ item.is_liked ? '已点赞' : '点赞' }}
                  </text>
                </view>
                <view class="rca-item">
                  <text class="rca-icon">&#xec3a;</text>
                  <text class="rca-num">{{ item.comments_count > 0 ? item.comments_count + '条评论' : '评论' }}</text>
                </view>
              </view>
              <view class="rca-item">
                <text class="rca-icon">&#xf37d;</text>
                <text class="rca-num">分享</text>
              </view>
            </view>
          </view>
        </template>
      </view>

      <view class="bottom-safe" />
    </scroll-view>

    <!-- 底部导航 -->
    <BottomNav :active="1" />
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  background: $color-bg-page;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow-x: hidden;
}

// ===== 内容区 =====
.content-scroll {
  flex: 1;
  padding: 0 28rpx;
  height: 0;
  box-sizing: border-box;
  width: 100%;
}

// ===== 孩子信息卡片 =====
.child-card {
  background: $gradient-brand;
  border-radius: $border-radius-xl;
  padding: $component-card-padding;
  margin-bottom: $component-card-gap;
  margin-top: 16rpx;
  box-shadow: $shadow-md;
}

.child-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.child-left {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
}

.child-avatar,
.child-avatar-placeholder {
  width: 120rpx;
  height: 120rpx;
  border-radius: $border-radius-full;
  // 品牌渐变上的白色描边属于装饰叠加，tokens.json 无对应项
  border: 6rpx solid rgba(255,255,255,0.3);
}

.child-avatar-placeholder {
  // 渐变上的半透明白底属于装饰叠加，tokens.json 无对应项
  background: rgba(255,255,255,0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  text {
    font-size: $typography-font-size-3xl;
    font-weight: $typography-font-weight-bold;
    color: $color-text-on-brand;
  }
}

.child-meta {
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
}

.child-name {
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-bold;
  color: $color-text-on-brand;
}

.child-sub {
  font-size: $typography-font-size-md;
  color: $color-text-on-brand;
  opacity: 0.85;
}

.export-btn {
  padding: $spacing-xs 20rpx;
  border-radius: $border-radius-full;
  // 渐变上的半透明白底属于装饰叠加，tokens.json 无对应项
  background: rgba(255,255,255,0.18);
  text {
    font-size: $typography-font-size-base;
    font-weight: $typography-font-weight-medium;
    color: $color-text-on-brand;
  }
  &:active {
    opacity: 0.7;
  }
}

.child-stats {
  display: flex;
  justify-content: space-between;
  margin-top: 24rpx;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4rpx;
}

.stat-num {
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-bold;
  color: $color-text-on-brand;
}

.stat-label {
  font-size: $typography-font-size-sm;
  color: $color-text-on-brand;
  opacity: 0.8;
}

// ===== 查看方式 =====
.view-switch {
  margin-bottom: $component-card-gap;
}

.view-tabs {
  display: flex;
  gap: $spacing-xs;
  padding: $spacing-xs;
  background: $color-brand-primary-bg;
  border-radius: $border-radius-full;
}

.view-tab {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2rpx;
  min-height: 88rpx;
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius-full;
  &:active {
    opacity: 0.8;
  }
}

.view-tab--active {
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.view-tab-label {
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  color: $color-text-muted;
}

.view-tab--active .view-tab-label {
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
}

.view-tab-count {
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.view-tab--active .view-tab-count {
  color: $color-text-muted;
}

.view-hint {
  display: block;
  margin-top: $spacing-sm;
  padding: 0 $spacing-sm;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  color: $color-text-muted;
}

// ===== 时光轴筛选 =====
.timeline-bar {
  margin-bottom: 16rpx;
}

.tl-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24rpx;
}

.tl-count {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

.tl-title-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.tl-title {
  font-size: $typography-font-size-2xl;
  font-weight: 600;
  color: $color-text-primary;
}

.tl-year-badge {
  padding: 4rpx 16rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  text {
    font-size: $typography-font-size-sm;
    font-weight: 500;
    color: $color-brand-primary;
  }
}

// ===== 记录列表 =====
.mode-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: $spacing-lg;
  padding: $component-card-padding;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.mode-empty-title {
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  color: $color-text-primary;
}

.mode-empty-text {
  margin-top: $spacing-sm;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  color: $color-text-muted;
}

.mode-empty-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: $spacing-lg;
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  &:active {
    opacity: 0.8;
  }
}

.record-list {
  display: flex;
  flex-direction: column;
}

.month-marker {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  padding: $spacing-lg 0 20rpx;
}

.month-dot {
  width: 16rpx;
  height: 16rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.month-text {
  font-size: $typography-font-size-xl;
  font-weight: 600;
  color: $color-text-primary;
}

.record-card {
  background: $color-bg-card;
  border-radius: $component-card-radius;
  padding: $component-card-padding;
  box-shadow: $shadow-sm;
  margin-bottom: $component-card-gap;
  transition: transform $animation-duration-fast $animation-easing-default;
  &:active {
    transform: scale(0.995);
  }
}

// ===== 记录卡片头部 =====
.rc-head {
  display: flex;
  align-items: flex-start;
}

.rc-teacher-avatar {
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  overflow: hidden;
}

.rc-teacher-avatar--placeholder {
  background: $color-brand-primary-bg;
}

.rc-ta-text {
  font-size: $typography-font-size-3xl;
  color: $color-text-muted;
}

.rc-teacher-meta {
  flex: 1;
  margin-left: 20rpx;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.rc-teacher-name {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
}

.rc-time {
  font-size: $typography-font-size-sm;
  color: $color-text-muted;
}

// 照片归属标识：没有 target_type，不冒充「专属记录」。
.rc-owner {
  flex-shrink: 0;
  margin-left: auto;
  padding: $spacing-xs $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.3;
  white-space: nowrap;
}

.rc-owner--individual,
.rc-owner--personal {
  background: $color-brand-primary-bg;
  color: $color-brand-primary;
}

.rc-owner--individual {
  font-weight: $typography-font-weight-semibold;
}

.rc-owner--collective {
  background: $color-bg-page;
  color: $color-text-muted;
}

.rc-owner--other {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning;
}

.rc-owner--plain {
  background: $color-bg-page;
  color: $color-text-hint;
}

.rc-badge-row {
  display: flex;
  margin-top: $spacing-md;
}

// 获奖徽章：彩底 + 图标 + 文字（对齐设计稿「小小艺术家」/「全勤小明星」）。
// 色值只用 token 语义色，按 color_kind 分两族。
.rc-badge {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  padding: $spacing-xs $spacing-md;
  border-radius: $border-radius-full;
  flex-shrink: 0;
  align-self: flex-start;

  &--warning {
    background: $color-semantic-warning-bg;
    .rc-badge-text { color: $color-semantic-warning; }
  }
  &--info {
    background: $color-semantic-info-bg;
    .rc-badge-text { color: $color-semantic-info; }
  }
}

.rc-badge-icon {
  width: 32rpx;
  height: 32rpx;
  border-radius: $border-radius-sm;
}

.rc-badge-text {
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
}

// ===== 内容 =====
.rc-content {
  display: block;
  margin-top: 24rpx;
  font-size: $typography-font-size-md;
  color: $color-text-body;
  line-height: 1.65;
}

// ===== 照片 =====
.rc-photos {
  display: flex;
  gap: $spacing-lg;
  margin-top: 24rpx;
}

.rc-photo-box {
  width: 200rpx;
  height: 200rpx;
  border-radius: $border-radius-xl;
  overflow: hidden;
  flex-shrink: 0;
}

.rc-photo-full {
  width: 100%;
}

.rc-photo-placeholder {
  width: 100%;
  height: 100%;
  display: block;
  background: $color-bg-page;
}

.rc-photo-note {
  margin-top: $spacing-md;
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius-md;
  background: $color-bg-page;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  color: $color-text-muted;
}

// ===== 互动栏 =====
.rc-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 24rpx;
}

.rca-left {
  display: flex;
  align-items: center;
  gap: $spacing-xl;
}

.rca-item {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  transition: opacity $animation-duration-fast $animation-easing-default;
  &:active {
    opacity: 0.6;
  }
}

.rca-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  color: $color-text-hint;
}

.rca-num {
  font-size: $typography-font-size-base;
  color: $color-text-muted;
}

.rca-icon.liked, .rca-num.liked {
  color: $color-brand-primary;
}

.bottom-safe {
  height: calc(120rpx + env(safe-area-inset-bottom));
}
</style>
