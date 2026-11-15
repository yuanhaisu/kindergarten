import axios, { type AxiosInstance, type AxiosRequestConfig } from 'axios'
import type { ApiEnvelope } from '~/types/api'
import {
  PLATFORM_TOKEN_COOKIE,
  PLATFORM_TOKEN_MAX_AGE,
  buildAuthCookieOptions,
} from '~/utils/authCookie'

let platformClient: AxiosInstance | null = null

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    platformClient = null
  })
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

/**
 * 平台方 HTTP 客户端。与园所后台的 `useHttp` 是两套独立登录态：
 * 凭证放在 `platform_token`，401 时回平台登录页（ADR-0014）。
 */
export function usePlatformHttp() {
  const config = useRuntimeConfig()

  if (!platformClient) {
    platformClient = axios.create({
      baseURL: config.public.apiBase as string,
      timeout: 60000,
      headers: { 'Content-Type': 'application/json' },
    })

    platformClient.interceptors.request.use((req) => {
      const token = useCookie<string | null>(
        PLATFORM_TOKEN_COOKIE,
        buildAuthCookieOptions({ maxAge: PLATFORM_TOKEN_MAX_AGE }),
      )
      if (token.value) {
        req.headers.Authorization = `Bearer ${token.value}`
      }
      return req
    })

    platformClient.interceptors.response.use(
      (res) => {
        res.data = unwrapEnvelope(res.data)
        return res
      },
      (err) => {
        const data = err.response?.data
        const msg =
          (data && typeof data === 'object' && 'message' in data && (data as { message?: string }).message) ||
          err.message ||
          '网络错误'
        if (import.meta.client && err.response?.status === 401) {
          const token = useCookie<string | null>(PLATFORM_TOKEN_COOKIE, buildAuthCookieOptions())
          const route = useRoute()
          if (route.path !== '/platform/login') {
            token.value = null
            navigateTo('/platform/login')
          }
        }
        const wrapped = new Error(String(msg)) as Error & { status?: number }
        wrapped.status = err.response?.status
        return Promise.reject(wrapped)
      },
    )
  }

  const http = {
    async get<T = unknown>(url: string, axiosConfig?: AxiosRequestConfig) {
      const res = await platformClient!.get<T>(url, axiosConfig)
      return res.data
    },
    async post<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig) {
      const res = await platformClient!.post<T>(url, body, axiosConfig)
      return res.data
    },
    async put<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig) {
      const res = await platformClient!.put<T>(url, body, axiosConfig)
      return res.data
    },
    async patch<T = unknown>(url: string, body?: unknown, axiosConfig?: AxiosRequestConfig) {
      const res = await platformClient!.patch<T>(url, body, axiosConfig)
      return res.data
    },
    async del<T = unknown>(url: string, axiosConfig?: AxiosRequestConfig) {
      const res = await platformClient!.delete<T>(url, axiosConfig)
      return res.data
    },
  }

  return http
}
