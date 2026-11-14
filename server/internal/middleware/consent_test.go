package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/service"
)

type fakeConsentChecker struct {
	enforced  bool
	consented bool
	err       error
}

func (f fakeConsentChecker) ConsentEnforced() bool { return f.enforced }

func (f fakeConsentChecker) HasConsented(_ context.Context, _ uint64) (bool, error) {
	return f.consented, f.err
}

func newConsentTestEngine(checker service.ConsentChecker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(CtxClaimsKey, &auth.AccessClaims{UserID: "7"})
		c.Next()
	})
	r.Use(RequireConsent(checker))
	r.GET("/x", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func doConsentRequest(engine *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	engine.ServeHTTP(w, req)
	return w
}

func TestRequireConsent_PassesWhenNotEnforced(t *testing.T) {
	w := doConsentRequest(newConsentTestEngine(fakeConsentChecker{enforced: false, consented: false}))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestRequireConsent_PassesWhenConsented(t *testing.T) {
	w := doConsentRequest(newConsentTestEngine(fakeConsentChecker{enforced: true, consented: true}))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestRequireConsent_BlocksWithRecognizableCode(t *testing.T) {
	w := doConsentRequest(newConsentTestEngine(fakeConsentChecker{enforced: true, consented: false}))
	require.Equal(t, http.StatusForbidden, w.Code)

	var env struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Equal(t, ConsentRequiredCode, env.Code)
}

func TestRequireConsent_UnavailableOnCheckerError(t *testing.T) {
	w := doConsentRequest(newConsentTestEngine(fakeConsentChecker{enforced: true, err: errors.New("db down")}))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestRequireConsent_NilCheckerPasses(t *testing.T) {
	w := doConsentRequest(newConsentTestEngine(nil))
	require.Equal(t, http.StatusOK, w.Code)
}
