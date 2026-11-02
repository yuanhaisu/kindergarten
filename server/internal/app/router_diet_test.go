package app

import (
	"context"
	"encoding/json"
	"fmt"
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
		&domain.Kindergarten{}, &domain.AcademicYear{}, &domain.Class{},
		&domain.DietGroupObject{}, &domain.DietGroupObjectMeal{}, &domain.DietGroupObjectClass{},
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
		"DELETE FROM diet_group_object_classes WHERE kindergarten_id = ?",
		"DELETE FROM diet_group_object_meals WHERE kindergarten_id = ?",
		"DELETE FROM diet_group_objects WHERE kindergarten_id = ?",
		"DELETE FROM diet_foods WHERE kindergarten_id = ?",
		"DELETE FROM classes WHERE kindergarten_id = ?",
		"DELETE FROM academic_years WHERE kindergarten_id = ?",
		"DELETE FROM kindergartens WHERE id = ?",
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

	foodRepo := repository.NewDietFoodRepo(db)
	recipeRepo := repository.NewDietRecipeRepo(db)
	classRepo := repository.NewClassRepo(db)
	yearRepo := repository.NewAcademicYearRepo(db)
	foodSvc := service.NewDietFoodService(foodRepo, recipeRepo)
	recipeSvc := service.NewDietRecipeService(recipeRepo, foodRepo)
	groupObjectSvc := service.NewDietGroupObjectService(
		repository.NewDietGroupObjectRepo(db), yearRepo, classRepo)
	planSvc := service.NewDietPlanService(
		repository.NewDietPlanRepo(db),
		repository.NewDietTemplateRepo(db),
		foodRepo,
		classRepo,
		yearRepo,
		repository.NewDietDriRepo(db),
		groupObjectSvc,
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
	registerAdminRoutes(adminG, handler.NewDietAdminHandler(groupObjectSvc, foodSvc, recipeSvc, planSvc, analysisSvc))
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

// routerDietGroupObjectBody 生成标准集体对象请求体：日托、市品重量、标准比例、
// 五餐启用 + 晚点停用（系数 0.95）。classIDs 为空用于验证「至少一个就餐班级」。
func routerDietGroupObjectBody(yearID uint64, name string, classIDs ...uint64) string {
	ids := make([]string, 0, len(classIDs))
	for _, id := range classIDs {
		ids = append(ids, strconv.FormatUint(id, 10))
	}
	return fmt.Sprintf(`{
  "academic_year_id": %d,
  "name": %q,
  "care_type": "day_care",
  "food_input_type": "market_weight",
  "ratio_mode": "standard",
  "class_ids": [%s],
  "meals": [
    { "meal_key": "breakfast",       "time": "07:00", "energy_percent": 25, "enabled": true },
    { "meal_key": "morning_snack",   "time": "09:00", "energy_percent": 5,  "enabled": true },
    { "meal_key": "lunch",           "time": "11:00", "energy_percent": 35, "enabled": true },
    { "meal_key": "afternoon_snack", "time": "14:30", "energy_percent": 5,  "enabled": true },
    { "meal_key": "dinner",          "time": "17:00", "energy_percent": 25, "enabled": true },
    { "meal_key": "evening_snack",   "time": "20:00", "energy_percent": 5,  "enabled": false }
  ]
}`, yearID, name, strings.Join(ids, ","))
}

// seedRouterDietSchool 建一条园所、学年与班级：集体对象与周食谱都需要真实班级行
// （classes 对 kindergartens 有外键，保存对象时还会校验班级属于本园）。
func seedRouterDietSchool(t *testing.T, db *gorm.DB) (*domain.AcademicYear, *domain.Class) {
	t.Helper()
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testRouterDietKG},
		Name:      "S3b路由测试园",
	}).Error)
	year := &domain.AcademicYear{
		KindergartenID: testRouterDietKG,
		Year:           "2026-2027",
		IsCurrent:      true,
	}
	require.NoError(t, db.Create(year).Error)
	class := &domain.Class{
		KindergartenID: testRouterDietKG,
		Name:           "S3b小一班",
		Grade:          "小班",
		AcademicYear:   year.Year,
		Status:         1,
	}
	require.NoError(t, db.Create(class).Error)
	return year, class
}

