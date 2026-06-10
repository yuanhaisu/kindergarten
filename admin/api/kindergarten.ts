import type { KindergartenInfo } from '~/types/api'

export function getKindergartenInfo() {
  return useHttp().get<KindergartenInfo>('/admin/kindergarten/info')
}

export function updateKindergartenInfo(body: Partial<KindergartenInfo>) {
  return useHttp().put<{ id: number; updated_at: string }>('/admin/kindergarten/info', body)
}
