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

/** 按本地日历日格式化为 YYYY-MM-DD；不用 toISOString，避免时区导致的日期偏移。 */
export function formatLocalDate(date: Date): string {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

/**
 * 新建订单的默认付款到期日：今天 +15 天（宽限期）。
 * 服务端缺省仍回落到服务起始日；UI 必须显式发送未来日期，
 * 否则订单当天到期、次日即不可确认（#202）。
 */
export function defaultPayDueAt(now: Date = new Date()): string {
  const due = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 15)
  return formatLocalDate(due)
}

/**
 * 付款到期日不得早于本地今天；当天允许（与服务端 PAY_DUE_AT_IN_PAST 口径一致）。
 * 空值不在此处拦截，交给必填规则。
 */
export function isPastPayDueAt(value: string, now: Date = new Date()): boolean {
  if (!value) return false
  const [y, m, d] = value.split('-').map(Number)
  if (!y || !m || !d) return false
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  return new Date(y, m - 1, d).getTime() < today.getTime()
}
