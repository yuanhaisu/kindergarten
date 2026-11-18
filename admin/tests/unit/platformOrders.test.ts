import { describe, expect, it } from 'vitest'
import { buildOrderListParams } from '../../utils/platformOrders'

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
