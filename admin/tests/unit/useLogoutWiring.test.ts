import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

/**
 * 中心化退出的 Nuxt 接线（#199）：
 * - 园所端：带 Authorization + refresh 调 /auth/logout，最后清 Pinia/Cookie 并跳登录；
 * - 平台端同构；access 过期时手动续期，第二次登出必须带轮换后的 R_new；
 * - 平台本地显示名一并清理。
 */

const mocks = vi.hoisted(() => ({
  adminPost: vi.fn(),
  platformPost: vi.fn(),
  adminSession: {
    getAccessToken: vi.fn(),
    getRefreshToken: vi.fn(),
    saveTokens: vi.fn(),
    clearSession: vi.fn(),
  },
  platformSession: {
    getAccessToken: vi.fn(),
    getRefreshToken: vi.fn(),
    saveTokens: vi.fn(),
    clearSession: vi.fn(),
  },
  authClearSession: vi.fn(),
  navigateTo: vi.fn(),
  warn: vi.fn(),
}))

vi.mock('~/composables/useHttp', () => ({
  useHttp: () => ({ raw: { post: mocks.adminPost } }),
  createNuxtAdminSession: () => mocks.adminSession,
}))

vi.mock('~/composables/usePlatformHttp', () => ({
  usePlatformHttp: () => ({ raw: { post: mocks.platformPost } }),
  createNuxtPlatformSession: () => mocks.platformSession,
}))

vi.mock('~/stores/auth', () => ({
  useAuthStore: () => ({ clearSession: mocks.authClearSession }),
}))

function httpError(status: number): Error & { status: number } {
  return Object.assign(new Error(`HTTP ${status}`), { status })
}

beforeAll(() => {
  vi.stubGlobal('navigateTo', mocks.navigateTo)
  vi.stubGlobal('ElMessage', { warning: mocks.warn, error: vi.fn(), success: vi.fn() })
})

beforeEach(() => {
  document.body.innerHTML = ''
  mocks.adminPost.mockReset()
  mocks.platformPost.mockReset()
  mocks.authClearSession.mockReset()
  mocks.navigateTo.mockReset()
  mocks.warn.mockReset()
  mocks.adminSession.getAccessToken.mockReset().mockReturnValue('old-access')
  mocks.adminSession.getRefreshToken.mockReset().mockReturnValue('old-refresh')
  mocks.adminSession.saveTokens.mockReset()
  mocks.platformSession.getAccessToken.mockReset().mockReturnValue('expired-access')
  mocks.platformSession.getRefreshToken.mockReset().mockReturnValue('old-refresh')
  mocks.platformSession.saveTokens.mockReset()
  mocks.platformSession.clearSession.mockReset()
  localStorage.clear()
})

describe('useAdminLogout', () => {
  it('带双凭证调用 /auth/logout，finally 清态并跳登录', async () => {
    mocks.adminPost.mockResolvedValue({ data: {} })
    const { useAdminLogout } = await import('~/composables/useLogout')

    await useAdminLogout()()

    expect(mocks.adminPost).toHaveBeenCalledWith(
      '/auth/logout',
      { refresh_token: 'old-refresh' },
      expect.objectContaining({
        skipAdminAuthRefresh: true,
        headers: { Authorization: 'Bearer old-access' },
      }),
    )
    expect(mocks.authClearSession).toHaveBeenCalledTimes(1)
    expect(mocks.navigateTo).toHaveBeenCalledWith('/login')
    expect(mocks.warn).not.toHaveBeenCalled()
  })
})

describe('usePlatformLogout', () => {
  it('access 过期：手动续期一次，再用轮换后的 R_new 登出；清本地显示名', async () => {
    localStorage.setItem('platform_user_name', '平台运营')
    mocks.platformPost.mockImplementation(async (url: string, body: unknown, config: any) => {
      if (url === '/platform/auth/refresh') {
        return { data: { token: 'rotated-access', refresh_token: 'rotated-refresh' } }
      }
      if (url === '/platform/auth/logout') {
        const auth = config?.headers?.Authorization
        if (auth === 'Bearer expired-access') throw httpError(401)
        return { data: {} }
      }
      throw new Error(`unexpected url ${url}`)
    })
    const { usePlatformLogout } = await import('~/composables/useLogout')

    await usePlatformLogout()()

    const logoutCalls = mocks.platformPost.mock.calls.filter(
      (call) => call[0] === '/platform/auth/logout',
    )
    expect(logoutCalls).toHaveLength(2)
    expect(logoutCalls[0][1]).toEqual({ refresh_token: 'old-refresh' })
    expect(logoutCalls[0][2]).toMatchObject({
      skipPlatformAuthRefresh: true,
      headers: { Authorization: 'Bearer expired-access' },
    })
    expect(logoutCalls[1][1]).toEqual({ refresh_token: 'rotated-refresh' })
    expect(logoutCalls[1][2]).toMatchObject({
      skipPlatformAuthRefresh: true,
      headers: { Authorization: 'Bearer rotated-access' },
    })
    expect(mocks.platformSession.saveTokens).toHaveBeenCalledWith({
      token: 'rotated-access',
      refresh_token: 'rotated-refresh',
    })
    expect(mocks.platformSession.clearSession).toHaveBeenCalledTimes(1)
    expect(localStorage.getItem('platform_user_name')).toBeNull()
    expect(mocks.navigateTo).toHaveBeenCalledWith('/platform/login')
    expect(mocks.warn).not.toHaveBeenCalled()
  })
})
