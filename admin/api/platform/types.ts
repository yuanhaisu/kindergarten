export interface Paged<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

/** 平台登录与续期共用的路径常量（续期拦截器也引用，避免字面量漂移）。 */
export const PLATFORM_AUTH_LOGIN_PATH = '/platform/auth/login'
export const PLATFORM_AUTH_REFRESH_PATH = '/platform/auth/refresh'

/** 登录与续期响应同构：访问凭证 + 轮换后的续期凭证。 */
export interface PlatformAuthTokens {
  token: string
  refresh_token?: string
  expires_in?: number
  admin?: { id: number; username: string; name: string }
}
