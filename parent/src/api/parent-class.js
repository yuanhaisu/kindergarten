import { requestWithAuth } from "@/api/http"

// silent：页面自带整页错误态与重试，避免 http 层再叠一次全局 toast。
export function fetchClassInfo(studentId) {
  return requestWithAuth({
    url: "/parent/class/info",
    method: "GET",
    data: { student_id: studentId },
    silent: true
  })
}

// start/end 由调用页按「视图月份 ∪ 今天~今天+7天」算出后透传，缺了只能拿到
// 服务端默认窗口，翻月会是空的。
// silent：日历页自带整页错误态与重试，http 层再弹一次 toast 会与面板重复。
export function fetchClassCalendar(params = {}) {
  return requestWithAuth({
    url: "/parent/class/calendar",
    method: "GET",
    data: params,
    silent: true
  })
}
