// 授权缓存的 key 与清理逻辑独立于 uni 与 api 层。
// entitlements.js 依赖 @/api/entitlements → @/api/http → utils/auth，
// 若在 utils/auth.js 里反向 import entitlements.js 会成环；登录/登出流程只依赖本模块。
export const ENTITLEMENT_CACHE_KEY = "entitlement_cache"
export const ENTITLEMENT_CACHE_AT_KEY = "entitlement_cache_at"

// storage 由调用方注入（小程序里是 uni），便于 node --test 直接验证清理行为。
export function clearEntitlementCache(storage) {
  if (!storage) return
  try {
    storage.removeStorageSync(ENTITLEMENT_CACHE_KEY)
    storage.removeStorageSync(ENTITLEMENT_CACHE_AT_KEY)
  } catch {
    // 存储不可用不能卡住登出/登录：缓存本身只是展示优化。
  }
}
