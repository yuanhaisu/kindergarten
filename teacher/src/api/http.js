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
        terminal_from: "teacher",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...header
      },
      success(res) {
        const body = normalizeResponse(res.data)

        if (res.statusCode === 401) {
          clearAuth()
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

// 文件上传（multipart）：与 request() 同一套 token 注入、401 处理与错误归一化约定。
// 服务端契约见 POST /common/upload：form 字段 file + biz_type；成功判定同为 2xx + code===0。
// 用法（成长记录）：uploadFile({ url: "/common/upload", filePath, formData: { biz_type: "growth" } })
// onProgressUpdate 可选：把 uni.uploadFile 返回任务的进度回调整体透传出去
// （参数 { progress, totalBytesSent, totalBytesExpectedToSend }）；不传时行为不变。
export function uploadFile({
  url,
  filePath,
  name = "file",
  formData = {},
  header = {},
  silent = false,
  onProgressUpdate = null
}) {
  return new Promise((resolve, reject) => {
    const token = getToken()
    const task = uni.uploadFile({
      url: `${config.baseURL}${url}`,
      filePath,
      name,
      formData,
      header: {
        platform: "mp-weixin",
        terminal_from: "teacher",
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
          clearAuth()
          uni.reLaunch({ url: "/pages/auth/login" })
          reject(body)
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
