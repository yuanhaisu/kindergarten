import { beforeEach, describe, expect, it, vi } from 'vitest'

/**
 * 平台登出：必须把续期凭证交给服务端吊销，并在本地清掉它。
 * 用替身替换 Nuxt 的 useCookie 与 usePlatformHttp，断言外部可观察的请求与副作用。
 */

const mocks = vi.hoisted(() => ({
  post: vi.fn(),
}))

vi.mock('~/composables/usePlatformHttp', () => ({
  usePlatformHttp: () => ({ post: mocks.post }),
}))

function stubCookie(value: string | null) {
  const cookie = { value }
  const useCookie = vi.fn(() => cookie)
  vi.stubGlobal('useCookie', useCookie)
  return cookie
}

describe('platformLogout', () => {
  beforeEach(() => {
    vi.resetModules()
    mocks.post.mockReset()
  })

  it('把续期凭证交给服务端吊销，并清掉本地续期 Cookie', async () => {
    const cookie = stubCookie('old-refresh')
    mocks.post.mockResolvedValue({})
    const { platformLogout } = await import('../../api/platform/auth')

    await platformLogout()

    expect(mocks.post).toHaveBeenCalledWith('/platform/auth/logout', {
      refresh_token: 'old-refresh',
    })
    expect(cookie.value).toBeNull()
  })

  it('请求失败时仍然清掉本地续期 Cookie', async () => {
    const cookie = stubCookie('old-refresh')
    mocks.post.mockRejectedValue(new Error('网络错误'))
    const { platformLogout } = await import('../../api/platform/auth')

    await expect(platformLogout()).rejects.toThrow('网络错误')
    expect(cookie.value).toBeNull()
  })

  it('没有续期凭证时不发送空字段', async () => {
    stubCookie(null)
    mocks.post.mockResolvedValue({})
    const { platformLogout } = await import('../../api/platform/auth')

    await platformLogout()

    expect(mocks.post).toHaveBeenCalledWith('/platform/auth/logout', {})
  })
})
