<script setup>
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import {
  fetchGrowthDetail,
  likeGrowth,
  unlikeGrowth,
  fetchGrowthComments,
  postGrowthComment
} from "@/api/parent-growth"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { formatCommentTime, formatRecordTime } from "@/utils/time"
import { relationLabel } from "@/utils/relation"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  heart: "", // heart-line
  heartFill: "", // heart-fill
  comment: "", // chat-3-line
  medal: "", // medal-line
  left: "", // arrow-left-s-line
  right: "", // arrow-right-s-line
  close: "" // close-line
}

const logId = ref(0)
const detail = ref({})
const comments = ref([])
const loading = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")
const retrying = ref(false)
const posting = ref(false)
const commentText = ref("")
const replyTo = ref(null)
const inputFocused = ref(false)

// 服务端分页只切顶层（回复随所属顶层返回）：首屏 20 条，加载更多只追加下一页顶层。
const COMMENT_PAGE_SIZE = 20
const commentPage = ref(1)
const hasMoreComments = ref(false)
const rootCommentTotal = ref(0)
const loadingMore = ref(false)

// ---- 照片点名筛选（design D7）：默认「只看我的孩子」 ----
const myStudentId = ref(getCurrentStudentId())
const onlyMine = ref(true)
// 个体记录（发给指定学生）不做筛选——照片本就属于这些孩子，仅呈现标识
const isClassRecord = computed(() => detail.value.target_type !== 2)
// 无当前孩子 id 时无法筛选：退化为「看全部」，不展示切换
const canFilter = computed(() => isClassRecord.value && !!myStudentId.value)

const isInitialLoading = computed(() => loading.value && !hasLoaded.value)
const showError = computed(() => Boolean(loadError.value) && !hasLoaded.value)
const locked = computed(() => isInitialLoading.value || showError.value)

const creatorName = computed(() => detail.value.creator?.nickname || "老师")
const recordTime = computed(() => formatRecordTime(detail.value.date, detail.value.created_at))

// 当日状态：任一字段有值才渲染整行，缺项显示「—」而不是 0
const RATING = ["", "差", "中", "好"]

function ratingLabel(v) {
  return RATING[v] || "—"
}

const dayStatus = computed(() => {
  const d = detail.value
  const hasAny =
    Boolean(d.breakfast_rating) || Boolean(d.lunch_rating) || Boolean(d.nap_duration) || Boolean(d.morning_mood)
  if (!hasAny) return []
  return [
    { key: "breakfast", label: "早餐", value: ratingLabel(d.breakfast_rating) },
    { key: "lunch", label: "午餐", value: ratingLabel(d.lunch_rating) },
    { key: "nap", label: "午睡", value: d.nap_duration ? `${d.nap_duration}分钟` : "—" },
    { key: "mood", label: "情绪", value: ratingLabel(d.morning_mood) }
  ]
})

const tags = computed(() =>
  Array.isArray(detail.value.performance_tags) ? detail.value.performance_tags.filter(Boolean) : []
)

function isVideo(m) {
  return m && m.file_type === "video"
}

function isMyPhoto(m) {
  return (m.student_ids || []).includes(myStudentId.value)
}

const allMedia = computed(() => (Array.isArray(detail.value.media) ? detail.value.media : []))

// 筛选只作用于展示，不改变服务端数据；个体记录直接全量展示
function inFilter(media) {
  if (!canFilter.value || !onlyMine.value) return true
  return isMyPhoto(media)
}

const photos = computed(() => allMedia.value.filter(m => !isVideo(m) && inFilter(m)))
const videos = computed(() => allMedia.value.filter(m => isVideo(m) && inFilter(m)))
const mediaEmpty = computed(
  () => allMedia.value.length > 0 && photos.value.length === 0 && videos.value.length === 0
)

// 标题与互动栏统一用服务端顶层评论数（root_total），不再数已加载数组；
// 否则分页后「评论 (20)」会把「其实有 36 条」说成 20 条。
const commentCount = computed(() => rootCommentTotal.value)

const sendDisabled = computed(() => locked.value || posting.value || !commentText.value.trim())

// 存储无当前孩子时自愈一次（与首页/成长列表同入口），保证筛选与服务端视角一致
async function ensureCurrentStudent() {
  if (myStudentId.value) return
  try {
    const res = await fetchParentHomeOverview(null)
    syncCurrentStudentFromOverview(res.data)
    myStudentId.value = getCurrentStudentId()
  } catch (err) {
    // 自愈失败不阻断浏览：退化为看全部
    console.warn("[growth-detail] resolve student failed:", err?.message || err)
  }
}

