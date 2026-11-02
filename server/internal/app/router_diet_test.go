package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		"DELETE FROM diet_foods WHERE kindergarten_id = ?",
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
	foodSvc := service.NewDietFoodService(repository.NewDietFoodRepo(db), repository.NewDietRecipeRepo(db))

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
	registerAdminRoutes(adminG, handler.NewDietAdminHandler(configSvc, foodSvc))
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
	assert.Len(t, got, 10, "diet 组当前应恰好 2 条配置 + 8 条食物路由")
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
