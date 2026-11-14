import type { LoginPasswordBody, LoginResult } from '~/types/api'

export function loginPassword(body: LoginPasswordBody) {
  return useHttp().post<LoginResult>('/auth/login/password', body)
}

export function refreshToken(refresh_token: string) {
  return useHttp().post<LoginResult>('/auth/refresh', { refresh_token })
}

export function changePassword(body: { old_password: string; new_password: string }) {
  return useHttp().put<{ must_change_password: boolean }>('/common/user/password', body)
}
