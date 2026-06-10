import type { PageParams, PageResult, TaskItem } from '~/types/api'

export function getTasks(params?: PageParams & { class_id?: number; status?: number }) {
  return useHttp().get<PageResult<TaskItem>>('/admin/tasks', { params })
}

export function getTask(id: number) {
  return useHttp().get<Record<string, unknown>>(`/admin/tasks/${id}`)
}

export function createTask(body: {
  title: string
  description?: string
  class_id: number
  frequency: 'daily' | 'weekly' | 'once'
  start_date: string
  end_date: string
  points?: number
}) {
  return useHttp().post<{ id: number; created_at: string }>('/admin/tasks', body)
}

export function updateTask(
  id: number,
  body: {
    title: string
    description?: string
    class_id: number
    frequency: 'daily' | 'weekly' | 'once'
    start_date: string
    end_date: string
    points?: number
    status: number
  },
) {
  return useHttp().put<{ id: number }>(`/admin/tasks/${id}`, body)
}

export function deleteTask(id: number) {
  return useHttp().delete(`/admin/tasks/${id}`)
}

export function getTaskStats(params?: { start_date?: string; end_date?: string }) {
  return useHttp().get<Record<string, unknown>>('/admin/tasks/stats', { params })
}
