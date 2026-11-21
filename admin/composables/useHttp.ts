import axios, {
  type AxiosAdapter,
  type AxiosInstance,
  type AxiosRequestConfig,
  type AxiosResponse,
} from 'axios'
import type { ApiEnvelope } from '~/types/api'
import {
  ADMIN_AUTH_LOGIN_PATH,
  ADMIN_AUTH_REFRESH_PATH,
  ADMIN_CONSENT_PAGE_PATH,
  ADMIN_LOGIN_PAGE_PATH,
} from '~/utils/authPaths'
import {
  ADMIN_REFRESH_COOKIE,
  ADMIN_TOKEN_COOKIE,
  ADMIN_TOKEN_MAX_AGE,
  buildAuthCookieOptions,
} from '~/utils/authCookie'

declare module 'axios' {
  interface AxiosRequestConfig {
    /** 该请求本身是登录/续期或已显式排除，401 时不进入续期 */
    skipAdminAuthRefresh?: boolean
    /** 已用新凭证重放过一次，再次 401 不再续期 */
    adminAuthRetried?: boolean
  }
}

let client: AxiosInstance | null = null

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    // 续期 Promise 是客户端闭包内的状态，随单例一起丢弃，避免热更新后
    // 复用一个持有旧会话的续期结果。
    client = null
  })
}

/** 管理端登录与续期响应里续期拦截器关心的字段。 */
export interface AdminAuthTokens {
  token: string
  refresh_token?: string
}

/** 管理端会话的读写与跳转；拦截器只依赖这个接口，便于单测注入替身。 */
export interface AdminSessionAdapter {
  getAccessToken(): string | null
  getRefreshToken(): string | null
  saveTokens(tokens: AdminAuthTokens): void
  clearSession(): void
  redirectToLogin(): void
  onLoginPage(): boolean
  /** 未同意当前隐私政策（403 + 40301）时跳到同意页并带回跳地址 */
  redirectToConsent(): void
}

export interface CreateAdminHttpOptions {
  baseURL: string
  session: AdminSessionAdapter
  /** 测试注入用；默认走 axios 自带 adapter */
  adapter?: AxiosAdapter
}

