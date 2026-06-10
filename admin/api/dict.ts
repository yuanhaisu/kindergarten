import type { DictTypeDetail, DictItemDetail } from '~/types/api'

export function listDictTypes() {
  return useHttp().get<{ list: DictTypeDetail[] }>('/admin/dicts/types')
}

export function createDictType(body: {
  type_key: string
  type_name: string
  description?: string
  sort_order?: number
}) {
  return useHttp().post<DictTypeDetail>('/admin/dicts/types', body)
}

export function updateDictType(
  id: number,
  body: Partial<{ type_name: string; description: string; sort_order: number }>,
) {
  return useHttp().put<DictTypeDetail>(`/admin/dicts/types/${id}`, body)
}

export function deleteDictType(id: number) {
  return useHttp().delete<void>(`/admin/dicts/types/${id}`)
}

export function listDictItems(typeId: number) {
  return useHttp().get<{ list: DictItemDetail[] }>(`/admin/dicts/types/${typeId}/items`)
}

export function createDictItem(
  typeId: number,
  body: {
    item_key: string
    item_label: string
    extra?: Record<string, unknown>
    sort_order?: number
    is_enabled?: boolean
  },
) {
  return useHttp().post<DictItemDetail>(`/admin/dicts/types/${typeId}/items`, body)
}

export function updateDictItem(
  id: number,
  body: Partial<{
    item_key: string
    item_label: string
    extra: Record<string, unknown>
    sort_order: number
    is_enabled: boolean
  }>,
) {
  return useHttp().put<DictItemDetail>(`/admin/dicts/items/${id}`, body)
}

export function deleteDictItem(id: number) {
  return useHttp().delete<void>(`/admin/dicts/items/${id}`)
}

export function batchSortDictItems(body: { items: { id: number; sort_order: number }[] }) {
  return useHttp().put<void>('/admin/dicts/items/sort', body)
}
