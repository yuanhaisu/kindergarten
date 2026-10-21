import type { PageParams, PageResult, ParentItem } from '~/types/api'

export function getParents(params?: PageParams & { class_id?: number; keyword?: string }) {
  return useHttp().get<PageResult<ParentItem>>('/admin/parents', { params })
}

export function getParent(id: number) {
  return useHttp().get<Record<string, unknown>>(`/admin/parents/${id}`)
}

export function createParent(body: Record<string, unknown>) {
  return useHttp().post<{ id: number; created_at: string }>('/admin/parents', body)
}

export function updateParent(id: number, body: Record<string, unknown>) {
  return useHttp().put<{ id: number; updated_at: string }>(`/admin/parents/${id}`, body)
}

export function updateParentStatus(id: number, body: { status: number; reason?: string }) {
  return useHttp().put<{ id: number; status: number; status_label: string; updated_at: string }>(
    `/admin/parents/${id}/status`,
    body,
  )
}
