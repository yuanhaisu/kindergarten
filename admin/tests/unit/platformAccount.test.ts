import { describe, expect, it } from 'vitest'
import {
  buildAccountListQuery,
  validatePlatformPassword,
} from '../../utils/platformAccount'

/**
 * 平台账号口令策略必须与后端 `validatePassword` 同口径；
 * 账号列表查询串必须与园所列表同样跳过空关键词。
 */
describe('validatePlatformPassword', () => {
  it('合规口令通过', () => {
    expect(validatePlatformPassword('Abc12345')).toBeNull()
  })

  it('少于 8 位被拒', () => {
    expect(validatePlatformPassword('Ab1')).toBe('密码长度不能少于 8 位')
  })

  it('缺少大写 / 小写 / 数字分别被拒', () => {
    expect(validatePlatformPassword('abc12345')).toBe('密码须包含大写字母、小写字母和数字')
    expect(validatePlatformPassword('ABC12345')).toBe('密码须包含大写字母、小写字母和数字')
    expect(validatePlatformPassword('Abcdefgh')).toBe('密码须包含大写字母、小写字母和数字')
  })
})

describe('buildAccountListQuery', () => {
  it('默认只带分页参数', () => {
    expect(buildAccountListQuery({ page: 1, pageSize: 20 })).toEqual({ page: 1, page_size: 20 })
  })

  it('关键词去空白后传递，空关键词不出现', () => {
    expect(buildAccountListQuery({ page: 1, pageSize: 20, keyword: ' ops ' }).keyword).toBe('ops')
    const blank = buildAccountListQuery({ page: 1, pageSize: 20, keyword: '  ' })
    expect('keyword' in blank).toBe(false)
  })

  it('保留页码与页大小', () => {
    expect(buildAccountListQuery({ page: 3, pageSize: 50 })).toEqual({ page: 3, page_size: 50 })
  })
})
