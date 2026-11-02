package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

// dietConfigRequest 是 PUT /admin/diet/config 的请求体（契约 23-diet.md）。
// name/sort/coefficient 只读，客户端传了也不采纳。
type dietConfigRequest struct {
	CareType      domain.DietCareType          `json:"care_type"`
	FoodInputType domain.DietFoodInputType     `json:"food_input_type"`
	RatioMode     domain.DietRatioMode         `json:"ratio_mode"`
	Meals         []dietConfigRequestMealInput `json:"meals"`
}

type dietConfigRequestMealInput struct {
	MealKey       domain.DietMealKey `json:"meal_key"`
	Time          string             `json:"time"`
	EnergyPercent float64            `json:"energy_percent"`
	Enabled       bool               `json:"enabled"`
}

func (r dietConfigRequest) toInput() service.DietMealConfigInput {
	meals := make([]service.DietMealConfigMealInput, 0, len(r.Meals))
	for _, meal := range r.Meals {
		meals = append(meals, service.DietMealConfigMealInput{
			MealKey:       meal.MealKey,
			Time:          meal.Time,
			EnergyPercent: meal.EnergyPercent,
			Enabled:       meal.Enabled,
		})
	}
	return service.DietMealConfigInput{
		CareType:      r.CareType,
		FoodInputType: r.FoodInputType,
		RatioMode:     r.RatioMode,
		Meals:         meals,
	}
}

// NewDietAdminHandler 组装膳食模块（配置 + 食物库 + 菜谱库）的 Handler，供路由级测试使用。
// 生产路径仍由 NewHandler 全量注入；这里避免构造数十个无关 service。
func NewDietAdminHandler(
	dietConfigSvc *service.DietConfigService,
	dietFoodSvc *service.DietFoodService,
	dietRecipeSvc *service.DietRecipeService,
) *Handler {
	return &Handler{
		dietConfigSvc: dietConfigSvc,
		dietFoodSvc:   dietFoodSvc,
		dietRecipeSvc: dietRecipeSvc,
	}
}

// GetDietConfig 返回园所供餐配置；未配置时返回默认值。
func (h *Handler) GetDietConfig(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.dietConfigSvc.GetConfig(c.Request.Context(), kgID)
	if err != nil {
		respondDietConfigError(c, err)
		return
	}
	response.OK(c, data)
}

// PutDietConfig 保存园所供餐配置（整组替换餐次），响应与 GET 同构。
func (h *Handler) PutDietConfig(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req dietConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	data, err := h.dietConfigSvc.SaveConfig(c.Request.Context(), kgID, req.toInput())
	if err != nil {
		respondDietConfigError(c, err)
		return
	}
	response.OK(c, data)
}

// respondDietConfigError 把 service 错误映射到项目统一错误包络。
// 业务校验失败返回 422 且带可读 message；仓储等内部错误不向外泄露细节。
func respondDietConfigError(c *gin.Context, err error) {
	var invalid *service.DietConfigValidationError
	if errors.As(err, &invalid) {
		response.ErrValidation(c, invalid.Message)
		return
	}
	response.ErrInternal(c, "膳食配置服务异常")
}

// --- 食物库（契约 23-diet.md 第二、三节） ---

// dietFoodNutrientsRequest 是 27 项营养的请求体；未提交或提交 null 都写 NULL。
type dietFoodNutrientsRequest struct {
	Energy        *float64 `json:"energy"`
	Heat          *float64 `json:"heat"`
	Protein       *float64 `json:"protein"`
	Fats          *float64 `json:"fats"`
	Carb          *float64 `json:"carb"`
	DietaryFiber  *float64 `json:"dietary_fiber"`
	Ash           *float64 `json:"ash"`
	Water         *float64 `json:"water"`
	Cholesterol   *float64 `json:"cholesterol"`
	Retinol       *float64 `json:"retinol"`
	Carotin       *float64 `json:"carotin"`
	VitaminA      *float64 `json:"vitamin_a"`
	VitaminB1     *float64 `json:"vitamin_b1"`
	VitaminB2     *float64 `json:"vitamin_b2"`
	NicotinicAcid *float64 `json:"nicotinic_acid"`
	VitaminC      *float64 `json:"vitamin_c"`
	VitaminE      *float64 `json:"vitamin_e"`
	Calcium       *float64 `json:"calcium"`
	Phosphorus    *float64 `json:"phosphorus"`
	Potassium     *float64 `json:"potassium"`
	Sodium        *float64 `json:"sodium"`
	Magnesium     *float64 `json:"magnesium"`
	Iron          *float64 `json:"iron"`
	Zinc          *float64 `json:"zinc"`
	Selenium      *float64 `json:"selenium"`
	Copper        *float64 `json:"copper"`
	Manganese     *float64 `json:"manganese"`
	// EdiblePercent 只用于识别并拒绝契约禁止的写法（食部在顶层）。
	EdiblePercent *float64 `json:"edible_percent"`
}

