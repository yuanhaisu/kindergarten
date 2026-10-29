<script setup>
import { computed, ref } from "vue"
import { onShow, onPullDownRefresh } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchGrowthFeed, likeGrowth, unlikeGrowth } from "@/api/parent-growth"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { fetchAchievements } from "@/api/parent-points"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { syncTabBar } from "@/utils/tabbar"
import { formatMonthLabel, formatRecordTime, formatYearRange, monthKeyOf } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  heart: "", // heart-line
  heartFill: "", // heart-fill
  comment: "", // chat-3-line
  share: "", // share-line
  medal: "", // medal-line
  photos: "", // image-line
  download: "", // download-2-line
  leaf: "", // leaf-line
  error: "", // error-warning-line
  refresh: "" // refresh-line
}

const feeds = ref([])
const loading = ref(false)
const refreshing = ref(false)
const retrying = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const viewMode = ref("child")
const currentStudentId = ref(getCurrentStudentId())

const childInfo = ref({
  name: "",
  class: "",
  avatar: "",
  // 只有 records / medals 有真实来源；photos / attendance 无数据源，
  // 页面显示「—」而不是 0，避免把「暂无数据」读成「真的是 0」。
  stats: { records: 0, medals: 0 }
})

// 每次 loadData 自增，只允许最后一次请求写数据：切孩子与 onShow 刷新可能并发。
let loadSeq = 0

const isFirstLoading = computed(() => loading.value && !hasLoaded.value)
const showError = computed(() => Boolean(loadError.value) && feeds.value.length === 0)
// 空态不带 !loading：空列表下拉刷新时保留空态，而不是先闪一段空白
const showEmpty = computed(
  () => hasLoaded.value && !showError.value && feeds.value.length === 0
)
const pageLocked = computed(() => isFirstLoading.value || showError.value)

const stats = computed(() => [
  { key: "records", label: "成长记录", value: String(childInfo.value.stats.records || 0) },
  { key: "photos", label: "精彩照片", value: "—" },
  { key: "medals", label: "获得奖章", value: String(childInfo.value.stats.medals || 0) },
  { key: "attendance", label: "出勤率", value: "—" }
])

function goDetail(id) {
  uni.navigateTo({ url: `/pages/growth/detail?id=${id}` })
}

// 「导出相册」胶囊按设计位置承担相册入口（design 09 的 js-export）。
function goAlbum() {
  if (pageLocked.value) return
  uni.navigateTo({ url: "/pages/growth/album" })
}

// 列表 scope（individual/class）由服务端给出，个体记录的照片都属于当前孩子，
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
  if (!currentStudentId.value) return media
  return media.filter(mediaBelongsToChild)
}

function isVideo(m) {
  return m && m.file_type === "video"
}

function photoList(item) {
  return mediaFor(item).filter(m => !isVideo(m))
}

