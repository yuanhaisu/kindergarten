package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	handler "kindergarten-server/internal/handler/http"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

// dietTestResolver 是权限中间件的桩：只返回指定权限码，不依赖 DB。
type dietTestResolver struct {
	codes    []string
	wildcard bool
}

func (s dietTestResolver) ResolveAdminCodes(context.Context, uint64, uint64) ([]string, bool, error) {
	return s.codes, s.wildcard, nil
}

// 独立测试园所 ID，避开真实数据与其他测试文件。
const testRouterDietKG uint64 = 990982

var routerDietDB *gorm.DB

func openRouterDietTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if routerDietDB != nil {
		return routerDietDB
	}
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err, "load config")
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "open db")
	require.NoError(t, db.AutoMigrate(
		&domain.MealConfig{}, &domain.MealConfigMeal{},
		&domain.FoodCategory{}, &domain.Food{},
		&domain.Recipe{}, &domain.RecipeItem{},
		&domain.MealPlan{}, &domain.MealPlanItem{},
		&domain.TemplatePlan{}, &domain.TemplatePlanItem{},
		&domain.DriAgeTarget{},
	), "auto migrate diet tables")
	routerDietDB = db
	return db
}

func cleanupRouterDietData(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, sql := range []string{
		"DELETE FROM diet_meal_config_meals WHERE kindergarten_id = ?",
		"DELETE FROM diet_meal_configs WHERE kindergarten_id = ?",
		"DELETE FROM diet_recipe_items WHERE kindergarten_id = ?",
		"DELETE FROM diet_recipes WHERE kindergarten_id = ?",
		"DELETE FROM diet_meal_plan_items WHERE kindergarten_id = ?",
		"DELETE FROM diet_meal_plans WHERE kindergarten_id = ?",
		"DELETE FROM diet_template_plan_items WHERE kindergarten_id = ?",
		"DELETE FROM diet_template_plans WHERE kindergarten_id = ?",
		"DELETE FROM diet_foods WHERE kindergarten_id = ?",
		"DELETE FROM academic_years WHERE kindergarten_id = ?",
	} {
		if err := db.Exec(sql, testRouterDietKG).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
}

// newDietAdminEngine 只装配 admin 路由与细闸中间件，不经过 AuthJWT/Casbin/Redis。
// 膳食服务打真实 MySQL（与 repository/service 测试同源），无 DB 时整组跳过。
func newDietAdminEngine(t *testing.T, resolver middleware.PermissionResolver) *gin.Engine {
	t.Helper()
	db := openRouterDietTestDB(t)
	cleanupRouterDietData(t, db)
	t.Cleanup(func() { cleanupRouterDietData(t, db) })

	configSvc := service.NewDietConfigService(repository.NewDietConfigRepo(db))
	foodRepo := repository.NewDietFoodRepo(db)
	recipeRepo := repository.NewDietRecipeRepo(db)
	foodSvc := service.NewDietFoodService(foodRepo, recipeRepo)
	recipeSvc := service.NewDietRecipeService(recipeRepo, foodRepo)
	planSvc := service.NewDietPlanService(
		repository.NewDietPlanRepo(db),
		repository.NewDietTemplateRepo(db),
		foodRepo,
		repository.NewClassRepo(db),
		repository.NewAcademicYearRepo(db),
		repository.NewDietDriRepo(db),
		configSvc,
	)
	analysisSvc := service.NewDietAnalysisService(repository.NewDietPlanRepo(db), foodRepo)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	adminG := r.Group("/api/v1/admin")
	adminG.Use(func(c *gin.Context) {
		c.Set(middleware.CtxClaimsKey, &auth.AccessClaims{
			UserID: "4", Roles: []string{"admin"}, KindergartenID: testRouterDietKG,
		})
		c.Next()
	})
	adminG.Use(middleware.RequirePermission(resolver))
	registerAdminRoutes(adminG, handler.NewDietAdminHandler(configSvc, foodSvc, recipeSvc, planSvc, analysisSvc))
	return r
}

func doDietRequest(t *testing.T, engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var payload *strings.Reader
	if body != "" {
		payload = strings.NewReader(body)
	}
	var req *http.Request
	if payload == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, payload)
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

type dietEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeDietEnvelope(t *testing.T, w *httptest.ResponseRecorder) dietEnvelope {
	t.Helper()
	var env dietEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env),
		"响应必须是 JSON 包络：%s", w.Body.String())
	return env
}

