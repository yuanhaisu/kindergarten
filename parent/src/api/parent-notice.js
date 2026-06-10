import { requestWithAuth } from "@/api/http"

export function fetchNotices(params = {}) {
  return requestWithAuth({
    url: "/parent/notices",
    method: "GET",
    data: params
  })
}

export function fetchNoticeDetail(id) {
  return requestWithAuth({
    url: `/parent/notices/${id}`,
    method: "GET"
  })
}

export function confirmNotice(id) {
  return requestWithAuth({
    url: `/parent/notices/${id}/confirm`,
    method: "PUT",
    data: {}
  })
}
