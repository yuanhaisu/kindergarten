import { requestWithAuth } from "@/api/http"

export function fetchTeacherTasks(params = {}) {
  return requestWithAuth({ url: "/teacher/tasks", method: "GET", data: params })
}

export function fetchTeacherTask(id) {
  return requestWithAuth({ url: `/teacher/tasks/${id}`, method: "GET" })
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

export function fetchTeacherTaskCheckins(id, params = {}) {
  return requestWithAuth({ url: `/teacher/tasks/${id}/checkins`, method: "GET", data: params })
}

export function respondTeacherTask(taskId, checkinId, data) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/checkins/${checkinId}/response`,
    method: "POST",
    data
  })
}

export function batchRespondTeacherTask(taskId, data) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/batch-response`,
    method: "POST",
    data
  })
}

// 家长端任务详情的小程序码（PNG base64）。码由后端用家长端凭据生成，
// 客户端只负责落盘到相册；返回 { task_id, page, scene, env_version, image_base64 }。
export function fetchTaskShareCard(taskId) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/share-card`,
    method: "GET"
  })
}
