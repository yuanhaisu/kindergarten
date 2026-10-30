import { requestWithAuth } from "@/api/http"

// options 透传给 requestWithAuth（如 { silent: true }）：班级页自己渲染整页错误态，
// 不再叠加全局 toast。其余调用方不传时行为不变。
export function fetchTeacherClassInfo(options = {}) {
  return requestWithAuth({ url: "/teacher/class/info", method: "GET", ...options })
}

export function fetchTeacherClassStudents(params = {}, options = {}) {
  return requestWithAuth({ url: "/teacher/class/students", method: "GET", data: params, ...options })
}

export function fetchTeacherClassStudent(id) {
  return requestWithAuth({ url: `/teacher/class/students/${id}`, method: "GET" })
}
