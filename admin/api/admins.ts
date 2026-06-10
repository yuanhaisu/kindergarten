import type { AdminItem, PageParams, PageResult } from '~/types/api'

export function getAdmins(params?: PageParams) {
  return useHttp().get<PageResult<AdminItem>>('/admin/admins', { params })
}

export function createAdmin(body: { phone: string; nickname: string; role_id: number; password: string }) {
  return useHttp().post<{ id: number; created_at: string }>('/admin/admins', body)
}

export function updateAdmin(id: number, body: { nickname?: string; role_id?: number }) {
  return useHttp().put<{ id: number; updated_at: string }>(`/admin/admins/${id}`, body)
}

export function deleteAdmin(id: number) {
  return useHttp().delete(`/admin/admins/${id}`)
}

export function resetAdminPassword(id: number, body: { new_password: string }) {
  return useHttp().put<{ message: string }>(`/admin/admins/${id}/reset-password`, body)
}
