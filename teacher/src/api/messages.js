import { requestWithAuth } from "@/api/http"

// options 透传给 http 层（silent 用于页面自绘错误态，避免全局 toast 与内联错误条重复）。
export function fetchTeacherMessages(params = {}, options = {}) {
  return requestWithAuth({ url: "/teacher/messages", method: "GET", data: params, ...options })
}

// type 是 query 参数而不是 body（契约 10-messages.md），:id 的含义随 type 变化：
// notice/leave 用业务记录 id，system 用 messages.id。
export function markTeacherMessageRead(id, type, options = {}) {
  return requestWithAuth({
    url: `/teacher/messages/${id}/read?type=${encodeURIComponent(type)}`,
    method: "PUT",
    ...options
  })
}

// 作用域是全量，不是当前分类（契约明确）。
export function markAllTeacherMessagesRead(options = {}) {
  return requestWithAuth({ url: "/teacher/messages/read-all", method: "PUT", ...options })
}
