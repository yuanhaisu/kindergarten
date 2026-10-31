import config from "@/config/env"
import { clearAuth, getRefreshToken, getToken, setRefreshToken, setToken } from "@/utils/auth"
import { rememberPendingRoute } from "@/utils/pendingRoute"

const SUCCESS_CODE = 0
const LOGIN_URL = "/pages/auth/login"
const AUTH_EXPIRED_MESSAGE = "登录状态已失效，请重新登录"

// 后端 refresh token 一次性消费（Redis ConsumeRefresh 是 get + del）：并发 401 各发一次刷新时，
// 后到的那次必然失败，还会把刚续上的会话清掉。所以同一时刻只允许一次刷新，
// 其余请求共享这个 Promise，成功后各自重放一次。
let refreshPromise = null
// 同一批失败只跳一次登录页：多个 reLaunch 会互相打断。
let redirecting = false

// 401 发生在哪个页面，就把那个页面（含 query）记下来，登录后回跳。
// 深链尤其重要：从消息点进任务详情时 token 过期，不能把用户丢回首页。
function currentRoutePath() {
  try {
    const pages = getCurrentPages()
    const page = pages[pages.length - 1]
    if (!page || !page.route) return ""
    const opts = page.options || {}
    const qs = Object.keys(opts).map((k) => `${k}=${opts[k]}`).join("&")
    return `/${page.route}${qs ? `?${qs}` : ""}`
  } catch (e) {
    return ""
  }
}

function normalizeResponse(body) {
  if (typeof body?.code === "number") return body

  if (typeof body?.success === "boolean") {
    return body.success
      ? { code: SUCCESS_CODE, message: "success", data: body.data, request_id: body.request_id }
      : {
          code: body.error?.code || 50000,
          message: body.error?.message || "请求失败",
          data: null,
          request_id: body.request_id
        }
  }

  return { code: SUCCESS_CODE, message: "success", data: body }
}

function redirectToLogin() {
  if (redirecting) return
  redirecting = true
  uni.reLaunch({
    url: LOGIN_URL,
    complete() {
      redirecting = false
    }
  })
}

// 401 包络里的后端文案比本地兜底更准；网络层错误没有 code，用兜底文案。
function authError(err) {
  const hasServerMessage = typeof err?.code === "number" && err.code !== SUCCESS_CODE && !!err.message
  return {
    code: typeof err?.code === "number" ? err.code : 40100,
    message: hasServerMessage ? err.message : AUTH_EXPIRED_MESSAGE,
    data: null
  }
}

// 续期失败：清会话、记录回跳路由、跳登录（并发只跳一次），返回归一化错误供调用方 reject。
// tokenUsed 是触发 401 的那次请求实际带的 token。迟到的 401 可能在别处刷新/重新登录
// 之后才回来，此时 storage 里已是更新的会话，不能把它清掉，也不该再跳登录。
function failToLogin(err, tokenUsed) {
  const current = getToken()
  if (current && current !== tokenUsed) return authError(err)
  clearAuth()
  // 登录页自己不能成为回跳目标，否则登录成功后又被送回登录页。
  const path = currentRoutePath()
  if (path !== LOGIN_URL) rememberPendingRoute(path)
  redirectToLogin()
  return authError(err)
}

// 不走 request()：refresh 自身 401 会再次进入续期，形成递归。
function refreshSession() {
  if (refreshPromise) return refreshPromise

  const token = getRefreshToken()
  // 空 refresh token 不发起请求：结果与直接跳登录相同，徒增一次往返。
  if (!token) return Promise.reject({ code: 40100, message: AUTH_EXPIRED_MESSAGE, data: null })

  refreshPromise = new Promise((resolve, reject) => {
    uni.request({
      url: `${config.baseURL}/auth/refresh`,
      method: "POST",
      data: { refresh_token: token },
      header: {
        "Content-Type": "application/json",
        platform: "mp-weixin",
        terminal_from: "parent"
      },
      success(res) {
        const body = normalizeResponse(res.data)
        // 两个 token 缺一不可：只落 access 会留下空 refresh，下次过期就再也续不上。
        if (
          res.statusCode >= 200 &&
          res.statusCode < 300 &&
          body.code === SUCCESS_CODE &&
          body.data?.token &&
          body.data?.refresh_token
        ) {
          resolve(body.data)
          return
        }
        reject(body)
      },
      fail(err) {
        reject({ ...err, message: err?.message || "网络异常" })
      }
    })
  }).finally(() => {
    refreshPromise = null
  })

  return refreshPromise
}