const dietStandardConfigJSON = `{
  "care_type": "day_care",
  "food_input_type": "market_weight",
  "ratio_mode": "standard",
  "meals": [
    { "meal_key": "breakfast",       "time": "07:00", "energy_percent": 25, "enabled": true },
    { "meal_key": "morning_snack",   "time": "09:00", "energy_percent": 5,  "enabled": true },
    { "meal_key": "lunch",           "time": "11:00", "energy_percent": 35, "enabled": true },
    { "meal_key": "afternoon_snack", "time": "14:30", "energy_percent": 5,  "enabled": true },
    { "meal_key": "dinner",          "time": "17:00", "energy_percent": 25, "enabled": true },
    { "meal_key": "evening_snack",   "time": "20:00", "energy_percent": 5,  "enabled": false }
  ]
}`

// 自定模式下早餐 25 + 早点 6 = 31，超过日托上限 29。
const dietInvalidCustomConfigJSON = `{
  "care_type": "day_care",
  "food_input_type": "market_weight",
  "ratio_mode": "custom",
  "meals": [
    { "meal_key": "breakfast",       "time": "07:00", "energy_percent": 25, "enabled": true },
    { "meal_key": "morning_snack",   "time": "09:00", "energy_percent": 6,  "enabled": true },
    { "meal_key": "lunch",           "time": "11:00", "energy_percent": 35, "enabled": true },
    { "meal_key": "afternoon_snack", "time": "14:30", "energy_percent": 5,  "enabled": true },
    { "meal_key": "dinner",          "time": "17:00", "energy_percent": 25, "enabled": true },
    { "meal_key": "evening_snack",   "time": "20:00", "energy_percent": 5,  "enabled": false }
  ]
}`

// 真实行为：GET 返回默认配置，PUT 标准比例成功，再 GET 读回保存值。
func TestRouterDiet_ConfigRealBehavior(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})

	w := doDietRequest(t, engine, http.MethodGet, "/api/v1/admin/diet/config", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env := decodeDietEnvelope(t, w)
	require.Equal(t, 0, env.Code)
	var view service.DietConfigView
	require.NoError(t, json.Unmarshal(env.Data, &view))
	assert.Equal(t, domain.DietCareTypeDayCare, view.CareType)
	assert.Equal(t, domain.DietRatioModeStandard, view.RatioMode)
	assert.Equal(t, 0.95, view.Coefficient)
	require.Len(t, view.Meals, 6)

	w = doDietRequest(t, engine, http.MethodPut, "/api/v1/admin/diet/config", dietStandardConfigJSON)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	require.Equal(t, 0, env.Code)
	var saved service.DietConfigView
	require.NoError(t, json.Unmarshal(env.Data, &saved))
	assert.Equal(t, 0.95, saved.Coefficient)

	w = doDietRequest(t, engine, http.MethodGet, "/api/v1/admin/diet/config", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	var reloaded service.DietConfigView
	require.NoError(t, json.Unmarshal(env.Data, &reloaded))
	assert.Equal(t, saved, reloaded, "保存后读取必须一致")
}

// 业务校验失败返回 422 + 可读 message；非法 JSON 也走 422。
func TestRouterDiet_ConfigValidationReturns422(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})

	w := doDietRequest(t, engine, http.MethodPut, "/api/v1/admin/diet/config", dietInvalidCustomConfigJSON)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	env := decodeDietEnvelope(t, w)
	assert.Equal(t, 42200, env.Code)
	assert.NotEmpty(t, env.Message)

	w = doDietRequest(t, engine, http.MethodPut, "/api/v1/admin/diet/config", "{not-json")
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	assert.Equal(t, 42200, env.Code)
}

// 两条配置路由的权限码：GET=diet:view，PUT=diet:manage，互不替代。
func TestRouterDiet_ConfigPermissionCodes(t *testing.T) {
	viewOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view"}})
	assert.Equal(t, http.StatusOK,
		doDietRequest(t, viewOnly, http.MethodGet, "/api/v1/admin/diet/config", "").Code)
	assert.Equal(t, http.StatusForbidden,
		doDietRequest(t, viewOnly, http.MethodPut, "/api/v1/admin/diet/config", dietStandardConfigJSON).Code)

	manageOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:manage"}})
	assert.Equal(t, http.StatusForbidden,
		doDietRequest(t, manageOnly, http.MethodGet, "/api/v1/admin/diet/config", "").Code)
	assert.Equal(t, http.StatusOK,
		doDietRequest(t, manageOnly, http.MethodPut, "/api/v1/admin/diet/config", dietStandardConfigJSON).Code)
}

