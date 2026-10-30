// 站内消息的展示层映射：六类类型 → 中文/图标/色调，日期分组与时间文案。
// 页面只负责加载与交互，视图语义集中在这里，方便对照
// backend/api/api-common.md「消息」一节的 type 枚举。

import { formatClock, formatDateLabel, localDateKey } from "@/utils/time"

// remixicon 4.9.1 字形（App.vue 已全局注册字体）；WXML 不允许内联 SVG。
const TYPE_META = {
  notice: { label: "通知", icon: "\u{f2a2}", tone: "notice" }, // volume-up-line
  leave: { label: "请假", icon: "\u{eb23}", tone: "leave" }, // calendar-check-line
  pickup: { label: "接送", icon: "\u{eeda}", tone: "pickup" }, // logout-box-r-line
  growth: { label: "成长", icon: "\u{eb31}", tone: "growth" }, // camera-line
  task: { label: "任务", icon: "\u{f1e8}", tone: "task" }, // task-line
  system: { label: "系统", icon: "\u{f0e6}", tone: "system" } // settings-3-line
}

// 未知类型不猜语义：保留后端原码，图标退回中性收件箱。
const UNKNOWN_META = { label: "", icon: "\u{ee4f}", tone: "system" } // inbox-line

export function messageTypeMeta(type) {
  const code = String(type || "")
  const hit = TYPE_META[code]
  if (hit) return hit
  return { ...UNKNOWN_META, label: code }
}

export function messageTypeLabel(type) {
  // 空 type 很少见，但空标签会让 meta 行只剩时间，看起来像渲染缺失。
  return messageTypeMeta(type).label || "消息"
}

function yesterdayKey(now) {
  // 用本地日期构造而不是减 86400000：跨夏令时也不会偏一天。
  return localDateKey(new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1))
}

/**
 * 按本地日期把消息分成「今天 / 昨天 / 更早」三组，组内保持接口的 created_at DESC。
 * 接口 created_at 是 UTC，必须先转本地日期再比较，直接截字符串会跨零点错一天。
 */
export function groupMessagesByDay(list) {
  const now = new Date()
  const todayKey = localDateKey(now)
  const prevKey = yesterdayKey(now)
  const buckets = { today: [], yesterday: [], earlier: [] }

  for (const item of list || []) {
    const key = localDateKey(item?.created_at)
    if (key && key === todayKey) buckets.today.push(item)
    else if (key && key === prevKey) buckets.yesterday.push(item)
    else buckets.earlier.push(item)
  }

  return [
    { key: "today", label: "今天", items: buckets.today },
    { key: "yesterday", label: "昨天", items: buckets.yesterday },
    { key: "earlier", label: "更早", items: buckets.earlier }
  ].filter(group => group.items.length > 0)
}

/** 今天/昨天只显示时刻（HH:mm），更早显示「10月24日」 */
export function messageTimeText(ts) {
  const key = localDateKey(ts)
  if (!key) return ""
  const now = new Date()
  if (key === localDateKey(now) || key === yesterdayKey(now)) return formatClock(ts)
  return formatDateLabel(key)
}
