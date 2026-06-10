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
    }
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
