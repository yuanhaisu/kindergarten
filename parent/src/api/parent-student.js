import { requestWithAuth } from "@/api/http"

export function fetchStudentProfile(studentId) {
  return requestWithAuth({
    url: "/parent/student/profile",
    method: "GET",
    data: { student_id: studentId }
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
    method: "GET"
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
    method: "DELETE"
  })
}
