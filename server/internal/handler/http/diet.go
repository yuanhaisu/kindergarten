package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

// --- 基本参数：集体对象（契约 23-diet.md 第一节，design D7） ---

// dietGroupObjectRequest 是 POST/PUT /admin/diet/group-objects 的请求体。
// 学年在修改时只读：service 沿用对象原学年，但仍校验提交值属于本园。
type dietGroupObjectRequest struct {
	AcademicYearID uint64                       `json:"academic_year_id"`
	Name           string                       `json:"name"`
	CareType       domain.DietCareType          `json:"care_type"`
	FoodInputType  domain.DietFoodInputType     `json:"food_input_type"`
	RatioMode      domain.DietRatioMode         `json:"ratio_mode"`
	ClassIDs       []uint64                     `json:"class_ids"`
	Meals          []dietGroupObjectMealRequest `json:"meals"`
}

type dietGroupObjectMealRequest struct {
	MealKey       domain.DietMealKey `json:"meal_key"`
	Time          string             `json:"time"`
	EnergyPercent float64            `json:"energy_percent"`
	Enabled       bool               `json:"enabled"`
}

func (r dietGroupObjectRequest) toInput() service.DietGroupObjectInput {
	meals := make([]service.DietMealConfigMealInput, 0, len(r.Meals))
	for _, meal := range r.Meals {
		meals = append(meals, service.DietMealConfigMealInput{
			MealKey:       meal.MealKey,
			Time:          meal.Time,
			EnergyPercent: meal.EnergyPercent,
			Enabled:       meal.Enabled,
		})
	}
	return service.DietGroupObjectInput{
		AcademicYearID: r.AcademicYearID,
		Name:           r.Name,
		CareType:       r.CareType,
		FoodInputType:  r.FoodInputType,
		RatioMode:      r.RatioMode,
		ClassIDs:       r.ClassIDs,
		Meals:          meals,
	}
}

type listDietGroupObjectsQuery struct {
	AcademicYearID uint64 `form:"academic_year_id"`
	Keyword        string `form:"keyword"`
	Page           int    `form:"page"`
	PageSize       int    `form:"page_size"`
}

// NewDietAdminHandler 组装膳食模块（集体对象 + 食物库 + 菜谱库 + 周食谱/模板 + 营养分析）的
// Handler，供路由级测试使用。生产路径仍由 NewHandler 全量注入；这里避免构造数十个无关 service。
// settingsSvc / classSvc 只服务于 diet 自读选项接口（学年、就餐班级，DGO-S7）。
func NewDietAdminHandler(
	dietGroupObjectSvc *service.DietGroupObjectService,
	dietFoodSvc *service.DietFoodService,
	dietRecipeSvc *service.DietRecipeService,
	dietPlanSvc *service.DietPlanService,
	dietAnalysisSvc *service.DietAnalysisService,
	settingsSvc *service.SettingsService,
	classSvc *service.ClassService,
) *Handler {
	return &Handler{
		dietGroupObjectSvc: dietGroupObjectSvc,
		dietFoodSvc:        dietFoodSvc,
		dietRecipeSvc:      dietRecipeSvc,
		dietPlanSvc:        dietPlanSvc,
		dietAnalysisSvc:    dietAnalysisSvc,
		settingsSvc:        settingsSvc,
		classSvc:           classSvc,
	}
}

