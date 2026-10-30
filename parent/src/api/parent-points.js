import { requestWithAuth } from "@/api/http"

// silent：积分页自己渲染骨架 / 错误 / 重试，全局 toast 会与页面错误态重复；
// 成长页也刻意容忍成就接口失败（catch 成 null），不需要弹提示。
export function fetchPointsOverview(studentId) {
  return requestWithAuth({
    url: "/parent/points/overview",
    method: "GET",
    data: { student_id: studentId },
    silent: true
  })
}

export function fetchPointsHistory(params = {}) {
  return requestWithAuth({
    url: "/parent/points/history",
    method: "GET",
    data: params,
    silent: true
  })
}

export function fetchLeaderboard(params = {}) {
  return requestWithAuth({
    url: "/parent/points/leaderboard",
    method: "GET",
    data: params,
    silent: true
  })
}

export function fetchAchievements(studentId) {
  return requestWithAuth({
    url: "/parent/achievements",
    method: "GET",
    data: { student_id: studentId },
    silent: true
  })
}
