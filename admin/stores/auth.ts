import { defineStore } from 'pinia'
import type { AuthUser, LoginResult } from '~/types/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null)
  const refreshTokenValue = useCookie<string | null>('admin_refresh', { sameSite: 'lax' })

  // 隐私政策同意状态：登录响应先给一次，路由守卫再用 /common/consent 校准。
  const consentRequired = ref(false)
  const currentPolicyVersion = ref('')
  const consentChecked = ref(false)

  function setSession(data: LoginResult) {
    const token = useCookie<string | null>('admin_token', {
      sameSite: 'lax',
      maxAge: 60 * 60 * 24 * 7,
    })
    token.value = data.token
    refreshTokenValue.value = data.refresh_token
    user.value = data.user
    consentRequired.value = Boolean(data.consent_required)
    currentPolicyVersion.value = data.current_policy_version || ''
    consentChecked.value = false
  }

  function clearSession() {
    const token = useCookie<string | null>('admin_token')
    token.value = null
    refreshTokenValue.value = null
    user.value = null
    consentRequired.value = false
    currentPolicyVersion.value = ''
    consentChecked.value = false
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

  /** 返回 true 表示已同意当前政策。force 用于政策更新后重新校准。 */
  async function ensureConsent(force = false): Promise<boolean> {
    if (!force && consentChecked.value) return !consentRequired.value
    const { getConsentStatus } = await import('~/api/consent')
    const status = await getConsentStatus()
    consentChecked.value = true
    currentPolicyVersion.value = status.policy_version || currentPolicyVersion.value
    consentRequired.value = !status.consented
    return !consentRequired.value
  }

  function markConsented(status: { policy_version?: string }) {
    consentRequired.value = false
    consentChecked.value = true
    if (status?.policy_version) currentPolicyVersion.value = status.policy_version
  }

  return {
    user,
    consentRequired,
    currentPolicyVersion,
    setSession,
    clearSession,
    fetchProfile,
    ensureConsent,
    markConsented,
  }
})
