import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { computed, onMounted, reactive, ref } from 'vue'

/**
 * 订单页行为验收（#118 / TC-NEW-ORD-0015、0017）：
 * - 列表展示「信用开通」标签，详情展示信用开通理由；
 * - 零收款确认必须先填写理由，理由随请求提交；取消填写则不提交；
 * - 按钮规则：paid/expired 隐藏确认，cancelled 隐藏全部，expired 未收款仍可取消。
 */

const listOrders = vi.fn()
const getOrder = vi.fn()
const confirmOrder = vi.fn()
const cancelOrder = vi.fn()
const createOrder = vi.fn()
const addPayment = vi.fn()

vi.mock('~/api/platform/orders', () => ({
  listOrders: (...args: unknown[]) => listOrders(...args),
  getOrder: (...args: unknown[]) => getOrder(...args),
  confirmOrder: (...args: unknown[]) => confirmOrder(...args),
  cancelOrder: (...args: unknown[]) => cancelOrder(...args),
  createOrder: (...args: unknown[]) => createOrder(...args),
  addPayment: (...args: unknown[]) => addPayment(...args),
}))

vi.mock('~/api/platform/kindergartens', () => ({
  listKindergartens: vi.fn().mockResolvedValue({ list: [], total: 0, page: 1, page_size: 100 }),
}))

vi.mock('~/api/platform/plans', () => ({
  listPlans: vi.fn().mockResolvedValue({ list: [], total: 0, page: 1, page_size: 100 }),
}))

vi.mock('~/api/platform/distributors', () => ({
  listDistributors: vi.fn().mockResolvedValue({ list: [], total: 0, page: 1, page_size: 100 }),
}))

const baseDetail = {
  id: 12,
  order_no: 'ORD20261106001',
  kindergarten_id: 7,
  kindergarten_name: '阳光幼儿园',
  status: 'unpaid',
  amount_due_cents: 200000,
  amount_paid_cents: 0,
  starts_at: '2026-11-06',
  expires_at: '2027-05-05',
  pay_due_at: '2026-11-06',
  credit_granted: false,
  credit_reason: '',
  plan_snapshot: { plan_name: '旗舰版', module_codes: ['growth'], price_cents: 200000 },
  distributor: null,
  invoice_info: null,
  payments: [],
}

const listRow = {
  id: 12,
  order_no: 'ORD20261106001',
  kindergarten_id: 7,
  kindergarten_name: '阳光幼儿园',
  plan_name: '旗舰版',
  student_tier: 1,
  amount_due_cents: 200000,
  amount_paid_cents: 0,
  status: 'unpaid',
  starts_at: '2026-11-06',
  expires_at: '2027-05-05',
  pay_due_at: '2026-11-06',
  credit_granted: false,
  distributor_name: '',
  created_at: '2026-11-06T10:00:00+08:00',
}

let Page: any
let ElementPlus: any
let MessageBox: any

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
  vi.stubGlobal('ElMessage', { error: vi.fn(), success: vi.fn(), warning: vi.fn() })
  MessageBox = {
    prompt: vi.fn(),
    alert: vi.fn().mockResolvedValue(undefined),
    confirm: vi.fn().mockResolvedValue('confirm'),
  }
  vi.stubGlobal('ElMessageBox', MessageBox)
  ElementPlus = (await import('element-plus')).default
  Page = (await import('~/pages/platform/orders/index.vue')).default
})

beforeEach(() => {
  document.body.innerHTML = ''
  listOrders.mockReset().mockResolvedValue({ list: [listRow], total: 1, page: 1, page_size: 20 })
  getOrder.mockReset().mockResolvedValue({ ...baseDetail })
  confirmOrder.mockReset().mockResolvedValue({})
  cancelOrder.mockReset()
  addPayment.mockReset()
  MessageBox.prompt.mockReset()
})

function mountPage() {
  return mount(Page, { global: { plugins: [ElementPlus] } })
}