async function loadData(id) {
  loading.value = true
  loadError.value = ""
  try {
    await ensureCurrentStudent()
    const [d, c] = await Promise.all([
      fetchGrowthDetail(id, myStudentId.value || undefined),
      fetchGrowthComments(id, { page: 1, page_size: COMMENT_PAGE_SIZE })
    ])
    detail.value = d.data || {}
    comments.value = applyCommentPage(c.data)
    commentPage.value = 1
    hasLoaded.value = true
  } catch (err) {
    // http.js 已弹过全局 toast，这里只负责页面级错误态，不重复提示
    console.warn("[growth-detail] load failed:", err?.message || err)
    loadError.value = "这条记录加载失败"
  } finally {
    loading.value = false
  }
}

async function handleRetry() {
  if (loading.value) return
  retrying.value = true
  await loadData(logId.value)
  retrying.value = false
  if (loadError.value) {
    uni.showToast({ title: "仍无法加载，请稍后再试", icon: "none" })
  } else {
    uni.showToast({ title: "已重新加载", icon: "none" })
  }
}

// list 只含当前页顶层；total 含回复，root_total 是顶层数（标题计数口径）。
function applyCommentPage(data) {
  const list = Array.isArray(data?.list) ? data.list : []
  const rootTotal = data?.root_total
  rootCommentTotal.value = typeof rootTotal === "number" ? rootTotal : list.length
  hasMoreComments.value = data?.has_more === true
  return list
}

async function loadMoreComments() {
  if (locked.value || loadingMore.value || !hasMoreComments.value) return
  loadingMore.value = true
  try {
    const nextPage = commentPage.value + 1
    const res = await fetchGrowthComments(logId.value, {
      page: nextPage,
      page_size: COMMENT_PAGE_SIZE
    })
    // 追加而非替换：已加载的顶层评论不能因为翻页被丢弃。
    comments.value = comments.value.concat(applyCommentPage(res.data))
    commentPage.value = nextPage
  } catch (err) {
    // http.js 已提示；保留已加载内容与游标，用户可再次点击重试。
    console.warn("[growth-detail] load more comments failed:", err?.message || err)
  } finally {
    loadingMore.value = false
  }
}

function hasComment(id) {
  return comments.value.some(
    comment => comment.id === id || (comment.replies || []).some(reply => reply.id === id)
  )
}

// 发评论/回复后重取已加载的每一页再拼接，不能只重置到第 1 页丢掉后续页。
// 新顶层评论按 created_at ASC 排在末尾：先补到它应落的页，再兜底轮询直到看见。
async function refreshComments(focusId, focusIsRoot = false) {
  try {
    const pageCount = Math.max(
      1,
      commentPage.value,
      focusIsRoot && focusId ? Math.ceil((rootCommentTotal.value + 1) / COMMENT_PAGE_SIZE) : 1
    )
    const responses = await Promise.all(
      Array.from({ length: pageCount }, (_, index) =>
        fetchGrowthComments(logId.value, { page: index + 1, page_size: COMMENT_PAGE_SIZE })
      )
    )
    let list = []
    responses.forEach(res => {
      list = list.concat(applyCommentPage(res.data))
    })
    comments.value = list
    // 本轮实际取了 1..pageCount（可能为了新评论多取一页），游标必须跟上，否则下次加载会重取该页。
    commentPage.value = pageCount

    let page = pageCount
    while (focusId && !hasComment(focusId) && hasMoreComments.value) {
      page += 1
      const res = await fetchGrowthComments(logId.value, {
        page,
        page_size: COMMENT_PAGE_SIZE
      })
      const next = applyCommentPage(res.data)
      if (!next.length) break
      comments.value = comments.value.concat(next)
      commentPage.value = page
    }
  } catch (err) {
    console.warn("[growth-detail] refresh comments failed:", err?.message || err)
  }
}

// 同一记录在途时忽略后续点击：双击会并发发两个点赞/取消请求，
// 先返回的 500 会弹「操作失败」，与最终状态矛盾。
const likePending = ref(false)

