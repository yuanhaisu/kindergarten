import { hasPermission } from '~/utils/permission'

export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.directive('permission', {
    mounted(el, binding) {
      const store = usePermissionStore()
      const need = binding.value as string | string[]
      if (!hasPermission(store.codes, need)) {
        el.parentNode?.removeChild(el)
      }
    },
  })
})