type HttpError = Error & {
  status?: number
  data?: unknown
  code?: number
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

function unauthorizedError(): HttpError {
  const err = new Error('登录已过期，请重新登录') as HttpError
  err.status = 401
  return err
}

function wrapError(err: unknown): HttpError {
  if (err instanceof Error && 'isAxiosError' in err) {
    const axiosErr = err as Error & { response?: AxiosResponse }
    const data = axiosErr.response?.data
    const msg =
      (data && typeof data === 'object' && 'message' in data && (data as { message?: string }).message) ||
      err.message ||
      '网络错误'
    const wrapped = new Error(String(msg)) as HttpError
    // 保留 HTTP 状态码：调用方需要区分「无权限（403）」与真实故障
    wrapped.status = axiosErr.response?.status
    // 保留业务错误里的 data（如取消订单被拒时的 unrefunded_amount_cents），
    // 调用方据此展示结构化信息，而不是解析文案。
    wrapped.data =
      data && typeof data === 'object' && 'data' in data
        ? (data as { data?: unknown }).data
        : undefined
    const code =
      data && typeof data === 'object' && 'code' in data
        ? (data as { code?: unknown }).code
        : undefined
    if (typeof code === 'number') wrapped.code = code
    return wrapped
  }
  return err instanceof Error ? (err as HttpError) : (new Error(String(err)) as HttpError)
}

function shouldAttemptRefresh(
  config: AxiosRequestConfig | undefined,
  session: AdminSessionAdapter,
): boolean {
  if (!config) return false
  if (config.skipAdminAuthRefresh || config.adminAuthRetried) return false
  const url = config.url ?? ''
  // 登录 401 是「账号或密码错误」，续期 401 是「续期凭证失效」——都不能再触发续期。
  if (url === ADMIN_AUTH_LOGIN_PATH || url === ADMIN_AUTH_REFRESH_PATH) return false
  // 登录页上的请求不续期，避免 401 → 跳登录 → 再 401 的循环。
  return !session.onLoginPage()
}

/**
 * 创建管理端 axios 客户端。
 *
 * 401 时用续期凭证换取新凭证并重放原请求。并发 401 共享同一个续期 Promise
 * （单飞），续期失败只在单飞体内清一次登录态、跳一次登录页。
 */
export function createAdminHttpClient(options: CreateAdminHttpOptions): AxiosInstance {
  const { session } = options
  const instance = axios.create({
    baseURL: options.baseURL,
    timeout: 60000,
    headers: { 'Content-Type': 'application/json', terminal_from: 'admin' },
    ...(options.adapter ? { adapter: options.adapter } : {}),
  })

  let refreshPromise: Promise<void> | null = null

  async function performRefresh(): Promise<void> {
    try {
      const refreshToken = session.getRefreshToken()
      if (!refreshToken) throw unauthorizedError()
      const res = await instance.post<AdminAuthTokens>(
        ADMIN_AUTH_REFRESH_PATH,
        { refresh_token: refreshToken },
        { skipAdminAuthRefresh: true },
      )
      const tokens = res.data
      if (!tokens || typeof tokens !== 'object' || !tokens.token) {
        throw unauthorizedError()
      }
      // 成功必须同时写回 access 与 refresh：只写 access 会让下一次过期无法续期。
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

  instance.interceptors.request.use((req) => {
    if (req.skipAdminAuthRefresh) return req
    const token = session.getAccessToken()
    if (token) {
      req.headers.Authorization = `Bearer ${token}`
    }
    return req
  })

  instance.interceptors.response.use(
    (res) => {
      if (res.config.responseType === 'blob' || res.config.responseType === 'arraybuffer') {
        return res
      }
      res.data = unwrapEnvelope(res.data)
      return res
    },
    async (err) => {
      const status = err.response?.status
      const config = err.config as AxiosRequestConfig | undefined
      const data = err.response?.data

      if (status === 401 && shouldAttemptRefresh(config, session)) {
        try {
          await refreshOnce()
        } catch (refreshErr) {
          throw wrapError(refreshErr)
        }
        // 重放原请求；request 拦截器会用新凭证覆盖 Authorization。
        return instance.request({ ...config, adminAuthRetried: true })
      }

      if (status === 401 && config?.adminAuthRetried) {
        // 续期成功但重放仍 401：会话已不可用，收口一次登录态。
        session.clearSession()
        session.redirectToLogin()
      }

      const bizCode =
        data && typeof data === 'object' && 'code' in data
          ? (data as { code?: unknown }).code
          : undefined
      // 未同意当前隐私政策：服务端兜底 403 + 40301，先完成同意再回原页。
      if (status === 403 && bizCode === 40301) {
        session.redirectToConsent()
      }

      throw wrapError(err)
    },
  )

  return instance
}

function createNuxtAdminSession(): AdminSessionAdapter {
  const accessCookie = () =>
    useCookie<string | null>(
      ADMIN_TOKEN_COOKIE,
      buildAuthCookieOptions({ maxAge: ADMIN_TOKEN_MAX_AGE }),
    )
  // admin_refresh 是会话级 Cookie（不传 maxAge）
  const refreshCookie = () =>
    useCookie<string | null>(ADMIN_REFRESH_COOKIE, buildAuthCookieOptions())

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
      if (route.path !== ADMIN_LOGIN_PAGE_PATH) {
        navigateTo(ADMIN_LOGIN_PAGE_PATH)
      }
    },
    onLoginPage: () => useRoute().path === ADMIN_LOGIN_PAGE_PATH,
    redirectToConsent: () => {
      const route = useRoute()
      if (route.path !== ADMIN_CONSENT_PAGE_PATH) {
        navigateTo({ path: ADMIN_CONSENT_PAGE_PATH, query: { redirect: route.fullPath } })
      }
    },
  }
}

/**
 * 管理后台 HTTP 客户端（单例），自动附加 Token、解析统一 data 包装，
 * 访问凭证过期时单飞续期并重放原请求（#166）。
 */
export function useHttp() {
  const config = useRuntimeConfig()

  if (!client) {
    client = createAdminHttpClient({
      baseURL: config.public.apiBase as string,
      session: createNuxtAdminSession(),
    })
  }

  const http = {
    raw: client,
    async get<T = unknown>(url: string, axiosConfig?: AxiosRequestConfig) {
      const res = await client!.get<T>(url, axiosConfig)
      return res.data as T
    },
    async post<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig) {
      const res = await client!.post<T>(url, body, axiosConfig)
      return res.data as T
    },
    async put<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig) {
      const res = await client!.put<T>(url, body, axiosConfig)
      return res.data as T
    },
    async delete<T = unknown>(url: string, axiosConfig?: AxiosRequestConfig) {
      const res = await client!.delete<T>(url, axiosConfig)
      return res.data as T
    },
    /** 文件流（不走 JSON 解包逻辑时直接用 raw） */
    async getBlob(url: string, axiosConfig?: AxiosRequestConfig) {
      const res = await client!.get(url, {
        ...axiosConfig,
        responseType: 'blob',
        // 避免对 GET 二进制接口带上 application/json，减少预检/网关异常
        transformRequest: [
          (data, headers) => {
            delete headers['Content-Type']
            return data
          },
        ],
      })
      return res
    },
  }

  return http
}