func (r dietFoodNutrientsRequest) toInput() service.DietFoodNutrientsInput {
	return service.DietFoodNutrientsInput{
		Energy:        r.Energy,
		Heat:          r.Heat,
		Protein:       r.Protein,
		Fats:          r.Fats,
		Carb:          r.Carb,
		DietaryFiber:  r.DietaryFiber,
		Ash:           r.Ash,
		Water:         r.Water,
		Cholesterol:   r.Cholesterol,
		Retinol:       r.Retinol,
		Carotin:       r.Carotin,
		VitaminA:      r.VitaminA,
		VitaminB1:     r.VitaminB1,
		VitaminB2:     r.VitaminB2,
		NicotinicAcid: r.NicotinicAcid,
		VitaminC:      r.VitaminC,
		VitaminE:      r.VitaminE,
		Calcium:       r.Calcium,
		Phosphorus:    r.Phosphorus,
		Potassium:     r.Potassium,
		Sodium:        r.Sodium,
		Magnesium:     r.Magnesium,
		Iron:          r.Iron,
		Zinc:          r.Zinc,
		Selenium:      r.Selenium,
		Copper:        r.Copper,
		Manganese:     r.Manganese,
		EdiblePercent: r.EdiblePercent,
	}
}

// dietFoodRequest 是 POST /foods 与 PUT /foods/:id 的公共请求体。
// 修改时字段均可选（指针区分「未提交」），新建时由 service 校验必填项。
type dietFoodRequest struct {
	Name          *string                   `json:"name"`
	CategoryID    *uint64                   `json:"category_id"`
	EdiblePercent *float64                  `json:"edible_percent"`
	Group12       *int                      `json:"group12"`
	Group16       *domain.DietFoodGroup16   `json:"group16"`
	SourceFlags   []string                  `json:"source_flags"`
	Nutrients     *dietFoodNutrientsRequest `json:"nutrients"`
}

func (r dietFoodRequest) toInput() service.DietFoodInput {
	var nutrients *service.DietFoodNutrientsInput
	if r.Nutrients != nil {
		in := r.Nutrients.toInput()
		nutrients = &in
	}
	return service.DietFoodInput{
		Name:          r.Name,
		CategoryID:    r.CategoryID,
		EdiblePercent: r.EdiblePercent,
		Group12:       r.Group12,
		Group16:       r.Group16,
		SourceFlags:   r.SourceFlags,
		Nutrients:     nutrients,
	}
}

type listDietFoodsQuery struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	CategoryID uint64 `form:"category_id"`
	Keyword    string `form:"keyword"`
}

type dietFoodReplaceCandidatesQuery struct {
	Keyword string `form:"keyword"`
	Limit   int    `form:"limit"`
}

type dietFoodBatchDeleteRequest struct {
	IDs []uint64 `json:"ids"`
}

// ListDietFoodCategories 返回全局两级食物分类树。
func (h *Handler) ListDietFoodCategories(c *gin.Context) {
	if middleware.GetKindergartenID(c) == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	list, err := h.dietFoodSvc.ListCategories(c.Request.Context())
	if err != nil {
		response.ErrInternal(c, "获取食物分类失败")
		return
	}
	response.OK(c, map[string]any{"list": list})
}

