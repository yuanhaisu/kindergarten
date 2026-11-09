export default defineNuxtRouteMiddleware((to) => {
  // 平台方是独立信任域：/platform/* 由 platform-auth.global.ts 守着。
  if (to.path.startsWith('/platform')) return
  if (to.path === '/login') return
  const token = useCookie<string | null>('admin_token')
  if (!token.value) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }
})
