import { createNuxtAdminSession, useHttp } from '~/composables/useHttp'
import { createNuxtPlatformSession, usePlatformHttp } from '~/composables/usePlatformHttp'
import { useAuthStore } from '~/stores/auth'
import {
  ADMIN_AUTH_LOGOUT_PATH,
  ADMIN_AUTH_REFRESH_PATH,
  ADMIN_LOGIN_PAGE_PATH,
} from '~/utils/authPaths'
import {
  PLATFORM_AUTH_LOGOUT_PATH,
  PLATFORM_AUTH_REFRESH_PATH,
} from '~/api/platform/types'
import {
  createLogoutTransport,
  createSessionLogout,
} from '~/utils/sessionLogout'

/**
 * 中心化登出（#199）：园所端与平台端共用同一套「双凭证吊销 + 一次手动续期」逻辑。
 *
 * 为什么不能直接调用现有 http 封装：access 往往已过期，拦截器会自动续期并
 * 轮换 refresh，然后拿旧 body 重放——旧 refresh 已被消费，服务端跳过吊销，
 * 新的 refresh 反而活满 7 天。transport 因此强制带 skip 标志。
 */

const ADMIN_WARNING =
  '退出请求未能送达服务器，本机登录状态已清除。如为共享设备，请联系园所管理员确认会话已失效。'
const PLATFORM_WARNING =
  '退出请求未能送达服务器，本机登录状态已清除。如为共享设备，请联系平台管理员确认会话已失效。'

export function useAdminLogout() {
  const auth = useAuthStore()
  const session = createNuxtAdminSession()

  return createSessionLogout({
    transport: createLogoutTransport({
      client: useHttp().raw,
      logoutPath: ADMIN_AUTH_LOGOUT_PATH,
      refreshPath: ADMIN_AUTH_REFRESH_PATH,
      skipRefreshFlag: 'skipAdminAuthRefresh',
    }),
    session: {
      getAccessToken: session.getAccessToken,
      getRefreshToken: session.getRefreshToken,
      saveTokens: session.saveTokens,
      // 清 Pinia 同时清 Cookie 与权限缓存；不能用 setSession 反向覆盖。
      clearSession: () => auth.clearSession(),
      redirectToLogin: () => navigateTo(ADMIN_LOGIN_PAGE_PATH),
    },
    notifyWarning: (message) => ElMessage.warning(message),
    logError: (error) => console.error('[auth] admin logout failed', error),
    warningMessage: ADMIN_WARNING,
  })
}

export function usePlatformLogout() {
  const session = createNuxtPlatformSession()

  return createSessionLogout({
    transport: createLogoutTransport({
      client: usePlatformHttp().raw,
      logoutPath: PLATFORM_AUTH_LOGOUT_PATH,
      refreshPath: PLATFORM_AUTH_REFRESH_PATH,
      skipRefreshFlag: 'skipPlatformAuthRefresh',
    }),
    session: {
      getAccessToken: session.getAccessToken,
      getRefreshToken: session.getRefreshToken,
      saveTokens: session.saveTokens,
      clearSession: () => {
        session.clearSession()
        if (typeof localStorage !== 'undefined') localStorage.removeItem('platform_user_name')
      },
      redirectToLogin: () => navigateTo('/platform/login'),
    },
    notifyWarning: (message) => ElMessage.warning(message),
    logError: (error) => console.error('[auth] platform logout failed', error),
    warningMessage: PLATFORM_WARNING,
  })
}
