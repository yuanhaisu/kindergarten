import type { GrowthLogItem, PageParams, PageResult } from '~/types/api'

export function getGrowthLogs(
  params?: PageParams & { class_id?: number; teacher_id?: number; date?: string },
) {
  return useHttp().get<PageResult<GrowthLogItem>>('/admin/growth/logs', { params })
}

export function getGrowthLog(id: number) {
  return useHttp().get<Record<string, unknown>>(`/admin/growth/logs/${id}`)
}

export function deleteGrowthLog(id: number) {
  return useHttp().delete(`/admin/growth/logs/${id}`)
}

export function auditGrowthLog(id: number, body: { action: 'pass' | 'reject'; remark?: string }) {
  return useHttp().put<{ id: number; audit_status: string; audited_at: string }>(
    `/admin/growth/logs/${id}/audit`,
    body,
  )
}

export function getGrowthStats(params?: { start_date?: string; end_date?: string }) {
  return useHttp().get<Record<string, unknown>>('/admin/growth/stats', { params })
}
