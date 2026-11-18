package app

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	handler "kindergarten-server/internal/handler/http"
)

// 续期端点必须真实注册：平台访问凭证 15 分钟就过期，少了它客户端只能
// 反复要求重新登录（design D5，TC-NEW-PLT-0011）。
func TestPlatformRefreshRouteRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, err := NewRouter(&AppHandler{inner: &handler.Handler{}}, nil, JWTSecret("test-secret"), nil, nil, nil, nil)
	require.NoError(t, err)

	for _, r := range engine.Routes() {
		if r.Method == http.MethodPost && r.Path == "/api/v1/platform/auth/refresh" {
			return
		}
	}
	t.Fatal("POST /api/v1/platform/auth/refresh 未注册")
}
