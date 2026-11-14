package app

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
)

func readyTestRouter(checks ...readinessCheck) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := (&AppHandler{}).WithReadyChecks(checks...)
	r.GET("/ready", h.Ready)
	return r
}

func TestReadyReturnsOKWhenChecksPass(t *testing.T) {
	r := readyTestRouter(
		readinessCheck{name: "mysql", check: func(context.Context) error { return nil }},
		readinessCheck{name: "redis", check: func(context.Context) error { return nil }},
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ready", body["status"])
}

func TestReadyReturns503WhenDependencyDown(t *testing.T) {
	r := readyTestRouter(
		readinessCheck{name: "mysql", check: func(context.Context) error { return errors.New("down") }},
		readinessCheck{name: "redis", check: func(context.Context) error { return nil }},
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "not_ready", body["status"])
	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "down", checks["mysql"])
	assert.Equal(t, "ok", checks["redis"])
}

func TestReadyWithoutInfraIsReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &AppHandler{}
	r.GET("/ready", h.Ready)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}
