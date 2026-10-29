import { requestWithAuth } from "@/api/http"

/**
 * 契约默认值（backend/api/api-parent/11-settings.md）。
 *
 * 后端无记录时也按这套返回，前端再兜一层：响应里缺键、类型不是布尔的键
 * 一律回落到默认值，保证页面永远拿到 8 个布尔开关，不会渲染出 undefined。
 */
export const SETTINGS_DEFAULTS = {
  notification: {
    notice_push: true,
    leave_push: true,
    pickup_push: true,
    growth_push: true,
    task_push: false,
    announcement_push: true
  },
  privacy: {
    show_phone_to_teacher: true,
    allow_growth_share: false
  }
}

function pickGroup(source, defaults) {
  const group = source && typeof source === "object" ? source : {}
  return Object.keys(defaults).reduce((result, key) => {
    result[key] = typeof group[key] === "boolean" ? group[key] : defaults[key]
    return result
  }, {})
}

/** 把 GET /parent/settings 的响应归一化为「两组八键、值全为布尔」。 */
export function normalizeSettings(data) {
  return {
    notification: pickGroup(data?.notification, SETTINGS_DEFAULTS.notification),
    privacy: pickGroup(data?.privacy, SETTINGS_DEFAULTS.privacy)
  }
}

export function fetchSettings() {
  return requestWithAuth({
    url: "/parent/settings",
    method: "GET"
  })
}

export function updateSettings(data) {
  return requestWithAuth({
    url: "/parent/settings",
    method: "PUT",
    data
  })
}
