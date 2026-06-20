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

export function getAuthorizedPersons(
  params?: PageParams & { status?: string },
) {
  return useHttp().get<
    PageResult<{
      id: number
      student_id: number
      student_name: string
      class_name: string
      name: string
      relation: string
      phone: string
      auth_type: string
      review_status: number
      review_label: string
      audited_at: string | null
      audit_remark: string
      allow_pickup: boolean
      allow_view_record: boolean
      allow_growth_view: boolean
      created_at: string
    }>
  >('/admin/pickup/authorized-persons', { params })
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