function videoList(item) {
  return mediaFor(item).filter(isVideo)
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

// 分段控件计数描述的是「本页已加载的记录」，与孩子卡的全量总数是两个口径，
// 所以文案带「本页」前缀，避免「成长记录 6」和「5 条」被读成自相矛盾。
const childFeedCount = computed(() => feeds.value.filter(isVisibleInChildView).length)
const allFeedCount = computed(() => feeds.value.length)
const childPhotoCount = computed(() =>
  feeds.value.reduce((total, item) => total + mediaFor(item, "child").filter(m => !isVideo(m)).length, 0)
)
const allPhotoCount = computed(() =>
  feeds.value.reduce((total, item) => total + mediaFor(item, "all").filter(m => !isVideo(m)).length, 0)
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
  if (pageLocked.value || viewMode.value === mode) return
  viewMode.value = mode
}

// 月分组：服务端已按 date DESC 排序，这里按「首次出现顺序」聚合同月记录，
// 避免用行内 marker 时同月被非相邻记录切碎。
const monthGroups = computed(() => {
  const groups = new Map()
  for (const item of visibleFeeds.value) {
    const key = monthKeyOf(item.date)
    if (!groups.has(key)) {
      groups.set(key, { key, label: formatMonthLabel(key), records: [] })
    }
    groups.get(key).records.push(item)
  }
  return [...groups.values()]
})

// 年份不写死：覆盖多年时显示区间，只有一年时显示该年。
const yearLabel = computed(() => formatYearRange(visibleFeeds.value.map(item => item.date)))

// 个体记录由列表 scope 明确，不再靠照片点名猜测；全班记录才按照片点名标归属。
function ownerKind(item) {
  if (item.scope === "individual") return "individual"
  if (!currentStudentId.value) return "plain"
  const media = Array.isArray(item.media) ? item.media : []
  if (media.length === 0) return "plain"
  const hasAnyTag = media.some(m => Array.isArray(m.student_ids) && m.student_ids.length > 0)
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
  const all = (Array.isArray(item.media) ? item.media : []).filter(m => !isVideo(m)).length
  return Math.max(0, all - photoList(item).length)
}

function creatorInitial(item) {
  const name = String(item?.creator?.nickname || "")
  return name ? name.slice(0, 1) : "师"
}

// 徽章配色按 color_kind 映射到 design token 语义色（design D3：色值不落库）。
function badgeKind(badge) {
  return badge && badge.color_kind === "warning" ? "warning" : "info"
}

// 与详情页 handleLike 同口径：调接口后以服务端返回的 is_liked 为准，不做乐观翻转。
// 同一记录在途时忽略后续点击：快速双击会并发发两个 POST，先返回的 500 会弹「操作失败」，
// 与最终已点赞的状态矛盾（服务端已做幂等，这里只是避免无谓请求和误导性提示）。
const likePending = new Set()

async function toggleLike(item) {
  if (pageLocked.value || likePending.has(item.id)) return
  likePending.add(item.id)
  try {
    const r = item.is_liked ? await unlikeGrowth(item.id) : await likeGrowth(item.id)
    const next = r?.data?.is_liked
    // 缺少布尔值时不静默翻转成 false —— 宁可报错也不把未知状态当取消点赞。
    if (typeof next !== "boolean") throw new Error("点赞状态未更新")
    // 小程序渲染层对「深层属性原地赋值」的 setData diff 不可靠，整体替换数组最稳。
    feeds.value = feeds.value.map(f => (f.id === item.id ? { ...f, is_liked: next } : f))
  } catch (err) {
    console.warn("[growth-index] like failed:", err?.message || err)
    uni.showToast({ title: err?.message || "操作失败", icon: "none" })
  } finally {
    likePending.delete(item.id)
  }
}

async function loadData() {
  const seq = ++loadSeq
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    const sid = getCurrentStudentId()
    currentStudentId.value = sid || null
    // 成就数是次要指标，失败不连累 feed——单独吞错，按 0 显示。
    const achievementsPromise = fetchAchievements(sid || undefined).catch(() => null)
    const [overviewRes, feedRes, achRes] = await Promise.all([
      fetchParentHomeOverview(sid),
      fetchGrowthFeed({ page: 1, page_size: 20, student_id: sid || undefined }),
      achievementsPromise
    ])
    if (seq !== loadSeq) return
    // 与首页同样承担自愈：存储为空时把服务端认定的当前孩子落盘
    syncCurrentStudentFromOverview(overviewRes.data)
    currentStudentId.value = getCurrentStudentId() || null
    const stu = overviewRes.data?.current_student
    if (stu) {
      childInfo.value = {
        name: stu.name || "",
        class: stu.class_name || "",
        avatar: stu.avatar || "",
        stats: {
          records: feedRes.data?.total ?? 0,
          medals: achRes?.data?.earned_count ?? 0
        }
      }
    }
    feeds.value = feedRes.data?.list || []
    hasLoaded.value = true
    uni.setNavigationBarTitle({
      title: childInfo.value.name ? `${childInfo.value.name}的成长时光` : "成长时光"
    })
  } catch (err) {
    if (seq !== loadSeq) return
    console.warn("[growth-index] load failed:", err?.message || err)
    // 刷新失败时保留已加载内容：只有「一句都还没拿到」才切到整页错误态。
    if (feeds.value.length === 0) {
      loadError.value = "加载成长时光失败"
      uni.setNavigationBarTitle({ title: "成长时光" })
    }
  } finally {
    if (seq === loadSeq) {
      loading.value = false
      uni.stopPullDownRefresh()
    }
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

async function handleRefresh() {
  if (loading.value) return
  await loadData()
  if (!loadError.value) uni.showToast({ title: "已刷新", icon: "none" })
}

async function onRefresher() {
  refreshing.value = true
  await loadData()
  refreshing.value = false
}

// tabBar 页必须用 onShow：switchTab 切回来不重跑 onLoad，切完孩子会看不到变化。
// 同时把原生底栏高亮校正到「成长」（index 1）——组件内算不准当前 tab，见 utils/tabbar.js。
onShow(() => {
  syncTabBar(1)
  loadData()
})
onPullDownRefresh(loadData)
</script>

<template>
  <PageShell flush-top max-width="430px" :safe-bottom="false">
    <scroll-view
      scroll-y
      class="feed-scroll"
      :refresher-enabled="true"
      :refresher-triggered="refreshing"
      @refresherrefresh="onRefresher"
    >
      <!-- ===== 孩子卡：整页唯一暖色块 ===== -->
      <view class="child-card">
        <view class="child-top">
          <view class="child-avatar">
            <image
              v-if="childInfo.avatar"
              class="child-avatar-img"
              :src="childInfo.avatar"
              mode="aspectFill"
            />
            <text v-else class="child-avatar-initial">{{ childInfo.name ? childInfo.name.slice(0, 1) : "?" }}</text>
          </view>
          <view class="child-info">
            <text class="child-name">{{ childInfo.name || "孩子" }}</text>
            <text class="child-class">{{ childInfo.class || "暂无班级" }}</text>
          </view>
          <view class="export-btn" :class="{ 'is-locked': pageLocked }" @click="goAlbum">
            <text class="export-icon">{{ ICON.download }}</text>
            <text class="export-label">导出相册</text>
          </view>
        </view>

        <view class="child-stats">
          <view v-for="st in stats" :key="st.key" class="stat">
            <view v-if="isFirstLoading" class="stat-skel">
              <view class="sk stat-skel-bar" />
              <view class="sk stat-skel-bar stat-skel-bar--short" />
            </view>
            <template v-else>
              <text class="stat-value" :class="{ 'is-dash': st.value === '—' }">{{ st.value }}</text>
              <text class="stat-label">{{ st.label }}</text>
            </template>
          </view>
        </view>
      </view>

      <!-- ===== 视角切换：空态/错误态下隐藏（设计稿覆盖状态表） ===== -->
      <view v-if="!showEmpty && !showError" class="view-switch">
        <view class="seg" :class="{ 'is-locked': pageLocked }">
          <view
            class="seg-btn"
            :class="{ 'is-active': viewMode === 'child' }"
            @click="switchView('child')"
          >
            <text class="seg-label">只看孩子</text>
            <text class="seg-meta">本页 {{ childFeedCount }} 条 · {{ childPhotoCount }} 张</text>
          </view>
          <view
            class="seg-btn"
            :class="{ 'is-active': viewMode === 'all' }"
            @click="switchView('all')"
          >
            <text class="seg-label">看全部</text>
            <text class="seg-meta">本页 {{ allFeedCount }} 条 · {{ allPhotoCount }} 张</text>
          </view>
        </view>
        <text class="seg-hint">{{ viewHint }}</text>
      </view>

      <!-- ===== 时光轴 ===== -->
      <view v-if="feeds.length > 0 && !showError" class="timeline">
        <view class="timeline-head">
          <view class="timeline-title-wrap">
            <text class="timeline-title">时光轴</text>
            <view class="year-pill">
              <text>{{ yearLabel }}</text>
            </view>
          </view>
          <text class="timeline-count">{{ visibleFeeds.length }} 条</text>
        </view>

        <!-- 视角筛选后为空：保留标题与「0 条」，用提示卡代替空白列表 -->
        <view v-if="visibleFeeds.length === 0" class="mode-empty">
          <text class="mode-empty-title">{{ modeEmptyTitle }}</text>
          <text class="mode-empty-text">{{ modeEmptyText }}</text>
        </view>

        <view v-else class="months">
          <view v-for="group in monthGroups" :key="group.key" class="month-group">
            <view class="month-head">
              <view class="month-dot" />
              <text class="month-label">{{ group.label }}</text>
              <view class="month-line" />
            </view>

            <view class="month-records">
              <view v-for="item in group.records" :key="item.id" class="rec" @click="goDetail(item.id)">
                <view class="rec-head">
                  <view class="rec-avatar">
                    <image
                      v-if="item.creator && item.creator.avatar"
                      class="rec-avatar-img"
                      :src="item.creator.avatar"
                      mode="aspectFill"
                    />
                    <text v-else class="rec-avatar-initial">{{ creatorInitial(item) }}</text>
                  </view>
                  <view class="rec-who">
                    <text class="rec-name">{{ (item.creator && item.creator.nickname) || "老师" }}</text>
                    <text class="rec-time">{{ formatRecordTime(item.date, item.created_at) }}</text>
                  </view>
                  <view class="scope" :class="'scope--' + ownerKind(item)">
                    <view class="scope-dot" />
                    <text>{{ ownerLabel(item) }}</text>
                  </view>
                </view>

                <!-- 获奖见证：仅当本记录是某次获奖的见证时渲染（design D1：见证可空） -->
                <view v-if="item.badge" class="award" :class="'award--' + badgeKind(item.badge)">
                  <text class="award-tag">获奖见证</text>
                  <text class="award-icon">{{ ICON.medal }}</text>
                  <text class="award-name">{{ item.badge.name }}</text>
                </view>

                <text v-if="item.content" class="rec-text">{{ item.content }}</text>

                <!-- 照片：≥2 张走 3 列网格换行，不裁切；单张按 4:3 铺满 -->
                <view
                  v-if="photoList(item).length"
                  class="photo-grid"
                  :class="{ 'photo-grid--single': photoList(item).length === 1 }"
                >
                  <view v-for="m in photoList(item)" :key="m.id" class="photo-cell">
                    <image class="photo-img" :src="m.thumb_url || m.file_url" mode="aspectFill" />
                  </view>
                </view>

                <!-- 视频：单独用 <video> 播放，不再当图片渲染 -->
                <video
                  v-for="m in videoList(item)"
                  :key="'v' + m.id"
                  class="rec-video"
                  :src="m.file_url"
                  :poster="m.thumb_url || ''"
                  controls
                  object-fit="contain"
                  @click.stop
                />

                <view v-if="hiddenPhotoCount(item) > 0" class="rec-more">
                  <text class="rec-more-icon">{{ ICON.photos }}</text>
                  <text class="rec-more-text">还有 {{ hiddenPhotoCount(item) }} 张班级照片，切到「看全部」查看</text>
                </view>

                <view class="rec-actions">
                  <view class="action-slot">
                    <view
                      class="action"
                      :class="{ 'is-liked': item.is_liked }"
                      @click.stop="toggleLike(item)"
                    >
                      <text class="action-icon">{{ item.is_liked ? ICON.heartFill : ICON.heart }}</text>
                      <text class="action-label">{{ item.is_liked ? "已点赞" : "点赞" }}</text>
                    </view>
                    <view class="action" @click.stop="goDetail(item.id)">
                      <text class="action-icon">{{ ICON.comment }}</text>
                      <text class="action-label">{{ item.comments_count > 0 ? item.comments_count + "条评论" : "评论" }}</text>
                    </view>
                  </view>
                  <!-- 源码没有分享能力：按设计审查 P2-6 的落点建议，分享回落到详情页 -->
                  <view class="action" @click.stop="goDetail(item.id)">
                    <text class="action-icon">{{ ICON.share }}</text>
                    <text class="action-label">分享</text>
                  </view>
                </view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <!-- ===== 加载态：三张骨架卡 ===== -->
      <view v-if="isFirstLoading" class="skeleton-list">
        <view v-for="i in 3" :key="i" class="skel-card">
          <view class="skel-head">
            <view class="sk skel-avatar" />
            <view class="skel-lines">
              <view class="sk skel-bar" />
              <view class="sk skel-bar skel-bar--short" />
            </view>
          </view>
          <view class="sk skel-photo" />
        </view>
      </view>

      <!-- ===== 空态：一条记录都没有 ===== -->
      <view v-if="showEmpty" class="empty-state">
        <view class="empty-orb">
          <text class="empty-icon">{{ ICON.leaf }}</text>
        </view>
        <text class="empty-title">老师还没有发布成长记录</text>
        <text class="empty-desc">等老师记录后，这里会显示{{ childInfo.name || "孩子" }}的成长时光</text>
        <view class="btn-primary" @click="handleRefresh">
          <text class="btn-icon">{{ ICON.refresh }}</text>
          <text>刷新看看</text>
        </view>
      </view>

      <!-- ===== 错误态：无缓存数据时才替换整页 ===== -->
      <view v-if="showError" class="error-state">
        <view class="error-card">
          <view class="error-orb">
            <text class="error-icon">{{ ICON.error }}</text>
          </view>
          <text class="error-title">{{ loadError }}</text>
          <view class="btn-primary" :class="{ 'is-busy': retrying }" @click="handleRetry">
            <text>{{ retrying ? "重试中…" : "重试" }}</text>
          </view>
        </view>
      </view>

      <!-- 原生 custom-tab-bar 固定覆盖在底部，这里留出等高占位 + 安全区 -->
      <view class="tabbar-space" />
    </scroll-view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 成长时光 · 暖橙时间轴上的成长流
// 冷雾白底；暖色只留在孩子卡与「已点赞」上。
// ============================================================

// scroll-view 自己占满视口，底部留白放在内容里（容器 padding 会裁掉最后一张卡）
.feed-scroll {
  height: 100vh;
  box-sizing: border-box;
}

.tabbar-space {
  height: calc(140rpx + env(safe-area-inset-bottom));
}

// ---- 孩子卡 ----------------------------------------------------
.child-card {
  margin-top: $spacing-lg;
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $gradient-brand-soft;
  box-shadow: $shadow-md;
}

.child-top {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
}

// 白底圆头像压在暖色卡上，是设计稿里唯一的「实心白块」
.child-avatar {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: $component-avatar-md;
  height: $component-avatar-md;
  overflow: hidden;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.child-avatar-img {
  width: 100%;
  height: 100%;
}

.child-avatar-initial {
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.child-info {
  flex: 1;
  min-width: 0;
}

.child-name {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.child-class {
  display: block;
  margin-top: 4rpx;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.export-btn {
  flex: none;
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  min-height: 60rpx;
  padding: 0 20rpx;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.96);
  }
}

.export-btn.is-locked {
  background: $color-border-subtle;
  color: $color-text-hint;
  box-shadow: none;
  pointer-events: none;
}

.export-icon {
  font-family: remixicon;
  font-size: $typography-font-size-base;
  line-height: 1;
  color: $color-brand-primary-dark;
}

.export-btn.is-locked .export-icon {
  color: $color-text-hint;
}

.export-label {
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
}

// 四项统计放在白色面板里，让孩子的名字与数字各自有落点
.child-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-top: $spacing-xl;
  padding: 20rpx 0;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  min-width: 0;
  min-height: 88rpx;
  padding: 0 $spacing-xs;
  text-align: center;

  & + & {
    border-left: 1rpx solid $color-border-subtle;
  }
}

.stat-value {
  color: $color-text-primary;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.15;
}

.stat-value.is-dash {
  color: $color-text-muted;
}

.stat-label {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.3;
}

.stat-skel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-xs;
}

.stat-skel-bar {
  width: 56rpx;
  height: 24rpx;
  border-radius: $border-radius-full;
}

.stat-skel-bar--short {
  width: 72rpx;
  height: 12rpx;
}

// ---- 加载呼吸 --------------------------------------------------
.sk {
  background: $color-border-subtle;
  animation: sk-breathe 1.8s $animation-easing-default infinite;
}

.sk-bar {
  width: 128rpx;
  height: 18rpx;
  border-radius: $border-radius-full;
}

.sk-bar--short {
  width: 208rpx;
  height: 16rpx;
}

// ---- 视角切换 --------------------------------------------------
.view-switch {
  margin-top: $spacing-lg;
}

.seg {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: $spacing-xs;
  padding: $spacing-xs;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
}

.seg.is-locked {
  background: $color-border-subtle;
  border-color: $color-border-subtle;
  pointer-events: none;
}

.seg-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2rpx;
  min-width: 0;
  min-height: 104rpx;
  padding: $spacing-sm $spacing-xs;
  border: 1rpx solid $color-bg-card;
  border-radius: $border-radius-full;
  text-align: center;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.985);
  }
}

