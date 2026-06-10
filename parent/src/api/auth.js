import { rawRequest, requestWithAuth } from "@/api/http"

export function wxCodeLogin({ code, encryptedData, iv }) {
  return rawRequest({
    url: "/auth/login/wechat",
    method: "POST",
    data: {
      code,
      role: "parent",
      encrypted_data: encryptedData,
      iv
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

export function bindParentPhone(wxPhoneCode) {
  return requestWithAuth({
    url: "/common/user/bind-parent-phone",
    method: "POST",
    data: { wx_phone_code: wxPhoneCode }
  })
}

export function fetchCurrentUser() {
  return requestWithAuth({
    url: "/common/user/profile",
    method: "GET"
  })
}
