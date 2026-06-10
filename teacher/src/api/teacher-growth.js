import { requestWithAuth } from "@/api/http"

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

export function deleteTeacherGrowthLog(id) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}`, method: "DELETE" })
}

export function fetchTeacherGrowthComments(id) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}/comments`, method: "GET" })
}

export function createTeacherGrowthComment(id, content) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}/comments`, method: "POST", data: { content } })
}

export function fetchTeacherGrowthLikes(id) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}/likes`, method: "GET" })
}

export function fetchTeacherGrowthRelatedStudents(id) {
  return requestWithAuth({ url: `/teacher/growth/logs/${id}/related-students`, method: "GET" })
}

export function createTeacherGrowthMedia(data) {
  return requestWithAuth({ url: "/teacher/growth/media/upload", method: "POST", data })
}
