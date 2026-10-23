import { requestWithAuth } from "@/api/http"

export function fetchMessages(params = {}) {
  return requestWithAuth({
    url: "/common/messages",
    method: "GET",
    data: params
  })
}

// 单条已读。路由是 `PUT /common/messages/:id/read`——原先这里写的是
// `PUT /common/messages`（不存在），因为没有调用点所以一直没暴露。
// silent：标记失败不该打断用户看内容，错误由调用方记日志。
export function markMessageRead(id) {
  return requestWithAuth({
    url: `/common/messages/${id}/read`,
    method: "PUT",
    data: {},
    silent: true
  })
}

export function fetchUnreadCount() {
  return requestWithAuth({
    url: "/common/messages/unread-count",
    method: "GET"
  })
}
