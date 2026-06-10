import { requestWithAuth } from "@/api/http"

export function fetchLeaves(params = {}) {
  return requestWithAuth({
    url: "/parent/leaves",
    method: "GET",
    data: params
  })
}

export function fetchLeaveDetail(id) {
  return requestWithAuth({
    url: `/parent/leaves/${id}`,
    method: "GET"
  })
}

export function createLeave(data) {
  return requestWithAuth({
    url: "/parent/leaves",
    method: "POST",
    data
  })
}

export function updateLeave(id, data) {
  return requestWithAuth({
    url: `/parent/leaves/${id}`,
    method: "PUT",
    data
  })
}

export function cancelLeave(id) {
  return requestWithAuth({
    url: `/parent/leaves/${id}/cancel`,
    method: "PUT",
    data: {}
  })
}
