package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

// 管理端权限中间件行为（wire-admin-permissions，TC-NEW-PERM-0004/0005/0006）。
// 用桩 resolver + gin 测试引擎，不依赖 DB。

type stubResolver struct {
	codes    []string
	wildcard bool
	err      error
}

func (s stubResolver) ResolveAdminCodes(_ context.Context, _, _ uint64) ([]string, bool, error) {
	return s.codes, s.wildcard, s.err
}

func newPermEngine(resolver PermissionResolver) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1/admin")
	g.Use(func(c *gin.Context) {
		c.Set(CtxClaimsKey, &auth.AccessClaims{UserID: "4", Roles: []string{"admin"}, KindergartenID: 4})
		c.Next()
	})
	g.Use(RequirePermission(resolver))
	g.GET("/dashboard/overview", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	g.GET("/diet/config", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	g.PUT("/diet/config", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	g.GET("/unregistered-probe", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func doGet(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func doPut(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, path, nil)
	r.ServeHTTP(w, req)
	return w
}

// TC-NEW-PERM-0004 持有对应权限码的调用放行。
func TestRequirePermission_AllowWithCode(t *testing.T) {
	r := newPermEngine(stubResolver{codes: []string{"dashboard:view"}})
	w := doGet(r, "/api/v1/admin/dashboard/overview")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

// TC-NEW-PERM-0005 缺少权限码的调用被拒（403），handler 不执行。
func TestRequirePermission_DenyWithoutCode(t *testing.T) {
	r := newPermEngine(stubResolver{codes: []string{"student:view"}})
	w := doGet(r, "/api/v1/admin/dashboard/overview")
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NotEqual(t, "ok", w.Body.String())
}

// 通配账号通过一切检查。
func TestRequirePermission_WildcardPasses(t *testing.T) {
	r := newPermEngine(stubResolver{codes: []string{"*"}, wildcard: true})
	w := doGet(r, "/api/v1/admin/dashboard/overview")
	assert.Equal(t, http.StatusOK, w.Code)
}

// 解析出错按 500 处理，绝不按“无权限”吞掉。
func TestRequirePermission_ResolverErrorIs500(t *testing.T) {
	r := newPermEngine(stubResolver{err: errors.New("db down")})
	w := doGet(r, "/api/v1/admin/dashboard/overview")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC-NEW-PERM-0006 登记表未覆盖的接口被拒（fail-closed），handler 不执行。
func TestRequirePermission_UnregisteredRouteDenied(t *testing.T) {
	r := newPermEngine(stubResolver{codes: []string{"*"}, wildcard: true})
	w := doGet(r, "/api/v1/admin/unregistered-probe")
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NotEqual(t, "ok", w.Body.String())
}

// TC-NEW-PERM-0007 缺口检测：未登记的管理端路由必须被报出；非管理端前缀不受影响。
func TestAdminPermissionGaps(t *testing.T) {
	routes := []gin.RouteInfo{
		{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/overview"},
		{Method: http.MethodGet, Path: "/api/v1/admin/notices"},
		{Method: http.MethodGet, Path: "/api/v1/teacher/pickup/records"}, // 非管理端，忽略
	}
	table := map[string]string{
		"GET /api/v1/admin/dashboard/overview": "dashboard:view",
	}
	gaps := adminPermissionGaps(routes, table)
	require.Len(t, gaps, 1)
	assert.Equal(t, "GET /api/v1/admin/notices", gaps[0])

	// 真实登记表必须覆盖这两条真实路由（抽样保真）
	assert.Empty(t, adminPermissionGaps(routes[:2], adminRoutePermission))
}

// AQR-S1b 2.3：供餐配置两条路由必须登记到 diet:view / diet:manage。
func TestAdminPermission_DietConfigRoutesRegistered(t *testing.T) {
	routes := []gin.RouteInfo{
		{Method: http.MethodGet, Path: "/api/v1/admin/diet/config"},
		{Method: http.MethodPut, Path: "/api/v1/admin/diet/config"},
	}
	assert.Empty(t, adminPermissionGaps(routes, adminRoutePermission), "diet 配置路由不得有登记缺口")
	assert.Equal(t, "diet:view", adminRoutePermission["GET /api/v1/admin/diet/config"])
	assert.Equal(t, "diet:manage", adminRoutePermission["PUT /api/v1/admin/diet/config"])
}

// 查看码可以读配置，但不能保存。
func TestAdminPermission_DietConfigViewCode(t *testing.T) {
	r := newPermEngine(stubResolver{codes: []string{"diet:view"}})
	assert.Equal(t, http.StatusOK, doGet(r, "/api/v1/admin/diet/config").Code)
	assert.Equal(t, http.StatusForbidden, doPut(r, "/api/v1/admin/diet/config").Code)
}

// 管理码可以保存配置，但不能读配置（两个码互不替代）。
func TestAdminPermission_DietConfigManageCode(t *testing.T) {
	r := newPermEngine(stubResolver{codes: []string{"diet:manage"}})
	assert.Equal(t, http.StatusForbidden, doGet(r, "/api/v1/admin/diet/config").Code)
	assert.Equal(t, http.StatusOK, doPut(r, "/api/v1/admin/diet/config").Code)
}

func openAdminPermTestDB(t *testing.T) *gorm.DB {
	t.Helper()
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
	return db
}

// 角色目录新增 diet 模块（view/manage/export），角色管理页据此渲染勾选项。
// PermissionTree 会先确保权限目录已落库，因此这条用例需要真实 DB；无 DB 时跳过，
// 路由与权限码断言仍由上面不依赖 DB 的用例覆盖。
func TestAdminPermission_RoleCatalogHasDietModule(t *testing.T) {
	db := openAdminPermTestDB(t)
	svc := service.NewRoleService(db, repository.NewAdminRepo(db), repository.NewOperationLogRepo(db))

	tree, err := svc.PermissionTree(context.Background())
	require.NoError(t, err)

	codes := map[string]string{}
	for _, module := range tree {
		if module.Module != "diet" {
			continue
		}
		for _, item := range module.Items {
			codes[item.Key] = item.Label
		}
	}
	require.Len(t, codes, 3, "diet 模块必须恰好包含 view/manage/export 三个权限码")
	assert.Contains(t, codes, "diet:view")
	assert.Contains(t, codes, "diet:manage")
	assert.Contains(t, codes, "diet:export")
}
