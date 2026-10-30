// 积分流水 / 成就徽章 / 意见反馈的展示映射。
// 中文标签优先用后端返回值（B16 已补齐 type_label、source_label、
// category_label、status_label），这里的表只负责本地兜底与样式归类，
// 不重复发明业务枚举（见 backend/api/api-parent/07-points.md、10-feedback.md）。

/** 流水方向：以 amount 符号为准，type 缺失时也能判断 */
export function historyTone(item) {
  if (item?.type === "earn" || item?.type === "spend") return item.type
  return Number(item?.amount) < 0 ? "spend" : "earn"
}

/** 流水标题：source_label 优先，缺省时退回 source 原值 */
export function historyTitle(item) {
  return item?.source_label || item?.source || "积分变动"
}

/** 带符号分值："+10" / "-20"（后端 amount 已带符号，0 原样显示） */
export function signedAmount(amount) {
  const n = Number(amount) || 0
  return n > 0 ? `+${n}` : String(n)
}

const FEEDBACK_CATEGORY_META = {
  suggestion: { label: "建议", tone: "info" },
  complaint: { label: "投诉", tone: "warning" },
  bug: { label: "问题", tone: "danger" }
}

export const FEEDBACK_CATEGORIES = [
  { key: "suggestion", ...FEEDBACK_CATEGORY_META.suggestion },
  { key: "complaint", ...FEEDBACK_CATEGORY_META.complaint },
  { key: "bug", ...FEEDBACK_CATEGORY_META.bug }
]

/** 反馈类型：category_label 优先；未知类型按中性 info 兜底，不吞成空白 */
export function feedbackCategory(category, label) {
  const meta = FEEDBACK_CATEGORY_META[String(category || "")] || { label: "", tone: "info" }
  return { label: label || meta.label || String(category || "反馈"), tone: meta.tone }
}

const FEEDBACK_STATUS_META = {
  0: { label: "已提交", tone: "submitted" },
  1: { label: "已回复", tone: "replied" },
  2: { label: "已关闭", tone: "closed" }
}

/** 反馈状态：status_label 优先；未知状态按「已提交」兜底 */
export function feedbackStatus(status, label) {
  const meta = FEEDBACK_STATUS_META[Number(status)] || FEEDBACK_STATUS_META[0]
  return { label: label || meta.label, tone: meta.tone }
}

// 成就没有稳定的图标枚举：category 是自由文本、icon_url 可能为空。
// 按关键词归到页面已有的 remixicon 字形，未命中用勋章兜底。
const BADGE_ICON_RULES = [
  ["attend", "calendar"],
  ["check", "calendar"],
  ["read", "book"],
  ["study", "book"],
  ["sport", "health"],
  ["health", "health"],
  ["art", "image"],
  ["growth", "image"],
  ["photo", "image"],
  ["manner", "heart"],
  ["share", "heart"],
  ["care", "heart"],
  ["task", "task"],
  ["labor", "task"]
]

export function badgeIconKey(category) {
  const key = String(category || "").toLowerCase()
  const hit = BADGE_ICON_RULES.find(([needle]) => key.includes(needle))
  return hit ? hit[1] : "medal"
}
