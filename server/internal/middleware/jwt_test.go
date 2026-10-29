package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
)

const middlewareTestSecret = "middleware-test-secret"

type stubRevoker struct {
	revoked bool
	err     error
	calls   []string
}

func (s *stubRevoker) Revoked(_ context.Context, jti string) (bool, error) {
	s.calls = append(s.calls, jti)
	return s.revoked, s.err
}

func (s *stubRevoker) Revoke(context.Context, string, time.Duration) error { return nil }

func newGuardedEngine(revoker auth.TokenRevoker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("")
	g.Use(AuthJWT([]byte(middlewareTestSecret), revoker))
	g.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func signedToken(t *testing.T) string {
	t.Helper()
	token, err := auth.SignAccess([]byte(middlewareTestSecret), "1", []string{"parent"}, 0, time.Minute)
	require.NoError(t, err)
	return token
}

func signedLegacyToken(t *testing.T) string {
	t.Helper()
	now := time.Now()
	claims := auth.AccessClaims{
		UserID: "1",
		Roles:  []string{"parent"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "kindergarten-api",
			Subject:   "1",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(middlewareTestSecret))
	require.NoError(t, err)
	return token
}

func callProtected(engine *gin.Engine, token string) int {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w.Code
}

func TestAuthJWTRejectsMissingOrInvalidToken(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{})
	assert.Equal(t, http.StatusUnauthorized, callProtected(engine, ""))
	assert.Equal(t, http.StatusUnauthorized, callProtected(engine, "not-a-token"))
}

func TestAuthJWTAllowsValidTokenAndChecksItsJTI(t *testing.T) {
	revoker := &stubRevoker{}
	engine := newGuardedEngine(revoker)

	require.Equal(t, http.StatusNoContent, callProtected(engine, signedToken(t)))
	require.Len(t, revoker.calls, 1, "带 jti 的 token 必须查一次黑名单")
	assert.NotEmpty(t, revoker.calls[0])
}

func TestAuthJWTRejectsRevokedToken(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{revoked: true})
	assert.Equal(t, http.StatusUnauthorized, callProtected(engine, signedToken(t)))
}

// Redis 故障失败关闭：返回 503，而不是放行。
func TestAuthJWTFailsClosedWhenRevokerErrors(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{err: errors.New("redis down")})
	assert.Equal(t, http.StatusServiceUnavailable, callProtected(engine, signedToken(t)))
}

func TestAuthJWTRejectsWhenRevokerNotConfigured(t *testing.T) {
	engine := newGuardedEngine(nil)
	assert.Equal(t, http.StatusServiceUnavailable, callProtected(engine, signedToken(t)))
}

// 加 jti 之前签发的旧 token 无法判断吊销状态，仍按原逻辑放行到过期。
func TestAuthJWTAllowsLegacyTokenWithoutJTI(t *testing.T) {
	revoker := &stubRevoker{err: errors.New("redis down")}
	engine := newGuardedEngine(revoker)

	assert.Equal(t, http.StatusNoContent, callProtected(engine, signedLegacyToken(t)))
	assert.Empty(t, revoker.calls, "无 jti 的 token 不应触发黑名单查询")

	assert.Equal(t, http.StatusNoContent, callProtected(newGuardedEngine(nil), signedLegacyToken(t)))
}
