import { requestWithAuth } from "@/api/http"

export function fetchTeacherCalendarEvents(params = {}) {
  return requestWithAuth({ url: "/teacher/calendar/events", method: "GET", data: params })
}

export function fetchTeacherCalendarEventsByDate(date) {
  return requestWithAuth({ url: `/teacher/calendar/events/${date}`, method: "GET" })
}

export function createTeacherCalendarEvent(data) {
  return requestWithAuth({ url: "/teacher/calendar/events", method: "POST", data })
}

export function updateTeacherCalendarEvent(id, data) {
  return requestWithAuth({ url: `/teacher/calendar/events/${id}`, method: "PUT", data })
}

export function deleteTeacherCalendarEvent(id) {
  return requestWithAuth({ url: `/teacher/calendar/events/${id}`, method: "DELETE" })
}

export function fetchTeacherCalendarDots(params = {}) {
  return requestWithAuth({ url: "/teacher/calendar/dots", method: "GET", data: params })
}
