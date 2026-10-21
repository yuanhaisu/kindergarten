import type {
  TimetableClassOverridesResult,
  TimetableEntry,
  TimetableOverride,
  TimetableSyncResult,
  TimetableTemplateResult,
} from '~/types/api'

export function getTimetableTemplate(grade: string) {
  return useHttp().get<TimetableTemplateResult>(
    `/admin/timetable/templates/${encodeURIComponent(grade)}`,
  )
}

/** 全量替换该年级模板；开启课表时服务端会自动重算未来窗口 */
export function saveTimetableTemplate(grade: string, entries: TimetableEntry[]) {
  return useHttp().put<TimetableTemplateResult>(
    `/admin/timetable/templates/${encodeURIComponent(grade)}`,
    { entries },
  )
}

export function getTimetableClassOverrides(classId: number) {
  return useHttp().get<TimetableClassOverridesResult>(`/admin/timetable/classes/${classId}`)
}

/** 全量替换该班微调项；开启课表时服务端会自动重算未来窗口 */
export function saveTimetableClassOverrides(classId: number, overrides: TimetableOverride[]) {
  return useHttp().put<TimetableClassOverridesResult>(`/admin/timetable/classes/${classId}`, {
    overrides,
  })
}

export function getTimetableSetting() {
  return useHttp().get<{ timetable_enabled: boolean }>('/admin/timetable/setting')
}

export function updateTimetableSetting(timetableEnabled: boolean) {
  return useHttp().put<{ timetable_enabled: boolean }>('/admin/timetable/setting', {
    timetable_enabled: timetableEnabled,
  })
}

/** 立即同步（与每日滚动等价、幂等；排障用） */
export function syncTimetable() {
  return useHttp().post<TimetableSyncResult>('/admin/timetable/sync')
}
