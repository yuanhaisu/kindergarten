import { usePlatformHttp } from '~/composables/usePlatformHttp'

export interface DashboardExpiringKindergarten {
  id: number
  name: string
  expires_at: string
}

export interface DashboardSettleableCommission {
  distributor_id: number
  distributor_name: string
  amount_cents: number
  commission_count: number
}

export interface PlatformDashboard {
  kindergarten_total: number
  kindergarten_active: number
  kindergarten_expiring_30d: number
  received_this_month_cents: number
  outstanding_cents: number
  commission_settleable_cents: number
  expiring_kindergartens: DashboardExpiringKindergarten[]
  settleable_commissions: DashboardSettleableCommission[]
}

/** 运营看板：6 个指标 + 2 个待办，后端一次聚合。 */
export function getDashboard() {
  return usePlatformHttp().get<PlatformDashboard>('/platform/dashboard')
}
