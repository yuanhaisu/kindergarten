/**
 * 平台账号页的纯逻辑。
 *
 * 口令策略与「开通园所」共用同一实现（`utils/platformKindergarten`），
 * 最终对齐后端 `validatePassword`：至少 8 位，且同时包含大写、小写与数字。
 * 两端各写一份校验是本项目已经踩过的漂移来源，这里只做引用。
 */
import { validateAdminPassword } from './platformKindergarten'

export const validatePlatformPassword = validateAdminPassword

export interface AccountListQueryInput {
  page: number
  pageSize: number
  keyword?: string
}

export interface AccountListQuery {
  page: number
  page_size: number
  keyword?: string
}

/** 空关键词不进入查询串，避免后端把空字符串当筛选值。 */
export function buildAccountListQuery(input: AccountListQueryInput): AccountListQuery {
  const query: AccountListQuery = { page: input.page, page_size: input.pageSize }
  const keyword = input.keyword?.trim()
  if (keyword) query.keyword = keyword
  return query
}
