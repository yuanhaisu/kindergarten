// 亲子任务的前端展示映射。
// 标签以 backend/api/api-parent/06-task.md 为唯一来源：
// - frequency：daily 每日 / weekly 每周 / once 一次性（monthly 仅为旧数据兜底）；
// - 列表三态：submitted_current_period + responded → 待完成 / 已提交 / 已回应；
// - 公开状态：服务端算好的 none / pending / approved / revoked，
//   详情 current_checkin 直接带 public_status，打卡历史按 public_* 时间字段推导。

const FREQUENCY_LABELS = {
  daily: "每日",
  weekly: "每周",
  monthly: "每月",
  once: "一次性"
}

/** 任务频次：「每日」；未知值回退「亲子」，与列表副标题拼成「亲子任务」 */
export function frequencyLabel(frequency) {
  return FREQUENCY_LABELS[String(frequency || "")] || "亲子"
}

/** 任务频次 + 任务：「每日任务」/「亲子任务」 */
export function frequencyTaskLabel(frequency) {
  return `${frequencyLabel(frequency)}任务`
}

/** 打卡区说明：与后端频率限制同一口径（每日任务当天一次等） */
export function checkinPeriodHint(frequency) {
  switch (frequency) {
    case "daily":
      return "每日任务每天可提交一次"
    case "weekly":
      return "每周任务每 7 天可提交一次"
    case "once":
      return "一次性任务仅可提交一次"
    default:
      return "每个周期可提交一次"
  }
}

/** 星星罐：task 显示目标星数，class 只写「班级星星罐」（接口不返回班级累计） */
export function starJarLabel(item) {
  if (item?.star_scope === "class") return "班级星星罐"
  const goal = Number(item?.star_goal) || 0
  return goal > 0 ? `星星罐 ${goal} 星` : "星星罐"
}

/** 详情任务卡的星星格：只保留数值部分，标签由页面固定为「星星罐」 */
export function starJarValue(item) {
  if (item?.star_scope === "class") return "班级星星罐"
  const goal = Number(item?.star_goal) || 0
  return goal > 0 ? `${goal} 星` : "—"
}

/** 列表徽标：已结束优先，避免已结束任务仍显示「待完成」 */
export function taskBadge(item) {
  if (Number(item?.status) === 0) return { label: "已结束", tone: "ended" }
  if (item?.responded) return { label: "已回应", tone: "reply" }
  if (item?.submitted_current_period) return { label: "已提交", tone: "done" }
  return { label: "待完成", tone: "todo" }
}

/** 周期状态行：本周期是否已提交的二元事实，已结束单独一态 */
export function cycleState(item) {
  if (Number(item?.status) === 0) return { label: "任务已结束", tone: "ended" }
  if (item?.responded) return { label: "老师已回应", tone: "reply" }
  if (item?.submitted_current_period) return { label: "本周期已提交", tone: "done" }
  return { label: "本周期待提交", tone: "todo" }
}

/** 主按钮：已结束任务禁用，避免对已结束任务发起提交 */
export function primaryAction(item) {
  if (Number(item?.status) === 0) return { label: "已结束", disabled: true }
  return item?.submitted_current_period
    ? { label: "查看记录", disabled: false }
    : { label: "去提交", disabled: false }
}

export const PUBLIC_STATUS = {
  NONE: "none",
  PENDING: "pending",
  APPROVED: "approved",
  REVOKED: "revoked"
}

/**
 * 公开状态：优先读服务端 public_status（current_checkin 直接返回），
 * 打卡历史只有 public_* 时间字段，按与后端 publicState 相同的顺序推导。
 */
export function publicStateOf(row) {
  const explicit = row?.public_status
  if (
    explicit === PUBLIC_STATUS.NONE ||
    explicit === PUBLIC_STATUS.PENDING ||
    explicit === PUBLIC_STATUS.APPROVED ||
    explicit === PUBLIC_STATUS.REVOKED
  ) {
    return explicit
  }
  if (row?.public_revoked_at) return PUBLIC_STATUS.REVOKED
  if (row?.public_approved_at) return PUBLIC_STATUS.APPROVED
  if (row?.is_public_requested) return PUBLIC_STATUS.PENDING
  return PUBLIC_STATUS.NONE
}

/** 公开徽标：none 不显示徽标（源码口径） */
export function publicBadge(state) {
  switch (state) {
    case PUBLIC_STATUS.PENDING:
      return { label: "待审核", tone: "pending" }
    case PUBLIC_STATUS.APPROVED:
      return { label: "老师已通过", tone: "ok" }
    case PUBLIC_STATUS.REVOKED:
      return { label: "未公开", tone: "muted" }
    default:
      return null
  }
}

/** 公开状态说明：已通过要区分「真的在墙上」（public_visible） */
export function publicNote(state, visible) {
  if (state === PUBLIC_STATUS.PENDING) return "已申请，等待老师审核"
  if (state === PUBLIC_STATUS.APPROVED) return visible ? "本班家长可见" : "暂不对本班展示"
  return ""
}

/** 公开动作文案：取消公开是两步确认，确认态由页面传入 */
export function publicActionLabel(state, confirming) {
  if (state === PUBLIC_STATUS.NONE) return "申请公开"
  if (state === PUBLIC_STATUS.REVOKED) return "重新申请"
  return confirming ? "确认撤下？" : "取消公开"
}

/** 当前是否处于「可撤回」的审核态（pending / approved）；撤回永远可用 */
export function isRevoking(state) {
  return state === PUBLIC_STATUS.PENDING || state === PUBLIC_STATUS.APPROVED
}

export const REACTIONS = [
  { key: "child_happy", label: "孩子很开心" },
  { key: "received", label: "收到了" },
  { key: "thanks", label: "谢谢老师" }
]

const RITUAL_LABELS = {
  sticker: "贴纸",
  voice: "语音",
  class_star: "班级星星",
  text: "老师的话"
}

/** 回应形式标签：贴纸 / 语音 / 班级星星 / 老师的话 */
export function ritualLabel(ritual) {
  return RITUAL_LABELS[String(ritual || "")] || "回应"
}

// sticker_key 是后端透传的英文 key，客户端只做展示映射。
const STICKER_GLYPHS = {
  star: "⭐",
  heart: "❤️",
  thumb: "👍",
  flower: "🌸",
  smile: "😊",
  clap: "👏"
}

export function stickerGlyph(key) {
  return STICKER_GLYPHS[key] || "🌟"
}

/** 回应卡左侧的图形位：贴纸用 sticker_key，其它形式给一个稳定的 emoji */
export function responseGlyph(response) {
  if (response?.ritual === "sticker") return stickerGlyph(response.sticker_key)
  if (response?.ritual === "voice") return "🎤"
  if (response?.ritual === "class_star") return "⭐"
  return "💬"
}
