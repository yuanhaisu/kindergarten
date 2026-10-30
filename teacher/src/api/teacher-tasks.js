import { requestWithAuth } from "@/api/http"

// 第三个参数 options 透传给 http 层（如 { silent: true }）。
// 页面自己渲染错误态时用 silent 避免再叠一条全局 toast。

export function fetchTeacherTasks(params = {}, options = {}) {
  return requestWithAuth({ url: "/teacher/tasks", method: "GET", data: params, ...options })
}

export function fetchTeacherTask(id, options = {}) {
  return requestWithAuth({ url: `/teacher/tasks/${id}`, method: "GET", ...options })
}

export function createTeacherTask(data, options = {}) {
  return requestWithAuth({ url: "/teacher/tasks", method: "POST", data, ...options })
}

export function updateTeacherTask(id, data, options = {}) {
  return requestWithAuth({ url: `/teacher/tasks/${id}`, method: "PUT", data, ...options })
}

export function deleteTeacherTask(id, options = {}) {
  return requestWithAuth({ url: `/teacher/tasks/${id}`, method: "DELETE", ...options })
}

export function fetchTeacherTaskStats(id, options = {}) {
  return requestWithAuth({ url: `/teacher/tasks/${id}/stats`, method: "GET", ...options })
}

export function fetchTeacherTaskCheckins(id, params = {}, options = {}) {
  return requestWithAuth({ url: `/teacher/tasks/${id}/checkins`, method: "GET", data: params, ...options })
}

export function respondTeacherTask(taskId, checkinId, data, options = {}) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/checkins/${checkinId}/response`,
    method: "POST",
    data,
    ...options
  })
}

export function batchRespondTeacherTask(taskId, data, options = {}) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/batch-response`,
    method: "POST",
    data,
    ...options
  })
}

// 汇总/单条回应之后追加个别补充。只能发给已有主回应的提交（待回应提交后端 422），
// 仪式仅支持 sticker / voice，class_star 会被拒绝（一条提交只有一个主回应能计星）。
export function supplementTeacherResponse(taskId, checkinId, data, options = {}) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/checkins/${checkinId}/supplement`,
    method: "POST",
    data,
    ...options
  })
}

// 老师审核家长公开申请：批准必须带 scope_confirmed=true（确认内容里只有提交者孩子）；
// 拒绝/撤回传 approve=false，任务关掉公开开关后仍可用（契约要求撤回永远可用）。
export function reviewTeacherTaskPublic(taskId, checkinId, data, options = {}) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/checkins/${checkinId}/public`,
    method: "PUT",
    data,
    ...options
  })
}

// 班级星星罐：目标与当前学期进度一起返回（stars / progress / over_goal）。
// 班级取自登录态，不能传 class_id；未配置时 configured=false、goal=0、progress=null。
export function fetchClassStarJar(options = {}) {
  return requestWithAuth({ url: "/teacher/class/star-jar", method: "GET", ...options })
}

// 设置班级星星罐目标：必须 > 0（0 表示未配置，后端 422）。
export function setClassStarJar(goal, options = {}) {
  return requestWithAuth({ url: "/teacher/class/star-jar", method: "PUT", data: { goal }, ...options })
}

// 家长端任务详情的小程序码（PNG base64）。码由后端用家长端凭据生成，
// 客户端只负责落盘到相册；返回 { task_id, page, scene, env_version, image_base64 }。
export function fetchTaskShareCard(taskId, options = {}) {
  return requestWithAuth({
    url: `/teacher/tasks/${taskId}/share-card`,
    method: "GET",
    ...options
  })
}
