import { rawRequest, requestWithAuth } from "@/api/http"

export function wxCodeLogin({ code, encryptedData, iv, inviteCode }) {
  return rawRequest({
    url: "/auth/login/wechat",
    method: "POST",
    data: {
      code,
      role: "teacher",
      encrypted_data: encryptedData,
      iv,
      invite_code: inviteCode || undefined
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
    data: { invite_code: inviteCode }
  })
}

export function fetchCurrentUser() {
  return requestWithAuth({
    url: "/common/user/profile",
    method: "GET"
  })
}
