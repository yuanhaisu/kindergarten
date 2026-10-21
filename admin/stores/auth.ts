import { defineStore } from 'pinia'
import type { AuthUser, LoginResult } from '~/types/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null)
  const refreshTokenValue = useCookie<string | null>('admin_refresh', { sameSite: 'lax' })

  function setSession(data: LoginResult) {
    const token = useCookie<string | null>('admin_token', {
      sameSite: 'lax',
      maxAge: 60 * 60 * 24 * 7,
    })
    token.value = data.token
    refreshTokenValue.value = data.refresh_token
    user.value = data.user
  }

  function clearSession() {
    const token = useCookie<string | null>('admin_token')
    token.value = null
    refreshTokenValue.value = null
    user.value = null
    usePermissionStore().reset()
  }

  async function fetchProfile() {
    const { getUserProfile } = await import('~/api/common')
    user.value = await getUserProfile()
    // 服务端下发的权限码同步进权限 store（菜单过滤与路由守卫唯一数据源）
    const codes = (user.value.roles || [])
      .filter((r) => r.role === 'admin')
      .flatMap((r) => r.permissions || [])
    usePermissionStore().setCodes(codes)
  }

  return { user, setSession, clearSession, fetchProfile }
})
