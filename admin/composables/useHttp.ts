import axios, { type AxiosInstance, type AxiosRequestConfig } from 'axios'
import type { ApiEnvelope } from '~/types/api'
import {
  ADMIN_TOKEN_COOKIE,
  ADMIN_TOKEN_MAX_AGE,
  buildAuthCookieOptions,
} from '~/utils/authCookie'

let client: AxiosInstance | null = null

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    client = null
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
 * 管理后台 HTTP 客户端（单例），自动附加 Token、解析统一 data 包装
 */
export function useHttp() {
  const config = useRuntimeConfig()

  if (!client) {
    client = axios.create({
      baseURL: config.public.apiBase as string,
      timeout: 60000,
      headers: { 'Content-Type': 'application/json', terminal_from: 'admin' },
    })

    client.interceptors.request.use((req) => {
      const token = useCookie<string | null>(
        ADMIN_TOKEN_COOKIE,
        buildAuthCookieOptions({ maxAge: ADMIN_TOKEN_MAX_AGE }),
      )
      if (token.value) {
        req.headers.Authorization = `Bearer ${token.value}`
      }
      return req
    })

    client.interceptors.response.use(
      (res) => {
        if (res.config.responseType === 'blob' || res.config.responseType === 'arraybuffer') {
          return res
        }
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
          const token = useCookie<string | null>(ADMIN_TOKEN_COOKIE, buildAuthCookieOptions())
          const route = useRoute()
          if (route.path !== '/login') {
            token.value = null
            navigateTo('/login')
          }
        }
        // 未同意当前隐私政策：服务端兜底 403 + 40301，先完成同意再回原页。
        const bizCode = (data && typeof data === 'object' && 'code' in data)
          ? (data as { code?: number }).code
          : undefined
        if (import.meta.client && err.response?.status === 403 && bizCode === 40301) {
          const route = useRoute()
          if (route.path !== '/consent') {
            navigateTo({ path: '/consent', query: { redirect: route.fullPath } })
          }
        }
        // 保留 HTTP 状态码：调用方需要区分「无权限（403）」与真实故障
        const wrapped = new Error(String(msg)) as Error & {
          status?: number
          data?: unknown
          code?: number
        }
        wrapped.status = err.response?.status
        // 保留业务错误里的 data（如取消订单被拒时的 unrefunded_amount_cents），
        // 调用方据此展示结构化信息，而不是解析文案。
        wrapped.data =
          data && typeof data === 'object' && 'data' in data
            ? (data as { data?: unknown }).data
            : undefined
        wrapped.code = bizCode
        return Promise.reject(wrapped)
      },
    )
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
