import type { PageParams, PageResult, StudentDetail, StudentItem } from '~/types/api'

export function getStudents(
  params?: PageParams & { class_id?: number; status?: number },
) {
  return useHttp().get<PageResult<StudentItem>>('/admin/students', { params })
}

export function getStudent(id: number) {
  return useHttp().get<StudentDetail>(`/admin/students/${id}`)
}

export function createStudent(body: Record<string, unknown>) {
  return useHttp().post<{ id: number; student_no: string; created_at: string }>(
    '/admin/students',
    body,
  )
}

export function updateStudent(id: number, body: Record<string, unknown>) {
  return useHttp().put<{ id: number; updated_at: string }>(`/admin/students/${id}`, body)
}

export function deleteStudent(id: number) {
  return useHttp().delete(`/admin/students/${id}`)
}

export function updateStudentStatus(id: number, body: { status: number; reason?: string }) {
  return useHttp().put<{ id: number; status: number; status_label: string; updated_at: string }>(
    `/admin/students/${id}/status`,
    body,
  )
}

export type StudentImportResult = {
  total: number
  success_count: number
  fail_count: number
  fail_details: { row: number; name: string; reason: string }[]
}

export function importStudents(file: File) {
  const fd = new FormData()
  fd.append('file', file)
  return useHttp()
    .raw.post<StudentImportResult>('/admin/students/import', fd)
    .then((r) => r.data as StudentImportResult)
}

export function exportStudentsBlob(params: { class_id?: number; status?: number; format?: string }) {
  return useHttp()
    .getBlob('/admin/students/export', { params })
    .then((r) => r.data as Blob)
}