// AQR-S3：8 条食物路由必须真实注册（分类 1 + 食物 7），不得有 501/stub。
func TestRouterDiet_FoodRoutesRegistered(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})

	want := []string{
		"GET /api/v1/admin/diet/food-categories",
		"GET /api/v1/admin/diet/foods",
		"POST /api/v1/admin/diet/foods",
		"POST /api/v1/admin/diet/foods/batch-delete",
		"GET /api/v1/admin/diet/foods/:id",
		"PUT /api/v1/admin/diet/foods/:id",
		"DELETE /api/v1/admin/diet/foods/:id",
		"GET /api/v1/admin/diet/foods/:id/replace-candidates",
	}
	got := map[string]bool{}
	for _, rt := range engine.Routes() {
		if strings.HasPrefix(rt.Path, "/api/v1/admin/diet/") {
			got[rt.Method+" "+rt.Path] = true
		}
	}
	for _, key := range want {
		assert.True(t, got[key], "缺少食物路由 %s", key)
	}
	assert.Len(t, got, 27, "diet 组当前应恰好 2 条配置 + 8 条食物 + 7 条菜谱 + 8 条周食谱/模板 + 2 条营养分析路由")
}

// AQR-S4：7 条菜谱路由必须真实注册，不得有 501/stub。
func TestRouterDiet_RecipeRoutesRegistered(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})

	want := []string{
		"GET /api/v1/admin/diet/recipes",
		"POST /api/v1/admin/diet/recipes",
		"GET /api/v1/admin/diet/recipes/suggest",
		"POST /api/v1/admin/diet/recipes/merge",
		"GET /api/v1/admin/diet/recipes/:id",
		"PUT /api/v1/admin/diet/recipes/:id",
		"DELETE /api/v1/admin/diet/recipes/:id",
	}
	got := map[string]bool{}
	for _, rt := range engine.Routes() {
		if strings.HasPrefix(rt.Path, "/api/v1/admin/diet/recipes") {
			got[rt.Method+" "+rt.Path] = true
		}
	}
	for _, key := range want {
		assert.True(t, got[key], "缺少菜谱路由 %s", key)
	}
	assert.Len(t, got, 7, "菜谱库应恰好 7 条路由")
}

// AQR-S3：GET 走 diet:view，写操作走 diet:manage，两个码互不替代。
func TestRouterDiet_FoodPermissionCodes(t *testing.T) {
	viewOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view"}})
	for _, path := range []string{
		"/api/v1/admin/diet/food-categories",
		"/api/v1/admin/diet/foods",
		"/api/v1/admin/diet/foods/999999999",
		"/api/v1/admin/diet/foods/999999999/replace-candidates",
	} {
		assert.NotEqual(t, http.StatusForbidden,
			doDietRequest(t, viewOnly, http.MethodGet, path, "").Code, "GET %s 应允许 diet:view", path)
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/admin/diet/foods", "{}"},
		{http.MethodPut, "/api/v1/admin/diet/foods/1", "{}"},
		{http.MethodDelete, "/api/v1/admin/diet/foods/1", ""},
		{http.MethodPost, "/api/v1/admin/diet/foods/batch-delete", `{"ids":[]}`},
	} {
		assert.Equal(t, http.StatusForbidden,
			doDietRequest(t, viewOnly, tc.method, tc.path, tc.body).Code,
			"%s %s 只读码必须被拒", tc.method, tc.path)
	}

	manageOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:manage"}})
	for _, path := range []string{
		"/api/v1/admin/diet/food-categories",
		"/api/v1/admin/diet/foods",
	} {
		assert.Equal(t, http.StatusForbidden,
			doDietRequest(t, manageOnly, http.MethodGet, path, "").Code,
			"GET %s 管理码不能替代查看码", path)
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/admin/diet/foods", "{}"},
		{http.MethodPut, "/api/v1/admin/diet/foods/999999999", "{}"},
		{http.MethodDelete, "/api/v1/admin/diet/foods/999999999", ""},
		{http.MethodPost, "/api/v1/admin/diet/foods/batch-delete", `{"ids":[]}`},
	} {
		assert.NotEqual(t, http.StatusForbidden,
			doDietRequest(t, manageOnly, tc.method, tc.path, tc.body).Code,
			"%s %s 应通过权限检查", tc.method, tc.path)
	}
}

