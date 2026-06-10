import { requestWithAuth } from "@/api/http"

export function fetchParentHomeOverview() {
  return requestWithAuth({
    url: "/parent/home/overview",
    method: "GET"
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
