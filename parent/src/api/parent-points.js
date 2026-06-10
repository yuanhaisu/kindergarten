import { requestWithAuth } from "@/api/http"

export function fetchPointsOverview(studentId) {
  return requestWithAuth({
    url: "/parent/points/overview",
    method: "GET",
    data: { student_id: studentId }
  })
}

export function fetchPointsHistory(params = {}) {
  return requestWithAuth({
    url: "/parent/points/history",
    method: "GET",
    data: params
  })
}

export function fetchLeaderboard(params = {}) {
  return requestWithAuth({
    url: "/parent/points/leaderboard",
    method: "GET",
    data: params
  })
}

export function fetchAchievements(studentId) {
  return requestWithAuth({
    url: "/parent/achievements",
    method: "GET",
    data: { student_id: studentId }
  })
}
