export interface Paged<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}
