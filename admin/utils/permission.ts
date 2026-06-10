/**
 * 检查是否拥有任一权限（* 表示超管）
 */
export function hasPermission(codes: string[] | undefined | null, need: string | string[]) {
  if (!codes?.length) return false
  if (codes.includes('*')) return true
  const list = Array.isArray(need) ? need : [need]
  return list.some((k) => codes.includes(k))
}
