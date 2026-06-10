import { requestWithAuth } from "@/api/http"

export function fetchTeacherTasks(params = {}) {
  return requestWithAuth({ url: "/teacher/tasks", method: "GET", data: params })
}

export function createTeacherTask(data) {
  return requestWithAuth({ url: "/teacher/tasks", method: "POST", data })
}

export function updateTeacherTask(id, data) {
  return requestWithAuth({ url: `/teacher/tasks/${id}`, method: "PUT", data })
}

export function deleteTeacherTask(id) {
  return requestWithAuth({ url: `/teacher/tasks/${id}`, method: "DELETE" })
}

export function fetchTeacherTaskStats(id) {
  return requestWithAuth({ url: `/teacher/tasks/${id}/stats`, method: "GET" })
}

export function fetchTeacherTaskCheckins(id) {
  return requestWithAuth({ url: `/teacher/tasks/${id}/checkins`, method: "GET" })
}
