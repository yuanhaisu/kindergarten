// 消息聚合列表的展示映射（类型 / 分段 / 相对时间）。
// 契约见 backend/api/api-teacher/10-messages.md：type 只有 notice / leave / system，
// is_read 是老师维度读状态，unread_count 是全量口径。

import { formatClock } from "./time.js"

export const MESSAGE_SEGMENTS = [
  { key: "all", label: "全部" },
  { key: "notice", label: "通知" },
  { key: "leave", label: "审批" },
  { key: "system", label: "系统" }
]

// remixicon PUA 字形（App.vue 已加载 remixicon 字体）。
// 设计稿用内联 SVG，小程序 WXML 不允许，改用字体图标。
const TYPE_META = {
  notice: { label: "通知", icon: "" }, // megaphone-line
  leave: { label: "审批", icon: "" }, // calendar-check-line
  system: { label: "系统", icon: "" } // server-line
}

const LEAVE_STATUS_TONE = {
  0: "pending",
  1: "approved",
  2: "rejected"
}

export function messageTypeMeta(type) {
  return TYPE_META[type] || TYPE_META.system
}

export function leaveStatusTone(status) {
  return LEAVE_STATUS_TONE[Number(status)] || "muted"
}

// 通知明细里的已读口径：契约同时给 read_rate 与 read_count/total_count，
// 优先用服务端算好的 read_rate，缺失时再按计数兜底。
export function noticeReadText(extra) {
  const rate = String(extra?.read_rate || "").trim()
  if (rate) return `已读 ${rate}`
  const readCount = Number(extra?.read_count)
  const totalCount = Number(extra?.total_count)
  if (Number.isFinite(readCount) && Number.isFinite(totalCount) && totalCount > 0) {
    return `已读 ${Math.round((readCount * 100) / totalCount)}%`
  }
  return ""
}

function iconToneOf(type, extra) {
  if (type === "notice") return "notice"
  if (type === "system") return "system"
  return leaveStatusTone(extra?.status)
}

function detailOf(type, extra) {
  if (type === "notice") {
    const text = [extra?.category_label, extra?.status_label, noticeReadText(extra)]
      .filter(Boolean)
      .join(" · ")
    return { text, tone: "" }
  }
  if (type === "leave") {
    return { text: String(extra?.status_label || ""), tone: leaveStatusTone(extra?.status) }
  }
  return { text: "", tone: "" }
}

export function normalizeMessage(raw) {
  const type = TYPE_META[raw?.type] ? raw.type : "system"
  const extra = raw?.extra && typeof raw.extra === "object" ? raw.extra : {}
  return {
    id: Number(raw?.reference_id) || 0,
    type,
    title: String(raw?.title || "").trim() || "未命名消息",
    content: String(raw?.content || "").trim(),
    extra,
    isRead: raw?.is_read === true,
    timeText: formatMessageTime(raw?.created_at),
    icon: TYPE_META[type].icon,
    iconTone: iconToneOf(type, extra),
    badge: TYPE_META[type].label,
    detail: detailOf(type, extra)
  }
}

export function segmentCount(list, key) {
  if (key === "all") return list.length
  return list.reduce((total, item) => total + (item.type === key ? 1 : 0), 0)
}

export function segmentUnread(list, key) {
  return list.reduce((total, item) => {
    if (item.isRead) return total
    return total + (key === "all" || item.type === key ? 1 : 0)
  }, 0)
}

/**
 * 列表相对时间：今天 → HH:mm；昨天 → 昨天 HH:mm；更早 → M月D日。
 * 一律走本地时区——后端 created_at 是 UTC，截字符串会差 8 小时。
 */
export function formatMessageTime(iso) {
  if (!iso) return ""
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ""
  const now = new Date()
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const targetStart = new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
  const gap = Math.round((todayStart - targetStart) / 86400000)
  const clock = formatClock(iso)
  if (gap === 0) return clock
  if (gap === 1) return `昨天 ${clock}`
  return `${date.getMonth() + 1}月${date.getDate()}日`
}
