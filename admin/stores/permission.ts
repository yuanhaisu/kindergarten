import { defineStore } from 'pinia'
import { hasPermission as check } from '~/utils/permission'

/**
 * 权限码列表 —— 由服务端 profile 下发（`roles[].permissions`，通配为 ['*']）。
 *
 * 默认 ['*'] 仅代表「尚未加载」的放开态，避免启动瞬间闪空菜单；
 * `setCodes` 一旦被调用（登录/取资料后），以服务端码集为准：
 * **空集就是空集** —— 零权限账号必须全隐藏（spec R5），不再回落为放开。
 */
export const usePermissionStore = defineStore('permission', () => {
  const codes = ref<string[]>(['*'])
  /** 码集是否已被服务端覆盖；未覆盖时路由守卫会先拉取 profile */
  const loaded = ref(false)

  function setCodes(next: string[]) {
    codes.value = [...next]
    loaded.value = true
  }

  /** 退出登录时复位为未加载态，防止残留上一账号的码集 */
  function reset() {
    codes.value = ['*']
    loaded.value = false
  }

  function can(need: string | string[]) {
    return check(codes.value, need)
  }

  return { codes, loaded, setCodes, reset, can }
})
