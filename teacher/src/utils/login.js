import { bindInvite, wxCodeLogin } from "@/api/auth"
import { getConsentStatus } from "@/api/consent"
import { getToken, setRefreshToken, setToken, setUser } from "@/utils/auth"
import { clearEntitlementCache } from "./entitlement-cache.mjs"

// #168：邀请码由启动参数写入，绑定成功后立即消费，避免下次启动拿旧码重试。
const INVITE_CODE_KEY = "invite_code"

let loginPromise = null

function uniLogin() {
  return new Promise((resolve, reject) => {
    uni.login({
      provider: "weixin",
      success: resolve,
      fail: reject
    })
  })
}

// 老师端 token 只携带当前客户端角色族：有 teacher 角色才算真正绑定。
export function hasTeacherRole(user) {
  const roles = Array.isArray(user?.roles) ? user.roles : []
  return roles.some((role) => role?.role === "teacher")
}

function applySession(data) {
  // 登录成功即换会话：未走登出直接登录另一账号时，也要丢弃上一会话的授权缓存。
  clearEntitlementCache(uni)
  setToken(data.token)
  setRefreshToken(data.refresh_token)
  setUser(data.user)
}

export async function silentLogin(extra = {}) {
  const loginRes = await uniLogin()
  const res = await wxCodeLogin({
    code: loginRes.code,
    encryptedData: extra.encryptedData,
    iv: extra.iv
  })

  let data = res.data || {}
  // 先落待绑定会话：邀请码绑定接口要求持有这枚 token。
  applySession(data)

  // 登录只负责识别身份；加入园所是独立一步（#168）。
  if (hasTeacherRole(data.user)) {
    // 已绑定账号不再需要邀请码：避免换微信登录时误用设备上残留的旧码。
    uni.removeStorageSync(INVITE_CODE_KEY)
  } else {
    const inviteCode = uni.getStorageSync(INVITE_CODE_KEY) || ""
    if (inviteCode) {
      const bound = await bindInvite(inviteCode)
      data = bound.data || {}
      uni.removeStorageSync(INVITE_CODE_KEY)
      applySession(data)
    }
  }
  return data
}

export async function bindPhoneLogin(extra = {}) {
  return silentLogin(extra)
}

export async function ensureLogin() {
  if (getToken()) return true

  if (!loginPromise) {
    loginPromise = silentLogin().finally(() => {
      loginPromise = null
    })
  }

  return loginPromise
}

// 已登录用户启动时检查隐私政策同意状态；缺同意时进入阻断页。
// 网络/服务异常不在这里阻断，交给后续业务请求的 403 兜底。
export async function ensureConsent() {
  if (!getToken()) return true
  try {
    const res = await getConsentStatus()
    const data = res.data || {}
    if (data.consented) return true
    uni.reLaunch({ url: "/pages/auth/consent" })
    return false
  } catch (err) {
    return true
  }
}

// 登录失败原因分档，供登录页渲染内联错误态（取代转瞬即逝的 toast）。
//
// 覆盖本页真实会遇到的失败：网络层（http.js fail）、微信层（uni.login fail）、
// 服务层（body.code），以及邀请绑定（#168）的业务分档：邀请码无效/已用、
// 客户端与园所通道不匹配、教师账号已绑其他微信、以及「没有邀请码」的待绑定态。
export function toLoginError(err) {
  if (err?.code === 40020) {
    return { title: "邀请码无效或已过期", detail: "请向园方索取新的邀请链接" }
  }
  if (err?.code === 40021) {
    return { title: "邀请码已被使用", detail: "请联系园方管理员重新获取邀请" }
  }
  if (err?.code === 40900) {
    return { title: "该教师账号已绑定其他微信", detail: "请联系园方管理员处理" }
  }
  if (err?.code === 40302) {
    return { title: "账号尚未绑定", detail: "请通过园方发放的邀请链接完成绑定" }
  }
  if (err?.code === 40002) {
    return { title: "微信登录失败", detail: "请稍后重试" }
  }
  // uni.login 的 fail 对象只带 errMsg，没有 message
  if (err?.errMsg && !err?.message) {
    return { title: "微信登录失败", detail: "请稍后重试" }
  }
  if (err?.message === "网络异常") {
    return { title: "网络异常", detail: "请检查网络后重试" }
  }
  return { title: err?.message || "登录失败", detail: "请稍后重试" }
}

// 登录成功但仍是待绑定账号：没有邀请码时不是错误，而是需要园方开通。
export const UNBOUND_LOGIN_ERROR = {
  title: "尚未绑定教师账号",
  detail: "请通过园方发放的邀请链接进入，或联系园方获取邀请码"
}
