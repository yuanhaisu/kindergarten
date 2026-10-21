import { requestWithAuth } from "@/api/http"

// 最近 7 天已完成的晨检记录；只允许查询本人绑定孩子。
export function fetchHealthRecords(studentId) {
  return requestWithAuth({
    url: "/parent/health/records",
    method: "GET",
    data: { student_id: studentId }
  })
}