// AQR-S4：菜谱 GET（列表/详情/联想）走 diet:view，写操作走 diet:manage。
func TestRouterDiet_RecipePermissionCodes(t *testing.T) {
	viewOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view"}})
	for _, path := range []string{
		"/api/v1/admin/diet/recipes",
		"/api/v1/admin/diet/recipes/suggest",
		"/api/v1/admin/diet/recipes/999999999",
	} {
		assert.NotEqual(t, http.StatusForbidden,
			doDietRequest(t, viewOnly, http.MethodGet, path, "").Code, "GET %s 应允许 diet:view", path)
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/admin/diet/recipes", "{}"},
		{http.MethodPost, "/api/v1/admin/diet/recipes/merge", "{}"},
		{http.MethodPut, "/api/v1/admin/diet/recipes/1", "{}"},
		{http.MethodDelete, "/api/v1/admin/diet/recipes/1", ""},
	} {
		assert.Equal(t, http.StatusForbidden,
			doDietRequest(t, viewOnly, tc.method, tc.path, tc.body).Code,
			"%s %s 只读码必须被拒", tc.method, tc.path)
	}

	manageOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:manage"}})
	for _, path := range []string{
		"/api/v1/admin/diet/recipes",
		"/api/v1/admin/diet/recipes/suggest",
	} {
		assert.Equal(t, http.StatusForbidden,
			doDietRequest(t, manageOnly, http.MethodGet, path, "").Code,
			"GET %s 管理码不能替代查看码", path)
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/admin/diet/recipes", "{}"},
		{http.MethodPost, "/api/v1/admin/diet/recipes/merge", "{}"},
		{http.MethodPut, "/api/v1/admin/diet/recipes/999999999", "{}"},
		{http.MethodDelete, "/api/v1/admin/diet/recipes/999999999", ""},
	} {
		assert.NotEqual(t, http.StatusForbidden,
			doDietRequest(t, manageOnly, tc.method, tc.path, tc.body).Code,
			"%s %s 应通过权限检查", tc.method, tc.path)
	}
}

// AQR-S4 真实行为：新增 201、同名冲突 409、替换 200、详情/联想/删除的包络与状态码。
func TestRouterDiet_RecipeRealBehavior(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})
	db := openRouterDietTestDB(t)

	energy := 286.0
	food := &domain.Food{
		KindergartenID: testRouterDietKG,
		Name:           "S4路由测试面粉",
		CategoryID:     1,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceCustom,
		Energy:         &energy,
	}
	require.NoError(t, db.Create(food).Error)
	require.NotZero(t, food.ID)

	createBody := `{"name":"S4路由测试云吞","items":[{"food_id":` +
		strconv.FormatUint(food.ID, 10) + `,"grams":10}],"replace":false}`
	w := doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/recipes", createBody)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	env := decodeDietEnvelope(t, w)
	require.Equal(t, 0, env.Code)
	var created service.DietRecipeSaveResult
	require.NoError(t, json.Unmarshal(env.Data, &created))
	require.NotZero(t, created.ID)

	// 同名且 replace=false → 409 / 40900，且不写入。
	w = doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/recipes", createBody)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	assert.Equal(t, 40900, env.Code)
	assert.Equal(t, "菜谱名称已存在", env.Message)

	// replace=true → 200，同一个菜谱被覆盖。
	replaceBody := `{"name":"S4路由测试云吞","items":[{"food_id":` +
		strconv.FormatUint(food.ID, 10) + `,"grams":12}],"replace":true}`
	w = doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/recipes", replaceBody)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doDietRequest(t, engine, http.MethodGet, "/api/v1/admin/diet/recipes/"+strconv.FormatUint(created.ID, 10), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	var detail service.DietRecipeDetail
	require.NoError(t, json.Unmarshal(env.Data, &detail))
	require.Len(t, detail.Items, 1)
	assert.Equal(t, "S4路由测试面粉", detail.Items[0].FoodName)
	assert.InDelta(t, 12.0, detail.Items[0].Grams, 1e-9)
	require.NotNil(t, detail.Items[0].Nutrients.Energy)
	assert.InDelta(t, 286.0, *detail.Items[0].Nutrients.Energy, 1e-9)

	w = doDietRequest(t, engine, http.MethodGet, "/api/v1/admin/diet/recipes?keyword=S4路由", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	var page struct {
		List  []service.DietRecipeListItem `json:"list"`
		Total int64                        `json:"total"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &page))
	assert.Equal(t, int64(1), page.Total)

	w = doDietRequest(t, engine, http.MethodGet, "/api/v1/admin/diet/recipes/suggest?keyword=S4路由&limit=5", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	var suggest struct {
		List []service.DietRecipeListItem `json:"list"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &suggest))
	assert.Len(t, suggest.List, 1)

	// 合并到自身 → 422。
	w = doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/recipes/merge",
		`{"source_id":`+strconv.FormatUint(created.ID, 10)+`,"target_id":`+strconv.FormatUint(created.ID, 10)+`}`)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())

	w = doDietRequest(t, engine, http.MethodDelete, "/api/v1/admin/diet/recipes/"+strconv.FormatUint(created.ID, 10), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doDietRequest(t, engine, http.MethodGet, "/api/v1/admin/diet/recipes/"+strconv.FormatUint(created.ID, 10), "")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	assert.Equal(t, 40400, env.Code)
}

