<script setup>
import { computed, ref } from "vue"
import { onLoad, onShow } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchClassCalendar } from "@/api/parent-class"
import CalendarEventCard from "@/components/CalendarEventCard.vue"

// 家长端日历（ui-design-notes §1）：简单列表 + 顶部变更提示条。
// 类型标签/颜色由服务端按园所字典解析后随事件下发（家长端无 dicts 接口）；草稿与未发布事件
// 由服务端硬过滤，本端不感知。

const events = ref([])
const loading = ref(false)
const expanded = ref(false)
const highlightId = ref(0)
let highlightTimer = null

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const res = await fetchClassCalendar()
    events.value = res.data?.list || []
  } finally { loading.value = false }
}

// 变更提示条：服务端只在「近 7 天修订且尚未发生」窗口内下发 change_summary。
// 显式提示而非静默更新——家长按旧时间行动扑空才是真正的伤害。
const changed = computed(() => events.value.filter(e => e.change_summary))
const visibleChanges = computed(() => (expanded.value ? changed.value : changed.value.slice(0, 3)))
const hiddenCount = computed(() => Math.max(0, changed.value.length - 3))

// 家长端无月历：点提示条条目 = 滚动到对应卡片并短暂高亮
function locate(e) {
  highlightId.value = e.id
  uni.createSelectorQuery().select(`#evt-${e.id}`).boundingClientRect(rect => {
    if (rect) uni.pageScrollTo({ scrollTop: Math.max(0, rect.top - 140), duration: 300 })
  }).exec()
  if (highlightTimer) clearTimeout(highlightTimer)
  highlightTimer = setTimeout(() => { highlightId.value = 0 }, 2400)
}

let skipNextShow = true
onLoad(loadData)
onShow(() => {
  // 从其他页返回时刷新；首次 onShow 紧随 onLoad，跳过
  if (skipNextShow) { skipNextShow = false; return }
  loadData()
})
</script>

<template>
  <view class="page">
    <!-- 变更提示条（不可关闭；窗口结束自然消失） -->
    <view v-if="changed.length" class="alert-card">
      <view class="alert-head">
        <view class="alert-icon-box"><text class="alert-icon">变</text></view>
        <text class="alert-title">安排有变更</text>
      </view>
      <view class="alert-list">
        <view v-for="e in visibleChanges" :key="e.id" class="alert-line" @click="locate(e)">
          <text class="alert-ev-title">{{ e.title }}</text>
          <text class="alert-summary">{{ e.change_summary }}</text>
        </view>
      </view>
      <view v-if="hiddenCount > 0" class="alert-more" @click="expanded = !expanded">
        <text>{{ expanded ? "收起" : `另有 ${hiddenCount} 条` }}</text>
      </view>
    </view>

    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="events.length === 0" class="state">暂无活动安排</view>
    <view v-else class="list">
      <!-- 卡片走 shared-ui，与老师端「预览家长视角」是同一段代码 -->
      <view v-for="item in events" :key="item.id" :id="'evt-' + item.id">
        <CalendarEventCard :event="item" :highlighted="highlightId === item.id" />
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg $component-page-padding-x; background: $color-bg-page; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }

/* 变更提示条：蜜蜡黄浅底承注意语义（非错误），图标承载识别点（不用色条） */
.alert-card {
  padding: $component-card-padding;
  border-radius: $component-card-radius;
  background: $color-semantic-warning-bg;
  margin-bottom: $spacing-lg;
}
.alert-head { display: flex; align-items: center; gap: $spacing-md; margin-bottom: $spacing-md; }
.alert-icon-box {
  display: flex; align-items: center; justify-content: center;
  width: 44rpx; height: 44rpx; border-radius: $border-radius-md;
  background: $color-bg-card;
}
.alert-icon { font-size: $typography-font-size-sm; font-weight: 700; color: $color-semantic-warning; }
.alert-title { font-size: $typography-font-size-md; font-weight: 700; color: $color-semantic-warning; }
.alert-list { display: flex; flex-direction: column; gap: $spacing-sm; }
.alert-line { display: flex; flex-direction: column; gap: 2rpx; padding: $spacing-sm $spacing-md; border-radius: $border-radius-md; background: $color-bg-card; }
.alert-ev-title { font-size: $typography-font-size-base; font-weight: 600; color: $color-text-primary; }
.alert-summary { font-size: $typography-font-size-sm; color: $color-text-body; }
.alert-more { margin-top: $spacing-sm; text-align: center; }
.alert-more text { font-size: $typography-font-size-sm; color: $color-semantic-warning; }

.list { display: flex; flex-direction: column; gap: 18rpx; }
</style>
