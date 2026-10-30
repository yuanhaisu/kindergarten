import { requestWithAuth } from "@/api/http"

export function fetchTeacherLeaves(params = {}, options = {}) {
  return requestWithAuth({ url: "/teacher/leaves", method: "GET", data: params, ...options })
}

export function fetchTeacherLeaveStats(options = {}) {
  return requestWithAuth({ url: "/teacher/leaves/stats", method: "GET", ...options })
}

export function fetchTeacherLeave(id, options = {}) {
  return requestWithAuth({ url: `/teacher/leaves/${id}`, method: "GET", ...options })
}

// 契约字段是 remark；后端为兼容旧客户端仍接受 reason，但新代码只发 remark。
export function approveTeacherLeave(id, remark = "", options = {}) {
  return requestWithAuth({ url: `/teacher/leaves/${id}/approve`, method: "PUT", data: { remark }, ...options })
}

export function rejectTeacherLeave(id, remark, options = {}) {
  return requestWithAuth({ url: `/teacher/leaves/${id}/reject`, method: "PUT", data: { remark }, ...options })
}
