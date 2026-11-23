import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { stubNuxtPageMeta, stubResizeObserver } from '../helpers/stubs'
import { flushPromises, mount } from '@vue/test-utils'
import { computed, onMounted, ref } from 'vue'

/**
 * 佣金台账页的行为验收（#117）：
 * - 分销商筛选与月份、状态可组合，筛选变化回到第 1 页；
 * - 点击台账行打开明细抽屉，展示原始佣金与冲回行；
 * - 汇总卡片与筛选后的明细来自同一次请求。
 *
 * 页面通过 Nuxt 自动导入使用 `ref`/`computed`/`onMounted`/`ElMessage`/`definePageMeta`，
 * 这里把它们挂到全局后再动态引入页面组件。
 */

const listCommissions = vi.fn()
const getCommission = vi.fn()
const listDistributors = vi.fn()

vi.mock('~/api/platform/commissions', () => ({
  listCommissions: (...args: unknown[]) => listCommissions(...args),
  getCommission: (...args: unknown[]) => getCommission(...args),
}))

vi.mock('~/api/platform/distributors', () => ({
  listDistributors: (...args: unknown[]) => listDistributors(...args),
}))

const originRow = {
  id: 41,
  order_no: 'ORD20261106001',
  kindergarten_name: '阳光幼儿园',
  distributor_name: '华东渠道',
  payment_id: 21,
  base_amount_cents: 100000,
  rate: 10,
  amount_cents: 10000,
  reversed_amount_cents: 0,
  net_amount_cents: 10000,
  status: 'settleable',
  received_at: '2026-11-06T14:00:00+08:00',
  eligible_at: '2026-12-06T00:00:00+08:00',
  reversal_of: null,
}

const listResponse = {
  list: [originRow],
  total: 1,
  page: 1,
  page_size: 20,
  summary: { pending_cents: 0, settleable_cents: 10000, settled_cents: 0, reversed_cents: 0 },
}

const detailResponse = {
  ...originRow,
  reversed_amount_cents: 4000,
  net_amount_cents: 6000,
  reversals: [
    {
      id: 52,
      payment_id: 25,
      base_amount_cents: 40000,
      amount_cents: -4000,
      status: 'reversed',
      created_at: '2026-11-20T10:00:00+08:00',
    },
  ],
}

const distributorRows = [
  { id: 3, name: '华东渠道', contact: '李经理', phone: '13900001234', commission_rate: 10, status: 1, created_at: '' },
]

let Page: any
let ElementPlus: any

beforeAll(async () => {
  stubResizeObserver()
  vi.stubGlobal('ref', ref)
  vi.stubGlobal('computed', computed)
  vi.stubGlobal('onMounted', onMounted)
  stubNuxtPageMeta()
  vi.stubGlobal('ElMessage', { error: vi.fn(), success: vi.fn() })
  ElementPlus = (await import('element-plus')).default
  Page = (await import('~/pages/platform/commissions/index.vue')).default
})

beforeEach(() => {
  listCommissions.mockReset().mockResolvedValue(listResponse)
  getCommission.mockReset().mockResolvedValue(detailResponse)
  listDistributors.mockReset().mockResolvedValue({ list: distributorRows, total: 1, page: 1, page_size: 20 })
})

function mountPage() {
  return mount(Page, { global: { plugins: [ElementPlus] } })
}

function findSelect(wrapper: ReturnType<typeof mountPage>, placeholder: string) {
  const select = wrapper
    .findAllComponents({ name: 'ElSelect' })
    .find((c) => c.props('placeholder') === placeholder)
  if (!select) throw new Error(`未找到占位符为「${placeholder}」的下拉框`)
  return select
}

async function setMonth(wrapper: ReturnType<typeof mountPage>, month: string) {
  const picker = wrapper.findComponent({ name: 'ElDatePicker' })
  picker.vm.$emit('update:modelValue', month)
  picker.vm.$emit('change', month)
  await flushPromises()
}

async function setSelect(wrapper: ReturnType<typeof mountPage>, placeholder: string, value: string | number) {
  const select = findSelect(wrapper, placeholder)
  select.vm.$emit('update:modelValue', value)
  select.vm.$emit('change', value)
  await flushPromises()
}

describe('佣金台账页', () => {
  it('分销商筛选与月份、状态组合，且筛选变化回到第 1 页', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const pagination = wrapper.findComponent({ name: 'ElPagination' })
    pagination.vm.$emit('update:current-page', 3)
    pagination.vm.$emit('current-change', 3)
    await flushPromises()
    expect(listCommissions).toHaveBeenLastCalledWith(expect.objectContaining({ page: 3 }))

    await setMonth(wrapper, '2026-11')
    await setSelect(wrapper, '全部状态', 'settleable')
    await setSelect(wrapper, '全部分销商', 3)

    expect(listCommissions).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 1, month: '2026-11', status: 'settleable', distributor_id: 3 }),
    )
  })

  it('点击台账行打开明细抽屉，展示原始佣金与冲回行', async () => {
    const wrapper = mountPage()
    await flushPromises()

    wrapper.findAllComponents({ name: 'ElTable' })[0].vm.$emit('row-click', originRow)
    await flushPromises()

    expect(getCommission).toHaveBeenCalledWith(41)
    const text = `${wrapper.text()}\n${document.body.textContent || ''}`
    expect(text).toContain('ORD20261106001')
    expect(text).toContain('华东渠道')
    expect(text).toContain('¥100.00')
    expect(text).toContain('#25')
    expect(text).toContain('-¥40.00')
    expect(text).toContain('¥60.00')
  })

  it('汇总卡片使用与筛选后明细同一次请求的 summary', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.text()).toContain('¥100.00')

    listCommissions.mockResolvedValueOnce({
      ...listResponse,
      summary: { pending_cents: 0, settleable_cents: 4200, settled_cents: 0, reversed_cents: 40000 },
    })
    await setSelect(wrapper, '全部状态', 'settleable')

    expect(wrapper.text()).toContain('¥42.00')
    expect(wrapper.text()).toContain('¥400.00')
  })
})
