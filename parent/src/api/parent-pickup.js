import { requestWithAuth } from "@/api/http"

export function fetchPickupQRCode(studentId) {
  return requestWithAuth({
    url: "/parent/pickup/qrcode",
    method: "GET",
    data: { student_id: studentId }
  })
}

export function fetchPickupRecords(params = {}) {
  return requestWithAuth({
    url: "/parent/pickup/records",
    method: "GET",
    data: params
  })
}

export function fetchAuthorizedPersons(studentId) {
  return requestWithAuth({
    url: "/parent/pickup/authorized-persons",
    method: "GET",
    data: { student_id: studentId }
  })
}

// 增改都必须把 student_id 放在 query 上：后端用的 `pc` 来自 parentCtx，
// 而 parentCtx 读的是 c.Query("student_id") —— body 里的 student_id 不会被读取。
export function createAuthorizedPerson(data, studentId) {
  const q = studentId ? `?student_id=${studentId}` : ""
  return requestWithAuth({
    url: `/parent/pickup/authorized-persons${q}`,
    method: "POST",
    data
  })
}

export function updateAuthorizedPerson(id, data, studentId) {
  const q = studentId ? `?student_id=${studentId}` : ""
  return requestWithAuth({
    url: `/parent/pickup/authorized-persons/${id}${q}`,
    method: "PUT",
    data
  })
}

export function deleteAuthorizedPerson(id) {
  return requestWithAuth({
    url: `/parent/pickup/authorized-persons/${id}`,
    method: "DELETE"
  })
}
