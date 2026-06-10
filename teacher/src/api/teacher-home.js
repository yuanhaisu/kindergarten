import { requestWithAuth } from "@/api/http"

export function fetchTeacherHomeOverview() {
  return requestWithAuth({
    url: "/teacher/home/overview",
    method: "GET"
  })
}

export function fetchTeacherTodos(params = {}) {
  return requestWithAuth({
    url: "/teacher/home/todos",
    method: "GET",
    data: params
  })
}

export function updateTeacherTodoAction(id, action, remark = "") {
  return requestWithAuth({
    url: `/teacher/home/todos/${id}/action`,
    method: "PUT",
    data: { action, remark }
  })
}
