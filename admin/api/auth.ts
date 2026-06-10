import type { LoginPasswordBody, LoginResult } from '~/types/api'

export function loginPassword(body: LoginPasswordBody) {
  return useHttp().post<LoginResult>('/auth/login/password', body)
}

export function refreshToken(refresh_token: string) {
  return useHttp().post<LoginResult>('/auth/refresh', { refresh_token })
}
