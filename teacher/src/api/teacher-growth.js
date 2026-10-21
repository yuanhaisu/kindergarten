import { requestWithAuth, uploadFile } from "@/api/http"

export function fetchTeacherGrowthLogs(params = {}) {
  return requestWithAuth({ url: "/teacher/growth/logs", method: "GET", data: params })
}

export function fetchTeacherGrowthLog(id) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}`, method: "GET" })
}

export function createTeacherGrowthLog(data) {
  return requestWithAuth({ url: "/teacher/growth/logs", method: "POST", data })
}

export function updateTeacherGrowthLog(id, data) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}`, method: "PUT", data })
}

export function fetchTeacherGrowthComments(id) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}/comments`, method: "GET" })
}

export function createTeacherGrowthComment(id, content) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}/comments`, method: "POST", data: { content } })
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
export function uploadGrowthMedia(filePath) {
  return uploadFile({
    url: "/common/upload",
    filePath,
    formData: { biz_type: "growth" }
  })
}
