import type { ClassDetail, ClassItem, PageParams, PageResult } from '~/types/api'

export function getClasses(params?: PageParams & { grade?: string }) {
  return useHttp().get<PageResult<ClassItem>>('/admin/classes', { params })
}

export function getClassDetail(id: number) {
  return useHttp().get<ClassDetail>(`/admin/classes/${id}`)
}

export function createClass(body: Record<string, unknown>) {
  return useHttp().post<{ id: number; created_at: string }>('/admin/classes', body)
}

export function updateClass(id: number, body: Record<string, unknown>) {
  return useHttp().put<{ id: number; updated_at: string }>(`/admin/classes/${id}`, body)
}

export function deleteClass(id: number) {
  return useHttp().delete(`/admin/classes/${id}`)
}

export function adjustClassStudents(
  id: number,
  body: { action: 'move_in' | 'move_out'; student_ids: number[]; from_class_id?: number },
) {
  return useHttp().post<{ adjusted_count: number; message: string }>(`/admin/classes/${id}/adjust`, body)
}
