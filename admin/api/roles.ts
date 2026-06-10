import type { PermissionModule, RoleItem } from '~/types/api'

export function getRoles() {
  return useHttp().get<{ list: RoleItem[] }>('/admin/roles')
}

export function createRole(body: { name: string; description?: string; permissions: string[] }) {
  return useHttp().post<{ id: number; created_at: string }>('/admin/roles', body)
}

export function updateRole(
  id: number,
  body: { name?: string; description?: string; permissions?: string[] },
) {
  return useHttp().put<{ id: number; updated_at: string }>(`/admin/roles/${id}`, body)
}

export function deleteRole(id: number) {
  return useHttp().delete(`/admin/roles/${id}`)
}

export function getPermissionTree() {
  return useHttp().get<{ permissions: PermissionModule[] }>('/admin/permissions')
}
