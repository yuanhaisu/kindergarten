import config from "@/config/env"
import { clearAuth, getToken } from "@/utils/auth"
import { rememberPendingRoute } from "@/utils/pendingRoute"

const SUCCESS_CODE = 0

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

// silent: true 时不弹全局 toast，由调用方自己渲染错误（登录页用内联错误态）。
// 默认 false，现有调用方行为不变。
function request({ url, method = "GET", data, header = {}, auth = false, silent = false }) {
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
          clearAuth()
          rememberPendingRoute(currentRoutePath())
          uni.reLaunch({ url: "/pages/auth/login" })
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
