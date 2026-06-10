import type { AttendanceOverview, PageParams, PageResult } from '~/types/api'

export function getAttendanceOverview(params?: { date?: string }) {
  return useHttp().get<AttendanceOverview>('/admin/attendance/overview', { params })
}

export function getAttendanceRecords(
  params?: PageParams & { class_id?: number; date?: string; status?: string },
) {
  return useHttp().get<PageResult<Record<string, unknown>>>('/admin/attendance/records', { params })
}

export function exportAttendanceReport(params: Record<string, string | undefined>) {
  return useHttp().getBlob('/admin/attendance/report', { params })
}
