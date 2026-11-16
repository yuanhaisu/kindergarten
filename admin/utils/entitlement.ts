const DAY_MS = 24 * 60 * 60 * 1000

export interface ExpiryNotice {
  /** 距到期还剩的天数（向上取整，不足一天计 1） */
  daysLeft: number
  /** 到期日，取服务端时间戳的日期部分 */
  expiresOn: string
}

/**
 * 「到期前 30 天转付费提示」的判定。
 *
 * 只返回「仍在期、且剩余不超过 thresholdDays 天」的授权；已过期（含宽限期）
 * 返回 null，由页面上既有的到期/降级提示承担，避免同一条到期事实叠加两条提示。
 * 用毫秒差而非按天取整判过期：剩余不足一天时取整结果为 -0，会被误当成在期。
 */
export function expiringSoon(
  expiresAt?: string | null,
  now: Date = new Date(),
  thresholdDays = 30,
): ExpiryNotice | null {
  if (!expiresAt) return null
  const expires = new Date(expiresAt)
  if (Number.isNaN(expires.getTime())) return null
  const diff = expires.getTime() - now.getTime()
  if (diff <= 0) return null
  const daysLeft = Math.ceil(diff / DAY_MS)
  if (daysLeft > thresholdDays) return null
  return { daysLeft, expiresOn: expiresAt.slice(0, 10) }
}
