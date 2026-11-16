import { fetchEntitlements } from "@/api/entitlements"

const CACHE_KEY = "entitlement_cache"
const CACHE_AT_KEY = "entitlement_cache_at"
// 缓存不是永久真相：园所续费/延长后必须在有限时间内反映到 30 天到期提示。
const CACHE_TTL_MS = 10 * 60 * 1000
const DAY_MS = 24 * 60 * 60 * 1000

// ADR-0018 #1：存量园所在到期前 30 天开始提示转付费。
export const EXPIRY_WARNING_DAYS = 30

const EXPIRED_HINT = "套餐已到期，部分功能仅可查看，请联系园所管理员续费"
const RENEW_HINT = "请联系园所管理员续费"

let inflight = null

function readCache() {
  try {
    return uni.getStorageSync(CACHE_KEY) || null
  } catch {
    return null
  }
}

function readCacheAt() {
  try {
    return Number(uni.getStorageSync(CACHE_AT_KEY)) || 0
  } catch {
    return 0
  }
}

// 无时间戳的旧缓存（升级前写入）视为已过期，强制拉一次新数据。
function isCacheFresh() {
  const cachedAt = readCacheAt()
  return !!readCache() && cachedAt > 0 && Date.now() - cachedAt < CACHE_TTL_MS
}

function writeCache(data) {
  try {
    uni.setStorageSync(CACHE_KEY, data)
    uni.setStorageSync(CACHE_AT_KEY, Date.now())
  } catch {
    // 存储失败不影响本次使用
  }
}

/**
 * 当前园所的功能授权。未加载成功时返回 null —— 调用方应「不隐藏」任何入口，
 * 由服务端继续做强校验；隐藏入口只是体验优化，不能替代服务端。
 */
export function getEntitlements() {
  return readCache()
}

export function hasModule(code) {
  if (!code) return true
  const data = readCache()
  if (!data || !Array.isArray(data.modules)) return true
  return data.modules.includes(code)
}

export function entitlementExpiresAt() {
  const data = readCache()
  return data?.expires_at || null
}

/**
 * 到期提示文案。天数向上取整：剩余不足一天计为 0 天，不算过期；
 * 过期满一天才提示「已到期」——保留既有语义，避免最后一天的园所被误判。
 */
export function expiryHintText(expiresAt, now = Date.now()) {
  if (!expiresAt) return ""
  const at = new Date(expiresAt)
  if (Number.isNaN(at.getTime())) return ""
  const days = Math.ceil((at.getTime() - now) / DAY_MS)
  if (days < 0) return EXPIRED_HINT
  if (days <= EXPIRY_WARNING_DAYS) {
    return `套餐将于 ${String(expiresAt).slice(0, 10)} 到期，${RENEW_HINT}`
  }
  return ""
}

export async function loadEntitlements(force = false) {
  if (!force && isCacheFresh()) return readCache()
  if (inflight) return inflight
  inflight = fetchEntitlements()
    .then((res) => {
      const data = res?.data || { modules: [], expires_at: null }
      writeCache(data)
      return data
    })
    // 网络失败时退回旧缓存：断网仍能按上次授权展示，不把入口全关掉。
    .catch(() => readCache())
    .finally(() => {
      inflight = null
    })
  return inflight
}

export function clearEntitlements() {
  try {
    uni.removeStorageSync(CACHE_KEY)
    uni.removeStorageSync(CACHE_AT_KEY)
  } catch {
    // ignore
  }
}
