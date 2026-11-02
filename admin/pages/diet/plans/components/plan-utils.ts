import type { DietNutrients } from '~/types/api'
import {
  CORE_NUTRIENTS,
  type CoreTotals,
  type DishDraft,
  type FoodDraft,
  type WeekDay,
} from './plan-types'

/** 日期只按本地日历运算，避免 `new Date('YYYY-MM-DD')` 按 UTC 解析导致跨日偏移。 */
export const WEEKDAY_LABELS = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'] as const

export function parseISODate(iso: string): Date {
  const [year, month, day] = iso.split('-').map(Number)
  return new Date(year || 1970, (month || 1) - 1, day || 1)
}

export function formatISODate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function addDaysISO(iso: string, days: number): string {
  const date = parseISODate(iso)
  date.setDate(date.getDate() + days)
  return formatISODate(date)
}

/** 1=周一 … 7=周日（契约 weekday 口径） */
export function isoWeekday(iso: string): number {
  const day = parseISODate(iso).getDay()
  return day === 0 ? 7 : day
}

export function isMondayISO(iso: string): boolean {
  return isoWeekday(iso) === 1
}

/** 含今天在内的下一个周一 */
export function nextMondayISO(base = new Date()): string {
  const day = base.getDay()
  const diff = (8 - day) % 7
  const date = new Date(base.getFullYear(), base.getMonth(), base.getDate() + diff)
  return formatISODate(date)
}

export function buildWeekDays(startISO: string): WeekDay[] {
  if (!startISO) return []
  return Array.from({ length: 7 }, (_, index) => {
    const date = addDaysISO(startISO, index)
    return { date, weekday: index + 1, label: WEEKDAY_LABELS[index] }
  })
}

export function emptyCoreTotals(): CoreTotals {
  return {
    energy: null,
    protein: null,
    fats: null,
    carb: null,
    calcium: null,
    iron: null,
  }
}

interface EffectiveNutrition {
  nutrients: DietNutrients
  edible_percent: number | null
}

/**
 * 小计口径（brief §6.7）：
 * - 已保存且未改动的行用 `nutrition_snapshot`（保存当时浓度，不随食物库变化）；
 * - 新增 / 改克数 / 同类替换后的行用当前食物详情，与服务端保存时重新固化的快照一致。
 * 详情尚未返回时回退到快照，避免编辑过程中小计闪烁为「—」。
 */
export function effectiveNutrition(food: FoodDraft): EffectiveNutrition | null {
  if (food.dirty) {
    if (food.detail) {
      return { nutrients: food.detail.nutrients, edible_percent: food.detail.edible_percent }
    }
    if (food.snapshot) {
      return { nutrients: food.snapshot, edible_percent: food.snapshot.edible_percent }
    }
    return null
  }
  if (food.snapshot) {
    return { nutrients: food.snapshot, edible_percent: food.snapshot.edible_percent }
  }
  if (food.detail) {
    return { nutrients: food.detail.nutrients, edible_percent: food.detail.edible_percent }
  }
  return null
}

/** 单行贡献 = 克数 × 食部% × 每 100g 浓度 / 100；缺测返回 null（不按 0 计入）。 */
export function foodCoreTotals(food: FoodDraft): CoreTotals {
  const effective = effectiveNutrition(food)
  const grams = typeof food.grams === 'number' && Number.isFinite(food.grams) ? food.grams : 0
  const edible = effective?.edible_percent == null ? null : effective.edible_percent / 100
  const out = emptyCoreTotals()

  for (const meta of CORE_NUTRIENTS) {
    const per100g = effective?.nutrients?.[meta.key]
    out[meta.key] =
      per100g == null || edible == null ? null : (grams * edible * per100g) / 100
  }
  return out
}

export function sumCoreTotals(list: CoreTotals[]): CoreTotals {
  const out = emptyCoreTotals()
  for (const meta of CORE_NUTRIENTS) {
    let sum = 0
    let seen = false
    for (const item of list) {
      const value = item[meta.key]
      if (value == null || !Number.isFinite(value)) continue
      sum += value
      seen = true
    }
    out[meta.key] = seen ? sum : null
  }
  return out
}

export function dishesCoreTotals(dishes: DishDraft[]): CoreTotals {
  return sumCoreTotals(dishes.flatMap((dish) => dish.foods.map((food) => foodCoreTotals(food))))
}

/** 缺测显示「—」；1 位小数，整数不补 0 */
export function formatCore(value: number | null): string {
  if (value == null || !Number.isFinite(value)) return '—'
  return String(Math.round(value * 10) / 10)
}
