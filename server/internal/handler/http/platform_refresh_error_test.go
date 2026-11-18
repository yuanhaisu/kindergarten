package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/fakeredis"
)

// newPlatformSessionOnlyService 只装配平台会话依赖：这些 handler 测试验证
// 错误映射，不碰 MySQL，因此仓储传 nil。
func newPlatformSessionOnlyService(t *testing.T, rdbPresent bool) (*service.PlatformService, *fakeredis.Server) {
	t.Helper()
	if !rdbPresent {
		return service.NewPlatformService(
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
			[]byte("test-platform-secret"), 15*time.Minute, 7*24*time.Hour,
		), nil
	}
	f := fakeredis.New(t)
	rdb := f.Client(t)
	return service.NewPlatformService(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		auth.NewPlatformEpochStore(rdb), auth.NewTokenRevoker(rdb), rdb,
		[]byte("test-platform-secret"), 15*time.Minute, 7*24*time.Hour,
	), f
}

func newPlatformSessionEngine(t *testing.T, svc *service.PlatformService) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := &Handler{platformSvc: svc}
	engine := gin.New()
	engine.POST("/api/v1/platform/auth/refresh", h.PlatformRefresh)
	engine.POST("/api/v1/platform/auth/logout", h.PlatformLogout)
	return engine
}

type platformEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func callPlatformSession(t *testing.T, engine *gin.Engine, path, token, body string) (int, platformEnvelope) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	var env platformEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return w.Code, env
}

// 续期请求必须带 refresh_token；缺失按项目约定返回 422，不是 401。
func TestPlatformRefreshRejectsMissingToken(t *testing.T) {
	svc, _ := newPlatformSessionOnlyService(t, true)
	engine := newPlatformSessionEngine(t, svc)

	code, _ := callPlatformSession(t, engine, "/api/v1/platform/auth/refresh", "", `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
}

// 无法兑换的续期凭证返回 401，不能泄露「键不存在 / 账号停用」的区别。
func TestPlatformRefreshMapsInvalidTokenTo401(t *testing.T) {
	svc, _ := newPlatformSessionOnlyService(t, true)
	engine := newPlatformSessionEngine(t, svc)

	code, env := callPlatformSession(t, engine, "/api/v1/platform/auth/refresh", "", `{"refresh_token":"unknown-jti"}`)
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.NotZero(t, env.Code)
}

// 平台会话存储不可用时续期失败关闭，返回 503（design D5）。
func TestPlatformRefreshMapsStorageUnavailableTo503(t *testing.T) {
	svc, _ := newPlatformSessionOnlyService(t, false)
	engine := newPlatformSessionEngine(t, svc)

	code, _ := callPlatformSession(t, engine, "/api/v1/platform/auth/refresh", "", `{"refresh_token":"any-jti"}`)
	assert.Equal(t, http.StatusServiceUnavailable, code)
}

// 平台登出未携带续期凭证时仍要吊销访问凭证并返回成功。
func TestPlatformLogoutAcceptsEmptyBodyAndRevokesAccess(t *testing.T) {
	svc, f := newPlatformSessionOnlyService(t, true)
	engine := newPlatformSessionEngine(t, svc)

	access, err := auth.SignPlatformAccess([]byte("test-platform-secret"), "7", time.Minute)
	require.NoError(t, err)

	code, _ := callPlatformSession(t, engine, "/api/v1/platform/auth/logout", access, `{}`)
	require.Equal(t, http.StatusOK, code)

	claims, err := auth.ParseAccess([]byte("test-platform-secret"), access)
	require.NoError(t, err)
	ok, _ := f.Exists("token_blacklist:" + claims.ID)
	assert.True(t, ok, "登出必须吊销当前访问凭证")
}

// 平台登出携带续期凭证时必须删除平台键并写黑名单（TC-NEW-PLT-0024）。
func TestPlatformLogoutRevokesRefreshTokenFromBody(t *testing.T) {
	svc, f := newPlatformSessionOnlyService(t, true)
	engine := newPlatformSessionEngine(t, svc)

	access, err := auth.SignPlatformAccess([]byte("test-platform-secret"), "7", time.Minute)
	require.NoError(t, err)
	const jti = "logout-refresh-jti"
	require.NoError(t, auth.SaveRefreshScoped(context.Background(), f.Client(t), auth.PlatformRefreshScope, jti, "7:1700000000", 7*24*time.Hour))

	code, _ := callPlatformSession(t, engine, "/api/v1/platform/auth/logout", access, fmt.Sprintf(`{"refresh_token":%q}`, jti))
	require.Equal(t, http.StatusOK, code)

	ok, _ := f.Exists("refresh:platform:" + jti)
	assert.False(t, ok, "登出必须删除平台续期键")
	ok, ttl := f.Exists("token_blacklist:" + jti)
	assert.True(t, ok, "登出必须把续期 jti 写入黑名单")
	assert.Greater(t, ttl, time.Duration(0))
}

// 登出存储故障返回 503，不能返回 200 让客户端以为会话已结束。
func TestPlatformLogoutMapsStorageUnavailableTo503(t *testing.T) {
	svc, _ := newPlatformSessionOnlyService(t, false)
	engine := newPlatformSessionEngine(t, svc)

	access, err := auth.SignPlatformAccess([]byte("test-platform-secret"), "7", time.Minute)
	require.NoError(t, err)

	code, _ := callPlatformSession(t, engine, "/api/v1/platform/auth/logout", access, `{"refresh_token":"any"}`)
	assert.Equal(t, http.StatusServiceUnavailable, code)
}
