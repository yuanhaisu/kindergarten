// 时间展示统一按本地时区。
// 后端返回 UTC（created_at 形如 2026-03-19T11:30:00Z），直接 slice(11, 16)
// 会把中国时区的时间显示成早 8 小时，跨零点时日期也会错一天。
// 纯日期串（YYYY-MM-DD）不经过 Date：new Date("2026-10-20") 按 UTC 解析，
// 负时区会倒退到前一天。

function pad(n) {
  return String(n).padStart(2, "0")
}

function toDate(ts) {
  if (!ts) return null
  const d = new Date(ts)
  return Number.isNaN(d.getTime()) ? null : d
}

/** ISO 时刻 → "15:20"（本地时区） */
export function formatClock(ts) {
  const d = toDate(ts)
  return d ? `${pad(d.getHours())}:${pad(d.getMinutes())}` : ""
}

/** 纯日期 → "10月20日"；格式不符返回空串，由调用方决定兜底 */
export function formatDateLabel(date) {
  const m = String(date || "").match(/^(\d{4})-(\d{2})-(\d{2})/)
  return m ? `${Number(m[2])}月${Number(m[3])}日` : ""
}

/** 记录卡头部："10月24日 15:20"（日期取服务端 date，时刻取本地 created_at） */
export function formatRecordTime(date, createdAt) {
  return [formatDateLabel(date), formatClock(createdAt)].filter(Boolean).join(" ")
}

/** 评论时间："10月22日 10:26"；不是今年时带上年份 */
export function formatCommentTime(ts) {
  const d = toDate(ts)
  if (!d) return ""
  const prefix = d.getFullYear() === new Date().getFullYear() ? "" : `${d.getFullYear()}年`
  return `${prefix}${d.getMonth() + 1}月${d.getDate()}日 ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** "2026-10" → "2026年10月"；用于月分组标题与相册月份条 */
export function formatMonthLabel(ym) {
  const m = String(ym || "").match(/^(\d{4})-(\d{2})/)
  return m ? `${m[1]}年${Number(m[2])}月` : ""
}

/** 记录所属月份 key："2026-10" */
export function monthKeyOf(date) {
  return String(date || "").slice(0, 7)
}

/**
 * 时光轴年份胶囊。
 * 不写死年份：只有一年时显示「2026年」，跨年时显示「2025–2026」，
 * 让这一格承担「这批记录覆盖哪几年」的信息，而不是重复月份标题里的年份。
 */
export function formatYearRange(dates) {
  const years = [...new Set((dates || []).map(d => String(d || "").slice(0, 4)).filter(Boolean))].sort()
  if (years.length === 0) return `${new Date().getFullYear()}年`
  if (years.length === 1) return `${years[0]}年`
  return `${years[0]}–${years[years.length - 1]}`
}

/** 相册月份条：「10月9日 - 10月24日」；传入升序日期，单日只显示一天 */
export function formatDateRange(dates) {
  const labels = (dates || []).map(formatDateLabel).filter(Boolean)
  if (labels.length === 0) return ""
  if (labels.length === 1) return labels[0]
  return `${labels[0]} - ${labels[labels.length - 1]}`
}
