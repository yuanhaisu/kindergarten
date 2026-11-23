package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
)

type stubPasswordChangeChecker struct {
	must  bool
	err   error
	calls int
}

func (s *stubPasswordChangeChecker) MustChangePassword(context.Context, uint64) (bool, error) {
	s.calls++
	return s.must, s.err
}

func newPasswordChangeEngine(claims *auth.AccessClaims, checker PasswordChangeChecker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if claims != nil {
			c.Set(CtxClaimsKey, claims)
		}
		c.Next()
	})
	r.Use(RequirePasswordChange(checker))
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	r.GET("/api/v1/common/consent", ok)
	r.POST("/api/v1/common/consent", ok)
	r.GET("/api/v1/common/user/profile", ok)
	r.PUT("/api/v1/common/user/profile", ok)
	r.PUT("/api/v1/common/user/password", ok)
	r.POST("/api/v1/auth/logout", ok)
	r.GET("/api/v1/common/messages", ok)
	r.POST("/api/v1/common/upload", ok)
	r.GET("/api/v1/admin/kindergarten/info", ok)
	r.GET("/api/v1/parent/home", ok)
	return r
}

func adminWebClaims(userID string) *auth.AccessClaims {
	return &auth.AccessClaims{
		UserID:          userID,
		Roles:           []string{"admin"},
		KindergartenID:  7,
		ClientScopeKind: auth.ClientScopeAdminWeb,
	}
}

func doPasswordChangeRequest(t *testing.T, r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodePasswordChangeCode(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	var env struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return env.Code
}

// 管理端未改密时，除精确豁免外一律 403/40303。客户端据此跳改密页，
// 与普通 40300「无权限」和 401「会话失效」区分开。
func TestRequirePasswordChangeBlocksAdminWebBusinessRoutes(t *testing.T) {
	checker := &stubPasswordChangeChecker{must: true}
	r := newPasswordChangeEngine(adminWebClaims("42"), checker)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/admin/kindergarten/info"},
		{http.MethodGet, "/api/v1/common/messages"},
		{http.MethodPost, "/api/v1/common/upload"},
		{http.MethodPut, "/api/v1/common/user/profile"},
		{http.MethodGet, "/api/v1/parent/home"},
	} {
		w := doPasswordChangeRequest(t, r, tc.method, tc.path)
		require.Equal(t, http.StatusForbidden, w.Code, "%s %s", tc.method, tc.path)
		assert.Equal(t, PasswordChangeRequiredCode, decodePasswordChangeCode(t, w), "%s %s", tc.method, tc.path)
	}
}

// 豁免是 method + path 精确匹配：同意、个人资料读取、改密、登出。
func TestRequirePasswordChangeAllowsExactExemptions(t *testing.T) {
	checker := &stubPasswordChangeChecker{must: true}
	r := newPasswordChangeEngine(adminWebClaims("42"), checker)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/common/consent"},
		{http.MethodPost, "/api/v1/common/consent"},
		{http.MethodGet, "/api/v1/common/user/profile"},
		{http.MethodPut, "/api/v1/common/user/password"},
		{http.MethodPost, "/api/v1/auth/logout"},
	} {
		w := doPasswordChangeRequest(t, r, tc.method, tc.path)
		assert.Equal(t, http.StatusNoContent, w.Code, "%s %s 必须在豁免列表内", tc.method, tc.path)
	}
}

// 改密后同一 access 的下一个请求立即放行：checker 每请求读库，不缓存。
func TestRequirePasswordChangeReadsFlagOnEveryRequest(t *testing.T) {
	checker := &stubPasswordChangeChecker{must: true}
	r := newPasswordChangeEngine(adminWebClaims("42"), checker)

	blocked := doPasswordChangeRequest(t, r, http.MethodGet, "/api/v1/admin/kindergarten/info")
	require.Equal(t, http.StatusForbidden, blocked.Code)

	checker.must = false
	allowed := doPasswordChangeRequest(t, r, http.MethodGet, "/api/v1/admin/kindergarten/info")
	assert.Equal(t, http.StatusNoContent, allowed.Code, "改密后不得继续拦截")
	assert.Equal(t, 2, checker.calls, "每次请求都要读一次真实标记")
}

// 门禁只作用于 admin_web：小程序凭证与旧式无作用域凭证不受影响，
// 也不应产生额外的每请求查库成本。
func TestRequirePasswordChangeIgnoresNonAdminWebTokens(t *testing.T) {
	for name, claims := range map[string]*auth.AccessClaims{
		"mini-program parent": {
			UserID: "42", Roles: []string{"parent"}, KindergartenID: 7,
			AppID: "wx-parent-test", ClientScopeKind: "shared",
		},
		"legacy unscoped": {UserID: "42", Roles: []string{"parent"}, KindergartenID: 7},
	} {
		t.Run(name, func(t *testing.T) {
			checker := &stubPasswordChangeChecker{must: true}
			r := newPasswordChangeEngine(claims, checker)
			w := doPasswordChangeRequest(t, r, http.MethodGet, "/api/v1/parent/home")
			assert.Equal(t, http.StatusNoContent, w.Code)
			assert.Zero(t, checker.calls, "非管理端凭证不应触发改密检查")
		})
	}
}

// 读取失败必须失败关闭：宁可 503，也不能把可能未改密的会话放进来。
func TestRequirePasswordChangeFailsClosedOnCheckerError(t *testing.T) {
	checker := &stubPasswordChangeChecker{err: errors.New("db down")}
	r := newPasswordChangeEngine(adminWebClaims("42"), checker)

	w := doPasswordChangeRequest(t, r, http.MethodGet, "/api/v1/admin/kindergarten/info")
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, 50300, decodePasswordChangeCode(t, w))
}

func TestRequirePasswordChangeFailsClosedWithoutChecker(t *testing.T) {
	r := newPasswordChangeEngine(adminWebClaims("42"), nil)

	w := doPasswordChangeRequest(t, r, http.MethodGet, "/api/v1/admin/kindergarten/info")
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, 50300, decodePasswordChangeCode(t, w))
}

// admin_web 凭证的 uid 必须可解析；损坏的 uid 按未认证处理，不得放行。
func TestRequirePasswordChangeRejectsUnparsableUserID(t *testing.T) {
	checker := &stubPasswordChangeChecker{}
	r := newPasswordChangeEngine(adminWebClaims("not-a-number"), checker)

	w := doPasswordChangeRequest(t, r, http.MethodGet, "/api/v1/admin/kindergarten/info")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Zero(t, checker.calls)
}
