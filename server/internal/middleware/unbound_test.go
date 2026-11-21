package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
)

func newUnboundEngine(claims *auth.AccessClaims) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if claims != nil {
			c.Set(CtxClaimsKey, claims)
		}
		c.Next()
	})
	r.Use(RequireBoundAccount())
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	r.GET("/api/v1/common/user/profile", ok)
	r.PUT("/api/v1/common/user/profile", ok)
	r.PUT("/api/v1/common/user/avatar", ok)
	r.GET("/api/v1/common/user/kindergartens", ok)
	r.POST("/api/v1/common/user/bind-parent-phone", ok)
	r.GET("/api/v1/parent/home", ok)
	return r
}

func doUnboundRequest(t *testing.T, r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeEnvelopeCode(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	var env struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return env.Code
}

// 待绑定 token（app_id 存在且角色为空）只能访问白名单：用户资料、园所列表、
// 绑定与同意都在里面，#160 的「未绑定 token 的强制白名单」据此收口。
func TestRequireBoundAccountAllowsWhitelistedRoutes(t *testing.T) {
	r := newUnboundEngine(&auth.AccessClaims{UserID: "1", AppID: "wx-parent-test"})
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/common/user/profile"},
		{http.MethodPut, "/api/v1/common/user/profile"},
		{http.MethodGet, "/api/v1/common/user/kindergartens"},
		{http.MethodPost, "/api/v1/common/user/bind-parent-phone"},
	} {
		w := doUnboundRequest(t, r, tc.method, tc.path)
		assert.Equal(t, http.StatusNoContent, w.Code, "%s %s 应在白名单内", tc.method, tc.path)
	}
}

// 头像不在白名单：待绑定账号还不需要改头像，多一条写入口就多一分暴露面。
func TestRequireBoundAccountRejectsAvatarForUnboundToken(t *testing.T) {
	r := newUnboundEngine(&auth.AccessClaims{UserID: "1", AppID: "wx-parent-test"})
	w := doUnboundRequest(t, r, http.MethodPut, "/api/v1/common/user/avatar")
	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, UnboundCode, decodeEnvelopeCode(t, w))
}

func TestRequireBoundAccountRejectsBusinessRoutes(t *testing.T) {
	r := newUnboundEngine(&auth.AccessClaims{UserID: "1", AppID: "wx-parent-test"})
	w := doUnboundRequest(t, r, http.MethodGet, "/api/v1/parent/home")
	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, UnboundCode, decodeEnvelopeCode(t, w), "客户端据此跳绑定页而不是提示无权限")
}

// 已绑定用户与没有 app_id 的管理端/平台凭证行为完全不变。
func TestRequireBoundAccountPassesBoundAndNonClientTokens(t *testing.T) {
	bound := newUnboundEngine(&auth.AccessClaims{UserID: "1", AppID: "wx-parent-test", Roles: []string{"parent"}})
	assert.Equal(t, http.StatusNoContent,
		doUnboundRequest(t, bound, http.MethodGet, "/api/v1/parent/home").Code)
	assert.Equal(t, http.StatusNoContent,
		doUnboundRequest(t, bound, http.MethodPut, "/api/v1/common/user/avatar").Code)

	admin := newUnboundEngine(&auth.AccessClaims{UserID: "2", Scope: "admin", Roles: []string{"admin"}})
	assert.Equal(t, http.StatusNoContent,
		doUnboundRequest(t, admin, http.MethodGet, "/api/v1/parent/home").Code,
		"管理端凭证不经过未绑定限制")
}
