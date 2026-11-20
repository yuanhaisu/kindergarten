import { usePlatformHttp } from '~/composables/usePlatformHttp'
import type { OrderListQuery } from '~/utils/platformOrders'
import type { Paged } from './types'

export interface OrderRow {
  id: number
  order_no: string
  kindergarten_id: number
  kindergarten_name: string
  plan_name: string
  student_tier: number
  amount_due_cents: number
  amount_paid_cents: number
  status: string
  starts_at: string
  expires_at: string
  pay_due_at: string
  // 零收款确认的信用开通标记；理由只在详情返回。
  credit_granted: boolean
  distributor_name: string
  created_at: string
}

export interface OrderDetail {
  id: number
  order_no: string
  kindergarten_id: number
  kindergarten_name: string
  status: string
  amount_due_cents: number
  amount_paid_cents: number
  starts_at: string
  expires_at: string
  pay_due_at: string
  credit_granted: boolean
  credit_reason: string
  plan_snapshot: { plan_name: string; module_codes: string[]; price_cents: number }
  distributor: { id: number; name: string; commission_rate: number } | null
  invoice_info: Record<string, string> | null
  payments: {
    id: number
    amount_cents: number
    method: string
    received_at: string
    remark: string
    refund_of_payment_id: number | null
  }[]
}

export function listOrders(params: OrderListQuery) {
  return usePlatformHttp().get<Paged<OrderRow>>('/platform/orders', { params })
}

export function createOrder(body: Record<string, unknown>) {
  return usePlatformHttp().post<{
    id: number
    order_no: string
    amount_due_cents: number
    pay_due_at: string
  }>('/platform/orders', body)
}

export function getOrder(id: number) {
  return usePlatformHttp().get<OrderDetail>(`/platform/orders/${id}`)
}

// 净收款为零时后端要求 reason（信用开通理由），缺省返回 422。
export function confirmOrder(id: number, reason?: string) {
  return usePlatformHttp().post(`/platform/orders/${id}/confirm`, reason ? { reason } : {})
}

export function cancelOrder(id: number, reason: string) {
  return usePlatformHttp().post(`/platform/orders/${id}/cancel`, { reason })
}

// 幂等键随请求头提交：同一笔转账重复点击只落一条收款，重试返回首次结果。
export function addPayment(id: number, body: Record<string, unknown>, idempotencyKey?: string) {
  return usePlatformHttp().post(
    `/platform/orders/${id}/payments`,
    body,
    idempotencyKey ? { headers: { 'Idempotency-Key': idempotencyKey } } : undefined,
  )
}
