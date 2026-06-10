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
