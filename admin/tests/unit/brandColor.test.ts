import { describe, expect, it } from 'vitest'
import { BRAND_COLOR_PRESETS, normalizeBrandColorInput } from '../../utils/brandColor'

/**
 * 管理端保存品牌色前的归一化（#178）。
 *
 * 服务端仍会独立校验预设色板与对比度；这里锁定的是「空值清空、大小写收敛、
 * 非法格式显式失败」这三条，避免把非法值悄悄提交上去或把清空当成空提交。
 */
describe('normalizeBrandColorInput', () => {
  it('空值表示清空，回落默认主题', () => {
    expect(normalizeBrandColorInput('')).toBe('')
    expect(normalizeBrandColorInput('   ')).toBe('')
    expect(normalizeBrandColorInput(null)).toBe('')
    expect(normalizeBrandColorInput(undefined)).toBe('')
  })

  it('合法 hex 去空格并转大写', () => {
    expect(normalizeBrandColorInput('#1d4ed8')).toBe('#1D4ED8')
    expect(normalizeBrandColorInput('  #F97316  ')).toBe('#F97316')
  })

  it('非法格式显式返回 null，交由调用方提示', () => {
    expect(normalizeBrandColorInput('red')).toBeNull()
    expect(normalizeBrandColorInput('#C24')).toBeNull()
    expect(normalizeBrandColorInput('#GGGGGG')).toBeNull()
    // 带 alpha 的 8 位值服务端不接受，前端不能放过
    expect(normalizeBrandColorInput('#1D4ED8FF')).toBeNull()
  })

  it('预设色板本身都能通过归一化', () => {
    for (const preset of BRAND_COLOR_PRESETS) {
      expect(normalizeBrandColorInput(preset)).toBe(preset)
    }
  })
})
