import { requestWithAuth } from "@/api/http"

// silent：列表错误与提交失败各有自己的持久面板 / 失败条（两条独立通道），
// 让全局 toast 再弹一次会把「独立重试」的语义冲掉。
export function submitFeedback(data) {
  return requestWithAuth({
    url: "/parent/feedback",
    method: "POST",
    data,
    silent: true
  })
}

export function fetchFeedbackList(params = {}) {
  return requestWithAuth({
    url: "/parent/feedback",
    method: "GET",
    data: params,
    silent: true
  })
}
