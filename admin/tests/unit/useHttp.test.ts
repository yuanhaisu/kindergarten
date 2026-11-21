import {
  AxiosError,
  type AxiosAdapter,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios'
import { describe, expect, it } from 'vitest'
import {
  createAdminHttpClient,
  type AdminAuthTokens,
  type AdminSessionAdapter,
} from '../../composables/useHttp'

/**
 * 管理端单飞续期：拦截器是唯一观察点。
 * 用自定义 axios adapter 驱动真实拦截器链路，注入会话替身，
 * 断言「续期次数、重放结果、两个凭证的写入与跳转」这些外部可观察行为。
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

function createSession(
  initial: { accessToken?: string | null; refreshToken?: string | null; onLoginPage?: boolean } = {},
) {
  let accessToken = initial.accessToken === undefined ? 'old-access' : initial.accessToken
  let refreshToken = initial.refreshToken === undefined ? 'old-refresh' : initial.refreshToken
  const saved: AdminAuthTokens[] = []
  const calls = { clear: 0, redirect: 0, consent: 0 }

  const session: AdminSessionAdapter = {
    getAccessToken: () => accessToken,
    getRefreshToken: () => refreshToken,
    saveTokens: (tokens) => {
      saved.push(tokens)
      accessToken = tokens.token
      if (tokens.refresh_token) refreshToken = tokens.refresh_token
    },
    clearSession: () => {
      calls.clear += 1
      accessToken = null
      refreshToken = null
    },
    redirectToLogin: () => {
      calls.redirect += 1
    },
    onLoginPage: () => Boolean(initial.onLoginPage),
    redirectToConsent: () => {
      calls.consent += 1
    },
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

const REFRESH_URL = '/auth/refresh'
const LOGIN_URL = '/auth/login/password'

describe('useHttp 管理端单飞续期', () => {
  it('并发 401 只续期一次，成功后写入两个凭证并重放原请求', async () => {
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

    const client = createAdminHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    const [students, classes] = await Promise.all([
      client.get('/admin/students'),
      client.get('/admin/classes'),
    ])

    expect(refreshBodies).toHaveLength(1)
    expect(refreshBodies[0]).toEqual({ refresh_token: 'old-refresh' })
    // 续期请求不带过期的访问凭证
    expect(refreshAuthHeaders[0]).toBeUndefined()
    expect(fake.accessToken).toBe('new-access')
    expect(fake.refreshToken).toBe('new-refresh')
    // 成功时必须同时写回 access 与 refresh，缺一个都会让下一次过期无法续期
    expect(fake.saved).toEqual([
      expect.objectContaining({ token: 'new-access', refresh_token: 'new-refresh' }),
    ])
    expect(fake.calls).toEqual({ clear: 0, redirect: 0, consent: 0 })
    expect(students.data).toEqual({ path: '/admin/students' })
    expect(classes.data).toEqual({ path: '/admin/classes' })
  })

  it('登录页上的 401 不续期、不清态、不跳转', async () => {
    const fake = createSession({ onLoginPage: true })
    let refreshCount = 0

    const adapter = adapterOf((config) => {
      if (config.url === REFRESH_URL) {
        refreshCount += 1
        return { status: 200, data: { code: 0, data: { token: 'x', refresh_token: 'y' } } }
      }
      return { status: 401, data: { code: 401, message: '登录已过期' } }
    })

    const client = createAdminHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    await expect(client.get('/admin/dashboard/summary')).rejects.toMatchObject({ status: 401 })
    expect(refreshCount).toBe(0)
    expect(fake.calls).toEqual({ clear: 0, redirect: 0, consent: 0 })
  })

  it('登录接口 401（密码错误）不触发续期', async () => {
    const fake = createSession()
    let refreshCount = 0

    const adapter = adapterOf((config) => {
      if (config.url === REFRESH_URL) {
        refreshCount += 1
        return { status: 200, data: { code: 0, data: { token: 'x', refresh_token: 'y' } } }
      }
      return { status: 401, data: { code: 401, message: '账号或密码错误' } }
    })

    const client = createAdminHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    await expect(client.post(LOGIN_URL, { phone: '1', password: 'wrong' })).rejects.toMatchObject({
      message: '账号或密码错误',
      status: 401,
    })
    expect(refreshCount).toBe(0)
    expect(fake.calls).toEqual({ clear: 0, redirect: 0, consent: 0 })
  })

  it('续期接口自身 401 不再次续期', async () => {
    const fake = createSession()
    let refreshCount = 0

    const adapter = adapterOf((config) => {
      if (config.url === REFRESH_URL) {
        refreshCount += 1
        return { status: 401, data: { code: 401, message: '续期凭证已失效' } }
      }
      return { status: 200, data: { code: 0, data: { ok: true } } }
    })

    const client = createAdminHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    await expect(client.post(REFRESH_URL, { refresh_token: 'old-refresh' })).rejects.toMatchObject({
      status: 401,
    })
    expect(refreshCount).toBe(1)
    expect(fake.calls).toEqual({ clear: 0, redirect: 0, consent: 0 })
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

    const client = createAdminHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    const results = await Promise.allSettled([
      client.get('/admin/students'),
      client.get('/admin/classes'),
      client.get('/admin/teachers'),
    ])

    expect(results.map((r) => r.status)).toEqual(['rejected', 'rejected', 'rejected'])
    expect(refreshCount).toBe(1)
    expect(fake.calls).toEqual({ clear: 1, redirect: 1, consent: 0 })
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

    const client = createAdminHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    await expect(client.get('/admin/students')).rejects.toMatchObject({ status: 401 })
    expect(refreshCount).toBe(1)
    expect(fake.calls).toEqual({ clear: 1, redirect: 1, consent: 0 })
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

    const client = createAdminHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    await expect(client.get('/admin/students')).rejects.toBeInstanceOf(Error)
    expect(refreshCount).toBe(0)
    expect(fake.calls).toEqual({ clear: 1, redirect: 1, consent: 0 })
  })

  it('403 + 40301 走隐私同意页，不触发续期', async () => {
    const fake = createSession()
    let refreshCount = 0

    const adapter = adapterOf((config) => {
      if (config.url === REFRESH_URL) {
        refreshCount += 1
        return { status: 200, data: { code: 0, data: { token: 'x', refresh_token: 'y' } } }
      }
      return { status: 403, data: { code: 40301, message: '请先同意隐私政策' } }
    })

    const client = createAdminHttpClient({
      baseURL: '/api/v1',
      session: fake.session,
      adapter,
    })

    await expect(client.get('/admin/students')).rejects.toMatchObject({
      message: '请先同意隐私政策',
      status: 403,
      code: 40301,
    })
    expect(refreshCount).toBe(0)
    expect(fake.calls).toEqual({ clear: 0, redirect: 0, consent: 1 })
  })
})
