import { ADMIN_CHANGE_PASSWORD_PAGE_PATH, ADMIN_CONSENT_PAGE_PATH } from '~/utils/authPaths'

/** 登录/同意后仍需跨过的门槛；顺序即契约顺序：先同意，再改密。 */
export interface AuthGateState {
  consentRequired?: boolean
  mustChangePassword?: boolean
}

export interface AuthGateDestination {
  path: string
  query?: { redirect: string }
}

/**
 * 把「还差哪些门槛」翻译成下一跳。
 *
 * 同意必须排在改密前（#160/#198）：未同意时服务端只放行同意相关接口，
 * 直接去改密页会被 40301 弹回，来回跳。
 */
export function resolveAuthGate(
  state: AuthGateState,
  redirect = '/',
): AuthGateDestination {
  if (state.consentRequired) {
    return { path: ADMIN_CONSENT_PAGE_PATH, query: { redirect } }
  }
  if (state.mustChangePassword) {
    return { path: ADMIN_CHANGE_PASSWORD_PAGE_PATH, query: { redirect } }
  }
  return { path: redirect }
}
