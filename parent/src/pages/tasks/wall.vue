<script setup>
import { computed, ref } from "vue"
import { onLoad, onPullDownRefresh, onReachBottom } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchPublicWall, withStudentScope } from "@/api/parent-task"

const PAGE_SIZE = 20

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loadingMore = ref(false)
const noMore = ref(false)
const loadError = ref("")

function pad(n) {
  return String(n).padStart(2, "0")
}

// 后端给的是 UTC；分组与时间都按设备本地时区显示，不能直接截字符串。
function localDate(iso) {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? null : d
}

function clockLabel(iso) {
  const d = localDate(iso)
  return d ? `${pad(d.getHours())}:${pad(d.getMinutes())}` : ""
}

// 按提交日期分段：今天 / 昨天 / M月D日。
// 用 date（提交当天）而不是 created_at 的日期，家长认的是「孩子哪天做的」。
function dayLabel(dateStr) {
  const parts = String(dateStr || "").split("-")
  if (parts.length !== 3) return dateStr || ""
  const now = new Date()
  const today = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
  const y = new Date(now.getTime() - 86400000)
  const yesterday = `${y.getFullYear()}-${pad(y.getMonth() + 1)}-${pad(y.getDate())}`
  if (dateStr === today) return "今天"
  if (dateStr === yesterday) return "昨天"
  return `${Number(parts[1])}月${Number(parts[2])}日`
}

const groups = computed(() => {
  const out = []
  let current = ""
  for (const item of list.value) {
    const key = String(item.date || "")
    if (key !== current) {
      current = key
      out.push({ key: key || String(out.length), label: dayLabel(key), items: [] })
    }
    out[out.length - 1].items.push(item)
  }
  return out
})

function initial(item) {
  const name = item?.student?.name || ""
  return name ? name.slice(0, 1) : "同"
}

function preview(item) {
  if (!item?.media_url) return
  uni.previewImage({ urls: [item.media_url], current: item.media_url })
}

async function load(reset) {
  if (reset) {
    page.value = 1
    noMore.value = false
    // 已有数据时静默刷新：下拉或重试时把整页闪成「加载中」体验很差。
    if (list.value.length === 0) loading.value = true
  } else {
    if (loadingMore.value || noMore.value) return
    loadingMore.value = true
  }
  loadError.value = ""
  try {
    await ensureLogin()
    const scoped = await withStudentScope((sid) => fetchPublicWall({
      student_id: sid,
      page: page.value,
      page_size: PAGE_SIZE
    }))
    if (!scoped) {
      loadError.value = "未绑定孩子信息"
      return
    }
    const data = scoped.result?.data || {}
    const rows = data.list || []
    list.value = reset ? rows : list.value.concat(rows)
    total.value = Number(data.total || list.value.length)
    noMore.value = rows.length < PAGE_SIZE || list.value.length >= total.value
    if (!noMore.value) page.value += 1
  } catch (err) {
    // 不清空已有内容：清空会把「刷新失败」伪装成「本班没有分享」。
    loadError.value = err?.message || "加载失败"
  } finally {
    loading.value = false
    loadingMore.value = false
    uni.stopPullDownRefresh()
  }
}

onLoad(() => load(true))
onPullDownRefresh(() => load(true))
onReachBottom(() => load(false))
</script>