// ListDietGroupObjects 返回本园某学年的集体对象；academic_year_id 缺省取当前学年，
// 该学年没有对象时返回空列表，不在读取路径创建（TC-NEW-DIE-0065）。
func (h *Handler) ListDietGroupObjects(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listDietGroupObjectsQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	list, total, page, pageSize, err := h.dietGroupObjectSvc.List(
		c.Request.Context(), kgID, q.AcademicYearID, q.Keyword, pq,
	)
	if err != nil {
		respondDietGroupObjectError(c, err)
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

// CreateDietGroupObject 新增集体对象，HTTP 201。
func (h *Handler) CreateDietGroupObject(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req dietGroupObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	data, err := h.dietGroupObjectSvc.Create(c.Request.Context(), kgID, req.toInput())
	if err != nil {
		respondDietGroupObjectError(c, err)
		return
	}
	response.Created(c, data)
}

// GetDietGroupObject 返回集体对象详情（完整餐次、比例模式与系数）。
func (h *Handler) GetDietGroupObject(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietGroupObjectID(c)
	if !ok {
		return
	}
	data, err := h.dietGroupObjectSvc.Get(c.Request.Context(), kgID, id)
	if err != nil {
		respondDietGroupObjectError(c, err)
		return
	}
	response.OK(c, data)
}

// UpdateDietGroupObject 修改集体对象，响应返回保存后的详情，便于前端就地刷新。
func (h *Handler) UpdateDietGroupObject(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietGroupObjectID(c)
	if !ok {
		return
	}
	var req dietGroupObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	data, err := h.dietGroupObjectSvc.Update(c.Request.Context(), kgID, id, req.toInput())
	if err != nil {
		respondDietGroupObjectError(c, err)
		return
	}
	response.OK(c, data)
}

func dietGroupObjectID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的集体对象 ID")
		return 0, false
	}
	return id, true
}

// respondDietGroupObjectError 映射 404/422；业务校验文案原样透出。
// 名称冲突目前由 service 以校验错误返回（422），契约若要求 409 再在 S4 收紧。
func respondDietGroupObjectError(c *gin.Context, err error) {
	var invalid *service.DietConfigValidationError
	switch {
	case errors.As(err, &invalid):
		response.ErrValidation(c, invalid.Message)
	case errors.Is(err, service.ErrDietGroupObjectNotFound):
		response.ErrNotFound(c, "集体对象不存在")
	default:
		response.ErrInternal(c, "集体对象服务异常")
	}
}

// --- 基本参数选项：学年与就餐班级（DGO-S7，契约 23-diet.md §一 选项接口） ---
//
// 只有 diet:* 权限的角色（保健医/后勤/园长）不能读 system:kindergarten 与 student:view，
// 因此基本参数页不能直连 /settings/academic-years 与 /classes。这两个自读接口把
// 「供餐配置所需的选项数据」收进 diet 权限域，避免给角色扩权。

// ListDietAcademicYears 返回本园学年列表，供基本参数页与膳食计划页的学年下拉使用。
func (h *Handler) ListDietAcademicYears(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.settingsSvc.ListAcademicYears(c.Request.Context(), kgID)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取学年列表失败")
		return
	}
	response.OK(c, data)
}

// dietClassOption 是就餐班级选项的精简形状：不返回教师、学生数等无关字段（契约 §一）。
type dietClassOption struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Grade        string `json:"grade"`
	AcademicYear string `json:"academic_year"`
	Status       int8   `json:"status"`
}

