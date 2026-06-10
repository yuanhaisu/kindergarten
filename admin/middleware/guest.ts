export default defineNuxtRouteMiddleware(() => {
  const token = useCookie<string | null>('admin_token')
  if (token.value) {
    return navigateTo('/')
  }
})
