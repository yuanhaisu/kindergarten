import { requestWithAuth } from "@/api/http"

export function fetchHealthQueue(params = {}) {
  return requestWithAuth({ url: "/doctor/health/queue", method: "GET", data: params })
}

export function fetchCheckDetail(id) {
  return requestWithAuth({ url: `/doctor/health/check/${id}`, method: "GET" })
}

export function submitCheck(id, body) {
  return requestWithAuth({ url: `/doctor/health/check/${id}`, method: "PUT", data: body })
}

export function addCheckMedia(id, body) {
  return requestWithAuth({ url: `/doctor/health/check/${id}/media`, method: "POST", data: body })
}

export function fetchTodayStats(date) {
  return requestWithAuth({ url: "/doctor/health/today/stats", method: "GET", data: { date } })
}
