import {
  AxiosError,
  type AxiosAdapter,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios'
import { describe, expect, it } from 'vitest'
import {
  createPlatformHttpClient,
  type PlatformSessionAdapter,
  type PlatformTokenPair,
} from '../../composables/usePlatformHttp'

/**
 * 平台端单飞续期：拦截器是唯一观察点。
 * 用自定义 axios adapter 驱动真实拦截器链路，注入会话替身，
 * 断言「请求次数、重放结果、Cookie 写入与跳转」这些外部可观察行为。
 */

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

function createSession(initial: { accessToken?: string | null; refreshToken?: string | null } = {}) {
  let accessToken = initial.accessToken === undefined ? 'old-access' : initial.accessToken
  let refreshToken = initial.refreshToken === undefined ? 'old-refresh' : initial.refreshToken
  const saved: PlatformTokenPair[] = []
  const calls = { clear: 0, redirect: 0 }
  const session: PlatformSessionAdapter = {
    getAccessToken: () => accessToken,
    getRefreshToken: () => refreshToken,
    saveTokens: (pair) => {
      saved.push(pair)
      accessToken = pair.token
      if (pair.refresh_token) refreshToken = pair.refresh_token
    },
    clearSession: () => {
      calls.clear += 1
      accessToken = null
      refreshToken = null
    },
    redirectToLogin: () => {
      calls.redirect += 1
    },
    onLoginPage: () => false,
  }
  return {
    session,
    saved,
    calls,
    get accessToken() {
      return accessToken
    },
    get refreshToken() {
      return refreshToken
    },
  }
}

const REFRESH_URL = '/platform/auth/refresh'
const LOGIN_URL = '/platform/auth/login'

describe('usePlatformHttp 单飞续期', () => {
  it('并发 401 只触发一次续期，成功后更新两个凭证并重放原请求', async () => {
    const fake = createSession()
    const refreshBodies: unknown[] = []
    const refreshAuthHeaders: unknown[] = []

    const adapter = adapterOf((config) => {
      const url = config.url ?? ''
      if (url === REFRESH_URL) {
        refreshBodies.push(typeof config.data === 'string' ? JSON.parse(config.data) : config.data)
        refreshAuthHeaders.push(config.headers?.Authorization)
        return {
          status: 200,
          data: {
            code: 0,
            message: 'ok',
            data: { token: 'new-access', refresh_token: 'new-refresh', expires_in: 900 },
          },
        }
      }
      const auth = String(config.headers?.Authorization ?? '')
      if (auth === 'Bearer new-access') {
        return { status: 200, data: { code: 0, message: 'ok', data: { path: url } } }
      }
      return { status: 401, data: { code: 401, message: '访问凭证已过期' } }
    })

    const client = createPlatformHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    const [orders, plans] = await Promise.all([
      client.get('/platform/orders'),
      client.get('/platform/plans'),
    ])

    expect(refreshBodies).toHaveLength(1)
    expect(refreshBodies[0]).toEqual({ refresh_token: 'old-refresh' })
    // 续期请求不带过期的访问凭证
    expect(refreshAuthHeaders[0]).toBeUndefined()
    expect(fake.accessToken).toBe('new-access')
    expect(fake.refreshToken).toBe('new-refresh')
    expect(fake.saved).toEqual([
      expect.objectContaining({ token: 'new-access', refresh_token: 'new-refresh' }),
    ])
    expect(fake.calls).toEqual({ clear: 0, redirect: 0 })
    expect(orders.data).toEqual({ path: '/platform/orders' })
    expect(plans.data).toEqual({ path: '/platform/plans' })
  })

  it('登录端点返回 401（密码错误）不触发续期、不清态、不跳转', async () => {
    const fake = createSession()
    let refreshCount = 0

    const adapter = adapterOf((config) => {
      if (config.url === REFRESH_URL) {
        refreshCount += 1
        return { status: 200, data: { code: 0, data: { token: 'x', refresh_token: 'y' } } }
      }
      return { status: 401, data: { code: 401, message: '用户名或密码错误' } }
    })

    const client = createPlatformHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    await expect(client.post(LOGIN_URL, { username: 'ops', password: 'wrong' })).rejects.toMatchObject({
      message: '用户名或密码错误',
      status: 401,
    })
    expect(refreshCount).toBe(0)
    expect(fake.calls).toEqual({ clear: 0, redirect: 0 })
  })

  it('续期失败只清一次态、只跳转一次，所有等待的请求都失败', async () => {
    const fake = createSession()
    let refreshCount = 0

    const adapter = adapterOf((config) => {
      if (config.url === REFRESH_URL) {
        refreshCount += 1
        return { status: 401, data: { code: 401, message: '续期凭证已失效' } }
      }
      return { status: 401, data: { code: 401, message: '访问凭证已过期' } }
    })

    const client = createPlatformHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    const results = await Promise.allSettled([
      client.get('/platform/orders'),
      client.get('/platform/plans'),
      client.get('/platform/distributors'),
    ])

    expect(results.map((r) => r.status)).toEqual(['rejected', 'rejected', 'rejected'])
    expect(refreshCount).toBe(1)
    expect(fake.calls).toEqual({ clear: 1, redirect: 1 })
    expect(fake.accessToken).toBeNull()
    expect(fake.refreshToken).toBeNull()
  })

  it('重放后的请求再次 401 不会进入第二次续期', async () => {
    const fake = createSession()
    let refreshCount = 0

    const adapter = adapterOf((config) => {
      if (config.url === REFRESH_URL) {
        refreshCount += 1
        return {
          status: 200,
          data: { code: 0, data: { token: 'new-access', refresh_token: 'new-refresh' } },
        }
      }
      // 续期成功但业务请求仍然 401：不能继续续期
      return { status: 401, data: { code: 401, message: '无权限' } }
    })

    const client = createPlatformHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    await expect(client.get('/platform/orders')).rejects.toMatchObject({ status: 401 })
    expect(refreshCount).toBe(1)
    expect(fake.calls.redirect).toBe(1)
  })

  it('没有续期凭证时直接清态跳登录，不发续期请求', async () => {
    const fake = createSession({ refreshToken: null })
    let refreshCount = 0

    const adapter = adapterOf((config) => {
      if (config.url === REFRESH_URL) {
        refreshCount += 1
        return { status: 200, data: { code: 0, data: { token: 'x', refresh_token: 'y' } } }
      }
      return { status: 401, data: { code: 401, message: '访问凭证已过期' } }
    })

    const client = createPlatformHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    await expect(client.get('/platform/orders')).rejects.toBeInstanceOf(Error)
    expect(refreshCount).toBe(0)
    expect(fake.calls).toEqual({ clear: 1, redirect: 1 })
  })
})
