import { describe, expect, it } from 'vitest'
import { formatMoney } from '../../utils/format'

/**
 * 金额格式化收敛到 `utils/format` 后，平台页展示的币种、精度与取整规则
 * 必须保持稳定：分转元、保留两位小数、不丢尾零。
 */
describe('formatMoney', () => {
  it('分转元并保留两位小数', () => {
    expect(formatMoney(0)).toBe('¥0.00')
    expect(formatMoney(1)).toBe('¥0.01')
    expect(formatMoney(100000)).toBe('¥1000.00')
    expect(formatMoney(123456)).toBe('¥1234.56')
  })

  it('负数（退款场景）保留符号', () => {
    expect(formatMoney(-123456)).toBe('¥-1234.56')
  })
})
