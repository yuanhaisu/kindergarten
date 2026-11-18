import { usePlatformHttp } from '~/composables/usePlatformHttp'
import type { Paged } from './types'

/** 平台操作审计记录，字段与 `backend/api/api-platform/06-audit.md` 对齐。 */
export type PlatformLogRow = {
  id: number
  operator: string
  module: string
  action: string
  target_type: string
  target_id: number
  /** 目标园所；套餐 / 分销商 / 平台账号类操作为 0。 */
  kindergarten_id: number
  detail: string
  ip: string
  /** 与响应头 X-Request-ID 及后端请求日志一致。 */
  request_id: string
  created_at: string
  updated_at: string
}

/** 查询平台操作审计。参数由 `buildPlatformLogQuery` 生成（时间区间左闭右开）。 */
export function listPlatformLogs(params: Record<string, unknown>) {
  return usePlatformHttp().get<Paged<PlatformLogRow>>('/platform/logs', { params })
}
