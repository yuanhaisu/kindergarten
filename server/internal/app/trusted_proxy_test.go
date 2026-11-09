package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	handler "kindergarten-server/internal/handler/http"
)

func TestRouterTrustsForwardedForFromLocalhost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, err := NewRouter(&AppHandler{inner: &handler.Handler{}}, nil, JWTSecret("test-secret"), nil, nil, nil)
	require.NoError(t, err)

	// 复用生产路由后追加一个探针，验证 Nginx（127.0.0.1）传来的 X-Forwarded-For
	// 会成为 ClientIP，而不是被当成客户端伪造头忽略。
	engine.GET("/client-ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })

	req := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "203.0.113.10", w.Body.String())
}
