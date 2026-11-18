import { usePlatformHttp } from '~/composables/usePlatformHttp'
import type { Paged } from './types'

export interface CommissionReversalRow {
  id: number
  payment_id: number
  base_amount_cents: number
  amount_cents: number
  status: string
  created_at: string
}

export interface CommissionRow {
  id: number
  order_no: string
  kindergarten_name: string
  distributor_name: string
  payment_id: number
  base_amount_cents: number
  rate: number
  amount_cents: number
  reversed_amount_cents: number
  net_amount_cents: number
  status: string
  received_at: string
  eligible_at: string
  reversal_of: number | null
  reversals?: CommissionReversalRow[]
}

export interface CommissionSummary {
  pending_cents: number
  settleable_cents: number
  settled_cents: number
  reversed_cents: number
}

export function listCommissions(params: Record<string, unknown>) {
  return usePlatformHttp().get<Paged<CommissionRow> & { summary: CommissionSummary }>('/platform/commissions', { params })
}