<template>
  <view class="page">
    <!-- 导航栏已经是「班级墙」，这里只说规则，不重复标题 -->
    <text class="intro-hint">老师确认后才会出现在这里 · 仅本班可见</text>

    <view v-if="loading" class="state">
      <text class="state-text">加载中...</text>
    </view>
    <view v-else-if="loadError && list.length === 0" class="state">
      <text class="state-text">{{ loadError }}</text>
      <view class="state-btn" @click="load(true)">重试</view>
    </view>
    <view v-else-if="list.length === 0" class="state">
      <text class="state-text">本班还没有分享</text>
      <text class="state-hint">家长提交、老师确认之后，会出现在这里</text>
    </view>
    <template v-else>
      <view v-if="loadError" class="inline-error" @click="load(true)">
        <text>{{ loadError }}，点击重试</text>
      </view>

      <view v-for="group in groups" :key="group.key" class="group">
        <text class="group-label">{{ group.label }}</text>
        <view
          v-for="item in group.items"
          :key="item.checkin_id"
          class="moment"
          :class="{ words: !item.media_url }"
        >
          <image
            v-if="item.media_url"
            class="moment-media"
            :src="item.media_url"
            mode="aspectFill"
            @click="preview(item)"
          />
          <view class="moment-body">
            <text v-if="item.content" class="moment-text">{{ item.content }}</text>
            <view class="moment-meta">
              <view class="who">
                <image v-if="item.student && item.student.avatar" class="avatar" :src="item.student.avatar" />
                <view v-else class="avatar avatar-fallback">
                  <text class="avatar-initial">{{ initial(item) }}</text>
                </view>
                <text class="who-name">{{ item.student && item.student.name ? item.student.name : "同班家长" }}</text>
              </view>
              <text class="moment-time">{{ clockLabel(item.created_at) }}</text>
            </view>
            <text class="moment-task">{{ item.task_title }}</text>
          </view>
        </view>
      </view>

      <view v-if="loadingMore" class="footer">
        <text class="footer-text">正在加载更多...</text>
      </view>
      <view v-else-if="noMore" class="footer">
        <text class="footer-text">没有更多了</text>
      </view>
      <view class="bottom-safe" />
    </template>
  </view>
</template>

<style scoped lang="scss">
.page {
  min-height: 100vh;
  padding: $component-page-padding-y $component-page-padding-x $spacing-xl;
  background: $color-bg-page;
  box-sizing: border-box;
}

.intro-hint {
  display: block;
  padding: 0 $spacing-xs;
  margin-bottom: $spacing-lg;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $spacing-md;
  min-height: 50vh;
}
.state-text { color: $color-text-muted; font-size: $typography-font-size-md; }
.state-hint { color: $color-text-hint; font-size: $typography-font-size-sm; }
.state-btn {
  padding: $spacing-sm 40rpx;
  border-radius: $border-radius-full;
  background: $color-brand-primary-bg;
  color: $color-brand-primary;
  font-size: $typography-font-size-sm;
}

.inline-error {
  padding: $spacing-sm $spacing-md;
  margin-bottom: $spacing-md;
  border-radius: $border-radius-md;
  background: $color-semantic-danger-bg;
  color: $color-semantic-danger;
  font-size: $typography-font-size-sm;
}

.group { margin-bottom: $spacing-lg; }
.group-label {
  display: block;
  padding: 0 $spacing-xs;
  margin-bottom: $spacing-sm;
  color: $color-text-hint;
  font-size: $typography-font-size-xs;
  font-weight: $typography-font-weight-semibold;
}

// 有图的内容是「照片」，没图的内容是「一句话」——两种形态用底色区分，
// 而不是给每张卡都套同一个模板。
.moment {
  margin-bottom: $spacing-md;
  border-radius: $component-card-radius;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
  overflow: hidden;
}
.moment.words { background: $color-brand-primary-bg; }

.moment-media {
  display: block;
  width: 100%;
  height: 420rpx;
  background: $color-bg-page;
}

.moment-body { padding: $component-card-padding; }

.moment-text {
  display: block;
  color: $color-text-primary;
  font-size: $typography-font-size-base;
  line-height: 1.6;
  word-break: break-all;
}

.moment-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: $spacing-md;
}
.who { display: flex; align-items: center; gap: $spacing-sm; }
.avatar {
  width: $component-avatar-sm;
  height: $component-avatar-sm;
  border-radius: $border-radius-full;
  background: $color-bg-page;
}
.avatar-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  background: $color-bg-card;
}
.avatar-initial {
  color: $color-brand-primary;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-bold;
}
.who-name {
  color: $color-text-body;
  font-size: $typography-font-size-sm;
  font-weight: $typography-font-weight-medium;
}
.moment-time {
  color: $color-text-hint;
  font-size: $typography-font-size-xs;
}

.moment-task {
  display: block;
  margin-top: $spacing-sm;
  color: $color-text-muted;
  font-size: $typography-font-size-xs;
}

.footer { padding: $spacing-md 0; text-align: center; }
.footer-text { color: $color-text-hint; font-size: $typography-font-size-xs; }
.bottom-safe { height: calc(48rpx + env(safe-area-inset-bottom)); }
</style>
