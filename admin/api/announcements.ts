import type { AnnouncementDetail, AnnouncementItem, PageParams, PageResult } from '~/types/api'

export function getAnnouncements(params?: PageParams & { status?: number }) {
  return useHttp().get<PageResult<AnnouncementItem>>('/admin/announcements', { params })
}

export function getAnnouncement(id: number) {
  return useHttp().get<AnnouncementDetail>(`/admin/announcements/${id}`)
}

export function createAnnouncement(body: Record<string, unknown>) {
  return useHttp().post<{ id: number; status: number; published_at: string }>(
    '/admin/announcements',
    body,
  )
}

export function updateAnnouncement(id: number, body: Record<string, unknown>) {
  return useHttp().put<{ id: number; updated_at: string }>(`/admin/announcements/${id}`, body)
}

export function deleteAnnouncement(id: number) {
  return useHttp().delete(`/admin/announcements/${id}`)
}

export function setAnnouncementTop(id: number, body: { is_top: boolean }) {
  return useHttp().put<{ id: number; is_top: boolean; updated_at: string }>(
    `/admin/announcements/${id}/top`,
    body,
  )
}
