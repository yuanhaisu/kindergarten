import type { PageParams, PageResult, PickupOverview, PickupRecordItem } from '~/types/api'

export function getPickupOverview(params?: { date?: string }) {
  return useHttp().get<PickupOverview>('/admin/pickup/overview', { params })
}

export function getPickupRecords(
  params?: PageParams & { class_id?: number; date?: string; is_abnormal?: boolean },
) {
  return useHttp().get<PageResult<PickupRecordItem>>('/admin/pickup/records', { params })
}

export function getPickupAbnormal(params?: PageParams & { date?: string }) {
  return useHttp().get<PageResult<Record<string, unknown>>>('/admin/pickup/abnormal', { params })
}

export function auditPickupAuthorizedPerson(
  id: number,
  body: { action: 'approve' | 'reject'; remark?: string },
) {
  return useHttp().put<{ id: number; status: number; status_label: string; audited_at: string }>(
    `/admin/pickup/authorized-persons/${id}/audit`,
    body,
  )
}
