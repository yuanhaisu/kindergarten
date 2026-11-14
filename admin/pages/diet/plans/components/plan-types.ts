import type { DietFoodDetail, DietMealKey, DietNutritionSnapshot } from '~/types/api'

/**
 * 周食谱编辑器共享类型。
 *
 * 编辑器的真相是「界面顺序 = sort」：菜在单元格内、食物在菜内的数组顺序就是保存顺序，
 * 因此这里不保存 sort 字段，移动操作只调整数组。
 */

/** 食物行草稿。snapshot 是已保存行的历史快照，detail 是新增/改克数/替换后拉到的当前食物数据。 */
export interface FoodDraft {
  key: string
  food_id: number
  food_name: string
  grams: number | null | undefined
  snapshot: DietNutritionSnapshot | null
  detail: DietFoodDetail | null
  /** true 时小计与保存口径都改用食物详情（服务端保存时会重新固化快照） */
  dirty: boolean
  loadingDetail: boolean
}

export interface DishDraft {
  key: string
  name: string
  recipe_id: number | null
  foods: FoodDraft[]
}

export interface MealRow {
  meal_key: DietMealKey
  name: string
  time: string
  sort: number
}

export interface WeekDay {
  date: string
  weekday: number
  label: string
}

export interface AcademicYearOption {
  id: number
  year: string
  is_current: boolean
}

export interface DishDraftPayload {
  name: string
  recipe_id: number | null
  foods: FoodDraft[]
}

export const CORE_NUTRIENTS = [
  { key: 'energy', label: '能量', unit: '千卡' },
  { key: 'protein', label: '蛋白质', unit: '克' },
  { key: 'fats', label: '脂肪', unit: '克' },
  { key: 'carb', label: '碳水', unit: '克' },
  { key: 'calcium', label: '钙', unit: '毫克' },
  { key: 'iron', label: '铁', unit: '毫克' },
] as const

export type CoreKey = (typeof CORE_NUTRIENTS)[number]['key']
export type CoreTotals = Record<CoreKey, number | null>

let keySeed = 0

/** 生成仅用于本地渲染/高亮的 key；不参与保存，也不做稳定排序依据。 */
export function nextKey(prefix: string): string {
  keySeed += 1
  return `${prefix}${keySeed}`
}

export function cellKey(date: string, mealKey: DietMealKey): string {
  return `${date}|${mealKey}`
}
