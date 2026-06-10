import { requestWithAuth } from "@/api/http"

export function fetchClassInfo(studentId) {
  return requestWithAuth({
    url: "/parent/class/info",
    method: "GET",
    data: { student_id: studentId }
  })
}

export function fetchClassCalendar(params = {}) {
  return requestWithAuth({
    url: "/parent/class/calendar",
    method: "GET",
    data: params
  })
}
