import type { HealthStats } from '~/types/api'

export function getHealthStats(params: { date?: string }) {
  return useHttp().get<HealthStats>('/admin/health/stats', params)
}
