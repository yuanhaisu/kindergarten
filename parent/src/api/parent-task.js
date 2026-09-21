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

// student_id 必须拼在 query 上：打卡 handler 只从 body 里取 content/image_url，
// body 中的 student_id 会被丢弃，真正生效的是 parentCtx 读的 query 参数。
export function checkinTask(id, data, studentId) {
  const q = studentId ? `?student_id=${studentId}` : ""
  return requestWithAuth({
    url: `/parent/tasks/${id}/checkin${q}`,
    method: "POST",
    data
  })
}

// 打卡记录的 handler 走 parentCtx，支持 query 形式的 student_id
export function fetchCheckinHistory(id, params = {}) {
  return requestWithAuth({
    url: `/parent/tasks/${id}/checkin-history`,
    method: "GET",
    data: params
  })
}
