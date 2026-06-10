import config from "@/config/env"
import { clearAuth, getToken } from "@/utils/auth"

const SUCCESS_CODE = 0

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

function request({ url, method = "GET", data, header = {}, auth = false }) {
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
          uni.reLaunch({ url: "/pages/auth/phone" })
          reject(body)
          return
        }

        if (res.statusCode >= 200 && res.statusCode < 300 && body.code === SUCCESS_CODE) {
          resolve(body)
          return
        }

        uni.showToast({ title: body.message || "请求失败", icon: "none" })
        reject(body)
      },
      fail(err) {
        uni.showToast({ title: "网络异常", icon: "none" })
        reject(err)
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
