import type { PageParams, PageResult, TeacherDetail, TeacherInvitation, TeacherItem } from '~/types/api'

export function getTeachers(params?: PageParams & { class_id?: number; status?: number }) {
  return useHttp().get<PageResult<TeacherItem>>('/admin/teachers', { params })
}

export function getTeacher(id: number) {
  return useHttp().get<TeacherDetail>(`/admin/teachers/${id}`)
}

export function createTeacher(body: Record<string, unknown>) {
  return useHttp().post<{ id: number; invite_code: string; expires_at: string; created_at: string }>('/admin/teachers', body)
}

export function updateTeacher(id: number, body: Record<string, unknown>) {
  return useHttp().put<{ id: number; updated_at: string }>(`/admin/teachers/${id}`, body)
}

export function deleteTeacher(id: number) {
  return useHttp().delete(`/admin/teachers/${id}`)
}

export function updateTeacherStatus(id: number, body: { status: number; reason?: string }) {
  return useHttp().put<{ id: number; status: number; status_label: string; updated_at: string }>(
    `/admin/teachers/${id}/status`,
    body,
  )
}

export function getTeacherInvitation(id: number) {
  return useHttp().get<TeacherInvitation>(`/admin/teachers/${id}/invitation`)
}

export function regenerateTeacherInvitation(id: number) {
  return useHttp().post<TeacherInvitation>(`/admin/teachers/${id}/invitation`)
}
