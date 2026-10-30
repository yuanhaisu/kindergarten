// 班级日历的展示映射（老师端三页共用）。
//
// 契约 backend/api/api-teacher/08-calendar.md 明确：event_type_label 与 color
// 由服务端按园所字典 calendar_event_type 解析下发，客户端不得硬编码类型 → 颜色映射。
// 因此本文件只做文案/排序/日期格式化，任何颜色都来自接口返回值（或字典 extra.color），
// 不提供内置类型的兜底色。

// 文案与日期格式复用 utils/time，避免三页各写一套周几/月日换算。
import { formatDateLabel, formatWeekdayShort } from "./time.js"

export function pad2(n) {
  return String(n).padStart(2, "0")
}

/** 纯日期 → "10月26日"（复用 time.js） */
export const formatMonthDay = formatDateLabel

/** 纯日期 → "周一"（复用 time.js） */
export const weekdayShort = formatWeekdayShort

/** 纯日期串（YYYY-MM-DD）→ { y, m, d }；非法返回 null。手工拆字符串，避免 UTC 解析跨零点 */
export function parseYmd(date) {
  const match = String(date || "").match(/^(\d{4})-(\d{1,2})-(\d{1,2})/)
  if (!match) return null
  const y = Number(match[1])
  const m = Number(match[2])
  const d = Number(match[3])
  if (m < 1 || m > 12 || d < 1 || d > daysInMonth(y, m)) return null
  return { y, m, d }
}

export function formatYmd(y, m, d) {
  return `${y}-${pad2(m)}-${pad2(d)}`
}

export function daysInMonth(year, month) {
  return new Date(year, month, 0).getDate()
}

/** 纯日期 → "10月26日 周一" */
export function formatMonthDayWeek(date) {
  const day = formatMonthDay(date)
  const week = weekdayShort(date)
  return [day, week].filter(Boolean).join(" ")
}

/** 纯日期 → "2026年10月26日 周一"（详情信息卡） */
export function formatDateCn(date) {
  const p = parseYmd(date)
  if (!p) return ""
  return `${p.y}年${p.m}月${p.d}日 ${weekdayShort(date)}`
}

/** 列表标题旁的状态 chip：只有草稿需要标；已发布是默认值，不额外占位 */
export function statusChipText(status) {
  return status === "draft" ? "草稿" : ""
}

/** 详情状态标签：家长可见性是关键信息，草稿要说清「家长不可见」 */
export function statusDetailText(status) {
  return status === "draft" ? "草稿（家长不可见）" : "已发布"
}

/** 参与范围文案；本班是默认值 */
export function scopeText(scope) {
  return scope === "kindergarten" ? "全园" : "本班"
}

/** 事件时间：全天 > 起止 > 单边 > 空串（调用方决定是否隐藏分隔符） */
export function eventTimeText(event) {
  if (!event) return ""
  if (event.is_all_day) return "全天"
  const start = event.start_time || ""
  const end = event.end_time || ""
  if (start && end) return `${start}–${end}`
  return start || end || ""
}

/** 类型名：优先服务端字典解析值，再退回原始 key；绝不内置映射 */
export function typeLabelOf(event, dictMeta = {}) {
  if (!event) return ""
  const meta = dictMeta[event.event_type]
  return event.event_type_label || meta?.label || event.event_type || ""
}

/** 类型色：优先事件随包下发的 color，其次字典 extra.color；空值由调用方走中性兜底 */
export function typeColorOf(event, dictMeta = {}) {
  if (!event) return ""
  return event.color || dictMeta[event.event_type]?.color || ""
}

/** 8 位 hex 的浅底；颜色来自接口，仅做透明度派生，不引入新色值 */
export function typeTint(color) {
  return color ? `${color}1F` : ""
}

/** 全天在前，其余按开始时间升序；无开始时间的排在有时刻的后面 */
export function sortEvents(list) {
  return [...(list || [])].sort((a, b) => {
    const aAllDay = !!a?.is_all_day
    const bAllDay = !!b?.is_all_day
    if (aAllDay !== bAllDay) return aAllDay ? -1 : 1
    const at = a?.start_time || "99:99"
    const bt = b?.start_time || "99:99"
    return String(at).localeCompare(String(bt))
  })
}

/** 描述按空行分段（详情页正文与大段预览共用） */
export function splitParagraphs(text) {
  return String(text || "")
    .split(/\n\s*\n/)
    .map((part) => part.trim())
    .filter(Boolean)
}
