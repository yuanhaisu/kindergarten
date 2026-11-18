import axios, {
  type AxiosAdapter,
  type AxiosInstance,
  type AxiosRequestConfig,
  type AxiosResponse,
} from 'axios'
import type { ApiEnvelope } from '~/types/api'
import {
  PLATFORM_AUTH_LOGIN_PATH,
  PLATFORM_AUTH_REFRESH_PATH,
  type PlatformAuthTokens,
} from '~/api/platform/types'
import {
  PLATFORM_REFRESH_COOKIE,
  PLATFORM_REFRESH_MAX_AGE,
  PLATFORM_TOKEN_COOKIE,
  PLATFORM_TOKEN_MAX_AGE,
  buildAuthCookieOptions,
} from '~/utils/authCookie'

declare module 'axios' {
  interface AxiosRequestConfig {
    /** 该请求本身是登录/续期或已显式排除，401 时不进入续期 */
    skipPlatformAuthRefresh?: boolean
    /** 已用新凭证重放过一次，再次 401 不再续期 */
    platformAuthRetried?: boolean
  }
}

let platformClient: AxiosInstance | null = null

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    platformClient = null
  })
}

/** 平台会话的读写与跳转；拦截器只依赖这个接口，便于单测注入替身。 */
export interface PlatformSessionAdapter {
  getAccessToken(): string | null
  getRefreshToken(): string | null
  saveTokens(tokens: PlatformAuthTokens): void
  clearSession(): void
  redirectToLogin(): void
  onLoginPage(): boolean
}

export interface CreatePlatformHttpOptions {
  baseURL: string
  session: PlatformSessionAdapter
  /** 测试注入用；默认走 axios 自带 adapter */
  adapter?: AxiosAdapter
}

export interface PlatformHttp {
  get<T = unknown>(url: string, axiosConfig?: AxiosRequestConfig): Promise<T>
  post<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig): Promise<T>
  put<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig): Promise<T>
  patch<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig): Promise<T>
  del<T = unknown>(url: string, axiosConfig?: AxiosRequestConfig): Promise<T>
}

function unwrapEnvelope<T>(raw: unknown): T {
  if (raw && typeof raw === 'object' && 'code' in raw) {
    const body = raw as ApiEnvelope<T>
    if (body.code !== 0) {
      throw new Error(body.message || '请求失败')
    }
    return body.data as T
  }
  return raw as T
}

function wrapError(err: unknown): Error & { status?: number } {
  if (err instanceof Error && 'isAxiosError' in err) {
    const axiosErr = err as Error & {
      response?: AxiosResponse
    }
    const data = axiosErr.response?.data
    const msg =
      (data && typeof data === 'object' && 'message' in data && (data as { message?: string }).message) ||
      err.message ||
      '网络错误'
    const wrapped = new Error(String(msg)) as Error & { status?: number }
    wrapped.status = axiosErr.response?.status
    return wrapped
  }
  return err instanceof Error ? (err as Error & { status?: number }) : new Error(String(err))
}

function unauthorizedError(): Error & { status?: number } {
  const err = new Error('登录已过期，请重新登录') as Error & { status?: number }
  err.status = 401
  return err
}

function shouldAttemptRefresh(
  config: AxiosRequestConfig | undefined,
  session: PlatformSessionAdapter,
): boolean {
  if (!config) return false
  if (config.skipPlatformAuthRefresh || config.platformAuthRetried) return false
  const url = config.url ?? ''
  // 登录 401 是「密码错误」，续期 401 是「续期凭证失效」——都不能再触发续期。
  if (url === PLATFORM_AUTH_LOGIN_PATH || url === PLATFORM_AUTH_REFRESH_PATH) return false
  // 登录页上的请求不续期，避免 401 → 跳登录 → 再 401 的循环。
  return !session.onLoginPage()
}

/**
 * 创建平台方 axios 客户端。
 *
 * 401 时用续期凭证换取新凭证并重放原请求。并发 401 共享同一个续期 Promise
 * （单飞），续期失败只在单飞体内清一次登录态、跳一次登录页。
 */
