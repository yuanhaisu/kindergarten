import { defineStore } from 'pinia'
import { hasPermission as check } from '~/utils/permission'

/**
 * 权限码列表。若后端登录/资料接口未返回 permissions，默认放开（*）。
 * 接入真实权限后：在登录成功或 fetchProfile 后 setCodes(...)
 */
export const usePermissionStore = defineStore('permission', () => {
  const codes = ref<string[]>(['*'])

  function setCodes(next: string[]) {
    codes.value = next.length ? next : ['*']
  }

  function can(need: string | string[]) {
    return check(codes.value, need)
  }

  return { codes, setCodes, can }
})
