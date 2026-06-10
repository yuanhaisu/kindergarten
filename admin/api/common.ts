import type { AuthUser } from '~/types/api'

/** 当前用户资料（三端通用） */
export function getUserProfile() {
  return useHttp().get<AuthUser & { created_at?: string }>('/common/user/profile')
}
