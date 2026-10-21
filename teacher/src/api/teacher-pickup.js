import { requestWithAuth } from "@/api/http"

export function fetchTeacherPickupOverview() {
  return requestWithAuth({ url: "/teacher/pickup/overview", method: "GET" })
}

export function fetchTeacherPickupStudents(params = {}) {
  return requestWithAuth({ url: "/teacher/pickup/students", method: "GET", data: params })
}

// 扫码只传凭证内容，学生与接走人由后端从码自解释
export function scanTeacherPickup(data) {
  return requestWithAuth({ url: "/teacher/pickup/scan", method: "POST", data })
}

// 手动登记：authorization_id 选授权人 或 is_authorized_override+override_name 标例外
// remark 在重复接走/提前接走时必填
export function manualTeacherPickup(data) {
  return requestWithAuth({ url: "/teacher/pickup/manual", method: "POST", data })
}

// 一键通知滞留儿童家长
export function notifyOverdueParent(studentId) {
  return requestWithAuth({ url: "/teacher/pickup/notify-overdue", method: "POST", data: { student_id: studentId } })
}

// 按学生查有效授权接送人（手动登记选人用）
export function fetchStudentAuthPersons(studentId) {
  return requestWithAuth({ url: `/teacher/pickup/students/${studentId}/authorized-persons`, method: "GET" })
}

export function fetchTeacherPickupRecords(params = {}) {
  return requestWithAuth({ url: "/teacher/pickup/records", method: "GET", data: params })
}

export function fetchTeacherPickupRecord(id) {
  return requestWithAuth({ url: `/teacher/pickup/records/${id}`, method: "GET" })
}

export function markTeacherPickupAbnormal(id, reason) {
  return requestWithAuth({ url: `/teacher/pickup/records/${id}/abnormal`, method: "PUT", data: { reason } })
}
