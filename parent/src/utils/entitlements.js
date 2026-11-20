import { fetchEntitlements } from "@/api/entitlements"
import {
  ENTITLEMENT_CACHE_KEY as CACHE_KEY,
  ENTITLEMENT_CACHE_AT_KEY as CACHE_AT_KEY,
  entitlementCacheEpoch
} from "./entitlement-cache.mjs"
import { EXPIRY_WARNING_DAYS, expiryHintText as baseExpiryHintText } from "./expiry-hint.mjs"

// 缓存不是永久真相：园所续费/延长后必须在有限时间内反映到 30 天到期提示。
const CACHE_TTL_MS = 10 * 60 * 1000

// ADR-0018 #1：存量园所在到期前 30 天开始提示转付费。
export { EXPIRY_WARNING_DAYS }

const RENEW_HINT = "请联系园所管理员续费"

let inflight = null
// inflight 发起时的缓存代数：代数变了说明会话被清理过（登出/换账号），旧响应不可再复用。
let inflightEpoch = 0

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
 * 到期提示文案：先按时间戳判断已到期，再按剩余天数决定是否进入 30 天提示窗口。
 */
export function expiryHintText(expiresAt, now = Date.now()) {
  return baseExpiryHintText(expiresAt, now, RENEW_HINT)
}

export async function loadEntitlements(force = false) {
  if (!force && isCacheFresh()) return readCache()

  const requestEpoch = entitlementCacheEpoch()
  // 只有同代会话的在途请求才能复用；清理过的旧请求属于上一个会话，必须重新发起。
  if (inflight && inflightEpoch === requestEpoch) return inflight

  const request = fetchEntitlements()
    .then((res) => {
      const data = res?.data || { modules: [], expires_at: null }
      // 请求在途期间会话被清理：结果只返回给调用方，不写回缓存，避免旧租户数据续命。
      if (entitlementCacheEpoch() === requestEpoch) writeCache(data)
      return data
    })
    // 网络失败时退回旧缓存：断网仍能按上次授权展示，不把入口全关掉。
    .catch(() => readCache())
    .finally(() => {
      // 只清自己：旧请求晚于新请求结束，不能把新的 inflight 清掉。
      if (inflight === request) inflight = null
    })
  inflight = request
  inflightEpoch = requestEpoch
  return request
}
