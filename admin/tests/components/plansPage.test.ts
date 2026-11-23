import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { computed, onMounted, reactive, ref } from 'vue'

/**
 * #202：套餐价格必须大于 0。
 * 服务端拒绝 0 元套餐（PLAN_PRICE_NOT_POSITIVE），管理端必须在提交前拦下，
 * 避免点「保存」只收到一个 422。
 */

const listPlans = vi.fn()
const listModules = vi.fn()
const createPlan = vi.fn()
const updatePlan = vi.fn()
const listPlanTiers = vi.fn()
const replacePlanTiers = vi.fn()

vi.mock('~/api/platform/plans', () => ({
  listPlans: (...args: unknown[]) => listPlans(...args),
  listModules: (...args: unknown[]) => listModules(...args),
  createPlan: (...args: unknown[]) => createPlan(...args),
  updatePlan: (...args: unknown[]) => updatePlan(...args),
  listPlanTiers: (...args: unknown[]) => listPlanTiers(...args),
  replacePlanTiers: (...args: unknown[]) => replacePlanTiers(...args),
}))

const planRow = {
  id: 3,
  code: 'flagship',
  name: '旗舰版',
  tier: 1,
  price_cents: 100000,
  term_months: 6,
  module_codes: ['base', 'growth'],
  status: 'active',
}

let Page: any
let ElementPlus: any
let message: { error: ReturnType<typeof vi.fn>; success: ReturnType<typeof vi.fn> }

beforeAll(async () => {
  class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  vi.stubGlobal('ResizeObserver', ResizeObserverStub)
  vi.stubGlobal('ref', ref)
  vi.stubGlobal('reactive', reactive)
  vi.stubGlobal('computed', computed)
  vi.stubGlobal('onMounted', onMounted)
  vi.stubGlobal('definePageMeta', vi.fn())
  message = { error: vi.fn(), success: vi.fn() }
  vi.stubGlobal('ElMessage', message)
  ElementPlus = (await import('element-plus')).default
  Page = (await import('~/pages/platform/plans/index.vue')).default
})

beforeEach(() => {
  document.body.innerHTML = ''
  listPlans.mockReset().mockResolvedValue({ list: [planRow], total: 1, page: 1, page_size: 20 })
  listModules.mockReset().mockResolvedValue({ list: [{ code: 'base', name: '基础' }] })
  createPlan.mockReset().mockResolvedValue({ id: 4 })
  updatePlan.mockReset().mockResolvedValue({})
  listPlanTiers.mockReset().mockResolvedValue({ list: [] })
  replacePlanTiers.mockReset().mockResolvedValue({})
  Object.values(message).forEach((fn) => fn.mockReset())
})

function mountPage() {
  return mount(Page, { global: { plugins: [ElementPlus] } })
}

function clickButton(wrapper: ReturnType<typeof mountPage>, label: string) {
  const local = wrapper.findAll('button').find((b) => b.text().trim() === label)
  if (local) {
    local.element.click()
    return
  }
  const teleported = Array.from(document.querySelectorAll('button')).find(
    (b) => (b.textContent || '').trim() === label,
  )
  if (!teleported) throw new Error(`未找到「${label}」按钮`)
  teleported.click()
}

function formInput(wrapper: ReturnType<typeof mountPage>, label: string) {
  const item = wrapper
    .findAllComponents({ name: 'ElFormItem' })
    .find((c) => c.props('label') === label)
  if (!item) throw new Error(`未找到「${label}」表单项`)
  return item
}

async function setPrice(wrapper: ReturnType<typeof mountPage>, yuan: number) {
  const input = formInput(wrapper, '价格（元）').findComponent({ name: 'ElInputNumber' })
  input.vm.$emit('update:modelValue', yuan)
  await flushPromises()
}

describe('套餐价格校验', () => {
  it('新建 0 元套餐不会提交，改成正数后才提交', async () => {
    const wrapper = mountPage()
    await flushPromises()
    clickButton(wrapper, '新建套餐')
    await flushPromises()

    const code = formInput(wrapper, '套餐编码').findComponent({ name: 'ElInput' })
    code.vm.$emit('update:modelValue', 'starter')
    const name = formInput(wrapper, '名称').findComponent({ name: 'ElInput' })
    name.vm.$emit('update:modelValue', '入门版')
    await setPrice(wrapper, 0)

    clickButton(wrapper, '保存')
    await flushPromises()
    expect(createPlan).not.toHaveBeenCalled()
    expect(message.error).toHaveBeenCalledWith('价格必须大于 0')

    await setPrice(wrapper, 100)
    clickButton(wrapper, '保存')
    await flushPromises()
    expect(createPlan).toHaveBeenCalledTimes(1)
    expect(createPlan.mock.calls[0][0]).toMatchObject({ price_cents: 10000 })
  })

  it('把已有套餐改成 0 元不会提交', async () => {
    const wrapper = mountPage()
    await flushPromises()
    clickButton(wrapper, '编辑')
    await flushPromises()

    await setPrice(wrapper, 0)

    clickButton(wrapper, '保存')
    await flushPromises()

    expect(updatePlan).not.toHaveBeenCalled()
    expect(message.error).toHaveBeenCalledWith('价格必须大于 0')
  })
})
