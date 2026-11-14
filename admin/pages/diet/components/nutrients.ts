import type { DietFoodSource, DietGroup16, DietNutrients, DietSourceFlag } from '~/types/api'

/**
 * 膳食模块内共享的营养字段元数据与展示口径。
 * 食物库、菜谱库、周食谱编辑器都按同一份「27 项」清单渲染，
 * 避免各处各写一份后字段/单位漂移。
 */

export interface NutrientMeta {
  key: keyof DietNutrients
  label: string
  unit: string
}

export const NUTRIENT_GROUPS: { title: string; items: NutrientMeta[] }[] = [
  {
    title: '能量与宏量营养素',
    items: [
      { key: 'energy', label: '能量', unit: '千卡' },
      { key: 'heat', label: '热量', unit: '千焦' },
      { key: 'protein', label: '蛋白质', unit: '克' },
      { key: 'fats', label: '脂肪', unit: '克' },
      { key: 'carb', label: '碳水化合物', unit: '克' },
      { key: 'dietary_fiber', label: '不溶性纤维', unit: '克' },
      { key: 'ash', label: '灰分', unit: '克' },
      { key: 'water', label: '水分', unit: '克' },
      { key: 'cholesterol', label: '胆固醇', unit: '毫克' },
    ],
  },
  {
    title: '维生素',
    items: [
      { key: 'retinol', label: '视黄醇', unit: '微克' },
      { key: 'carotin', label: '胡萝卜素', unit: '微克' },
      { key: 'vitamin_a', label: '维生素A', unit: '微克 RE' },
      { key: 'vitamin_b1', label: '硫胺素（维生素B1）', unit: '毫克' },
      { key: 'vitamin_b2', label: '核黄素（维生素B2）', unit: '毫克' },
      { key: 'nicotinic_acid', label: '尼克酸（烟酸）', unit: '毫克' },
      { key: 'vitamin_c', label: '维生素C', unit: '毫克' },
      { key: 'vitamin_e', label: '维生素E', unit: '毫克 α-TE' },
    ],
  },
  {
    title: '矿物质',
    items: [
      { key: 'calcium', label: '钙', unit: '毫克' },
      { key: 'phosphorus', label: '磷', unit: '毫克' },
      { key: 'potassium', label: '钾', unit: '毫克' },
      { key: 'sodium', label: '钠', unit: '毫克' },
      { key: 'magnesium', label: '镁', unit: '毫克' },
      { key: 'iron', label: '铁', unit: '毫克' },
      { key: 'zinc', label: '锌', unit: '毫克' },
      { key: 'selenium', label: '硒', unit: '微克' },
      { key: 'copper', label: '铜', unit: '毫克' },
      { key: 'manganese', label: '锰', unit: '毫克' },
    ],
  },
]

export const ALL_NUTRIENTS: NutrientMeta[] = NUTRIENT_GROUPS.flatMap((g) => g.items)

export const FOOD_SOURCE_LABELS: Record<DietFoodSource, string> = {
  system: '平台预置',
  custom: '园所自建',
}

export const GROUP16_LABELS: Record<DietGroup16, string> = {
  grain: '谷类',
  tuber: '薯类',
  vegetable: '蔬菜',
  fruit: '水果',
  meat: '畜禽肉类',
  egg: '蛋类',
  aquatic: '水产类',
  soybean: '大豆',
  dairy: '乳制品',
  oil: '油脂类',
  nut: '坚果',
  instant_food: '速食小吃',
  beverage: '饮料',
  confection: '糖果蜜饯',
  condiment: '调味品',
  other: '其他',
}

export const SOURCE_FLAG_LABELS: Record<DietSourceFlag, string> = {
  animal: '动物性',
  plant: '植物性',
  milk: '乳类',
  meat: '肉类',
}

/** 缺测显示「—」，不得显示成 0 */
export function formatNutrient(value: number | null | undefined) {
  return value == null ? '—' : String(value)
}
