import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { onMounted, ref } from 'vue'

/**
 * 同意页之后（#199）：同意完成不等于可以进业务页。
 * - 账号仍带 must_change_password → 专用改密页（带原回跳）；
 * - 否则回业务页；
 * - 硬刷新后 store 无 user：用 /common/user/profile 校准后再决定，
 *   避免同一账号换设备登录时漏掉强制改密。
 */

const getConsentStatus = vi.fn()
const recordConsent = vi.fn()

vi.mock('~/api/consent', () => ({
  getConsentStatus: (...args: unknown[]) => getConsentStatus(...args),
  recordConsent: (...args: unknown[]) => recordConsent(...args),
}))

const adminLogout = vi.fn()

const authUser = ref<Record<string, unknown> | null>(null)
const fetchProfile = vi.fn()
const markConsented = vi.fn()
const navigateTo = vi.fn()

let Page: any
let ElementPlus: any

beforeAll(async () => {
  class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  vi.stubGlobal('ResizeObserver', ResizeObserverStub)
  vi.stubGlobal('ref', ref)
  vi.stubGlobal('onMounted', onMounted)
  vi.stubGlobal('definePageMeta', vi.fn())
  vi.stubGlobal('useAuthStore', () => ({
    get user() {
      return authUser.value
    },
    fetchProfile,
    markConsented,
  }))
  vi.stubGlobal('useRoute', () => ({ query: { redirect: '/billing' } }))
  vi.stubGlobal('navigateTo', navigateTo)
  vi.stubGlobal('useRuntimeConfig', () => ({ public: { siteUrl: 'https://example.test' } }))
  // 页面通过 Nuxt 自动导入拿到 useAdminLogout，测试环境需手工挂到全局。
  vi.stubGlobal('useAdminLogout', () => adminLogout)
  vi.stubGlobal('ElMessage', { error: vi.fn(), success: vi.fn(), warning: vi.fn() })
  ElementPlus = (await import('element-plus')).default
  Page = (await import('~/pages/consent.vue')).default
}, 60000)

beforeEach(() => {
  document.body.innerHTML = ''
  authUser.value = null
  fetchProfile.mockReset()
  markConsented.mockReset()
  navigateTo.mockReset()
  adminLogout.mockReset()
  getConsentStatus.mockReset().mockResolvedValue({ policy_version: 'v1', consented: false })
  recordConsent.mockReset().mockResolvedValue({ policy_version: 'v1' })
})

async function mountAndAgree() {
  const wrapper = mount(Page, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  await wrapper.find('input[type="checkbox"]').setValue(true)
  const button = wrapper.findAll('button').find((b) => b.text().trim() === '同意并继续')
  if (!button) throw new Error('未找到「同意并继续」按钮')
  await button.trigger('click')
  await flushPromises()
  return wrapper
}

describe('同意后去向', () => {
  it('must_change_password 时进入专用改密页并保留回跳', async () => {
    authUser.value = { must_change_password: true }

    await mountAndAgree()

    expect(markConsented).toHaveBeenCalledTimes(1)
    expect(navigateTo).toHaveBeenCalledWith({
      path: '/change-password',
      query: { redirect: '/billing' },
    })
  })

  it('无需改密时回业务页', async () => {
    authUser.value = { must_change_password: false }

    await mountAndAgree()

    expect(navigateTo).toHaveBeenCalledWith({ path: '/billing' })
  })

  it('硬刷新后 store 无 user：先读 profile 再决定去改密页', async () => {
    authUser.value = null
    fetchProfile.mockImplementation(async () => {
      authUser.value = { must_change_password: true }
    })
    getConsentStatus.mockResolvedValue({ policy_version: 'v1', consented: true })

    mount(Page, { global: { plugins: [ElementPlus] } })
    await flushPromises()

    expect(fetchProfile).toHaveBeenCalledTimes(1)
    expect(navigateTo).toHaveBeenCalledWith({
      path: '/change-password',
      query: { redirect: '/billing' },
    })
  })

  it('拒绝政策走中心化退出，不再只清本地', async () => {
    const wrapper = mount(Page, { global: { plugins: [ElementPlus] } })
    await flushPromises()

    const decline = wrapper.findAll('button').find((b) => b.text().trim() === '暂不同意')
    if (!decline) throw new Error('未找到「暂不同意」按钮')
    await decline.trigger('click')
    await flushPromises()

    expect(adminLogout).toHaveBeenCalledTimes(1)
  })
})