// DGO-S3b：4 条集体对象路由必须真实注册；旧的 /diet/config 单例端点已移除（404）。
func TestRouterDiet_GroupObjectRoutesRegistered(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})

	want := []string{
		"GET /api/v1/admin/diet/group-objects",
		"POST /api/v1/admin/diet/group-objects",
		"GET /api/v1/admin/diet/group-objects/:id",
		"PUT /api/v1/admin/diet/group-objects/:id",
	}
	got := map[string]bool{}
	for _, rt := range engine.Routes() {
		if strings.HasPrefix(rt.Path, "/api/v1/admin/diet/group-objects") {
			got[rt.Method+" "+rt.Path] = true
		}
	}
	for _, key := range want {
		assert.True(t, got[key], "缺少集体对象路由 %s", key)
	}
	assert.Len(t, got, 4, "集体对象应恰好 4 条路由")

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		assert.Equal(t, http.StatusNotFound,
			doDietRequest(t, engine, method, "/api/v1/admin/diet/config", "").Code,
			"%s /diet/config 必须已移除", method)
	}
}

// DGO-S3b：GET 走 diet:view，POST/PUT 走 diet:manage，两个码互不替代。
func TestRouterDiet_GroupObjectPermissionCodes(t *testing.T) {
	viewOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view"}})
	for _, path := range []string{
		"/api/v1/admin/diet/group-objects",
		"/api/v1/admin/diet/group-objects/999999999",
	} {
		assert.NotEqual(t, http.StatusForbidden,
			doDietRequest(t, viewOnly, http.MethodGet, path, "").Code, "GET %s 应允许 diet:view", path)
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/admin/diet/group-objects", "{}"},
		{http.MethodPut, "/api/v1/admin/diet/group-objects/1", "{}"},
	} {
		assert.Equal(t, http.StatusForbidden,
			doDietRequest(t, viewOnly, tc.method, tc.path, tc.body).Code,
			"%s %s 只读码必须被拒", tc.method, tc.path)
	}

	manageOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:manage"}})
	for _, path := range []string{
		"/api/v1/admin/diet/group-objects",
		"/api/v1/admin/diet/group-objects/1",
	} {
		assert.Equal(t, http.StatusForbidden,
			doDietRequest(t, manageOnly, http.MethodGet, path, "").Code,
			"GET %s 管理码不能替代查看码", path)
	}
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/admin/diet/group-objects", "{}"},
		{http.MethodPut, "/api/v1/admin/diet/group-objects/999999999", "{}"},
	} {
		assert.NotEqual(t, http.StatusForbidden,
			doDietRequest(t, manageOnly, tc.method, tc.path, tc.body).Code,
			"%s %s 应通过权限检查", tc.method, tc.path)
	}
}

