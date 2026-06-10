import { requestWithAuth } from "@/api/http"

export function fetchTeacherClassInfo() {
  return requestWithAuth({ url: "/teacher/class/info", method: "GET" })
}

export function fetchTeacherClassStudents(params = {}) {
  return requestWithAuth({ url: "/teacher/class/students", method: "GET", data: params })
}

export function fetchTeacherClassStudent(id) {
  return requestWithAuth({ url: `/teacher/class/students/${id}`, method: "GET" })
}
