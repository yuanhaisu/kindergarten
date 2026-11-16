// 授权到期提示的纯逻辑，独立成模块以便 Node 直接跑边界测试。
export const EXPIRY_WARNING_DAYS = 30

const DAY_MS = 24 * 60 * 60 * 1000
const EXPIRED_HINT = "套餐已到期，部分功能仅可查看，请联系园所管理员续费"

/**
 * 先按时间戳判断是否已过期，再计算剩余天数窗口。
 * 不能只靠 ceil(diff/DAY_MS)：到期后不足一天时 ceil 仍为 0，
 * 会把已过期的园所误显示为「将于某日到期」。
 */
export function expiryHintText(expiresAt, now = Date.now(), renewHint = "") {
  if (!expiresAt) return ""
  const at = new Date(expiresAt)
  if (Number.isNaN(at.getTime())) return ""
  const diff = at.getTime() - now
  if (diff <= 0) return EXPIRED_HINT
  const days = Math.ceil(diff / DAY_MS)
  if (days <= EXPIRY_WARNING_DAYS) {
    return `套餐将于 ${String(expiresAt).slice(0, 10)} 到期，${renewHint}`
  }
  return ""
}
