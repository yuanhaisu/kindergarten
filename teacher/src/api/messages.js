import { requestWithAuth } from "@/api/http"

/** 获取老师端消息聚合列表 */
export function fetchTeacherMessages(params = {}) {
  return requestWithAuth({ url: "/teacher/messages", method: "GET", data: params })
}
