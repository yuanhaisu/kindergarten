import { requestWithAuth } from "@/api/http"

// options 透传给 http 层（silent 用于页面自绘错误态，避免全局 toast 与内联错误条重复）。
export function fetchTeacherCalendarEvents(params = {}, options = {}) {
  return requestWithAuth({ url: "/teacher/calendar/events", method: "GET", data: params, ...options })
}

export function fetchTeacherCalendarEventsByDate(date, options = {}) {
  return requestWithAuth({ url: `/teacher/calendar/events/${date}`, method: "GET", ...options })
}

export function createTeacherCalendarEvent(data, options = {}) {
  return requestWithAuth({ url: "/teacher/calendar/events", method: "POST", data, ...options })
}

export function updateTeacherCalendarEvent(id, data, options = {}) {
  return requestWithAuth({ url: `/teacher/calendar/events/${id}`, method: "PUT", data, ...options })
}

export function deleteTeacherCalendarEvent(id, options = {}) {
  return requestWithAuth({ url: `/teacher/calendar/events/${id}`, method: "DELETE", ...options })
}

export function fetchTeacherCalendarDots(params = {}, options = {}) {
  return requestWithAuth({ url: "/teacher/calendar/dots", method: "GET", data: params, ...options })
}

export function fetchTeacherCalendarEvent(id, options = {}) {
  return requestWithAuth({ url: `/teacher/calendar/events/detail/${id}`, method: "GET", ...options })
}
