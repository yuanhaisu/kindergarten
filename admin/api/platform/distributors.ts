import { usePlatformHttp } from '~/composables/usePlatformHttp'
import type { Paged } from './types'

export interface DistributorRow {
  id: number
  name: string
  contact: string
  phone: string
  commission_rate: number
  status: number
  created_at: string
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
