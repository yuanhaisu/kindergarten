import { requestWithAuth } from "@/api/http"

export function fetchTeacherTasks(params = {}) {
  return requestWithAuth({ url: "/teacher/tasks", method: "GET", data: params })
}

export function fetchTeacherTask(id) {
  return requestWithAuth({ url: `/teacher/tasks/${id}`, method: "GET" })
}

export function createTeacherTask(data) {
  return requestWithAuth({ url: "/teacher/tasks", method: "POST", data })
}

export function updateTeacherTask(id, data) {
  return requestWithAuth({ url: `/teacher/tasks/${id}`, method: "PUT", data })
}

export function deleteTeacherTask(id) {
  return requestWithAuth({ url: `/teacher/tasks/${id}`, method: "DELETE" })
}

export function fetchTeacherTaskStats(id) {
  return requestWithAuth({ url: `/teacher/tasks/${id}/stats`, method: "GET" })
}

export function fetchTeacherTaskCheckins(id, params = {}) {
  return requestWithAuth({ url: `/teacher/tasks/${id}/checkins`, method: "GET", data: params })
}

export function respondTeacherTask(taskId, checkinId, data) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/checkins/${checkinId}/response`,
    method: "POST",
    data
  })
}

export function batchRespondTeacherTask(taskId, data) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/batch-response`,
    method: "POST",
    data
  })
}

// 汇总/单条回应之后追加个别补充。只能发给已有主回应的提交（待回应提交后端 422），
// 仪式仅支持 sticker / voice，class_star 会被拒绝（一条提交只有一个主回应能计星）。
export function supplementTeacherResponse(taskId, checkinId, data) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/checkins/${checkinId}/supplement`,
    method: "POST",
    data
  })
}

// 老师审核家长公开申请：批准必须带 scope_confirmed=true（确认内容里只有提交者孩子）；
// 拒绝/撤回传 approve=false，任务关掉公开开关后仍可用（契约要求撤回永远可用）。
export function reviewTeacherTaskPublic(taskId, checkinId, data) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/checkins/${checkinId}/public`,
    method: "PUT",
    data
  })
}

// 班级星星罐：目标与当前学期进度一起返回（stars / progress / over_goal）。
// 班级取自登录态，不能传 class_id；未配置时 configured=false、goal=0、progress=null。
export function fetchClassStarJar() {
  return requestWithAuth({ url: "/teacher/class/star-jar", method: "GET" })
}

// 设置班级星星罐目标：必须 > 0（0 表示未配置，后端 422）。
export function setClassStarJar(goal) {
  return requestWithAuth({ url: "/teacher/class/star-jar", method: "PUT", data: { goal } })
}

// 家长端任务详情的小程序码（PNG base64）。码由后端用家长端凭据生成，
// 客户端只负责落盘到相册；返回 { task_id, page, scene, env_version, image_base64 }。
export function fetchTaskShareCard(taskId) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/share-card`,
    method: "GET"
  })
}
