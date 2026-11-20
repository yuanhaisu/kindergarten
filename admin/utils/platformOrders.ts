/**
 * 平台订单列表的查询参数翻译。
 *
 * 页面只持有控件状态，参数命名以后端契约为准
 * （backend/api/api-platform/04-orders.md）：空筛选不进入 query，
 * 日期范围拆成 start_date / end_date，未清余额开关只在打开时出现。
 */

export interface OrderListQueryInput {
  page: number
  pageSize: number
  orderNo?: string
  dateRange?: [string, string] | null
  kindergartenId?: number
  status?: string
  outstandingOnly?: boolean
}

export interface OrderListQuery {
  page: number
  page_size: number
  order_no?: string
  start_date?: string
  end_date?: string
  kindergarten_id?: number
  status?: string
  outstanding_only?: boolean
}

export function buildOrderListParams(input: OrderListQueryInput): OrderListQuery {
  const params: OrderListQuery = { page: input.page, page_size: input.pageSize }
  const orderNo = input.orderNo?.trim()
  if (orderNo) params.order_no = orderNo
  if (input.dateRange && input.dateRange.length === 2) {
    const [start, end] = input.dateRange
    if (start) params.start_date = start
    if (end) params.end_date = end
  }
  if (input.kindergartenId) params.kindergarten_id = input.kindergartenId
  if (input.status) params.status = input.status
  if (input.outstandingOnly) params.outstanding_only = true
  return params
}
