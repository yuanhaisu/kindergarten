import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  copyPublicCode,
  hasPublicCode,
  normalizePublicCode,
  publicCodeText,
} from '../../utils/publicCode'

/**
 * 园所家长入口码（`public_code`）的展示与复制。
 *
 * 验收背景（#204）：后端在开通响应/列表/详情都返回 10 位入口码，但旧 UI 完全
 * 丢弃；修复后空码必须显示「未生成」，且 HTTP staging 下 `navigator.clipboard`
 * 可能不存在（旧教师页写法会同步抛 TypeError，`.catch` 接不到），必须降级为
 * 手动选中复制。
 */

const clipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, 'clipboard')

function stubClipboard(value: unknown) {
  Object.defineProperty(navigator, 'clipboard', { value, configurable: true })
}

afterEach(() => {
  if (clipboardDescriptor) {
    Object.defineProperty(navigator, 'clipboard', clipboardDescriptor)
  } else {
    // jsdom 默认没有 clipboard；恢复为「不存在」而不是 undefined 属性。
    delete (navigator as { clipboard?: unknown }).clipboard
  }
})

describe('normalizePublicCode', () => {
  it('去掉首尾空白', () => {
    expect(normalizePublicCode('  ab12cd34ef ')).toBe('ab12cd34ef')
  })

  it('空值/空串/纯空白归一为空串', () => {
    expect(normalizePublicCode(undefined)).toBe('')
    expect(normalizePublicCode(null)).toBe('')
    expect(normalizePublicCode('')).toBe('')
    expect(normalizePublicCode('   ')).toBe('')
  })
})

describe('publicCodeText', () => {
  it('有码时原样展示', () => {
    expect(publicCodeText('ab12cd34ef')).toBe('ab12cd34ef')
  })

  it('无码时显示未生成，而不是 undefined', () => {
    expect(publicCodeText(undefined)).toBe('未生成')
    expect(publicCodeText('')).toBe('未生成')
    expect(publicCodeText('   ')).toBe('未生成')
  })
})

describe('hasPublicCode', () => {
  it('只有非空码才可复制', () => {
    expect(hasPublicCode('ab12cd34ef')).toBe(true)
    expect(hasPublicCode(undefined)).toBe(false)
    expect(hasPublicCode('  ')).toBe(false)
  })
})

describe('copyPublicCode', () => {
  it('写入精确码并返回 copied', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    stubClipboard({ writeText })

    await expect(copyPublicCode(' ab12cd34ef ')).resolves.toBe('copied')
    expect(writeText).toHaveBeenCalledTimes(1)
    expect(writeText).toHaveBeenCalledWith('ab12cd34ef')
  })

  it('clipboard 不存在时降级为 manual，且不抛异常', async () => {
    stubClipboard(undefined)
    await expect(copyPublicCode('ab12cd34ef')).resolves.toBe('manual')
  })

  it('writeText 被拒绝时降级为 manual', async () => {
    stubClipboard({ writeText: vi.fn().mockRejectedValue(new Error('denied')) })
    await expect(copyPublicCode('ab12cd34ef')).resolves.toBe('manual')
  })

  it('空码不触发 clipboard 写入', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    stubClipboard({ writeText })
    await expect(copyPublicCode('   ')).resolves.toBe('manual')
    expect(writeText).not.toHaveBeenCalled()
  })
})
