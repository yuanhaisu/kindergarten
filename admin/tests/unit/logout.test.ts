import {
  AxiosError,
  type AxiosAdapter,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios'
import { describe, expect, it, vi } from 'vitest'
import { createAdminHttpClient } from '../../composables/useHttp'
import {
  createLogoutTransport,
  createSessionLogout,
  type LogoutTokenPair,
  type LogoutTransport,
} from '../../utils/sessionLogout'

/**
 * 登出收口（#199）的可观察行为：
 * - 访问凭证有效：一次 revoke，参数带 access + refresh，随后清本地并跳登录；
 * - 访问凭证过期：只手动续期一次，第二次 revoke 必须用**轮换后的** access/refresh；
 * - refresh 已死（401）：无可吊销，不告警，仍清本地；
 * - 网络/5xx：清本地 + 告警一次 + console.error；
 * - 双击只发一次 revoke（完成后重复调用也不再发）。
 *
 * 另外用真实 axios 拦截器链路验证 transport 的 skip 标志：缺少它时，
 * 第一次 401 会被拦截器自动续期并重放，body 仍是旧 refresh——这正是要避免的假修复。
 */

function httpError(status: number, message = 'failed'): Error & { status: number } {
  return Object.assign(new Error(message), { status })
}

interface HarnessOptions {
  revoke?: (access: string | null, refresh: string | null) => Promise<void>
  refresh?: (refresh: string) => Promise<LogoutTokenPair>
  accessToken?: string | null
  refreshToken?: string | null
}

function createHarness(options: HarnessOptions = {}) {
  const calls = {
    revoke: [] as Array<[string | null, string | null]>,
    refresh: [] as string[],
    saved: [] as LogoutTokenPair[],
    clear: 0,
    redirect: 0,
    warn: [] as string[],
    logged: [] as unknown[],
  }
  const transport: LogoutTransport = {
    revoke: async (access, refresh) => {
      calls.revoke.push([access, refresh])
      if (options.revoke) await options.revoke(access, refresh)
    },
    refresh: async (refresh) => {
      calls.refresh.push(refresh)
      if (options.refresh) return options.refresh(refresh)
      return { token: 'rotated-access', refresh_token: 'rotated-refresh' }
    },
  }
  const session = {
    getAccessToken: () => options.accessToken ?? 'old-access',
    getRefreshToken: () =>
      options.refreshToken === undefined ? 'old-refresh' : options.refreshToken,
    saveTokens: (tokens) => {
      calls.saved.push(tokens)
    },
    clearSession: () => {
      calls.clear += 1
    },
    redirectToLogin: () => {
      calls.redirect += 1
    },
  }
  const logout = createSessionLogout({
    transport,
    session,
    notifyWarning: (message) => calls.warn.push(message),
    logError: (err) => calls.logged.push(err),
  })
  return { logout, calls }
}

describe('createSessionLogout', () => {
  it('访问凭证有效时只 revoke 一次，带双凭证，随后清本地并跳登录', async () => {
    const harness = createHarness()

    await harness.logout()

    expect(harness.calls.revoke).toEqual([['old-access', 'old-refresh']])
    expect(harness.calls.refresh).toEqual([])
    expect(harness.calls.clear).toBe(1)
    expect(harness.calls.redirect).toBe(1)
    expect(harness.calls.warn).toEqual([])
    expect(harness.calls.logged).toEqual([])
  })

  it('访问凭证过期：手动续期一次，用轮换后的双凭证再 revoke', async () => {
    let attempt = 0
    const harness = createHarness({
      revoke: async () => {
        attempt += 1
        if (attempt === 1) throw httpError(401, 'access expired')
      },
    })

    await harness.logout()

    expect(harness.calls.revoke).toEqual([
      ['old-access', 'old-refresh'],
      ['rotated-access', 'rotated-refresh'],
    ])
    expect(harness.calls.refresh).toEqual(['old-refresh'])
    // 只写双凭证，不重置 user/consent
    expect(harness.calls.saved).toEqual([
      { token: 'rotated-access', refresh_token: 'rotated-refresh' },
    ])
    expect(harness.calls.clear).toBe(1)
    expect(harness.calls.redirect).toBe(1)
    expect(harness.calls.warn).toEqual([])
  })

  it('refresh 已失效（401）：无可吊销也不告警，仍清本地', async () => {
    const harness = createHarness({
      revoke: async () => {
        throw httpError(401, 'access expired')
      },
      refresh: async () => {
        throw httpError(401, 'refresh expired')
      },
    })

    await harness.logout()

    expect(harness.calls.revoke).toHaveLength(1)
    expect(harness.calls.refresh).toHaveLength(1)
    expect(harness.calls.clear).toBe(1)
    expect(harness.calls.redirect).toBe(1)
    expect(harness.calls.warn).toEqual([])
  })

  it('网络或 5xx：本地清态、告警一次、记录一次错误', async () => {
    const harness = createHarness({
      revoke: async () => {
        throw httpError(503, '服务不可用')
      },
    })

    await harness.logout()

    expect(harness.calls.clear).toBe(1)
    expect(harness.calls.redirect).toBe(1)
    expect(harness.calls.warn).toHaveLength(1)
    expect(harness.calls.warn[0]).toContain('本机登录状态已清除')
    expect(harness.calls.logged).toHaveLength(1)
  })

  it('双击只 revoke 一次；完成后再次调用也不再发请求', async () => {
    let release!: () => void
    const pending = new Promise<void>((resolve) => {
      release = resolve
    })
    const harness = createHarness({ revoke: () => pending })

    const first = harness.logout()
    const second = harness.logout()
    expect(harness.calls.revoke).toHaveLength(1)
    release()
    await Promise.all([first, second])

    await harness.logout()
    expect(harness.calls.revoke).toHaveLength(1)
    expect(harness.calls.clear).toBe(1)
  })

  it('没有 refresh 时 revoke 401 视为会话已失效，不续期不告警', async () => {
    const harness = createHarness({
      accessToken: 'old-access',
      refreshToken: null,
      revoke: async () => {
        throw httpError(401, '缺少认证凭证')
      },
    })

    await harness.logout()

    expect(harness.calls.revoke).toEqual([['old-access', null]])
    expect(harness.calls.refresh).toEqual([])
    expect(harness.calls.warn).toEqual([])
    expect(harness.calls.clear).toBe(1)
  })
})

// ---- 真实 axios 拦截器链路：skip 标志与轮换后的 refresh ----

interface HandlerResult {
  status: number
  data: unknown
}

function respond(
  config: InternalAxiosRequestConfig,
  { status, data }: HandlerResult,
): AxiosResponse {
  const response = {
    data,
    status,
    statusText: '',
    headers: {},
    config,
  } as AxiosResponse
  if (status >= 200 && status < 300) return response
  throw new AxiosError(
    `Request failed with status code ${status}`,
    'ERR_BAD_REQUEST',
    config,
    null,
    response,
  )
}

function adapterOf(handler: (config: InternalAxiosRequestConfig) => HandlerResult): AxiosAdapter {
  return async (config) => respond(config, handler(config))
}

function bodyOf(config: InternalAxiosRequestConfig): unknown {
  if (typeof config.data === 'string') return JSON.parse(config.data)
  return config.data
}

describe('createLogoutTransport + 管理端拦截器', () => {
  it('access 过期时不会被拦截器静默续期；第二次 revoke 用轮换后的 refresh', async () => {
    const logoutRequests: Array<{ auth?: string; body: unknown }> = []
    const refreshBodies: unknown[] = []

    const adapter = adapterOf((config) => {
      const url = config.url ?? ''
      if (url === '/auth/logout') {
        logoutRequests.push({
          auth: String(config.headers?.Authorization ?? '') || undefined,
          body: bodyOf(config),
        })
        if (config.headers?.Authorization === 'Bearer expired-access') {
          return { status: 401, data: { code: 401, message: '访问凭证已过期' } }
        }
        if (config.headers?.Authorization === 'Bearer rotated-access') {
          return { status: 200, data: { code: 0, message: 'ok', data: {} } }
        }
        return { status: 401, data: { code: 401, message: '缺少认证凭证' } }
      }
      if (url === '/auth/refresh') {
        refreshBodies.push(bodyOf(config))
        return {
          status: 200,
          data: {
            code: 0,
            message: 'ok',
            data: { token: 'rotated-access', refresh_token: 'rotated-refresh' },
          },
        }
      }
      return { status: 404, data: { code: 404, message: 'not found' } }
    })

    let accessToken: string | null = 'expired-access'
    let refreshToken: string | null = 'old-refresh'
    const saved: LogoutTokenPair[] = []
    const client = createAdminHttpClient({
      baseURL: '/api/v1',
      session: {
        getAccessToken: () => accessToken,
        getRefreshToken: () => refreshToken,
        saveTokens: (tokens) => {
          saved.push(tokens)
          accessToken = tokens.token
          if (tokens.refresh_token) refreshToken = tokens.refresh_token
        },
        clearSession: () => {
          accessToken = null
          refreshToken = null
        },
        redirectToLogin: vi.fn(),
        onLoginPage: () => false,
        redirectToConsent: vi.fn(),
        redirectToChangePassword: vi.fn(),
        onChangePasswordPage: () => false,
      },
      adapter,
    })

    const logout = createSessionLogout({
      transport: createLogoutTransport({
        client,
        logoutPath: '/auth/logout',
        refreshPath: '/auth/refresh',
        skipRefreshFlag: 'skipAdminAuthRefresh',
      }),
      session: {
        getAccessToken: () => accessToken,
        getRefreshToken: () => refreshToken,
        saveTokens: (tokens) => {
          saved.push(tokens)
          accessToken = tokens.token
          if (tokens.refresh_token) refreshToken = tokens.refresh_token
        },
        clearSession: () => {
          accessToken = null
          refreshToken = null
        },
        redirectToLogin: vi.fn(),
      },
    })

    await logout()

    expect(refreshBodies).toEqual([{ refresh_token: 'old-refresh' }])
    expect(saved).toEqual([{ token: 'rotated-access', refresh_token: 'rotated-refresh' }])
    expect(logoutRequests).toEqual([
      { auth: 'Bearer expired-access', body: { refresh_token: 'old-refresh' } },
      { auth: 'Bearer rotated-access', body: { refresh_token: 'rotated-refresh' } },
    ])
    expect(accessToken).toBeNull()
    expect(refreshToken).toBeNull()
  })
})
