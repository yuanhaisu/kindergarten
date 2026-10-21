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
  const res = await wxCodeLogin({
    code: loginRes.code,
    encryptedData: extra.encryptedData,
    iv: extra.iv
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
// 只覆盖本页真实会遇到的失败：网络层（http.js fail）、微信层（uni.login fail）、
// 服务层（body.code）。注意「账号未开通」不在其中——/auth/login/wechat 命中
// resolveWeChatUser 时会自动创建空白 User，不会因此失败；家长账号未开通的失败
// （40031 PARENT_RECORD_NOT_FOUND）发生在首页绑卡，不在这一页。
export function toLoginError(err) {
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
