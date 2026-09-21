import { clearCurrentStudentId } from "@/utils/currentStudent"

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

export function clearAuth() {
  uni.removeStorageSync(TOKEN_KEY)
  uni.removeStorageSync(REFRESH_TOKEN_KEY)
  uni.removeStorageSync(USER_KEY)
  // 一并清掉当前孩子：换账号登录后，上一个账号的孩子 id 属于别人，
  // 留着会让 ResolveContext 静默回退，排查起来很费劲。
  clearCurrentStudentId()
}
