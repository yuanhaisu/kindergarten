export function usePagination(initial?: { page?: number; page_size?: number }) {
  const page = ref(initial?.page ?? 1)
  const pageSize = ref(initial?.page_size ?? 20)
  const total = ref(0)

  function resetPage() {
    page.value = 1
  }

  return { page, pageSize, total, resetPage }
}
