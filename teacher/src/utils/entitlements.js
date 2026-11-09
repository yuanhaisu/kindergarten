import { fetchEntitlements } from "@/api/entitlements"

const CACHE_KEY = "entitlement_cache"
let inflight = null

function readCache() {
  try {
    return uni.getStorageSync(CACHE_KEY) || null
  } catch {
    return null
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

export async function loadEntitlements(force = false) {
  if (!force && readCache()) return readCache()
  if (inflight) return inflight
  inflight = fetchEntitlements()
    .then((res) => {
      const data = res?.data || { modules: [], expires_at: null }
      try {
        uni.setStorageSync(CACHE_KEY, data)
      } catch {
        // 存储失败不影响本次使用
      }
      return data
    })
    .catch(() => readCache())
    .finally(() => {
      inflight = null
    })
  return inflight
}

export function clearEntitlements() {
  try {
    uni.removeStorageSync(CACHE_KEY)
  } catch {
    // ignore
  }
}
