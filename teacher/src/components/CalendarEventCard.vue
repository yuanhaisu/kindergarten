<template>
  <view
    class="event-card"
    :class="{ 'is-highlighted': highlighted, 'is-parent-surface': surface === 'parent' }"
  >
    <view class="card-head">
      <text class="title">{{ event.title }}</text>
      <text
        class="type-chip"
        :class="{ 'is-unknown': !event.color }"
        :style="typeStyle"
      >{{ event.event_type_label || "活动" }}</text>
    </view>
    <text class="meta">{{ metaText }}</text>
    <text v-if="event.description" class="desc">{{ event.description }}</text>
  </view>
</template>

<script setup>
/**
 * CalendarEventCard — 日历里的「一日活动」卡片
 *
 * 家长端日历页与老师端「预览家长视角」共用这一个组件。
 * 「一模一样」只能靠同一段代码保证：两端各写一份长得像的样式，改一次就漂。
 *
 * event 为归一化后的形状（两端字段名不同，由调用方映射）：
 *   { title, event_type_label, color, event_date, is_all_day, start_time, end_time, location, description }
 * color 由园所字典解析：家长端随事件下发，老师端取字典 Extra；空值走 is-unknown 兜底。
 */
import { computed } from "vue"

const props = defineProps({
  event: { type: Object, required: true },
  highlighted: { type: Boolean, default: false },
  // 表面色来源：host = 用本端 token（家长端日历页）；parent = 强制套家长端的表面值，
  // 供老师端「预览家长视角」使用——两端卡片底色不同（老师端暖奶 / 家长端纯白）。
  surface: { type: String, default: "host" }
})

const typeStyle = computed(() => {
  const c = props.event?.color || ""
  return c ? { backgroundColor: c + "1F", color: c } : {}
})

const timeText = computed(() => {
  const e = props.event || {}
  if (e.is_all_day) return "全天"
  if (e.start_time && e.end_time) return `${e.start_time}-${e.end_time}`
  return e.start_time || ""
})

const metaText = computed(() => {
  const e = props.event || {}
  const t = timeText.value
  return `${e.event_date || ""}${t ? " · " + t : ""}${e.location ? " · " + e.location : ""}`
})
</script>

<style scoped lang="scss">
.event-card {
  padding: 26rpx;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  box-shadow: $shadow-sm;
}
.event-card.is-highlighted {
  background: $color-semantic-warning-bg;
  box-shadow: $shadow-md;
}

// 预览家长视角时套用家长端的表面值（tokens.json 的 parent 分支，非本端 token）：
// 卡片纯白、类型兜底胶囊用家长端品牌浅底/深色。老师端直接用本端 token 会偏暖奶色，
// 预览就不再是「家长看到的样子」。这两处硬编码是刻意的，改 tokens 时需同步。
.event-card.is-parent-surface { background: #FFFFFF; }
.event-card.is-parent-surface .type-chip.is-unknown { background: #FFF7ED; color: #EA580C; }

.card-head { display: flex; align-items: center; gap: $spacing-md; }
.title {
  flex: 1; min-width: 0;
  color: $color-text-primary; font-size: $typography-font-size-md; font-weight: 700;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.type-chip {
  flex-shrink: 0; padding: 2rpx $component-badge-padding-x;
  border-radius: $component-badge-radius;
  font-size: $component-badge-font-size; font-weight: 500;
}
.type-chip.is-unknown { background: $color-brand-primary-bg; color: $color-brand-primary-dark; }

.meta { display: block; margin-top: 8rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
.desc {
  margin-top: 6rpx; color: $color-text-body; font-size: $typography-font-size-sm; line-height: 1.5;
  display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden;
}
</style>
