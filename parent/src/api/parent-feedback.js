import { requestWithAuth } from "@/api/http"

export function submitFeedback(data) {
  return requestWithAuth({
    url: "/parent/feedback",
    method: "POST",
    data
  })
}

export function fetchFeedbackList(params = {}) {
  return requestWithAuth({
    url: "/parent/feedback",
    method: "GET",
    data: params
  })
}
