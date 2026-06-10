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

export function createAuthorizedPerson(data) {
  return requestWithAuth({
    url: "/parent/pickup/authorized-persons",
    method: "POST",
    data
  })
}

export function updateAuthorizedPerson(id, data) {
  return requestWithAuth({
    url: `/parent/pickup/authorized-persons/${id}`,
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
