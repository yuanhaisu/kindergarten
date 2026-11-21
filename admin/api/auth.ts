import type { LoginPasswordBody, LoginResult } from '~/types/api'
import { ADMIN_AUTH_LOGIN_PATH, ADMIN_AUTH_REFRESH_PATH } from '~/utils/authPaths'

export function loginPassword(body: LoginPasswordBody) {
  return useHttp().post<LoginResult>(ADMIN_AUTH_LOGIN_PATH, body, { skipAdminAuthRefresh: true })
}

export function refreshToken(refresh_token: string) {
  return useHttp().post<LoginResult>(
    ADMIN_AUTH_REFRESH_PATH,
    { refresh_token },
    { skipAdminAuthRefresh: true },
  )
}

export function changePassword(body: { old_password: string; new_password: string }) {
  return useHttp().put<{ must_change_password: boolean }>('/common/user/password', body)
}
