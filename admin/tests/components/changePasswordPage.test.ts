import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { stubNuxtPageMeta, stubResizeObserver } from '../helpers/stubs'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive, ref } from 'vue'

/**
 * 专用改密页（#199）：
 * - 原密码 + 新密码 + 确认，服务端错误必须展示，不能再被空 catch 吞掉；
 * - 成功后清掉 must_change_password 并回原页；
 * - 本地能先拦下明显弱口令与两次不一致，避免白跑服务端；
 * - 回跳只允许站内路径。
 */

const changePassword = vi.fn()
const navigateTo = vi.fn()
const adminLogout = vi.fn()

vi.mock('~/api/auth', () => ({
  changePassword: (...args: unknown[]) => changePassword(...args),
}))

const authUser = ref<Record<string, unknown> | null>({ must_change_password: true })
const routeQuery = ref<Record<string, unknown>>({ redirect: '/billing' })

let Page: any
let ElementPlus: any

beforeAll(async () => {
  stubResizeObserver()
  vi.stubGlobal('ref', ref)
  vi.stubGlobal('reactive', reactive)
  stubNuxtPageMeta()
  vi.stubGlobal(
    'useAuthStore',
    () => ({
      get user() {
        return authUser.value
      },
    }),
  )
  vi.stubGlobal('useRoute', () => ({ query: routeQuery.value }))
  vi.stubGlobal('navigateTo', navigateTo)
  // 页面通过 Nuxt 自动导入拿到 useAdminLogout，测试环境需手工挂到全局。
  vi.stubGlobal('useAdminLogout', () => adminLogout)
  vi.stubGlobal('ElMessage', { error: vi.fn(), success: vi.fn(), warning: vi.fn() })
  ElementPlus = (await import('element-plus')).default
  Page = (await import('~/pages/change-password.vue')).default
}, 60000)

beforeEach(() => {
  document.body.innerHTML = ''
  changePassword.mockReset()
  navigateTo.mockReset()
  adminLogout.mockReset()
  authUser.value = { must_change_password: true }
  routeQuery.value = { redirect: '/billing' }
})

async function mountAndSubmit(
  values: { old?: string; next?: string; confirm?: string } = {},
) {
  const wrapper = mount(Page, { global: { plugins: [ElementPlus] } })
  await wrapper
    .find('input[placeholder="请输入当前密码"]')
    .setValue(values.old ?? 'initial-pass')
  await wrapper
    .find('input[placeholder="至少 8 位，含大写、小写与数字"]')
    .setValue(values.next ?? 'NewPass123')
  await wrapper
    .find('input[placeholder="再次输入新密码"]')
    .setValue(values.confirm ?? values.next ?? 'NewPass123')
  await wrapper.find('form').trigger('submit')
  await flushPromises()
  return wrapper
}

describe('专用改密页', () => {
  it('提交成功后清标记并回原页', async () => {
    changePassword.mockResolvedValue({ must_change_password: false })

    await mountAndSubmit()

    expect(changePassword).toHaveBeenCalledWith({
      old_password: 'initial-pass',
      new_password: 'NewPass123',
    })
    expect(authUser.value?.must_change_password).toBe(false)
    expect(navigateTo).toHaveBeenCalledWith({ path: '/billing' })
  })

  it('服务端错误（原密码不正确）展示在页面上，不静默吞掉', async () => {
    changePassword.mockRejectedValue(
      Object.assign(new Error('原密码不正确'), { status: 422 }),
    )

    const wrapper = await mountAndSubmit()

    expect(wrapper.text()).toContain('原密码不正确')
    expect(navigateTo).not.toHaveBeenCalled()
    expect(authUser.value?.must_change_password).toBe(true)
  })

  it('两次新密码不一致时本地拦截，不发请求', async () => {
    const wrapper = await mountAndSubmit({ next: 'NewPass123', confirm: 'NewPass124' })

    expect(changePassword).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('两次输入的新密码不一致')
  })

  it('明显弱口令本地拦截，不发请求', async () => {
    const wrapper = await mountAndSubmit({ next: 'abcdefgh', confirm: 'abcdefgh' })

    expect(changePassword).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('新密码须包含大写字母、小写字母和数字')
  })

  it('回跳地址为外链时回首页', async () => {
    routeQuery.value = { redirect: 'https://evil.example.com' }
    changePassword.mockResolvedValue({ must_change_password: false })

    await mountAndSubmit()

    expect(navigateTo).toHaveBeenCalledWith({ path: '/' })
  })

  it('「退出登录」走中心化登出', async () => {
    const wrapper = mount(Page, { global: { plugins: [ElementPlus] } })
    await flushPromises()

    const button = wrapper.findAll('button').find((b) => b.text().trim() === '退出登录')
    if (!button) throw new Error('未找到「退出登录」按钮')
    await button.trigger('click')

    expect(adminLogout).toHaveBeenCalledTimes(1)
  })
})
