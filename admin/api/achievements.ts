import type { AchievementItem, PageParams, PageResult } from '~/types/api'

export function getAchievements(params?: PageParams) {
  return useHttp().get<PageResult<AchievementItem>>('/admin/achievements', { params })
}

export function createAchievement(body: Record<string, unknown>) {
  return useHttp().post<{ id: number; created_at: string }>('/admin/achievements', body)
}

export function updateAchievement(id: number, body: Record<string, unknown>) {
  return useHttp().put<{ id: number; updated_at: string }>(`/admin/achievements/${id}`, body)
}

export function deleteAchievement(id: number) {
  return useHttp().delete(`/admin/achievements/${id}`)
}
