<script setup>
/**
 * 班级相册详情 · 照片网格 + 大图预览 + 逐张保存原图
 *
 * 照片列表由服务端按「当前孩子 + 班内共享」过滤后返回，顺序仍是老师创建时的
 * sort_num、id；端上只做渲染，不重新筛选。Phase 1 的「导出」= 逐张保存原图
 * （design D8），不做打包下载。
 */
import { computed, ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchParentClassAlbum } from "@/api/parent-class-album"
import { fetchParentHomeOverview } from "@/api/parent-home"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { formatDateLabel } from "@/utils/time"
import PageShell from "@/components/PageShell.vue"

// remixicon 字形：小程序 WXML 不允许内联 SVG，字体已在 App.vue 全局注册。
const ICON = {
  gallery: "\u{eda5}", // gallery-line
  alert: "\u{eca1}", // error-warning-line
  left: "\u{ea64}", // arrow-left-s-line
  right: "\u{ea6e}", // arrow-right-s-line
  close: "\u{eb99}", // close-line
  download: "\u{ec54}" // download-2-line
}

const CATEGORY_LABELS = {
  activity: "活动",
  daily: "日常",
  work: "作品",
  festival: "节日"
}

const CATEGORY_TONES = {
  activity: "tag--activity",
  daily: "tag--daily",
  work: "tag--work",
  festival: "tag--festival"
}

const albumId = ref(0)
const studentId = ref(null)
const album = ref(null)
const loading = ref(true)
const retrying = ref(false)
const hasLoaded = ref(false)
const loadError = ref("")

const photos = computed(() => {
  const list = Array.isArray(album.value?.photos) ? album.value.photos : []
  // 没有可用地址的照片无法渲染，只做渲染保护，不改变服务端给的可见性口径。
  return list.filter(photo => photo && (photo.file_url || photo.thumb_url))
})

const isEmptyAlbum = computed(
  () => hasLoaded.value && !loadError.value && photos.value.length === 0
)

const photoCountText = computed(() => `${photos.value.length} 张照片`)

function categoryText() {
  return album.value?.category_label || CATEGORY_LABELS[album.value?.category] || "相册"
}

function tagClass() {
  return CATEGORY_TONES[album.value?.category] || "tag--neutral"
}

async function ensureStudent() {
  if (studentId.value) return
  try {
    const res = await fetchParentHomeOverview(null)
    syncCurrentStudentFromOverview(res.data)
    studentId.value = getCurrentStudentId()
  } catch (err) {
    console.warn("[class-album-detail] resolve student failed:", err?.message || err)
  }
}

async function loadData() {
  if (!albumId.value) {
    // 深链少了 id：不发请求，直接给可返回的错误态。
    loading.value = false
    loadError.value = "相册不存在"
    return
  }
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    await ensureStudent()
    const res = await fetchParentClassAlbum(albumId.value, {
      student_id: studentId.value || undefined
    })
    album.value = res.data || {}
    hasLoaded.value = true
  } catch (err) {
    // http.js 已弹过全局 toast；404（下架/跨班/留存过期）统一走页面错误态。
    console.warn("[class-album-detail] load failed:", err?.message || err)
    loadError.value = err?.message || "相册加载失败"
  } finally {
    loading.value = false
  }
}

async function handleRetry() {
  if (retrying.value || loading.value) return
  retrying.value = true
  await loadData()
  retrying.value = false
  if (loadError.value) {
    uni.showToast({ title: "仍无法加载，请稍后再试", icon: "none" })
  }
}

function goBack() {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
    return
  }
  uni.reLaunch({ url: "/pages/album/index" })
}

// ---------------- 大图预览 ----------------
const preview = ref({ open: false, index: 0 })
let touchStartX = null

const currentPhoto = computed(() => photos.value[preview.value.index] || null)
const canPrev = computed(() => preview.value.index > 0)
const canNext = computed(() => preview.value.index < photos.value.length - 1)

