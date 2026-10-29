import { requestWithAuth } from "@/api/http"

// options 透传给 requestWithAuth（如 { silent: true }），
// 让弹层/页面错误态自己渲染错误，避免与全局 toast 重复。
export function fetchTeacherPickupOverview(options = {}) {
  return requestWithAuth({ url: "/teacher/pickup/overview", method: "GET", ...options })
}

export function fetchTeacherPickupStudents(params = {}, options = {}) {
  return requestWithAuth({ url: "/teacher/pickup/students", method: "GET", data: params, ...options })
}

// 扫码只传凭证内容，学生与接走人由后端从码自解释
export function scanTeacherPickup(data, options = {}) {
  return requestWithAuth({ url: "/teacher/pickup/scan", method: "POST", data, ...options })
}

// 手动登记：authorization_id 选授权人 或 is_authorized_override+override_name 标例外
// remark 在重复接走/提前接走时必填
export function manualTeacherPickup(data, options = {}) {
  return requestWithAuth({ url: "/teacher/pickup/manual", method: "POST", data, ...options })
}

// 一键通知滞留儿童家长
export function notifyOverdueParent(studentId, options = {}) {
  return requestWithAuth({ url: "/teacher/pickup/notify-overdue", method: "POST", data: { student_id: studentId }, ...options })
}

// 按学生查有效授权接送人（手动登记选人用）
export function fetchStudentAuthPersons(studentId, options = {}) {
  return requestWithAuth({ url: `/teacher/pickup/students/${studentId}/authorized-persons`, method: "GET", ...options })
}

export function fetchTeacherPickupRecords(params = {}, options = {}) {
  return requestWithAuth({ url: "/teacher/pickup/records", method: "GET", data: params, ...options })
}

export function fetchTeacherPickupRecord(id) {
  return requestWithAuth({ url: `/teacher/pickup/records/${id}`, method: "GET" })
}

export function markTeacherPickupAbnormal(id, reason) {
  return requestWithAuth({ url: `/teacher/pickup/records/${id}/abnormal`, method: "PUT", data: { reason } })
}