// ListDietFoods 返回本园食物列表（列表列裁剪，不含 27 项全量）。
func (h *Handler) ListDietFoods(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listDietFoodsQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	list, total, page, pageSize, err := h.dietFoodSvc.ListFoods(
		c.Request.Context(), kgID, q.CategoryID, q.Keyword, pq,
	)
	if err != nil {
		respondDietFoodError(c, err, "获取食物列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

// GetDietFood 返回食物详情（27 项全量，缺失值为 null）。
func (h *Handler) GetDietFood(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietFoodID(c)
	if !ok {
		return
	}
	data, err := h.dietFoodSvc.GetFood(c.Request.Context(), kgID, id)
	if err != nil {
		respondDietFoodError(c, err, "获取食物详情失败")
		return
	}
	response.OK(c, data)
}

// CreateDietFood 新增园所自建食物，HTTP 201。
func (h *Handler) CreateDietFood(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req dietFoodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	data, err := h.dietFoodSvc.CreateFood(c.Request.Context(), kgID, req.toInput())
	if err != nil {
		respondDietFoodError(c, err, "新增食物失败")
		return
	}
	response.Created(c, data)
}

// UpdateDietFood 修改园所食物；对预置副本的修改会标记 is_customized。
func (h *Handler) UpdateDietFood(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietFoodID(c)
	if !ok {
		return
	}
	var req dietFoodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	if err := h.dietFoodSvc.UpdateFood(c.Request.Context(), kgID, id, req.toInput()); err != nil {
		respondDietFoodError(c, err, "修改食物失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

// DeleteDietFood 软删除园所食物；被菜谱引用时返回 409。
func (h *Handler) DeleteDietFood(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietFoodID(c)
	if !ok {
		return
	}
	if err := h.dietFoodSvc.DeleteFood(c.Request.Context(), kgID, id); err != nil {
		respondDietFoodError(c, err, "删除食物失败")
		return
	}
	response.OK(c, map[string]any{})
}

// BatchDeleteDietFoods 批量删除：逐条判定，返回 deleted 与 skipped。
func (h *Handler) BatchDeleteDietFoods(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req dietFoodBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	data, err := h.dietFoodSvc.BatchDeleteFoods(c.Request.Context(), kgID, req.IDs)
	if err != nil {
		respondDietFoodError(c, err, "批量删除食物失败")
		return
	}
	response.OK(c, data)
}

// ListDietFoodReplaceCandidates 返回同一一级分类下的其他本园食物。
func (h *Handler) ListDietFoodReplaceCandidates(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietFoodID(c)
	if !ok {
		return
	}
	var q dietFoodReplaceCandidatesQuery
	_ = c.ShouldBindQuery(&q)
	list, err := h.dietFoodSvc.ListReplaceCandidates(c.Request.Context(), kgID, id, q.Keyword, q.Limit)
	if err != nil {
		respondDietFoodError(c, err, "获取同类替换候选失败")
		return
	}
	response.OK(c, map[string]any{"list": list})
}

func dietFoodID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的食物 ID")
		return 0, false
	}
	return id, true
}

// respondDietFoodError 把食物服务错误映射到 404/409/422 契约包络。
func respondDietFoodError(c *gin.Context, err error, fallback string) {
	var invalid *service.DietFoodValidationError
	switch {
	case errors.As(err, &invalid):
		response.ErrValidation(c, invalid.Message)
	case errors.Is(err, service.ErrDietFoodNotFound):
		response.ErrNotFound(c, "食物不存在")
	case errors.Is(err, service.ErrDietFoodReferencedByRecipe):
		response.ErrConflict(c, "食物仍被菜谱引用，无法删除")
	default:
		response.ErrInternal(c, fallback)
	}
}

// --- 菜谱库（契约 23-diet.md 第四节） ---

type dietRecipeItemRequest struct {
	FoodID uint64  `json:"food_id"`
	Grams  float64 `json:"grams"`
}

func toDietRecipeItemInputs(rows []dietRecipeItemRequest) []service.DietRecipeItemInput {
	out := make([]service.DietRecipeItemInput, 0, len(rows))
	for _, row := range rows {
		out = append(out, service.DietRecipeItemInput{FoodID: row.FoodID, Grams: row.Grams})
	}
	return out
}

// dietRecipeSaveRequest 是 POST /recipes 的请求体。
// replace 只影响同名分支；缺省 false = 触发前端「替换 / 仅本次」询问。
type dietRecipeSaveRequest struct {
	Name    string                  `json:"name"`
	Items   []dietRecipeItemRequest `json:"items"`
	Replace bool                    `json:"replace"`
}

// dietRecipeUpdateRequest 是 PUT /recipes/:id 的请求体。
// name/items 均可选（改名与/或改配方）；items 一旦提交即整体替换。
type dietRecipeUpdateRequest struct {
	Name  *string                  `json:"name"`
	Items *[]dietRecipeItemRequest `json:"items"`
}

type dietRecipeMergeRequest struct {
	SourceID uint64 `json:"source_id"`
	TargetID uint64 `json:"target_id"`
}

type listDietRecipesQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
}

type dietRecipeSuggestQuery struct {
	Keyword string `form:"keyword"`
	Limit   int    `form:"limit"`
}

// ListDietRecipes 返回本园菜谱列表（名称/拼音码/配料数）。
func (h *Handler) ListDietRecipes(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listDietRecipesQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	list, total, page, pageSize, err := h.dietRecipeSvc.ListRecipes(
		c.Request.Context(), kgID, q.Keyword, pq,
	)
	if err != nil {
		respondDietRecipeError(c, err, "获取菜谱列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

// GetDietRecipe 返回菜谱详情（配料 + 每种食物当前 27 项营养）。
func (h *Handler) GetDietRecipe(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietRecipeID(c)
	if !ok {
		return
	}
	data, err := h.dietRecipeSvc.GetRecipe(c.Request.Context(), kgID, id)
	if err != nil {
		respondDietRecipeError(c, err, "获取菜谱详情失败")
		return
	}
	response.OK(c, data)
}

// CreateDietRecipe 保存菜谱：新增返回 201，replace=true 覆盖原菜谱返回 200；
// 同名且 replace=false 返回 409（前端据此弹「替换 / 仅本次」）。
func (h *Handler) CreateDietRecipe(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req dietRecipeSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	data, err := h.dietRecipeSvc.SaveRecipe(c.Request.Context(), kgID, service.DietRecipeSaveInput{
		Name:    req.Name,
		Items:   toDietRecipeItemInputs(req.Items),
		Replace: req.Replace,
	})
	if err != nil {
		respondDietRecipeError(c, err, "保存菜谱失败")
		return
	}
	if data.Created {
		response.Created(c, data)
		return
	}
	response.OK(c, data)
}

// UpdateDietRecipe 改名与/或改配方；改名撞上其他菜谱返回 409。
func (h *Handler) UpdateDietRecipe(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietRecipeID(c)
	if !ok {
		return
	}
	var req dietRecipeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	in := service.DietRecipeUpdateInput{Name: req.Name}
	if req.Items != nil {
		items := toDietRecipeItemInputs(*req.Items)
		in.Items = &items
	}
	if err := h.dietRecipeSvc.UpdateRecipe(c.Request.Context(), kgID, id, in); err != nil {
		respondDietRecipeError(c, err, "修改菜谱失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

// DeleteDietRecipe 软删除菜谱；历史周食谱保持原菜名、克数与营养快照。
func (h *Handler) DeleteDietRecipe(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietRecipeID(c)
	if !ok {
		return
	}
	if err := h.dietRecipeSvc.DeleteRecipe(c.Request.Context(), kgID, id); err != nil {
		respondDietRecipeError(c, err, "删除菜谱失败")
		return
	}
	response.OK(c, map[string]any{})
}

// MergeDietRecipes 合并重复菜谱：源的同种食物克数累加到目标，然后删除源。
func (h *Handler) MergeDietRecipes(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req dietRecipeMergeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	data, err := h.dietRecipeSvc.MergeRecipes(c.Request.Context(), kgID, req.SourceID, req.TargetID)
	if err != nil {
		respondDietRecipeError(c, err, "合并菜谱失败")
		return
	}
	response.OK(c, data)
}

// SuggestDietRecipes 编辑器菜名联想：名称包含或拼音码前缀匹配，不含配料。
func (h *Handler) SuggestDietRecipes(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q dietRecipeSuggestQuery
	_ = c.ShouldBindQuery(&q)
	list, err := h.dietRecipeSvc.SuggestRecipes(c.Request.Context(), kgID, q.Keyword, q.Limit)
	if err != nil {
		respondDietRecipeError(c, err, "获取菜谱联想失败")
		return
	}
	response.OK(c, map[string]any{"list": list})
}

func dietRecipeID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的菜谱 ID")
		return 0, false
	}
	return id, true
}

// respondDietRecipeError 把菜谱服务错误映射到 404/409/422 契约包络。
func respondDietRecipeError(c *gin.Context, err error, fallback string) {
	var invalid *service.DietRecipeValidationError
	switch {
	case errors.As(err, &invalid):
		response.ErrValidation(c, invalid.Message)
	case errors.Is(err, service.ErrDietRecipeNotFound):
		response.ErrNotFound(c, "菜谱不存在")
	case errors.Is(err, service.ErrDietRecipeNameConflict):
		response.ErrConflict(c, "菜谱名称已存在")
	default:
		response.ErrInternal(c, fallback)
	}
}
