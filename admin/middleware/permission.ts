export default defineNuxtRouteMiddleware((to) => {
  const perm = to.meta.permission as string | string[] | undefined
  if (!perm) return
  const store = usePermissionStore()
  if (!store.can(perm)) {
    return navigateTo('/')
  }
})
