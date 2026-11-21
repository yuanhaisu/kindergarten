import { clearEntitlementCache } from "./entitlement-cache.mjs"

const TOKEN_KEY = "teacher_token"
const REFRESH_TOKEN_KEY = "teacher_refresh_token"
const USER_KEY = "teacher_user"

export function setToken(token) {
  uni.setStorageSync(TOKEN_KEY, token || "")
}

export function getToken() {
  return uni.getStorageSync(TOKEN_KEY) || ""
}

export function setRefreshToken(token) {
  uni.setStorageSync(REFRESH_TOKEN_KEY, token || "")
}

export function getRefreshToken() {
  return uni.getStorageSync(REFRESH_TOKEN_KEY) || ""
}

export function setUser(user) {
  uni.setStorageSync(USER_KEY, user || null)
}

export function getUser() {
  return uni.getStorageSync(USER_KEY) || null
}

export function clearAuth() {
  uni.removeStorageSync(TOKEN_KEY)
  uni.removeStorageSync(REFRESH_TOKEN_KEY)
  uni.removeStorageSync(USER_KEY)
  // 授权缓存属于上一个会话：共享设备换账号后不能继续按上一园所展示入口与到期日。
  clearEntitlementCache(uni)
}

// 切换园所（#169）后清「跟随园所」的本地缓存，但保留刚换发的 token 与用户资料。
// 与 clearAuth 的区别是登录态仍然有效；授权缓存按园所解析，必须作废。
export function clearScopedCaches() {
  clearEntitlementCache(uni)
}
