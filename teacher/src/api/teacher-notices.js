import { requestWithAuth } from "@/api/http"

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


export function fetchTeacherNoticeReadStatus(id) {
  return requestWithAuth({ url: `/teacher/notices/${id}/read-status`, method: "GET" })
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