// 401 后调用：已有请求续过期就直接用新 token 重放，否则发一次刷新并落盘轮换后的 token。
function recoverSession(tokenUsed) {
  const current = getToken()
  if (current && current !== tokenUsed) return Promise.resolve()

  return refreshSession()
    .then((data) => {
      setToken(data.token)
      setRefreshToken(data.refresh_token)
    })
    .catch((err) => {
      throw failToLogin(err, tokenUsed)
    })
}

// silent: true 时不弹全局 toast，由调用方自己渲染错误（登录页用内联错误态）。
// 默认 false，现有调用方行为不变。
// replay: 已完成一次刷新的重放；重放再 401 不再续期，直接按失败处理，防止循环。
function request({ url, method = "GET", data, header = {}, auth = false, silent = false }, replay = false) {
  return new Promise((resolve, reject) => {
    const token = auth ? getToken() : ""

    uni.request({
      url: `${config.baseURL}${url}`,
      method,
      data,
      header: {
        "Content-Type": "application/json",
        platform: "mp-weixin",
        terminal_from: "parent",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...header
      },
      success(res) {
        const body = normalizeResponse(res.data)

        if (res.statusCode === 401) {
          // 只有带 token 的请求才续期；rawRequest（登录/刷新）保持原行为：立即回登录页。
          if (auth) {
            if (replay) {
              reject(failToLogin(body, token))
              return
            }
            recoverSession(token).then(
              () => request({ url, method, data, header, auth, silent }, true).then(resolve, reject),
              reject
            )
            return
          }
          clearAuth()
          rememberPendingRoute(currentRoutePath())
          redirectToLogin()
          reject(body)
          return
        }

        if (res.statusCode >= 200 && res.statusCode < 300 && body.code === SUCCESS_CODE) {
          resolve(body)
          return
        }

        if (!silent) {
          uni.showToast({ title: body.message || "请求失败", icon: "none" })
        }
        reject(body)
      },
      fail(err) {
        if (!silent) {
          uni.showToast({ title: "网络异常", icon: "none" })
        }
        // 归一化：调用方读 err.message，而微信 fail 对象只带 errMsg，
        // 不补这一层则页面的 err?.message 恒为 undefined，永远走到兜底文案。
        reject({ ...err, message: err?.message || "网络异常" })
      }
    })
  })
}

export function rawRequest(options) {
  return request({ ...options, auth: false })
}

export function requestWithAuth(options) {
  return request({ ...options, auth: true })
}

// 文件上传（multipart）：与 request() 共用 token 注入、单飞续期与错误归一化。
// url 可传完整地址（config.uploadURL）或相对 /common/upload；成功判定同为 2xx + code===0。
// silent 默认 true：上传点的 loading 与文案由调用方自己渲染。
export function uploadFile(options) {
  return sendUpload(options, false)
}

// replay 语义与 request() 相同：续期后重放一次，重放再 401 直接失败。
function sendUpload({
  url,
  filePath,
  name = "file",
  formData = {},
  header = {},
  silent = true,
  onProgressUpdate = null
}, replay) {
  return new Promise((resolve, reject) => {
    const token = getToken()
    const target = /^https?:\/\//i.test(url || "") ? url : `${config.baseURL}${url}`
    const task = uni.uploadFile({
      url: target,
      filePath,
      name,
      formData,
      header: {
        platform: "mp-weixin",
        terminal_from: "parent",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...header
      },
      success(res) {
        let body = res.data
        if (typeof body === "string") {
          try { body = JSON.parse(body) } catch { body = null }
        }
        body = normalizeResponse(body)

        if (res.statusCode === 401) {
          if (replay) {
            reject(failToLogin(body, token))
            return
          }
          recoverSession(token).then(
            () =>
              sendUpload({ url, filePath, name, formData, header, silent, onProgressUpdate }, true).then(
                resolve,
                reject
              ),
            reject
          )
          return
        }
        if (res.statusCode >= 200 && res.statusCode < 300 && body.code === SUCCESS_CODE) {
          resolve(body)
          return
        }
        if (!silent) {
          uni.showToast({ title: body.message || "上传失败", icon: "none" })
        }
        reject(body)
      },
      fail(err) {
        if (!silent) {
          uni.showToast({ title: "网络异常", icon: "none" })
        }
        reject({ ...err, message: err?.message || "上传失败" })
      }
    })
    // 老基础库 / 非上传环境可能不返回 UploadTask，此时静默跳过进度，不影响上传本身
    if (typeof onProgressUpdate === "function" && task && typeof task.onProgressUpdate === "function") {
      task.onProgressUpdate(({ progress }) => onProgressUpdate(progress))
    }
  })
}
