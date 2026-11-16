const DAY_MS = 24 * 60 * 60 * 1000

/** ADR-0018 #1：授权到期前 30 天进入转付费提示窗口。 */
export const EXPIRY_WARNING_DAYS = 30

export interface ExpiryNotice {
  /** 到期日，取服务端时间戳的日期部分 */
  expiresOn: string
}

/**
 * 「到期前 30 天转付费提示」的判定。
 *
 * 只返回「仍在期、且已进入提示窗口」的授权；已过期（含宽限期）返回 null，
 * 由页面上既有的到期/降级提示承担，避免同一条到期事实叠加两条提示。
 */
export function expiringSoon(
  expiresAt?: string | null,
  now: Date = new Date(),
): ExpiryNotice | null {
  if (!expiresAt) return null
  const expires = new Date(expiresAt)
  if (Number.isNaN(expires.getTime())) return null
  const diff = expires.getTime() - now.getTime()
  if (diff <= 0 || diff > EXPIRY_WARNING_DAYS * DAY_MS) return null
  return { expiresOn: expiresAt.slice(0, 10) }
}
