import { requestWithAuth } from "@/api/http"

// 授予入口的徽章选择器：只列本园已启用的徽章
export function fetchTeacherAchievements() {
  return requestWithAuth({ url: "/teacher/achievements", method: "GET" })
}

// 授予徽章给学生，可选绑定成长记录作为获奖见证（log_id 不传 = 不绑见证）
export function grantAchievement(achievementId, data) {
  return requestWithAuth({
    url: `/teacher/achievements/${achievementId}/grant`,
    method: "POST",
    data,
  })
}
