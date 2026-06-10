import type { OperationLogItem, PageParams, PageResult } from '~/types/api'

export function getOperationLogs(
  params?: PageParams & {
    module?: string
    operator_id?: number
    start_date?: string
    end_date?: string
  },
) {
  return useHttp().get<PageResult<OperationLogItem>>('/admin/logs', { params })
}
