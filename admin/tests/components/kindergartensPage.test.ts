import { beforeAll, beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { stubNuxtPageMeta, stubResizeObserver } from '../helpers/stubs'
import { flushPromises, mount } from '@vue/test-utils'
import { computed, onMounted, reactive, ref } from 'vue'

/**
 * 平台园所页的家长入口码交付（#204）：
 * - 开通成功后的弹窗必须持久（不随 toast 消失），展示园所 ID + 10 位入口码，
 *   支持复制与跳转详情，且不回显初始口令；
 * - 剪贴板不可用（HTTP staging）时降级为可手动选中的只读输入框并给出提示；
 * - 详情抽屉可事后找回入口码，空码显示「未生成」且复制禁用；
 * - 列表不加入口码列；
 * - 弹窗/抽屉在移动宽度下不溢出。
 */

const listKindergartens = vi.fn()
const getKindergarten = vi.fn()
const onboardKindergarten = vi.fn()
const setKindergartenStatus = vi.fn()

vi.mock('~/api/platform/kindergartens', () => ({
  listKindergartens: (...args: unknown[]) => listKindergartens(...args),
  getKindergarten: (...args: unknown[]) => getKindergarten(...args),
  onboardKindergarten: (...args: unknown[]) => onboardKindergarten(...args),
  setKindergartenStatus: (...args: unknown[]) => setKindergartenStatus(...args),
}))

const listRow = {
  id: 7,
  name: '阳光幼儿园',
  principal: '王园长',
  status: 'active',
  student_count: 180,
  active_plan_name: '—',
  soonest_expire_at: null,
  created_at: '2026-11-06T10:00:00+08:00',
  tier_max_students: 0,
  over_limit: false,
}

const detailResponse = {
  id: 21,
  name: '审计园-ENTRY',
  principal: '王园长',
  status: 'active',
  student_count: 0,
  public_code: 'ab12cd34ef',
  entitlements: [],
  recent_orders: [],
}

const onboardResult = {
  kindergarten_id: 21,
  admin_phone: '13800001234',
  must_change_password: true,
  trial_expires_at: '2026-12-23T10:00:00+08:00',
  public_code: 'ab12cd34ef',
}

// 假口令需同时满足前端策略（大小写 + 数字）并含 dummy 占位词，避免 secret 扫描误报。
const PASSWORD = 'Dummy-Pass-123'

let Page: any
let ElementPlus: any
let message: { error: ReturnType<typeof vi.fn>; success: ReturnType<typeof vi.fn>; warning: ReturnType<typeof vi.fn> }

const clipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, 'clipboard')

function removeClipboard() {
  delete (navigator as { clipboard?: unknown }).clipboard
}

beforeAll(async () => {
  stubResizeObserver()
  vi.stubGlobal('ref', ref)
  vi.stubGlobal('reactive', reactive)
  vi.stubGlobal('computed', computed)
  vi.stubGlobal('onMounted', onMounted)
  stubNuxtPageMeta()
  message = { error: vi.fn(), success: vi.fn(), warning: vi.fn() }
  vi.stubGlobal('ElMessage', message)
  vi.stubGlobal('ElMessageBox', { confirm: vi.fn().mockResolvedValue('confirm') })
  ElementPlus = (await import('element-plus')).default
  Page = (await import('~/pages/platform/kindergartens/index.vue')).default
})

beforeEach(() => {
  document.body.innerHTML = ''
  removeClipboard()
  listKindergartens.mockReset().mockResolvedValue({ list: [listRow], total: 1, page: 1, page_size: 20 })
  getKindergarten.mockReset().mockResolvedValue({ ...detailResponse })
  onboardKindergarten.mockReset().mockResolvedValue({ ...onboardResult })
  setKindergartenStatus.mockReset()
  Object.values(message).forEach((fn) => fn.mockReset())
})

afterEach(() => {
  vi.useRealTimers()
  if (clipboardDescriptor) {
    Object.defineProperty(navigator, 'clipboard', clipboardDescriptor)
  } else {
    removeClipboard()
  }
})

function mountPage() {
  return mount(Page, { global: { plugins: [ElementPlus] } })
}

