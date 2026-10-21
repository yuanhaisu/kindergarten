import { requestWithAuth } from "@/api/http"
import { getCurrentStudentId, syncCurrentStudentFromOverview } from "@/utils/currentStudent"
import { fetchParentHomeOverview } from "@/api/parent-home"

// 本地缓存的孩子可能已被解绑/移走：任务接口会返回 403。
// 用 overview 重新落盘服务端认定的当前孩子，供调用方重试一次。
export async function refreshCurrentStudentId() {
  const res = await fetchParentHomeOverview()
  syncCurrentStudentFromOverview(res.data)
  return getCurrentStudentId()
}

export function fetchTasks(params = {}) {
  return requestWithAuth({
    url: "/parent/tasks",
    method: "GET",
    data: params
  })
}

export function fetchTaskDetail(id, params = {}) {
  return requestWithAuth({
    url: `/parent/tasks/${id}`,
    method: "GET",
    data: params
  })
}

// student_id 必须拼在 query 上：打卡 handler 只从 body 里取 content/media_file_id，
// body 中的 student_id 会被丢弃，真正生效的是 parentTaskCtx 读的 query 参数。
export function checkinTask(id, data, studentId) {
  const q = studentId ? `?student_id=${studentId}` : ""
  return requestWithAuth({
    url: `/parent/tasks/${id}/checkin${q}`,
    method: "POST",
    data
  })
}

// 打卡记录的 handler 走 parentTaskCtx，student_id 必填
export function fetchCheckinHistory(id, params = {}) {
  return requestWithAuth({
    url: `/parent/tasks/${id}/checkin-history`,
    method: "GET",
    data: params
  })
}
