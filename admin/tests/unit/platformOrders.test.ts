import { describe, expect, it } from 'vitest'
import {
  buildOrderListParams,
  canConfirm,
  defaultPayDueAt,
  isPastPayDueAt,
  isServiceEnded,
  isUnderpaid,
} from '../../utils/platformOrders'

/**
 * 订单检索参数是页面与后端契约之间的唯一翻译点：
 * 空筛选不得出现在 query 中，日期范围与未清余额必须转成后端认识的参数名。
 */
describe('buildOrderListParams', () => {
  it('只带分页时不出现任何筛选参数', () => {
    expect(buildOrderListParams({ page: 1, pageSize: 20 })).toEqual({ page: 1, page_size: 20 })
  })

  it('订单号去空白，空串视为未填', () => {
    expect(buildOrderListParams({ page: 2, pageSize: 20, orderNo: '  ORD2026  ' })).toMatchObject({
      order_no: 'ORD2026',
    })
    expect(buildOrderListParams({ page: 1, pageSize: 20, orderNo: '   ' })).not.toHaveProperty('order_no')
  })

  it('日期范围映射为 start_date / end_date', () => {
    const params = buildOrderListParams({
      page: 1,
      pageSize: 20,
      dateRange: ['2026-11-01', '2026-11-18'],
    })
    expect(params.start_date).toBe('2026-11-01')
    expect(params.end_date).toBe('2026-11-18')
  })

  it('未清余额开关只在打开时出现', () => {
    expect(buildOrderListParams({ page: 1, pageSize: 20, outstandingOnly: true })).toMatchObject({
      outstanding_only: true,
    })
    expect(buildOrderListParams({ page: 1, pageSize: 20, outstandingOnly: false })).not.toHaveProperty(
      'outstanding_only',
    )
  })

  it('园所与状态按原值传递', () => {
    expect(
      buildOrderListParams({ page: 1, pageSize: 20, kindergartenId: 7, status: 'expired' }),
    ).toMatchObject({ kindergarten_id: 7, status: 'expired' })
  })
})

/**
 * #202：新建订单的付款到期日默认今天 +15 天，过去日期（不含当天）必须在客户端拒绝，
 * 与服务端 PAY_DUE_AT_IN_PAST 的口径一致。
 */
describe('付款到期日', () => {
  const now = new Date(2026, 10, 23, 10, 0, 0)

  it('默认今天 +15 天', () => {
    expect(defaultPayDueAt(now)).toBe('2026-12-08')
  })

  it('跨月时 +15 天正确滚动', () => {
    expect(defaultPayDueAt(new Date(2026, 11, 20, 8, 0, 0))).toBe('2027-01-04')
  })

  it('当天允许，昨天拒绝，明天允许', () => {
    expect(isPastPayDueAt('2026-11-22', now)).toBe(true)
    expect(isPastPayDueAt('2026-11-23', now)).toBe(false)
    expect(isPastPayDueAt('2026-11-24', now)).toBe(false)
  })
})

/**
 * #209 回归：详情接口的 expires_at 是 RFC3339（如 2027-05-05T00:00:00+08:00），
 * 旧实现拼成 `${expires_at}T00:00:00` 得到 Invalid Date，服务期守卫永远不生效。
 * 这里锁定「按真实时刻比较」和「历史纯日期值的兼容口径」。
 */
describe('服务期结束判定', () => {
  it('RFC3339 按绝对时刻比较，到期瞬间即算结束', () => {
    const now = new Date('2026-11-23T10:00:00+08:00')
    expect(isServiceEnded({ expires_at: '2026-11-23T09:59:59+08:00' }, now)).toBe(true)
    expect(isServiceEnded({ expires_at: '2026-11-23T10:00:00+08:00' }, now)).toBe(true)
    expect(isServiceEnded({ expires_at: '2026-11-23T10:00:01+08:00' }, now)).toBe(false)
    expect(isServiceEnded({ expires_at: '2027-05-05T00:00:00+08:00' }, now)).toBe(false)
  })

  it('历史纯日期值按当地日历日结束：最后一天仍有效，次日才结束', () => {
    const now = new Date(2026, 10, 23, 10, 0, 0)
    expect(isServiceEnded({ expires_at: '2026-11-22' }, now)).toBe(true)
    expect(isServiceEnded({ expires_at: '2026-11-23' }, now)).toBe(false)
  })

  it('空值与无法解析的值一律不判定为结束', () => {
    const now = new Date(2026, 10, 23, 10, 0, 0)
    expect(isServiceEnded({ expires_at: '' }, now)).toBe(false)
    expect(isServiceEnded({ expires_at: 'not-a-date' }, now)).toBe(false)
  })
})

describe('确认按钮谓词', () => {
  const now = new Date('2026-11-23T10:00:00+08:00')
  const base = {
    status: 'paid',
    amount_due_cents: 200000,
    amount_paid_cents: 200000,
    confirmed_at: null,
    expires_at: '2027-05-05T00:00:00+08:00',
  }

  it('服务期已结束即使已全额收款也不可确认', () => {
    expect(canConfirm({ ...base, expires_at: '2020-01-01T00:00:00+08:00' }, now)).toBe(false)
  })

  it('服务期未结束、已收清或零收款可确认', () => {
    expect(canConfirm(base, now)).toBe(true)
    expect(canConfirm({ ...base, status: 'unpaid', amount_paid_cents: 0 }, now)).toBe(true)
  })

  it('已确认、已取消、已过期、部分收款均不可确认', () => {
    expect(canConfirm({ ...base, confirmed_at: '2026-11-20T10:00:00+08:00' }, now)).toBe(false)
    expect(canConfirm({ ...base, status: 'cancelled' }, now)).toBe(false)
    expect(canConfirm({ ...base, status: 'expired' }, now)).toBe(false)
    expect(canConfirm({ ...base, status: 'partial', amount_paid_cents: 100000 }, now)).toBe(false)
  })

  it('部分收款只在未终止、未过期且服务期未结束时展示尚欠提示', () => {
    const partial = { ...base, status: 'partial', amount_paid_cents: 100000 }
    expect(isUnderpaid(partial, now)).toBe(true)
    expect(isUnderpaid({ ...partial, expires_at: '2020-01-01T00:00:00+08:00' }, now)).toBe(false)
    expect(isUnderpaid({ ...partial, status: 'expired' }, now)).toBe(false)
    expect(isUnderpaid({ ...partial, confirmed_at: '2026-11-20T10:00:00+08:00' }, now)).toBe(false)
  })
})