function openPreview(index) {
  if (!photos.value.length) return
  preview.value = {
    open: true,
    index: Math.min(Math.max(index, 0), photos.value.length - 1)
  }
}

function closePreview() {
  preview.value = { open: false, index: 0 }
}

function stepPreview(delta) {
  if (!preview.value.open) return
  const next = preview.value.index + delta
  if (next < 0 || next > photos.value.length - 1) return
  preview.value = { ...preview.value, index: next }
}

function onStageTouchStart(event) {
  const touch = event?.touches && event.touches[0]
  touchStartX = touch ? touch.clientX : null
}

function onStageTouchEnd(event) {
  if (touchStartX === null) return
  const touch = event?.changedTouches && event.changedTouches[0]
  const dx = touch ? touch.clientX - touchStartX : 0
  touchStartX = null
  if (Math.abs(dx) < 36) return
  stepPreview(dx < 0 ? 1 : -1)
}

function noop() {}

// ---------------- 保存原图 ----------------
const saving = ref(false)

function getAlbumAuth() {
  return new Promise(resolve => {
    // false = 曾经拒绝过，再调 saveImageToPhotosAlbum 只会静默失败；
    // 未询问过（undefined）时让它照常弹系统授权框。
    uni.getSetting({
      success(res) {
        resolve(res?.authSetting?.["scope.writePhotosAlbum"] !== false)
      },
      fail() {
        resolve(true)
      }
    })
  })
}

function promptOpenSetting() {
  uni.showModal({
    title: "需要相册权限",
    content: "保存照片需要允许写入相册，请在设置中开启后重试。",
    confirmText: "去设置",
    success(res) {
      if (res.confirm) uni.openSetting({})
    }
  })
}

function downloadImage(url) {
  return new Promise((resolve, reject) => {
    uni.downloadFile({
      url,
      success(res) {
        if (res.statusCode === 200 && res.tempFilePath) resolve(res.tempFilePath)
        else reject({ code: "download" })
      },
      fail() {
        reject({ code: "download" })
      }
    })
  })
}

function saveToAlbum(filePath) {
  return new Promise((resolve, reject) => {
    uni.saveImageToPhotosAlbum({
      filePath,
      success: resolve,
      fail(err) {
        const msg = String(err?.errMsg || "")
        if (msg.includes("cancel")) reject({ code: "cancel" })
        else if (/auth|deny|denied|permission/i.test(msg)) reject({ code: "auth" })
        else reject({ code: "save" })
      }
    })
  })
}

// 保存原图是纯客户端行为，Phase 1 用微信分析事件，不新增后端端点。
// 埋点失败不能影响“已保存”的结果，因此单独 try/catch。
function reportPhotoSave(photo) {
  if (typeof uni.reportAnalytics !== "function") return
  try {
    uni.reportAnalytics("album_photo_save", {
      album_id: albumId.value,
      photo_id: photo?.id || 0,
      student_id: studentId.value || 0
    })
  } catch (e) {
    // 忽略埋点异常
  }
}

async function saveCurrent() {
  const photo = currentPhoto.value
  if (!photo || saving.value) return
  const url = String(photo.file_url || photo.thumb_url || "").trim()
  if (!url) {
    uni.showToast({ title: "图片地址无效", icon: "none" })
    return
  }

  saving.value = true
  let loadingVisible = true
  const stopLoading = () => {
    if (!loadingVisible) return
    loadingVisible = false
    uni.hideLoading()
  }
  uni.showLoading({ title: "保存中…", mask: true })

  try {
    if (!(await getAlbumAuth())) {
      stopLoading()
      promptOpenSetting()
      return
    }
    const filePath = await downloadImage(url)
    await saveToAlbum(filePath)
    stopLoading()
    uni.showToast({ title: "已保存到相册", icon: "success" })
    reportPhotoSave(photo)
  } catch (err) {
    stopLoading()
    if (err?.code === "auth") promptOpenSetting()
    else if (err?.code === "cancel") uni.showToast({ title: "已取消保存", icon: "none" })
    else if (err?.code === "download") {
      uni.showToast({ title: "下载失败，请检查网络", icon: "none" })
    } else {
      uni.showToast({ title: "保存失败，请稍后再试", icon: "none" })
    }
  } finally {
    saving.value = false
  }
}