// ListDietClasses 返回本园班级精简列表，供基本参数页选择就餐班级。
// academic_year 是学年标签（如 2026-2027），缺省返回全部班级。
func (h *Handler) ListDietClasses(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	academicYear := strings.TrimSpace(c.Query("academic_year"))
	// page_size 100 是分页上限；本园班级数远小于 100，截断风险见 DGO-S7 known_gaps。
	list, _, _, _, err := h.classSvc.ListClasses(
		c.Request.Context(), kgID, pagination.Query{Page: 1, PageSize: 100}, "",
	)
	if err != nil {
		response.ErrInternal(c, "获取班级列表失败")
		return
	}
	out := make([]dietClassOption, 0, len(list))
	for _, item := range list {
		if academicYear != "" && item.AcademicYear != academicYear {
			continue
		}
		out = append(out, dietClassOption{
			ID:           item.ID,
			Name:         item.Name,
			Grade:        item.Grade,
			AcademicYear: item.AcademicYear,
			Status:       item.Status,
		})
	}
	response.OK(c, map[string]any{"list": out})
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

// --- 周食谱与模板（契约 23-diet.md 第五、六节） ---

type listDietPlansQuery struct {
	AcademicYearID uint64 `form:"academic_year_id"`
	Keyword        string `form:"keyword"`
	Page           int    `form:"page"`
	PageSize       int    `form:"page_size"`
}

type dietPlanCreateRequest struct {
	Name           string                       `json:"name"`
	AcademicYearID uint64                       `json:"academic_year_id"`
	GroupObjectID  uint64                       `json:"group_object_id"`
	StartDate      string                       `json:"start_date"`
	EndDate        string                       `json:"end_date"`
	WeekOfYear     int                          `json:"week_of_year"`
	CreatedMethod  domain.DietPlanCreatedMethod `json:"created_method"`
	TemplateID     *uint64                      `json:"template_id"`
	SourcePlanID   *uint64                      `json:"source_plan_id"`
}

func (r dietPlanCreateRequest) toInput() service.DietPlanCreateInput {
	return service.DietPlanCreateInput{
		Name:           r.Name,
		AcademicYearID: r.AcademicYearID,
		GroupObjectID:  r.GroupObjectID,
		StartDate:      r.StartDate,
		EndDate:        r.EndDate,
		WeekOfYear:     r.WeekOfYear,
		CreatedMethod:  r.CreatedMethod,
		TemplateID:     r.TemplateID,
		SourcePlanID:   r.SourcePlanID,
	}
}

// dietPlanItemRequest 是整单保存的一行明细。sort 不接受客户端输入。
type dietPlanItemRequest struct {
	Date       string             `json:"date"`
	MealKey    domain.DietMealKey `json:"meal_key"`
	RecipeName string             `json:"recipe_name"`
	RecipeID   *uint64            `json:"recipe_id"`
	FoodID     uint64             `json:"food_id"`
	FoodName   string             `json:"food_name"`
	Grams      float64            `json:"grams"`
}

func toDietPlanItemInputs(rows []dietPlanItemRequest) []service.DietPlanItemInput {
	out := make([]service.DietPlanItemInput, 0, len(rows))
	for _, row := range rows {
		out = append(out, service.DietPlanItemInput{
			Date:       row.Date,
			MealKey:    row.MealKey,
			RecipeName: row.RecipeName,
			RecipeID:   row.RecipeID,
			FoodID:     row.FoodID,
			FoodName:   row.FoodName,
			Grams:      row.Grams,
		})
	}
	return out
}

// dietPlanSaveRequest 是 PUT /plans/:id 的请求体。
// Items 用指针区分「未提交」与「提交空数组」：前者 422，后者等同清空（契约规则 1）。
type dietPlanSaveRequest struct {
	Name  *string                `json:"name"`
	Items *[]dietPlanItemRequest `json:"items"`
}

// ListDietPlans 返回本园周食谱列表；academic_year_id 缺省取当前学年。
func (h *Handler) ListDietPlans(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listDietPlansQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	list, total, page, pageSize, err := h.dietPlanSvc.ListPlans(
		c.Request.Context(), kgID, q.AcademicYearID, q.Keyword, pq,
	)
	if err != nil {
		respondDietPlanError(c, err, "获取周食谱列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

// CreateDietPlan 新建周食谱（自定义/模板/复制），HTTP 201。
func (h *Handler) CreateDietPlan(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req dietPlanCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	data, err := h.dietPlanSvc.CreatePlan(c.Request.Context(), kgID, req.toInput())
	if err != nil {
		respondDietPlanError(c, err, "新建周食谱失败")
		return
	}
	response.Created(c, data)
}

// GetDietPlan 返回周食谱详情（配置快照 + 全部明细）。
func (h *Handler) GetDietPlan(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietPlanID(c)
	if !ok {
		return
	}
	data, err := h.dietPlanSvc.GetPlan(c.Request.Context(), kgID, id)
	if err != nil {
		respondDietPlanError(c, err, "获取周食谱详情失败")
		return
	}
	response.OK(c, data)
}

// UpdateDietPlan 整单保存明细（单事务：校验 → 合并 → 重算快照 → 替换）。
func (h *Handler) UpdateDietPlan(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietPlanID(c)
	if !ok {
		return
	}
	var req dietPlanSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求体格式错误")
		return
	}
	in := service.DietPlanSaveInput{Name: req.Name}
	if req.Items != nil {
		in.Items = toDietPlanItemInputs(*req.Items)
	}
	data, err := h.dietPlanSvc.SavePlan(c.Request.Context(), kgID, id, in)
	if err != nil {
		respondDietPlanError(c, err, "保存周食谱失败")
		return
	}
	response.OK(c, data)
}

// DeleteDietPlan 软删除周食谱；列表与详情不再返回。
func (h *Handler) DeleteDietPlan(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietPlanID(c)
	if !ok {
		return
	}
	if err := h.dietPlanSvc.DeletePlan(c.Request.Context(), kgID, id); err != nil {
		respondDietPlanError(c, err, "删除周食谱失败")
		return
	}
	response.OK(c, map[string]any{})
}

// ClearDietPlan 清空全部明细并保留食谱本身。
func (h *Handler) ClearDietPlan(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietPlanID(c)
	if !ok {
		return
	}
	data, err := h.dietPlanSvc.ClearPlan(c.Request.Context(), kgID, id)
	if err != nil {
		respondDietPlanError(c, err, "清空周食谱失败")
		return
	}
	response.OK(c, data)
}

// ListDietTemplates 返回本园可用模板（只读，不分页）。
func (h *Handler) ListDietTemplates(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	list, err := h.dietPlanSvc.ListTemplates(c.Request.Context(), kgID)
	if err != nil {
		respondDietPlanError(c, err, "获取模板列表失败")
		return
	}
	response.OK(c, map[string]any{"list": list})
}

// GetDietTemplate 返回模板详情（只读）。
func (h *Handler) GetDietTemplate(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的模板 ID")
		return
	}
	data, err := h.dietPlanSvc.GetTemplate(c.Request.Context(), kgID, id)
	if err != nil {
		respondDietPlanError(c, err, "获取模板详情失败")
		return
	}
	response.OK(c, data)
}

func dietPlanID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的食谱 ID")
		return 0, false
	}
	return id, true
}

