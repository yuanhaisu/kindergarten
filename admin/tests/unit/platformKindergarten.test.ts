import { describe, expect, it } from 'vitest'
import {
  buildKindergartenListQuery,
  validateAdminPassword,
  validatePhone,
} from '../../utils/platformKindergarten'

/**
 * 平台「开通园所」表单与园所列表筛选的纯逻辑。
 * 口令策略必须与后端 `validatePassword`（server/internal/service/system_service.go）
 * 保持同一口径：至少 8 位，且同时包含大写字母、小写字母与数字。
 */

describe('validateAdminPassword', () => {
  it('合规口令通过', () => {
    expect(validateAdminPassword('Abc12345')).toBeNull()
    expect(validateAdminPassword('Abc123!@#')).toBeNull()
  })

  it('少于 8 位时提示与后端一致', () => {
    expect(validateAdminPassword('Ab1')).toBe('密码长度不能少于 8 位')
    expect(validateAdminPassword('Abc1234')).toBe('密码长度不能少于 8 位')
  })

  it('缺少大写字母时提示与后端一致', () => {
    expect(validateAdminPassword('abc12345')).toBe('密码须包含大写字母、小写字母和数字')
  })

  it('缺少小写字母时提示与后端一致', () => {
    expect(validateAdminPassword('ABC12345')).toBe('密码须包含大写字母、小写字母和数字')
  })

  it('缺少数字时提示与后端一致', () => {
    expect(validateAdminPassword('Abcdefgh')).toBe('密码须包含大写字母、小写字母和数字')
  })

  it('只含小写与数字也被拒', () => {
    expect(validateAdminPassword('abcdefg1')).toBe('密码须包含大写字母、小写字母和数字')
  })

  it('空口令按长度不足处理', () => {
    expect(validateAdminPassword('')).toBe('密码长度不能少于 8 位')
  })
})

describe('validatePhone', () => {
  it('合法手机号通过', () => {
    expect(validatePhone('13800001234')).toBeNull()
    expect(validatePhone('19912345678')).toBeNull()
  })

  it('非法手机号提示与后端一致', () => {
    const message = '手机号格式不正确'
    expect(validatePhone('12800001234')).toBe(message)
    expect(validatePhone('1380000123')).toBe(message)
    expect(validatePhone('138000012345')).toBe(message)
    expect(validatePhone('23800001234')).toBe(message)
    expect(validatePhone('1380000123a')).toBe(message)
    expect(validatePhone('')).toBe(message)
  })
})

describe('buildKindergartenListQuery', () => {
  it('默认只带分页参数', () => {
    const query = buildKindergartenListQuery({ page: 1, pageSize: 20 })
    expect(query).toEqual({ page: 1, page_size: 20 })
    expect('keyword' in query).toBe(false)
    expect('status' in query).toBe(false)
  })

  it('保留分页页码（筛选由调用方重置到第 1 页）', () => {
    expect(buildKindergartenListQuery({ page: 3, pageSize: 20 }).page).toBe(3)
  })

  it('关键词去空白后传递，空关键词不出现', () => {
    expect(buildKindergartenListQuery({ page: 1, pageSize: 20, keyword: ' 阳光 ' }).keyword).toBe('阳光')
    const blank = buildKindergartenListQuery({ page: 1, pageSize: 20, keyword: '   ' })
    expect('keyword' in blank).toBe(false)
  })

  it('状态筛选按原值传递，空状态不出现', () => {
    expect(buildKindergartenListQuery({ page: 1, pageSize: 20, status: 'suspended' }).status).toBe('suspended')
    expect('status' in buildKindergartenListQuery({ page: 1, pageSize: 20, status: '' })).toBe(false)
  })
})
