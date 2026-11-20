import { usePlatformHttp } from '~/composables/usePlatformHttp'
import type { Paged } from './types'

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

export function listModules() {
  return usePlatformHttp().get<{ list: ModuleDef[] }>('/platform/modules')
}

export interface PlanListParams {
  status?: string
  page?: number
  page_size?: number
}

export function listPlans(params: PlanListParams = {}) {
  return usePlatformHttp().get<Paged<PlanRow>>('/platform/plans', { params })
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
