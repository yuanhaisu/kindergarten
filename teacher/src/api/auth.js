import config from "@/config/env"
import { rawRequest, requestWithAuth, switchKindergartenSession } from "@/api/http"

export function wxCodeLogin({ code, encryptedData, iv }) {
  return rawRequest({
    url: "/auth/login/wechat",
    method: "POST",
    data: {
      // 客户端身份（#163/#175）：后端按 app_id 查客户端注册表决定角色与作用域。
      app_id: config.appId,
      code,
      role: "teacher",
      encrypted_data: encryptedData,
      iv
    },
    // 登录由登录页自己渲染错误态，不走全局 toast ——
    // 否则 http.js 弹一次、页面 catch 再弹一次，后者还会盖掉前者更准的文案。
    silent: true
  })
}

export function refreshToken(refreshToken) {
  return rawRequest({
    url: "/auth/refresh",
    method: "POST",
    data: {
      refresh_token: refreshToken
    }
  })
}

export function bindInvite(inviteCode) {
  return requestWithAuth({
    url: "/common/user/bind-invite",
    method: "POST",
    data: { invite_code: inviteCode },
    // 绑定失败由登录页渲染内联错误条，避免与页面 catch 各弹一次。
    silent: true
  })
}

export function fetchCurrentUser() {
  return requestWithAuth({
    url: "/common/user/profile",
    method: "GET"
  })
}

// 我的园所（#169）：后端已按当前客户端角色族与园所通道过滤，前端不再做二次筛选。
export function fetchMyKindergartens() {
  return requestWithAuth({
    url: "/common/user/kindergartens",
    method: "GET"
  })
}

export function switchKindergarten(kindergartenID) {
  return switchKindergartenSession(kindergartenID)
}
