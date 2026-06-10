import { requestWithAuth } from "@/api/http"

export function fetchTeacherLeaves(params = {}) {
  return requestWithAuth({ url: "/teacher/leaves", method: "GET", data: params })
}

export function fetchTeacherLeaveStats() {
  return requestWithAuth({ url: "/teacher/leaves/stats", method: "GET" })
}

export function fetchTeacherLeave(id) {
  return requestWithAuth({ url: `/teacher/leaves/${id}`, method: "GET" })
}

export function approveTeacherLeave(id, remark = "") {
  return requestWithAuth({ url: `/teacher/leaves/${id}/approve`, method: "PUT", data: { remark } })
}

export function rejectTeacherLeave(id, reason) {
  return requestWithAuth({ url: `/teacher/leaves/${id}/reject`, method: "PUT", data: { reason } })
}
