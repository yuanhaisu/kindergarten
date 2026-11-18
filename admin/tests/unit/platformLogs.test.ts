import { describe, expect, it } from 'vitest'
import {
  buildPlatformLogQuery,
  formatLogTime,
  moduleLabel,
  actionLabel,
} from '../../utils/platformLogs'

/**
 * 平台审计页的查询参数与展示映射。
 *
 * 契约（backend/api/api-platform/06-audit.md）：
 * - `from` / `to` 为 RFC 3339，时间区间左闭右开 `[from, to)`
 * - 空筛选值不进入查询串，后端把空字符串当筛选值
 * - 页码由调用方重置（筛选变化回第 1 页）
 */

describe('buildPlatformLogQuery', () => {
  it('默认只有分页参数', () => {
    const query = buildPlatformLogQuery({ page: 1, pageSize: 20 })
    expect(query).toEqual({ page: 1, page_size: 20 })
    expect('operator' in query).toBe(false)
    expect('module' in query).toBe(false)
    expect('kindergarten_id' in query).toBe(false)
    expect('from' in query).toBe(false)
    expect('to' in query).toBe(false)
  })

  it('操作人去空白后传递，空操作人不出现', () => {
    expect(buildPlatformLogQuery({ page: 1, pageSize: 20, operator: '  ops ' }).operator).toBe('ops')
    expect('operator' in buildPlatformLogQuery({ page: 1, pageSize: 20, operator: '   ' })).toBe(false)
  })

  it('模块按原值传递，空模块不出现', () => {
    expect(buildPlatformLogQuery({ page: 1, pageSize: 20, module: 'order' }).module).toBe('order')
    expect('module' in buildPlatformLogQuery({ page: 1, pageSize: 20, module: '' })).toBe(false)
  })

  it('园所 ID 有效时才传递', () => {
    expect(buildPlatformLogQuery({ page: 1, pageSize: 20, kindergartenID: 7 }).kindergarten_id).toBe(7)
    expect('kindergarten_id' in buildPlatformLogQuery({ page: 1, pageSize: 20, kindergartenID: 0 })).toBe(false)
    expect('kindergarten_id' in buildPlatformLogQuery({ page: 1, pageSize: 20, kindergartenID: null })).toBe(false)
  })

  it('开始日期是当天零点，结束日期换算成次日零点（右开区间含结束当天）', () => {
    const query = buildPlatformLogQuery({
      page: 1,
      pageSize: 20,
      range: ['2026-11-01', '2026-11-01'],
    })
    expect(query.from).toMatch(/^2026-11-01T00:00:00[+-]\d{2}:\d{2}$/)
    expect(query.to).toMatch(/^2026-11-02T00:00:00[+-]\d{2}:\d{2}$/)
    // 区间长度正好一天：说明 to 是结束日期的次日零点，而不是结束日期零点。
    expect(Date.parse(query.to!) - Date.parse(query.from!)).toBe(24 * 60 * 60 * 1000)
  })

  it('多日区间按起止日期换算', () => {
    const query = buildPlatformLogQuery({
      page: 1,
      pageSize: 20,
      range: ['2026-11-01', '2026-11-05'],
    })
    expect(query.from).toMatch(/^2026-11-01T00:00:00/)
    expect(query.to).toMatch(/^2026-11-06T00:00:00/)
  })

  it('非法日期不出现在查询串中', () => {
    const query = buildPlatformLogQuery({
      page: 1,
      pageSize: 20,
      range: ['not-a-date', '2026-11-05'],
    })
    expect('from' in query).toBe(false)
    expect('to' in query).toBe(false)
  })

  it('起点晚于终点时不传时间筛选', () => {
    const query = buildPlatformLogQuery({
      page: 1,
      pageSize: 20,
      range: ['2026-11-05', '2026-11-01'],
    })
    expect('from' in query).toBe(false)
    expect('to' in query).toBe(false)
  })

  it('保留调用方页码（重置页码由页面负责）', () => {
    expect(buildPlatformLogQuery({ page: 3, pageSize: 20 }).page).toBe(3)
  })
})

describe('formatLogTime', () => {
  it('RFC 3339 转成可读的本地时间文本', () => {
    expect(formatLogTime('2026-11-06T10:00:00+08:00')).toBe('2026-11-06 10:00:00')
  })

  it('空值显示占位符', () => {
    expect(formatLogTime('')).toBe('—')
    expect(formatLogTime(null)).toBe('—')
  })
})

describe('moduleLabel / actionLabel', () => {
  it('已知模块与动作使用中文标签', () => {
    expect(moduleLabel('order')).toBe('订单')
    expect(moduleLabel('payment')).toBe('收款')
    expect(actionLabel('confirm')).toBe('确认订单')
    expect(actionLabel('update_status')).toBe('变更状态')
  })

  it('未知值回退到原始代码，空值显示占位符', () => {
    expect(moduleLabel('account')).toBe('account')
    expect(actionLabel('rotate_secret')).toBe('rotate_secret')
    expect(moduleLabel('')).toBe('—')
    expect(actionLabel('')).toBe('—')
  })
})
