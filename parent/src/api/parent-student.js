import { requestWithAuth } from "@/api/http"

// 孩子信息页专用的三个接口（profile / students / unbind）走静默：
// 页面自带错误态与 toast，避免 http 层再叠一次全局提示。
export function fetchStudentProfile(studentId) {
  return requestWithAuth({
    url: "/parent/student/profile",
    method: "GET",
    data: { student_id: studentId },
    silent: true
  })
}

export function updateStudentProfile(data) {
  return requestWithAuth({
    url: "/parent/student/profile",
    method: "PUT",
    data
  })
}

export function fetchStudents() {
  return requestWithAuth({
    url: "/parent/students",
    method: "GET",
    silent: true
  })
}

export function bindStudent(inviteCode) {
  return requestWithAuth({
    url: "/parent/students/bind",
    method: "POST",
    data: { invite_code: inviteCode }
  })
}

export function unbindStudent(studentId) {
  return requestWithAuth({
    url: `/parent/students/${studentId}`,
    method: "DELETE",
    silent: true
  })
}
