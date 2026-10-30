// 亲子任务的枚举展示与状态映射。
// 后端只透传英文 key（frequency / recognition_ritual / public_status），
// 列表与详情共用这里的口径，避免两页对同一状态给出不同说法。

export const FREQ_OPTIONS = [
  { value: "once", label: "一次性" },
  { value: "daily", label: "每日" },
  { value: "weekly", label: "每周" }
]

export const RITUAL_OPTIONS = [
  { value: "sticker", label: "贴纸" },
  { value: "voice", label: "语音" },
  { value: "class_star", label: "班级星星" }
]

const FREQ_LABELS = Object.fromEntries(FREQ_OPTIONS.map((item) => [item.value, item.label]))
const RITUAL_LABELS = Object.fromEntries(RITUAL_OPTIONS.map((item) => [item.value, item.label]))

export function freqLabel(value) {
  return FREQ_LABELS[value] || "未设置"
}

export function ritualLabel(value) {
  return RITUAL_LABELS[value] || "未设置"
}

// 契约：进行中 = status=1 且 end_date >= 今天（Asia/Shanghai）。
// 不能只看 status：后台把任务标成进行中但日期已过时，不算进行中。
export function isTaskActive(task, today) {
  if (!task || Number(task.status) !== 1) return false
  const end = String(task.end_date || "").slice(0, 10)
  if (!end || !today) return true
  return end >= today
}

export function taskStatusLabel(task, today) {
  return isTaskActive(task, today) ? "进行中" : "已结束"
}

// 星星是否值得展示：班级罐任务看贡献星数，任务罐要求目标 > 0（D6）。
export function starTracking(task) {
  if (!task) return false
  if (task.star_scope === "class") return true
  return Number(task.star_goal || 0) > 0
}

// 列表卡片星条：star_count 为 null 表示「没拿到」，必须隐藏而不是显示 0。
export function taskStarText(task) {
  if (!task || task.star_count == null) return ""
  const count = Number(task.star_count)
  if (task.star_scope === "class") return `班级星星罐 · 本任务已集 ${count} 星`
  const goal = Number(task.star_goal || 0)
  if (goal <= 0) return ""
  if (count > goal) return `星星已超额 · ${count}/${goal}`
  if (count === goal) return `星星已集满 · ${count}/${goal}`
  return `${count}/${goal}`
}

// public_status 的四种取值；none 表示家长从未申请公开，审核块不渲染。
export const REVIEW_FILTERS = [
  { key: "all", label: "全部" },
  { key: "pending", label: "待审核" },
  { key: "approved", label: "已公开" },
  { key: "revoked", label: "未公开" }
]

export function reviewKey(row) {
  const state = row?.public_status
  return state === "pending" || state === "approved" || state === "revoked" ? state : "none"
}

export function publicStateText(row) {
  if (row?.public_status === "pending") return "家长申请公开，待你审核"
  if (row?.public_status === "approved") {
    return row.public_visible ? "已公开 · 本班家长可见" : "已通过，暂不对本班展示"
  }
  // 老师拒绝与家长取消写的是同一个 revoked_at，不替任何一方下结论。
  if (row?.public_status === "revoked") return "未公开"
  return ""
}

// approved 不等于此刻在墙上：任务结束/停用、关闭公开、改成个体任务、
// 孩子转班都会让 public_visible=false，原因必须说清楚。
export function publicStateHint(row) {
  if (row?.public_status !== "approved" || row.public_visible) return ""
  return "任务已结束或停用、未开启公开、已改成个体任务，或孩子已不在本班，本班家长暂时看不到"
}

// sticker_key 是后端透传的英文 key，客户端只做展示映射。
// 用 Remix Icon 字形而不是 emoji：emoji 在不同机型上字形/大小不可控。
const glyph = (code) => String.fromCharCode(code)
const STICKER_GLYPHS = {
  star: glyph(0xf186),
  heart: glyph(0xee0e),
  thumb: glyph(0xf206),
  flower: glyph(0xf409),
  smile: glyph(0xec8c),
  clap: glyph(0xedf2)
}

export function stickerGlyph(key) {
  return STICKER_GLYPHS[key] || STICKER_GLYPHS.star
}
