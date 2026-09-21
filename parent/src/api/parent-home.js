import { requestWithAuth } from "@/api/http"

// student_id 以 query 传给后端（parentCtx 读的是 c.Query("student_id")）。
// 不传时后端回退到绑定关系里的第一个孩子。
export function fetchParentHomeOverview(studentId) {
  return requestWithAuth({
    url: "/parent/home/overview",
    method: "GET",
    data: { student_id: studentId || undefined }
  })
}

export function switchStudent(studentId) {
  return requestWithAuth({
    url: "/parent/home/switch-student",
    method: "PUT",
    data: { student_id: studentId }
  })
}

export function fetchAnnouncements(limit = 5) {
  return requestWithAuth({
    url: "/parent/home/announcements",
    method: "GET",
    data: { limit }
  })
}

export function fetchHomeFeeds(params = {}) {
  return requestWithAuth({
    url: "/parent/home/feeds",
    method: "GET",
    data: params
  })
}
