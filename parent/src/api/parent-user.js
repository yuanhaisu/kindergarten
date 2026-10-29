import { requestWithAuth } from "@/api/http"

/**
 * 退出登录（使当前 access token 失效）。
 *
 * silent：设置页在确认退出后无论请求成败都会清本地并回登录页，失败提示由页面
 * 自己给；交给 http.js 弹全局 toast 会和页面的「已清除本地登录」提示互相覆盖。
 */
export function logout() {
  return requestWithAuth({
    url: "/auth/logout",
    method: "POST",
    data: {},
    silent: true
  })
}