export function createPlatformHttpClient(options: CreatePlatformHttpOptions): AxiosInstance {
  const { session } = options
  const client = axios.create({
    baseURL: options.baseURL,
    timeout: 60000,
    headers: { 'Content-Type': 'application/json' },
    ...(options.adapter ? { adapter: options.adapter } : {}),
  })

  let refreshPromise: Promise<void> | null = null

  async function performRefresh(): Promise<void> {
    try {
      const refreshToken = session.getRefreshToken()
      if (!refreshToken) throw unauthorizedError()
      const res = await client.post<PlatformAuthTokens>(
        PLATFORM_AUTH_REFRESH_PATH,
        { refresh_token: refreshToken },
        { skipPlatformAuthRefresh: true },
      )
      const tokens = res.data
      if (!tokens || typeof tokens !== 'object' || !tokens.token) {
        throw unauthorizedError()
      }
      session.saveTokens(tokens)
    } catch (err) {
      // 单飞体内只执行一次：并发等待者不会再各自清态/跳转。
      session.clearSession()
      session.redirectToLogin()
      throw err
    }
  }

  function refreshOnce(): Promise<void> {
    if (!refreshPromise) {
      refreshPromise = performRefresh().finally(() => {
        refreshPromise = null
      })
    }
    return refreshPromise
  }

  client.interceptors.request.use((req) => {
    if (req.skipPlatformAuthRefresh) return req
    const token = session.getAccessToken()
    if (token) {
      req.headers.Authorization = `Bearer ${token}`
    }
    return req
  })

  client.interceptors.response.use(
    (res) => {
      res.data = unwrapEnvelope(res.data)
      return res
    },
    async (err) => {
      const status = err.response?.status
      const config = err.config as AxiosRequestConfig | undefined

      if (status === 401 && shouldAttemptRefresh(config, session)) {
        try {
          await refreshOnce()
        } catch (refreshErr) {
          throw wrapError(refreshErr)
        }
        // 重放原请求；request 拦截器会用新凭证覆盖 Authorization。
        return client.request({ ...config, platformAuthRetried: true })
      }

      if (status === 401 && config?.platformAuthRetried) {
        // 续期成功但重放仍 401：会话已不可用，收口一次登录态。
        session.clearSession()
        session.redirectToLogin()
      }

      throw wrapError(err)
    },
  )

  return client
}

function createNuxtPlatformSession(): PlatformSessionAdapter {
  const accessCookie = () =>
    useCookie<string | null>(
      PLATFORM_TOKEN_COOKIE,
      buildAuthCookieOptions({ maxAge: PLATFORM_TOKEN_MAX_AGE }),
    )
  const refreshCookie = () =>
    useCookie<string | null>(
      PLATFORM_REFRESH_COOKIE,
      buildAuthCookieOptions({ maxAge: PLATFORM_REFRESH_MAX_AGE }),
    )

  return {
    getAccessToken: () => accessCookie().value ?? null,
    getRefreshToken: () => refreshCookie().value ?? null,
    saveTokens: (tokens) => {
      accessCookie().value = tokens.token
      if (tokens.refresh_token) refreshCookie().value = tokens.refresh_token
    },
    clearSession: () => {
      accessCookie().value = null
      refreshCookie().value = null
    },
    redirectToLogin: () => {
      const route = useRoute()
      if (route.path !== '/platform/login') {
        navigateTo('/platform/login')
      }
    },
    onLoginPage: () => useRoute().path === '/platform/login',
  }
}

function createHttpMethods(client: AxiosInstance): PlatformHttp {
  return {
    async get<T = unknown>(url: string, axiosConfig?: AxiosRequestConfig) {
      const res = await client.get<T>(url, axiosConfig)
      return res.data
    },
    async post<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig) {
      const res = await client.post<T>(url, body, axiosConfig)
      return res.data
    },
    async put<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig) {
      const res = await client.put<T>(url, body, axiosConfig)
      return res.data
    },
    async patch<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig) {
      const res = await client.patch<T>(url, body, axiosConfig)
      return res.data
    },
    async del<T = unknown>(url: string, axiosConfig?: AxiosRequestConfig) {
      const res = await client.delete<T>(url, axiosConfig)
      return res.data
    },
  }
}

/**
 * 平台方 HTTP 客户端。与园所后台的 `useHttp` 是两套独立登录态：
 * 凭证放在 `platform_token` / `platform_refresh`，访问凭证过期时单飞续期（ADR-0014）。
 */
export function usePlatformHttp(): PlatformHttp {
  const config = useRuntimeConfig()

  if (!platformClient) {
    platformClient = createPlatformHttpClient({
      baseURL: config.public.apiBase as string,
      session: createNuxtPlatformSession(),
    })
  }

  return createHttpMethods(platformClient)
}
