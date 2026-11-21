package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
)

type fakeConsentRepoForHandler struct {
	rows map[string]*domain.UserConsent
}

func (f *fakeConsentRepoForHandler) key(userID uint64, version string) string {
	return fmt.Sprintf("%d:%s", userID, version)
}

func (f *fakeConsentRepoForHandler) FindByUserAndVersion(_ context.Context, userID uint64, version string) (*domain.UserConsent, error) {
	return f.rows[f.key(userID, version)], nil
}

func (f *fakeConsentRepoForHandler) CreateIfAbsent(_ context.Context, c *domain.UserConsent) error {
	k := f.key(c.UserID, c.PolicyVersion)
	if _, ok := f.rows[k]; !ok {
		f.rows[k] = c
	}
	return nil
}

func (f *fakeConsentRepoForHandler) ListByUser(_ context.Context, _ uint64) ([]domain.UserConsent, error) {
	return nil, nil
}

func newConsentHandlerEngine(t *testing.T, repo service.ConsentRepository, roles []string) *gin.Engine {
	t.Helper()
	svc := service.NewAuthService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 0, 0, repo, true)
	h := &Handler{authSvc: svc}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxClaimsKey, &auth.AccessClaims{UserID: "7", Roles: roles})
		c.Next()
	})
	r.GET("/consent", h.GetConsent)
	r.POST("/consent", h.PostConsent)
	return r
}

func TestConsentHandler_GetAndRecord(t *testing.T) {
	repo := &fakeConsentRepoForHandler{rows: map[string]*domain.UserConsent{}}
	engine := newConsentHandlerEngine(t, repo, []string{"parent"})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/consent", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Data struct {
			PolicyVersion string `json:"policy_version"`
			Consented     bool   `json:"consented"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.False(t, got.Data.Consented)
	require.Equal(t, service.CurrentPrivacyPolicyVersion, got.Data.PolicyVersion)

	body := `{"policy_version":"` + service.CurrentPrivacyPolicyVersion + `"}`
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	recorded, ok := repo.rows[repo.key(7, service.CurrentPrivacyPolicyVersion)]
	require.True(t, ok)
	require.Equal(t, domain.ConsentSourceParent, recorded.Source)
}

func TestConsentHandler_RejectsStaleVersion(t *testing.T) {
	repo := &fakeConsentRepoForHandler{rows: map[string]*domain.UserConsent{}}
	engine := newConsentHandlerEngine(t, repo, []string{"parent"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/consent", strings.NewReader(`{"policy_version":"v0"}`))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	var env struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Equal(t, 40030, env.Code)
}

func TestConsentHandler_RejectsUnknownSource(t *testing.T) {
	repo := &fakeConsentRepoForHandler{rows: map[string]*domain.UserConsent{}}
	engine := newConsentHandlerEngine(t, repo, []string{"unknown"})

	w := httptest.NewRecorder()
	body := `{"policy_version":"` + service.CurrentPrivacyPolicyVersion + `"}`
	req := httptest.NewRequest(http.MethodPost, "/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsentHandler_HeaderOverridesRoleOrder(t *testing.T) {
	repo := &fakeConsentRepoForHandler{rows: map[string]*domain.UserConsent{}}
	engine := newConsentHandlerEngine(t, repo, []string{"teacher", "parent"})

	w := httptest.NewRecorder()
	body := `{"policy_version":"` + service.CurrentPrivacyPolicyVersion + `"}`
	req := httptest.NewRequest(http.MethodPost, "/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("terminal_from", "parent")
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	recorded := repo.rows[repo.key(7, service.CurrentPrivacyPolicyVersion)]
	require.NotNil(t, recorded)
	require.Equal(t, domain.ConsentSourceParent, recorded.Source)
}

func TestConsentHandler_RejectsHeaderNotInRoles(t *testing.T) {
	repo := &fakeConsentRepoForHandler{rows: map[string]*domain.UserConsent{}}
	engine := newConsentHandlerEngine(t, repo, []string{"parent"})

	w := httptest.NewRecorder()
	body := `{"policy_version":"` + service.CurrentPrivacyPolicyVersion + `"}`
	req := httptest.NewRequest(http.MethodPost, "/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("terminal_from", "admin")
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	var env struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Equal(t, 40031, env.Code)
	require.Empty(t, repo.rows)
}

func TestConsentHandler_FallsBackToRoleOrderWithoutHeader(t *testing.T) {
	repo := &fakeConsentRepoForHandler{rows: map[string]*domain.UserConsent{}}
	engine := newConsentHandlerEngine(t, repo, []string{"teacher", "parent"})

	w := httptest.NewRecorder()
	body := `{"policy_version":"` + service.CurrentPrivacyPolicyVersion + `"}`
	req := httptest.NewRequest(http.MethodPost, "/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	recorded := repo.rows[repo.key(7, service.CurrentPrivacyPolicyVersion)]
	require.NotNil(t, recorded)
	require.Equal(t, domain.ConsentSourceTeacher, recorded.Source, "缺省时保持旧行为")
}
