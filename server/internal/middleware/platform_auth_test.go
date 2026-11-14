package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
)

func newPlatformEngine(revoker auth.TokenRevoker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("")
	g.Use(PlatformAuth([]byte(middlewareTestSecret), revoker))
	g.GET("/platform/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func signedPlatformToken(t *testing.T) string {
	t.Helper()
	token, err := auth.SignPlatformAccess([]byte(middlewareTestSecret), "7", time.Minute)
	require.NoError(t, err)
	return token
}

func doPlatformRequest(engine *gin.Engine, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/platform/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestPlatformAuthAcceptsPlatformToken(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{})
	w := doPlatformRequest(engine, signedPlatformToken(t))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// 两类凭证互不可代换：园所凭证进不了平台接口。
func TestPlatformAuthRejectsTenantToken(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{})
	w := doPlatformRequest(engine, signedToken(t))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// 反方向：平台凭证进不了园所接口。
func TestAuthJWTRejectsPlatformToken(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{})
	code := callProtected(engine, signedPlatformToken(t))
	assert.Equal(t, http.StatusForbidden, code)
}

func TestPlatformAuthRejectsRevokedToken(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{revoked: true})
	w := doPlatformRequest(engine, signedPlatformToken(t))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
