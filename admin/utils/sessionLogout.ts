import type { AxiosInstance, AxiosRequestConfig } from 'axios'

/**
 * 会话登出的纯逻辑层（#199）。
 *
 * 园所端与平台端唯一共用的一份：先读 refresh → 必要时手动续期一次 →
 * 带 Authorization + refresh 调登出接口 → finally 清本地。
 *
 * 不能用拦截器的自动续期：登出时 access 往往已过期，拦截器续期会轮换 refresh
 * 并用新 access 重放，但请求 body 里仍是旧 refresh；旧 refresh 已被消费，
 * 服务端跳过吊销，新的 refresh 反而活满 7 天。
 */

export interface LogoutTokenPair {
  token: string
  refresh_token?: string
}

export interface LogoutTransport {
  /** 带双凭证调用登出接口；服务端 401 表示 access 不可用 */
  revoke(accessToken: string | null, refreshToken: string | null): Promise<void>
  /** 手动续期；服务端会轮换 refresh */
  refresh(refreshToken: string): Promise<LogoutTokenPair>
}

export interface LogoutSessionPort {
  getAccessToken(): string | null
  getRefreshToken(): string | null
  /** 只写两个凭证；不得重置 user / consent（别用 setSession） */
  saveTokens(tokens: LogoutTokenPair): void
  clearSession(): void
  redirectToLogin(): void | Promise<void>
}

export interface SessionLogoutOptions {
  transport: LogoutTransport
  session: LogoutSessionPort
  /** 网络/5xx 时的一次性提示；401 不告警 */
  notifyWarning?: (message: string) => void
  logError?: (error: unknown) => void
  warningMessage?: string
}

const DEFAULT_WARNING =
  '退出请求未能送达服务器，本机登录状态已清除。如为共享设备，请联系园所管理员确认会话已失效。'

export function statusOfError(error: unknown): number | undefined {
  if (!error || typeof error !== 'object') return undefined
  const direct = (error as { status?: unknown }).status
  if (typeof direct === 'number') return direct
  const response = (error as { response?: { status?: unknown } }).response
  return typeof response?.status === 'number' ? response.status : undefined
}

export function createSessionLogout(options: SessionLogoutOptions): () => Promise<void> {
  let inFlight: Promise<void> | null = null
  let finished = false

  async function run(): Promise<void> {
    const { transport, session } = options
    try {
      const refreshToken = session.getRefreshToken()
      const accessToken = session.getAccessToken()
      try {
        await transport.revoke(accessToken, refreshToken)
      } catch (err) {
        // 只有「access 过期/缺失」且手里还有 refresh 时才值得续期一次。
        if (statusOfError(err) !== 401 || !refreshToken) throw err
        const rotated = await transport.refresh(refreshToken)
        if (!rotated?.token) throw new Error('续期响应缺少访问凭证')
        session.saveTokens(rotated)
        await transport.revoke(rotated.token, rotated.refresh_token ?? refreshToken)
      }
    } catch (err) {
      // 401 意味着服务端已经没有任何可吊销的会话，不算失败。
      if (statusOfError(err) !== 401) {
        options.logError?.(err)
        options.notifyWarning?.(options.warningMessage ?? DEFAULT_WARNING)
      }
    } finally {
      finished = true
      inFlight = null
      session.clearSession()
      await session.redirectToLogin()
    }
  }

  return () => {
    if (finished) return Promise.resolve()
    if (!inFlight) inFlight = run()
    return inFlight
  }
}

export interface CreateLogoutTransportOptions {
  client: AxiosInstance
  logoutPath: string
  refreshPath: string
  /** 管理端与平台端的「跳过滤拦截器自动续期」标志并不相同 */
  skipRefreshFlag: 'skipAdminAuthRefresh' | 'skipPlatformAuthRefresh'
}

function skipRefreshConfig(
  flag: CreateLogoutTransportOptions['skipRefreshFlag'],
  accessToken?: string | null,
): AxiosRequestConfig {
  const config: AxiosRequestConfig =
    flag === 'skipAdminAuthRefresh'
      ? { skipAdminAuthRefresh: true }
      : { skipPlatformAuthRefresh: true }
  if (accessToken) config.headers = { Authorization: `Bearer ${accessToken}` }
  return config
}

/**
 * 用真实 axios 客户端构造 transport。
 *
 * 两个请求都必须带 skip 标志：登出第一跳 401 不能被拦截器静默续期，
 * 手动续期请求本身也不能再次进入续期分支。
 */
export function createLogoutTransport(
  options: CreateLogoutTransportOptions,
): LogoutTransport {
  const { client, skipRefreshFlag } = options
  return {
    async revoke(accessToken, refreshToken) {
      await client.post(
        options.logoutPath,
        refreshToken ? { refresh_token: refreshToken } : {},
        skipRefreshConfig(skipRefreshFlag, accessToken),
      )
    },
    async refresh(refreshToken) {
      const res = await client.post<LogoutTokenPair>(
        options.refreshPath,
        { refresh_token: refreshToken },
        skipRefreshConfig(skipRefreshFlag),
      )
      return res.data
    },
  }
}
