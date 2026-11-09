import { usePlatformHttp } from '~/composables/usePlatformHttp'

export interface Paged<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface PlatformKindergartenRow {
  id: number
  name: string
  principal: string
  status: 'active' | 'suspended'
  student_count: number
  active_plan_name: string
  soonest_expire_at: string | null
  created_at: string
}

export interface PlatformEntitlementRow {
  module_code: string
  module_name: string
  source: 'trial' | 'subscription' | 'gift'
  starts_at: string
  expires_at: string
  state: 'active' | 'grace' | 'expired'
}

export interface PlatformKindergartenDetail {
  id: number
  name: string
  principal: string
  status: 'active' | 'suspended'
  student_count: number
  entitlements: PlatformEntitlementRow[]
  recent_orders: { id: number; order_no: string; amount_due_cents: number; status: string }[]
}

export interface PlanRow {
  id: number
  code: string
  name: string
  tier: number
  price_cents: number
  term_months: number
  module_codes: string[]
  status: string
  created_at: string
}

export interface ModuleDef {
  code: string
  name: string
  category: 'base' | 'premium'
  sellable: boolean
}

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
  plan_snapshot: { plan_name: string; module_codes: string[]; price_cents: number }
  distributor: { id: number; name: string; commission_rate: number } | null
  invoice_info: Record<string, string> | null
  payments: { id: number; amount_cents: number; method: string; received_at: string; remark: string }[]
}

export interface DistributorRow {
  id: number
  name: string
  contact: string
  phone: string
  commission_rate: number
  status: number
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
  status: string
  received_at: string
  eligible_at: string
  reversal_of: number | null
}

export interface CommissionSummary {
  pending_cents: number
  settleable_cents: number
  settled_cents: number
  reversed_cents: number
}

export interface SubscriptionView {
  plan_name: string
  modules: PlatformEntitlementRow[]
  expires_at: string | null
  grace_until: string | null
  state: 'active' | 'grace' | 'expired'
}

export function platformLogin(username: string, password: string) {
  return usePlatformHttp().post<{ token: string; expires_in: number; admin: { id: number; username: string; name: string } }>(
    '/platform/auth/login',
    { username, password },
  )
}

export function platformLogout() {
  return usePlatformHttp().post('/platform/auth/logout', {})
}

export function listKindergartens(params: Record<string, unknown>) {
  return usePlatformHttp().get<Paged<PlatformKindergartenRow>>('/platform/kindergartens', { params })
}

export function getKindergarten(id: number) {
  return usePlatformHttp().get<PlatformKindergartenDetail>(`/platform/kindergartens/${id}`)
}

export function onboardKindergarten(body: Record<string, unknown>) {
  return usePlatformHttp().post<{ kindergarten_id: number; admin_phone: string; must_change_password: boolean; trial_expires_at: string }>(
    '/platform/kindergartens',
    body,
  )
}

export function setKindergartenStatus(id: number, status: string) {
  return usePlatformHttp().patch(`/platform/kindergartens/${id}/status`, { status })
}

export function listModules() {
  return usePlatformHttp().get<{ list: ModuleDef[] }>('/platform/modules')
}

export function listPlans(status?: string) {
  return usePlatformHttp().get<Paged<PlanRow>>('/platform/plans', { params: { status } })
}

export function createPlan(body: Record<string, unknown>) {
  return usePlatformHttp().post<{ id: number }>('/platform/plans', body)
}

export function updatePlan(id: number, body: Record<string, unknown>) {
  return usePlatformHttp().patch(`/platform/plans/${id}`, body)
}

export function listPlanTiers(id: number) {
  return usePlatformHttp().get<{ list: { tier: number; min_students: number; max_students: number; price_multiplier: number }[] }>(
    `/platform/plans/${id}/tiers`,
  )
}

export function replacePlanTiers(id: number, list: unknown[]) {
  return usePlatformHttp().put(`/platform/plans/${id}/tiers`, { list })
}

export function listOrders(params: Record<string, unknown>) {
  return usePlatformHttp().get<Paged<OrderRow>>('/platform/orders', { params })
}

export function createOrder(body: Record<string, unknown>) {
  return usePlatformHttp().post<{ id: number; order_no: string; amount_due_cents: number }>('/platform/orders', body)
}

export function getOrder(id: number) {
  return usePlatformHttp().get<OrderDetail>(`/platform/orders/${id}`)
}

export function confirmOrder(id: number) {
  return usePlatformHttp().post(`/platform/orders/${id}/confirm`, {})
}

export function cancelOrder(id: number, reason: string) {
  return usePlatformHttp().post(`/platform/orders/${id}/cancel`, { reason })
}

export function addPayment(id: number, body: Record<string, unknown>) {
  return usePlatformHttp().post(`/platform/orders/${id}/payments`, body)
}

export function listDistributors(params: Record<string, unknown>) {
  return usePlatformHttp().get<Paged<DistributorRow>>('/platform/distributors', { params })
}

export function createDistributor(body: Record<string, unknown>) {
  return usePlatformHttp().post<{ id: number }>('/platform/distributors', body)
}

export function updateDistributor(id: number, body: Record<string, unknown>) {
  return usePlatformHttp().patch(`/platform/distributors/${id}`, body)
}

export function listCommissions(params: Record<string, unknown>) {
  return usePlatformHttp().get<Paged<CommissionRow> & { summary: CommissionSummary }>('/platform/commissions', { params })
}

export function getSubscription() {
  return useHttp().get<SubscriptionView>('/admin/billing/subscription')
}
