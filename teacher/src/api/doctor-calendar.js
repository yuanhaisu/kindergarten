import { requestWithAuth } from "@/api/http"

// 医生端园级事件（保健医发布；契约见 backend/api/api-teacher/08-calendar.md 医生端段落）

export function fetchDoctorCalendarEvents(params = {}) {
  return requestWithAuth({ url: "/doctor/calendar/events", method: "GET", data: params })
}

export function createDoctorCalendarEvent(data) {
  return requestWithAuth({ url: "/doctor/calendar/events", method: "POST", data })
}

export function updateDoctorCalendarEvent(id, data) {
  return requestWithAuth({ url: `/doctor/calendar/events/${id}`, method: "PUT", data })
}

export function deleteDoctorCalendarEvent(id) {
  return requestWithAuth({ url: `/doctor/calendar/events/${id}`, method: "DELETE" })
}
