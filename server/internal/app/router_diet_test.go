package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	handler "kindergarten-server/internal/handler/http"
	"kindergarten-server/internal/middleware"
)

// dietTestResolver 是权限中间件的桩：只返回指定权限码，不依赖 DB。
type dietTestResolver struct {
	codes    []string
	wildcard bool
}

func (s dietTestResolver) ResolveAdminCodes(context.Context, uint64, uint64) ([]string, bool, error) {
	return s.codes, s.wildcard, nil
}

// newDietAdminEngine 只装配 admin 路由与细闸中间件，不经过 AuthJWT/Casbin/Redis。
// Handler 用零值：S1b 的 DietConfigService 是纯占位、不触碰字段，因此 nil receiver
// 也能返回 501。S2 实现真实读写后，本测试需要改为注入真实 service。
func newDietAdminEngine(t *testing.T, resolver middleware.PermissionResolver) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	adminG := r.Group("/api/v1/admin")
	adminG.Use(func(c *gin.Context) {
		c.Set(middleware.CtxClaimsKey, &auth.AccessClaims{
			UserID: "4", Roles: []string{"admin"}, KindergartenID: 4,
		})
		c.Next()
	})
	adminG.Use(middleware.RequirePermission(resolver))
	registerAdminRoutes(adminG, &handler.Handler{})
	return r
}

func doDietRequest(t *testing.T, engine *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if method == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}
	engine.ServeHTTP(w, req)
	return w
}

func decodeDietEnvelope(t *testing.T, w *httptest.ResponseRecorder) (code int, message string) {
	t.Helper()
	var env struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env),
		"响应必须是 JSON 包络：%s", w.Body.String())
	return env.Code, env.Message
}

// S1b 骨架：GET/PUT /admin/diet/config 必须返回 501 + 项目统一错误包络。
func TestRouterDiet_ConfigSkeletonReturns501(t *testing.T) {
	engine := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view", "diet:manage"}})

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/admin/diet/config"},
		{http.MethodPut, "/api/v1/admin/diet/config"},
	} {
		t.Run(tc.method, func(t *testing.T) {
			w := doDietRequest(t, engine, tc.method, tc.path)
			require.Equal(t, http.StatusNotImplemented, w.Code, w.Body.String())
			code, message := decodeDietEnvelope(t, w)
			assert.Equal(t, 50100, code)
			assert.NotEmpty(t, message)
		})
	}
}

// 两条配置路由的权限码：GET=diet:view，PUT=diet:manage，互不替代。
func TestRouterDiet_ConfigPermissionCodes(t *testing.T) {
	viewOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:view"}})
	assert.Equal(t, http.StatusNotImplemented,
		doDietRequest(t, viewOnly, http.MethodGet, "/api/v1/admin/diet/config").Code)
	assert.Equal(t, http.StatusForbidden,
		doDietRequest(t, viewOnly, http.MethodPut, "/api/v1/admin/diet/config").Code)

	manageOnly := newDietAdminEngine(t, dietTestResolver{codes: []string{"diet:manage"}})
	assert.Equal(t, http.StatusForbidden,
		doDietRequest(t, manageOnly, http.MethodGet, "/api/v1/admin/diet/config").Code)
	assert.Equal(t, http.StatusNotImplemented,
		doDietRequest(t, manageOnly, http.MethodPut, "/api/v1/admin/diet/config").Code)
}