function pageText(wrapper: ReturnType<typeof mountPage>): string {
  // Element Plus 弹窗/抽屉在本配置下渲染在组件树内；两边都读，兼容 teleport。
  return `${wrapper.text()}\n${document.body.textContent || ''}`
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

function findDialogByTitle(wrapper: ReturnType<typeof mountPage>, title: string) {
  const dialog = wrapper.findAllComponents({ name: 'ElDialog' }).find((c) => c.props('title') === title)
  if (!dialog) throw new Error(`未找到标题为「${title}」的弹窗`)
  return dialog
}

function setInput(wrapper: ReturnType<typeof mountPage>, placeholder: string, value: string) {
  const input = wrapper
    .findAllComponents({ name: 'ElInput' })
    .find((c) => c.props('placeholder') === placeholder)
  if (!input) throw new Error(`未找到占位符为「${placeholder}」的输入框`)
  input.vm.$emit('update:modelValue', value)
}

async function openAndSubmitCreate(wrapper: ReturnType<typeof mountPage>) {
  clickButton(wrapper, '开通园所')
  await flushPromises()

  setInput(wrapper, '如：阳光幼儿园', '审计园-ENTRY')
  setInput(wrapper, '园长称呼', '王园长')
  setInput(wrapper, '用于登录园所后台', '13800001234')
  setInput(wrapper, '含大写、小写与数字，至少 8 位', PASSWORD)
  await flushPromises()

  clickButton(wrapper, '开通')
  await flushPromises()
  await flushPromises()
}

function stubClipboardSuccess() {
  const writeText = vi.fn().mockResolvedValue(undefined)
  Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
  return writeText
}

describe('平台园所页的入口码交付', () => {
  it('开通成功后持久弹窗展示园所 ID 与入口码，不回显口令并清空表单', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await openAndSubmitCreate(wrapper)

    // toast 只保留一句「已开通」，入口码靠弹窗交付。
    expect(message.success).toHaveBeenCalledWith('已开通')

    const text = pageText(wrapper)
    expect(text).toContain('家长入口码')
    expect(text).toContain('21')
    expect(text).toContain('园长')
    expect(text).not.toContain(PASSWORD)
    // 码本身在只读输入框里（可选中），不在 textContent 中。
    const codeInput = findDialogByTitle(wrapper, '开通成功').find('input').element as HTMLInputElement
    expect(codeInput.value).toBe('ab12cd34ef')
    for (const call of message.success.mock.calls) {
      expect(String(call[0])).not.toContain(PASSWORD)
    }

    // 表单已清空（含初始口令）；不要靠重新打开弹窗兜底。
    const passwordInput = wrapper
      .findAllComponents({ name: 'ElInput' })
      .find((c) => c.props('placeholder') === '含大写、小写与数字，至少 8 位')
    expect(passwordInput?.props('modelValue')).toBe('')
  })

  it('成功弹窗超过 5 秒仍在，直到显式关闭', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await openAndSubmitCreate(wrapper)

    const successDialog = findDialogByTitle(wrapper, '开通成功')
    expect(successDialog.props('modelValue')).toBe(true)

    vi.useFakeTimers()
    vi.advanceTimersByTime(6000)
    expect(successDialog.props('modelValue')).toBe(true)
    vi.useRealTimers()

    clickButton(wrapper, '关闭')
    await flushPromises()
    expect(successDialog.props('modelValue')).toBe(false)
  })

  it('复制按钮写入精确入口码并给出已复制提示', async () => {
    const writeText = stubClipboardSuccess()
    const wrapper = mountPage()
    await flushPromises()
    await openAndSubmitCreate(wrapper)

    clickButton(wrapper, '复制')
    await flushPromises()

    expect(writeText).toHaveBeenCalledTimes(1)
    expect(writeText).toHaveBeenCalledWith('ab12cd34ef')
    expect(pageText(wrapper)).toContain('已复制')
  })

  it('剪贴板不可用时降级为可手动选中的只读码并提示', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await openAndSubmitCreate(wrapper)

    clickButton(wrapper, '复制')
    await flushPromises()

    expect(message.warning).toHaveBeenCalled()
    expect(pageText(wrapper)).toContain('手动选中')

    const input = findDialogByTitle(wrapper, '开通成功').find('input').element as HTMLInputElement
    expect(input.readOnly).toBe(true)
    expect(input.value).toBe('ab12cd34ef')
  })

  it('详情抽屉显示入口码并可复制', async () => {
    const writeText = stubClipboardSuccess()
    const wrapper = mountPage()
    await flushPromises()

    clickButton(wrapper, '详情')
    await flushPromises()

    const text = pageText(wrapper)
    expect(text).toContain('家长入口码')
    const codeInput = wrapper.find('.el-drawer input').element as HTMLInputElement
    expect(codeInput.value).toBe('ab12cd34ef')

    clickButton(wrapper, '复制')
    await flushPromises()

    expect(writeText).toHaveBeenCalledWith('ab12cd34ef')
  })

  it('详情抽屉空码显示未生成且复制禁用', async () => {
    getKindergarten.mockResolvedValue({ ...detailResponse, public_code: '   ' })
    const wrapper = mountPage()
    await flushPromises()

    clickButton(wrapper, '详情')
    await flushPromises()

    expect(pageText(wrapper)).toContain('家长入口码')
    const codeInput = wrapper.find('.el-drawer input').element as HTMLInputElement
    expect(codeInput.value).toBe('未生成')

    const copyButton = wrapper
      .findAll('.el-drawer button')
      .find((b) => b.text().trim() === '复制')
    expect(copyButton).toBeTruthy()
    expect((copyButton?.element as HTMLButtonElement).disabled).toBe(true)
  })

  it('列表不新增入口码列，也不渲染入口码', async () => {
    listKindergartens.mockResolvedValue({
      list: [{ ...listRow, public_code: 'listonly99' }],
      total: 1,
      page: 1,
      page_size: 20,
    })
    const wrapper = mountPage()
    await flushPromises()

    const headers = wrapper.findAll('th').map((th) => th.text())
    expect(headers.join('|')).not.toContain('入口码')
    expect(wrapper.text()).not.toContain('listonly99')
  })

  it('弹窗与抽屉使用移动宽度上限', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await openAndSubmitCreate(wrapper)

    expect(findDialogByTitle(wrapper, '开通园所').props('width')).toBe('min(480px, 92vw)')
    expect(findDialogByTitle(wrapper, '开通成功').props('width')).toBe('min(480px, 92vw)')

    clickButton(wrapper, '查看该园详情')
    await flushPromises()
    expect(wrapper.findComponent({ name: 'ElDrawer' }).props('size')).toBe('min(520px, 92vw)')
  })
})
