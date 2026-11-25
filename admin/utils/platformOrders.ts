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

// ==================== 确认按钮判定（#201 / #203 / #209） ====================

/** 订单确认判定需要的最小字段；详情接口的 OrderDetail 可直接结构匹配。 */
export interface OrderConfirmView {
  status: string
  amount_due_cents: number
  amount_paid_cents: number
  confirmed_at: string | null
  expires_at: string
}

/**
 * 解析服务期结束时刻。
 *
 * 详情接口返回 RFC3339（如 2027-05-05T00:00:00+08:00），必须按绝对时刻解析；
 * 旧实现拼 `${expires_at}T00:00:00` 会得到 Invalid Date，守卫永远不生效（#209）。
 * 历史纯日期值按「服务期最后一天」解释：结束于次日 0 点，当天仍可确认。
 */
export function parseServiceEnd(value: string): Date | null {
  const text = (value || '').trim()
  if (!text) return null
  if (/^\d{4}-\d{2}-\d{2}$/.test(text)) {
    const [y, m, d] = text.split('-').map(Number)
    const end = new Date(y, m - 1, d + 1)
    return Number.isNaN(end.getTime()) ? null : end
  }
  const end = new Date(text)
  return Number.isNaN(end.getTime()) ? null : end
}

/**
 * 服务期是否已结束。与后端 `expires_at <= now` 同口径（order_service.go 的
 * ErrOrderServiceEnded）；零值/无法解析的到期时间不判定为结束，交给服务端兜底。
 */
export function isServiceEnded(order: Pick<OrderConfirmView, 'expires_at'>, now: Date = new Date()): boolean {
  const end = parseServiceEnd(order.expires_at)
  return end !== null && end.getTime() <= now.getTime()
}

/**
 * 确认按钮的显示规则（#201 / #203）：
 * - 已确认（confirmed_at 非空）不再提供确认；
 * - 服务期已结束的订单不提供确认，即使已全额收款；
 * - 已过付款到期日（expired）与已取消的订单不提供确认；
 * - 部分收款（0 < 已收 < 应收）一律不可确认，理由不能替代收清；
 * - 零收款（信用开通）与已收清（含超收、全额晚付）可确认。
 */
export function canConfirm(order: OrderConfirmView, now: Date = new Date()): boolean {
  if (order.confirmed_at) return false
  if (order.status.startsWith('cancelled')) return false
  if (isServiceEnded(order, now)) return false
  if (order.status === 'expired') return false
  return order.amount_paid_cents === 0 || order.amount_paid_cents >= order.amount_due_cents
}

/** 部分收款（尚欠）提示：仅在订单仍可能确认时才展示，避免与终止/过期提示叠加。 */
export function isUnderpaid(order: OrderConfirmView, now: Date = new Date()): boolean {
  return (
    !order.confirmed_at &&
    !order.status.startsWith('cancelled') &&
    order.status !== 'expired' &&
    !isServiceEnded(order, now) &&
    order.amount_paid_cents > 0 &&
    order.amount_paid_cents < order.amount_due_cents
  )
}
