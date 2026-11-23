/**
 * 管理端登录/续期/门户路径常量。
 *
 * 接口封装与续期拦截器共用同一组常量：拦截器要按 URL 排除「登录 401 是密码错误」
 * 与「续期 401 是凭证失效」，字面量各写一份迟早会漂移。
 */
export const ADMIN_AUTH_LOGIN_PATH = '/auth/login/password'
export const ADMIN_AUTH_REFRESH_PATH = '/auth/refresh'
export const ADMIN_AUTH_LOGOUT_PATH = '/auth/logout'
export const ADMIN_LOGIN_PAGE_PATH = '/login'
export const ADMIN_CONSENT_PAGE_PATH = '/consent'
export const ADMIN_CHANGE_PASSWORD_PAGE_PATH = '/change-password'

/** 只允许站内相对路径回跳，挡掉 `https://…` 与 `//host` 外跳。 */
export function safeRedirectPath(raw: unknown, fallback = '/'): string {
  if (typeof raw !== 'string') return fallback
  if (!raw.startsWith('/') || raw.startsWith('//') || raw.startsWith('/\\')) return fallback
  return raw
}