// DGO-S3b 真实行为：新增 201、重名/缺班级 422、列表裁剪、详情与修改读回、跨园 404。
func TestRouterDiet_GroupObjectRealBehavior(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})
	db := openRouterDietTestDB(t)
	year, class := seedRouterDietSchool(t, db)
	yearID := strconv.FormatUint(year.ID, 10)
	createBody := routerDietGroupObjectBody(year.ID, "全体幼儿", class.ID)

	w := doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/group-objects", createBody)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	env := decodeDietEnvelope(t, w)
	require.Equal(t, 0, env.Code)
	var created service.DietGroupObjectDetail
	require.NoError(t, json.Unmarshal(env.Data, &created))
	require.NotZero(t, created.ID)
	assert.Equal(t, "全体幼儿", created.Name)
	assert.Equal(t, year.ID, created.AcademicYearID)
	assert.Equal(t, domain.DietCareTypeDayCare, created.CareType)
	assert.Equal(t, "日托", created.CareTypeLabel)
	assert.Equal(t, "市品重量", created.FoodInputTypeLabel)
	assert.Equal(t, "DRIs2000", created.PolicyName)
	assert.InDelta(t, 0.95, created.Coefficient, 1e-9)
	assert.Len(t, created.Meals, 6)
	require.Len(t, created.Classes, 1)
	assert.Equal(t, class.ID, created.Classes[0].ID)
	assert.Equal(t, "S3b小一班", created.Classes[0].Name)

	// 同学年重名：service 以业务校验错误返回，handler 透出 422 + 原文。
	w = doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/group-objects", createBody)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	assert.Equal(t, 42200, env.Code)
	assert.Contains(t, env.Message, "同名")

	// 列表：行只含展示字段，餐次只返回已启用项。
	w = doDietRequest(t, engine, http.MethodGet,
		"/api/v1/admin/diet/group-objects?academic_year_id="+yearID+"&keyword=全体", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	var page struct {
		List  []service.DietGroupObjectListItem `json:"list"`
		Total int64                             `json:"total"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &page))
	assert.Equal(t, int64(1), page.Total)
	require.Len(t, page.List, 1)
	assert.Equal(t, []string{"早餐", "早点", "午餐", "午点", "晚餐"}, page.List[0].MealNames)
	assert.Equal(t, "日托", page.List[0].CareTypeLabel)
	require.Len(t, page.List[0].Classes, 1)

	objectID := strconv.FormatUint(created.ID, 10)
	w = doDietRequest(t, engine, http.MethodGet, "/api/v1/admin/diet/group-objects/"+objectID, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	var detail service.DietGroupObjectDetail
	require.NoError(t, json.Unmarshal(env.Data, &detail))
	assert.Equal(t, created, detail)

	// 修改：改名并调整早餐时间，响应即保存后的详情。
	updateBody := strings.Replace(
		routerDietGroupObjectBody(year.ID, "全体幼儿（改）", class.ID),
		`"time": "07:00"`, `"time": "07:15"`, 1)
	w = doDietRequest(t, engine, http.MethodPut,
		"/api/v1/admin/diet/group-objects/"+objectID, updateBody)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	var updated service.DietGroupObjectDetail
	require.NoError(t, json.Unmarshal(env.Data, &updated))
	assert.Equal(t, "全体幼儿（改）", updated.Name)
	require.Len(t, updated.Meals, 6)
	assert.Equal(t, "07:15", updated.Meals[0].Time)
	assert.InDelta(t, 0.95, updated.Coefficient, 1e-9)

	// 跨园/不存在一律 404。
	w = doDietRequest(t, engine, http.MethodGet, "/api/v1/admin/diet/group-objects/999999999", "")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	assert.Equal(t, 40400, env.Code)

	w = doDietRequest(t, engine, http.MethodPut,
		"/api/v1/admin/diet/group-objects/999999999", updateBody)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// 未选就餐班级：422 + 可读文案，且不落库。
	w = doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/group-objects",
		routerDietGroupObjectBody(year.ID, "无班级对象"))
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	assert.Contains(t, env.Message, "至少选择一个就餐班级")
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
	assert.Len(t, got, 29, "diet 组当前应恰好 4 条集体对象 + 8 条食物 + 7 条菜谱 + 8 条周食谱/模板 + 2 条营养分析路由")
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
// DGO-S3b：周食谱必须绑定集体对象，快照携带对象身份。
func TestRouterDiet_PlanRealBehavior(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})
	db := openRouterDietTestDB(t)
	year, class := seedRouterDietSchool(t, db)

	w := doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/group-objects",
		routerDietGroupObjectBody(year.ID, "全体幼儿", class.ID))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var object service.DietGroupObjectDetail
	require.NoError(t, json.Unmarshal(decodeDietEnvelope(t, w).Data, &object))
	require.NotZero(t, object.ID)

	yearID := strconv.FormatUint(year.ID, 10)
	body := `{"name":"S6路由第九周","academic_year_id":` + yearID +
		`,"group_object_id":` + strconv.FormatUint(object.ID, 10) +
		`,"start_date":"2026-10-26","end_date":"2026-11-01","week_of_year":9,"created_method":"custom"}`

	w = doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/plans", body)
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
	assert.Equal(t, object.ID, detail.GroupObjectID)
	assert.Equal(t, "全体幼儿", detail.GroupObjectName)
	assert.Equal(t, object.ID, detail.ConfigSnapshot.GroupObjectID)
	assert.Equal(t, "全体幼儿", detail.ConfigSnapshot.GroupObjectName)
	assert.Equal(t, []uint64{class.ID}, detail.ConfigSnapshot.GroupObjectClassIDs)
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

// DGO-S3b：未指定集体对象或对象不属于本园时创建被拒（TC-NEW-DIE-0071/0072）。
func TestRouterDiet_PlanRequiresGroupObject(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})
	db := openRouterDietTestDB(t)
	year, _ := seedRouterDietSchool(t, db)
	yearID := strconv.FormatUint(year.ID, 10)

	body := `{"name":"缺对象","academic_year_id":` + yearID +
		`,"start_date":"2026-11-02","end_date":"2026-11-08","week_of_year":10,"created_method":"custom"}`
	w := doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/plans", body)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	env := decodeDietEnvelope(t, w)
	assert.Equal(t, 42200, env.Code)
	assert.Contains(t, env.Message, "请选择集体对象")

	body = `{"name":"对象不存在","academic_year_id":` + yearID +
		`,"group_object_id":999999999,"start_date":"2026-11-02","end_date":"2026-11-08","week_of_year":10,"created_method":"custom"}`
	w = doDietRequest(t, engine, http.MethodPost, "/api/v1/admin/diet/plans", body)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	env = decodeDietEnvelope(t, w)
	assert.Contains(t, env.Message, "集体对象不存在")
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
