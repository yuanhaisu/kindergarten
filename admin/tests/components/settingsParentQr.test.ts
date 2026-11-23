import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { computed, onMounted, ref } from 'vue'

/**
 * 家长入口二维码（#205）行为验收：
 * - 生成后展示后端返回的 PNG 与 public_code；
 * - 下载使用 image_base64 解码出的 PNG，文件名含入口码；
 * - public_code 为空时提示「未生成」，不渲染图片、下载禁用；
 * - 端点失败显示可操作错误，绝不渲染占位/假二维码。
 */

const getSettings = vi.fn()
const updateSettings = vi.fn()
const getAcademicYears = vi.fn()
const createAcademicYear = vi.fn()
const getKindergartenInfo = vi.fn()
const updateKindergartenInfo = vi.fn()
const getParentEntryQRCode = vi.fn()
const uploadFile = vi.fn()
const downloadBlob = vi.fn()

vi.mock('~/api/settings', () => ({
  getSettings: (...args: unknown[]) => getSettings(...args),
  updateSettings: (...args: unknown[]) => updateSettings(...args),
  getAcademicYears: (...args: unknown[]) => getAcademicYears(...args),
  createAcademicYear: (...args: unknown[]) => createAcademicYear(...args),
}))

vi.mock('~/api/kindergarten', () => ({
  getKindergartenInfo: (...args: unknown[]) => getKindergartenInfo(...args),
  updateKindergartenInfo: (...args: unknown[]) => updateKindergartenInfo(...args),
  getParentEntryQRCode: (...args: unknown[]) => getParentEntryQRCode(...args),
}))

vi.mock('~/api/common', () => ({
  uploadFile: (...args: unknown[]) => uploadFile(...args),
}))

vi.mock('~/utils/export', () => ({
  downloadBlob: (...args: unknown[]) => downloadBlob(...args),
  parseFilenameFromDisposition: vi.fn(),
}))

// 1x1 透明 PNG；用作后端 image_base64 的已知样本。
const PNG_BASE64 =
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=='

const settingsPayload = {
  basic: { academic_year: '2026-2027', semester: 'fall', semester_start: '2026-09-01', semester_end: '2027-01-31' },
  attendance: {
    arrive_start_time: '07:30',
    arrive_end_time: '09:00',
    pickup_start_time: '16:30',
    late_threshold_minutes: 15,
  },
  notice: { need_audit: true, auto_remind_hours: 24 },
  growth: { need_audit: false, allow_parent_comment: true, allow_parent_like: true },
  pickup: { qr_code_expire_hours: 24, need_authorization_audit: true },
  points: { task_checkin_points: 10, weekly_attendance_bonus: 5, monthly_attendance_bonus: 20 },
  health: { temperature_required: true, fever_threshold: 37.3 },
  album: { classwide_enabled: false },
}

const kindergartenPayload = {
  id: 7,
  name: '阳光幼儿园',
  address: '北京市朝阳区阳光路123号',
  phone: '010-12345678',
  principal: '张园长',
  logo_url: '',
  brand_color: '',
  license_no: '',
  capacity: 120,
  current_count: 88,
  class_count: 4,
  teacher_count: 9,
  established_at: '2015-09-01',
  description: '',
}

const qrPayload = {
  public_code: 'k3m9x2qa7t',
  scene: 'kg_k3m9x2qa7t',
  page: 'pages/index/index',
  env_version: 'release',
  image_base64: PNG_BASE64,
}

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
  vi.stubGlobal('computed', computed)
  vi.stubGlobal('onMounted', onMounted)
  vi.stubGlobal('definePageMeta', vi.fn())
  vi.stubGlobal('usePermissionStore', () => ({ can: () => true }))
  vi.stubGlobal('useMediaRefresh', () => () => {})
  vi.stubGlobal('ElMessage', { error: vi.fn(), success: vi.fn(), warning: vi.fn() })
  ElementPlus = (await import('element-plus')).default
  Page = (await import('~/pages/settings/index.vue')).default
})

