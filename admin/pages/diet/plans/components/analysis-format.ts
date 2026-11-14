/**
 * 营养分析页的数值展示。
 *
 * 后端已按报表口径决定小数位，前端只补空值占位；任何二次 toFixed/取整都会
 * 让页面与接口、导出 Excel 对不上（ADR-0004：保留参考产品的非标准口径）。
 */
export function formatNumber(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '—'
  return String(value)
}

export function formatPercent(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '—'
  return `${value}%`
}
