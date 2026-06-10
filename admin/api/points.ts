import type { PageParams, PageResult, PointsOverview, PointRecordItem } from '~/types/api'

export function getPointsOverview() {
  return useHttp().get<PointsOverview>('/admin/points/overview')
}

export function getPointRecords(
  params?: PageParams & { student_id?: number; class_id?: number; type?: string },
) {
  return useHttp().get<PageResult<PointRecordItem>>('/admin/points/records', { params })
}

export function adjustPoints(body: {
  student_id: number
  type: 'earn' | 'spend'
  amount: number
  reason: string
}) {
  return useHttp().post<{ id: number; balance: number; created_at: string }>(
    '/admin/points/adjust',
    body,
  )
}

export function getPointRules() {
  return useHttp().get<{
    rules: { id: number; source: string; label: string; points: number; editable: boolean }[]
  }>('/admin/points/rules')
}

export function updatePointRules(body: { rules: { id: number; points: number }[] }) {
  return useHttp().put<{ updated_count: number; updated_at: string }>('/admin/points/rules', body)
}
