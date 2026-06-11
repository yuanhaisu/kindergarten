<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"

const albums = ref([])
const loading = ref(false)
const activeCategory = ref("全部")
const categories = ["全部", "活动", "日常", "作品", "节日"]

const categoryClassMap = {
  "活动": "tag-activity",
  "日常": "tag-daily",
  "作品": "tag-artwork",
  "节日": "tag-festival",
}

const stats = ref({ total_photos: 386, monthly_new: 42, album_count: 12 })

const mockAlbums = [
  { id: 1, title: "春季出游活动", category: "活动", photo_count: 28, date: "2026-03-19", cover: "" },
  { id: 2, title: "美术课作品展", category: "作品", photo_count: 35, date: "2026-03-15", cover: "" },
  { id: 3, title: "晨间活动日常", category: "日常", photo_count: 56, date: "2026-03-10", cover: "" },
  { id: 4, title: "三八妇女节活动", category: "节日", photo_count: 42, date: "2026-03-08", cover: "" },
  { id: 5, title: "音乐课堂", category: "活动", photo_count: 22, date: "2026-03-05", cover: "" },
  { id: 6, title: "开学第一周", category: "日常", photo_count: 48, date: "2026-03-01", cover: "" },
]

const filteredAlbums = computed(() => {
  if (activeCategory.value === "全部") return albums.value
  return albums.value.filter(a => a.category === activeCategory.value)
})

function selectCategory(cat) { activeCategory.value = cat }

function openSearch() { uni.showToast({ title: "搜索功能开发中", icon: "none" }) }
function openUpload() { uni.showToast({ title: "上传功能开发中", icon: "none" }) }

function goAlbumDetail(id) {
  uni.showToast({ title: `查看相册 ${id}`, icon: "none" })
}

async function loadAlbums() {
  loading.value = true
  try {
    // const res = await fetchTeacherAlbums()
    // albums.value = res.data?.list || []
    albums.value = mockAlbums
  } catch (err) {
    albums.value = mockAlbums
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
}

onLoad(loadAlbums)
onPullDownRefresh(loadAlbums)
</script>

<template>
  <view class="page">
    <!-- 统计卡片 -->
    <view class="stats-card">
      <view class="stats-row">
        <view class="stat-item">
          <view class="stat-icon-box is-blue">
            <text class="stat-icon">照</text>
          </view>
          <view class="stat-text">
            <text class="stat-label">照片总数</text>
            <text class="stat-num">{{ stats.total_photos }}</text>
          </view>
        </view>
        <view class="stat-divider" />
        <view class="stat-item">
          <view class="stat-icon-box is-green">
            <text class="stat-icon">新</text>
          </view>
          <view class="stat-text">
            <text class="stat-label">本月新增</text>
            <text class="stat-num is-green-num">{{ stats.monthly_new }}</text>
          </view>
        </view>
        <view class="stat-divider" />
        <view class="stat-item">
          <view class="stat-icon-box is-purple">
            <text class="stat-icon">册</text>
          </view>
          <view class="stat-text">
            <text class="stat-label">相册数</text>
            <text class="stat-num is-purple-num">{{ stats.album_count }}</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 分类标签 -->
    <scroll-view scroll-x class="category-scroll" :show-scrollbar="false">
      <view class="category-row">
        <view
          v-for="cat in categories"
          :key="cat"
          class="category-pill"
          :class="{ active: activeCategory === cat }"
          @click="selectCategory(cat)"
        >
          <text>{{ cat }}</text>
        </view>
      </view>
    </scroll-view>

    <!-- 相册列表 -->
    <view v-if="loading" class="state-wrap">
      <text class="state-text">加载中...</text>
    </view>
    <view v-else-if="filteredAlbums.length === 0" class="state-wrap">
      <text class="state-text">暂无相册</text>
    </view>
    <view v-else class="album-list">
      <view
        v-for="album in filteredAlbums"
        :key="album.id"
        class="album-card"
        @click="goAlbumDetail(album.id)"
      >
        <!-- 封面区 -->
        <view class="cover-area">
          <view class="cover-main">
            <image
              v-if="album.cover"
              :src="album.cover"
              mode="aspectFill"
              class="cover-main-img"
            />
            <view v-else class="cover-main-placeholder">
              <text class="cover-placeholder-icon">图</text>
            </view>
          </view>
          <view class="cover-side">
            <view class="cover-thumb">
              <image
                v-if="album.cover"
                :src="album.cover"
                mode="aspectFill"
                class="cover-thumb-img"
              />
              <view v-else class="cover-thumb-placeholder" />
            </view>
            <view class="cover-thumb">
              <image
                v-if="album.cover"
                :src="album.cover"
                mode="aspectFill"
                class="cover-thumb-img"
              />
              <view v-else class="cover-thumb-placeholder" />
            </view>
          </view>
        </view>

        <!-- 相册信息 -->
        <view class="album-info">
          <view class="album-head">
            <text class="album-title">{{ album.title }}</text>
            <view class="album-count">
              <text>{{ album.photo_count }}张</text>
            </view>
          </view>
          <view class="album-foot">
            <text class="album-date">{{ album.date }}</text>
            <view class="album-tag" :class="categoryClassMap[album.category]">
              <text>{{ album.category }}</text>
            </view>
          </view>
        </view>
      </view>
    </view>

    <view class="bottom-safe" />
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: $spacing-lg 28rpx;
  background: $color-bg-page;
}

