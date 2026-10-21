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

export function bindParentPhone(wxPhoneCode) {
  return requestWithAuth({
    url: "/common/user/bind-parent-phone",
    method: "POST",
    data: { wx_phone_code: wxPhoneCode },
    // 首页绑卡自己渲染 bind-err（含 40031 的恢复指引），不走全局 toast ——
    // 否则 http.js 弹一次、卡片再渲染一次，前者还会盖掉后者更准的文案。
    silent: true
  })
}

export function fetchCurrentUser() {
  return requestWithAuth({
    url: "/common/user/profile",
    method: "GET"
  })
}

