import type { ExportTaskStatus } from '~/types/api'

export function exportAttendance(body: Record<string, unknown>) {
  return useHttp().post<ExportTaskStatus>('/admin/export/attendance', body)
}

export function exportStudentsTask(body: Record<string, unknown>) {
  return useHttp().post<ExportTaskStatus>('/admin/export/students', body)
}

export function exportGrowthTask(body: Record<string, unknown>) {
  return useHttp().post<ExportTaskStatus>('/admin/export/growth', body)
}

export function getExportTask(taskId: string) {
  return useHttp().get<ExportTaskStatus>(`/admin/export/${taskId}`)
}

/** 带 Bearer 下载已完成任务（勿用 window.open 直链） */
export async function downloadExportBlob(taskId: string) {
  const res = await useHttp().getBlob(`/admin/export/download/${taskId}`)
  return res.data as Blob
}
