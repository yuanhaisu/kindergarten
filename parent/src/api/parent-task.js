import { requestWithAuth } from "@/api/http"

export function fetchTasks(params = {}) {
  return requestWithAuth({
    url: "/parent/tasks",
    method: "GET",
    data: params
  })
}

export function fetchTaskDetail(id) {
  return requestWithAuth({
    url: `/parent/tasks/${id}`,
    method: "GET"
  })
}

export function checkinTask(id, data) {
  return requestWithAuth({
    url: `/parent/tasks/${id}/checkin`,
    method: "POST",
    data
  })
}

export function fetchCheckinHistory(id, params = {}) {
  return requestWithAuth({
    url: `/parent/tasks/${id}/checkin-history`,
    method: "GET",
    data: params
  })
}
