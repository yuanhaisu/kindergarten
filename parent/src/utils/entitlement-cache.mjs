// 授权缓存的 key、清理与「清理代数」独立于 uni 与 api 层。
// entitlements.js 依赖 @/api/entitlements → @/api/http → utils/auth，
// 若在 utils/auth.js 里反向 import entitlements.js 会成环；登录/登出流程只依赖本模块。
export const ENTITLEMENT_CACHE_KEY = "entitlement_cache"
export const ENTITLEMENT_CACHE_AT_KEY = "entitlement_cache_at"

// 每次清理递增。在途请求带着发起时的代数返回；代数已变说明会话被清理过，
// 其结果不得写回缓存，否则旧租户数据会在换账号后继续续命一整个 TTL。
let epoch = 0

export function entitlementCacheEpoch() {
  return epoch
}

// storage 由调用方注入（小程序里是 uni），便于 node --test 直接验证清理行为。
export function clearEntitlementCache(storage) {
  // 先递增：即使 storage 不可用，在途请求也必须失去写回资格。
  epoch += 1
  if (!storage) return
  try {
    storage.removeStorageSync(ENTITLEMENT_CACHE_KEY)
    storage.removeStorageSync(ENTITLEMENT_CACHE_AT_KEY)
  } catch (err) {
    // 存储不可用不能卡住登出/登录：缓存本身只是展示优化，但要留痕便于排查。
    console.warn("[entitlements] clear cache failed:", err?.message || err)
  }
}
