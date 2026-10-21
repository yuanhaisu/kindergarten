import type {
  GardenCalendarEventItem,
  GardenCalendarInput,
  GardenCalendarListResult,
} from '~/types/api'

/** 园级事件列表（仅 scope=kindergarten；status 缺省返回全部） */
export function listGardenEvents(params?: { start?: string; end?: string; status?: string }) {
  return useHttp().get<GardenCalendarListResult>('/admin/calendar/events', { params })
}

export function createGardenEvent(body: GardenCalendarInput) {
  return useHttp().post<{ id: number; created_at: string }>('/admin/calendar/events', body)
}

export function updateGardenEvent(id: number, body: GardenCalendarInput) {
  return useHttp().put<{ id: number }>(`/admin/calendar/events/${id}`, body)
}

export function deleteGardenEvent(id: number) {
  return useHttp().delete(`/admin/calendar/events/${id}`)
}

export type { GardenCalendarEventItem }