.seg-btn.is-active {
  border-color: $color-brand-primary-light;
  background: $color-brand-primary-bg;
  box-shadow: $shadow-sm;
}

.seg.is-locked .seg-btn,
.seg.is-locked .seg-btn.is-active {
  background: $color-border-subtle;
  border-color: $color-border-subtle;
  box-shadow: none;
}

.seg-label {
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.seg-btn.is-active .seg-label {
  color: $color-text-primary;
}

.seg-meta {
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

// P2-3：说明性正文 11px → 13px（$typography-font-size-base），390px 视口下可读
.seg-hint {
  display: block;
  margin: $spacing-sm $spacing-xs 0;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

// ---- 时光轴 ----------------------------------------------------
.timeline {
  margin-top: $spacing-xl;
}

.timeline-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
}

.timeline-title-wrap {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-width: 0;
}

.timeline-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.year-pill {
  flex: none;
  display: inline-flex;
  align-items: center;
  height: 40rpx;
  padding: 0 $spacing-xs;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.timeline-count {
  flex: none;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.months {
  margin-top: $spacing-md;
}

.month-group + .month-group {
  margin-top: $spacing-xl;
}

.month-head {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: $spacing-md;
}

.month-dot {
  flex: none;
  width: 12rpx;
  height: 12rpx;
  border-radius: $border-radius-full;
  background: $color-text-muted;
}

.month-label {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.month-line {
  flex: 1;
  height: 1rpx;
  border-radius: $border-radius-full;
  background: $color-border-subtle;
}

.month-records {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

// ---- 记录卡 ----------------------------------------------------
.rec {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.995);
  }
}

.rec-head {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.rec-avatar {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  overflow: hidden;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.rec-avatar-img {
  width: 100%;
  height: 100%;
}

.rec-avatar-initial {
  color: $color-text-body;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.rec-who {
  flex: 1;
  min-width: 0;
}

.rec-name {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.rec-time {
  display: block;
  margin-top: 2rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

// 归属胶囊：左侧小圆点承担色相，文字统一用正文色保证对比度
.scope {
  flex: none;
  max-width: 48%;
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  padding: $spacing-xs $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
}

.scope-dot {
  flex: none;
  width: 10rpx;
  height: 10rpx;
  border-radius: $border-radius-full;
  background: $color-text-hint;
}

.scope--individual {
  background: $color-brand-primary-bg;

  .scope-dot {
    background: $color-brand-primary;
  }
}

.scope--personal {
  background: $color-semantic-success-bg;

  .scope-dot {
    background: $color-semantic-success;
  }
}

.scope--collective {
  background: $color-semantic-info-bg;

  .scope-dot {
    background: $color-semantic-info;
  }
}

.scope--plain,
.scope--other {
  background: $color-bg-page;
}

.scope--other {
  .scope-dot {
    background: $color-text-hint;
  }
}

// ---- 获奖见证 --------------------------------------------------
.award {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  min-height: 68rpx;
  margin-top: $spacing-md;
  padding: 10rpx $spacing-md;
  border-radius: $border-radius-lg;

  &--warning {
    background: $color-semantic-warning-bg;

    .award-icon {
      color: $color-semantic-warning;
    }
  }

  &--info {
    background: $color-semantic-info-bg;

    .award-icon {
      color: $color-semantic-info;
    }
  }
}

.award-tag {
  flex: none;
  display: inline-flex;
  align-items: center;
  height: 40rpx;
  padding: 0 $spacing-sm;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

.award-icon {
  flex: none;
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.award-name {
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

// ---- 正文 ------------------------------------------------------
.rec-text {
  display: -webkit-box;
  margin-top: $spacing-md;
  overflow: hidden;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.7;
  overflow-wrap: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 4;
}

// ---- 照片网格 --------------------------------------------------
.photo-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
  margin-top: $spacing-lg;
}

.photo-grid--single {
  grid-template-columns: minmax(0, 1fr);
}

// aspect-ratio 小程序不支持：用 padding-top 撑正方形，图片绝对定位铺满
.photo-cell {
  position: relative;
  width: 100%;
  padding-top: 100%;
  overflow: hidden;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.photo-grid--single .photo-cell {
  padding-top: 75%;
}

.photo-img {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

.rec-video {
  width: 100%;
  height: 360rpx;
  margin-top: $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-text-primary;
}

// P2-3：隐藏照片提示 11px → 13px
.rec-more {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  margin-top: $spacing-md;
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.rec-more-icon {
  flex: none;
  margin-top: 2rpx;
  color: $color-text-hint;
  font-family: remixicon;
  font-size: $typography-font-size-sm;
  line-height: 1;
}

.rec-more-text {
  min-width: 0;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.5;
}

// ---- 互动栏 ----------------------------------------------------
.rec-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: $spacing-lg;
  padding-top: $spacing-xs;
  border-top: 1rpx solid $color-border-subtle;
}

.action-slot {
  display: flex;
  align-items: center;
  min-width: 0;
}

.action {
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  min-height: 88rpx;
  padding: 0 $spacing-xs;
  border-radius: $border-radius-full;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.96);
  }
}

.action.is-liked {
  background: $color-brand-primary-bg;

  .action-icon,
  .action-label {
    color: $color-brand-primary-dark;
  }
}

.action-icon {
  font-family: remixicon;
  font-size: $typography-font-size-xl;
  line-height: 1;
}

.action-label {
  white-space: nowrap;
}

// ---- 骨架卡 ----------------------------------------------------
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  margin-top: $spacing-xl;
}

.skel-card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.skel-head {
  display: flex;
  align-items: center;
  gap: $spacing-md;
}

.skel-avatar {
  flex: none;
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
}

.skel-lines {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
}

.skel-bar {
  width: 128rpx;
  height: 18rpx;
}

.skel-bar--short {
  width: 208rpx;
  height: 16rpx;
}

.skel-photo {
  height: 192rpx;
  margin-top: $spacing-lg;
  border-radius: $border-radius-lg;
}

@keyframes sk-breathe {
  0%,
  100% {
    opacity: 0.45;
  }
  50% {
    opacity: 0.95;
  }
}

// ---- 空态 / 错误态 ---------------------------------------------
.empty-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 84rpx $spacing-xl $spacing-xl;
  text-align: center;
}

.empty-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 192rpx;
  height: 192rpx;
  margin-bottom: $spacing-xl;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.empty-icon {
  font-family: remixicon;
  font-size: 72rpx;
  line-height: 1;
}

.empty-title,
.error-title {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.empty-desc {
  max-width: 560rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.7;
}

.error-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  padding: 56rpx $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.error-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 96rpx;
  height: 96rpx;
  margin-bottom: $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
}

.error-icon {
  font-family: remixicon;
  font-size: $typography-font-size-3xl;
  line-height: 1;
}

// ---- 视角筛选为空的提示卡 --------------------------------------
.mode-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
  margin-top: $spacing-md;
  padding: 64rpx $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  text-align: center;
}

.mode-empty-title {
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.mode-empty-text {
  max-width: 560rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.7;
}

// ---- 主按钮 ----------------------------------------------------
.btn-primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  min-height: $component-input-height;
  margin-top: $spacing-xl;
  padding: 0 44rpx;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.2;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.97);
    box-shadow: $shadow-sm;
  }
}

.btn-icon {
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

.btn-primary.is-busy {
  opacity: 0.82;
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
    opacity: 0.7;
  }
}
</style>
