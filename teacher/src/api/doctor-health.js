import { requestWithAuth, uploadFile } from "@/api/http"

// 队列响应的 data 为 { list, policy }；
// policy = { temperature_required, fever_threshold }，来自园所设置。
// options 透传 silent 等请求选项：页面自己渲染错误态，避免全局 toast 重复。
export function fetchHealthQueue(params = {}, options = {}) {
  return requestWithAuth({ ...options, url: "/doctor/health/queue", method: "GET", data: params })
}

export function fetchCheckDetail(id, options = {}) {
  return requestWithAuth({ ...options, url: `/doctor/health/check/${id}`, method: "GET" })
}

// 提交晨检结论。响应 data 额外含 notified_count / notify_failed_count。
export function submitCheck(id, body, options = {}) {
  return requestWithAuth({ ...options, url: `/doctor/health/check/${id}`, method: "PUT", data: body })
}

// 上传晨检照片，走统一上传通道（素材管理基线：biz_type=health）。
// 返回 data.id 即关联接口所需的 file_id。
export function uploadHealthPhoto(filePath, options = {}) {
  return uploadFile({
    url: "/common/upload",
    filePath,
    formData: { biz_type: "health" },
    silent: Boolean(options.silent)
  })
}

// 关联照片：body = { file_id, category, ref_desc }
export function addCheckMedia(id, body, options = {}) {
  return requestWithAuth({ ...options, url: `/doctor/health/check/${id}/media`, method: "POST", data: body })
}

export function fetchTodayStats(date, options = {}) {
  return requestWithAuth({ ...options, url: "/doctor/health/today/stats", method: "GET", data: { date } })
}
