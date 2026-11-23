import { usePlatformHttp } from '~/composables/usePlatformHttp'
import {
  PLATFORM_AUTH_LOGIN_PATH,
  PLATFORM_AUTH_REFRESH_PATH,
  type PlatformAuthTokens,
} from '~/api/platform/types'

export function platformLogin(username: string, password: string) {
  return usePlatformHttp().post<PlatformAuthTokens>(
    PLATFORM_AUTH_LOGIN_PATH,
    { username, password },
  )
}

/**
 * 用续期凭证换取新凭证（服务端会轮换续期凭证）。
 * 常规续期由 `usePlatformHttp` 拦截器自动完成；此函数供需要显式续期的调用方使用。
 */
export function platformRefresh(refreshToken: string) {
  return usePlatformHttp().post<PlatformAuthTokens>(
    PLATFORM_AUTH_REFRESH_PATH,
    { refresh_token: refreshToken },
    { skipPlatformAuthRefresh: true },
  )
}
