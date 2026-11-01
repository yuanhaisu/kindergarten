import type { SystemSettings } from '~/types/api'

/** 班级相册全班视图开关（园所级，默认关闭） */
export interface AlbumSettings {
  classwide_enabled: boolean
}

/**
 * `SystemSettings` 之上的 album 块。
 * 声明为必填与 health 同一口径：旧后端未下发时由设置页补默认值，避免 v-model 绑到 undefined。
 */
export type SystemSettingsWithAlbum = SystemSettings & { album: AlbumSettings }

export function getSettings() {
  return useHttp().get<SystemSettingsWithAlbum>('/admin/settings')
}

export function updateSettings(body: Partial<SystemSettingsWithAlbum>) {
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
