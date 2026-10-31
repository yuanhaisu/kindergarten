import { requestWithAuth, uploadFile } from "@/api/http"

export function fetchTeacherGrowthLogs(params = {}) {
  return requestWithAuth({ url: "/teacher/growth/logs", method: "GET", data: params })
}

export function fetchTeacherGrowthLog(id) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}`, method: "GET" })
}

// options.silent：创建页用「失败红条 + 重试」承载错误，不需要全局 toast 重复提示。
export function createTeacherGrowthLog(data, options = {}) {
  return requestWithAuth({ url: "/teacher/growth/logs", method: "POST", data, silent: options.silent })
}

export function updateTeacherGrowthLog(id, data, options = {}) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}`, method: "PUT", data, silent: options.silent })
}

export function fetchTeacherGrowthComments(id, params = { page: 1, page_size: 20 }) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}/comments`, method: "GET", data: params })
}

// parent_id 只接受该记录下的顶层评论（回复只有一层，B10 契约）；不传即一级评论。
export function createTeacherGrowthComment(id, content, parentId = null) {
  const data = { content }
  if (parentId) data.parent_id = parentId
  return requestWithAuth({ url: `/teacher/growth/logs/${id}/comments`, method: "POST", data })
}

// 评论赞/取消赞（幂等）：返回 { liked, likes_count }，客户端直接以服务端最新值渲染。
export function likeTeacherGrowthComment(commentId) {
  return requestWithAuth({ url: `/teacher/growth/comments/${commentId}/like`, method: "POST", data: {} })
}

export function unlikeTeacherGrowthComment(commentId) {
  return requestWithAuth({ url: `/teacher/growth/comments/${commentId}/like`, method: "DELETE" })
}

// 照片识别状态与建议（详情页轮询用）：
// { pending, media: [{ id, recognition_status, face_count, is_group_photo, suggestions }] }
export function fetchGrowthTaggingStatus(id) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}/tagging-status`, method: "GET" })
}

// 当日到场学生（D11 一键全班的名单来源）：{ list: [studentId], total }。
// 口径由服务端定（在读 ∧ 未被明确排除；无考勤数据保守视为全员），前端不重复判定。
export function fetchPresentStudents(date) {
  return requestWithAuth({ url: "/teacher/class/present-students", method: "GET", data: { date } })
}

// 大合影「一键全班」（D11）：把记录当日的到场学生落为该照片的确认点名（并集，不复活已拒绝项）。
export function applyClassRoster(mediaId) {
  return requestWithAuth({ url: `/teacher/growth/media/${mediaId}/class-roster`, method: "POST", data: {} })
}

// 识别建议确认/拒绝：action = "confirm" | "reject"（仅建议态可流转，重复同一动作幂等）
export function reviewMediaSuggestion(mediaId, studentId, action) {
  return requestWithAuth({
    url: `/teacher/growth/media/${mediaId}/students/${studentId}`,
    method: "PUT",
    data: { action }
  })
}

// 一键确认该记录全部高置信建议（D10 省力路径）：低置信与已处理行不动。返回 {confirmed: N}
export function confirmHighSuggestions(logId) {
  return requestWithAuth({
    url: `/teacher/growth/logs/${logId}/suggestions/confirm-high`,
    method: "POST",
    data: {}
  })
}

// 上传成长记录素材，走统一上传通道（素材管理基线 §6：biz_type=growth）。
// 返回体 data：{ id, file_url, thumb_url, file_type, file_name, file_size }；
// 其中 id 即创建/编辑记录时 media[].file_id 所用的素材标识。
// onProgressUpdate 可选：上传进度百分比（0–100），创建页用于展示真实进度。
export function uploadGrowthMedia(filePath, onProgressUpdate = null) {
  return uploadFile({
    url: "/common/upload",
    filePath,
    formData: { biz_type: "growth" },
    onProgressUpdate
  })
}
