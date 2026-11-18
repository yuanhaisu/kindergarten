import { usePlatformHttp } from '~/composables/usePlatformHttp'
import type { Paged } from './types'

export type PlatformAccountStatus = 'enabled' | 'disabled'

export interface PlatformAccountRow {
  id: number
  username: string
  name: string
  status: PlatformAccountStatus
  created_at: string
}

export interface CreateAccountBody {
  username: string
  name?: string
  password: string
}

export function listAccounts(params: Record<string, unknown>) {
  return usePlatformHttp().get<Paged<PlatformAccountRow>>('/platform/admins', { params })
}

export function createAccount(body: CreateAccountBody) {
  return usePlatformHttp().post<PlatformAccountRow>('/platform/admins', body)
}

export function setAccountStatus(id: number, status: PlatformAccountStatus) {
  return usePlatformHttp().patch<{ id: number; status: PlatformAccountStatus; updated_at: string }>(
    `/platform/admins/${id}/status`,
    { status },
  )
}

/** 重置他人口令；响应不回显口令。 */
export function resetAccountPassword(id: number, password: string) {
  return usePlatformHttp().post<{ id: number; updated_at: string }>(
    `/platform/admins/${id}/reset-password`,
    { password },
  )
}

/** 修改本人密码；成功后当前令牌立即失效，需要重新登录。 */
export function changeOwnPassword(oldPassword: string, newPassword: string) {
  return usePlatformHttp().put<{ changed: boolean }>('/platform/admins/me/password', {
    old_password: oldPassword,
    new_password: newPassword,
  })
}
