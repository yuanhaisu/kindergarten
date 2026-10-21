package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
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
	g.GET("/unregistered-probe", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func doGet(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
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
