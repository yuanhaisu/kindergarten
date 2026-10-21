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

// silent: 这三个接口参与「403 → 刷新孩子 → 重试」，全局 toast 会在重试成功前
// 先弹一句「无权查看该孩子的任务」；错误提示由调用页自己渲染。
// 列表 / 详情共用同一条「解析当前孩子 → 403 刷新 overview 重试一次」路径，
// 避免两处副本各自漂移（例如只在一处静默、或重试次数不一致）。
// fetcher(sid) 返回接口结果（可以是 Promise.all 的数组）；返回 null 表示取不到有效孩子。
export async function withStudentScope(fetcher) {
  let sid = getCurrentStudentId()
  if (!sid) sid = await refreshCurrentStudentId()
  if (!sid) return null

  try {
    return { sid, result: await fetcher(sid) }
  } catch (err) {
    if (err?.code !== 40300) throw err
    const refreshed = await refreshCurrentStudentId()
    if (!refreshed) return null
    return { sid: refreshed, result: await fetcher(refreshed) }
  }
}

export function fetchTasks(params = {}) {
  return requestWithAuth({
    url: "/parent/tasks",
    method: "GET",
    data: params,
    silent: true
  })
}

export function fetchTaskDetail(id, params = {}) {
  return requestWithAuth({
    url: `/parent/tasks/${id}`,
    method: "GET",
    data: params,
    silent: true
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
    data: params,
    silent: true
  })
}
