import { describe, expect, it } from 'vitest'
import { resolveAuthGate } from '../../utils/authFlow'
import { safeRedirectPath } from '../../utils/authPaths'

/**
 * 登录/同意后的去向（#199）：
 * 同意优先于改密（契约顺序：先同意、再改密），改密优先于业务页。
 * 回跳地址只接受站内路径，外链回退到首页。
 */

describe('resolveAuthGate', () => {
  it('未同意时先走同意页，即使同时需要改密', () => {
    expect(
      resolveAuthGate({ consentRequired: true, mustChangePassword: true }, '/billing'),
    ).toEqual({ path: '/consent', query: { redirect: '/billing' } })
  })

  it('已同意但未改密时进入专用改密页并保留回跳', () => {
    expect(
      resolveAuthGate({ consentRequired: false, mustChangePassword: true }, '/growth'),
    ).toEqual({ path: '/change-password', query: { redirect: '/growth' } })
  })

  it('无门槛时回到业务页', () => {
    expect(resolveAuthGate({}, '/dashboard')).toEqual({ path: '/dashboard' })
  })
})

describe('safeRedirectPath', () => {
  it('接受站内路径', () => {
    expect(safeRedirectPath('/billing?tab=1')).toBe('/billing?tab=1')
  })

  it('拒绝外链与协议相对地址', () => {
    expect(safeRedirectPath('https://example.com')).toBe('/')
    expect(safeRedirectPath('//evil.example.com')).toBe('/')
    expect(safeRedirectPath('/\\evil.example.com')).toBe('/')
    expect(safeRedirectPath(undefined)).toBe('/')
    expect(safeRedirectPath(['/a'])).toBe('/')
  })

  it('支持自定义兜底页', () => {
    expect(safeRedirectPath('https://example.com', '/platform')).toBe('/platform')
  })
})
