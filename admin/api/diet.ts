import type {
  DietAnalysis,
  DietConfig,
  DietConfigUpdateBody,
  DietFoodBatchDeleteBody,
  DietFoodBatchDeleteResult,
  DietFoodCategory,
  DietFoodCreateBody,
  DietFoodDetail,
  DietFoodListQuery,
  DietFoodListResult,
  DietFoodUpdateBody,
  DietPlanCreateBody,
  DietPlanCreateResult,
  DietPlanClearResult,
  DietPlanDetail,
  DietPlanListQuery,
  DietPlanListResult,
  DietPlanSaveBody,
  DietPlanSaveResult,
  DietRecipeCreateBody,
  DietRecipeDetail,
  DietRecipeListQuery,
  DietRecipeListResult,
  DietRecipeMergeBody,
  DietRecipeMergeResult,
  DietRecipeSuggestQuery,
  DietRecipeSuggestResult,
  DietRecipeUpdateBody,
  DietReplaceCandidateQuery,
  DietReplaceCandidateResult,
  DietTemplateDetail,
  DietTemplateListResult,
} from '~/types/api'

/**
 * 膳食管理接口（契约：backend/api/api-admin/23-diet.md，27 个）。
 * 统一走 useHttp，响应由拦截器解包；错误 message 直接展示给用户。
 */

// ===== 供餐配置 =====

export function getDietConfig() {
  return useHttp().get<DietConfig>('/admin/diet/config')
}

export function updateDietConfig(body: DietConfigUpdateBody) {
  return useHttp().put<DietConfig>('/admin/diet/config', body)
}

// ===== 食物分类 =====

export function getDietFoodCategories() {
  return useHttp().get<{ list: DietFoodCategory[] }>('/admin/diet/food-categories')
}

// ===== 食物库 =====

export function getDietFoods(params: DietFoodListQuery) {
  return useHttp().get<DietFoodListResult>('/admin/diet/foods', { params })
}

export function createDietFood(body: DietFoodCreateBody) {
  return useHttp().post<{ id: number; name: string; spell_code: string }>('/admin/diet/foods', body)
}

export function getDietFood(id: number) {
  return useHttp().get<DietFoodDetail>(`/admin/diet/foods/${id}`)
}

export function updateDietFood(id: number, body: DietFoodUpdateBody) {
  return useHttp().put<{ id: number }>(`/admin/diet/foods/${id}`, body)
}

export function deleteDietFood(id: number) {
  return useHttp().delete(`/admin/diet/foods/${id}`)
}

export function batchDeleteDietFoods(body: DietFoodBatchDeleteBody) {
  return useHttp().post<DietFoodBatchDeleteResult>('/admin/diet/foods/batch-delete', body)
}

export function getDietFoodReplaceCandidates(id: number, params: DietReplaceCandidateQuery) {
  return useHttp().get<DietReplaceCandidateResult>(`/admin/diet/foods/${id}/replace-candidates`, {
    params,
  })
}

// ===== 菜谱库 =====

export function getDietRecipes(params: DietRecipeListQuery) {
  return useHttp().get<DietRecipeListResult>('/admin/diet/recipes', { params })
}

export function createDietRecipe(body: DietRecipeCreateBody) {
  return useHttp().post<{ id: number; name: string; spell_code: string; item_count: number }>(
    '/admin/diet/recipes',
    body,
  )
}

export function getDietRecipe(id: number) {
  return useHttp().get<DietRecipeDetail>(`/admin/diet/recipes/${id}`)
}

export function updateDietRecipe(id: number, body: DietRecipeUpdateBody) {
  return useHttp().put<{ id: number }>(`/admin/diet/recipes/${id}`, body)
}

export function deleteDietRecipe(id: number) {
  return useHttp().delete(`/admin/diet/recipes/${id}`)
}

export function mergeDietRecipes(body: DietRecipeMergeBody) {
  return useHttp().post<DietRecipeMergeResult>('/admin/diet/recipes/merge', body)
}

export function suggestDietRecipes(params: DietRecipeSuggestQuery) {
  return useHttp().get<DietRecipeSuggestResult>('/admin/diet/recipes/suggest', { params })
}

// ===== 周食谱 =====

export function getDietPlans(params: DietPlanListQuery) {
  return useHttp().get<DietPlanListResult>('/admin/diet/plans', { params })
}

export function createDietPlan(body: DietPlanCreateBody) {
  return useHttp().post<DietPlanCreateResult>('/admin/diet/plans', body)
}

export function getDietPlan(id: number) {
  return useHttp().get<DietPlanDetail>(`/admin/diet/plans/${id}`)
}

export function updateDietPlan(id: number, body: DietPlanSaveBody) {
  return useHttp().put<DietPlanSaveResult>(`/admin/diet/plans/${id}`, body)
}

export function deleteDietPlan(id: number) {
  return useHttp().delete(`/admin/diet/plans/${id}`)
}

export function clearDietPlan(id: number) {
  return useHttp().post<DietPlanClearResult>(`/admin/diet/plans/${id}/clear`)
}

// ===== 模板食谱 =====

export function getDietTemplates() {
  return useHttp().get<DietTemplateListResult>('/admin/diet/templates')
}

export function getDietTemplate(id: number) {
  return useHttp().get<DietTemplateDetail>(`/admin/diet/templates/${id}`)
}

// ===== 营养分析 =====

export function getDietPlanAnalysis(id: number) {
  return useHttp().get<DietAnalysis>(`/admin/diet/plans/${id}/analysis`)
}

/** 导出走 getBlob（带 Bearer），不得用 window.open */
export async function exportDietPlanAnalysis(id: number) {
  const res = await useHttp().getBlob(`/admin/diet/plans/${id}/analysis/export`)
  return res.data as Blob
}
