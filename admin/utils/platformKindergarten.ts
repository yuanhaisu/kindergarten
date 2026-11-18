/**
 * 平台「开通园所」表单校验与园所列表查询参数。
 *
 * 口令策略必须与后端 `validatePassword`（server/internal/service/system_service.go）
 * 保持同一口径，否则会出现「前端放行、后端 422」或「前端误拦」：
 * - 至少 8 位
 * - 同时包含大写字母、小写字母与数字（Unicode 分类，与 Go 的
 *   `unicode.IsUpper/IsLower/IsDigit` 对齐）
 * 提示文案直接复用后端返回的策略说明。
 */

const PHONE_RE = /^1[3-9]\d{9}$/

export function validateAdminPassword(password: string): string | null {
  if (Array.from(password).length < 8) {
    return '密码长度不能少于 8 位'
  }
  const hasUpper = /\p{Lu}/u.test(password)
  const hasLower = /\p{Ll}/u.test(password)
  const hasDigit = /\p{Nd}/u.test(password)
  if (!hasUpper || !hasLower || !hasDigit) {
    return '密码须包含大写字母、小写字母和数字'
  }
  return null
}

export function validatePhone(phone: string): string | null {
  return PHONE_RE.test(phone) ? null : '手机号格式不正确'
}

export interface KindergartenListQueryInput {
  page: number
  pageSize: number
  keyword?: string
  status?: string
}

export interface KindergartenListQuery {
  page: number
  page_size: number
  keyword?: string
  status?: string
}

/** 空关键词/空状态不进入查询串，避免后端把空字符串当筛选值。 */
export function buildKindergartenListQuery(input: KindergartenListQueryInput): KindergartenListQuery {
  const query: KindergartenListQuery = { page: input.page, page_size: input.pageSize }
  const keyword = input.keyword?.trim()
  if (keyword) query.keyword = keyword
  if (input.status) query.status = input.status
  return query
}
