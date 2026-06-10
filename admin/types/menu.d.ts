export interface MenuItem {
  title: string
  path: string
  icon?: string
  permission?: string
  children?: MenuItem[]
}
