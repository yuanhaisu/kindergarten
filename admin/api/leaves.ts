import type { LeaveItem, PageParams, PageResult } from '~/types/api'

export function getLeaves(
  params?: PageParams & { class_id?: number; status?: number; start_date?: string; end_date?: string },
) {
  return useHttp().get<PageResult<LeaveItem>>('/admin/leaves', { params })
}

export function getLeaveStats(params?: { start_date?: string; end_date?: string }) {
  return useHttp().get<Record<string, unknown>>('/admin/leaves/stats', { params })
}

/** 审批请假：status 1 同意，2 驳回 */
export function auditLeave(id: number, body: { status: number; review_remark?: string }) {
  return useHttp().put<Record<string, unknown>>(`/admin/leaves/${id}/audit`, body)
}
