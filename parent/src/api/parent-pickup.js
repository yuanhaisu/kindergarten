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

// 指定今日接走人（pickup-custody-evidence 新增）
//
// student_id 与其它接口一样放 query（后端 parentCtx 只读 c.Query("student_id")）。
// 指定会覆盖该孩当日此前的指定，并作废当日已生成的凭证 —— 调用方须提示用户。
export function designatePickupPerson(authorizationId, studentId) {
  const q = studentId ? `?student_id=${studentId}` : ""
  return requestWithAuth({
    url: `/parent/pickup/designate${q}`,
    method: "POST",
    data: { authorization_id: authorizationId }
  })
}

// 只读查「今日接走人」指定（无副作用，不生成凭证）。
// 供授权人列表页展示当前指定。
export function fetchDesignatedPerson(studentId) {
  return requestWithAuth({
    url: "/parent/pickup/designation",
    method: "GET",
    data: { student_id: studentId }
  })
}
