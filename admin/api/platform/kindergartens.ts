import { usePlatformHttp } from '~/composables/usePlatformHttp'
import type { Paged } from './types'

export interface PlatformKindergartenRow {
  id: number
  name: string
  principal: string
  status: 'active' | 'suspended'
  student_count: number
  active_plan_name: string
  soonest_expire_at: string | null
  created_at: string
  tier_max_students: number
  over_limit: boolean
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
