import { ADMIN_MENU_GROUPS, matchMenuByPath } from '~/constants/menus'

/**
 * 路由守卫：所需权限码从菜单表派生（matchMenuByPath），与侧栏过滤同源——
 * 菜单可见与路由可达不允许分叉（wire-admin-permissions design D5）。
 *
 * 执行顺序在 auth.global 之后（文件名字典序），此时未登录已先跳 /login。
 */

/** 菜单顺序里第一个当前账号可访问的页面路径 */
function firstAccessiblePath(can: (need: string) => boolean): string | null {
  for (const g of ADMIN_MENU_GROUPS) {
    for (const m of g.children) {
      if (!m.permission || can(m.permission)) return m.path
    }
  }
  return null
}

export default defineNuxtRouteMiddleware(async (to) => {
  if (to.path === '/login') return
  const token = useCookie<string | null>('admin_token')
  if (!token.value) return

  const perm = usePermissionStore()
  const auth = useAuthStore()

  // 首次进入：先用服务端码集覆盖未加载的默认态，再判定
  if (!perm.loaded) {
    try {
      await auth.fetchProfile()
    } catch (e: unknown) {
      // 不静默：记录后放行，由页面接口的 403 兜底提示；不做锁定式跳转
      console.error('[permission] 获取用户资料失败，本次跳过权限判定', e)
      return
    }
  }

  const entry = matchMenuByPath(to.path)
  if (!entry?.permission) return
  if (perm.can(entry.permission)) return

  const first = firstAccessiblePath((need) => perm.can(need))
  if (first && first !== to.path) return navigateTo(first)
  return navigateTo('/no-permission')
})
