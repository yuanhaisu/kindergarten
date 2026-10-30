import { requestWithAuth } from "@/api/http"

// 医生端园级事件（保健医发布；契约见 backend/api/api-teacher/08-calendar.md 医生端段落）
// options 透传 silent 等请求选项：页面自己渲染错误态，避免全局 toast 重复。

export function fetchDoctorCalendarEvents(params = {}, options = {}) {
  return requestWithAuth({ ...options, url: "/doctor/calendar/events", method: "GET", data: params })
}

export function createDoctorCalendarEvent(data, options = {}) {
  return requestWithAuth({ ...options, url: "/doctor/calendar/events", method: "POST", data })
}

export function updateDoctorCalendarEvent(id, data, options = {}) {
  return requestWithAuth({ ...options, url: `/doctor/calendar/events/${id}`, method: "PUT", data })
}

export function deleteDoctorCalendarEvent(id, options = {}) {
  return requestWithAuth({ ...options, url: `/doctor/calendar/events/${id}`, method: "DELETE" })
}
