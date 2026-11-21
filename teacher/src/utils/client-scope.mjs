/**
 * 小程序客户端身份（#163）。
 *
 * app_id 与客户端版本由构建期注入（VITE_APP_ID / VITE_CLIENT_VERSION），
 * 缺省回落到构建内置值，保证产物总是带得动客户端身份。
 *
 * 登录（#175）把 appId 随登录请求上报，后端据此在 clients 注册表里
 * 找到客户端并裁剪角色；版本用于诊断，不参与鉴权。
 */
export function resolveClientScope(env, { fallbackAppId = "", fallbackVersion = "0.0.0" } = {}) {
  const appId = String(env?.VITE_APP_ID || fallbackAppId || "").trim()
  const clientVersion = String(env?.VITE_CLIENT_VERSION || fallbackVersion || "").trim()
  return { appId, clientVersion }
}