function buttonLabels(wrapper: ReturnType<typeof mountPage>): string[] {
  const local = wrapper.findAll('button').map((b) => b.text().trim())
  const teleported = Array.from(document.querySelectorAll('button')).map((b) => (b.textContent || '').trim())
  return [...local, ...teleported]
}

async function openDetail(wrapper: ReturnType<typeof mountPage>, detail: Record<string, unknown>) {
  getOrder.mockResolvedValue({ ...baseDetail, ...detail })
  const detailButton = wrapper.findAll('button').find((b) => b.text().trim() === '详情')
  if (!detailButton) throw new Error('未找到「详情」按钮')
  await detailButton.trigger('click')
  await flushPromises()
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

describe('订单页面', () => {
  it('列表展示信用开通标签', async () => {
    listOrders.mockResolvedValue({
      list: [
        {
          id: 12,
          order_no: 'ORD20261106001',
          kindergarten_id: 7,
          kindergarten_name: '阳光幼儿园',
          plan_name: '旗舰版',
          student_tier: 1,
          amount_due_cents: 200000,
          amount_paid_cents: 0,
          status: 'unpaid',
          starts_at: '2026-11-06',
          expires_at: '2027-05-05',
          pay_due_at: '2026-11-06',
          credit_granted: true,
          distributor_name: '',
          created_at: '2026-11-06T10:00:00+08:00',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
    })
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.text()).toContain('信用开通')
  })

  it('详情展示信用开通理由', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await openDetail(wrapper, {
      status: 'partial',
      amount_paid_cents: 100000,
      credit_granted: true,
      credit_reason: '合同已签署，对公转账到账后补录',
    })
    const text = `${wrapper.text()}\n${document.body.textContent || ''}`
    expect(text).toContain('信用开通')
    expect(text).toContain('合同已签署，对公转账到账后补录')
  })

  it('零收款确认必须先填写理由，理由随请求提交', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await openDetail(wrapper, { status: 'unpaid', amount_paid_cents: 0 })
    MessageBox.prompt.mockResolvedValue({ value: '合同已签署，先开课' })

    clickButton(wrapper, '确认订单')
    await flushPromises()

    expect(MessageBox.prompt).toHaveBeenCalled()
    expect(confirmOrder).toHaveBeenCalledWith(12, '合同已签署，先开课')
  })

  it('取消填写理由时不提交确认', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await openDetail(wrapper, { status: 'unpaid', amount_paid_cents: 0 })
    MessageBox.prompt.mockRejectedValue(new Error('cancel'))

    clickButton(wrapper, '确认订单')
    await flushPromises()

    expect(confirmOrder).not.toHaveBeenCalled()
  })

  it('已有收款的确认不弹理由', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await openDetail(wrapper, { status: 'partial', amount_paid_cents: 100000 })

    clickButton(wrapper, '确认订单')
    await flushPromises()

    expect(MessageBox.prompt).not.toHaveBeenCalled()
    expect(confirmOrder).toHaveBeenCalledWith(12)
  })

  it('paid 隐藏确认，cancelled 隐藏全部，expired 未收款仍可取消', async () => {
    const paid = mountPage()
    await flushPromises()
    await openDetail(paid, { status: 'paid', amount_paid_cents: 200000 })
    let labels = buttonLabels(paid)
    expect(labels).not.toContain('确认订单')
    expect(labels).toContain('取消订单')

    document.body.innerHTML = ''
    const expired = mountPage()
    await flushPromises()
    await openDetail(expired, { status: 'expired', amount_paid_cents: 0 })
    labels = buttonLabels(expired)
    expect(labels).not.toContain('确认订单')
    expect(labels).toContain('取消订单')

    document.body.innerHTML = ''
    const cancelled = mountPage()
    await flushPromises()
    await openDetail(cancelled, { status: 'cancelled', amount_paid_cents: 0 })
    labels = buttonLabels(cancelled)
    expect(labels).not.toContain('确认订单')
    expect(labels).not.toContain('登记收款')
    expect(labels).not.toContain('取消订单')
  })
})
