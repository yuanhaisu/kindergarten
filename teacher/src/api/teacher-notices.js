import { requestWithAuth, uploadFile } from "@/api/http"

// 上传通知附件，走统一上传通道（biz_type=notice）。
// 返回体 data：{ id, file_url, file_type, file_name, file_size }，与创建/编辑入参 attachments 同形。
// onProgressUpdate 可选：上传进度百分比（0–100），创建页用于展示真实进度。
export function uploadNoticeAttachment(filePath, onProgressUpdate = null) {
  return uploadFile({
    url: "/common/upload",
    filePath,
    formData: { biz_type: "notice" },
    onProgressUpdate
  })
}

export function fetchTeacherNotices(params = {}) {
  return requestWithAuth({ url: "/teacher/notices", method: "GET", data: params })
}

export function fetchTeacherNotice(id) {
  return requestWithAuth({ url: `/teacher/notices/${id}`, method: "GET" })
}

export function createTeacherNotice(data) {
  return requestWithAuth({ url: "/teacher/notices", method: "POST", data })
}

export function updateTeacherNotice(id, data) {
  return requestWithAuth({ url: `/teacher/notices/${id}`, method: "PUT", data })
}

export function deleteTeacherNotice(id) {
  return requestWithAuth({ url: `/teacher/notices/${id}`, method: "DELETE" })
}

export function revokeTeacherNotice(id) {
  return requestWithAuth({ url: `/teacher/notices/${id}/revoke`, method: "PUT" })
}


export function fetchTeacherNoticeReadStatus(id, params = {}) {
  return requestWithAuth({ url: `/teacher/notices/${id}/read-status`, method: "GET", data: params })
}

export function remindTeacherNotice(id) {
  return requestWithAuth({ url: `/teacher/notices/${id}/remind`, method: "POST" })
}

export function fetchTeacherNoticeTemplates() {
  return requestWithAuth({ url: "/teacher/notices/templates", method: "GET" })
}

export function createTeacherNoticeTemplate(data) {
  return requestWithAuth({ url: "/teacher/notices/templates", method: "POST", data })
}

export function updateTeacherNoticeTemplate(id, data) {
  return requestWithAuth({ url: `/teacher/notices/templates/${id}`, method: "PUT", data })
}

export function deleteTeacherNoticeTemplate(id) {
  return requestWithAuth({ url: `/teacher/notices/templates/${id}`, method: "DELETE" })
}