onLoad(query => {
  albumId.value = Number(query?.id) || 0
  studentId.value = Number(query?.student_id) || getCurrentStudentId()
  loadData()
})
</script>

<template>
  <PageShell max-width="430px" bottom-space="48rpx">
    <!-- 加载：头部 + 3×3 网格骨架 -->
    <view v-if="loading" class="skeleton">
      <view class="sk-head">
        <view class="sk sk-line sk-line--title" />
        <view class="sk sk-line sk-line--meta" />
      </view>
      <view class="sk-grid">
        <view v-for="i in 9" :key="i" class="sk sk-photo" />
      </view>
    </view>

    <!-- 失败 / 404：错误态带返回，不泄露相册是否存在 -->
    <view v-else-if="loadError" class="error-state">
      <view class="error-icon">
        <text class="error-glyph">{{ ICON.alert }}</text>
      </view>
      <text class="error-title">相册无法查看</text>
      <text class="error-desc">{{ loadError }}</text>
      <view class="error-actions">
        <view
          class="state-action"
          :class="{ 'is-busy': retrying }"
          :aria-disabled="retrying ? 'true' : 'false'"
          @click="handleRetry"
        >
          <text class="state-action-text">{{ retrying ? "重试中…" : "重试" }}</text>
        </view>
        <view class="state-action state-action--ghost" @click="goBack">
          <text class="state-action-text">返回相册列表</text>
        </view>
      </view>
    </view>

    <template v-else>
      <!-- 相册头部：标题、日期、分类、可见张数 -->
      <view class="album-head">
        <text class="head-title">{{ album.title || "班级相册" }}</text>
        <view class="head-meta">
          <text class="head-date">{{ formatDateLabel(album.album_date) }}</text>
          <view class="tag" :class="tagClass()">
            <text>{{ categoryText() }}</text>
          </view>
          <text class="head-count">共 {{ photoCountText }}</text>
        </view>
      </view>

      <!-- 后端已按可见性过滤；空数组是正常结果，不是错误 -->
      <view v-if="isEmptyAlbum" class="empty-state">
        <view class="empty-art">
          <text class="empty-glyph">{{ ICON.gallery }}</text>
        </view>
        <text class="empty-title">这本相册里暂时没有孩子的照片</text>
        <text class="empty-desc">老师发布新照片后会出现在这里</text>
        <view class="link-btn" @click="goBack">
          <text>看看其它相册</text>
        </view>
      </view>

      <template v-else>
        <view class="photo-grid">
          <view
            v-for="(photo, index) in photos"
            :key="photo.id"
            class="photo-tile"
            :aria-label="'查看第 ' + (index + 1) + ' 张照片'"
            @click="openPreview(index)"
          >
            <image
              class="photo-img"
              :src="photo.thumb_url || photo.file_url"
              mode="aspectFill"
            />
          </view>
        </view>
        <text class="page-note">点开照片可保存原图</text>
      </template>
    </template>

    <!-- 大图预览：左右切换，保存当前这张的原图 -->
    <view
      v-if="preview.open"
      class="viewer"
      @touchmove.stop.prevent="noop"
    >
      <view class="viewer-scrim" @click="closePreview" />

      <view class="viewer-top">
        <text class="viewer-title">{{ album && album.title ? album.title : "班级相册" }}</text>
        <view class="viewer-close" aria-label="关闭大图" @click="closePreview">
          <text class="viewer-close-glyph">{{ ICON.close }}</text>
        </view>
      </view>

      <view
        class="viewer-stage"
        @touchstart="onStageTouchStart"
        @touchend="onStageTouchEnd"
      >
        <image
          v-if="currentPhoto"
          class="viewer-img"
          :src="currentPhoto.file_url || currentPhoto.thumb_url"
          mode="aspectFit"
        />
      </view>

      <view class="viewer-nav-row">
        <view
          class="viewer-nav"
          :class="{ 'is-disabled': !canPrev }"
          aria-label="上一张"
          @click="stepPreview(-1)"
        >
          <text class="viewer-nav-glyph">{{ ICON.left }}</text>
        </view>
        <text class="viewer-count">
          第 {{ preview.index + 1 }} / {{ photos.length }} 张
        </text>
        <view
          class="viewer-nav"
          :class="{ 'is-disabled': !canNext }"
          aria-label="下一张"
          @click="stepPreview(1)"
        >
          <text class="viewer-nav-glyph">{{ ICON.right }}</text>
        </view>
      </view>

      <view class="viewer-actions">
        <view
          class="save-btn"
          :class="{ 'is-busy': saving }"
          :aria-disabled="saving ? 'true' : 'false'"
          @click="saveCurrent"
        >
          <text class="save-glyph">{{ ICON.download }}</text>
          <text>{{ saving ? "保存中…" : "保存原图" }}</text>
        </view>
      </view>
    </view>
  </PageShell>
