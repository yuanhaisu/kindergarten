import type {
  AttendanceTrendRes,
  ClassActivityRes,
  ClassRankingRes,
  DashboardOverview,
  GrowthStatsRes,
  LeaveReasonRes,
} from '~/types/api'

export function getDashboardOverview() {
  return useHttp().get<DashboardOverview>('/admin/dashboard/overview')
}

export function getAttendanceTrend(period: '7d' | '30d' | '12m' = '30d') {
  return useHttp().get<AttendanceTrendRes>('/admin/dashboard/attendance-trend', {
    params: { period },
  })
}

export function getClassRanking() {
  return useHttp().get<ClassRankingRes>('/admin/dashboard/class-ranking')
}

export function getClassActivity() {
  return useHttp().get<ClassActivityRes>('/admin/dashboard/class-activity')
}

export function getGrowthStats(period = '30d') {
  return useHttp().get<GrowthStatsRes>('/admin/dashboard/growth-stats', { params: { period } })
}

export function getLeaveReasons(period: '7d' | '30d' = '30d') {
  return useHttp().get<LeaveReasonRes>('/admin/dashboard/leave-reasons', {
    params: { period },
  })
}

export function getTodoList() {
  return useHttp().get<{ items: import('~/types/api').TodoItem[] }>('/admin/dashboard/todos')
}