async function handleLike() {
  if (locked.value || likePending.value) return
  likePending.value = true
  try {
    const r = detail.value.is_liked
      ? await unlikeGrowth(detail.value.id)
      : await likeGrowth(detail.value.id)
    const next = r?.data?.is_liked
    // 缺少布尔值时不静默翻转 —— 宁可报错也不把未知状态当取消点赞。
    if (typeof next !== "boolean") throw new Error("点赞状态未更新")
    detail.value = { ...detail.value, is_liked: next }
    if (typeof r?.data?.likes_count === "number") {
      detail.value = { ...detail.value, likes_count: r.data.likes_count }
    }
  } catch (err) {
    // 不做乐观翻转：失败时本地仍是服务端已确认的状态，无需回滚。
    console.warn("[growth-detail] like failed:", err?.message || err)
    uni.showToast({ title: err?.message || "操作失败", icon: "none" })
  } finally {
    likePending.value = false
  }
}

function focusInput() {
  if (locked.value) return
  inputFocused.value = true
}

function startReply(comment) {
  if (locked.value) return
  replyTo.value = { id: comment.id, name: comment.user?.nickname || "这条评论" }
  inputFocused.value = true
}

function cancelReply() {
  replyTo.value = null
}

async function submitComment() {
  if (sendDisabled.value) return
  const text = commentText.value.trim()
  if (!text) return
  posting.value = true
  try {
    const isRootComment = !replyTo.value
    const created = await postGrowthComment(logId.value, {
      content: text,
      parent_id: replyTo.value ? replyTo.value.id : null
    })
    commentText.value = ""
    replyTo.value = null
    uni.showToast({ title: "评论成功", icon: "success" })
    // 与服务端同口径：重取已加载页（created_at ASC），新评论落在末尾
    await refreshComments(created?.data?.id, isRootComment)
  } catch (err) {
    // 失败保留输入内容，http.js 已提示错误
    console.warn("[growth-detail] post comment failed:", err?.message || err)
  } finally {
    posting.value = false
  }
}

// 关系标签：nickname 里已含孩子名（如「乐乐妈妈」）时不再重复拼 student_name。
// 共享枚举覆盖 9 码（含外公/外婆/叔叔/阿姨）；未知码 fallback 为空串，保持不渲染。
function relationText(user) {
  const label = relationLabel(user?.relation, "")
  if (!label) return ""
  const nickname = String(user?.nickname || "")
  const studentName = String(user?.student_name || "")
  return studentName && !nickname.includes(studentName) ? `${studentName}的${label}` : label
}

// 徽章配色按 color_kind 映射到 design token 语义色（design D3）
function badgeKind(badge) {
  return badge && badge.color_kind === "warning" ? "warning" : "info"
}

// ---- 照片弹层：在当前筛选结果内翻页 ----
const previewOpen = ref(false)
const previewIndex = ref(0)
const previewImages = computed(() => photos.value)

function openPreview(idx) {
  if (locked.value || !photos.value.length) return
  previewIndex.value = idx
  previewOpen.value = true
}

function closePreview() {
  previewOpen.value = false
}

function stepPreview(delta) {
  const total = previewImages.value.length
  if (total < 2) return
  previewIndex.value = (previewIndex.value + delta + total) % total
}

function onPreviewChange(e) {
  const current = Number(e?.detail?.current)
  if (Number.isInteger(current)) previewIndex.value = current
}

onLoad((opts) => {
  const id = Number(opts?.id)
  if (!id) {
    loadError.value = "这条记录加载失败"
    return
  }
  logId.value = id
  loadData(id)
})
</script>

