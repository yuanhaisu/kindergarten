import { describe, expect, it } from 'vitest'
import {
  ADMIN_REFRESH_COOKIE,
  ADMIN_TOKEN_COOKIE,
  PLATFORM_TOKEN_COOKIE,
  buildAuthCookieOptions,
} from '../../utils/authCookie'

/**
 * 认证 Cookie 的写入属性集中在 `utils/authCookie`：
 * `Secure` 由当前页面协议决定（HTTPS 才带），`SameSite` 固定 `lax`。
 * 这里只验证纯函数的可观察输出，不依赖 Nuxt 运行时。
 */

describe('auth cookie names', () => {
  it('三个 Cookie 名称稳定', () => {
    expect(ADMIN_TOKEN_COOKIE).toBe('admin_token')
    expect(ADMIN_REFRESH_COOKIE).toBe('admin_refresh')
    expect(PLATFORM_TOKEN_COOKIE).toBe('platform_token')
  })
})

describe('buildAuthCookieOptions', () => {
  it('HTTP 环境返回 secure: false', () => {
    expect(buildAuthCookieOptions({ protocol: 'http:' }).secure).toBe(false)
  })

  it('HTTPS 环境返回 secure: true', () => {
    expect(buildAuthCookieOptions({ protocol: 'https:' }).secure).toBe(true)
  })

  it('sameSite 始终为 lax', () => {
    expect(buildAuthCookieOptions({ protocol: 'http:' }).sameSite).toBe('lax')
    expect(buildAuthCookieOptions({ protocol: 'https:' }).sameSite).toBe('lax')
  })

  it('maxAge 透传，未传时不出现', () => {
    const sevenDays = 60 * 60 * 24 * 7

    expect(buildAuthCookieOptions({ protocol: 'https:', maxAge: sevenDays }).maxAge).toBe(sevenDays)
    expect('maxAge' in buildAuthCookieOptions({ protocol: 'https:' })).toBe(false)
  })

  it('不传协议时按当前页面判定（jsdom 为 http，不应误带 secure）', () => {
    expect(buildAuthCookieOptions().secure).toBe(false)
  })
})
