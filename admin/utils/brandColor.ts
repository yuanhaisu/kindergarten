/**
 * 园所品牌主题色的前端辅助（#178）。
 *
 * 合法性最终由服务端裁决：预设色板与对比度校验（WCAG，相对白色 >= 3:1）都在
 * `server/internal/service/brand_theme.go`，前端复刻对比度公式只会两边漂移。
 * 这里只做「展示预设色 + 格式归一化」，让管理员在下发前就看到格式错误。
 *
 * 预设色板与 Go 侧 `presetBrandColors` 同源，增删颜色时两边都要改。
 */
export const BRAND_COLOR_PRESETS = [
  '#F97316', // 家长端品牌橙（默认）
  '#FF8A3D', // 老师端品牌橙
  '#E8A23A', // identity 琥珀
  '#C36540', // DNA 陶土
  '#5C7359', // DNA 鼠尾草绿
  '#173220', // DNA 松绿
  '#3B82F6', // 管理端品牌蓝
] as const

/**
 * 归一化管理员输入的主题色：
 * - 空串（含纯空白）= 清空，回落默认主题；
 * - 合法 `#RRGGBB` 转大写（服务端存大写，避免仅大小写差异的重复写入）；
 * - 其他返回 null，由调用方提示，不静默丢弃输入。
 */
export function normalizeBrandColorInput(raw: string | null | undefined): string | null {
  const trimmed = (raw ?? '').trim()
  if (!trimmed) return ''
  const normalized = trimmed.toUpperCase()
  return /^#[0-9A-F]{6}$/.test(normalized) ? normalized : null
}
