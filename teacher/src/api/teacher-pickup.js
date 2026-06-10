import { requestWithAuth } from "@/api/http"

export function fetchTeacherPickupOverview() {
  return requestWithAuth({ url: "/teacher/pickup/overview", method: "GET" })
}

export function fetchTeacherPickupStudents(params = {}) {
  return requestWithAuth({ url: "/teacher/pickup/students", method: "GET", data: params })
}

export function scanTeacherPickup(data) {
  return requestWithAuth({ url: "/teacher/pickup/scan", method: "POST", data })
}

export function manualTeacherPickup(data) {
  return requestWithAuth({ url: "/teacher/pickup/manual", method: "POST", data })
}

export function fetchTeacherPickupRecords(params = {}) {
  return requestWithAuth({ url: "/teacher/pickup/records", method: "GET", data: params })
}

export function fetchTeacherPickupRecord(id) {
  return requestWithAuth({ url: `/teacher/pickup/records/${id}`, method: "GET" })
}

export function markTeacherPickupAbnormal(id, reason) {
  return requestWithAuth({ url: `/teacher/pickup/records/${id}/abnormal`, method: "PUT", data: { reason } })
}
