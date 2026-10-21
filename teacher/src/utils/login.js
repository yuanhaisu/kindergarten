import { wxCodeLogin } from "@/api/auth"
import { getToken, setRefreshToken, setToken, setUser } from "@/utils/auth"

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

export async function silentLogin(extra = {}) {
  const loginRes = await uniLogin()
  const inviteCode = uni.getStorageSync('invite_code') || ''
  const res = await wxCodeLogin({
    code: loginRes.code,
    encryptedData: extra.encryptedData,
    iv: extra.iv,
    inviteCode
  })

  const data = res.data || {}
  setToken(data.token)
  setRefreshToken(data.refresh_token)
  setUser(data.user)
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

  await loginPromise
  return true
}

// 登录失败原因分档，供登录页渲染内联错误态（取代转瞬即逝的 toast）。
//
// 覆盖本页真实会遇到的失败：网络层（http.js fail）、微信层（uni.login fail）、
// 服务层（body.code），外加邀请码分档——带 invite_code 登录时会走
// weChatLoginWithInvite，40020/40021 是可达的死胡同。
export function toLoginError(err) {
  if (err?.code === 40020) {
    return { title: "邀请码无效或已过期", detail: "请向园方索取新的邀请码" }
  }
  if (err?.code === 40021) {
    return { title: "邀请码关联的教师账号不存在", detail: "请联系园方确认你的账号" }
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
