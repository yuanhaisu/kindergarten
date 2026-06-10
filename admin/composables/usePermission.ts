export function usePermission() {
  const store = usePermissionStore()
  return {
    can: store.can,
    codes: computed(() => store.codes),
  }
}