// AQR-S6：8 条周食谱/模板路由必须真实注册，不得有 501/stub。
func TestRouterDiet_PlanRoutesRegistered(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})

	want := []string{
		"GET /api/v1/admin/diet/plans",
		"POST /api/v1/admin/diet/plans",
		"GET /api/v1/admin/diet/plans/:id",
		"PUT /api/v1/admin/diet/plans/:id",
		"DELETE /api/v1/admin/diet/plans/:id",
		"POST /api/v1/admin/diet/plans/:id/clear",
		"GET /api/v1/admin/diet/templates",
		"GET /api/v1/admin/diet/templates/:id",
	}
	got := map[string]bool{}
	for _, rt := range engine.Routes() {
		if strings.HasPrefix(rt.Path, "/api/v1/admin/diet/plans") ||
			strings.HasPrefix(rt.Path, "/api/v1/admin/diet/templates") {
			got[rt.Method+" "+rt.Path] = true
		}
	}
	for _, key := range want {
		assert.True(t, got[key], "缺少周食谱/模板路由 %s", key)
	}
	assert.Len(t, got, 10, "周食谱与模板 8 条 + AQR-S7 营养分析 2 条（analysis 挂在 /plans/:id 下）")
}

// AQR-S6：GET 走 diet:view，写操作走 diet:manage，两个码互不替代。
func TestRouterDiet_PlanPermissionCodes(t *testing.T) {
	viewOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view"}})
	for _, path := range []string{
		"/api/v1/admin/diet/plans",
		"/api/v1/admin/diet/plans/999999999",
		"/api/v1/admin/diet/templates",
		"/api/v1/admin/diet/templates/999999999",
	} {
		assert.NotEqual(t, http.StatusForbidden,
			doDietRequest(t, viewOnly, http.MethodGet, path, "").Code, "GET %s 应允许 diet:view", path)
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/admin/diet/plans", "{}"},
		{http.MethodPut, "/api/v1/admin/diet/plans/1", "{}"},
		{http.MethodDelete, "/api/v1/admin/diet/plans/1", ""},
		{http.MethodPost, "/api/v1/admin/diet/plans/1/clear", ""},
	} {
		assert.Equal(t, http.StatusForbidden,
			doDietRequest(t, viewOnly, tc.method, tc.path, tc.body).Code,
			"%s %s 只读码必须被拒", tc.method, tc.path)
	}

	manageOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:manage"}})
	for _, path := range []string{
		"/api/v1/admin/diet/plans",
		"/api/v1/admin/diet/templates",
		"/api/v1/admin/diet/templates/1",
	} {
		assert.Equal(t, http.StatusForbidden,
			doDietRequest(t, manageOnly, http.MethodGet, path, "").Code,
			"GET %s 管理码不能替代查看码", path)
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/admin/diet/plans", "{}"},
		{http.MethodPut, "/api/v1/admin/diet/plans/999999999", "{}"},
		{http.MethodDelete, "/api/v1/admin/diet/plans/999999999", ""},
		{http.MethodPost, "/api/v1/admin/diet/plans/999999999/clear", ""},
	} {
		assert.NotEqual(t, http.StatusForbidden,
			doDietRequest(t, manageOnly, tc.method, tc.path, tc.body).Code,
			"%s %s 应通过权限检查", tc.method, tc.path)
	}
}

