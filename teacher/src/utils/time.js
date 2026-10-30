// 时间展示统一按本地时区。
// 后端 created_at 是 UTC（形如 2026-03-19T11:30:00Z），直接 slice(11, 16)
// 会把东八区时间显示成早 8 小时；纯日期串（YYYY-MM-DD）也按 UTC 解析会跨零点倒退一天。
// 因此纯日期一律手工拆字符串，时刻一律交给 Date 走本地时区。

const WEEK_CN = ["日", "一", "二", "三", "四", "五", "六"]

function pad(n) {
  return String(n).padStart(2, "0")
}

function toDate(ts) {
  if (!ts) return null
  const d = new Date(ts)
  return Number.isNaN(d.getTime()) ? null : d
}

// "2026-10-24" → 本地零点的 Date；不经过 new Date("2026-10-24")（那是 UTC 解析）
function fromYmd(date) {
  const m = String(date || "").match(/^(\d{4})-(\d{2})-(\d{2})/)
  if (!m) return null
  return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]))
}

function toYmd(d) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** 今天（本地时区）的 YYYY-MM-DD */
export function todayKey() {
  return toYmd(new Date())
}

/** 纯日期 → "10月24日"；格式不符返回空串，由调用方兜底 */
export function formatDateLabel(date) {
  const d = fromYmd(date)
  return d ? `${d.getMonth() + 1}月${d.getDate()}日` : ""
}

/** 纯日期 → "星期六" */
export function formatWeekdayLong(date) {
  const d = fromYmd(date)
  return d ? `星期${WEEK_CN[d.getDay()]}` : ""
}

/** 纯日期 → "周六" */
export function formatWeekdayShort(date) {
  const d = fromYmd(date)
  return d ? `周${WEEK_CN[d.getDay()]}` : ""
}

/** ISO 时刻 → "15:20"（本地时区） */
export function formatClock(ts) {
  const d = toDate(ts)
  return d ? `${pad(d.getHours())}:${pad(d.getMinutes())}` : ""
}

/** ISO 时刻 → "10-24"（本地时区，月日补零）；用于通知卡等只给月日的场景 */
export function formatMonthDay(ts) {
  const d = toDate(ts)
  return d ? `${pad(d.getMonth() + 1)}-${pad(d.getDate())}` : ""
}

/** 相对今天的日差：0 今天 / 1 昨天 / 2 前天 / 其他 null */
function relativeDay(date) {
  const target = fromYmd(date)
  const today = fromYmd(todayKey())
  if (!target || !today) return null
  const diff = Math.round((today.getTime() - target.getTime()) / 86400000)
  return diff >= 0 && diff <= 2 ? diff : null
}

/** 列表头部：{ main: "10月24日", weekday: "星期六" } */
export function formatHeadlineDate(date = todayKey()) {
  return { main: formatDateLabel(date), weekday: formatWeekdayLong(date) }
}

/** 创建页日期锚点：{ main: "10月24日 周六", sub: "2026年 · 今天" } */
export function formatDateAnchor(date) {
  const d = fromYmd(date)
  if (!d) return { main: "", sub: "" }
  const rel = relativeDay(date)
  const relText = rel === 0 ? "今天" : rel === 1 ? "昨天" : rel === 2 ? "前天" : ""
  return {
    main: `${formatDateLabel(date)} ${formatWeekdayShort(date)}`,
    sub: `${d.getFullYear()}年${relText ? ` · ${relText}` : ""}`
  }
}

/** 详情作者行："10月24日 15:20"（日期取记录 date，时刻取本地 created_at） */
export function formatRecordTime(date, createdAt) {
  return [formatDateLabel(date), formatClock(createdAt)].filter(Boolean).join(" ")
}

/**
 * 列表卡片日期：今天 → "今天 11:20"；其他 → "2026-10-24 11:20"（记录日期 + 本地时刻）。
 * 日期取记录的 date（教学日），时刻取 created_at 的本地时区——两者语义不同，不互相替换。
 */
export function formatCardTime(date, createdAt) {
  const clock = formatClock(createdAt)
  if (date && date === todayKey()) return clock ? `今天 ${clock}` : "今天"
  const created = toDate(createdAt)
  const day = String(date || "").slice(0, 10) || (created ? toYmd(created) : "")
  return [day, clock].filter(Boolean).join(" ")
}

/** 时间戳所在本地日到今天的差：0 今天 / 1 昨天 / 其他整数（未来为负）/ null 无法解析 */
function dayGapFromNow(ts) {
  const d = toDate(ts)
  if (!d) return null
  const target = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  const today = fromYmd(todayKey())
  if (!today) return null
  return Math.round((today.getTime() - target.getTime()) / 86400000)
}

/**
 * 列表时间戳：今天 09:12 / 昨天 20:15 / 10月24日 09:12；跨年只给年月日。
 * 时刻走本地时区——后端 created_at 是 UTC，直接截字符串会差 8 小时。
 */
export function formatRelativeStamp(ts) {
  const d = toDate(ts)
  if (!d) return ""
  const gap = dayGapFromNow(ts)
  const clock = formatClock(ts)
  if (gap === 0) return clock ? `今天 ${clock}` : "今天"
  if (gap === 1) return clock ? `昨天 ${clock}` : "昨天"
  const day = `${d.getMonth() + 1}月${d.getDate()}日`
  if (d.getFullYear() !== new Date().getFullYear()) return `${d.getFullYear()}年${day}`
  return clock ? `${day} ${clock}` : day
}

/** 列表日期（不带时刻）：今天 / 昨天 / 10月24日 */
export function formatRelativeDay(ts) {
  const d = toDate(ts)
  if (!d) return ""
  const gap = dayGapFromNow(ts)
  if (gap === 0) return "今天"
  if (gap === 1) return "昨天"
  return `${d.getMonth() + 1}月${d.getDate()}日`
}

/** 评论时间："10月24日 15:42"；不是今年时带上年份 */
export function formatCommentTime(ts) {
  const d = toDate(ts)
  if (!d) return ""
  const prefix = d.getFullYear() === new Date().getFullYear() ? "" : `${d.getFullYear()}年`
  return `${prefix}${d.getMonth() + 1}月${d.getDate()}日 ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** "2026-10" → "2026年10月"；用于月分组标题 */
export function formatMonthLabel(ym) {
  const m = String(ym || "").match(/^(\d{4})-(\d{2})/)
  return m ? `${m[1]}年${Number(m[2])}月` : ""
}

/** 记录所属月份 key："2026-10" */
export function monthKeyOf(date) {
  return String(date || "").slice(0, 7)
}

/** 纯日期加减天数；输入非法时原样返回 */
export function addDays(date, n) {
  const d = fromYmd(date)
  if (!d) return date
  d.setDate(d.getDate() + n)
  return toYmd(d)
}