<template>
  <PageShell max-width="430px" :bottom-space="replyTo ? '272rpx' : '184rpx'">
    <!-- ===== 加载：整页骨架 ===== -->
    <view v-if="isInitialLoading" class="skeleton">
      <view class="sk-card">
        <view class="sk-author">
          <view class="sk sk-avatar" />
          <view class="sk-lines">
            <view class="sk sk-line sk-line--two" />
            <view class="sk sk-line sk-line--one" />
          </view>
        </view>
        <view class="sk-status">
          <view v-for="i in 4" :key="i" class="sk-stat">
            <view class="sk sk-mini" />
          </view>
        </view>
      </view>
      <view class="sk-card">
        <view class="sk-body">
          <view class="sk sk-line" />
          <view class="sk sk-line sk-line--nine" />
          <view class="sk sk-line sk-line--three" />
        </view>
        <view class="sk-photo-grid">
          <view v-for="i in 6" :key="i" class="sk sk-photo" />
        </view>
        <view class="sk-actions">
          <view class="sk sk-line sk-line--two" />
          <view class="sk sk-line sk-line--two" />
        </view>
      </view>
      <view class="sk-card sk-comment-card">
        <view class="sk sk-line sk-line--three" />
        <view class="sk-comment">
          <view class="sk sk-line sk-line--four" />
          <view class="sk sk-line sk-line--nine" />
        </view>
      </view>
    </view>

    <!-- ===== 失败：整页错误卡 + 重试 ===== -->
    <view v-else-if="showError" class="error-wrap">
      <view class="error-card">
        <view class="error-mark">!</view>
        <text class="error-title">这条记录加载失败</text>
        <text class="error-desc">网络异常，或老师已删除这条记录</text>
        <view class="retry-btn" :class="{ 'is-busy': retrying }" @click="handleRetry">
          <text>{{ retrying ? "重试中…" : "重试" }}</text>
        </view>
      </view>
    </view>

    <!-- ===== 正常 ===== -->
    <template v-else>
      <!-- 作者卡 + 当日状态 -->
      <view class="card author-card" aria-label="记录概况">
        <view class="author-row">
          <view class="author-avatar">
            <image
              v-if="detail.creator?.avatar"
              class="author-avatar-img"
              :src="detail.creator.avatar"
              mode="aspectFill"
            />
            <text v-else class="author-avatar-initial">{{ creatorName.slice(0, 1) }}</text>
          </view>
          <view class="author-copy">
            <text class="author-name">{{ creatorName }}</text>
            <text class="author-time">{{ recordTime }}</text>
          </view>
          <view v-if="detail.badge" class="award" :class="'award--' + badgeKind(detail.badge)">
            <text class="award-icon">{{ ICON.medal }}</text>
            <text class="award-name">{{ detail.badge.name }}</text>
          </view>
        </view>

        <view v-if="dayStatus.length" class="day-status">
          <view v-for="st in dayStatus" :key="st.key" class="day-cell">
            <text class="day-label">{{ st.label }}</text>
            <text class="day-value">{{ st.value }}</text>
          </view>
        </view>
      </view>

      <!-- 正文 + 标签 + 照片 + 互动 -->
      <view class="card record-card">
        <text v-if="detail.content" class="record-copy">{{ detail.content }}</text>

        <view v-if="tags.length" class="tag-list">
          <view v-for="tag in tags" :key="tag" class="tag">
            <text>{{ tag }}</text>
          </view>
        </view>

        <view v-if="allMedia.length" class="media-block">
          <view v-if="canFilter" class="filter-group">
            <view
              class="filter-chip"
              :class="{ 'is-active': onlyMine }"
              @click="onlyMine = true"
            >
              只看我的孩子
            </view>
            <view
              class="filter-chip"
              :class="{ 'is-active': !onlyMine }"
              @click="onlyMine = false"
            >
              看全部（{{ allMedia.length }}）
            </view>
          </view>

          <view v-if="photos.length" class="photo-grid">
            <view
              v-for="(m, idx) in photos"
              :key="m.id"
              class="photo-cell"
              @click="openPreview(idx)"
            >
              <image class="photo-img" :src="m.thumb_url || m.file_url" mode="aspectFill" />
              <view v-if="myStudentId && isMyPhoto(m)" class="photo-mine">
                <view class="photo-mine-dot" />
                <text>我的孩子</text>
              </view>
            </view>
          </view>

          <view v-for="m in videos" :key="'v' + m.id" class="video-block">
            <video
              class="video-item"
              :src="m.file_url"
              :poster="m.thumb_url || ''"
              controls
              object-fit="contain"
            />
            <view v-if="myStudentId && isMyPhoto(m)" class="video-mine">
              <view class="photo-mine-dot" />
              <text>我的孩子</text>
            </view>
          </view>

          <!-- 默认视图里没拍到自家孩子：引导切到全部 -->
          <view v-if="mediaEmpty" class="media-empty">
            <text class="media-empty-text">这些照片里暂时没有拍到您的孩子</text>
            <view class="media-empty-btn" @click="onlyMine = false">
              查看全部照片（{{ allMedia.length }}）
            </view>
          </view>
        </view>

        <!-- 相关孩子：仅发给指定学生的记录有值 -->
        <view v-if="detail.related_students?.length" class="related-row">
          <text class="related-label">相关孩子（{{ detail.related_students.length }}人）</text>
          <text v-for="s in detail.related_students" :key="s.id" class="related-name">{{ s.name }}</text>
        </view>

        <view class="engagement-bar">
          <view class="like-btn" :class="{ 'is-liked': detail.is_liked }" @click="handleLike">
            <text class="like-icon">{{ detail.is_liked ? ICON.heartFill : ICON.heart }}</text>
            <text class="like-count">{{ detail.likes_count || 0 }}</text>
          </view>
          <view class="comment-tally">
            <text class="tally-icon">{{ ICON.comment }}</text>
            <text>{{ commentCount }}</text>
          </view>
          <view class="comment-action" @click="focusInput">评论</view>
        </view>
      </view>

      <!-- 评论 -->
      <view class="card comments-card">
        <text class="comments-title">评论 ({{ commentCount }})</text>

        <view v-if="comments.length === 0" class="comments-empty">
          <view class="comments-empty-orb">
            <text class="comments-empty-icon">{{ ICON.comment }}</text>
          </view>
          <text class="comments-empty-title">还没有评论</text>
          <text class="comments-empty-desc">来说说这条记录里你看到的小细节</text>
          <view class="btn-primary" @click="focusInput">写第一条评论</view>
        </view>

        <view v-else class="comment-list">
          <view v-for="c in comments" :key="c.id" class="comment">
            <view class="comment-head">
              <text class="comment-author">{{ c.user?.nickname || "家长" }}</text>
              <text v-if="relationText(c.user)" class="relation-tag">{{ relationText(c.user) }}</text>
              <text class="comment-time">{{ formatCommentTime(c.created_at) }}</text>
            </view>
            <text class="comment-text">{{ c.content }}</text>
            <view class="comment-foot">
              <view class="reply-btn" @click="startReply(c)">回复</view>
            </view>

            <!-- 子回复：B06 只嵌一层，按缩进 + 竖线单独渲染 -->
            <view v-if="c.replies?.length" class="reply-thread">
              <view v-for="r in c.replies" :key="r.id" class="comment reply-comment">
                <view class="comment-head">
                  <text class="comment-author">{{ r.user?.nickname || "家长" }}</text>
                  <text v-if="relationText(r.user)" class="relation-tag">{{ relationText(r.user) }}</text>
                  <text class="comment-time">{{ formatCommentTime(r.created_at) }}</text>
                </view>
                <text class="comment-text">{{ r.content }}</text>
              </view>
            </view>
          </view>
          <view
            v-if="hasMoreComments"
            class="comment-more"
            :class="{ 'is-busy': loadingMore }"
            @click="loadMoreComments"
          >
            <text>{{ loadingMore ? "加载中…" : "加载更多评论" }}</text>
          </view>
        </view>
      </view>
    </template>

    <!-- 底部评论输入栏（固定，PageShell 用 bottom-space 预留高度） -->
    <template #bottom>
      <view class="composer">
        <view v-if="replyTo" class="reply-strip">
          <text class="reply-context">回复中 · {{ replyTo.name }}</text>
          <view class="cancel-reply" @click="cancelReply">取消</view>
        </view>
        <view class="composer-form">
          <input
            v-model="commentText"
            class="composer-input"
            :focus="inputFocused"
            :disabled="locked"
            placeholder="写评论..."
            placeholder-style="color: #94A3B8"
            confirm-type="send"
            :cursor-spacing="24"
            @blur="inputFocused = false"
            @confirm="submitComment"
          />
          <view class="composer-send" :class="{ 'is-disabled': sendDisabled }" @click="submitComment">
            <text>{{ posting ? "发送中…" : "发送" }}</text>
          </view>
        </view>
      </view>
    </template>

    <!-- 照片弹层：当前筛选结果内翻页 -->
    <view v-if="previewOpen" class="photo-sheet">
      <view class="sheet-scrim" @click="closePreview" />
      <view class="sheet-panel">
        <view class="sheet-head">
          <view class="sheet-heading">
            <text class="sheet-title">照片预览</text>
            <text class="preview-count">第 {{ previewIndex + 1 }} 张 / 共 {{ previewImages.length }} 张</text>
          </view>
          <view class="sheet-close" @click="closePreview">
            <text class="sheet-close-icon">{{ ICON.close }}</text>
          </view>
        </view>

        <swiper class="preview-swiper" :current="previewIndex" @change="onPreviewChange">
          <swiper-item v-for="m in previewImages" :key="m.id">
            <image class="preview-img" :src="m.file_url || m.thumb_url" mode="aspectFit" />
          </swiper-item>
        </swiper>

        <view v-if="previewImages.length > 1" class="preview-nav">
          <view
            class="preview-nav-btn"
            :class="{ 'is-disabled': previewIndex === 0 }"
            @click="stepPreview(-1)"
          >
            <text class="nav-icon">{{ ICON.left }}</text>
            <text>上一张</text>
          </view>
          <view
            class="preview-nav-btn"
            :class="{ 'is-disabled': previewIndex === previewImages.length - 1 }"
            @click="stepPreview(1)"
          >
            <text>下一张</text>
            <text class="nav-icon">{{ ICON.right }}</text>
          </view>
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 成长详情 · 一条记录的家庭阅读页
// 照片优先，正文像相册页边注脚；橙色只留在头像底、已赞与发送键上。
// ============================================================

