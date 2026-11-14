export default defineNuxtRouteMiddleware((to) => {
  // 园所后台的全局守卫先跑；这里只管平台路由树。
  if (!to.path.startsWith('/platform')) return
  if (to.path === '/platform/login') return
  const token = useCookie<string | null>('platform_token')
  if (!token.value) {
    return navigateTo({ path: '/platform/login', query: { redirect: to.fullPath } })
  }
})
