import { usePlatformHttp } from '~/composables/usePlatformHttp'

export function platformLogin(username: string, password: string) {
  return usePlatformHttp().post<{ token: string; expires_in: number; admin: { id: number; username: string; name: string } }>(
    '/platform/auth/login',
    { username, password },
  )
}

export function platformLogout() {
  return usePlatformHttp().post('/platform/auth/logout', {})
}
