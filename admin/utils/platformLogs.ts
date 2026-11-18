/**
 * 平台审计页的查询参数与展示映射。
 *
 * 后端契约见 `backend/api/api-platform/06-audit.md`：
 * - `from` / `to` 为 RFC 3339，区间左闭右开 `[from, to)`
 * - 空筛选值不进入查询串（后端会把空字符串当筛选值）
 * - 页码由调用方重置：筛选变化时页面回到第 1 页
 */

export interface PlatformLogQueryInput {
  page: number
  pageSize: number
  operator?: string
  module?: string
  kindergartenID?: number | null
  /** [开始日, 结束日]，格式 `YYYY-MM-DD`（Element Plus daterange 的 value-format） */
  range?: [string, string] | null
}

export interface PlatformLogQuery {
  page: number
  page_size: number
  operator?: string
  module?: string
  kindergarten_id?: number
  from?: string
  to?: string
}

const DAY_RE = /^(\d{4})-(\d{2})-(\d{2})$/

/** 按本地时区解析 `YYYY-MM-DD`；非法或被 Date 静默进位的日期返回 null。 */
function parseLocalDay(value: string): Date | null {
  const match = DAY_RE.exec(value.trim())
  if (!match) return null
  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  const date = new Date(year, month - 1, day)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return null
  }
  return date
}

/** 输出带本地时区偏移的 RFC 3339，后端用 time.Parse(time.RFC3339) 解析。 */
function localRFC3339(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  const offsetMinutes = -date.getTimezoneOffset()
  const sign = offsetMinutes >= 0 ? '+' : '-'
  const abs = Math.abs(offsetMinutes)
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}` +
    `${sign}${pad(Math.floor(abs / 60))}:${pad(abs % 60)}`
  )
}

export function buildPlatformLogQuery(input: PlatformLogQueryInput): PlatformLogQuery {
  const query: PlatformLogQuery = { page: input.page, page_size: input.pageSize }

  const operator = input.operator?.trim()
  if (operator) query.operator = operator
  if (input.module) query.module = input.module
  if (input.kindergartenID && input.kindergartenID > 0) {
    query.kindergarten_id = input.kindergartenID
  }

  if (input.range) {
    const [startRaw, endRaw] = input.range
    const start = parseLocalDay(startRaw)
    const end = parseLocalDay(endRaw)
    if (start && end) {
      // 契约是左闭右开：结束日期换算成次日零点，否则会漏掉结束当天的记录。
      const endExclusive = new Date(end.getFullYear(), end.getMonth(), end.getDate() + 1)
      if (start.getTime() < endExclusive.getTime()) {
        query.from = localRFC3339(start)
        query.to = localRFC3339(endExclusive)
      }
    }
  }

  return query
}

export function formatLogTime(value?: string | null): string {
  if (!value) return '—'
  const match = /^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2}:\d{2})/.exec(value)
  return match ? `${match[1]} ${match[2]}` : value
}

const MODULE_LABELS: Record<string, string> = {
  kindergarten: '园所',
  order: '订单',
  payment: '收款',
  plan: '套餐',
  distributor: '分销商',
}

const ACTION_LABELS: Record<string, string> = {
  onboard: '开通园所',
  update_status: '变更状态',
  confirm: '确认订单',
  cancel: '取消订单',
  record: '登记收款',
  create: '新建',
  update: '更新',
}

export function moduleLabel(value?: string | null): string {
  if (!value) return '—'
  return MODULE_LABELS[value] || value
}

export function actionLabel(value?: string | null): string {
  if (!value) return '—'
  return ACTION_LABELS[value] || value
}

/** 模块筛选项，与 06-audit.md 的 `module` 取值一一对应。 */
export const PLATFORM_LOG_MODULES = [
  { value: 'kindergarten', label: '园所' },
  { value: 'order', label: '订单' },
  { value: 'payment', label: '收款' },
  { value: 'plan', label: '套餐' },
  { value: 'distributor', label: '分销商' },
] as const
