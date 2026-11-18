import { readFileSync, readdirSync } from 'node:fs'
import { dirname, extname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import {
  ADMIN_REFRESH_COOKIE,
  ADMIN_TOKEN_COOKIE,
  PLATFORM_REFRESH_COOKIE,
  PLATFORM_REFRESH_MAX_AGE,
  PLATFORM_TOKEN_COOKIE,
  PLATFORM_TOKEN_MAX_AGE,
  buildAuthCookieOptions,
} from '../../utils/authCookie'

/**
 * 认证 Cookie 的写入属性集中在 `utils/authCookie`：
 * `Secure` 由当前页面协议决定（HTTPS 才带），`SameSite` 固定 `lax`。
 * 这里只验证纯函数的可观察输出，不依赖 Nuxt 运行时。
 */

describe('auth cookie names', () => {
  it('四个 Cookie 名称稳定', () => {
    expect(ADMIN_TOKEN_COOKIE).toBe('admin_token')
    expect(ADMIN_REFRESH_COOKIE).toBe('admin_refresh')
    expect(PLATFORM_TOKEN_COOKIE).toBe('platform_token')
    expect(PLATFORM_REFRESH_COOKIE).toBe('platform_refresh')
  })

  it('平台续期 Cookie 有效期 7 天，与访问凭证 Cookie 分开', () => {
    expect(PLATFORM_REFRESH_MAX_AGE).toBe(60 * 60 * 24 * 7)
    expect(PLATFORM_TOKEN_MAX_AGE).toBe(60 * 60 * 24)
    expect(PLATFORM_REFRESH_MAX_AGE).toBeGreaterThan(PLATFORM_TOKEN_MAX_AGE)
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

describe('cookie literal regression guard', () => {
  const adminRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
  const SCANNED_DIRS = ['composables', 'stores', 'middleware', 'pages', 'layouts']
  const COOKIE_NAMES = [
    ADMIN_TOKEN_COOKIE,
    ADMIN_REFRESH_COOKIE,
    PLATFORM_TOKEN_COOKIE,
    PLATFORM_REFRESH_COOKIE,
  ]
  const SOURCE_EXTENSIONS = new Set(['.ts', '.js', '.vue'])

  function* walk(dir: string): Generator<string> {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const full = join(dir, entry.name)
      if (entry.isDirectory()) yield* walk(full)
      else if (entry.isFile()) yield full
    }
  }

  it('业务代码只通过 utils/authCookie 引用 Cookie 名称', () => {
    const offenders: string[] = []

    for (const dir of SCANNED_DIRS) {
      for (const file of walk(join(adminRoot, dir))) {
        if (!SOURCE_EXTENSIONS.has(extname(file))) continue
        const text = readFileSync(file, 'utf8')
        for (const name of COOKIE_NAMES) {
          // 只拦字符串字面量：注释里提到名称不算绕过写入侧统一构造器。
          if (new RegExp(`['"]${name}['"]`).test(text)) {
            offenders.push(`${relative(adminRoot, file)}: ${name}`)
          }
        }
      }
    }

    expect(offenders).toEqual([])
  })
})
