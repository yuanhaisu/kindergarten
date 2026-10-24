function pad(n: number) {
  return n < 10 ? `0${n}` : `${n}`
}

export function formatDateTime(iso?: string | null) {
  if (!iso) return '-'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return String(iso)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function formatDate(iso?: string | null) {
  if (!iso) return '-'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return String(iso)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function formatPercent(v: string | number) {
  if (typeof v === 'number') return `${v}%`
  return v || '-'
}

/**
 * 比率的展示值。后端把「无法计算」（分母为 0）一律返回 null，
 * 与真正的 0% 是两件事，必须显示「—」而不是 Number(null) → 0%。
 * 已是 "NN%" 字符串，不要再乘 100。
 */
export function formatRate(v?: string | null) {
  return v == null || v === '' ? '—' : v
}

/** 数值型指标（如中位数小时数）：null 同样显示「—」 */
export function formatMetric(v?: number | null, unit = '') {
  return v == null ? '—' : `${v}${unit}`
}
