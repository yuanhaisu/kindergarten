/**
 * 管理端认证 Cookie 的唯一写入来源。
 *
 * `Secure` 由当前页面协议决定：HTTPS 下带 Secure，本地 HTTP 开发不带——
 * 否则浏览器会直接丢弃 Cookie，登录态根本建不起来。管理端是 SPA（ssr: false），
 * 判定在客户端完成；若未来改为 SSR，需要重新评估服务端写入时的协议来源。
 *
 * 不设 `HttpOnly`：令牌需要被 JS 读取后放进 `Authorization` 头，
 * 改成 HttpOnly 得让后端下发 Set-Cookie，属于另一项契约变更。
 */

export const ADMIN_TOKEN_COOKIE = 'admin_token'
export const ADMIN_REFRESH_COOKIE = 'admin_refresh'
export const PLATFORM_TOKEN_COOKIE = 'platform_token'
export const PLATFORM_REFRESH_COOKIE = 'platform_refresh'

/** admin_token 有效期 7 天（沿用既有行为） */
export const ADMIN_TOKEN_MAX_AGE = 60 * 60 * 24 * 7
/** platform_token 有效期 24 小时；访问凭证本身约 15 分钟，过期后由续期刷新 */
export const PLATFORM_TOKEN_MAX_AGE = 60 * 60 * 24
/** platform_refresh 有效期 7 天，与服务端 jwt.refresh_ttl_day 对齐 */
export const PLATFORM_REFRESH_MAX_AGE = 60 * 60 * 24 * 7

/** 不随环境变化的 Cookie 默认属性 */
const authCookieDefaults = { sameSite: 'lax' } as const

export interface AuthCookieOptions {
  sameSite: 'lax'
  secure: boolean
  maxAge?: number
}

export interface BuildAuthCookieOptionsInput {
  /** 不传即会话级 Cookie（admin_refresh 依赖此行为） */
  maxAge?: number
  /** 当前页面协议；默认取 window.location.protocol，测试可显式传入 */
  protocol?: string
}

function currentProtocol(): string {
  return typeof window !== 'undefined' ? window.location.protocol : 'http:'
}

export function buildAuthCookieOptions(
  input: BuildAuthCookieOptionsInput = {},
): AuthCookieOptions {
  const protocol = input.protocol ?? currentProtocol()
  const options: AuthCookieOptions = {
    ...authCookieDefaults,
    secure: protocol === 'https:',
  }
  // maxAge 缺省时不出现该键：Nuxt 据此把 Cookie 写成会话级。
  if (input.maxAge !== undefined) options.maxAge = input.maxAge
  return options
}
