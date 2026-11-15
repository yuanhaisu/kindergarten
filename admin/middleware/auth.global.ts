import { ADMIN_TOKEN_COOKIE } from '~/utils/authCookie'

export default defineNuxtRouteMiddleware(async (to) => {
  // 平台方是独立信任域：/platform/* 由 platform-auth.global.ts 守着。
  if (to.path.startsWith('/platform')) return
  if (to.path === '/login') return
  const token = useCookie<string | null>(ADMIN_TOKEN_COOKIE)
  if (!token.value) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }
  // 阻断页自身不再做同意检查，避免循环。
  if (to.path === '/consent') return

  const auth = useAuthStore()
  try {
    const ok = await auth.ensureConsent()
    if (!ok) {
      return navigateTo({ path: '/consent', query: { redirect: to.fullPath } })
    }
  } catch {
    // 网络异常不阻断导航；服务端 403 会在业务请求时兜底并跳到 /consent。
  }
})
