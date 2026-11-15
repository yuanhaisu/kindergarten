import { PLATFORM_TOKEN_COOKIE } from '~/utils/authCookie'

export default defineNuxtRouteMiddleware((to) => {
  // 园所后台的全局守卫先跑；这里只管平台路由树。
  if (!to.path.startsWith('/platform')) return
  if (to.path === '/platform/login') return
  const token = useCookie<string | null>(PLATFORM_TOKEN_COOKIE)
  if (!token.value) {
    return navigateTo({ path: '/platform/login', query: { redirect: to.fullPath } })
  }
})