beforeEach(() => {
  document.body.innerHTML = ''
  getSettings.mockReset().mockResolvedValue({ ...settingsPayload })
  getAcademicYears.mockReset().mockResolvedValue({ list: [] })
  getKindergartenInfo.mockReset().mockResolvedValue({ ...kindergartenPayload })
  getParentEntryQRCode.mockReset()
  downloadBlob.mockReset()
})

function mountPage() {
  return mount(Page, { global: { plugins: [ElementPlus] } })
}

function button(wrapper: ReturnType<typeof mountPage>, label: string) {
  const found = wrapper.findAll('button').find((b) => b.text().trim() === label)
  if (!found) throw new Error(`未找到「${label}」按钮`)
  return found
}

function isDisabled(btn: ReturnType<typeof button>) {
  return btn.attributes('disabled') !== undefined || btn.classes().includes('is-disabled')
}

async function generate(wrapper: ReturnType<typeof mountPage>) {
  await button(wrapper, '生成二维码').trigger('click')
  await flushPromises()
}

const qrImage = (wrapper: ReturnType<typeof mountPage>) =>
  wrapper.find('img[alt="家长端园所入口二维码"]')

describe('家长入口二维码', () => {
  it('生成后展示真实 PNG 与入口码', async () => {
    getParentEntryQRCode.mockResolvedValue({ ...qrPayload })
    const wrapper = mountPage()
    await flushPromises()

    await generate(wrapper)

    expect(getParentEntryQRCode).toHaveBeenCalledTimes(1)
    expect(getParentEntryQRCode).toHaveBeenCalledWith(undefined)
    expect(qrImage(wrapper).exists()).toBe(true)
    expect(qrImage(wrapper).attributes('src')).toBe(`data:image/png;base64,${PNG_BASE64}`)
    expect(wrapper.text()).toContain('k3m9x2qa7t')
    expect(wrapper.text()).not.toContain('未生成')
    expect(isDisabled(button(wrapper, '下载二维码'))).toBe(false)
  })

  it('下载把 image_base64 解码为 PNG，文件名含入口码', async () => {
    getParentEntryQRCode.mockResolvedValue({ ...qrPayload })
    const wrapper = mountPage()
    await flushPromises()
    await generate(wrapper)

    await button(wrapper, '下载二维码').trigger('click')
    await flushPromises()

    expect(downloadBlob).toHaveBeenCalledTimes(1)
    const [blob, filename] = downloadBlob.mock.calls[0] as [Blob, string]
    expect(blob).toBeInstanceOf(Blob)
    expect(blob.type).toBe('image/png')
    expect(blob.size).toBeGreaterThan(0)
    expect(filename).toBe('家长入口二维码-k3m9x2qa7t.png')
  })

  it('public_code 为空时提示未生成，不渲染图片且禁用下载', async () => {
    getParentEntryQRCode.mockResolvedValue({ ...qrPayload, public_code: '', scene: '' })
    const wrapper = mountPage()
    await flushPromises()

    await generate(wrapper)

    expect(wrapper.text()).toContain('未生成')
    expect(qrImage(wrapper).exists()).toBe(false)
    expect(wrapper.html()).not.toContain('data:image/png')
    expect(isDisabled(button(wrapper, '下载二维码'))).toBe(true)
  })

  it('端点失败显示可操作错误，绝不渲染假二维码', async () => {
    getParentEntryQRCode.mockRejectedValue(
      Object.assign(new Error('当前园所的通道与家长端小程序不匹配，无法生成二维码'), { status: 409 }),
    )
    const wrapper = mountPage()
    await flushPromises()

    await generate(wrapper)

    const text = wrapper.text()
    expect(text).toContain('通道')
    expect(text).toContain('联系平台')
    expect(qrImage(wrapper).exists()).toBe(false)
    expect(wrapper.html()).not.toContain('data:image/png')
    expect(isDisabled(button(wrapper, '下载二维码'))).toBe(true)
  })
})