// AQR-S6 真实行为：新建 201、同学年重名 409、详情含配置快照、清空保留食谱、删除后 404。
func TestRouterDiet_PlanRealBehavior(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})
	db := openRouterDietTestDB(t)

	year := &domain.AcademicYear{
		KindergartenID: testRouterDietKG,
		Year:           "2026-2027",
		IsCurrent:      true,
	}
	require.NoError(t, db.Create(year).Error)

	yearID := strconv.FormatUint(year.ID, 10)
	body := `{"name":"S6路由第九周","academic_year_id":` + yearID +
		`,"start_date":"2026-10-26","end_date":"2026-11-01","week_of_year":9,"created_method":"custom"}`

	w := doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/plans", body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	env := decodeDietEnvelope(t, w)
	require.Equal(t, 0, env.Code)
	var created service.DietPlanCreateResult
	require.NoError(t, json.Unmarshal(env.Data, &created))
	require.NotZero(t, created.ID)
	assert.Zero(t, created.DayNum)
	assert.Equal(t, "2026-10-26", created.StartDate)

	// 同学年重名 → 409 / 40900。
	w = doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/plans", body)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	assert.Equal(t, 40900, env.Code)

	planID := strconv.FormatUint(created.ID, 10)
	w = doDietRequest(t, engine, http.MethodGet, "/api/v1/admin/diet/plans/"+planID, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	var detail service.DietPlanDetail
	require.NoError(t, json.Unmarshal(env.Data, &detail))
	assert.Equal(t, "S6路由第九周", detail.Name)
	assert.Equal(t, year.ID, detail.AcademicYearID)
	assert.Len(t, detail.ConfigSnapshot.Meals, 6)
	assert.InDelta(t, 0.95, detail.ConfigSnapshot.Coefficient, 1e-9)
	assert.Empty(t, detail.Items)

	w = doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/plans/"+planID+"/clear", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	var cleared service.DietPlanClearResult
	require.NoError(t, json.Unmarshal(env.Data, &cleared))
	assert.Equal(t, created.ID, cleared.ID)
	assert.Zero(t, cleared.DayNum)

	w = doDietRequest(t, engine, http.MethodDelete, "/api/v1/admin/diet/plans/"+planID, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doDietRequest(t, engine, http.MethodGet, "/api/v1/admin/diet/plans/"+planID, "")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	assert.Equal(t, 40400, env.Code)
}

// AQR-S7：2 条营养分析路由必须真实注册，不得有 501/stub。
func TestRouterDiet_AnalysisRoutesRegistered(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:export"}})

	want := []string{
		"GET /api/v1/admin/diet/plans/:id/analysis",
		"GET /api/v1/admin/diet/plans/:id/analysis/export",
	}
	got := map[string]bool{}
	for _, rt := range engine.Routes() {
		if strings.HasPrefix(rt.Path, "/api/v1/admin/diet/plans/") &&
			strings.Contains(rt.Path, "analysis") {
			got[rt.Method+" "+rt.Path] = true
		}
	}
	for _, key := range want {
		assert.True(t, got[key], "缺少营养分析路由 %s", key)
	}
	assert.Len(t, got, 2, "营养分析应恰好 2 条路由")
}

// AQR-S7：analysis 走 diet:view，export 走 diet:export，两个码互不替代。
func TestRouterDiet_AnalysisPermissionCodes(t *testing.T) {
	const (
		analysisPath = "/api/v1/admin/diet/plans/999999999/analysis"
		exportPath   = "/api/v1/admin/diet/plans/999999999/analysis/export"
	)

	viewOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view"}})
	assert.NotEqual(t, http.StatusForbidden,
		doDietRequest(t, viewOnly, http.MethodGet, analysisPath, "").Code,
		"analysis 应允许 diet:view")
	assert.Equal(t, http.StatusForbidden,
		doDietRequest(t, viewOnly, http.MethodGet, exportPath, "").Code,
		"export 只读码不能替代导出码")

	exportOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:export"}})
	assert.Equal(t, http.StatusForbidden,
		doDietRequest(t, exportOnly, http.MethodGet, analysisPath, "").Code,
		"导出码不能替代查看码")
	assert.NotEqual(t, http.StatusForbidden,
		doDietRequest(t, exportOnly, http.MethodGet, exportPath, "").Code,
		"export 应允许 diet:export")
}
