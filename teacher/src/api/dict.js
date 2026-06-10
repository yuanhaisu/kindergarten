import { requestWithAuth } from "@/api/http"

/**
 * 获取指定类型的字典项（仅返回启用的项，按 sort_order 排序）
 * @param {string} typeKey - 字典类型标识，如 "performance_tag"
 * @returns {Promise<{data: {type_key: string, items: Array<{item_key: string, item_label: string, extra?: any, sort_order: number}>}>}
 */
export function fetchDict(typeKey) {
  return requestWithAuth({ url: `/teacher/dicts/${typeKey}`, method: "GET" })
}

/**
 * 批量获取多个字典类型
 * @param {string[]} typeKeys - 字典类型标识数组
 * @returns {Promise<{data: {dicts: Record<string, Array>}>}
 */
export function fetchDictsBatch(typeKeys) {
  const types = Array.isArray(typeKeys) ? typeKeys.join(",") : typeKeys
  return requestWithAuth({ url: `/teacher/dicts?types=${types}`, method: "GET" })
}
