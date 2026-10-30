import { requestWithAuth } from "@/api/http"

// silent：页面自带整页错误态与重试，避免 http 层再叠一次全局 toast。
export function fetchClassInfo(studentId) {
  return requestWithAuth({
    url: "/parent/class/info",
    method: "GET",
    data: { student_id: studentId },
    silent: true
  })
}

export function fetchClassCalendar(params = {}) {
  return requestWithAuth({
    url: "/parent/class/calendar",
    method: "GET",
    data: params
  })
}