</template>

<style scoped lang="scss">
// ============================================================
// 相册详情 · 三列网格 + 深色大图预览
// 相册信息只在顶部出现一次；照片本身承担页面主体。
// ============================================================

// ---- 相册头部 ------------------------------------------------
.album-head {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-md;
}

.head-title {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-3xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  overflow-wrap: break-word;
}

.head-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: $spacing-sm;
  margin-top: $spacing-md;
}

.head-date {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.head-count {
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.tag {
  display: inline-flex;
  align-items: center;
  padding: 6rpx $spacing-md;
  border-radius: $border-radius-full;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.tag--activity {
  background: $color-semantic-success-bg;
  color: $color-semantic-success-strong;
}

.tag--daily {
  background: $color-semantic-info-bg;
  color: $color-semantic-info;
}

.tag--work {
  background: $color-brand-primary-bg;
  color: $color-brand-primary-dark;
}

.tag--festival {
  background: $color-semantic-warning-bg;
  color: $color-semantic-warning-strong;
}

.tag--neutral {
  background: $color-bg-page;
  color: $color-text-muted;
}

// ---- 照片网格 ------------------------------------------------
.photo-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-xs;
  margin-top: $spacing-xl;
}

// aspect-ratio 小程序不支持：padding-top 撑正方形 + 图片绝对定位。
.photo-tile {
  position: relative;
  width: 100%;
  padding-top: 100%;
  overflow: hidden;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  transition: transform $animation-duration-fast $animation-easing-default;

  &:active {
    transform: scale(0.985);
  }
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

.page-note {
  display: block;
  margin-top: $spacing-lg;
  color: $color-text-muted;
  font-size: $typography-font-size-sm;
  line-height: 1.5;
  text-align: center;
}

// ---- 空态 / 错误态 -------------------------------------------
.empty-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 56vh;
  padding: $spacing-xl 0 96rpx;
  text-align: center;
}

.error-state {
  min-height: 70vh;
}

.empty-art {
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

.empty-glyph {
  font-family: remixicon;
  font-size: 72rpx;
  line-height: 1;
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

.empty-title,
.error-title {
  color: $color-text-primary;
  font-size: $typography-font-size-xl;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
}

.empty-desc,
.error-desc {
  max-width: 520rpx;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-base;
  line-height: 1.6;
}

.error-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
}

.state-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 224rpx;
  min-height: $component-input-height;
  margin-top: $spacing-xl;
  padding: 0 calc(#{$spacing-xl} + #{$spacing-xs});
  border-radius: $border-radius-full;
  background: $gradient-brand;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.98);
  }
}

.state-action--ghost {
  background: $color-bg-card;
  box-shadow: $shadow-sm;

  .state-action-text {
    color: $color-text-body;
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

.link-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 88rpx;
  margin-top: $spacing-md;
  padding: 0 $spacing-md;
  border-radius: $border-radius-full;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  font-weight: $typography-font-weight-semibold;
  text-decoration: underline;
  text-decoration-color: $color-border-hairline;
  text-underline-offset: 10rpx;
}

// ---- 加载骨架 ------------------------------------------------
.skeleton {
  padding-bottom: $spacing-lg;
}

.sk {
  display: block;
  background: $color-border-subtle;
  border-radius: $border-radius-full;
  animation: sk-breathe 1.6s $animation-easing-default infinite;
}

.sk-head {
  padding: $spacing-xl;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}

.sk-line {
  height: 28rpx;
}

.sk-line--title {
  width: 52%;
  height: 36rpx;
}

.sk-line--meta {
  width: 34%;
  height: 20rpx;
  margin-top: $spacing-lg;
}

.sk-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: $spacing-xs;
  margin-top: $spacing-xl;
}

.sk-photo {
  width: 100%;
  height: 216rpx;
  border-radius: $border-radius-lg;
}

@keyframes sk-breathe {
  0%,
  100% {
    opacity: 0.5;
  }
  50% {
    opacity: 1;
  }
}

// ---- 大图预览 ------------------------------------------------
.viewer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 60;
  display: flex;
  flex-direction: column;
  padding: $spacing-lg $spacing-lg calc(#{$spacing-lg} + env(safe-area-inset-bottom));
  // 全屏看图遮罩：tokens.json 无遮罩色，属允许的单处例外。
  background: rgba(0, 0, 0, 0.9);
}

.viewer-scrim {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
}

.viewer-top {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  padding: $spacing-xs $spacing-sm;
}

.viewer-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: $color-text-on-accent;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1.4;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.viewer-close {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  // 深色遮罩上的半透明白底：tokens.json 无对应透明度，属允许的单处例外。
  background: rgba(255, 255, 255, 0.14);
  color: $color-text-on-accent;
}

.viewer-close-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.viewer-stage {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 0;
  padding: $spacing-md 0;
}

.viewer-img {
  width: 100%;
  height: 100%;
}

.viewer-nav-row {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-lg;
  padding: $spacing-xs $spacing-sm;
}

.viewer-nav {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 88rpx;
  height: 88rpx;
  border-radius: $border-radius-full;
  background: rgba(255, 255, 255, 0.14);
  color: $color-text-on-accent;

  &:active {
    transform: scale(0.94);
  }
}

.viewer-nav.is-disabled {
  background: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.3);
  pointer-events: none;
}

.viewer-nav-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-2xl;
  line-height: 1;
}

.viewer-count {
  color: $color-text-on-accent;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
  line-height: 1.4;
  font-variant-numeric: tabular-nums;
}

.viewer-actions {
  position: relative;
  z-index: 1;
  display: flex;
  justify-content: center;
  padding: $spacing-md 0 $spacing-xs;
}

.save-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  min-width: 320rpx;
  min-height: $component-input-height;
  padding: 0 $spacing-xl;
  border-radius: $border-radius-full;
  background: $gradient-brand;
  color: $color-text-on-accent;
  font-size: $typography-font-size-md;
  font-weight: $typography-font-weight-semibold;
  line-height: 1;
  box-shadow: $shadow-md;

  &:active {
    transform: scale(0.97);
  }
}

.save-btn.is-busy {
  opacity: 0.72;
}

.save-glyph {
  font-family: remixicon;
  font-size: $typography-font-size-lg;
  line-height: 1;
}

@media (prefers-reduced-motion: reduce) {
  .sk {
    animation: none;
    opacity: 0.7;
  }

  .photo-tile,
  .viewer-nav,
  .save-btn,
  .state-action {
    transition: none;
  }
}
</style>
