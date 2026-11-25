package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	handler "kindergarten-server/internal/handler/http"
	"kindergarten-server/internal/middleware"
)

type stubPasswordChangeChecker struct {
	must bool
	err  error
}

func (s stubPasswordChangeChecker) MustChangePassword(context.Context, uint64) (bool, error) {
	return s.must, s.err
}

// allowAllRevoker 让 AuthJWT 在测试里通过吊销检查，把断言聚焦到改密门禁。
type allowAllRevoker struct{}

func (allowAllRevoker) Revoked(context.Context, string) (bool, error) { return false, nil }
func (allowAllRevoker) Revoke(context.Context, string, time.Duration) error {
	return nil
}

type zeroTenantEpochs struct{}

func (zeroTenantEpochs) TenantEpoch(context.Context, string) (time.Time, error) {
	return time.Time{}, nil
}
func (zeroTenantEpochs) BumpTenantEpoch(context.Context, string, time.Time) error { return nil }

// 生产路由必须把改密门禁接在 AuthJWT 之后、Casbin/ModuleGate 之前：
// 未改密的 admin_web 凭证访问园所业务接口应得到 403/40303，而不是落到业务 handler。
func TestRouterEnforcesPasswordChangeForAdminWeb(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, err := NewRouter(
		&AppHandler{
			inner:          &handler.Handler{},
			tokenRevoker:   allowAllRevoker{},
			tenantEpochs:   zeroTenantEpochs{},
			passwordChange: stubPasswordChangeChecker{must: true},
		},
		nil, JWTSecret(testJWTSecret), PlatformJWTSecret("test-platform-secret"), nil, nil, nil, nil,
	)
	require.NoError(t, err)

	token, err := auth.SignAdminAccess([]byte(testJWTSecret), "42", []string{"admin"}, 7, time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/kindergarten/info", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code, "未改密的管理端会话不得访问业务接口：%s", w.Body.String())
	var env struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, middleware.PasswordChangeRequiredCode, env.Code)
}
