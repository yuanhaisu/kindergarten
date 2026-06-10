import type {
  CreateNoticeBody,
  NoticeDetail,
  NoticeItem,
  PageParams,
  PageResult,
  UpdateNoticeBody,
} from '~/types/api'

export function getNotices(
  params?: PageParams & { class_id?: number; category?: string; status?: number; keyword?: string },
) {
  return useHttp().get<PageResult<NoticeItem>>('/admin/notices', { params })
}

export function getNotice(id: number) {
  return useHttp().get<NoticeDetail>(`/admin/notices/${id}`)
}

export function createNotice(body: CreateNoticeBody) {
  return useHttp().post<{ id: number; status: number; created_at: string }>('/admin/notices', body)
}

export function updateNotice(id: number, body: UpdateNoticeBody) {
  return useHttp().put<{ id: number }>(`/admin/notices/${id}`, body)
}

export function deleteNotice(id: number) {
  return useHttp().delete(`/admin/notices/${id}`)
}

export function auditNotice(id: number, body: { action: 'pass' | 'reject'; remark?: string }) {
  return useHttp().put<{ id: number; audit_status: string; audited_at: string }>(
    `/admin/notices/${id}/audit`,
    body,
  )
}
