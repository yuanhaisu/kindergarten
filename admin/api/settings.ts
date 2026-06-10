import type { SystemSettings } from '~/types/api'

export function getSettings() {
  return useHttp().get<SystemSettings>('/admin/settings')
}

export function updateSettings(body: Partial<SystemSettings>) {
  return useHttp().put<{ updated_at: string }>('/admin/settings', body)
}

export function getAcademicYears() {
  return useHttp().get<{ list: { year: string; is_current: boolean }[] }>(
    '/admin/settings/academic-years',
  )
}

export interface CreateAcademicYearBody {
  year: string
  semester_spring_start: string
  semester_spring_end: string
  semester_fall_start: string
  semester_fall_end: string
}

export function createAcademicYear(body: CreateAcademicYearBody) {
  return useHttp().post<{ year: string; created_at: string }>(
    '/admin/settings/academic-years',
    body,
  )
}
