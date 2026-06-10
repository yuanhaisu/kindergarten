import type { AuditListItem, PageParams, PageResult } from '~/types/api'

export function getAuditList(
  params?: PageParams & { type?: string; status?: string },
) {
  return useHttp().get<PageResult<AuditListItem>>('/admin/audit/list', { params })
}

export function auditContent(id: number, body: { action: 'pass' | 'reject'; remark?: string }) {
  return useHttp().put<{ id: number; status: string; audited_at: string }>(
    `/admin/audit/${id}`,
    body,
  )
}

export function getAuditStats() {
  return useHttp().get<Record<string, unknown>>('/admin/audit/stats')
}
