import { ADMIN_TOKEN_COOKIE } from '~/utils/authCookie'

export default defineNuxtRouteMiddleware(() => {
  const token = useCookie<string | null>(ADMIN_TOKEN_COOKIE)
  if (token.value) {
    return navigateTo('/')
  }
})