// respondDietPlanError 把周食谱服务错误映射到 404/409/422 契约包络。
func respondDietPlanError(c *gin.Context, err error, fallback string) {
	var invalid *service.DietPlanValidationError
	switch {
	case errors.As(err, &invalid):
		response.ErrValidation(c, invalid.Message)
	case errors.Is(err, service.ErrDietPlanNotFound):
		response.ErrNotFound(c, "周食谱不存在")
	case errors.Is(err, service.ErrDietTemplateNotFound):
		response.ErrNotFound(c, "模板不存在")
	case errors.Is(err, service.ErrDietPlanNameConflict):
		response.ErrConflict(c, "该学年已存在同名周食谱")
	default:
		response.ErrInternal(c, fallback)
	}
}

// --- 营养分析（契约 23-diet.md 第七节） ---

// GetDietPlanAnalysis 返回某周食谱的 7 组报表与自动结论（只读）。
func (h *Handler) GetDietPlanAnalysis(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietPlanID(c)
	if !ok {
		return
	}
	data, err := h.dietAnalysisSvc.Analyze(c.Request.Context(), kgID, id)
	if err != nil {
		respondDietAnalysisError(c, err)
		return
	}
	response.OK(c, data)
}

// ExportDietPlanAnalysis 同步生成营养分析 Excel，直接返回二进制而不是 JSON 包络。
func (h *Handler) ExportDietPlanAnalysis(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, ok := dietPlanID(c)
	if !ok {
		return
	}
	data, err := h.dietAnalysisSvc.ExportExcel(c.Request.Context(), kgID, id)
	if err != nil {
		respondDietAnalysisError(c, err)
		return
	}
	const xlsxMIME = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	c.Header("Content-Type", xlsxMIME)
	// 契约要求 filename*=UTF-8''<urlencoded>，保留「营养分析」中文文件名。
	c.Header("Content-Disposition",
		"attachment; filename*=UTF-8''"+url.PathEscape("营养分析")+".xlsx")
	c.Data(http.StatusOK, xlsxMIME, data)
}

// respondDietAnalysisError 映射营养分析错误：无明细 422、跨园/不存在 404。
func respondDietAnalysisError(c *gin.Context, err error) {
	var invalid *service.DietPlanValidationError
	switch {
	case errors.As(err, &invalid):
		response.ErrValidation(c, invalid.Message)
	case errors.Is(err, service.ErrDietAnalysisEmpty):
		response.ErrValidation(c, "该食谱暂无明细，无法生成营养分析")
	case errors.Is(err, service.ErrDietPlanNotFound):
		response.ErrNotFound(c, "周食谱不存在")
	default:
		response.ErrInternal(c, "营养分析服务异常")
	}
}
