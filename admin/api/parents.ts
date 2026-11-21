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

// ---- 身份支持（#172）：解绑绑错的微信、更正录入错误的手机号 ----

export interface SupportIdentity {
  id: number
  app_id: string
  open_id_masked: string
  created_at: string
}

export interface SupportMembership {
  kindergarten_id: number
  kindergarten_name: string
  role: string
  class_id?: number
  class_name?: string
  title?: string
  title_label?: string
}

export interface ParentSupportView {
  user_id: number
  nickname: string
  phone: string
  identities: SupportIdentity[]
  memberships: SupportMembership[]
}

export function getParentSupport(id: number) {
  return useHttp().get<ParentSupportView>(`/admin/parents/${id}/support`)
}

export function unbindParentIdentity(id: number, identityId: number) {
  return useHttp().delete<{ user_id: number; unbound_identity_id: number; identities: SupportIdentity[] }>(
    `/admin/parents/${id}/identities/${identityId}`,
  )
}

export function correctParentPhone(id: number, phone: string) {
  return useHttp().put<ParentSupportView>(`/admin/parents/${id}/phone`, { phone })
}
