import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { stubNuxtPageMeta, stubResizeObserver } from '../helpers/stubs'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive, ref } from 'vue'

/**
 * 登录页去向（#199）：
 * - 未同意：先同意页（即使同时未改密）；
 * - 已同意但 must_change_password：直接去专用改密页；
 * - 否则回业务页。
 * 旧的「首次登录」可取消弹窗必须消失——改密不再是可跳过的软提示。
 */

const loginPassword = vi.fn()
const changePassword = vi.fn()

vi.mock('~/api/auth', () => ({
  loginPassword: (...args: unknown[]) => loginPassword(...args),
  changePassword: (...args: unknown[]) => changePassword(...args),
}))

const setSession = vi.fn()
const navigateTo = vi.fn()
const messageBoxPrompt = vi.fn()

let Page: any
let ElementPlus: any

function loginResult(mustChange: boolean, consentRequired: boolean) {
  return {
    token: 'token',
    refresh_token: 'refresh',
    expires_at: '',
    consent_required: consentRequired,
    current_policy_version: 'v1',
    user: {
      id: 1,
      phone: '19900005001',
      nickname: '园长',
      gender: 1,
      must_change_password: mustChange,
      roles: [],
    },
  }
}

beforeAll(async () => {
  stubResizeObserver()
  vi.stubGlobal('ref', ref)
  vi.stubGlobal('reactive', reactive)
  stubNuxtPageMeta()
  vi.stubGlobal('useAuthStore', () => ({ user: null, setSession }))
  vi.stubGlobal('useRoute', () => ({ query: { redirect: '/billing' } }))
  vi.stubGlobal('navigateTo', navigateTo)
  vi.stubGlobal('ElMessage', { error: vi.fn(), success: vi.fn(), warning: vi.fn() })
  vi.stubGlobal('ElMessageBox', { prompt: messageBoxPrompt })
  ElementPlus = (await import('element-plus')).default
  Page = (await import('~/pages/login.vue')).default
}, 60000)

beforeEach(() => {
  document.body.innerHTML = ''
  loginPassword.mockReset()
  changePassword.mockReset()
  setSession.mockReset()
  navigateTo.mockReset()
  messageBoxPrompt.mockReset()
})

async function mountAndSubmit() {
  const wrapper = mount(Page, { global: { plugins: [ElementPlus] } })
  await wrapper.find('input[placeholder="手机号"]').setValue('19900005001')
  await wrapper.find('input[placeholder="密码"]').setValue('initial-pass')
  await wrapper.find('form').trigger('submit')
  await flushPromises()
  return wrapper
}

describe('登录后去向', () => {
  it('未同意时先走同意页，即使同时未改密', async () => {
    loginPassword.mockResolvedValue(loginResult(true, true))

    await mountAndSubmit()

    expect(navigateTo).toHaveBeenCalledWith({
      path: '/consent',
      query: { redirect: '/billing' },
    })
  })

  it('已同意但 must_change_password 时进入专用改密页', async () => {
    loginPassword.mockResolvedValue(loginResult(true, false))

    await mountAndSubmit()

    expect(navigateTo).toHaveBeenCalledWith({
      path: '/change-password',
      query: { redirect: '/billing' },
    })
    // 可取消的软提示已移除
    expect(messageBoxPrompt).not.toHaveBeenCalled()
  })

  it('无门槛时直接回业务页', async () => {
    loginPassword.mockResolvedValue(loginResult(false, false))

    await mountAndSubmit()

    expect(navigateTo).toHaveBeenCalledWith({ path: '/billing' })
    expect(messageBoxPrompt).not.toHaveBeenCalled()
  })
})