.stats-card {
  padding: $spacing-xl;
  border-radius: $border-radius-xl;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  margin-bottom: $spacing-lg;
}

.stats-row {
  display: flex;
  align-items: center;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  flex: 1;
}

.stat-icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border-radius: $border-radius-xl;
}

.stat-icon-box.is-blue   { background: $color-semantic-info-bg; }
.stat-icon-box.is-green  { background: #DCFCE7; }
.stat-icon-box.is-purple { background: #F3E8FF; }

.stat-icon {
  font-size: $typography-font-size-xl;
  font-weight: 700;

  .is-blue &   { color: $color-semantic-info; }
  .is-green &  { color: #16A34A; }
  .is-purple & { color: #9333EA; }
}

.stat-text {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.stat-label {
  font-size: $typography-font-size-xs;
  color: $color-text-hint;
}

.stat-num {
  font-size: $typography-font-size-xl;
  font-weight: 800;
  color: $color-text-primary;
}

.stat-num.is-green-num  { color: #16A34A; }
.stat-num.is-purple-num { color: #9333EA; }

.stat-divider {
  width: 4rpx;
  height: 64rpx;
  border-radius: 2rpx;
  background: #E2E8F0;
  margin: 0 $spacing-lg;
}

.category-scroll {
  margin-bottom: $spacing-lg;
  white-space: nowrap;
}

.category-row {
  display: flex;
  gap: $spacing-md;
}

.category-pill {
  flex-shrink: 0;
  padding: $spacing-md 36rpx;
  border-radius: $border-radius-full;
  background: #fff;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.04);

  text {
    font-size: $typography-font-size-base;
    color: $color-text-muted;
  }
}

.category-pill.active {
  background: $color-semantic-info;

  text {
    color: #fff;
    font-weight: 600;
  }
}

.state-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 50vh;
}

.state-text {
  font-size: $typography-font-size-md;
  color: $color-text-hint;
}

.album-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
}

.album-card {
  border-radius: $border-radius-xl;
  background: #fff;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.05);
  overflow: hidden;
}

.cover-area {
  display: flex;
  gap: $spacing-xs;
  height: 320rpx;
}

.cover-main {
  flex: 1;
  overflow: hidden;
}

.cover-main-img {
  width: 100%;
  height: 100%;
}

.cover-main-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: #F1F5F9;
}

.cover-placeholder-icon {
  font-size: 72rpx;
  font-weight: 700;
  color: #CBD5E1;
}

.cover-side {
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
  width: 180rpx;
  flex-shrink: 0;
}

.cover-thumb {
  flex: 1;
  overflow: hidden;
}

.cover-thumb-img {
  width: 100%;
  height: 100%;
}

.cover-thumb-placeholder {
  width: 100%;
  height: 100%;
  background: #F1F5F9;
}

.album-info {
  padding: $spacing-lg $spacing-xl 28rpx;
}

.album-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12rpx;
}

.album-title {
  font-size: $typography-font-size-lg;
  font-weight: 600;
  color: $color-text-primary;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.album-count {
  flex-shrink: 0;
  padding: 4rpx $spacing-md;
  border-radius: $border-radius-sm;
  background: #F1F5F9;

  text {
    font-size: $typography-font-size-sm;
    color: $color-text-muted;
  }
}

.album-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.album-date {
  font-size: $typography-font-size-base;
  color: $color-text-hint;
}

.album-tag {
  padding: 6rpx 20rpx;
  border-radius: $border-radius-full;

  text { font-size: $typography-font-size-sm; font-weight: 500; }
}

.tag-activity { background: #DCFCE7; text { color: #16A34A; } }
.tag-artwork  { background: #F3E8FF; text { color: #9333EA; } }
.tag-daily    { background: $color-semantic-info-bg; text { color: $color-semantic-info; } }
.tag-festival { background: #FEF3C7; text { color: #D97706; } }

.bottom-safe {
  height: calc(60rpx + env(safe-area-inset-bottom));
}
</style>
