import { requestWithAuth } from "@/api/http"

export function fetchMessages(params = {}) {
  return requestWithAuth({
    url: "/common/messages",
    method: "GET",
    data: params
  })
}

export function updateMessage(data) {
  return requestWithAuth({
    url: "/common/messages",
    method: "PUT",
    data
  })
}

export function fetchUnreadCount() {
  return requestWithAuth({
    url: "/common/messages/unread-count",
    method: "GET"
  })
}