.card {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

// ---- 作者卡 ----------------------------------------------------
.author-row {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
}

.author-avatar {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: $component-avatar-md;
  height: $component-avatar-md;
  overflow: hidden;
  border-radius: $border-radius-full;
  // 设计稿头像底是品牌渐变；渐变起点/终点均为 token 色
  background: $gradient-brand;
  box-shadow: $shadow-sm;
}

.author-avatar-img {
  width: 100%;
  height: 100%;
}

.author-avatar-initial {
  color: $color-text-on-accent;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.author-copy {
  flex: 1;
  min-width: 0;
}

.author-name {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.3;
}

.author-time {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.award {
  flex: none;
  max-width: 46%;
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  min-height: 48rpx;
  padding: 6rpx $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;

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

.award-icon {
  flex: none;
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

.award-name {
  min-width: 0;
  color: $color-text-primary;
  line-height: 1.35;
}

// 当日状态：四列等宽，竖线分隔
.day-status {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-top: $spacing-xl;
  padding-top: $spacing-lg;
  border-top: 1rpx solid $color-border-subtle;
}

.day-cell {
  min-width: 0;
  padding: 0 $spacing-md;
  border-left: 1rpx solid $color-border-subtle;

  &:first-child {
    padding-left: 0;
    border-left: 0;
  }

  &:last-child {
    padding-right: 0;
  }
}

.day-label {
  display: block;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

// 比作者名低一档（14px/500），次要状态不抢主视觉
.day-value {
  display: block;
  margin-top: 2rpx;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

// ---- 正文卡 ----------------------------------------------------
.record-card {
  margin-top: $spacing-lg;
}

.record-copy {
  display: block;
  color: $color-text-body;
  font-size: $typography-font-size-lg;
  line-height: 1.8;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
  margin-top: $spacing-lg;
}

// P2-4：11px / accent-deep 在 accent-wash 上只有 3.35:1。
// 改用 semantic-warning-strong 文案 + 13px，对比度回到 AA 以上。
.tag {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  min-height: 48rpx;
  padding: 6rpx $component-badge-padding-x;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-semantic-warning-strong;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.media-block {
  margin-top: $spacing-xl;
}

.filter-group {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-sm;
}

.filter-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  padding: 0 $spacing-lg;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.25;
}

// 选中态用深色而不是品牌橙：一屏只保留点赞与发送两处橙色
.filter-chip.is-active {
  border-color: $color-text-primary;
  background: $color-text-primary;
  color: $color-text-on-brand;
}

.photo-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
  margin-top: $spacing-md;
}

// aspect-ratio 小程序不支持：padding-top 撑正方形 + 图片绝对定位
.photo-cell {
  position: relative;
  width: 100%;
  padding-top: 100%;
  overflow: hidden;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
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

.photo-mine {
  position: absolute;
  bottom: $spacing-xs;
  left: $spacing-xs;
  display: inline-flex;
  align-items: center;
  gap: 6rpx;
  max-width: calc(100% - 16rpx);
  padding: 2rpx $spacing-sm;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
  box-shadow: $shadow-sm;
}

.photo-mine-dot {
  flex: none;
  width: 8rpx;
  height: 8rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary;
}

.video-block {
  position: relative;
  margin-top: $spacing-md;
}

.video-item {
  width: 100%;
  height: 420rpx;
  border-radius: $border-radius-lg;
  background: $color-text-primary;
}

.video-mine {
  position: absolute;
  bottom: $spacing-xs;
  left: $spacing-xs;
  display: inline-flex;
  align-items: center;
  gap: 6rpx;
  padding: 2rpx $spacing-sm;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.media-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-md;
  margin-top: $spacing-md;
  padding: $spacing-xl $spacing-lg;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
  text-align: center;
}

.media-empty-text {
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.media-empty-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
}

.related-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-sm;
  margin-top: $spacing-lg;
}

.related-label {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.related-name {
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-full;
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
  font-size: $typography-font-size-xs;
}

// ---- 互动栏 ----------------------------------------------------
.engagement-bar {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  margin-top: $spacing-xl;
  padding-top: $spacing-sm;
  border-top: 1rpx solid $color-border-subtle;
}

.like-btn {
  display: inline-flex;
  align-items: center;
  gap: $spacing-sm;
  min-width: 88rpx;
  min-height: 88rpx;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

.like-btn.is-liked {
  color: $color-brand-primary-dark;
}

.like-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.like-count {
  font-variant-numeric: tabular-nums;
}

.comment-tally {
  display: inline-flex;
  align-items: center;
  gap: $spacing-sm;
  min-height: 88rpx;
  padding: 0 $spacing-xs;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
  font-variant-numeric: tabular-nums;
}

.tally-icon {
  color: $color-text-muted;
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.comment-action {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;
  color: $color-text-body;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-medium;
}

// ---- 评论 ------------------------------------------------------
.comments-card {
  margin-top: $spacing-lg;
}

.comments-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.comment-list {
  margin-top: $spacing-md;
}

// 文本按钮：沿用评论区字号与品牌色，不新增底色，避免把翻页做成第二个主操作。
.comment-more {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  color: $color-brand-primary-dark;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.comment-more.is-busy {
  color: $color-text-hint;
}

.comment {
  padding: $spacing-lg 0;
}

.comment-list > .comment + .comment {
  border-top: 1rpx solid $color-border-subtle;
}

.comment-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-sm;
  min-width: 0;
}

.comment-author {
  min-width: 0;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.relation-tag {
  flex: none;
  padding: 2rpx $spacing-sm;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
}

.comment-time {
  margin-left: auto;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.comment-text {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-body;
  font-size: $typography-font-size-md;
  line-height: 1.75;
  overflow-wrap: anywhere;
}

.comment-foot {
  display: flex;
  justify-content: flex-end;
  margin-top: $spacing-xs;
}

.reply-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 88rpx;
  min-height: 64rpx;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
}

// 子回复：左侧竖线 + 浅底，与顶层评论分开
.reply-thread {
  margin-top: $spacing-sm;
  padding-left: $spacing-lg;
  border-left: 4rpx solid $color-border-subtle;
}

.reply-comment {
  margin-top: $spacing-xs;
  padding: $spacing-md $spacing-lg $spacing-xs;
  border-radius: $border-radius-lg;
  background: $color-bg-page;

  .comment-author {
    font-size: $typography-font-size-sm;
  }

  .comment-text {
    margin-top: 2rpx;
    font-size: $typography-font-size-base;
    line-height: 1.7;
  }
}

// ---- 空评论 ----------------------------------------------------
.comments-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-xl 0 $spacing-xs;
  text-align: center;
}

.comments-empty-orb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112rpx;
  height: 112rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-muted;
}

.comments-empty-icon {
  font-family: remixicon;
  font-size: $typography-font-size-3xl;
  line-height: 1;
}

.comments-empty-title {
  margin-top: $spacing-md;
  color: $color-text-primary;
  font-size: $typography-font-size-lg;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.comments-empty-desc {
  max-width: 520rpx;
  margin-top: $spacing-xs;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.7;
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: $component-input-height;
  margin-top: $spacing-lg;
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1.2;
  box-shadow: $shadow-sm;
}

// ---- 骨架 ------------------------------------------------------
.sk {
  display: block;
  background: $color-border-subtle;
  animation: sk-breathe 1.8s $animation-easing-default infinite;
}

.sk-card {
  margin-top: $spacing-lg;
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;

  &:first-child {
    margin-top: 0;
  }
}

.sk-author {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
}

.sk-avatar {
  flex: none;
  width: $component-avatar-md;
  height: $component-avatar-md;
  border-radius: $border-radius-full;
}

.sk-lines {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.sk-line {
  width: 100%;
  height: 18rpx;
  border-radius: $border-radius-full;
}

.sk-line--one {
  width: 50%;
}

.sk-line--two {
  width: 40%;
}

.sk-line--three {
  width: 60%;
}

.sk-line--four {
  width: 80%;
}

.sk-line--nine {
  width: 90%;
}

.sk-status {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: $spacing-md;
  margin-top: $spacing-xl;
}

.sk-stat {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 108rpx;
  border-radius: $border-radius-lg;
  background: $color-bg-page;
}

.sk-mini {
  width: 48rpx;
  height: 16rpx;
  border-radius: $border-radius-full;
}

.sk-body {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
}

.sk-photo-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-sm;
  margin-top: $spacing-xl;
}

.sk-photo {
  width: 100%;
  height: 200rpx;
  border-radius: $border-radius-lg;
}

.sk-actions {
  display: flex;
  justify-content: space-between;
  margin-top: $spacing-xl;
  padding-top: $spacing-lg;
  border-top: 1rpx solid $color-border-subtle;
}

.sk-comment-card {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.sk-comment {
  display: flex;
  flex-direction: column;
  gap: $spacing-sm;
  padding-top: $spacing-lg;
  border-top: 1rpx solid $color-border-subtle;
}

@keyframes sk-breathe {
  0%,
  100% {
    opacity: 0.48;
  }
  50% {
    opacity: 0.9;
  }
}

// ---- 错误态 ----------------------------------------------------
.error-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}

.error-card {
  width: 100%;
  padding: 64rpx $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
  text-align: center;
}

.error-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 120rpx;
  height: 120rpx;
  margin: 0 auto $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-2xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
}

.error-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.error-desc {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.7;
}

.retry-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 224rpx;
  min-height: $component-input-height;
  margin-top: $spacing-lg;
  padding: 0 $spacing-xl;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
}

.retry-btn.is-busy {
  color: $color-text-muted;
}

// ---- 底部输入栏 ------------------------------------------------
.composer {
  padding: $spacing-md $component-page-padding-x calc(16rpx + env(safe-area-inset-bottom));
  border-radius: $border-radius-xl $border-radius-xl 0 0;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.reply-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-sm;
  margin-bottom: $spacing-xs;
  padding-left: $spacing-lg;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}

.reply-context {
  min-width: 0;
  padding: $spacing-xs 0;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.cancel-reply {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 88rpx;
  min-height: 64rpx;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
}

.composer-form {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  min-width: 0;
}

.composer-input {
  flex: 1;
  min-width: 0;
  height: $component-input-height;
  padding: 0 $spacing-lg;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-primary;
  font-size: $typography-font-size-md;
}

.composer-send {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 128rpx;
  height: $component-input-height;
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-medium;
  line-height: 1;
  box-shadow: $shadow-sm;
}

// 空输入/请求中：让「不可发」可见，而不是点了没反应
.composer-send.is-disabled {
  background: $color-bg-page;
  color: $color-text-hint;
  box-shadow: none;
}

// ---- 照片弹层 --------------------------------------------------
.photo-sheet {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 60;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.sheet-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  // 遮罩半透明黑：tokens.json 无遮罩色，属允许的单处例外
  background: rgba(0, 0, 0, 0.45);
}

.sheet-panel {
  position: relative;
  z-index: 1;
  width: calc(100% - 32rpx);
  max-height: 90vh;
  margin: 0 $spacing-md calc(16rpx + env(safe-area-inset-bottom));
  padding: $spacing-lg;
  border-radius: $border-radius-2xl;
  background: $color-bg-card;
  box-shadow: $shadow-lg;
}

.sheet-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: $spacing-lg;
  padding: 0 $spacing-xs $spacing-lg;
}

.sheet-heading {
  min-width: 0;
}

.sheet-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.35;
}

.preview-count {
  display: block;
  margin-top: 4rpx;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-medium;
  line-height: 1.35;
}

.sheet-close {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  background: $color-bg-page;
  color: $color-text-body;
}

.sheet-close-icon {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.preview-swiper {
  width: 100%;
  height: 560rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-page;
}

.preview-img {
  width: 100%;
  height: 100%;
}

.preview-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  margin-top: $spacing-md;
}

.preview-nav-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  min-width: 200rpx;
  min-height: $component-input-height;
  padding: 0 $spacing-lg;
  border: 1rpx solid $color-border-subtle;
  border-radius: $border-radius-full;
  background: $color-bg-card;
  color: $color-text-primary;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}

.preview-nav-btn.is-disabled {
  border-color: $color-bg-page;
  background: $color-bg-page;
  color: $color-text-hint;
  pointer-events: none;
}

.nav-icon {
  font-family: remixicon;
  font-size: $typography-font-size-md;
  line-height: 1;
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
    opacity: 0.65;
  }
}
</style>
