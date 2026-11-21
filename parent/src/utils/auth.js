import { clearCurrentStudentId } from "./currentStudent.js"
import { clearEntitlementCache } from "./entitlement-cache.mjs"

const TOKEN_KEY = "parent_token"
const REFRESH_TOKEN_KEY = "parent_refresh_token"
const USER_KEY = "parent_user"

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

// 当前会话的园所 id：登录/切换返回的 roles 已裁剪到当前园（#169），
// 取首个带 kindergarten_id 的角色即可。0 表示尚无园所（待绑定）。
export function getCurrentKindergartenID() {
  const roles = Array.isArray(getUser()?.roles) ? getUser().roles : []
  const role = roles.find((item) => item && Number(item.kindergarten_id) > 0)
  return role ? Number(role.kindergarten_id) : 0
}

export function clearAuth() {
  uni.removeStorageSync(TOKEN_KEY)
  uni.removeStorageSync(REFRESH_TOKEN_KEY)
  uni.removeStorageSync(USER_KEY)
  // 一并清掉当前孩子：换账号登录后，上一个账号的孩子 id 属于别人，
  // 留着会让 ResolveContext 静默回退，排查起来很费劲。
  clearCurrentStudentId()
  // 授权缓存属于上一个会话：共享设备换账号后不能继续按上一园所展示入口与到期日。
  clearEntitlementCache(uni)
}

// 切换园所（#169）后清「跟随园所」的本地缓存，但保留刚换发的 token 与用户资料。
// 与 clearAuth 的区别是登录态仍然有效；当前孩子 id 属于上一个园，
// 授权缓存也按园所解析，两者都必须作废。
export function clearScopedCaches() {
  clearCurrentStudentId()
  clearEntitlementCache(uni)
}
