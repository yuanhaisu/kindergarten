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

/**
 * 纯日期 → "2022-05-18"；格式不符（含后端零值下发的空串）返回空串，
 * 由调用方用「—」占位。不经过 Date，避免时区把日期推前一天。
 */
export function formatDateOnly(date) {
  const m = String(date || "").match(/^(\d{4})-(\d{2})-(\d{2})/)
  // Go 零值 time.Time 若漏过服务端过滤会下发 0001-01-01，它不是真实日期。
  if (!m || m[1] === "0001") return ""
  return `${m[1]}-${m[2]}-${m[3]}`
}

/**
 * 日期分组标题：今天是「今天」、昨天是「昨天」，其余「10月24日」。
 * 按设备本地日期比较，避免把 UTC 的 created_at 直接 slice 后跨零点错一天。
 */
export function formatDayGroupLabel(date) {
  const key = String(date || "").slice(0, 10)
  if (!/^\d{4}-\d{2}-\d{2}$/.test(key)) return formatDateLabel(date)
  const now = new Date()
  const todayKey = localDateKey(now)
  const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1)
  if (key === todayKey) return "今天"
  if (key === localDateKey(yesterday)) return "昨天"
  return formatDateLabel(key)
}

/**
 * 纯日期 → 本地零点 Date；格式不符返回 null。
 * 不能用 new Date("2026-10-20")：它按 UTC 解析，负时区会倒退一天。
 */
export function parseDateOnly(date) {
  const m = String(date || "").match(/^(\d{4})-(\d{2})-(\d{2})/)
  if (!m) return null
  return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]))
}

const WEEKDAY_NAMES = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"]

/** 纯日期 → "周三"；格式不符返回空串 */
export function formatWeekday(date) {
  const d = parseDateOnly(date)
  return d ? WEEKDAY_NAMES[d.getDay()] : ""
}

/** 纯日期 → "2026-10-28 周三"；格式不符返回空串 */
export function formatDateWithWeekday(date) {
  const d = parseDateOnly(date)
  if (!d) return ""
  return `${String(date).slice(0, 10)} ${WEEKDAY_NAMES[d.getDay()]}`
}

/** 纯日期 → "10月27日 周二" */
export function formatMonthDayWeekday(date) {
  const label = formatDateLabel(date)
  const weekday = formatWeekday(date)
  return label && weekday ? `${label} ${weekday}` : label
}

/** 纯日期区间含首尾的天数；任一缺失或结束早于开始返回 0 */
export function inclusiveDays(start, end) {
  const a = parseDateOnly(start)
  const b = parseDateOnly(end)
  if (!a || !b || b < a) return 0
  return Math.round((b - a) / 86400000) + 1
}

/** ISO 时刻 → 本地日期 "2026-10-26"；空值/非法返回空串（审批时间按本地日期显示） */
export function localDateKey(ts) {
  const d = toDate(ts)
  if (!d) return ""
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** 年月（month 为 0 起）→ "2026年10月" */
export function formatMonthTitle(year, month) {
  return `${year}年${month + 1}月`
}

/** 记录卡头部："10月24日 15:20"（日期取服务端 date，时刻取本地 created_at） */
export function formatRecordTime(date, createdAt) {
  return [formatDateLabel(date), formatClock(createdAt)].filter(Boolean).join(" ")
}

/** ISO 时刻 → "10月25日 20:30"（本地时区）；不是今年时带上年份 */
export function formatDateTimeLabel(ts) {
  const d = toDate(ts)
  if (!d) return ""
  const prefix = d.getFullYear() === new Date().getFullYear() ? "" : `${d.getFullYear()}年`
  return `${prefix}${d.getMonth() + 1}月${d.getDate()}日 ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** 评论时间："10月22日 10:26"；不是今年时带上年份 */
export function formatCommentTime(ts) {
  return formatDateTimeLabel(ts)
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
