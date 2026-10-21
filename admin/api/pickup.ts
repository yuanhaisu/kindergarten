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

// 滞留名单（实时口径：已入园且此刻仍未接走；学生挂在 by_class[].students 下）
// 形状以 backend/api/api-admin/08-pickup.md 与 Go ListOverdueStudents 为准
export function getPickupOverdue(params?: { date?: string }) {
  return useHttp().get<{
    date: string
    overdue_total: number
    by_class: Array<{
      class_id: number
      class_name: string
      pickup_deadline: string
      overdue_count: number
      students: Array<{
        student_id: number
        name: string
        avatar: string
        arrive_time: string
        pickup_deadline: string
      }>
    }>
  }>('/admin/pickup/overdue', { params })
}

// 超时未接计数（累计口径：当日越过放学时刻即计一次，随后被接走也不清零）
export function getPickupOverdueCount(params?: { date?: string }) {
  return useHttp().get<{ date: string; overdue_not_picked: number }>(
    '/admin/pickup/overdue/count',
    { params },
  )
}

// 一键通知滞留儿童家长
export function notifyPickupOverdue(body: { student_id?: number }) {
  return useHttp().put<{ notified: number; student_id?: number }>(
    '/admin/pickup/overdue/notify',
    body,
  )
}
